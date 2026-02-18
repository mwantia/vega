# Runtime

**Package:** `pkg/vm/` (files `runtime.go`, `vm.go`)

The runtime executes bytecode instructions within call frames, managing the expression stack and the byte-array variable buffer.

---

## Architecture

```go
type Runtime struct {
    Frames    []*CallFrame      // up to 256 frames
    Index     int               // current frame index
    exprStack *ExprStack        // expression stack (nil outside alloc blocks)
    allocator *alloc.Allocator  // byte buffer manager (nil outside alloc blocks)
    slots     []SlotEntry       // variable slot table (nil outside alloc blocks)
}
```

The `exprStack`, `allocator`, and `slots` are created together by `STACK_ALLOC` and destroyed together by `STACK_FREE`. Between those instructions, all three are non-nil.

---

## Expression Stack

```go
type ExprStack struct {
    data []value.Value
}
```

A plain `[]value.Value` with no capacity tracking. The expression stack is used for:

- Pushing constants (`LOAD_CONST`)
- Pushing decoded variable values (`VAR_LOAD`)
- Popping values for storage (`VAR_STORE`)
- Discarding expression results (`STACK_POP`)

The byte budget is enforced solely by the allocator. The expression stack has no size limit — it holds Go-managed `Value` interfaces that are transient (pushed and popped within a single expression evaluation).

The design cleanly separates concerns: the expression stack holds transient values (Go-managed), the allocator holds persistent variables (byte-managed).

---

## Call Frames

```go
type CallFrame struct {
    ByteCode           *compiler.ByteCode
    InstructionPointer int
    BasePointer        int
}
```

Call frames no longer have a `Locals` map. Named variables are stored in the byte buffer, not in a Go map. The `BasePointer` remains for future function call support.

---

## Slot Table

```go
type SlotEntry struct {
    Offset  int           // byte offset into the allocator buffer
    Size    int           // number of bytes occupied
    Tag     value.TypeTag  // current variant tag (0 = uninitialized)
    Mask    byte          // bitmask of allowed type tags
    Alive   bool          // false after VAR_FREE
    Alias   bool          // true = manually positioned pointer, not allocator-owned
    Stencil bool          // true = stencil-based allocation (struct/tuple)
}
```

The slot table is a `[]SlotEntry` indexed by slot ID. It grows dynamically as `VAR_ALLOC`, `VAR_PTR`, and `STENCIL_ALLOC` instructions are executed (the compiler guarantees sequential slot IDs starting from 0).

- `Mask` stores the set of allowed types as a bitmask (set by `VAR_ALLOC` or `VAR_PTR`). For stencil slots, `Mask=0`.
- `Tag` tracks the **current** variant — updated on every `VAR_STORE`, used by `VAR_LOAD` for decoding. Starts at 0 (uninitialized) until the first store. For alias slots, `Tag` is set immediately by `VAR_PTR`. For stencil slots, `Tag=0` since type checking is per-field.
- `Alias` marks pointer alias slots. These point to explicit offsets and cannot be freed via `VAR_FREE`.
- `Stencil` marks struct/tuple slots. These hold multiple fields at known offsets, accessed via `FIELD_LOAD` and `FIELD_STORE`.

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
        // ... fetch instruction, advance IP, execute
    }
}
```

### Handler Summary

| Instruction | Stack effect | Allocator effect |
|-------------|-------------|------------------|
| `STACK_ALLOC N` | Create stack | Create allocator(N) + slot table |
| `STACK_FREE` | Destroy stack | Destroy allocator + slot table |
| `LOAD_CONST i` | Push `constants[i]` | — |
| `STACK_POP` | Pop + discard | — |
| `STACK_DUP` | Push copy of top | — |
| `VAR_ALLOC slot=S mask=M` | — | `Alloc(MaxSizeForMask(M))`, record in `slots[S]` with `Tag=0` |
| `VAR_STORE slot=S` | Pop value | Encode + `Write(offset, bytes)` |
| `VAR_LOAD slot=S` | Push decoded value | `Read(offset, size)` + Decode |
| `VAR_FREE slot=S` | — | `Free(offset, size)`, mark dead (rejects aliases) |
| `VAR_PTR slot=S tag=T` | Pop offset | Bounds-check, create alias slot at offset |
| `STENCIL_ALLOC slot=S size=N` | — | `Alloc(N)`, record in `slots[S]` with `Stencil=true` |
| `FIELD_STORE slot=S off=O tag=T` | Pop value | Type-check tag, write into `buffer[slot.Offset+O]` |
| `FIELD_LOAD slot=S off=O tag=T` | Push decoded value | Read `buffer[slot.Offset+O]`, wrap as value |

### VAR_STORE Detail

1. Pop the top value from the expression stack.
2. Assert it implements `Allocable` (strings and nil cannot be stored).
3. Assert `TagInMask(TagFor(value), slot.Mask)` — the value's type must be in the allowed set.
4. Update `slot.Tag` to the value's actual tag (via pointer to avoid copy: `&r.slots[slotID]`).
5. Get the destination slice via `allocator.Slice(slot.Offset, slot.Size)`.
6. Copy `alloc.View()` bytes into the destination, zeroing any remaining bytes.

### VAR_LOAD Detail

1. Assert `slot.Tag != 0` — the slot must have been stored to at least once.
2. Compute `typeSize = SizeForTag(slot.Tag)`.
3. Get a view into the buffer: `allocator.Slice(slot.Offset, typeSize)`.
4. Wrap the view as a value via `value.Wrap(slot.Tag, view)` — the value reads directly from the alloc buffer (no copy).
5. Push the value onto the expression stack.

### VAR_PTR Detail

1. Pop the top value from the expression stack (the offset).
2. Assert the value is `Allocable` and extract an integer via `value.ToInt()`.
3. Compute `size = SizeForTag(tag)` from the instruction's `Extra` byte.
4. Bounds-check: `offset + size <= allocator.Capacity()`.
5. Create a `SlotEntry` with `{Offset: offset, Size: size, Tag: tag, Mask: MaskForTag(tag), Alive: true, Alias: true}`.

No allocator call is made. The alias slot views whatever bytes are at the specified offset. `VAR_STORE` and `VAR_LOAD` operate on alias slots identically to regular slots — they just read/write at `slot.Offset`.

### VAR_FREE and Aliases

Before freeing, `VAR_FREE` checks `slot.Alias`. If true, it returns an error — alias slots don't own their memory and cannot return it to the free list.

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
| Store non-allocable | "instr 'OpVarSTORE': value type X is not allocable" |
| Type mismatch | "instr 'OpVarSTORE': type mismatch: slot mask XXXXXXXX does not allow tag Y" |
| Load uninitialized | "instr 'OpVarLOAD': slot N is uninitialized" |
| Use after free | "instr 'OpVarLOAD': use after free on slot N" |
| Double free | "instr 'OpVarFREE': double free on slot N" |
| Free pointer alias | "instr 'OpVarFREE': cannot free pointer alias on slot N" |
| Pointer out of bounds | "instr 'OpVarPTR': pointer out of bounds (offset=N, size=M, capacity=C)" |
| Pointer offset not allocable | "instr 'OpVarPTR': offset value is not allocable" |
| No allocator for pointer | "instr 'OpVarPTR': no allocator active" |
| Stencil out of memory | "instr 'OpStencilALLOC': out of memory: ..." |
| No allocator for stencil | "instr 'OpStencilALLOC': no allocator active" |
| Field store to dead slot | "instr 'OpFieldSTORE': slot N is not alive" |
| Field store type mismatch | "instr 'OpFieldSTORE': type mismatch: expected tag X, got Y" |
| Field store non-allocable | "instr 'OpFieldSTORE': value is not allocable" |
| Field load from dead slot | "instr 'OpFieldLOAD': slot N is not alive" |
| Stack overflow (expression) | — (not enforced; Go manages the slice) |
| Stack underflow | "stack underflow" |
| Undefined stack | "undefined stack" |

---

## VM

```go
type VM struct {
    mu     sync.RWMutex
    fs     vfs.VirtualFileSystem
    stdin  io.Reader
    stdout io.Writer
    stderr io.Writer
}
```

The VM is the top-level entry point. `Run()` creates a `Runtime` with a single call frame and executes the bytecode. The VM is thread-safe via `sync.RWMutex`. The `fs` field holds the VFS instance for filesystem operations.

```go
func (v *VM) Run(ctx context.Context, bytecode *compiler.ByteCode) (int, error)
```

Returns `(0, nil)` on success, `(1, error)` on failure.
