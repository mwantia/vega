# Memory Model

## Two-Region Design

Vega's runtime splits memory into two distinct regions within each `alloc` block:

| Region | Backing | Stores | Lifetime |
|--------|---------|--------|----------|
| **Expression stack** | `[]value.Value` (Go slice) | Temporaries during expression evaluation | Push/pop per expression |
| **Variable buffer** | `[]byte` (flat byte array) | Named variables as raw bytes | Explicit via `free()` or block exit |

The expression stack is unbounded and managed by Go's garbage collector. The variable buffer is a fixed-capacity byte array managed by a free list allocator — no GC involvement, no heap allocation per variable.

**Key invariant:** `alloc <N>` guarantees exactly `N` bytes for variable data. All named variable data lives exclusively in the alloc buffer, and the alloc capacity is the hard ceiling.

---

## Values as Views

All allocable runtime values implement the `Allocable` interface and hold a `view []byte` — a Go slice pointing into either the alloc buffer or the constants table. Values **do not own their data**; they are lightweight handles for reading and writing through the backing memory.

```go
// A CharValue does not store a rune — it reads one from the alloc buffer.
type CharValue struct {
    view []byte  // points into allocator.buffer[offset:offset+4]
}

func (v *CharValue) Data() rune {
    return rune(binary.LittleEndian.Uint32(v.view))
}
```

Since Go slices are reference types (pointer + length + capacity), creating a value from `allocator.Slice(offset, size)` shares the underlying array — no bytes are copied.

### Wrap Factory

**Package:** `pkg/value`

`Wrap(tag TypeTag, view []byte) (Allocable, error)` creates the correct value type for a given tag, wrapping the provided byte slice. Used by the VM during `OpLoadCONST` and `OpVarLOAD`.

### Where Data Lives

| Source | Backing store | Copy on creation? |
|--------|--------------|-------------------|
| `OpLoadCONST` | `ByteCode.Constants[i].Data` | No — view into constants table |
| `OpVarLOAD` | `allocator.buffer[offset:offset+size]` | No — view into alloc buffer |
| `OpVarSTORE` | copies view bytes → alloc buffer | Yes — one copy from source into slot |
| Expression temporaries (e.g. `x + y`) | Transient `[]byte` on Go heap | Yes — short-lived, discarded after store or pop |

The only copy point is `OpVarSTORE`, which transfers bytes from the source value's view into the alloc buffer. This is unavoidable — the data must move from wherever it originated (constants table, another slot, or a temporary) into the variable's allocated region.

---

## The `alloc` Block

Every variable must live inside an `alloc` block that declares the byte budget:

```
alloc <capacity> {
    <statements>
}
```

`capacity` is the number of bytes available for named variables. It must be an integer literal. When the block exits, all memory is released.

### What `alloc` Creates

At runtime, entering an `alloc` block creates three things:

1. **Expression stack** — a `[]value.Value` for push/pop temporaries.
2. **Allocator** — a `[]byte` of the declared capacity with a free list covering the full region.
3. **Slot table** — a `[]SlotEntry` mapping slot IDs to `{offset, size, tag, alive}`.

All three are destroyed when the block exits (`STACK_FREE`).

---

## Free List Allocator

**Package:** `pkg/alloc`

The allocator manages a contiguous `[]byte` buffer using a sorted free list.

### Data Structures

```go
type FreeBlock struct {
    Offset int
    Size   int
}

type Allocator struct {
    buffer   []byte
    freeList []FreeBlock  // sorted by offset
}
```

### Operations

| Method | Description |
|--------|-------------|
| `NewAllocator(capacity)` | Creates a buffer with one free block spanning the full capacity |
| `Alloc(size) (offset, error)` | First-fit: walks the free list, finds the first block >= size, splits if larger |
| `Free(offset, size)` | Zeros the memory, inserts into the free list, coalesces with neighbors |
| `Slice(offset, size) []byte` | Returns a writable sub-slice view of the buffer (no copy) |
| `Write(offset, data)` | Copies bytes into the buffer at the given offset |
| `Read(offset, size) []byte` | Returns a slice view of the buffer |
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

```
Before free(0, 4):
Buffer: [████████░░░░░░░░]
Free list: [{offset:8, size:8}]

After free(0, 4) + coalesce:
Buffer: [░░░░░░░░░░░░░░░░]
Free list: [{offset:0, size:16}]  ← merged with right neighbor
```

Three coalesce cases:
- **Right only** — new block's end touches right neighbor's start.
- **Left only** — left neighbor's end touches new block's start.
- **Both** — new block bridges two existing free blocks into one.

### Out-of-Memory

If no free block can satisfy a request, `Alloc` returns an error:

```
alloc 4 { x = 42l }  # Error: out of memory: need 8 bytes, have 4 free
```

This is a runtime error, not a compile-time error, because the allocator capacity is evaluated at runtime.

---

## Slot Table

The slot table maps compile-time slot IDs to runtime byte regions:

```go
type SlotEntry struct {
    Offset  int           // byte offset into allocator buffer
    Size    int           // number of bytes (max across all allowed types)
    Tag     value.TypeTag  // current variant tag (0 = uninitialized)
    Mask    byte          // bitmask of allowed type tags
    Alive   bool          // false after free()
    Alias   bool          // true = manually positioned pointer, not allocator-owned
    Stencil bool          // true = stencil-based allocation (struct/tuple)
}
```

Slot IDs are assigned sequentially by the compiler (0, 1, 2, ...). At runtime, `OpVarALLOC` populates a slot entry with `Tag=0` and the bitmask from the instruction's `Extra` byte; `OpVarSTORE` updates `Tag` to the current variant; `OpVarFREE` marks it dead. `OpVarPTR` creates alias slots with `Alias=true` — these point to explicit offsets and cannot be freed. `OpStencilALLOC` creates stencil slots with `Stencil=true` — these hold multiple fields accessed via `OpFieldLOAD`/`OpFieldSTORE`.

### Safety Checks

| Condition | Error |
|-----------|-------|
| `OpVarLOAD` on a dead slot | "use after free on slot N" |
| `OpVarLOAD` on uninitialized slot | "slot N is uninitialized" |
| `OpVarFREE` on a dead slot | "double free on slot N" |
| `OpVarFREE` on an alias slot | "cannot free pointer alias on slot N" |
| `OpVarSTORE` with type not in mask | "type mismatch: slot mask XXXXXXXX does not allow tag Y" |
| `OpVarPTR` with offset out of bounds | "pointer out of bounds (offset=N, size=M, capacity=C)" |

---

## Variable Lifecycle

```
alloc 64 {
    x = 42;        # 1. Compiler assigns slot 0, infers TagInteger, mask=00000010
                   # 2. VAR_ALLOC: allocator reserves 4 bytes, slot table records {offset, 4, tag=0, mask, true}
                   # 3. VAR_STORE: pop value, check tag in mask, copy view bytes into alloc buffer

    x;             # 4. VAR_LOAD: wrap allocator.Slice(offset, 4) as IntegerValue, push onto expr stack
                   #    (no copy — the value reads directly from the alloc buffer)

    free(x);       # 5. VAR_FREE: return 4 bytes to free list, mark slot as dead
                   # 6. Compiler removes 'x' from symbol table — subsequent references are compile errors

}                  # 7. STACK_FREE: destroy allocator, slot table, and expression stack
```

---

## Pointer Aliases

A pointer alias creates a slot that views an explicit byte offset in the allocator buffer. Unlike `OpVarALLOC`, which asks the allocator for a fresh region, `OpVarPTR` skips the allocator entirely and creates a slot at a user-specified offset.

### Syntax

```
y = *int(0)     # view bytes [0..4) as int
z = *short(2)   # view bytes [2..4) as short
```

The syntax is `*type(offset)` where `type` is a valid type name (`int`, `short`, `long`, `float`, `decimal`, `bool`, `byte`, `char`) and `offset` is an expression evaluating to a non-negative integer.

### Overlapping Views

Pointer aliases deliberately support overlapping regions. Multiple aliases can view the same bytes with different types:

```
alloc 8 {
    x = 42              # allocator assigns offset 0, 4 bytes
    y = *int(0)         # alias: views same bytes [0..4) as int
    z = *short(0)       # alias: views first 2 bytes [0..2) as short
}
```

Writing through `x` or `y` updates the same memory. Reading `z` returns whatever the first 2 bytes decode to as a `short`. This is union/overlay semantics — the programmer gets whatever bytes are there, reinterpreted through the alias type.

### Bounds Checking

The runtime validates `offset + SizeForTag(tag) <= allocator.Capacity()` before creating the alias. Out-of-bounds pointers are a runtime error:

```
alloc 8 {
    y = *int(6)    # Error: pointer out of bounds (offset=6, size=4, capacity=8)
}
```

### Lifecycle

- Alias slots have `Alias=true` in the slot table.
- `free()` on an alias is a runtime error — aliases don't own their memory.
- `OpVarSTORE` and `OpVarLOAD` work identically for aliases and regular slots — they just read/write at `slot.Offset`.
- When the `alloc` block exits (`STACK_FREE`), all slots including aliases are destroyed.

### Variable Lifecycle with Pointer Alias

```
alloc 8 {
    x = 42;         # 1. VAR_ALLOC: allocator reserves 4 bytes at offset 0
                     # 2. VAR_STORE: writes 42 into buffer[0..4)

    y = *int(0);    # 3. LOAD_CONST 0, then VAR_PTR: creates alias slot at offset 0
                     #    (no allocator call — just records {offset=0, size=4, tag=int, alias=true})

    z = y;          # 4. VAR_LOAD slot=1: reads buffer[0..4) as int → gets 42
                     # 5. VAR_STORE slot=2: writes 42 into z's region

    free(y);        # Runtime error: cannot free pointer alias
}
```

---

## What This Means for the Programmer

- Variables have a **constrained type** determined at first assignment. By default, the type is fixed to the inferred type. With explicit type declarations (`x: int|bool = 42`), a variable can hold any of the listed types.
- Variables consume a **known number of bytes** from the alloc budget. Overflow is a runtime error.
- `free()` is explicit and immediate. The bytes are available for reuse by subsequent allocations.
- Strings are **excluded** from the byte array. They live in the constants table as Go strings.
- **Pointer aliases** (`*type(offset)`) let you create overlapping views into the buffer for type reinterpretation and manual layout control. Aliases don't allocate — they just view existing bytes.
- **Structs and tuples** are compile-time stencils — layout recipes describing how to pack primitives contiguously. A struct allocates a single contiguous region; field access is resolved to byte offsets at compile time. No new type tags are needed.
- There is no garbage collector. Memory management is manual and deterministic.

---

## Scoped Arena: Stack, Heap, or Both?

Vega's `alloc` block is not purely a stack frame, not purely a heap region, and not just a flat byte array. It is a **scoped arena with an internal free list** — a bounded memory region with stack-like lifetime and heap-like allocation semantics inside.

### How a Traditional Stack Works

A call stack is a contiguous region where allocations follow strict LIFO discipline:

```
┌──────────────┐ ← stack pointer (grows downward)
│ z: bool (1B) │  offset -13
│ y: f64  (8B) │  offset -12
│ x: i32  (4B) │  offset -4
├──────────────┤ ← base pointer (frame start)
│ return addr  │
└──────────────┘
```

- **Offsets are compile-time constants** — `x` is always at `rbp-4`, no runtime lookup
- **No individual deallocation** — you can't free `x` while keeping `y`; the whole frame pops at once
- **No fragmentation** — everything is contiguous, freed in reverse order
- **Allocation is O(1)** — just decrement the stack pointer

### How a Traditional Heap Works

The heap is a large region managed by an allocator (`malloc`/`free`):

```
┌─────────┬──────┬─────────┬──────────────┐
│ x (4B)  │ free │ y (8B)  │    free      │
└─────────┴──────┴─────────┴──────────────┘
```

- **Offsets are runtime-determined** — the allocator finds space
- **Individual deallocation** — free any object at any time
- **Fragmentation** — free/alloc patterns create holes
- **Allocation is O(n)** — free list traversal, coalescing
- **Unbounded lifetime** — lives until explicitly freed or process exits

### What Vega's `alloc` Block Actually Is

The arena itself has stack semantics — created on block entry, destroyed on block exit, fixed capacity. The allocations *inside* it have heap semantics — arbitrary order, explicit free, fragmentation, coalescing:

```
alloc 64 {          ← arena creation (stack-like: scoped, bounded)
    x = 42          ← sub-allocation within arena (heap-like: free list, runtime offset)
    free(x)         ← individual deallocation (heap-like)
    y = 100l        ← reuses freed space (heap-like: coalescing)
}                   ← arena destruction (stack-like: everything dies at once)
```

### Property Comparison

| Property | Stack | Heap | Vega |
|----------|-------|------|------|
| Region lifetime | Function scope | Manual / GC | Block scope |
| Region size | Compiler-determined | Unbounded (OS pages) | Programmer-declared |
| Allocation offsets | Compile-time constants | Runtime (allocator) | Runtime (free list) |
| Individual free | No | Yes | Yes |
| Fragmentation | Impossible | Yes | Yes (within the arena) |
| Coalescing | N/A | Allocator-dependent | Yes |
| Slot identity | Stack offset | Pointer | Slot ID → offset |

### Where Vega Is Stack-Like

1. **Scoped lifetime** — the buffer is born and dies with the `alloc` block, just like a stack frame is born and dies with a function call.
2. **Fixed capacity** — declared upfront, no growth. Like a stack frame's size being known at compile time.
3. **Slot IDs are sequential integers** — assigned at compile time (0, 1, 2, ...), like how a C compiler assigns stack offsets.

### Where Vega Is Heap-Like

1. **Runtime offsets** — the free list decides *where* in the buffer a variable lands.
2. **Individual deallocation** — `free(x)` returns bytes mid-block.
3. **Reuse** — freed space is immediately available for new allocations.
4. **Fragmentation** — freeing a 4-byte slot between two live variables creates a 4-byte hole.

### Where Vega Is Neither

Pointer aliases (`*int(0)`) are something neither the stack nor the traditional heap offers directly. They are closer to C's `union` or pointer arithmetic — manual control over which bytes mean what. The stack doesn't let you reinterpret memory; the heap allocator doesn't let you choose your offset.

### The Closest Existing Concept

The closest analog is **region-based memory management** — a pattern used in arena allocators:

1. Allocate a region of known size.
2. Sub-allocate within it (with or without a free list).
3. Destroy the entire region when done.

Rust's `typed-arena` crate, game engine frame allocators, and compiler scratch arenas all follow this pattern. Vega formalizes it as a **language-level construct** rather than a library pattern — the `alloc` block is syntax, not an API call.
