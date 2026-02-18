# Type System

## Value Hierarchy

All runtime values implement the `Value` interface:

```go
type Value interface {
    Type() string
    String() string
}
```

Extended interfaces compose additional capabilities:

```
Value
├── Allocable ─── fixed-size, can live in the byte buffer
│   ├── Comparable ─── comparison operations (<, >, <=, >=)
│   └── Numeric ──── arithmetic operations (+, -, *, /, %, negation)
├── Methodable ─── extended method calls
├── Memberable ─── member access (.name, .size)
├── Indexable ──── subscript access ([key])
└── Iterable ──── for-loop iteration
```

---

## Allocable Types

These types have a known, fixed byte size and can be stored in the byte-array variable buffer.

| Type | Go type | Size | Tag constant | Tag value |
|------|---------|------|--------------|-----------|
| `short` | `int16` | 2 bytes | `TagShort` | `1` |
| `int` | `int32` | 4 bytes | `TagInteger` | `2` |
| `long` | `int64` | 8 bytes | `TagLong` | `3` |
| `float` | `float32` | 4 bytes | `TagFloat` | `4` |
| `decimal` | `float64` | 8 bytes | `TagDecimal` | `5` |
| `boolean` | `bool` | 1 byte | `TagBoolean` | `6` |
| `byte` | `uint8` | 1 byte | `TagByte` | `7` |
| `char` | `rune` | 4 bytes | `TagChar` | `8` |

### Literal Syntax

| Suffix | Type | Example |
|--------|------|---------|
| `b` | byte | `42b`, `0xFF` |
| `s` | short | `42s` |
| *(none, integer)* | int | `42` |
| `l` | long | `42l` |
| `f` | float | `3.14f` |
| *(none, decimal)* | decimal | `3.14` |
| `true`/`false` | boolean | `true` |
| `'...'` | char | `'A'` |

---

## Non-Allocable Types

These types do **not** live in the byte buffer. They exist only on the expression stack or in the constants table.

| Type | Storage | Reason |
|------|---------|--------|
| `string` | Constants table (static VM memory) | Variable length, borrowed semantics |
| `nil` | Singleton (`value.Nil`) | No data to encode |

---

## Type Tags

**Package:** `pkg/value` — file `typetag.go`

A `TypeTag` is a single byte that identifies how to interpret a region of the byte buffer.

```go
type TypeTag byte
```

### Functions

| Function | Signature | Purpose |
|----------|-----------|---------|
| `TagFor` | `(Allocable) TypeTag` | Returns the tag for a runtime value |
| `SizeForTag` | `(TypeTag) int` | Returns the byte size for a tag |
| `TagForName` | `(string) (TypeTag, bool)` | Resolves a type name (`"int"`, `"bool"`, ...) to its tag |
| `MaskForTag` | `(TypeTag) byte` | Returns a bitmask with the bit for the tag set (`1 << (tag-1)`) |
| `TagInMask` | `(TypeTag, byte) bool` | Checks whether a tag is present in a bitmask |
| `MaxSizeForMask` | `(byte) int` | Returns the max byte size across all tags in a mask |
| `ToInt` | `(Allocable) (int, error)` | Extracts an integer offset from byte/short/int/long values |

`TagFor` uses a type switch on the concrete value type. `SizeForTag` is a pure lookup — no runtime value needed. The bitmask functions support union types: a mask like `0b00100010` represents the set `{int, bool}`.

---

## Binary Encoding

**Package:** `pkg/value` — file `encoding.go`

Values are serialized to/from `[]byte` using `encoding/binary.LittleEndian`.

### `Encode(dst []byte, a Allocable) error`

Writes the value's raw bytes into `dst`. The caller must ensure `dst` is at least `SizeForTag(TagFor(a))` bytes.

| Type | Encoding |
|------|----------|
| `short` | `LittleEndian.PutUint16(dst, uint16(data))` |
| `int` | `LittleEndian.PutUint32(dst, uint32(data))` |
| `long` | `LittleEndian.PutUint64(dst, uint64(data))` |
| `float` | `LittleEndian.PutUint32(dst, math.Float32bits(data))` |
| `decimal` | `LittleEndian.PutUint64(dst, math.Float64bits(data))` |
| `boolean` | `dst[0] = 1` if true, `dst[0] = 0` if false |
| `byte` | `dst[0] = data` |
| `char` | `LittleEndian.PutUint32(dst, uint32(data))` |

### `Decode(src []byte, tag TypeTag) (Value, error)`

Reconstructs a `Value` from raw bytes and a type tag. The inverse of `Encode`.

| Tag | Decoding |
|-----|----------|
| `TagShort` | `int16(LittleEndian.Uint16(src))` → `NewShort(...)` |
| `TagInteger` | `int32(LittleEndian.Uint32(src))` → `NewInteger(...)` |
| `TagLong` | `int64(LittleEndian.Uint64(src))` → `NewLong(...)` |
| `TagFloat` | `math.Float32frombits(LittleEndian.Uint32(src))` → `NewFloat(...)` |
| `TagDecimal` | `math.Float64frombits(LittleEndian.Uint64(src))` → `NewDecimal(...)` |
| `TagBoolean` | `src[0] != 0` → `NewBoolean(...)` |
| `TagByte` | `src[0]` → `NewByte(...)` |
| `TagChar` | `rune(LittleEndian.Uint32(src))` → `NewChar(...)` |

### Round-Trip Guarantee

For all allocable types: `Decode(Encode(v), TagFor(v)) == v`. This is verified by unit tests for every type including edge cases (negative values, zero, max range).

---

## Type Inference

The compiler infers the type tag from the right-hand side of an assignment:

| Expression type | Inferred tag |
|-----------------|--------------|
| `ByteExpression` | `TagByte` |
| `ShortExpression` | `TagShort` |
| `IntegerExpression` | `TagInteger` |
| `LongExpression` | `TagLong` |
| `FloatExpression` | `TagFloat` |
| `DecimalExpression` | `TagDecimal` |
| `BooleanExpression` | `TagBoolean` |
| `CharExpression` | `TagChar` |
| `PointerExpression` | Tag resolved from the pointer's type name (e.g., `*int(0)` → `TagInteger`) |
| `IdentifierExpression` | Tag of the referenced variable (looked up in symbol table) |

Compound expressions (arithmetic, function calls) are not yet supported for type inference. This is intentional — the initial implementation covers direct literal, variable, and pointer alias assignments only.

---

## Type Safety

Variables are constrained by a **bitmask** established at first assignment. Each slot stores a mask of allowed type tags, and the runtime enforces that every store matches the mask.

### Single-type (inferred)

Untyped assignments infer the type and create a single-type mask:

```
alloc 64 {
    x = 42       # x mask = 00000010 (int only, 4 bytes)
    x = 100      # OK: int is in the mask
    x = true     # Runtime error: type mismatch
}
```

### Union types (explicit)

Typed declarations build a multi-type mask. The slot allocates `max(size)` across all allowed types:

```
alloc 64 {
    y: int|bool = 15    # y mask = 00100010 (int + bool, 8 bytes max)
    y = true            # OK: bool is in the mask
    y = 3.14f           # Runtime error: float not in the mask
}
```

### Supported type names

| Name | Tag |
|------|-----|
| `short` | `TagShort` |
| `int` | `TagInteger` |
| `long` | `TagLong` |
| `float` | `TagFloat` |
| `decimal` | `TagDecimal` |
| `bool` | `TagBoolean` |
| `byte` | `TagByte` |
| `char` | `TagChar` |

### How it works

The compiler resolves type constraint identifiers to tags via `TagForName`, builds the mask via `MaskForTag`, and emits the mask in `OpVarALLOC.Extra`. The runtime's `OpVarSTORE` checks `TagInMask(actualTag, slot.Mask)` and rejects mismatches. `SlotEntry.Tag` is updated on every successful store to track the current variant for `OpVarLOAD` decoding.
