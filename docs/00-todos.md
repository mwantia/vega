# Open Todo's and Idea's

## ~~1. Pointer Aliases~~ (Implemented)

Pointer aliases create a slot that views an explicit byte offset in the global allocator buffer — no new allocation, just a different window into existing bytes. This enables overlapping views, type reinterpretation, and manual memory layout control.

```
x = 42           # allocates 4 bytes at some offset
y = *int(0)      # alias: views bytes [0..4) as int — same memory as x
z = *short(0)    # alias: views bytes [0..2) as short — overlaps x
```

**Semantics:**
- `*type(offset)` is valid as an assignment RHS anywhere (no block scoping required)
- Bounds checking is mandatory: `offset + SizeForTag(tag)` must fit within allocator capacity
- Overlapping aliases are encouraged (union/overlay semantics)
- Reading zeroed/freed memory returns whatever bytes are there — no safety net
- `free()` on a pointer alias is a runtime error (aliases don't own memory)

**Implementation:** The compiler detects `PointerExpression` on the RHS of assignments, compiles the offset expression, resolves the type name to a `TypeTag`, and emits `VAR_PTR slot=N tag=T`. The runtime pops the offset from the expression stack, bounds-checks against `allocator.Capacity()`, and creates a `SlotEntry` with `Alias=true`. `VAR_FREE` rejects alias slots. `VAR_STORE` and `VAR_LOAD` work unchanged — they operate on `slot.Offset` and `slot.Capacity`.

`PTR_LOAD` is a companion opcode for pointer dereferences used as expressions (without creating a named slot). It pops an offset, bounds-checks, reads the value, and pushes it — transient, no slot created.

See: [Memory Model](02-memory-model.md), [Instruction Set](04-instruction-set.md), [Compiler](05-compiler.md), [Runtime](06-runtime.md).

---

## ~~2. Union Types and Explicit Type Declarations~~ (Implemented)

Typed variable declarations with union types are now supported:

```
y: int|bool = 15
y = true          # OK — bool is in the union
y = 3.14f         # Runtime error — float not in the union
```

**Implementation:** The `Extra` byte of `VAR_ALLOC` carries a bitmask where each bit corresponds to a type tag (tags 1–8 map to bits 0–7). `VAR_STORE` checks the value's tag against the mask. `SlotEntry.Tag` tracks the current variant for `VAR_LOAD` decoding. Untyped assignments (`x = 42`) produce a single-type mask automatically.

See: [Type System](03-type-system.md), [Compiler](05-compiler.md), [Instruction Set](04-instruction-set.md), [Runtime](06-runtime.md).

---

## ~~3. Stencil Table — Structs and Tuples~~ (Implemented)

Structs and tuples are not new allocable types. They are **compile-time stencils** — layout recipes that describe how to pack existing primitives contiguously in the byte buffer. At runtime, a struct is just bytes at known offsets. No new `TypeTag` values are needed; all complexity lives in the compiler.

```
struct point {
    x: int    // 4 bytes, offset 0
    y: int    // 4 bytes, offset 4
}
// total: 8 bytes

p = point { x = 10, y = 20 }
a = p.x    // reads 4 bytes at slot.Offset + 0, decode as integer
b = p.y    // reads 4 bytes at slot.Offset + 4, decode as integer
```

Tuples are anonymous stencils with positional access:

```
t = (42, true)    // layout: [4 bytes int][1 byte bool] = 5 bytes
a = t.0           // offset 0, TagInteger
b = t.1           // offset 4, TagBoolean
```

**Implementation:** The compiler maintains a `stencils` map populated by `StructStatement` declarations and `Compiler.RegisterStencil()` (for host-registered types). Each stencil records `FieldLayout` entries (name, byte offset, type tag) and a `TotalSize`. `SymbolInfo` carries a `Stencil` pointer for struct/tuple variables.

Assignment with a `StructExpression` RHS emits `STENCIL_ALLOC` (allocates `TotalSize` bytes in one call) followed by `FIELD_STORE` per field. Tuple assignment builds an anonymous stencil from inferred element types and follows the same pattern.

Field access (`p.x`, `t.0`) compiles to `FIELD_LOAD` — the compiler resolves the field name/index to a byte offset and type tag at compile time. No field names survive to runtime.

Three opcodes: `STENCIL_ALLOC` (arg=slotID, offset=totalSize), `FIELD_STORE` (arg=slotID, offset=fieldByteOffset, extra=tag), `FIELD_LOAD` (arg=slotID, offset=fieldByteOffset, extra=tag).

Structs can be passed to user-defined functions as arguments. The caller emits `VAR_LOAD_RAW` to serialize the stencil bytes as a `RawValue`; the callee's prologue emits `LOAD_ARG_STENCIL` to deserialize and allocate a fresh stencil slot.

See: [Memory Model](02-memory-model.md), [Instruction Set](04-instruction-set.md), [Compiler](05-compiler.md), [Runtime](06-runtime.md).

---

## ~~4. Allocator Interface — Swappable Allocation Strategies~~ (Implemented)

The `Allocator` is an interface in `pkg/alloc`, allowing the runtime to support multiple allocation strategies.

### Interface

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

The runtime operates through this interface — it doesn't care which implementation backs the buffer.

### Implemented Strategies

| Strategy | Constructor | `free()` support | Snapshottable | Best for |
|----------|-------------|-----------------|---------------|----------|
| **Free list** (default) | `NewFreeListAllocator(cap)` | Yes (first-fit, coalescing) | Yes | General use, mixed alloc/free patterns |
| **Bump** | `NewBumpAllocator(cap)` | No-op | No | Short-lived sessions, write-once |
| **Pool** | `NewPoolAllocator(cap, slotSize)` | Yes (O(1)) | No | Uniform-size workloads |

The `FreeListAllocator` additionally implements the `Snapshottable` interface, enabling `SnapshotManager` to record incremental deltas for history replay.

---

## ~~ 5. Primitive Type Budget: 8 covers all current types~~

**Decision:** 8 primitive types fit a single bitmask byte and cover all fundamental data widths:

| Tag | Type    | Width    | Bit | Purpose                      |
|-----|---------|----------|-----|------------------------------|
| 1   | short   | 2        | 0   | Small signed integer         |
| 2   | int     | 4        | 1   | Standard integer             |
| 3   | long    | 8        | 2   | Large integer                |
| 4   | float   | 4        | 3   | Single-precision float       |
| 5   | decimal | 8        | 4   | Double-precision float       |
| 6   | bool    | 1        | 5   | Boolean                      |
| 7   | byte    | 1        | 6   | Raw bytes, uint8             |
| 8   | slice   | N bytes  | 7   | Bounded byte slice; `string<N>` is the text form (SizeForTag → 0, capacity external) |

**Note on `char`:** There is no `char` type tag. Single-quoted character literals (`'A'`) are lexed and then lowered by the parser to `IntegerExpression` containing the Unicode codepoint as `int32`. `'A'` is identical to `65` at the AST level.

Unsigned variants, exotic widths (float16, int128), and composite types are either unnecessary for Vega's scope or handled by stencils. `slice` uses tag 8 and bypasses the bitmask union system — it uses `SLICE_ALLOC` with a fixed declared capacity. `SizeForTag(TagSlice)` returns 0; the capacity N is stored externally in `SlotEntry.Capacity`.

---

## ~~6. Region / Arena Model for Call Frames~~ (Implemented)

### Problem

Function-local slots are currently allocated from the same global `FreeListAllocator` as top-level variables. When a function returns, `doReturn()` frees each local slot individually. Because locals are interleaved with globals in the flat buffer, freed slots leave scattered holes between live global slots — holes that cannot coalesce and accumulate over many calls.

### Design

Each call frame owns a contiguous **region**: a single contiguous slice of the global allocator buffer claimed at frame entry and released atomically at frame exit. All local allocations within the frame use a bump pointer inside this region. The global allocator is touched exactly twice per call — once to claim the region, once to release it.

```
Global buffer layout after two nested calls:

[ globals... ] [ frame-A region ] [ frame-B region ] [ free tail ]
                ^RegionOffset     ^RegionOffset
                 RegionSize ------^
```

At frame exit the entire region block is returned to the free list as one `Region{Offset, Size}` entry. It coalesces with adjacent free blocks cleanly — no scattered holes.

**Pointer aliases (`*int(offset)`) are unaffected.** Regions are a runtime memory management detail. Raw pointer offsets remain global — `*int(0)` still means byte 0 of the global buffer regardless of which frame issues it. A pointer alias created inside a function reaches global memory; if it happens to point into the frame's region, it becomes a dangling alias after the frame exits. This is intentional and consistent with the unsafe-by-design pointer model.

### Semantics

- At function entry the runtime calls `allocator.Alloc(frameSize)` once to claim the region
- `frameSize` is the sum of `SizeForTag(tag)` for all fixed-size locals — computed by the compiler and stored in `FunctionDef.FrameSize`
- Local slots are assigned offsets via a bump pointer within `[RegionOffset, RegionOffset+FrameSize)` — no free-list lookups inside the frame
- At frame exit `doReturn()` calls `allocator.Free(frame.RegionOffset, frame.RegionSize)` once — no per-slot loop
- Slice locals (`string<N>`) have a fixed declared capacity N and therefore participate in the frame region bump allocation like any scalar
- Alias slots (`Alias=true`) are never part of the frame region and are never freed by `doReturn()` — unchanged from current behaviour

### Implementation

**Compiler (`pkg/compiler/compiler.go`)**
- Add `FrameSize int` to `FunctionDef`
- During function body compilation, accumulate `totalFrameSize` by summing `value.SizeForTag(tag)` for fixed-size locals and the declared capacity for `SLICE_ALLOC` locals; store in `FunctionDef.FrameSize` after compilation

**Runtime (`pkg/vm/runtime.go`)**
- Add `RegionOffset int` and `RegionSize int` to `CallFrame`
- At call entry (after pushing the new frame): `offset, _ = r.allocator.Alloc(def.FrameSize)`, store in `frame.RegionOffset` / `frame.RegionSize`; initialise a `bumpPtr = offset`
- Fixed-size local slot allocation: assign `slot.Offset = bumpPtr`, advance `bumpPtr += slot.Capacity` (or `SizeForTag(tag)` for scalars) — no `allocator.Alloc()` call per slot
- Slice-local slot allocation: uses the declared capacity N from `SLICE_ALLOC.Offset`; participates in region bump like any other scalar
- `doReturn()`: replace the per-slot free loop with a single `r.allocator.Free(frame.RegionOffset, frame.RegionSize)`

**Top-level frame**
- The top-level (global) frame has no region — `RegionOffset = 0`, `RegionSize = 0`
- Global variables continue to use `allocator.Alloc()` directly, as today

See: [Memory Model](02-memory-model.md), [Runtime](06-runtime.md), [Compiler](05-compiler.md).

---

## 7. Typed Memory References

### Motivation

Raw pointer aliases (`*int(offset)`) use global byte offsets into the full allocator buffer. They are deliberately unsafe: no bounds checking beyond allocator capacity, no lifetime tracking, no region awareness. This is a strength — they enable union overlays, manual layout control, and cross-scope aliasing.

However, there is a complementary need: a reference that is **scoped to a region**, carries its type, and can be reasoned about more safely within a call frame. This is not a replacement for raw pointers — it is a second declaration form that coexists with them.

A typed memory reference would encode:
- Which region it belongs to (the owning frame's region)
- A byte offset within that region
- The type tag for decoding

When the owning region is freed (i.e. the frame exits), a typed reference to that region becomes invalid. Unlike a raw pointer, this invalidity can in principle be detected at runtime (by comparing the reference's region against the current live region set).

### Open Questions

- **Syntax**: not yet decided. Candidates: `&int`, `ref int`, `@int`, or a distinct declaration keyword
- **Cross-region references**: a typed reference created in an inner frame that escapes to an outer frame needs a clear lifetime story — either forbidden, or the reference is materialised (copied) on frame exit analogous to how return values are materialised today
- **Interaction with union types**: can a typed reference carry a union mask, or is it always single-type?
- **First-class vs. syntax sugar**: are typed references a new value kind in the VM, or are they lowered by the compiler to a `(region_id, offset, tag)` triple stored in a stencil slot?

### Relationship to Raw Pointers

| | Raw pointer `*int(N)` | Typed reference (TBD) |
|---|---|---|
| Offset basis | Global buffer | Region-relative |
| Type safety | None | Tag-checked |
| Lifetime tracking | None | Region-aware |
| Cross-scope access | Yes (by design) | Restricted / explicit |
| Unsafe | Yes | Partially |

This item is deliberately left open until the region model (step 6) is fully implemented and the interaction between regions and lifetimes is understood in practice.

See: [Memory Model](02-memory-model.md), [Type System](03-type-system.md).
