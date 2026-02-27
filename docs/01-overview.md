# Vega Architecture Overview

**Status:** Active development
**Package root:** `pkg/`

---

## Design

Named variables are stored at runtime as raw bytes inside a fixed-capacity `[]byte` buffer managed by a global free-list allocator (1 MiB by default). Variable names exist only at compile time; at runtime, everything is slot IDs mapping to `{offset, size, tag}` entries.

The expression evaluation stack (push/pop temporaries during expression evaluation) is a `[]value.Value` slice. The expression stack and slot table are per-scope — they are saved and restored on function calls, but the global allocator is shared across all scopes for the lifetime of the session.

In REPL mode, variables persist between `Run()` calls because the `Session` holds the allocator and slot table across commands.

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
| `pkg/alloc` | Allocator interface and three implementations (free-list, bump, pool) |
| `pkg/vm` | VM, runtime, call frames, session management |

---

## Documentation Index

| Document | Covers |
|----------|--------|
| [Memory Model](02-memory-model.md) | Global byte-array allocator, slot table, function scoping, session |
| [Type System](03-type-system.md) | Type tags, binary encoding/decoding, allocable values |
| [Instruction Set](04-instruction-set.md) | All opcodes, instruction format |
| [Compiler](05-compiler.md) | Symbol table, type inference, scope management, function compilation |
| [Runtime](06-runtime.md) | Stack, allocator integration, call frames, session |
| [Language Comparison](07-language-comparison.md) | How Vega's memory model differs from other languages |

---

## Quick Examples

### Untyped (inferred types)

```
x = 42
y = 3.14
free(x)
z = 100l
```

What happens:

1. `x = 42` — compiler infers `TagInteger` (4 bytes), emits `VAR_ALLOC slot=0 mask=00000010`, then `VAR_STORE slot=0`. The global allocator reserves bytes `[0..4)`. The integer `42` is encoded as 4 little-endian bytes and written into the buffer.
2. `y = 3.14` — compiler infers `TagDecimal` (8 bytes), allocator reserves `[4..12)`.
3. `free(x)` — emits `VAR_FREE slot=0`. The allocator zeros bytes `[0..4)` and returns them to the free list. The compiler removes `x` from its symbol table — any subsequent reference to `x` is a compile error.
4. `z = 100l` — compiler infers `TagLong` (8 bytes). The allocator finds the freed block at `[0..4)` (too small for 8 bytes), then later free space; reserves 8 bytes.

### Pointer aliases

```
x = 42
y = *int(0)
z = *short(0)
```

What happens:

1. `x = 42` — compiler infers `TagInteger`, global allocator reserves 4 bytes at offset 0.
2. `y = *int(0)` — pointer alias: creates a slot viewing bytes `[0..4)` as int. No allocator call. Reading `y` returns the same value as `x` (42).
3. `z = *short(0)` — pointer alias: creates a slot viewing bytes `[0..2)` as short. Overlaps with `x` and `y`. Reading `z` returns the first 2 bytes of `x` reinterpreted as a `short`.

### Typed (union types)

```
y: int|bool = 15
y = true
```

What happens:

1. `y: int|bool = 15` — compiler resolves `int|bool` to bitmask `00100000` (int bit 1 + bool bit 5), emits `VAR_ALLOC slot=0 mask=00100010`. The allocator reserves `max(4, 1) = 4` bytes. On `VAR_STORE`, the runtime checks that `TagInteger` is in the mask, sets `slot.Tag = TagInteger`, and writes the encoded integer.
2. `y = true` — the variable already exists (no new `VAR_ALLOC`). `VAR_STORE` checks that `TagBoolean` is in the mask, updates `slot.Tag = TagBoolean`, and writes the encoded boolean.

### User-defined functions

```
fn greet(msg: string<20>) {
    print(msg)
}
greet("hello")
```

User-defined functions compile to separate `ByteCode` objects stored in the `Functions` map. On call, the current expression stack and slot table are saved inside the caller's frame; the callee gets fresh ones. On return, all alive non-alias slots from the callee are freed back to the global allocator, and the caller's state is restored.

### User-defined stencils

```
struct meta {
    id: string<36>
    key: string<20>
    
    mode: int
    size: long

    uid: long
    gid: long

    contentType: string<28>
    etag: string<32>
}
```