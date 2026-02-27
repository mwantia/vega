# Runtime

**Package:** `pkg/vm/` (files `runtime.go`, `vm.go`)

The runtime executes bytecode instructions within call frames, managing the expression stack, the global byte-array allocator, and the slot table.

---

## Session

```go
type Session struct {
    allocator alloc.Allocator
    slots     []SlotEntry
    funcs     map[string]*compiler.FunctionDef
    snapshots *alloc.SnapshotManager // nil if allocator does not implement Snapshottable
}
```

The `Session` holds the persistent state that survives across `Run()` calls. In REPL mode, variables and function definitions accumulate in the session. `ResetSession()` replaces the session with a fresh one, clearing all allocator state, slots, and defined functions.

The default allocator is a `FreeListAllocator` with `DefaultAllocSize = 1 MiB`. The `SnapshotManager` is initialized automatically when the allocator implements the `Snapshottable` interface (which `FreeListAllocator` does). It records incremental deltas for history replay and TUI visualization.

---

## Architecture

```go
type Runtime struct {
    Frames    []*CallFrame      // up to 256 frames
    Index     int               // current frame index (0 = top-level)
    exprStack *ExprStack        // expression stack (always non-nil in an active execution)
    allocator alloc.Allocator   // global allocator shared across all scopes
    slots     []SlotEntry       // variable slot table for the current scope
    native    *Native           // native function context (I/O streams, VFS)
    pendingArgs []value.Value   // arguments staged for the next CALL_FN
    userFuncs map[string]*compiler.FunctionDef // compiled user functions
}
```

The `allocator` is initialized once per session and is never nil during execution. It is shared across all function calls — only the `exprStack` and `slots` are per-scope. `pendingArgs` holds arguments popped from the caller's stack during `CALL_FN` and consumed by `LOAD_ARG` / `LOAD_ARG_STENCIL` in the callee's prologue.

---

## Expression Stack

```go
type ExprStack struct {
    data []value.Value
}
```

A plain `[]value.Value` with no capacity limit. The expression stack is used for:

- Pushing constants (`LOAD_CONST`)
- Pushing decoded variable values (`VAR_LOAD`, `FIELD_LOAD`)
- Popping values for storage (`VAR_STORE`, `FIELD_STORE`)
- Discarding expression results (`STACK_POP`)
- Passing arguments to function calls (`CALL_NAT`, `CALL_FN`)

The expression stack holds transient `Value` interfaces (Go-managed). The allocator holds persistent variables (byte-managed). These two concerns are cleanly separated.

---

## Call Frames

```go
type CallFrame struct {
    ByteCode           *compiler.ByteCode
    InstructionPointer int
    BasePointer        int

    // ExprCall is true when this frame was entered from an expression context
    // (the return value is expected on the caller's stack).
    ExprCall bool

    // Saved caller state — restored when this frame returns.
    savedExprStack *ExprStack
    savedSlots     []SlotEntry
}
```

When `CALL_FN` is executed, the caller's `exprStack` and `slots` are saved inside the **current** `CallFrame` before the new frame is pushed. When the callee returns (`doReturn`), these saved values are restored from the caller's frame.

The global `allocator` is never saved or restored — it is shared across all frames throughout the session.

---

## Slot Table

```go
type SlotEntry struct {
    Offset   int           // byte offset into the allocator buffer
    Capacity int           // number of bytes allocated (max across union types; declared capacity for slices)
    Tag      value.TypeTag  // current variant tag (0 = uninitialized)
    Mask     byte          // bitmask of allowed type tags (0 for stencils and slices)
    Alive    bool          // false after VAR_FREE
    Alias    bool          // true = manually positioned pointer, not allocator-owned
    Stencil  bool          // true = stencil-based allocation (struct/tuple)
}
```

The slot table is a `[]SlotEntry` indexed by slot ID. It grows dynamically as `VAR_ALLOC`, `VAR_PTR`, `STENCIL_ALLOC`, and `SLICE_ALLOC` instructions are executed.

- `Mask` stores the set of allowed types as a bitmask. For stencil and slice slots, `Mask=0`.
- `Capacity` holds the allocated byte count. For scalar slots it is `MaxSizeForMask(mask)`; for slice slots it is the declared `<N>` value.
- `Tag` tracks the **current** variant — updated on every `VAR_STORE`, used by `VAR_LOAD` for decoding. Starts at 0 (uninitialized) until the first store. For alias slots, `Tag` is set immediately by `VAR_PTR`.
- `Alias` marks pointer alias slots. These point to explicit offsets and cannot be freed via `VAR_FREE`.
- `Stencil` marks struct/tuple slots. Fields are accessed via `FIELD_LOAD` / `FIELD_STORE`.

---

## Instruction Execution

The `ExecuteFrames` loop runs the standard fetch-decode-execute cycle with context cancellation checks:

```go
func (r *Runtime) ExecuteFrames(ctx context.Context) error {
    for {
        select {
        case <-ctx.Done():
            return ctx.Err()
        default:
        }
        frame := r.IndexedFrame()
        if frame.InstructionPointer >= len(frame.ByteCode.Instructions) {
            if r.Index == 0 {
                return nil // top-level program complete
            }
            // Implicit void return at end of function
            if err := r.doReturn(frame, false, nil); err != nil {
                return err
            }
            continue
        }
        instr := frame.ByteCode.Instructions[frame.InstructionPointer]
        frame.InstructionPointer++
        if err := r.ExecuteInstruction(instr, frame); err != nil {
            return fmt.Errorf("line %d: %w", instr.SourceLine, err)
        }
    }
}
```

### Handler Summary

| Instruction | Stack effect | Allocator effect |
|-------------|-------------|------------------|
| `LOAD_CONST i` | Push `constants[i]` | — |
| `STACK_POP` | Pop + discard | — |
| `VAR_ALLOC slot=S mask=M` | — | `Alloc(MaxSizeForMask(M))`, record in `slots[S]` with `Tag=0` |
| `VAR_STORE slot=S` | Pop value | Encode + write bytes at slot offset; for slices: validate content ≤ capacity, zero-pad |
| `VAR_LOAD slot=S` | Push decoded value | Read `SizeForTag(tag)` bytes (or `slot.Capacity` for slices) at slot offset |
| `VAR_FREE slot=S` | — | `Free(offset, capacity)`, mark dead (rejects aliases) |
| `VAR_PTR slot=S tag=T` | Pop offset | Bounds-check, create alias slot at offset |
| `STENCIL_ALLOC slot=S size=N` | — | `Alloc(N)`, record in `slots[S]` with `Stencil=true` |
| `SLICE_ALLOC slot=S cap=N` | — | `Alloc(N)`, record in `slots[S]` with `Tag=TagSlice`, `Capacity=N` |
| `FIELD_STORE slot=S off=O tag=T size=Z` | Pop value | Type-check tag, write into `buffer[slot.Offset+O]`; slices: validate+zero-pad |
| `FIELD_LOAD slot=S off=O tag=T size=Z` | Push decoded value | Read `fieldSizeFor(T,Z)` bytes at `buffer[slot.Offset+O]`, wrap as value |
| `CALL_NAT name argc` | Pop `argc` args | — (native functions manage their own I/O) |
| `CALL_FN name argc` | Pop `argc` args into `pendingArgs`, push new frame | — (allocator not touched at call site) |
| `RETURN` | Materialize + pop return value | Free all callee slots, restore caller state |
| `LOAD_ARG i` | Push `pendingArgs[i]` | — |
| `VAR_LOAD_RAW slot=S size=N` | Push `RawValue` copy | Read `N` bytes from slot, copy to heap |
| `LOAD_ARG_STENCIL i slot=S size=N` | — | `Alloc(N)`, create stencil slot, bulk-copy from `pendingArgs[i]` |
| `PTR_LOAD tag=T` | Pop offset, push decoded value | Bounds-check, read at offset |

---

## Function Call Mechanics

### Calling (`CALL_FN`)

```
1. Pop argc values from exprStack into args[].
2. Save r.exprStack and r.slots into currentFrame.savedExprStack / savedSlots.
3. Set r.exprStack = fresh ExprStack{}.
4. Set r.slots = make([]SlotEntry, 0).
5. Set r.pendingArgs = args.
6. Increment r.Index, push new CallFrame for callee's ByteCode.
```

The global allocator is untouched. Only the expression-level state is swapped.

### Prologue (`LOAD_ARG` / `LOAD_ARG_STENCIL`)

Each function parameter is set up by the compiled prologue:

- **Primitive:** `LOAD_ARG i` pushes the argument; `VAR_ALLOC` + `VAR_STORE` allocates and fills a slot.
- **Struct:** `LOAD_ARG_STENCIL i slot=S size=N` reads the `RawValue` from `pendingArgs[i]`, allocates `N` bytes from the global allocator, and bulk-copies the struct's bytes into the new stencil slot.

### Returning (`doReturn`)

```go
func (r *Runtime) doReturn(frame *CallFrame, hasValue bool, retVal value.Value) error {
    // 1. Materialize the return value — copy its bytes out of allocator memory.
    // 2. Free all alive, non-alias slots in r.slots back to the global allocator.
    // 3. Decrement r.Index, restore r.exprStack and r.slots from callerFrame.saved*.
    // 4. If hasValue && frame.ExprCall, push the materialized value onto r.exprStack.
}
```

**Materialization** is the critical step: a return value may be a view into the allocator buffer (e.g., an `IntegerValue` whose `view` points into a slot region). If the slot is freed before the value escapes to the caller, the view would be dangling. Materialization copies the bytes into a fresh `[]byte` on the Go heap before any slots are freed.

---

## Native Functions

Native functions are Go functions registered in the `native.go` lookup table:

```go
type NativeFn func(*Native, []value.Value) error

func lookupNative(name string) (NativeFn, bool) { ... }
```

The `Native` struct carries I/O streams and the VFS instance:

```go
type Native struct {
    Stdin  io.Reader
    Stdout io.Writer
    Stderr io.Writer
    FS     vfs.VirtualFileSystem
}
```

Native functions receive `[]value.Value` (already popped from the expression stack) and write their output to the `Native` context. They do not interact with the slot table or the allocator directly.

---

## VM

```go
type VM struct {
    mu      sync.RWMutex
    fs      vfs.VirtualFileSystem
    session *Session
    stdin   io.Reader
    stdout  io.Writer
    stderr  io.Writer
}
```

The VM is the top-level entry point. `NewVM(fs)` creates a VM with a fresh session backed by a 1 MiB `FreeListAllocator`. `Run()` injects the session's known functions into the incoming bytecode, creates a `Runtime`, executes it, and persists the updated slots and new function definitions back into the session.

```go
func (v *VM) Run(ctx context.Context, bytecode *compiler.ByteCode) (int, error)
```

Returns `(0, nil)` on success, `(1, error)` on failure.

The VM is thread-safe via `sync.RWMutex`. Each `Run()` call takes an exclusive write lock, so concurrent executions are serialized.

---

## Error Reporting

All runtime errors include the source line number:

```go
return fmt.Errorf("line %d: %w", instr.SourceLine, err)
```

The VM wraps this further:

```go
return 1, fmt.Errorf("runtime execution failed: %w", err)
```

### Error Messages

| Situation | Message |
|-----------|---------|
| No allocator active | "instr 'OpVarALLOC': no allocator active" |
| Out of memory | "instr 'OpVarALLOC': out of memory: need N bytes, have M free" |
| Store to dead slot | "instr 'OpVarSTORE': slot N is not alive" |
| Store non-allocable | "instr 'OpVarSTORE': value is not allocable" |
| Type mismatch | "instr 'OpVarSTORE': type mismatch: slot mask XXXXXXXX does not allow tag Y" |
| Load uninitialized | "instr 'OpVarLOAD': slot N is uninitialized" |
| Use after free | "instr 'OpVarLOAD': use after free on slot N" |
| Double free | "instr 'OpVarFREE': double free on slot N" |
| Free pointer alias | "instr 'OpVarFREE': cannot free pointer alias on slot N" |
| Pointer out of bounds | "instr 'OpVarPTR': pointer out of bounds (offset=N, size=M, capacity=C)" |
| Pointer offset not allocable | "instr 'OpVarPTR': offset value is not allocable" |
| Stencil out of memory | "instr 'OpStencilALLOC': out of memory: ..." |
| Slice out of memory | "instr 'OpSliceALLOC': out of memory: ..." |
| Slice overflow on store | "instr 'OpVarSTORE': slice overflow: content N bytes exceeds capacity M" |
| Slice field overflow on store | "instr 'OpFieldSTORE': slice overflow: content N bytes exceeds capacity M" |
| Unknown native function | "instr 'OpCallNAT': unknown native function 'X'" |
| Unknown user function | "instr 'OpCallFN': unknown user function 'X'" |
| Argument count mismatch | "instr 'OpCallFN': 'X' expects N argument(s), got M" |
| Call stack overflow | "instr 'OpCallFN': call stack overflow" |
| Stack underflow | "stack underflow" |
