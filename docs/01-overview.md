# Vega Architecture Overview

**Status:** Active development
**Package root:** `pkg/`

---

## Design

Named variables are stored at runtime as raw bytes inside a fixed-size `[]byte` buffer managed by a free list allocator. Variable names exist only at compile time; at runtime, everything is slot IDs mapping to `{offset, size, typeTag}` entries.

The expression evaluation stack (push/pop temporaries during expression evaluation) remains a `[]value.Value` slice. Only named variable storage moves to the byte array.

---

## Pipeline

```
Source → Lexer → Tokens → Parser → AST → Compiler → Bytecode → VM → Result
```

Each stage is a separate package under `pkg/`.

---

## Package Map

| Package | Purpose |
|---------|---------|
| `pkg/lexer` | Tokenizer with position tracking |
| `pkg/parser` | Recursive descent parser, AST nodes |
| `pkg/compiler` | Bytecode generation with symbol table |
| `pkg/value` | Runtime value types, type tags, binary encoding |
| `pkg/alloc` | Free list allocator over `[]byte` |
| `pkg/vm` | VM, runtime, call frames |

---

## Documentation Index

| Document | Covers |
|----------|--------|
| [Memory Model](02-memory-model.md) | Byte-array backing store, free list allocator, slot table |
| [Type System](03-type-system.md) | Type tags, binary encoding/decoding, allocable values |
| [Instruction Set](04-instruction-set.md) | All opcodes, the new `VAR_*` family, instruction format |
| [Compiler](05-compiler.md) | Symbol table, type inference, scope management |
| [Runtime](06-runtime.md) | Stack, allocator integration, instruction handlers |
| [Language Comparison](07-language-comparison.md) | How Vega's memory model differs from other languages |

---

## Quick Examples

### Untyped (inferred types)

```
alloc 64 {
    x = 42
    y = 3.14
    free(x)
    z = 100l
    z
}
```

What happens:

1. `alloc 64` creates a 64-byte `[]byte` buffer and an empty slot table.
2. `x = 42` — compiler infers `TagInteger` (4 bytes), emits `VAR_ALLOC slot=0 mask=00000010`, then `VAR_STORE slot=0`. The allocator reserves bytes `[0..4)`. The integer `42` is encoded as 4 little-endian bytes and written into the buffer.
3. `y = 3.14` — compiler infers `TagDecimal` (8 bytes), allocator reserves `[4..12)`.
4. `free(x)` — emits `VAR_FREE slot=0`. The allocator zeros bytes `[0..4)` and returns them to the free list. The compiler removes `x` from its symbol table — any subsequent reference to `x` is a compile error.
5. `z = 100l` — compiler infers `TagLong` (8 bytes). The allocator finds the free block at `[0..4)` (only 4 bytes, too small), then the free block at `[12..64)` (52 bytes, fits), reserves `[12..20)`.
6. `z` — emits `VAR_LOAD slot=2`, reads 8 bytes from `[12..20)`, decodes as `int64`, pushes onto the expression stack.
7. `STACK_FREE` destroys the allocator and slot table.

### Pointer aliases

```
alloc 8 {
    x = 42
    y = *int(0)
    z = *short(0)
}
```

What happens:

1. `x = 42` — compiler infers `TagInteger`, allocator reserves 4 bytes at offset 0.
2. `y = *int(0)` — pointer alias: creates a slot viewing bytes `[0..4)` as int. No allocator call. Reading `y` returns the same value as `x` (42).
3. `z = *short(0)` — pointer alias: creates a slot viewing bytes `[0..2)` as short. Overlaps with `x` and `y`. Reading `z` returns the first 2 bytes of `x` reinterpreted as a `short`.

Aliases don't allocate memory — they just view existing bytes at explicit offsets. `free()` on an alias is a runtime error.

### Typed (union types)

```
alloc 64 {
    y: int|bool = 15
    y = true
    y
}
```

What happens:

1. `y: int|bool = 15` — compiler resolves `int|bool` to bitmask `00100010`, emits `VAR_ALLOC slot=0 mask=00100010`. The allocator reserves `max(4, 1) = 4` bytes. On `VAR_STORE`, the runtime checks that `TagInteger` is in the mask (it is), sets `slot.Tag = TagInteger`, and writes the encoded integer.
2. `y = true` — the variable already exists (no new `VAR_ALLOC`). `VAR_STORE` checks that `TagBoolean` is in the mask (it is), updates `slot.Tag = TagBoolean`, and writes the encoded boolean.
3. `y` — `VAR_LOAD` reads the bytes and decodes using the current `slot.Tag` (`TagBoolean`), producing `true`.
