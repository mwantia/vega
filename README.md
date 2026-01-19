# Vega

**Vega** (Virtual Execution & Graph Abstraction) is a lightweight scripting language designed for file system operations and automation. It features Python-like syntax, a stack-based virtual machine, and is written in pure Go with no CGO dependencies.

## Features

- **Simple syntax** - Python-like scripting that's easy to learn
- **Interactive REPL** - Explore and experiment with immediate feedback
- **Script execution** - Run `.vega` script files
- **Pure Go** - No CGO, easy cross-compilation
- **Built-in functions** - Rich standard library for strings, arrays, and I/O

## Installation

```bash
go install github.com/mwantia/vega/cmd/vega@latest
```

Or build from source:

```bash
git clone https://github.com/mwantia/vega.git
cd vega
go build -o vega ./cmd/vega
```

## Quick Start

### Interactive REPL

```bash
$ vega
Vega - Virtual Execution & Graph Abstraction
Type 'help' for available commands, 'quit' to exit.

vega> println("Hello, World!")
Hello, World!
vega> x = 42
vega> println(x * 2)
84
vega> quit
Goodbye!
```

### Execute a Command

```bash
vega -c 'println("Hello from Vega!")'
```

### Run a Script

```bash
vega script.vega
```

## Language Guide

### Variables

```vega
name = "Alice"
age = 30
pi = 3.14159
active = true
nothing = nil
```

### Data Types

| Type | Example |
|------|---------|
| Integer | `42`, `-17`, `0` |
| Float | `3.14`, `-0.5` |
| String | `"hello"`, `"world"` |
| Boolean | `true`, `false` |
| Nil | `nil` |
| Array | `[1, 2, 3]` |
| Map | `{name: "Alice", age: 30}` |

### Operators

```vega
# Arithmetic
x = 10 + 5      # 15
x = 10 - 5      # 5
x = 10 * 5      # 50
x = 10 / 5      # 2
x = 10 % 3      # 1
x = -5          # negation

# Comparison
x == y          # equal
x != y          # not equal
x < y           # less than
x <= y          # less or equal
x > y           # greater than
x >= y          # greater or equal

# Logical
a && b          # and (short-circuit)
a || b          # or (short-circuit)
!a              # not

# String concatenation
s = "Hello" + " " + "World"
```

### Control Flow

#### If/Else

```vega
if x > 10 {
    println("big")
} else {
    println("small")
}
```

#### While Loop

```vega
i = 0
while i < 5 {
    println(i)
    i = i + 1
}
```

#### For Loop

```vega
for item in [1, 2, 3, 4, 5] {
    println(item)
}

# With range()
for i in range(10) {
    println(i)
}
```

### Functions

```vega
fn greet(name) {
    println("Hello, " + name + "!")
}

greet("World")

fn add(a, b) {
    return a + b
}

result = add(3, 4)  # 7
```

#### Recursion

```vega
fn factorial(n) {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}

println(factorial(5))  # 120
```

### Arrays

```vega
arr = [1, 2, 3, 4, 5]

# Access elements
first = arr[0]      # 1
arr[0] = 10         # modify

# Built-in functions
len(arr)            # 5
push(arr, 6)        # append
last = pop(arr)     # remove last
contains(arr, 3)    # true
index(arr, 3)       # 2
```

### Maps

```vega
person = {name: "Alice", age: 30, city: "NYC"}

# Access values
name = person["name"]

# Modify
person["age"] = 31

# Built-in functions
len(person)         # 3
k = keys(person)    # ["name", "age", "city"]
```

### Comments

```vega
# This is a comment
x = 42  # inline comment
```

## Built-in Functions

### I/O

| Function | Description |
|----------|-------------|
| `print(args...)` | Print without newline |
| `println(args...)` | Print with newline |
| `input()` | Read line from stdin |
| `input(prompt)` | Print prompt, read line |

### Type Conversion

| Function | Description |
|----------|-------------|
| `type(value)` | Get type name as string |
| `string(value)` | Convert to string |
| `int(value)` | Convert to integer |
| `float(value)` | Convert to float |
| `bool(value)` | Convert to boolean |

### Collections

| Function | Description |
|----------|-------------|
| `len(x)` | Length of string, array, or map |
| `push(arr, val)` | Append to array |
| `pop(arr)` | Remove and return last element |
| `keys(map)` | Get array of map keys |
| `range(n)` | Array [0, 1, ..., n-1] |
| `range(start, end)` | Array [start, ..., end-1] |

### Strings

| Function | Description |
|----------|-------------|
| `upper(s)` | Convert to uppercase |
| `lower(s)` | Convert to lowercase |
| `trim(s)` | Remove leading/trailing whitespace |
| `split(s, sep)` | Split string into array |
| `join(arr, sep)` | Join array into string |
| `contains(s, sub)` | Check if contains substring |
| `startswith(s, pre)` | Check prefix |
| `endswith(s, suf)` | Check suffix |
| `replace(s, old, new)` | Replace all occurrences |
| `index(s, sub)` | Find index of substring (-1 if not found) |

### Utility

| Function | Description |
|----------|-------------|
| `assert(cond)` | Error if condition is false |
| `assert(cond, msg)` | Error with message if false |

## CLI Reference

```
Usage:
  vega                    Start interactive REPL
  vega <script.vega>      Execute a script file
  vega -c '<code>'        Execute a single command
  vega -S <script.vega>   Execute a script file (explicit)

Flags:
  -c string       Execute a single Vega command
  -S string       Execute a Vega script file
  -disasm         Show disassembled bytecode (debug)
  -version        Show version information
  -help           Show help message
```

## REPL Commands

| Command | Description |
|---------|-------------|
| `help` | Show available commands |
| `quit` | Exit the REPL |
| `exit` | Exit the REPL |
| `history` | Show command history |
| `clear` | Clear the screen |

The REPL supports multiline input. Lines ending with `{` continue on the next line until braces are balanced.

## Examples

### FizzBuzz

```vega
for i in range(1, 101) {
    if i % 15 == 0 {
        println("FizzBuzz")
    } else {
        if i % 3 == 0 {
            println("Fizz")
        } else {
            if i % 5 == 0 {
                println("Buzz")
            } else {
                println(i)
            }
        }
    }
}
```

### Fibonacci

```vega
fn fib(n) {
    if n <= 1 {
        return n
    }
    return fib(n - 1) + fib(n - 2)
}

for i in range(15) {
    println(fib(i))
}
```

### Word Counter

```vega
text = "the quick brown fox jumps over the lazy dog"
words = split(text, " ")
println("Word count: " + string(len(words)))

# Count occurrences of "the"
count = 0
for word in words {
    if word == "the" {
        count = count + 1
    }
}
println("Occurrences of 'the': " + string(count))
```

### Simple Calculator

```vega
fn calculate(a, op, b) {
    if op == "+" {
        return a + b
    }
    if op == "-" {
        return a - b
    }
    if op == "*" {
        return a * b
    }
    if op == "/" {
        return a / b
    }
    return nil
}

println(calculate(10, "+", 5))   # 15
println(calculate(10, "*", 3))   # 30
```

### Array Operations

```vega
numbers = [5, 2, 8, 1, 9, 3]

# Find max
max = numbers[0]
for n in numbers {
    if n > max {
        max = n
    }
}
println("Max: " + string(max))

# Sum
sum = 0
for n in numbers {
    sum = sum + n
}
println("Sum: " + string(sum))

# Filter even numbers
evens = []
for n in numbers {
    if n % 2 == 0 {
        push(evens, n)
    }
}
println("Evens: " + string(evens))
```

## Architecture

Vega uses a classic interpreter pipeline:

```
Source Code → Lexer → Tokens → Parser → AST → Compiler → Bytecode → VM → Result
```

| Component | Package | Description |
|-----------|---------|-------------|
| Lexer | `pkg/lexer` | Tokenizes source code |
| Parser | `pkg/parser` | Builds Abstract Syntax Tree |
| Compiler | `pkg/compiler` | Generates bytecode |
| VM | `pkg/vm` | Stack-based bytecode interpreter |
| REPL | `pkg/repl` | Interactive shell |

## License

MIT License - see LICENSE file for details.
