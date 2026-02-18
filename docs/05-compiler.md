# Compiler

**Package:** `pkg/compiler`

The compiler translates an AST into a flat sequence of bytecode instructions. The new compiler introduces a symbol table for tracking variable-to-slot mappings and type inference for determining the binary encoding of each variable.

---

## Compiler State

```go
type Compiler struct {
    scope    *SymbolTable        // nil outside alloc blocks
    stencils map[string]*Stencil // stencil registry (struct definitions)
}
```

The `scope` field is non-nil only while compiling statements inside an `alloc` block body. The `stencils` map persists across `Compile()` calls — struct definitions are global and accumulate.

---

## Symbol Table

```go
type SymbolInfo struct {
    SlotID  int
    Tag     value.TypeTag
    Mask    byte
    Stencil *Stencil  // non-nil for struct/tuple variables
}

type SymbolTable struct {
    symbols  map[string]SymbolInfo
    nextSlot int
}
```

The symbol table maps variable names to slot IDs, type tags, and type bitmasks. Slot IDs are assigned sequentially starting from 0. The `Mask` field stores the bitmask of allowed types for the variable. For struct/tuple variables, `Stencil` points to the layout recipe used for field access resolution.

| Method | Purpose |
|--------|---------|
| `Lookup(name) (SymbolInfo, bool)` | Check if a variable exists in the current scope |
| `Define(name, tag, mask) SymbolInfo` | Register a new variable with type mask, assign the next slot ID |
| `Remove(name)` | Delete a variable from the scope (used by `free()`) |

### Scope Lifecycle

1. `alloc` block entry → `scope = newSymbolTable()`
2. Variable assignments and lookups use this scope.
3. `free(x)` removes `x` from the scope.
4. `alloc` block exit → `scope = nil`

Variables referenced outside an `alloc` block produce a compile error.

---

## Statement Compilation

### AllocStatement

```
alloc <size> { <body> }
```

1. Emit `STACK_ALLOC` with the integer size.
2. Create a new `SymbolTable` scope.
3. Compile each statement in the body.
4. Destroy the scope.
5. Emit `STACK_FREE`.

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

Compilation:

1. **Compile RHS** — pushes the value onto the expression stack.
2. **First assignment?** If `x` is not in the symbol table:
   - **With constraints:** Resolve constraint identifiers to tags via `TagForName`, build the union bitmask via `MaskForTag`, OR the bits together.
   - **Without constraints:** Infer the type tag from the RHS expression, build a single-type mask via `MaskForTag(tag)`.
   - `scope.Define("x", 0, mask)` — assigns the next slot ID with the computed mask.
   - Emit `VAR_ALLOC slot=N mask=M`.
3. **Emit `VAR_STORE slot=N`** — pops the expression stack and writes into the byte buffer. The runtime validates the value's tag against the mask.

### Pointer Alias Assignment

```
y = *int(0)
```

When the RHS of an assignment is a `PointerExpression`, the compiler takes a different path:

1. **Compile the offset expression** — pushes the offset value onto the expression stack.
2. **Resolve type name** — `value.TagForName(typeName)` converts `"int"` → `TagInteger`. Unknown names produce a compile error.
3. **Define symbol** if first assignment — `scope.Define("y", tag, MaskForTag(tag))`.
4. **Emit `VAR_PTR slot=N tag=T`** — the runtime handles the rest.
5. **Return early** — no `VAR_ALLOC` or `VAR_STORE` is emitted.

This means pointer aliases skip the allocator entirely. The offset is runtime-evaluated (it's an expression on the stack), but the type is compile-time resolved.

### Constraint Resolution

The `resolveConstraintMask(constraints []Expression) (byte, error)` function:

1. Iterates over each constraint expression.
2. Asserts it is an `*IdentifierExpression` (type names must be bare identifiers).
3. Resolves the name via `value.TagForName` — returns an error for unknown names like `"foobar"`.
4. ORs `value.MaskForTag(tag)` into the accumulated mask.

### FreeStatement

```
free(x)
```

1. Look up `x` in the symbol table.
2. Emit `VAR_FREE slot=N`.
3. `scope.Remove("x")` — any subsequent reference to `x` is a compile error.

### StructStatement

```
struct point {
    x: int
    y: int
}
```

Pure compile-time declaration — no bytecode emitted.

1. Iterate over field declarations. For each field, resolve the type name to a `TypeTag` and compute the cumulative byte offset.
2. Build a `Stencil{Name, Fields, TotalSize}`.
3. Register in `compiler.stencils[name]`.

**Error:** Unknown type name in field declaration produces a compile error.

### Struct Literal Assignment

```
p = point { x = 10, y = 20 }
```

When the RHS of an assignment is a `StructExpression`:

1. Look up the struct name in `compiler.stencils`. Unknown names produce a compile error.
2. If the variable is new: `scope.Define(name, 0, 0)` with `Stencil` set to the looked-up stencil. Emit `STENCIL_ALLOC slot=N size=TotalSize`.
3. For each field in the literal (in declaration order):
   - Look up the field in the stencil. Unknown field names produce a compile error.
   - Compile the field value expression (pushes onto expr stack).
   - Emit `FIELD_STORE slot=N offset=fieldOffset tag=fieldTag`.

### Tuple Assignment

```
t = (42, true)
```

When the RHS is a `TupleExpression`:

1. Infer the type of each element via `inferTypeTag`.
2. Build an anonymous `Stencil` with positional field names (`"0"`, `"1"`, ...).
3. Same emission pattern as struct literals: `STENCIL_ALLOC` + `FIELD_STORE` per element.

---

## Expression Compilation

### Literals

All literal expressions (`ShortExpression`, `IntegerExpression`, etc.) add the value to the constant pool via `AddConstant()` and emit `LOAD_CONST` with the pool index. Duplicate constants are deduplicated by string comparison.

### IdentifierExpression

Look up the variable name in the symbol table. If found, emit `VAR_LOAD slot=N`. If not found, produce a compile error ("undefined variable").

This means:
- Variables can only be loaded after they've been assigned.
- Variables that have been `free()`d cannot be loaded (removed from symbol table).

### AttributeExpression (Field Access)

```
p.x      # struct field access
t.0      # tuple positional access
```

1. Resolve the object identifier in the symbol table.
2. Assert the symbol has a non-nil `Stencil`.
3. Look up the field name (or positional index) in the stencil. Unknown fields produce a compile error.
4. Emit `FIELD_LOAD slot=N offset=fieldOffset tag=fieldTag`.

The field offset and type tag are fully resolved at compile time — no runtime field lookup occurs.

---

## Type Inference

The `inferTypeTag(expr)` function determines the type tag from a parser expression node:

| Expression Node | Inferred Tag |
|-----------------|--------------|
| `*parser.ByteExpression` | `TagByte` |
| `*parser.ShortExpression` | `TagShort` |
| `*parser.IntegerExpression` | `TagInteger` |
| `*parser.LongExpression` | `TagLong` |
| `*parser.FloatExpression` | `TagFloat` |
| `*parser.DecimalExpression` | `TagDecimal` |
| `*parser.BooleanExpression` | `TagBoolean` |
| `*parser.CharExpression` | `TagChar` |
| `*parser.PointerExpression` | Tag resolved from the pointer's type name |
| `*parser.IdentifierExpression` | Tag of the referenced variable |
| `*parser.AttributeExpression` | Tag of the accessed field in the stencil |

For identifier expressions, the function looks up the referenced variable's tag in the symbol table, enabling type propagation through variable-to-variable assignment. For pointer expressions, the type is resolved from the type name string (e.g., `*int(0)` → `TagInteger`):

```
alloc 64 {
    x = 42;     # x: TagInteger (inferred from IntegerExpression)
    y = x;      # y: TagInteger (inferred from x's symbol info)
}
```

### Unsupported Inference

The following expression types produce a compile error:
- Arithmetic expressions (`x + y`) — would require an operator type promotion system.
- Function calls — would require return type tracking.
- String/nil expressions — not allocable.

These are deferred to future work.

---

## Error Handling

| Error | When |
|-------|------|
| "assignment outside alloc block" | `x = 42` with no active alloc scope |
| "free outside alloc block" | `free(x)` with no active alloc scope |
| "free: undefined variable 'x'" | `free(x)` when `x` is not in the symbol table |
| "cannot infer type for 'x'" | RHS expression type is not inferrable |
| "undefined variable 'x'" | Identifier reference not in symbol table |
| "identifier 'x' outside alloc block" | Identifier reference with no active alloc scope |
| "alloc size must be an integer literal" | `alloc "64" { ... }` or `alloc x { ... }` |
| "unknown type name 'foobar'" | Type constraint references a non-existent type |
| "type constraint must be an identifier" | Non-identifier expression used as a type constraint |
| "unknown type name 'X' in pointer" | Pointer alias references a non-existent type (`*foobar(0)`) |
| "undefined struct type 'X'" | Struct literal references an unregistered struct name |
| "struct 'X' has no field 'Y'" | Field name not found in the stencil (struct literal or field access) |
| "variable 'X' is not a struct or tuple" | Field access on a non-stencil variable |
| "struct 'X': unknown type 'Y' for field 'Z'" | Struct definition uses an unknown type name |

---

## Compilation Examples

### Untyped assignment

Source:
```
alloc 16 { x = 42; free(x); y = 100l; y }
```

Bytecode:
```
0: STACK_ALLOC 16                    # create 16-byte buffer
1: LOAD_CONST 0                      # push 42 (int32)
2: VAR_ALLOC slot=0 mask=00000010    # reserve 4 bytes for slot 0 (int only)
3: VAR_STORE slot=0                  # pop 42, encode, write to buffer[0..4)
4: VAR_FREE slot=0                   # free buffer[0..4), mark slot 0 dead
5: LOAD_CONST 1                      # push 100 (int64)
6: VAR_ALLOC slot=1 mask=00000100    # reserve 8 bytes for slot 1 (long only)
7: VAR_STORE slot=1                  # pop 100, encode, write to buffer[4..12)
8: VAR_LOAD slot=1                   # read buffer[4..12), decode as int64, push
9: STACK_POP                         # discard top of stack (expression statement)
10: STACK_FREE                       # destroy allocator and stack
```

Constants: `[42 (int), 100 (long)]`

### Typed union assignment

Source:
```
alloc 16 { y: int|bool = 15; y = true; y }
```

Bytecode:
```
0: STACK_ALLOC 16                    # create 16-byte buffer
1: LOAD_CONST 0                      # push 15 (int32)
2: VAR_ALLOC slot=0 mask=00100010    # reserve 4 bytes (max of int=4, bool=1), allow int+bool
3: VAR_STORE slot=0                  # pop 15, tag check passes (int in mask), encode, write
4: LOAD_CONST 1                      # push true (boolean)
5: VAR_STORE slot=0                  # pop true, tag check passes (bool in mask), encode, write
6: VAR_LOAD slot=0                   # read buffer, decode as bool (current tag), push
7: STACK_POP                         # discard
8: STACK_FREE                        # destroy allocator and stack
```

Constants: `[15 (int), true (boolean)]`

### Pointer alias assignment

Source:
```
alloc 8 { x = 42; y = *int(0) }
```

Bytecode:
```
0: STACK_ALLOC 8                     # create 8-byte buffer
1: LOAD_CONST 0                      # push 42 (int32)
2: VAR_ALLOC slot=0 mask=00000010    # reserve 4 bytes for slot 0 (int only)
3: VAR_STORE slot=0                  # pop 42, encode, write to buffer[0..4)
4: LOAD_CONST 1                      # push 0 (int32 — the offset)
5: VAR_PTR slot=1 tag=2              # create alias: slot 1 views buffer[0..4) as int
6: STACK_FREE                        # destroy allocator and stack
```

Constants: `[42 (int), 0 (int)]`

Note: No `VAR_ALLOC` or `VAR_STORE` is emitted for slot 1. The `VAR_PTR` instruction sets up the slot entry directly with `Alias=true`.

### Struct definition and use

Source:
```
struct point { x: int, y: int }
alloc 32 { p = point { x = 10, y = 20 }; a = p.x }
```

Bytecode:
```
0: STACK_ALLOC 32                        # create 32-byte buffer
1: LOAD_CONST 0                          # push 10 (int32)
2: STENCIL_ALLOC slot=0 size=8           # allocate 8 bytes for point (4+4)
3: FIELD_STORE slot=0 offset=0 tag=2     # pop 10, write to buffer[0..4) as int
4: LOAD_CONST 1                          # push 20 (int32)
5: FIELD_STORE slot=0 offset=4 tag=2     # pop 20, write to buffer[4..8) as int
6: FIELD_LOAD slot=0 offset=0 tag=2      # read buffer[0..4) as int → push 10
7: VAR_ALLOC slot=1 mask=00000010        # reserve 4 bytes for 'a' (int only)
8: VAR_STORE slot=1                      # pop 10, write to a's slot
9: STACK_FREE                            # destroy allocator and stack
```

Constants: `[10 (int), 20 (int)]`

Note: The `struct point` declaration produces no bytecode — it only registers a stencil in the compiler. Field names (`x`, `y`) are resolved to byte offsets at compile time and do not appear in the bytecode.

### Tuple creation and access

Source:
```
alloc 16 { t = (42, true); a = t.0 }
```

Bytecode:
```
0: STACK_ALLOC 16                        # create 16-byte buffer
1: LOAD_CONST 0                          # push 42 (int32)
2: STENCIL_ALLOC slot=0 size=5           # allocate 5 bytes (4 int + 1 bool)
3: FIELD_STORE slot=0 offset=0 tag=2     # pop 42, write to buffer[0..4) as int
4: LOAD_CONST 1                          # push true (boolean)
5: FIELD_STORE slot=0 offset=4 tag=6     # pop true, write to buffer[4..5) as bool
6: FIELD_LOAD slot=0 offset=0 tag=2      # read buffer[0..4) as int → push 42
7: VAR_ALLOC slot=1 mask=00000010        # reserve 4 bytes for 'a' (int only)
8: VAR_STORE slot=1                      # pop 42, write to a's slot
9: STACK_FREE                            # destroy allocator and stack
```

Constants: `[42 (int), true (boolean)]`

Note: Tuples use anonymous stencils built at compile time. Positional field names (`0`, `1`) are resolved to byte offsets.
