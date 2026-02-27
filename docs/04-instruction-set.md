# Instruction Set

## Instruction Format

```go
type Instruction struct {
    Operation  OperationCode  // 1-byte opcode
    Argument   int            // numeric argument (index, slot ID, count)
    Offset     int            // field byte offset (FIELD_LOAD/FIELD_STORE); name table index (CALL_NAT/CALL_FN); total size (STENCIL_ALLOC, VAR_LOAD_RAW, LOAD_ARG_STENCIL); capacity (SLICE_ALLOC)
    Extra      byte           // auxiliary byte: type bitmask (VAR_ALLOC), type tag (VAR_PTR, FIELD_LOAD/STORE, PTR_LOAD), slot ID (LOAD_ARG_STENCIL), return flag (RETURN)
    Size       int            // field byte size for FIELD_STORE/FIELD_LOAD on slice fields (0 for scalar fields where SizeForTag suffices)
    SourceLine int            // source line for error reporting
}
```

Function call target names are stored in `ByteCode.Names[]` (a string side table) and referenced by index via `Instruction.Offset`. This avoids a string field per instruction and keeps the instruction struct uniform.

The `Extra` field carries a **type bitmask** in `VAR_ALLOC` — each bit corresponds to a type tag (tags 1–8 map to bits 0–7). A single-type variable has one bit set; a union type has multiple bits set. For field opcodes, `Extra` carries the field's type tag.

The `Size` field carries the **declared capacity** for `FIELD_STORE`/`FIELD_LOAD` instructions that target slice fields (`TagSlice`). For scalar fields `SizeForTag(tag)` already gives the correct size, so `Size` is 0 and unused.

---

## Opcode Table

| Opcode | Code | Args | Description |
|--------|------|------|-------------|
| `STACK_POP` | `0` | — | Pop and discard the top value from the expression stack |
| `LOAD_CONST` | `1` | `Argument`: constant pool index | Push a constant onto the expression stack |
| `VAR_ALLOC` | `2` | `Argument`: slot ID, `Extra`: type bitmask | Reserve bytes in the allocator, create slot table entry |
| `VAR_STORE` | `3` | `Argument`: slot ID | Pop expression stack, encode, write into allocator |
| `VAR_LOAD` | `4` | `Argument`: slot ID | Read from allocator, decode, push onto expression stack |
| `VAR_FREE` | `5` | `Argument`: slot ID | Return slot's bytes to the free list, mark slot dead |
| `VAR_PTR` | `6` | `Argument`: slot ID, `Extra`: type tag | Create alias slot at explicit buffer offset (offset popped from stack) |
| `STENCIL_ALLOC` | `7` | `Argument`: slot ID, `Offset`: total size | Allocate stencil-sized slot for struct/tuple |
| `SLICE_ALLOC` | `8` | `Argument`: slot ID, `Offset`: capacity in bytes | Allocate a bounded slice slot (`string<N>`, `byte<N>`) |
| `FIELD_STORE` | `9` | `Argument`: slot ID, `Offset`: field byte offset, `Extra`: type tag, `Size`: field size (slices) | Pop expr stack, copy into struct field |
| `FIELD_LOAD` | `10` | `Argument`: slot ID, `Offset`: field byte offset, `Extra`: type tag, `Size`: field size (slices) | Load struct field, push onto expr stack |
| `CALL_NAT` | `11` | `Argument`: argc, `Offset`: Names[] index | Call a registered native (Go) function |
| `CALL_FN` | `12` | `Argument`: argc, `Offset`: Names[] index | Call a user-defined (Vega) function |
| `RETURN` | `13` | `Extra`: 0=void, 1=has return value | Return from user function |
| `LOAD_ARG` | `14` | `Argument`: argument index | Push a pending primitive argument onto the expr stack |
| `VAR_LOAD_RAW` | `15` | `Argument`: slot ID, `Offset`: total size | Push stencil slot's bytes as a `RawValue` (for struct argument passing) |
| `LOAD_ARG_STENCIL` | `16` | `Argument`: pending index, `Extra`: slot ID, `Offset`: total size | Restore a struct argument from pending args into a new stencil slot |
| `PTR_LOAD` | `17` | `Extra`: type tag | Pop offset, read tag-typed value from allocator at that offset, push result |

---

## Detailed Opcode Semantics

### LOAD_CONST (opcode 1)

**Emitted by:** All literal expression compilations.

**Runtime effect:** Reads `Constants[Argument]` from the bytecode's constant pool and pushes it onto the expression stack.

### VAR_ALLOC (opcode 2)

**Emitted by:** First assignment to a new (non-slice, non-pointer, non-stencil) variable.

**Arguments:**
- `Argument` — the slot ID (0, 1, 2, ... assigned sequentially by the compiler)
- `Extra` — the type bitmask (e.g., `00000010` for int only, `00100010` for `int|bool`)

**Runtime effect:**
1. Computes the byte size from the mask: `MaxSizeForMask(Extra)` — the maximum across all allowed types.
2. Calls `allocator.Alloc(size)` to get an offset.
3. Records `{offset, size, tag=0, mask, alive=true}` in the slot table at position `Argument`. Tag starts at 0 (uninitialized) until the first `VAR_STORE`.

**Error:** "out of memory" if the allocator cannot satisfy the request.

### VAR_STORE (opcode 3)

**Emitted by:** Every assignment to an existing variable (both first and subsequent, for all types including slices).

**Runtime effect:**
1. Pops the top value from the expression stack.
2. Verifies the value is `Allocable`.
3. Verifies `TagInMask(TagFor(value), slot.Mask)` — the value's type must be in the allowed set.
4. Updates `slot.Tag` to the value's actual tag.
5. Copies the value's backing bytes into the allocator at the slot's offset.

### VAR_LOAD (opcode 4)

**Emitted by:** Identifier expressions that reference a non-stencil variable.

**Runtime effect:**
1. Checks that `slot.Tag != 0` (the slot has been written to at least once).
2. For slices: reads `slot.Capacity` bytes; for other types: reads `SizeForTag(slot.Tag)` bytes.
3. Wraps the view as a value via `value.Wrap(slot.Tag, view)` — reads directly from the alloc buffer (no copy).
4. Pushes the resulting value onto the expression stack.

### VAR_FREE (opcode 5)

**Emitted by:** `free(x)` statements.

**Runtime effect:**
1. Checks the slot is not an alias (`Alias == false`).
2. Calls `allocator.Free(slot.Offset, slot.Capacity)` — zeros memory and returns to free list.
3. Marks `slot.Alive = false`.

**Errors:**
- "double free on slot N" if the slot is already dead.
- "cannot free pointer alias on slot N" if the slot is an alias.

### VAR_PTR (opcode 6)

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

**Key difference from VAR_ALLOC:** `VAR_PTR` skips the free list entirely — it just records an offset. Alias slots can overlap with allocated regions or with each other.

### STENCIL_ALLOC (opcode 7)

**Emitted by:** Struct literal and tuple assignments (first assignment only).

**Arguments:**
- `Argument` — the slot ID
- `Offset` — the total stencil size in bytes (sum of all field sizes)

**Runtime effect:**
1. Calls `allocator.Alloc(totalSize)` to get a contiguous region.
2. Records `{offset, totalSize, tag=0, mask=0, alive=true, stencil=true}` in the slot table.

**Key difference from VAR_ALLOC:** Allocates a multi-field region. The slot has `Mask=0` and `Tag=0` because type checking is per-field (via `FIELD_STORE`/`FIELD_LOAD`), not per-slot.

### SLICE_ALLOC (opcode 8)

**Emitted by:** First assignment to a `string<N>` or `byte<N>` variable or struct field.

**Arguments:**
- `Argument` — the slot ID
- `Offset` — the declared capacity in bytes (the `N` from `string<N>`)

**Runtime effect:**
1. Calls `allocator.Alloc(capacity)` to get a contiguous region of exactly N bytes.
2. Records `{offset, capacity, tag=TagSlice, mask=0, alive=true}` in the slot table (`Capacity = N`).

**Key difference from VAR_ALLOC:** The capacity is fixed at declaration time; `SizeForTag(TagSlice)` returns 0, so `VAR_ALLOC` cannot be used. `SLICE_ALLOC` encodes the capacity in `Offset` and stores it in `SlotEntry.Capacity`. `VAR_STORE` on a slice slot validates that the content fits within the capacity and zero-pads the rest.

### FIELD_STORE (opcode 9)

**Emitted by:** Struct literal and tuple initialization (one per field/element).

**Arguments:**
- `Argument` — the slot ID (identifies the stencil slot)
- `Offset` — the field's byte offset within the stencil
- `Extra` — the field's type tag
- `Size` — the field's byte size for slice fields (`TagSlice`); 0 for scalar fields

**Runtime effect:**
1. Pops the top value from the expression stack.
2. Asserts the value is `Allocable`.
3. Asserts `TagFor(value) == tag` — the value's type must match the field's declared type.
4. Computes field size via `fieldSizeFor(tag, instr.Size)` — uses `Size` for slices, `SizeForTag(tag)` for scalars.
5. Computes the absolute buffer position: `slot.Offset + fieldOffset`.
6. For slice fields: validates content length ≤ field capacity (runtime error on overflow), copies content bytes, zero-pads remainder.
7. For scalar fields: copies the value's bytes into the allocator at that position.

### FIELD_LOAD (opcode 10)

**Emitted by:** Field access expressions (`obj.field`, `tuple.0`).

**Arguments:**
- `Argument` — the slot ID
- `Offset` — the field's byte offset within the stencil
- `Extra` — the field's type tag
- `Size` — the field's byte size for slice fields; 0 for scalar fields

**Runtime effect:**
1. Computes field size via `fieldSizeFor(tag, instr.Size)`.
2. Computes the absolute buffer position: `slot.Offset + fieldOffset`.
3. Wraps the byte region as a view-based value via `value.Wrap(tag, view)`.
4. Pushes the value onto the expression stack.

### CALL_NAT (opcode 11)

**Emitted by:** Call statements to native (Go-registered) functions.

**Arguments:**
- `Argument` — number of arguments (argc)
- `Offset` — index into `ByteCode.Names[]` for the function name

**Runtime effect:**
1. Looks up the function name via `Names[Offset]`.
2. Pops `argc` values from the expression stack (right-to-left for correct ordering).
3. Calls the registered Go function with the arguments.

### CALL_FN (opcode 12)

**Emitted by:** Call statements to user-defined (Vega) functions.

**Arguments:**
- `Argument` — number of arguments (argc)
- `Offset` — index into `ByteCode.Names[]` for the function name

**Runtime effect:**
1. Looks up the compiled `FunctionDef` in `userFuncs`.
2. Pops `argc` values from the expression stack into `pendingArgs`.
3. Saves the caller's `exprStack` and `slots` inside the current frame.
4. Initializes fresh `exprStack` and `slots` for the callee.
5. Pushes a new `CallFrame` for the callee's bytecode.

**Note:** `CALL_FN` at statement context sets `ExprCall=false` on the new frame; the return value (if any) is discarded. Expression-context calls set `ExprCall=true` so the return value is pushed onto the caller's stack.

### RETURN (opcode 13)

**Emitted by:** `return` statements and the implicit end-of-function return.

**Arguments:**
- `Extra` — `1` if a return value is on the expression stack, `0` for void return

**Runtime effect (`doReturn`):**
1. If `Extra==1`, pops the return value and **materializes** it (copies bytes into a fresh `[]byte`) so it doesn't point into memory that is about to be freed.
2. Frees all alive, non-alias slots in the callee's scope back to the global allocator.
3. Steps back to the caller's frame and restores its saved `exprStack` and `slots`.
4. If `Extra==1` and the call site had `ExprCall=true`, pushes the materialized return value onto the caller's stack.

### LOAD_ARG (opcode 14)

**Emitted by:** Function prologue for primitive parameters.

**Arguments:**
- `Argument` — index into `pendingArgs`

**Runtime effect:** Pushes `pendingArgs[index]` onto the expression stack, where it is then consumed by `VAR_ALLOC` + `VAR_STORE` to create the parameter's slot.

### VAR_LOAD_RAW (opcode 15)

**Emitted by:** Identifier expressions referencing a stencil variable used as a function argument.

**Arguments:**
- `Argument` — slot ID
- `Offset` — total stencil size in bytes

**Runtime effect:** Reads `size` bytes from the stencil slot's region and pushes a `RawValue` containing a copy. Used to serialize struct variables for passing across function boundaries via `pendingArgs`.

### LOAD_ARG_STENCIL (opcode 16)

**Emitted by:** Function prologue for struct parameters.

**Arguments:**
- `Argument` — index into `pendingArgs`
- `Extra` — destination slot ID (within the callee's slot table)
- `Offset` — stencil total size in bytes

**Runtime effect:**
1. Reads the `RawValue` from `pendingArgs[index]`.
2. Allocates `size` bytes from the global allocator.
3. Creates a stencil slot at that offset and bulk-copies the raw bytes in.

### PTR_LOAD (opcode 17)

**Emitted by:** Pointer dereference expressions used as values (`*int(offset)` in expression position, without creating an alias slot).

**Arguments:**
- `Extra` — type tag to read as

**Runtime effect:**
1. Pops the offset from the expression stack.
2. Bounds-checks: `offset + SizeForTag(tag) <= allocator.Capacity()`.
3. Reads the bytes at that offset and pushes a view-based value.

**Key difference from VAR_PTR:** `PTR_LOAD` reads a value transiently without creating a named slot. `VAR_PTR` creates a persistent alias slot in the slot table.

---

## Bytecode Emission Helpers

| Method | Signature | Used by |
|--------|-----------|---------|
| `Emit` | `(op, line) int` | `STACK_POP`, `RETURN` (void) |
| `EmitArg` | `(op, arg, line) int` | `LOAD_CONST`, `VAR_STORE`, `VAR_LOAD`, `VAR_FREE`, `LOAD_ARG` |
| `EmitArgExtra` | `(op, arg, extra, line) int` | `VAR_ALLOC` (bitmask), `VAR_PTR` (type tag), `RETURN` (value flag), `PTR_LOAD` |
| `EmitField` | `(op, arg, offset, extra, line) int` | `STENCIL_ALLOC`, `SLICE_ALLOC`, `VAR_LOAD_RAW`, `LOAD_ARG_STENCIL` |
| `EmitFieldSize` | `(op, arg, offset, extra, size, line) int` | `FIELD_STORE`, `FIELD_LOAD` — also carries `Size` for slice field capacity |
| `EmitNameArg` | `(op, name, arg, line) int` | `CALL_NAT`, `CALL_FN` — interns name in `Names[]`, stores index in `Offset` |

---

## Disassembly Format

Each instruction has a `String()` method for debugging. The disassembler in `ByteCode.Disassemble()` prints the full bytecode including constants and any embedded function bodies.

```
=== Constants ===
   0: 2a000000 (int)
   1: 00000000 (int)

=== Instructions ===
   0: LOAD_CONST index=0
   1: VAR_ALLOC slot=0 mask=00000010
   2: VAR_STORE slot=0
   3: LOAD_CONST index=1
   4: VAR_PTR slot=1 tag=2
   5: STENCIL_ALLOC slot=2 size=8
   6: FIELD_STORE slot=2 offset=0 tag=2
   7: FIELD_LOAD slot=2 offset=0 tag=2
   8: RETURN void

=== Function: add ===
=== Constants ===
   0: 00000000 (int)
=== Instructions ===
   0: LOAD_ARG index=0
   1: VAR_ALLOC slot=0 mask=00000010
   2: VAR_STORE slot=0
   3: LOAD_ARG index=1
   4: VAR_ALLOC slot=1 mask=00000010
   5: VAR_STORE slot=1
   6: RETURN void
```

For `CALL_NAT` and `CALL_FN`, the disassembler looks up the name from `Names[]`:
```
   5: CALL_NAT print argc=1
   6: CALL_FN  add argc=2
```
