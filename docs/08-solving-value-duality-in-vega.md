# Solving Value Duality in Vega
## Architectural Options Given Raw-Bytes Priority

---

## The Problem, Precisely

Every `VAR_LOAD` currently does three things:

```
allocator.Slice(offset, size)  → []byte view     (zero-copy, good)
value.Wrap(tag, view)          → *Integer{...}   (Go heap alloc, GC managed)
exprStack.Push(val)            → []value.Value   (interface box, GC managed)
```

And every arithmetic operation dispatches through a Go interface. The allocator is the source of truth, but the expression stack lives in a completely separate world managed by Go's GC. That's the duality.

Critically: `value.Wrap` wraps a zero-copy view, but the *wrapper struct itself* (`*Integer`, `*Long`, etc.) is a heap allocation. You don't own the wrapper's lifecycle — Go's GC does.

---

## Option A: Tagged Stack Slot (Recommended)

**The idea:** Replace `[]value.Value` on the expression stack with a flat struct that stores the tag and raw bytes inline — no interfaces, no heap, no GC.

```go
// pkg/vm/slot.go (new)
type StackSlot struct {
    Tag  value.TypeTag
    Data [8]byte  // inline: covers all fixed-width primitives (max 8 bytes)
}

// For slices: Data encodes (offset uint32, length uint32) into the allocator
// — a fat pointer reference, not a copy.
type SliceRef struct {
    Offset uint32
    Length uint32
}
```

```go
// ExprStack becomes:
type ExprStack struct {
    data []StackSlot  // was []value.Value
    sp   int
}
```

### How operations change

**VAR_LOAD** — read from allocator into a stack slot:
```go
// Before
view := r.allocator.Slice(slot.Offset, size)
val, _ := value.Wrap(slot.Tag, view)      // heap alloc
r.exprStack.Push(val)

// After
s := StackSlot{Tag: slot.Tag}
copy(s.Data[:size], r.allocator.Slice(slot.Offset, size))  // 8-byte memcpy max
r.exprStack.Push(s)
```

**VAR_STORE** — write from stack slot back to allocator:
```go
// Before
alloc, _ := val.(value.Allocatable)
r.allocator.Write(slot.Offset, alloc.View())

// After
r.allocator.Write(slot.Offset, s.Data[:value.SizeForTag(s.Tag)])
```

**Arithmetic (ADD)** — fully inline, no dispatch:
```go
// Before: interface dispatch through Numeric.Add()
// After:
func addSlots(a, b StackSlot) (StackSlot, error) {
    if a.Tag != b.Tag {
        return StackSlot{}, fmt.Errorf("type mismatch")
    }
    out := StackSlot{Tag: a.Tag}
    switch a.Tag {
    case value.TagInteger:
        va := int32(binary.LittleEndian.Uint32(a.Data[:]))
        vb := int32(binary.LittleEndian.Uint32(b.Data[:]))
        binary.LittleEndian.PutUint32(out.Data[:], uint32(va+vb))
    case value.TagLong:
        va := int64(binary.LittleEndian.Uint64(a.Data[:]))
        vb := int64(binary.LittleEndian.Uint64(b.Data[:]))
        binary.LittleEndian.PutUint64(out.Data[:], uint64(va+vb))
    // ... etc.
    }
    return out, nil
}
```

### Slices and zero-copy

For `TagSlice`, `StackSlot.Data` stores a `SliceRef{Offset, Length}` — a reference into the allocator buffer, not a copy. The actual bytes stay in the allocator. VFS buffer reads, string operations, and byte slicing all remain zero-copy:

```go
func (s StackSlot) SliceRef() SliceRef {
    return SliceRef{
        Offset: binary.LittleEndian.Uint32(s.Data[0:4]),
        Length: binary.LittleEndian.Uint32(s.Data[4:8]),
    }
}

// To get the actual bytes (still zero-copy into allocator):
ref := s.SliceRef()
view := r.allocator.Slice(int(ref.Offset), int(ref.Length))
```

### Boundary with native functions

Native functions currently receive `[]value.Value`. You have two paths:

1. **Keep the boundary, convert at call time**: `CALL_NAT` converts `[]StackSlot` → `[]value.Value` only when crossing into Go-land. The hot path (pure Vega execution) stays allocation-free.
2. **Update native function signatures** to accept `[]StackSlot` directly. More work upfront, cleaner long-term.

Path 1 is the pragmatic migration path — change the hot path first, worry about the native boundary later.

### Trade-offs

| | Before | After |
|---|---|---|
| Stack element size | pointer (8B) + heap obj | 9 bytes inline |
| GC pressure per VAR_LOAD | 1 heap alloc | 0 |
| Arithmetic dispatch | interface virtual call | switch on TypeTag |
| Slice zero-copy | yes (view into allocator) | yes (SliceRef into allocator) |
| Struct/stencil support | RawValue (heap copy) | needs RawSlot variant (see below) |
| Migration effort | — | medium |

### Stencil (struct) slots

Structs are too large for `[8]byte`. Two approaches:
- **RawSlot** variant: `StackSlot{Tag: TagStencil, Ref: SliceRef{offset, totalSize}}` — the stencil's bytes stay in the allocator, same as slice refs. `FIELD_LOAD`/`FIELD_STORE` operate on the allocator directly via the ref.
- This is actually better than today: currently `VAR_LOAD_RAW` copies stencil bytes to the heap for arg-passing. With SliceRef, you pass the allocator reference and copy only at materialization points.

---

## Option B: Move the ExprStack into the Allocator

**The idea:** Don't have a separate `ExprStack` data structure at all. The expression stack is a contiguous region within the allocator buffer itself — a bump-allocated scratch area per frame.

```
Global allocator buffer:
[ global vars ] [ frame-A region ] [ frame-A expr scratch ] [ frame-B region ] [ free ]
```

Each stack slot is 9 bytes `(tag byte, data [8]byte)` written directly into the allocator. Push = advance a scratch bump pointer. Pop = retreat it.

```go
type CallFrame struct {
    // ... existing fields
    ScratchOffset int  // start of expr scratch region in allocator
    ScratchSize   int  // max scratch size (computed at compile time)
    ScratchPtr    int  // current push pointer within scratch
}
```

### Why this is interesting for Vega specifically

- **The allocator becomes the sole memory system.** No GC involvement whatsoever for any Vega runtime state.
- **SnapshotManager gets expression stack history for free.** Since the scratch region is part of the allocator buffer, incremental deltas capture stack state between instructions — unprecedented debuggability.
- **Pointer aliases can legally point into scratch.** `*int(scratch_offset)` would let a Vega program read its own stack — an intentionally unsafe but interesting capability.
- **Total memory budgeting.** The combined `SessionAllocSize` covers globals + all call regions + all scratch areas, giving you exact control over the VM's memory footprint. Useful for embedding in VFS where resource limits matter.

### The catch

You need the maximum scratch size per function at compile time (like CPython's `co_stacksize`). This is a straightforward watermark pass over the compiled instruction stream: simulate push/pop counts, track high-water mark, store in `FunctionDef.ScratchSize`. It's not hard and you need this pass anyway for stack-depth safety.

### Trade-offs

| | Option A (StackSlot) | Option B (Alloc-backed) |
|---|---|---|
| Implementation effort | Medium | High |
| GC involvement | Zero on hot path | Truly zero (no Go heap at all) |
| Debuggability / snapshotting | Unchanged | Stack history in SnapshotManager |
| Requires compile-time stack depth | No | Yes |
| Total memory budgeting | Partial | Complete |
| Risk | Low | Medium (allocator touched on every push/pop) |

---

## Recommended Path

**Start with Option A.** It's the right architectural direction, the migration is scoped and reversible, and it directly serves the raw-bytes priority.

**Leave Option B open** as a future evolution once Option A is stable. Moving the scratch region into the allocator is a clean evolution of the region/arena model you've already built — it's the logical conclusion of "the allocator owns everything."

The migration order:

```
1. Define StackSlot{Tag, [8]byte} and SliceRef{Offset, Length}
2. Replace ExprStack.data []value.Value → []StackSlot
3. Update Push/Pop signatures
4. Rewrite VAR_LOAD / VAR_STORE to copy bytes directly
5. Rewrite arithmetic opcodes (ADD, SUB, MUL, etc.) to operate on StackSlot inline
6. Keep CALL_NAT converting []StackSlot → []value.Value at the boundary (temporary shim)
7. Verify: run all existing tests — the observable behavior is identical
8. Optionally: migrate native function signatures to []StackSlot and remove the shim
```

The `value.Value` interface doesn't disappear — it becomes a **boundary type** used only at the native function interface, not an execution-time type. That's a fundamentally better role for it.