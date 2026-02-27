# Memory Model

## Two-Region Design

Vega's runtime splits memory into two distinct regions:

| Region | Backing | Stores | Lifetime |
|--------|---------|--------|----------|
| **Expression stack** | `[]value.Value` (Go slice) | Temporaries during expression evaluation | Push/pop per expression |
| **Variable buffer** | `[]byte` (flat byte array) | Named variables as raw bytes | Explicit via `free()` or session end |

The expression stack is unbounded and managed by Go's garbage collector. The variable buffer is a fixed-capacity byte array managed by a free-list allocator — no GC involvement, no heap allocation per variable.

Both regions are per-scope for function calls: when a function is entered, the caller's expression stack and slot table are saved; when the function returns, they are restored. The underlying **allocator is global** — shared across all function calls and all REPL commands within a session.

---

## Values as Views

All fixed-size runtime values implement the `Allocable` interface and hold a `view []byte` — a Go slice pointing into either the alloc buffer or the constants table. Values **do not own their data**; they are lightweight handles for reading and writing through the backing memory.

Since Go slices are reference types (pointer + length + capacity), creating a value from `allocator.Slice(offset, size)` shares the underlying array — no bytes are copied.

### Where Data Lives

| Source | Backing store | Copy on creation? |
|--------|--------------|-------------------|
| `LOAD_CONST` | `ByteCode.Constants[i].Data` | No — view into constants table |
| `VAR_LOAD` | `allocator.buffer[offset:offset+size]` | No — view into alloc buffer |
| `VAR_STORE` | copies view bytes → alloc buffer | Yes — one copy from source into slot |
| `VAR_STORE` (slice) | copies content bytes → alloc buffer, zero-pads remainder | Yes — content copied up to capacity |
| Expression temporaries (e.g. arithmetic results) | Transient `[]byte` on Go heap | Yes — short-lived, discarded after store or pop |

The only copy points are `VAR_STORE` instructions, which transfer bytes from wherever they originated (constants table, another slot, or a temporary) into the variable's allocated region.

---

## Global Allocator

**Package:** `pkg/alloc`

The VM initializes a single global allocator when it is created:

```go
const DefaultAllocSize = 1024 * 1024 // 1 MiB

allocator := alloc.NewFreeListAllocator(DefaultAllocSize)
```

This allocator backs all variables for the lifetime of the session. There is no per-function or per-block buffer — all scopes share the same 1 MiB byte pool.

### The `Allocator` Interface

```go
type Allocator interface {
    Alloc(size int) (int, error)
    Free(offset, size int)
    Slice(offset, size int) []byte
    Write(offset int, data []byte)
    Read(offset, size int) []byte
    Capacity() int
    FreeSpace() int
}
```

Three implementations are provided:

| Strategy | Constructor | `Free()` | Best for |
|----------|-------------|---------|----------|
| **Free list** (default) | `NewFreeListAllocator(cap)` | First-fit with coalescing | General use, mixed alloc/free |
| **Bump** | `NewBumpAllocator(cap)` | No-op (no reclaim) | Short-lived, write-once sessions |
| **Pool** | `NewPoolAllocator(cap, slotSize)` | O(1) slot reclaim | Uniform-size workloads |

Only `FreeListAllocator` implements the `Snapshottable` interface and is therefore compatible with `SnapshotManager`.

### Free List Operations

| Method | Description |
|--------|-------------|
| `NewFreeListAllocator(capacity)` | Creates a buffer with one free block spanning the full capacity |
| `Alloc(size) (offset, error)` | First-fit: walks the free list, finds the first block ≥ size, splits if larger |
| `Free(offset, size)` | Zeros the memory, inserts into the free list, coalesces with neighbors |
| `Slice(offset, size) []byte` | Returns a writable sub-slice view of the buffer (no copy) |
| `Capacity() int` | Total size of the backing buffer |
| `FreeSpace() int` | Sum of all free block sizes |

### Allocation Strategy: First-Fit

The allocator walks the free list from lowest offset to highest and returns the first block large enough. If the block is larger than needed, it splits — the allocated portion is removed and the remainder stays in the free list.

```
Buffer: [████░░░░░░░░░░░░]  (16 bytes, 4 used, 12 free)
Free list: [{offset:4, size:12}]

Alloc(4) → offset=4
Buffer: [████████░░░░░░░░]
Free list: [{offset:8, size:8}]
```

### Deallocation with Coalescing

When `Free()` is called, the freed region is:

1. **Zeroed** — all bytes set to 0.
2. **Inserted** into the free list at the correct position (sorted by offset).
3. **Coalesced** — if the new free block is adjacent to an existing free block on either side, they merge into a single larger block.

Three coalesce cases:
- **Right only** — new block's end touches right neighbor's start.
- **Left only** — left neighbor's end touches new block's start.
- **Both** — new block bridges two existing free blocks into one.

### Out-of-Memory

If no free block can satisfy a request, `Alloc` returns an error:

```
x = 42l    # needs 8 bytes
# Error (if allocator is nearly exhausted):
# out of memory: need 8 bytes, have N free
```

---

## Slot Table

The slot table maps compile-time slot IDs to runtime byte regions:

```go
type SlotEntry struct {
    Offset   int           // byte offset into allocator buffer
    Capacity int           // number of bytes allocated (max across union types; declared capacity for slices)
    Tag      value.TypeTag  // current variant tag (0 = uninitialized)
    Mask     byte          // bitmask of allowed type tags (0 for stencils and slices)
    Alive    bool          // false after free()
    Alias    bool          // true = manually positioned pointer, not allocator-owned
    Stencil  bool          // true = stencil-based allocation (struct/tuple)
}
```

Slot IDs are assigned sequentially by the compiler (0, 1, 2, ...). `VAR_ALLOC` populates a slot entry with `Tag=0` (uninitialized until first `VAR_STORE`); `VAR_FREE` marks it dead. `VAR_PTR` creates alias slots with `Alias=true`. `STENCIL_ALLOC` creates stencil slots with `Stencil=true`. `SLICE_ALLOC` creates slice slots with a fixed `Capacity` equal to the declared `<N>` value.

### Safety Checks

| Condition | Error |
|-----------|-------|
| `VAR_LOAD` on a dead slot | "use after free on slot N" |
| `VAR_LOAD` on uninitialized slot | "slot N is uninitialized" |
| `VAR_FREE` on a dead slot | "double free on slot N" |
| `VAR_FREE` on an alias slot | "cannot free pointer alias on slot N" |
| `VAR_STORE` with type not in mask | "type mismatch: slot mask XXXXXXXX does not allow tag Y" |
| `VAR_PTR` with offset out of bounds | "pointer out of bounds (offset=N, size=M, capacity=C)" |

---

## Variable Lifecycle

```
x = 42          # 1. Compiler assigns slot 0, infers TagInteger, mask=00000010
                # 2. VAR_ALLOC: allocator reserves 4 bytes, slot table records {offset, 4, tag=0, mask, true}
                # 3. VAR_STORE: pop value, check tag in mask, copy view bytes into alloc buffer

x               # (bare identifier is not valid syntax — only in REPL or expression context)
                # 4. VAR_LOAD: wrap allocator.Slice(offset, 4) as IntegerValue, push onto expr stack
                #    (no copy — the value reads directly from the alloc buffer)

free(x)         # 5. VAR_FREE: return 4 bytes to free list, mark slot as dead
                # 6. Compiler removes 'x' from symbol table — subsequent references are compile errors
```

---

## Slice Storage

Slices have a **fixed declared capacity** and use `SLICE_ALLOC` rather than the mask-based `VAR_ALLOC` path. `string<N>` is the text form of a bounded byte slice; `byte<N>` is the raw-byte form. Both share the same `TagSlice` tag.

```
greeting: string<10> = "hello"
```

1. `LOAD_CONST` pushes a `SliceValue` backed by the constant pool.
2. `SLICE_ALLOC slot=0 capacity=10`: allocates exactly 10 bytes from the global allocator. The slot's `Tag` is set to `TagSlice` and `Capacity` is set to 10.
3. `VAR_STORE slot=0`: pops the `SliceValue`, checks that `len(content) <= slot.Capacity` (runtime error on overflow), copies content bytes, and zero-pads the remainder.

Content ends at the first `\0` byte within the capacity region, or at the capacity boundary if no `\0` is present. Reading the slot returns only the effective (non-null) prefix. The slot mask is 0 (slice slots bypass type-mask checking and cannot participate in union types).

An untyped string literal (`msg = "hello"` without a `string<N>` constraint) is a **compile error** — the capacity must always be declared explicitly.

---

## Function Scoping

The global allocator is shared, but the expression stack and slot table are per-function. On `CALL_FN`:

1. The caller's `exprStack` and `slots` are saved inside the current `CallFrame`.
2. Fresh `exprStack` and `slots` are created for the callee.
3. Function arguments are passed via `pendingArgs`.

On `RETURN` (or implicit return at end of function):

1. The return value (if any) is **materialized** — copied out of the allocator into a fresh `[]byte` so it does not point into memory that is about to be freed.
2. All alive, non-alias slots in the callee's scope are freed back to the global allocator.
3. The caller's `exprStack` and `slots` are restored.
4. The materialized return value is pushed onto the caller's stack (if the call site expected a value).

This means function-local variables automatically return their memory to the global pool on return, without any GC involvement.

---

## Pointer Aliases

A pointer alias creates a slot that views an explicit byte offset in the global allocator buffer. Unlike `VAR_ALLOC`, which asks the allocator for a fresh region, `VAR_PTR` skips the allocator entirely and creates a slot at a user-specified offset.

### Syntax

```
y = *int(0)     # view bytes [0..4) as int
z = *short(2)   # view bytes [2..4) as short
```

The syntax is `*type(offset)` where `type` is a valid type name (`int`, `short`, `long`, `float`, `decimal`, `bool`, `byte`, `string`) and `offset` is an expression evaluating to a non-negative integer.

### Bounds Checking

The runtime validates `offset + SizeForTag(tag) <= allocator.Capacity()` before creating the alias. Out-of-bounds pointers are a runtime error.

### Lifecycle

- Alias slots have `Alias=true` in the slot table.
- `free()` on an alias is a runtime error — aliases don't own their memory.
- `VAR_STORE` and `VAR_LOAD` work identically for aliases and regular slots.
- When a function returns, alias slots are not freed (they don't own their memory); non-alias slots are freed.

---

## Session Persistence

The `Session` object wraps the allocator and slot table:

```go
type Session struct {
    allocator alloc.Allocator
    slots     []SlotEntry
    funcs     map[string]*compiler.FunctionDef
    snapshots *alloc.SnapshotManager
}
```

In REPL mode, the same `Session` is reused across `Run()` calls:

- Variables defined in one command are visible in subsequent commands.
- Functions defined in one command can be called from subsequent commands.
- `ResetSession()` replaces the session with a fresh one, erasing all state.
- `SnapshotManager` (available when using `FreeListAllocator`) records incremental deltas for history replay and TUI visualization.

---

## What This Means for the Programmer

- Variables have a **constrained type** determined at first assignment. By default the type is fixed to the inferred type. With explicit type declarations (`x: int|bool = 42`), a variable can hold any of the listed types.
- Variables consume a **known number of bytes** from the global 1 MiB pool. If the pool is exhausted, the next allocation fails with a runtime error.
- `free()` is explicit and immediate. The bytes are available for reuse by subsequent allocations.
- **Slices** (`string<N>`, `byte<N>`) live in the allocator via `SLICE_ALLOC`. The capacity N is fixed at declaration; storing content longer than N is a runtime error. Content is null-terminated within the capacity region.
- **Pointer aliases** (`*type(offset)`) create overlapping views into the buffer for type reinterpretation and manual layout control. Aliases don't allocate — they just view existing bytes.
- **Structs and tuples** are compile-time stencils — layout recipes describing how to pack primitives contiguously. A struct allocates a single contiguous region; field access is resolved to byte offsets at compile time. Slice fields (`string<N>`) are fully supported since their size is fixed.
- **Function locals** are automatically freed on return.
- There is no garbage collector. Memory management is manual and deterministic.
