# Compiler

**Package:** `pkg/compiler`

The compiler translates an AST into a flat sequence of bytecode instructions. It maintains a symbol table for tracking variable-to-slot mappings, a stencil registry for struct definitions, and a function table for user-defined functions.

---

## Compiler State

```go
type Compiler struct {
    scope    *SymbolTable        // current variable scope (top-level or function)
    stencils map[string]*Stencil // stencil registry (struct definitions)
}
```

The `scope` field is initialized in `Compile()` for the top-level program. When compiling a function body, a fresh `SymbolTable` is created and the outer scope is saved and restored around the function. The `stencils` map persists across `Compile()` calls — struct definitions are global and accumulate (useful in REPL mode where each command is compiled separately).

---

## Symbol Table

```go
type SymbolInfo struct {
    SlotID   int
    Tag      value.TypeTag
    Mask     byte
    Capacity int      // declared byte capacity for slice variables (>0 for string<N>/byte<N>)
    Stencil  *Stencil // non-nil for struct/tuple variables
}

type SymbolTable struct {
    symbols  map[string]SymbolInfo
    nextSlot int
}
```

The symbol table maps variable names to slot IDs, type tags, and type bitmasks. Slot IDs are assigned sequentially starting from 0. For struct/tuple variables, `Stencil` points to the layout recipe used for field access resolution. For slice variables, `Capacity` records the declared `<N>` value.

| Method | Purpose |
|--------|---------|
| `Lookup(name) (SymbolInfo, bool)` | Check if a variable exists in the current scope |
| `Define(name, tag, mask) SymbolInfo` | Register a new scalar variable with type mask, assign the next slot ID |
| `DefineSlice(name, capacity) SymbolInfo` | Register a slice variable (`string<N>`/`byte<N>`) with its declared capacity |
| `DefineStencil(name, stencil) SymbolInfo` | Register a struct/tuple variable with its stencil |
| `Remove(name)` | Delete a variable from the scope (used by `free()`) |

### Scope Lifecycle

1. `Compile()` is called → if `c.scope == nil`, a new `SymbolTable` is created. If `c.scope` already exists (REPL reuse), it is kept to preserve name→slot-ID assignments across commands.
2. Variable assignments and lookups use this scope throughout the compilation.
3. `free(x)` removes `x` from the scope — subsequent references to `x` are compile errors.
4. For function compilation: the outer scope is saved, a new scope is created for the function body, and the outer scope is restored after compilation.

---

## Statement Compilation

### AssignmentStatement

Two forms are supported:

**Untyped (inferred):**
```
x = <expr>
```

**Typed (explicit constraints):**
```
x: int|bool = <expr>
```

The compiler dispatches based on the RHS expression type and constraint:

- **Struct literal** (`x = point { x = 10 }`) → `compileStructAssignment`
- **Tuple** (`x = (42, true)`) → `compileTupleAssignment`
- **Pointer alias** (`y = *int(0)`) → `compilePointerAssignment`
- **Slice constraint** (`x: string<N> = ...` or `x: byte<N> = ...`) → `compileSliceAssignment`
- **Everything else** → `compileScalarAssignment`

**Slice compilation (`compileSliceAssignment`):**

1. Detect `*parser.SliceTypeExpression` on the constraint (e.g. `string<10>` or `byte<20>`).
2. **Compile RHS** via `compileExpression` — must return `TagSlice` or the string literal is a compile error.
3. If the variable is new: call `scope.DefineSlice(name, capacity)`, emit `SLICE_ALLOC slot=N cap=C`.
4. Emit `VAR_STORE slot=N` — the runtime validates content length ≤ capacity (overflow = runtime error) and zero-pads.

**Bare string literal without constraint is a compile error.** `x = "hello"` without an explicit `string<N>` type is rejected at compile time.

**Scalar compilation (`compileScalarAssignment`):**

1. **Compile RHS** via `compileExpression`, which returns `(TypeTag, error)`.
2. **Slice check:** If `rhsTag == TagSlice` and no `SliceTypeExpression` constraint was given, emit a compile error directing the user to add `string<N>` or `byte<N>`.
3. **First assignment?** If the variable is not in the symbol table:
   - **With constraints:** Resolve each constraint to a tag via `TagForName`, build a union bitmask, define the symbol.
   - **Without constraints:** Use the tag returned from `compileExpression`, build a single-type mask.
   - Emit `VAR_ALLOC slot=N mask=M`.
4. **Emit `VAR_STORE slot=N`** — pops the expression stack and writes into the byte buffer. The runtime validates the value's tag against the mask.

### FreeStatement

```
free(x)
```

1. Look up `x` in the symbol table — error if not found.
2. Emit `VAR_FREE slot=N`.
3. `scope.Remove("x")` — any subsequent reference to `x` is a compile error.

### FunctionStatement

```
fn add(a: int, b: int) {
    ...
}
```

1. Resolve each parameter's type constraint: primitive types (e.g., `int`) produce a tag/mask pair; struct types look up the stencil registry.
2. Compile the function body into a separate `ByteCode` object that shares the top-level `Functions` map (enabling mutual recursion).
3. Save the outer scope, create a fresh scope for the function.
4. For each parameter:
   - **Primitive:** Emit `LOAD_ARG index=i`, `VAR_ALLOC slot=N mask=M`, `VAR_STORE slot=N`.
   - **Struct:** Emit `LOAD_ARG_STENCIL index=i slot=N size=TotalSize`.
5. Compile each statement in the function body.
6. Emit `RETURN void` as an implicit tail return.
7. Restore the outer scope and register the `FunctionDef` in `b.Functions`.

### ReturnStatement

```
return <expr>    # return with value
return           # void return
```

- With value: compile the expression (pushes value onto stack), emit `RETURN` with `Extra=1`.
- Without value: emit `RETURN` with `Extra=0`.

### CallStatement

```
fn(arg1, arg2)
```

1. Compile each argument expression (left to right), pushing onto the stack.
2. Check if the function name is in `b.Functions`:
   - Yes → emit `CALL_FN name argc`.
   - No → emit `CALL_NAT name argc`.

The name is interned into `ByteCode.Names[]` and stored as an index in `Instruction.Offset`.

### StructStatement

```
struct point {
    x: int
    y: int
}

struct meta {
    name: string<32>
    size: long
}
```

Pure compile-time declaration — no bytecode emitted.

1. Iterate over field declarations. The parser uses `makeTypeAnnotation()` for each field type, which handles both plain `IDENT` (e.g. `int`) and `IDENT<N>` (e.g. `string<32>`).
2. For each field:
   - **Scalar field:** Resolve the type name to a `TypeTag` via `TagForName`. Field width = `SizeForTag(tag)`. If `SizeForTag` returns 0 for a non-slice type, this is a compile error.
   - **Slice field** (parsed as `SliceTypeExpression`): `Tag=TagSlice`, `Capacity=N`, field width = N.
3. Build a `Stencil{Name, Fields, TotalSize}` where `Fields` is `[]FieldLayout{Name, Offset, Tag, Capacity}`.
4. Register in `compiler.stencils[name]`.

**Error:** Unknown type name or missing `<N>` on a slice field produces a compile error.

### Struct Literal Assignment

```
p = point { x = 10, y = 20 }
m = meta { name = "hello", size = 64l }
```

1. Look up the struct name in `compiler.stencils`.
2. If the variable is new: call `emitStencilInit` to define the symbol and emit `STENCIL_ALLOC slot=N size=TotalSize`.
3. For each field in the literal (in declaration order):
   - Look up the field in the stencil (`FieldLayout{Name, Offset, Tag, Capacity}`).
   - Compile the field value expression (pushes onto expr stack).
   - **Scalar field:** Emit `FIELD_STORE slot=N offset=fieldOffset tag=fieldTag` (via `EmitField`).
   - **Slice field** (`Capacity > 0`): Emit `FIELD_STORE slot=N offset=fieldOffset tag=TagSlice size=Capacity` (via `EmitFieldSize`).

### Tuple Assignment

```
t = (42, true)
```

1. Pre-scan element types via `inferTypeTag` (without emitting code) to build an anonymous stencil.
2. Call `emitStencilInit` to define the symbol and emit `STENCIL_ALLOC slot=N size=TotalSize`.
3. Compile each element expression and emit `FIELD_STORE` per element with positional field names (`"0"`, `"1"`, ...).

### Pointer Alias Assignment

```
y = *int(0)
```

1. Compile the offset expression — pushes the offset value onto the expression stack.
2. Resolve type name → tag via `value.TagForName`.
3. Define the symbol if first assignment.
4. Emit `VAR_PTR slot=N tag=T` — the runtime handles the rest.

No `VAR_ALLOC` or `VAR_STORE` is emitted. The offset is runtime-evaluated; the type is compile-time resolved.

---

## Expression Compilation

`compileExpression(b, expr) (value.TypeTag, error)` emits code to push a value onto the expression stack and returns the value's type tag. The returned tag is used by callers (e.g., `compileScalarAssignment`) to make typing decisions without a separate inference pass.

### Literals

All literal expressions add the value to the constant pool via `AddConstant()` and emit `LOAD_CONST`. Duplicate constants are deduplicated by tag + data comparison.

Boolean constants (`true`/`false`) are package-level pre-allocated values (`boolConstTrue`, `boolConstFalse`) to avoid per-literal `[]byte` allocations.

### IdentifierExpression

Look up the variable name in the symbol table:
- **Stencil variable:** Emit `VAR_LOAD_RAW slot=N size=TotalSize` (pushes a `RawValue` for struct argument passing).
- **Regular variable:** Emit `VAR_LOAD slot=N`.
- **Not found:** Compile error ("undefined variable").

### AttributeExpression (Field Access)

```
p.x      # struct field access
t.0      # tuple positional access
m.name   # slice field access
```

1. Resolve the object identifier in the symbol table.
2. Assert the symbol has a non-nil `Stencil`.
3. Look up the field name/index in the stencil (`FieldLayout{Name, Offset, Tag, Capacity}`).
4. **Scalar field:** Emit `FIELD_LOAD slot=N offset=fieldOffset tag=fieldTag` (via `EmitField`).
5. **Slice field** (`Capacity > 0`): Emit `FIELD_LOAD slot=N offset=fieldOffset tag=TagSlice size=Capacity` (via `EmitFieldSize`).

Field offsets, type tags, and capacities are fully resolved at compile time — no runtime field lookup occurs.

### PointerExpression (as value)

```
*int(offset)    # used as an expression, not an assignment RHS
```

1. Compile the offset expression.
2. Resolve the type name → tag.
3. Emit `PTR_LOAD` with `Extra=tag`.

This is distinct from pointer alias assignment — it reads a value transiently without creating a named slot.

---

## Type Inference

`inferTypeTag(expr) (TypeTag, error)` determines the type tag of an expression **without emitting bytecode**. It is called only from `compileTupleAssignment` to pre-scan element types before any code is emitted (the stencil layout must be known before `STENCIL_ALLOC` can be emitted).

For all other cases, `compileExpression` returns the tag as a side-channel output — no separate inference step is needed.

---

## Constraint Resolution

`resolveConstraintMask(constraints []Expression) (byte, error)`:

1. Iterates over each constraint expression.
2. Asserts it is an `*IdentifierExpression` (bare type names for scalar union types). Returns an error if a `*SliceTypeExpression` (`string<N>`) is found — slice types cannot participate in union types.
3. Resolves the name via `value.TagForName` — returns an error for unknown names.
4. ORs `value.MaskForTag(tag)` into the accumulated mask.

`makeTypeAnnotation()` is the parser helper that produces either an `*IdentifierExpression` or a `*SliceTypeExpression` from type annotation positions (struct fields, function parameters, variable constraints). It handles `IDENT<N>` using the lexer's `LT`/`GT` tokens directly — not via the expression precedence climber, which would interpret `<` as comparison.

---

## Stencil Registration API

Stencils can also be registered programmatically before compilation, without requiring a `struct` declaration in the script:

```go
c := compiler.NewCompiler()
c.RegisterStencil("point",
    compiler.Field("x", value.TagInteger),
    compiler.Field("y", value.TagInteger),
)
```

This is useful for host-defined types that scripts can use without declaring the struct themselves.

---

## Error Handling

| Error | When |
|-------|------|
| "free: undefined variable 'x'" | `free(x)` when `x` is not in the symbol table |
| "undefined variable 'x'" | Identifier reference not in symbol table |
| "cannot infer type from expression %T" | RHS expression type is not inferrable in `inferTypeTag` |
| "unknown type name 'foobar'" | Type constraint references a non-existent type |
| "type constraint must be an identifier" | Non-identifier (e.g. `SliceTypeExpression`) used in a union type constraint |
| "slice type 'string<N>' cannot be used in union type constraint" | `string<N>` or `byte<N>` used inside `int\|string<N>` union |
| "unknown type name 'X' in pointer" | Pointer alias references a non-existent type (`*foobar(0)`) |
| "undefined struct type 'X'" | Struct literal references an unregistered struct name |
| "struct 'X' has no field 'Y'" | Field name not found in the stencil |
| "variable 'X' is not a struct or tuple" | Field access on a non-stencil variable |
| "struct 'X': unknown type 'Y' for field 'Z'" | Struct definition uses an unknown type name |
| "struct 'X': field 'Y' of type 'slice' requires capacity annotation (e.g. string<N>)" | Slice field declared without `<N>` |
| "string literal requires an explicit type constraint (e.g. x: string<N> = ...)" | `x = "..."` without a `string<N>` constraint |
| "function 'X': parameter 'Y' must have exactly one type constraint" | Parameter declared without or with multiple types |
| "unknown statement type: %T" | AST node not yet handled by the compiler |

---

## Compilation Examples

### Untyped assignment

Source:
```
x = 42
free(x)
y = 100l
```

Bytecode:
```
0: LOAD_CONST index=0                # push 42 (int32)
1: VAR_ALLOC slot=0 mask=00000010   # reserve 4 bytes for slot 0 (int only)
2: VAR_STORE slot=0                 # pop 42, encode, write to buffer
3: VAR_FREE slot=0                  # free buffer region, mark slot 0 dead
4: LOAD_CONST index=1               # push 100 (int64)
5: VAR_ALLOC slot=1 mask=00000100   # reserve 8 bytes for slot 1 (long only)
6: VAR_STORE slot=1                 # pop 100, encode, write to buffer
```

Constants: `[42 (int), 100 (long)]`

### Typed union assignment

Source:
```
y: int|bool = 15
y = true
```

Bytecode:
```
0: LOAD_CONST index=0               # push 15 (int32)
1: VAR_ALLOC slot=0 mask=00100010   # reserve 4 bytes (max of int=4, bool=1), allow int+bool
2: VAR_STORE slot=0                 # pop 15, tag check passes (int in mask), write
3: LOAD_CONST index=1               # push true (boolean)
4: VAR_STORE slot=0                 # pop true, tag check passes (bool in mask), write
```

Constants: `[15 (int), true (boolean)]`

### Slice assignment

Source:
```
msg: string<10> = "hello"
msg = "world"
```

Bytecode:
```
0: LOAD_CONST index=0               # push "hello" (slice)
1: SLICE_ALLOC slot=0 cap=10        # first assignment: alloc 10 bytes for slot 0
2: VAR_STORE slot=0                 # pop "hello", copy 5 bytes, zero-pad bytes [5..10)
3: LOAD_CONST index=1               # push "world" (slice)
4: VAR_STORE slot=0                 # reassignment: copy 5 bytes, zero-pad remainder
```

Constants: `["hello" (slice), "world" (slice)]`

Note: `SLICE_ALLOC` only appears on the first assignment (when the slot does not yet exist). Subsequent stores use `VAR_STORE` directly; the capacity is already recorded in the slot.

### Pointer alias

Source:
```
x = 42
y = *int(0)
```

Bytecode:
```
0: LOAD_CONST index=0               # push 42 (int32)
1: VAR_ALLOC slot=0 mask=00000010   # reserve 4 bytes for slot 0 (int only)
2: VAR_STORE slot=0                 # pop 42, write to buffer[0..4)
3: LOAD_CONST index=1               # push 0 (int32 — the offset)
4: VAR_PTR slot=1 tag=2             # create alias: slot 1 views buffer[0..4) as int
```

Constants: `[42 (int), 0 (int)]`

### Function definition and call

Source:
```
fn double(n: int) {
    return n
}
double(21)
```

Top-level bytecode:
```
0: LOAD_CONST index=0               # push 21 (int32)
1: CALL_FN double argc=1            # call user function 'double', argc=1
```

Function 'double' bytecode:
```
0: LOAD_ARG index=0                 # push pendingArgs[0] (the int 21)
1: VAR_ALLOC slot=0 mask=00000010   # alloc slot for parameter 'n'
2: VAR_STORE slot=0                 # store int into slot 0
3: VAR_LOAD slot=0                  # push n's value (21)
4: RETURN value                     # return with value on stack
5: RETURN void                      # implicit end-of-function return
```

### Struct definition and use

Source:
```
struct point { x: int, y: int }
p = point { x = 10, y = 20 }
a = p.x
```

Bytecode:
```
0: LOAD_CONST index=0               # push 10 (int32)
1: STENCIL_ALLOC slot=0 size=8      # allocate 8 bytes for point (4+4)
2: FIELD_STORE slot=0 offset=0 tag=2 size=0  # pop 10, write to buffer[0..4) as int
3: LOAD_CONST index=1               # push 20 (int32)
4: FIELD_STORE slot=0 offset=4 tag=2 size=0  # pop 20, write to buffer[4..8) as int
5: FIELD_LOAD slot=0 offset=0 tag=2 size=0   # read buffer[0..4) as int → push 10
6: VAR_ALLOC slot=1 mask=00000010   # reserve 4 bytes for 'a' (int only)
7: VAR_STORE slot=1                 # pop 10, write to a's slot
```

For a struct with a `string<20>` field (e.g. `struct meta { name: string<20>, size: long }`), the stencil total size is 20+8=28 bytes, and `FIELD_STORE`/`FIELD_LOAD` for the `name` field use `tag=TagSlice size=20`.

Constants: `[10 (int), 20 (int)]`

Note: The `struct point` declaration produces no bytecode — it only registers a stencil. Field names, offsets, tags, and capacities are all resolved at compile time and do not appear in the bytecode.
