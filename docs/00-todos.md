# Open Todo's and Idea's

## ~~Pointer Aliases~~ (Implemented)

Pointer aliases create a slot that views an explicit byte offset in the allocator buffer — no new allocation, just a different window into existing bytes. This enables overlapping views, type reinterpretation, and manual memory layout control.

```
alloc 8 {
    x = 42           # allocates 4 bytes at offset 0
    y = *int(0)      # alias: views bytes [0..4) as int — same memory as x
    z = *short(0)    # alias: views bytes [0..2) as short — overlaps x
}
```

**Semantics:**
- `*type(offset)` only valid as assignment RHS inside `alloc` blocks
- Bounds checking is mandatory: `offset + SizeForTag(tag)` must fit within allocator capacity
- Overlapping aliases are encouraged (union/overlay semantics)
- Reading zeroed/freed memory returns whatever bytes are there — no safety net
- `free()` on a pointer alias is a runtime error (aliases don't own memory)

**Implementation:** The compiler detects `PointerExpression` on the RHS of assignments, compiles the offset expression, resolves the type name to a `TypeTag`, and emits `OpVarPTR slot=N tag=T`. The runtime pops the offset from the expression stack, bounds-checks against `allocator.Capacity()`, and creates a `SlotEntry` with `Alias=true`. `OpVarFREE` rejects alias slots. `OpVarSTORE` and `OpVarLOAD` work unchanged — they operate on `slot.Offset` and `slot.Size`.

See: [Memory Model](02-memory-model.md), [Instruction Set](04-instruction-set.md), [Compiler](05-compiler.md), [Runtime](06-runtime.md).

---

## ~~Union Types and Explicit Type Declarations~~ (Implemented)

Typed variable declarations with union types are now supported:

```
y: int|bool = 15
y = true          # OK — bool is in the union
y = 3.14f         # Runtime error — float not in the union
```

**Implementation:** The `Extra` byte of `OpVarALLOC` carries a bitmask where each bit corresponds to a type tag (tags 1–8 map to bits 0–7). `OpVarSTORE` checks the value's tag against the mask. `SlotEntry.Tag` tracks the current variant for `OpVarLOAD` decoding. Untyped assignments (`x = 42`) produce a single-type mask automatically.

See: [Type System](03-type-system.md), [Compiler](05-compiler.md), [Instruction Set](04-instruction-set.md), [Runtime](06-runtime.md).

---

## ~~Stencil Table — Structs and Tuples~~ (Implemented)

Structs and tuples are not new allocable types. They are **compile-time stencils** — layout recipes that describe how to pack existing primitives contiguously in the byte buffer. At runtime, a struct is just bytes at known offsets. No new `TypeTag` values are needed; all complexity lives in the compiler.

```
struct point {
    x: int    // 4 bytes, offset 0
    y: int    // 4 bytes, offset 4
}
// total: 8 bytes

alloc 32 {
    p = point { x = 10, y = 20 }
    a = p.x    // reads 4 bytes at slot.Offset + 0, decode as integer
    b = p.y    // reads 4 bytes at slot.Offset + 4, decode as integer
}
```

Tuples are anonymous stencils with positional access:

```
alloc 16 {
    t = (42, true)    // layout: [4 bytes int][1 byte bool] = 5 bytes
    a = t.0           // offset 0, TagInteger
    b = t.1           // offset 4, TagBoolean
}
```

**Implementation:** The compiler maintains a `stencils` map populated by `StructStatement` declarations. Each stencil records `FieldLayout` entries (name, byte offset, type tag) and a `TotalSize`. `SymbolInfo` carries a `Stencil` pointer for struct/tuple variables.

Assignment with a `StructExpression` RHS emits `OpStencilALLOC` (allocates `TotalSize` bytes in one call) followed by `OpFieldSTORE` per field. Tuple assignment builds an anonymous stencil from inferred element types and follows the same pattern.

Field access (`p.x`, `t.0`) compiles to `OpFieldLOAD` — the compiler resolves the field name/index to a byte offset and type tag at compile time. No field names survive to runtime.

Three new opcodes: `OpStencilALLOC` (arg=slotID, offset=totalSize), `OpFieldSTORE` (arg=slotID, offset=fieldByteOffset, extra=tag), `OpFieldLOAD` (arg=slotID, offset=fieldByteOffset, extra=tag). The `Instruction` struct gained an `Offset` field to carry the field byte offset.

`SlotEntry` gained a `Stencil` boolean flag. Stencil slots have `Mask=0` and `Tag=0` since type checking happens per-field via the opcodes.

See: [Memory Model](02-memory-model.md), [Instruction Set](04-instruction-set.md), [Compiler](05-compiler.md), [Runtime](06-runtime.md).

---

## Allocator Interface — Swappable Allocation Strategies

**TODO** :: The `Allocator` is currently a concrete struct with a hardcoded strategy (first-fit free list with coalescing). Extract it into an interface so the runtime can support multiple allocation strategies. The Vega script author would select a strategy per `alloc` block.

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

### Candidate Strategies

| Strategy | Description | `free()` support | Best for |
|----------|-------------|-----------------|----------|
| **Free list** (current) | First-fit with coalescing | Yes | General use, mixed alloc/free patterns |
| **Bump** | Linear pointer advance, no per-slot free | No (`free()` is a no-op or error) | Short-lived blocks where everything is discarded at block exit |
| **Pool** | Fixed-size slots, bitmap tracking | Yes (O(1)) | Uniform-type workloads (all ints, all longs) |

### Syntax (tentative)

```
alloc 64 { ... }            # default (free list)
alloc 64 :bump { ... }      # bump allocator — fast, no free
alloc 64 :pool(4) { ... }   # pool of fixed 4-byte slots
```

The `:strategy` modifier would be parsed as part of the `alloc` statement. The compiler passes the strategy choice through to the runtime, which instantiates the appropriate `Allocator` implementation.

### Why this matters

A bump allocator is the natural fit for the common case in Vega — short-lived scripts that allocate a few variables, do their work, and exit. No `free()` calls, no fragmentation, just a pointer advancing through a buffer. The free list is only needed when scripts explicitly reclaim and reuse memory mid-block.

---

## Primitive Type Budget: 8 is sufficient

**Decision:** 8 primitive types fit a single bitmask byte and cover all fundamental data widths:

| Tag | Type    | Width | Bit | Purpose              |
|-----|---------|-------|-----|----------------------|
| 1   | short   | 2     | 0   | Small signed integer |
| 2   | int     | 4     | 1   | Standard integer     |
| 3   | long    | 8     | 2   | Large integer        |
| 4   | float   | 4     | 3   | Single-precision     |
| 5   | decimal | 8     | 4   | Double-precision     |
| 6   | bool    | 1     | 5   | Boolean              |
| 7   | byte    | 1     | 6   | Raw bytes, uint8     |
| 8   | char    | 4     | 7   | Unicode codepoint    |

Unsigned variants, exotic widths (float16, int128), and composite types are either unnecessary for Vega's scope or handled by stencils. The bitmask approach for union types is validated by this limit.
