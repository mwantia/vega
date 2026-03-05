# Vega

**Vega** (Virtual Execution & Graph Abstraction) is a lightweight scripting language with a stack-based bytecode VM, written in pure Go. It features a statically-typed value model, user-defined structs and functions, string interpolation, and first-class integration with the VFS library for multi-backend filesystem operations.

**Version**: 0.0.1-dev
**Module**: `github.com/mwantia/vega`

## Build

```bash
task build        # static Linux binary → ./build/vega
task test         # run tests with coverage
task run          # start interactive TUI REPL
task run -- -s test/comprehensive.vega  # run script
```

## CLI

```
vega [uri]                       Start interactive TUI REPL (default mount: ephemeral://)
vega compile <input> <output>    Compile .vega source to .vgc binary
vega run <script>                Execute a .vega or .vgc file
vega version                     Show version information
```

### Flags

**`vega`**
```
-d, --disasm    Show disassembled bytecode in the REPL
```

**`vega compile`**
```
-d, --debug     Include source-line debug info in .vgc
-c, --compress  Compress the .vgc payload with zlib
```

### VFS Mount URIs

The REPL mounts a VFS backend at `/` on startup:

```bash
vega                        # ephemeral (in-memory, default)
vega file:///path/to/dir    # local filesystem
vega sqlite:///path/to.db   # SQLite
vega s3://bucket/prefix     # S3
vega postgres://...         # PostgreSQL
vega consul://host/prefix   # Consul
```

## Language

### Types

| Type | Suffix | Size | Example |
|------|--------|------|---------|
| `byte` | `b` | 1 byte | `200b` |
| `short` | `s` | 2 bytes | `32767s` |
| `int` | _(default)_ | 4 bytes | `42` |
| `long` | `l` | 8 bytes | `9000l` |
| `float` | `f` | 4 bytes | `3.14f` |
| `decimal` | _(default)_ | 8 bytes | `2.718` |
| `bool` | | 1 byte | `true`, `false` |
| `string` | | bounded slice | `"hello"` |

`'A'` is a character literal — it resolves to its ASCII integer value.

### Variables

```vega
# Untyped — type inferred from literal
x = 42
name = "Vega"

# Type-annotated
count: int = 0
flag: bool = true

# Union type — accepts either int or bool at runtime
u: int|bool = 42
u = true

# Explicit string capacity (defaults to len(literal) for string literals)
buf: string<128> = ""
```

### String Interpolation

```vega
name = "World"
print($"Hello, {name}!")       # → Hello, World!
print($"x = {x}, y = {y}")

# Literal braces use {{ and }}
print($"JSON: {{key: {x}}}")   # → JSON: {key: 42}
```

### Operators

```vega
# Arithmetic
x + y    x - y    x * y    x / y    x % y    -x

# Comparison
x == y   x != y   x < y   x <= y   x > y   x >= y

# Logical
a && b   a || b   !a
```

### Control Flow

```vega
if x > 10 {
    print("big")
} else {
    print("small")
}

i = 0
while i < 5 {
    print($"i = {i}")
    i = i + 1
}
```

### Functions

Parameters are always type-annotated. Both primitive and struct types are supported.

```vega
fn add(a: int, b: int) {
    return a + b
}

result = add(3, 4)

fn greet(name: string) {
    print($"Hello, {name}!")
}

greet("Vega")

# Void return (no value)
fn reset(n: int) {
    print($"resetting {n}")
    return
}
```

### Structs

```vega
struct point {
    x: int
    y: int
}

p = point { x = 10, y = 20 }
print($"({p.x}, {p.y})")

# Structs as function parameters
fn distance(a: point, b: point) {
    dx = a.x - b.x
    dy = a.y - b.y
    return dx * dx + dy * dy
}

# String fields with explicit capacity
struct person {
    id:   int
    name: string<64>
}
```

### Tuples

Tuples pack mixed types into a single value. Fields are accessed by index.

```vega
pair = (10, true)
print($"first={pair.0} second={pair.1}")

mixed = (1s, 2, 3l, 3.14f, true)
print($"{mixed.0} {mixed.1} {mixed.2}")
```

### Pointer Aliases

`*type(offset)` creates a typed view into the allocator buffer at a raw byte offset.

```vega
x = 555
ptr = *int(0)   # alias pointing to allocator offset 0
print($"alias = {ptr}")
```

### Memory

```vega
temp = 12345
free(temp)   # return temp's slot to the allocator
```

## Built-in Functions

### Core

| Function | Description |
|----------|-------------|
| `print(args...)` | Print all args (space-separated) with a newline |
| `string(value)` | Convert any value to its string representation |
| `type(value)` | Return the type name of a value as a string |

### VFS Operations

Require a mounted VFS (default: `ephemeral://`).

| Function | Description |
|----------|-------------|
| `read(path, offset, size)` | Read `size` bytes from `path` starting at `offset` |
| `exists(path)` | Return `true` if `path` exists in the mounted VFS |

## String Methods

Methods are called with dot notation: `s.upper()`. Members (no parentheses) are accessed as `s.length`.

| Name | Kind | Description |
|------|------|-------------|
| `.length` | member | Byte length of the string content (up to first `\0`) |
| `.capacity` | member | Total allocated capacity of the string buffer |
| `.upper()` | method | Return uppercase copy |
| `.lower()` | method | Return lowercase copy |
| `.trim()` | method | Strip leading/trailing whitespace |
| `.contains(sub)` | method | `true` if string contains `sub` |
| `.startswith(prefix)` | method | `true` if string starts with `prefix` |
| `.endswith(suffix)` | method | `true` if string ends with `suffix` |
| `.index(sub)` | method | Byte index of `sub`, or `-1` if not found |
| `.trimprefix(prefix)` | method | Remove leading `prefix` if present |
| `.trimsuffix(suffix)` | method | Remove trailing `suffix` if present |
| `.replace(old, new)` | method | Replace all occurrences of `old` with `new` |
| `.slice(from[, length])` | method | Return substring starting at `from` |

## Compiled Objects (.vgc)

`vega compile` produces a `.vgc` binary file:

```
HEADER (16 bytes): magic "VEGA", format version, flags, source CRC, total size
SECTION TABLE:     per-section ID, offset, and byte length
SECTION 0:         constant pool (type tag + raw bytes per constant)
SECTION 1:         interned name table (UTF-8 strings)
SECTION 2:         instruction stream (fixed 10 bytes per instruction)
SECTION 3:         debug info — source line numbers (optional, -d flag)
SECTION 4:         function definitions with embedded bytecode
SECTION 5:         struct stencil layouts (when struct params are used)
```

When compiled with `-c`, sections 0-5 are compressed as a single zlib stream; the 16-byte header is always uncompressed. `Deserialize` auto-detects compression from the flags field.

## Architecture

```
Source → Lexer → Tokens → Parser → AST → Compiler → Bytecode → VM → Result
```

| Package | Purpose |
|---------|---------|
| `pkg/lexer` | Tokenises source; handles `$"..."` interpolation splitting |
| `pkg/parser` | Recursive-descent parser, precedence climbing |
| `pkg/compiler` | Bytecode emitter; symbol table; stencil layout |
| `pkg/vm` | Stack-based interpreter; call frames; allocator integration |
| `pkg/alloc` | Free-list / bump / pool allocators; snapshot manager |
| `pkg/slot` | `StackSlot` type (tag + inline data + optional heap backing) |
| `pkg/descriptor` | Global method/member/native registry |
| `pkg/extension` | Built-in and VFS function registrations |
| `pkg/repl` | TUI REPL (Bubble Tea) |

## License

MIT License — see LICENSE for details.
