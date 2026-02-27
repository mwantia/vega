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
├── Allocable ─── fixed-size (or fixed-capacity for slices), can live in the byte buffer
│   ├── Comparable ─── comparison operations (<, >, <=, >=)
│   └── Numeric ──── arithmetic operations (+, -, *, /, %, negation)
├── Methodable ─── extended method calls
├── Memberable ─── member access (.name, .size)
├── Indexable ──── subscript access ([key])
└── Iterable ──── for-loop iteration
```

---

## Allocable Types

These types implement the `Allocable` interface and can be stored in the byte-array variable buffer.

| Type | Go type | Size | Tag constant | Tag value |
|------|---------|------|--------------|-----------|
| `short` | `int16` | 2 bytes | `TagShort` | `1` |
| `int` | `int32` | 4 bytes | `TagInteger` | `2` |
| `long` | `int64` | 8 bytes | `TagLong` | `3` |
| `float` | `float32` | 4 bytes | `TagFloat` | `4` |
| `decimal` | `float64` | 8 bytes | `TagDecimal` | `5` |
| `boolean` | `bool` | 1 byte | `TagBoolean` | `6` |
| `byte` | `uint8` | 1 byte | `TagByte` | `7` |
| `slice` | `[]byte` | N bytes (capacity) | `TagSlice` | `8` |

`slice` is fixed-capacity — `SizeForTag(TagSlice)` returns 0 (capacity is always external). The declared capacity N is tracked in `SlotEntry.Capacity` and set by `SLICE_ALLOC`. `string<N>` is the text form; `byte<N>` is the raw-byte form. Both use `TagSlice`. Content ends at the first `\0` byte or at capacity, whichever comes first.

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
| `"..."` | slice (string content) | `"hello"` — must be used with a `string<N>` or `byte<N>` constraint |
| `'A'` | *(lowered to int)* | `65` — char literals are decoded to their Unicode codepoint as `int32` |

**Note on string literals:** A bare `"..."` literal produces a `SliceValue` of type `TagSlice`. It **must** appear on the RHS of a declaration with an explicit `string<N>` (or `byte<N>`) constraint. Using a string literal without a constraint is a compile error.

**Note on char literals:** There is no `char` type or `TagChar`. The lexer recognizes single-quoted character literals (`'A'`), but the parser lowers them to `IntegerExpression` with the Unicode codepoint value. `'A'` compiles identically to `65`.

---

## Non-Allocable Types

These types do **not** live in the byte buffer. They exist only on the expression stack or in the constants table.

| Type | Storage | Reason |
|------|---------|--------|
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
| `SizeForTag` | `(TypeTag) int` | Returns the byte size for a tag (0 for `TagSlice` — capacity is external) |
| `TagForName` | `(string) (TypeTag, bool)` | Resolves a type name to its tag (`"int"`, `"bool"`, `"string"`, `"slice"`, ...) |
| `NameForTag` | `(TypeTag) (string, bool)` | Resolves a tag to its name |
| `MaskForTag` | `(TypeTag) byte` | Returns a bitmask with the bit for the tag set (`1 << (tag-1)`) |
| `TagInMask` | `(TypeTag, byte) bool` | Checks whether a tag is present in a bitmask |
| `MaxSizeForMask` | `(byte) int` | Returns the max byte size across all tags in a mask |
| `ToInt` | `(Allocable) (int, error)` | Extracts an integer offset from byte/short/int/long values |

`TagFor` uses a type switch on the concrete value type. `SizeForTag` is a pure lookup. The bitmask functions support union types: a mask like `0b00100010` represents the set `{int, bool}`.

---

## Binary Encoding

**Package:** `pkg/value` — file `encoding.go`

Values are serialized to/from `[]byte` using `encoding/binary.LittleEndian`.

### `Encode(dst []byte, a Allocable) error`

Writes the value's raw bytes into `dst`. The caller must ensure `dst` is at least `SizeForTag(TagFor(a))` bytes (or `SlotEntry.Capacity` for slices).

| Type | Encoding |
|------|----------|
| `short` | `LittleEndian.PutUint16(dst, uint16(data))` |
| `int` | `LittleEndian.PutUint32(dst, uint32(data))` |
| `long` | `LittleEndian.PutUint64(dst, uint64(data))` |
| `float` | `LittleEndian.PutUint32(dst, math.Float32bits(data))` |
| `decimal` | `LittleEndian.PutUint64(dst, math.Float64bits(data))` |
| `boolean` | `dst[0] = 1` if true, `dst[0] = 0` if false |
| `byte` | `dst[0] = data` |
| `slice` | raw bytes copied directly into the capacity region; remainder zero-padded |

### `Decode(src []byte, tag TypeTag) (Value, error)`

Reconstructs a `Value` from raw bytes and a type tag. The inverse of `Encode`.

### Round-Trip Guarantee

For all fixed-size allocable types: `Decode(Encode(v), TagFor(v)) == v`. This is verified by unit tests for every type including edge cases (negative values, zero, max range).

---

## Type Inference

The compiler's `compileExpression` function returns `(value.TypeTag, error)`, giving the type tag of the compiled expression without a separate inference step. This returned tag is used directly when building slot masks for untyped assignments.

| Expression Node | Inferred tag |
|-----------------|--------------|
| `*parser.ByteExpression` | `TagByte` |
| `*parser.ShortExpression` | `TagShort` |
| `*parser.IntegerExpression` | `TagInteger` |
| `*parser.LongExpression` | `TagLong` |
| `*parser.FloatExpression` | `TagFloat` |
| `*parser.DecimalExpression` | `TagDecimal` |
| `*parser.BooleanExpression` | `TagBoolean` |
| `*parser.StringExpression` | `TagSlice` |
| `*parser.PointerExpression` | Tag resolved from the pointer's type name |
| `*parser.IdentifierExpression` | Tag of the referenced variable (looked up in symbol table) |
| `*parser.AttributeExpression` | Tag of the accessed field in the stencil |

`inferTypeTag` is a separate function used only from `compileTupleAssignment` — it performs the same inference but without emitting any bytecode, allowing stencil layout to be pre-computed before element expressions are compiled.

---

## Type Safety

Variables are constrained by a **bitmask** established at first assignment. Each slot stores a mask of allowed type tags, and the runtime enforces that every store matches the mask.

### Single-type (inferred)

Untyped assignments infer the type and create a single-type mask:

```
x = 42       # x mask = 00000010 (int only, 4 bytes)
x = 100      # OK: int is in the mask
x = true     # Runtime error: type mismatch
```

### Union types (explicit)

Typed declarations build a multi-type mask. The slot allocates `max(size)` across all allowed types:

```
y: int|bool = 15    # y mask = 00100010 (int + bool, 4 bytes max)
y = true            # OK: bool is in the mask
y = 3.14f           # Runtime error: float not in the mask
```

### Supported type names for constraints

| Name | Tag |
|------|-----|
| `short` | `TagShort` |
| `int` | `TagInteger` |
| `long` | `TagLong` |
| `float` | `TagFloat` |
| `decimal` | `TagDecimal` |
| `bool` | `TagBoolean` |
| `byte` | `TagByte` |
| `str` / `string` | `TagSlice` (requires `<N>` capacity parameter) |

**Note:** Slice types (`string<N>`, `byte<N>`) bypass the mask system entirely (`SLICE_ALLOC` handles them separately) and cannot participate in union types. The mask for a slice slot is always 0. Attempting to use `string` or `byte` without `<N>` as a constraint in a union type is a compile error.

### How it works

The compiler resolves type constraint identifiers to tags via `TagForName`, builds the mask via `MaskForTag`, and emits the mask in `VAR_ALLOC.Extra`. The runtime's `VAR_STORE` checks `TagInMask(actualTag, slot.Mask)` and rejects mismatches. `SlotEntry.Tag` is updated on every successful store to track the current variant for `VAR_LOAD` decoding.
