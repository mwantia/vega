# Instruction Set

## Instruction Format

```go
type Instruction struct {
    Operation  OperationCode  // 1-byte opcode
    Argument   int            // numeric argument (index, slot ID, count)
    Offset     int            // field byte offset (for FIELD_LOAD/FIELD_STORE)
    Name       string         // string argument (variable name — legacy, unused by new opcodes)
    Extra      byte           // auxiliary byte (type bitmask for VAR_ALLOC, type tag for field ops)
    SourceLine int            // source line for error reporting
}
```

The `Extra` field was added for the variable allocation opcodes. It carries a **type bitmask** in `VAR_ALLOC` — each bit corresponds to a type tag (tags 1–8 map to bits 0–7). A single-type variable has one bit set; a union type has multiple bits set. For field opcodes (`FIELD_LOAD`, `FIELD_STORE`), `Extra` carries the field's type tag and `Offset` carries the field's byte offset within the stencil.

---

## Opcode Table

| Opcode | Name | Args | Description |
|--------|------|------|-------------|
| `0` | `STACK_POP` | — | Pop and discard the top value from the expression stack |
| `1` | `STACK_ALLOC` | `Argument`: capacity in bytes | Create expression stack, allocator, and slot table |
| `2` | `STACK_FREE` | — | Destroy expression stack, allocator, and slot table |
| `3` | `LOAD_CONST` | `Argument`: constant pool index | Push a constant onto the expression stack |
| `4` | `VAR_ALLOC` | `Argument`: slot ID, `Extra`: type bitmask | Reserve bytes in the allocator, create slot table entry |
| `5` | `VAR_STORE` | `Argument`: slot ID | Pop expression stack, encode, write into allocator |
| `6` | `VAR_LOAD` | `Argument`: slot ID | Read from allocator, decode, push onto expression stack |
| `7` | `VAR_FREE` | `Argument`: slot ID | Return slot's bytes to the free list, mark slot dead |
| `8` | `VAR_PTR` | `Argument`: slot ID, `Extra`: type tag | Create alias slot at explicit buffer offset |
| `9` | `STENCIL_ALLOC` | `Argument`: slot ID, `Offset`: total size | Allocate stencil-sized slot for struct/tuple |
| `10` | `FIELD_STORE` | `Argument`: slot ID, `Offset`: field byte offset, `Extra`: type tag | Pop expr stack, copy into struct field |
| `11` | `FIELD_LOAD` | `Argument`: slot ID, `Offset`: field byte offset, `Extra`: type tag | Load struct field, push onto expr stack |

---

## Detailed Opcode Semantics

### STACK_ALLOC (opcode 1)

**Emitted by:** `AllocStatement` compilation.

**Runtime effect:**
1. Creates an expression stack (`[]value.Value`).
2. Creates a byte allocator with the given capacity.
3. Creates an empty slot table.

**Error:** "stack already allocated" if a stack is already active.

### STACK_FREE (opcode 2)

**Emitted by:** `AllocStatement` compilation (end of block).

**Runtime effect:** Sets the expression stack, allocator, and slot table to nil.

### LOAD_CONST (opcode 3)

**Emitted by:** All literal expression compilations.

**Runtime effect:** Reads `Constants[Argument]` from the bytecode's constant pool and pushes it onto the expression stack.

### VAR_ALLOC (opcode 4)

**Emitted by:** First assignment to a new variable inside an `alloc` block.

**Arguments:**
- `Argument` — the slot ID (0, 1, 2, ... assigned sequentially by the compiler)
- `Extra` — the type bitmask (e.g., `00000010` for int only, `00100010` for `int|bool`)

**Runtime effect:**
1. Computes the byte size from the mask: `MaxSizeForMask(Extra)` — the maximum across all allowed types.
2. Calls `allocator.Alloc(size)` to get an offset.
3. Records `{offset, size, tag=0, mask, alive=true}` in the slot table at position `Argument`. Tag starts at 0 (uninitialized) until the first `VAR_STORE`.

**Error:** "out of memory" if the allocator cannot satisfy the request.

### VAR_STORE (opcode 5)

**Emitted by:** Every assignment to a variable (both first and subsequent).

**Runtime effect:**
1. Pops the top value from the expression stack.
2. Verifies the value is `Allocable`.
3. Verifies `TagInMask(TagFor(value), slot.Mask)` — the value's type must be in the allowed set.
4. Updates `slot.Tag` to the value's actual tag (tracks the current variant for decoding).
5. Encodes the value into a temporary byte buffer.
6. Writes the bytes into the allocator at the slot's offset.

**Errors:**
- "slot N is not alive" if the slot was freed.
- "value type X is not allocable" for non-allocable values.
- "type mismatch" if the value's tag is not in the slot's mask.

### VAR_LOAD (opcode 6)

**Emitted by:** Identifier expressions that reference a variable in scope.

**Runtime effect:**
1. Checks that `slot.Tag != 0` (the slot has been written to at least once).
2. Reads bytes from the allocator at the slot's offset.
3. Decodes them using the slot's current type tag.
4. Pushes the resulting value onto the expression stack.

**Errors:**
- "use after free on slot N" if the slot is dead.
- "slot N is uninitialized" if the slot has never been stored to.

### VAR_FREE (opcode 7)

**Emitted by:** `free(x)` statements.

**Runtime effect:**
1. Checks the slot is not an alias (`Alias == false`).
2. Calls `allocator.Free(slot.Offset, slot.Size)` — zeros memory and returns to free list.
3. Marks `slot.Alive = false`.

**Errors:**
- "double free on slot N" if the slot is already dead.
- "cannot free pointer alias on slot N" if the slot is an alias.

### VAR_PTR (opcode 8)

**Emitted by:** Pointer alias assignments (`y = *int(0)`).

**Arguments:**
- `Argument` — the slot ID
- `Extra` — the type tag (e.g., `TagInteger = 2`)

**Runtime effect:**
1. Pops the offset value from the expression stack.
2. Extracts an integer offset via `value.ToInt()` (supports byte, short, int, long).
3. Computes `size = SizeForTag(tag)`.
4. Bounds-checks: `offset + size <= allocator.Capacity()`.
5. Creates a `SlotEntry` with `Alias=true`, `Tag=tag`, offset and size set directly — **no allocator call**.

**Errors:**
- "offset value is not allocable" if the popped value isn't an integer type.
- "pointer out of bounds" if the alias would exceed the buffer.
- "no allocator active" if outside an alloc block.

**Key difference from VAR_ALLOC:** `VAR_ALLOC` asks the free list for a region. `VAR_PTR` skips the free list entirely — it just records an offset. This means alias slots can overlap with allocated regions or with each other.

### STENCIL_ALLOC (opcode 9)

**Emitted by:** Struct literal and tuple assignments (first assignment only).

**Arguments:**
- `Argument` — the slot ID
- `Offset` — the total stencil size in bytes (sum of all field sizes)

**Runtime effect:**
1. Calls `allocator.Alloc(totalSize)` to get a contiguous region.
2. Records `{offset, totalSize, tag=0, mask=0, alive=true, stencil=true}` in the slot table.

**Error:** "out of memory" if the allocator cannot satisfy the request.

**Key difference from VAR_ALLOC:** `STENCIL_ALLOC` allocates a multi-field region. The slot has `Mask=0` and `Tag=0` because type checking is per-field (via `FIELD_STORE`/`FIELD_LOAD`), not per-slot.

### FIELD_STORE (opcode 10)

**Emitted by:** Struct literal and tuple initialization (one per field/element).

**Arguments:**
- `Argument` — the slot ID (identifies the stencil slot)
- `Offset` — the field's byte offset within the stencil
- `Extra` — the field's type tag

**Runtime effect:**
1. Pops the top value from the expression stack.
2. Asserts the value is `Allocable`.
3. Asserts `TagFor(value) == tag` — the value's type must match the field's declared type.
4. Computes the absolute buffer position: `slot.Offset + fieldOffset`.
5. Copies the value's bytes into the allocator at that position.

**Errors:**
- "slot N is not alive" if the stencil slot was freed.
- "value is not allocable" for non-allocable values.
- "type mismatch" if the value's tag doesn't match the field's tag.

### FIELD_LOAD (opcode 11)

**Emitted by:** Field access expressions (`obj.field`, `tuple.0`).

**Arguments:**
- `Argument` — the slot ID (identifies the stencil slot)
- `Offset` — the field's byte offset within the stencil
- `Extra` — the field's type tag

**Runtime effect:**
1. Computes the absolute buffer position: `slot.Offset + fieldOffset`.
2. Computes the field size: `SizeForTag(tag)`.
3. Wraps the byte region as a view-based value via `value.Wrap(tag, view)`.
4. Pushes the value onto the expression stack.

**Errors:**
- "slot N is not alive" if the stencil slot was freed.

---

## Bytecode Emission Helpers

| Method | Signature | Used by |
|--------|-----------|---------|
| `Emit` | `(op, line) int` | `STACK_POP`, `STACK_DUP`, `STACK_FREE` |
| `EmitArg` | `(op, arg, line) int` | `STACK_ALLOC`, `LOAD_CONST`, `VAR_STORE`, `VAR_LOAD`, `VAR_FREE` |
| `EmitArgExtra` | `(op, arg, extra, line) int` | `VAR_ALLOC` (bitmask in `extra`), `VAR_PTR` (type tag in `extra`) |
| `EmitField` | `(op, arg, offset, extra, line) int` | `STENCIL_ALLOC`, `FIELD_STORE`, `FIELD_LOAD` (offset + type tag) |
| `EmitName` | `(op, name, line) int` | Legacy — not used by new opcodes |
| `EmitNameArg` | `(op, name, arg, line) int` | Legacy — not used by new opcodes |

---

## Disassembly Format

Each instruction has a `String()` method for debugging:

```
STACK_ALLOC 64
LOAD_CONST 0
VAR_ALLOC slot=0 mask=00000010
VAR_STORE slot=0
VAR_LOAD slot=0
STACK_POP
VAR_FREE slot=0
STACK_FREE
```

The `VAR_ALLOC` format shows both the slot ID and the type bitmask in binary. For example, `mask=00000010` means only `TagInteger` (tag 2, bit 1) is allowed. A union like `int|bool` would show `mask=00100010`. The `VAR_PTR` format shows the slot ID and type tag number. The `VAR_STORE`, `VAR_LOAD`, and `VAR_FREE` formats show the slot ID only — the mask is recorded in the slot table. The stencil and field opcodes show the slot ID, byte offset, and tag:

```
VAR_PTR slot=1 tag=2
STENCIL_ALLOC slot=0 size=8
FIELD_STORE slot=0 offset=0 tag=2
FIELD_STORE slot=0 offset=4 tag=2
FIELD_LOAD slot=0 offset=0 tag=2
```
