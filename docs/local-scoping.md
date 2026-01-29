# Design: True Function-Local Scoping for Vega

## Problem Statement

Vega claims function-local variables, but the implementation is broken for recursion. Each function call creates a new `CallFrame` with its own `locals` map, however the VM uses `frameIndex` (a simple counter) to decide between local and global storage. When a function recurses, the **inner call's frame overwrites the outer call's variables** because each new call creates a fresh frame at `frameIndex + 1` -- but the real issue is that the `for` loop iterator state and variable assignments in the inner call corrupt the outer call's state once execution returns.

### How It Manifests

```
fn walk(path) {
    entries = readdir(path)       # frame.locals["entries"] = ...
    for entry in entries {        # vm.iterators["entry"] = ...
        if entry["isdir"] {
            walk("/" + entry.key) # NEW frame, but vm.iterators is SHARED
        }                         # After return: "entries" in THIS frame is fine,
    }                             # but vm.iterators["entry"] was deleted by inner call
}
```

The crash occurs because `vm.iterators` is a **single global map** on the VM -- not per-frame. When the inner `walk` call completes its own `for entry in entries` loop, it deletes `vm.iterators["entry"]`, destroying the outer call's active iterator.

### Cross-Function Iterator Collision

The problem is not limited to recursion. **Any** function call from inside a `for` loop will crash if the called function uses a `for` loop with the same variable name:

```
fn pad(str, width) {
    result = string(str)
    for i in range(0, width - result.length()) {   # Uses iterator "i"
        result = " " + result
    }                                               # Deletes vm.iterators["i"]
    return result
}

fn main() {
    for i in range(10) {        # Uses iterator "i"
        println(pad(i, 4))      # pad()'s 'for i' destroys main's 'for i' iterator
    }                           # CRASH: "iterator not initialized: i"
}
```

This is the same root cause: `vm.iterators["i"]` is shared across all call frames.

### Root Causes

There are two distinct problems:

1. **Iterators are global** (`vm.iterators` is a single `map[string]value.Iterator` shared across all frames). Any two active `for` loops with the same variable name -- whether from recursion or from cross-function calls -- corrupt each other's iterators.

2. **No block scoping exists**. All variables inside a function share one flat `locals` map. While this works for non-recursive functions (each call gets a fresh frame), it means `for` blocks, `if` blocks, and `while` blocks cannot have their own scope.

---

## Current Architecture

### Compiler (`pkg/compiler/compiler.go`)

- **No scope tracking**. The `Compiler` struct has no symbol table, no scope stack.
- Variables are compiled as `OpStoreVar` / `OpLoadVar` with just a **name string**.
- Function bodies are compiled by a fresh `Compiler` instance but with no scope metadata.
- Blocks (`compileBlock`) are transparent -- they just inline their statements.

### VM (`pkg/vm/vm.go`)

- **Two-level lookup**: `frame.locals` then `vm.globals`.
- **Store logic**: `if frameIndex > 0` -> locals, else -> globals.
- **Iterators**: Single `vm.iterators` map, keyed by loop variable name.
- **CallFrame**: Has its own `locals map[string]value.Value` but no scope chain.

### Instructions (`pkg/compiler/instruction.go`)

```go
type Instruction struct {
    Op   OpCode
    Arg  int    // numeric argument
    Name string // variable name
    Line int
}
```

No scope depth, no local index, no flags.

---

## Proposed Solution

### Fix 1: Per-Frame Iterators (Minimal, Fixes Recursion)

Move iterators from the VM to the `CallFrame`. This is the **smallest change** that fixes the recursion crash.

#### Changes

**`pkg/vm/types.go`** -- Add iterators to CallFrame:

```go
type CallFrame struct {
    bytecode   *compiler.Bytecode
    ip         int
    bp         int
    locals     map[string]value.Value
    iterators  map[string]value.Iterator  // NEW: per-frame iterators
}
```

**`pkg/vm/vm.go`** -- Remove `iterators` from VirtualMachine struct. Update all iterator access:

```go
// Before:
vm.iterators[instr.Name] = iterable.Iterator()

// After:
frame.iterators[instr.Name] = iterable.Iterator()
```

Applies to `OpIterInit`, `OpIterNext`, and frame initialization in `callFunction`.

**`pkg/vm/vm.go`** -- Initialize iterators map in `callFunction`:

```go
newFrame := &CallFrame{
    bytecode:  fn.Bytecode,
    ip:        0,
    bp:        vm.sp - argCount,
    locals:    make(map[string]value.Value),
    iterators: make(map[string]value.Iterator),  // NEW
}
```

Also initialize for the top-level frame in `Run()`.

#### Impact

- Fixes: Recursive functions with `for` loops.
- Does not fix: True block scoping (loop variables still leak).
- Risk: Very low. Only moves existing data to a different struct.
- Backwards compatible: Yes, no syntax or behavioral changes for non-recursive code.

---

### Fix 2: True Block Scoping (Full Solution)

Add block-level scoping so variables declared inside `if`, `while`, `for` blocks are scoped to that block. This requires compiler and VM changes.

#### 2a. Compiler: Scope Stack

Add a scope stack to the compiler that tracks variable declarations per scope level:

```go
type scope struct {
    variables map[string]int  // name -> local slot index
}

type Compiler struct {
    bytecode *Bytecode
    errors   errors.ErrorList
    scopes   []scope          // NEW: scope stack
    localCount int            // NEW: total locals allocated
}
```

**New opcodes**:

```go
OpPushScope  // Enter a new block scope
OpPopScope   // Exit block scope, discard locals
```

**Variable compilation changes**:

When compiling an assignment, the compiler checks whether the variable already exists in an outer scope. If not, it registers it in the current (innermost) scope. Load/store instructions carry a **scope depth** and **slot index** instead of (or in addition to) a name.

Option A -- index-based locals (faster, more complex):

```go
// Instruction gains scope info
type Instruction struct {
    Op    OpCode
    Arg   int
    Arg2  int    // NEW: scope depth or slot index
    Name  string
    Line  int
}
```

Option B -- name-based with scope push/pop (simpler, adequate):

Keep name-based lookup but add scope push/pop to the VM so it knows which variables to discard. The compiler emits `OpPushScope` at block entry and `OpPopScope` at block exit with the list of variable names to clean up.

#### 2b. VM: Scope Chain in CallFrame

Replace the flat `locals` map with a scope chain:

```go
type CallFrame struct {
    bytecode   *compiler.Bytecode
    ip         int
    bp         int
    scopes     []map[string]value.Value  // NEW: stack of scope maps
    iterators  map[string]value.Iterator
}
```

Variable lookup walks the scope chain from innermost to outermost:

```go
case compiler.OpLoadVar:
    // Search scopes innermost-first
    for i := len(frame.scopes) - 1; i >= 0; i-- {
        if val, ok := frame.scopes[i][instr.Name]; ok {
            vm.push(val)
            found = true
            break
        }
    }
    if !found {
        // Fall back to globals
        if val, ok := vm.globals[instr.Name]; ok {
            vm.push(val)
        } else {
            return fmt.Errorf("undefined variable: %s", instr.Name)
        }
    }
```

Variable store writes to the **innermost scope that already contains the variable**, or the current scope if it's a new variable:

```go
case compiler.OpStoreVar:
    val := vm.pop()
    if vm.frameIndex == 0 {
        vm.globals[instr.Name] = val
    } else {
        // Find existing binding or create in current scope
        stored := false
        for i := len(frame.scopes) - 1; i >= 0; i-- {
            if _, ok := frame.scopes[i][instr.Name]; ok {
                frame.scopes[i][instr.Name] = val
                stored = true
                break
            }
        }
        if !stored {
            // New variable in current (innermost) scope
            frame.scopes[len(frame.scopes)-1][instr.Name] = val
        }
    }
```

Scope push/pop:

```go
case compiler.OpPushScope:
    frame.scopes = append(frame.scopes, make(map[string]value.Value))

case compiler.OpPopScope:
    frame.scopes = frame.scopes[:len(frame.scopes)-1]
```

#### 2c. Compiler: Block Compilation Changes

```go
func (c *Compiler) compileBlock(block *ast.BlockStatement) {
    c.bytecode.Emit(OpPushScope, block.Pos().Line)
    for _, stmt := range block.Statements {
        c.compileStatement(stmt)
    }
    c.bytecode.Emit(OpPopScope, block.Pos().Line)
}
```

Function body gets an implicit scope push (from the frame initialization) so function parameters live in scope 0.

#### Impact

- Fixes: All scoping issues -- recursion, variable leaking, block isolation.
- Risk: Medium. Changes variable resolution semantics.
- Backwards compatibility concern: Code that relies on loop variable leaking will break. Example:

```
for i in range(5) {
    last = i
}
println(last)  # Currently works, would become undefined
```

This could be mitigated by only scoping `for`/`while` loop variables (the iteration variable `i`) but keeping explicit assignments (`last = i`) in the enclosing scope.

---

## Recommended Approach

**Phase 1**: Implement Fix 1 (per-frame iterators). This is a small, safe change that fixes the immediate recursion crash with zero risk of breaking existing scripts.

**Phase 2**: Implement Fix 2 (full block scoping). This is a larger change that should be done carefully with consideration for backwards compatibility.

### Files to Modify

| File | Fix 1 | Fix 2 |
|------|-------|-------|
| `pkg/vm/types.go` | Add `iterators` to CallFrame | Replace `locals` with `scopes` |
| `pkg/vm/vm.go` | Move iterator access to frame | Scope chain lookup/store, push/pop handling |
| `pkg/compiler/compiler.go` | -- | Add scope stack, emit push/pop |
| `pkg/compiler/opcode.go` | -- | Add `OpPushScope`, `OpPopScope` |
| `pkg/compiler/instruction.go` | -- | Possibly add `Arg2` field |

### Testing Strategy

**Fix 1 tests**:
- Recursive function with `for` loop (the original crash case)
- Recursive function with `while` loop
- Mutual recursion between two functions
- Deep recursion (100+ levels)

**Fix 2 tests**:
- Variable declared in `if` block is not visible outside
- Variable declared in `for` body is not visible outside
- Loop variable (`for i in ...`) scoped to loop
- Variable in outer scope readable from inner block
- Assignment to outer scope variable from inner block updates outer
- Nested blocks (3+ levels deep)
- Function parameters survive block scoping
- Global variables still accessible from any scope depth
