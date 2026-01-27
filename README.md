# Vega

**Vega** (Virtual Execution & Graph Abstraction) is a lightweight scripting language designed for file system operations and automation. It features Python-like syntax, a stack-based virtual machine, and is written in pure Go with no CGO dependencies.

**Version**: 0.0.1-dev

## Features

- **Simple syntax** - Python-like scripting that's easy to learn
- **Interactive TUI REPL** - Full-featured terminal UI with history search, autocomplete, and bytecode inspection
- **Script execution** - Run `.vega` script files
- **Pure Go** - No CGO, easy cross-compilation
- **Built-in functions** - Rich standard library for strings, arrays, I/O, and VFS operations
- **VFS integration** - Access multiple storage backends (local filesystem, SQLite, PostgreSQL, S3, Consul)

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
```

The TUI REPL provides:
- Command history with search (Ctrl+R)
- Tab completion for keywords and built-ins
- Scrollable output (mouse wheel)
- Bytecode disassembly panel (Ctrl+D)
- Multiline input (auto-detected by brace matching)

### Execute a Command

```bash
vega -c 'println("Hello from Vega!")'
```

### Run a Script

```bash
vega script.vega
# or explicitly:
vega -s script.vega
```

### With VFS Mount

```bash
# Mount ephemeral (in-memory) filesystem
vega ephemeral://

# Mount local directory
vega file:///path/to/dir

# Mount SQLite database
vega sqlite:///path/to/db.sqlite
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
| `input([prompt])` | Read line from stdin, optionally print prompt |

### Streams

| Function | Description |
|----------|-------------|
| `stdin()` | Get stdin as a stream |
| `stdout()` | Get stdout as a stream |
| `stderr()` | Get stderr as a stream |

### Type Conversion

| Function | Description |
|----------|-------------|
| `type(value)` | Get type name as string |
| `string(value)` | Convert to string |
| `integer(value)` | Convert to integer |
| `float(value)` | Convert to float |
| `boolean(value)` | Convert to boolean |

### Utility

| Function | Description |
|----------|-------------|
| `range(n)` | Array [0, 1, ..., n-1] |
| `range(start, end)` | Array [start, ..., end-1] |
| `assert(cond)` | Error if condition is false |
| `assert(cond, msg)` | Error with message if false |

### VFS Operations

| Function | Description |
|----------|-------------|
| `read(path[, offset, size])` | Read file contents |
| `write(path, data[, offset])` | Write data to file, returns bytes written |
| `stat(path)` | Get file metadata |
| `lookup(path)` | Check if path exists (returns boolean) |
| `readdir(path)` | List directory contents (array of metadata) |
| `createdir(path)` | Create directory |
| `remdir(path[, force])` | Remove directory |
| `unlink(path)` | Delete file |
| `rename(old, new)` | Rename/move file or directory |
| `open(path[, mode])` | Open file stream (modes: "r", "w", "a", "rw", "wx") |
| `exec(cmd, args...)` | Execute VFS command, returns exit code |
| `sexec(cmd, args...)` | Execute with stdin/stdout/stderr streams |
| `capture(cmd, args...)` | Execute and capture output as string |
| `etag(path[, size])` | Calculate or retrieve file ETag |

## Type Methods

Values support method calls using dot notation (e.g., `str.upper()`).

### Universal Methods (All Types)

These methods are available on all value types:

| Method | Description |
|--------|-------------|
| `v.string()` | Convert value to string |
| `v.type()` | Get type name |
| `v.boolean()` | Convert to boolean (truthy/falsy) |
| `v.equal(other)` | Check equality with another value |
| `v.compare(other)` | Compare values (-1, 0, 1) for comparable types |

### String Methods

| Method | Description |
|--------|-------------|
| `s.length()` | Get string length |
| `s.upper()` | Convert to uppercase |
| `s.lower()` | Convert to lowercase |
| `s.trim()` | Remove leading/trailing whitespace |
| `s.split(sep)` | Split into array |
| `s.contains(sub)` | Check if contains substring |
| `s.startswith(prefix)` | Check prefix |
| `s.endswith(suffix)` | Check suffix |
| `s.replace(old, new)` | Replace all occurrences |
| `s.index(sub)` | Find index of substring (-1 if not found) |

### Array Methods

| Method | Description |
|--------|-------------|
| `arr.length()` | Get array length |
| `arr.push(val)` | Append value to array |
| `arr.pop()` | Remove and return last element |
| `arr.join(sep)` | Join elements into string |
| `arr.contains(val)` | Check if array contains value |
| `arr.index(val)` | Find index of value (-1 if not found) |

### Map Methods

| Method | Description |
|--------|-------------|
| `m.length()` | Get number of key-value pairs |
| `m.keys()` | Get array of keys |

### Stream Methods

Streams are returned by `open()`, `stdin()`, `stdout()`, `stderr()`.

| Method | Description |
|--------|-------------|
| `stream.canread()` | Check if stream is readable |
| `stream.canwrite()` | Check if stream is writable |
| `stream.isclosed()` | Check if stream is closed |
| `stream.read()` | Read all available data |
| `stream.readln()` | Read a single line |
| `stream.readn(n)` | Read n bytes |
| `stream.write(data)` | Write data, returns bytes written |
| `stream.writeln(data)` | Write data with newline |
| `stream.copy(dest)` | Copy all data to destination stream |
| `stream.flush()` | Flush buffered data |
| `stream.close()` | Close the stream |

### Metadata Fields

Metadata values (from `stat()`, `readdir()`) support field access via indexing:

```vega
meta = stat("/file.txt")
println(meta["key"])      # file path
println(meta["size"])     # file size
println(meta["isdir"])    # is directory?
```

| Field | Description |
|-------|-------------|
| `id` | Unique identifier |
| `key` | File path |
| `mode` | Permission mode |
| `size` | File size in bytes |
| `accesstime` | Last access time (RFC3339) |
| `modifytime` | Last modification time (RFC3339) |
| `createtime` | Creation time (RFC3339) |
| `uid` | Owner user ID |
| `gid` | Owner group ID |
| `contenttype` | MIME content type |
| `etag` | Entity tag |
| `filetype` | Type string ("file", "dir", etc.) |
| `isdir` | Is directory (boolean) |
| `isfile` | Is regular file (boolean) |
| `ismount` | Is mount point (boolean) |
| `issymlink` | Is symbolic link (boolean) |
| `attributes` | Extended attributes (map) |

## CLI Reference

```
Usage:
  vega [uri]               Start REPL with optional VFS mount (default: ephemeral://)
  vega -s <script.vega>    Execute a script file
  vega -c '<code>'         Execute a single command

Flags:
  -c string       Execute a single Vega command
  -s string       Execute a Vega script file
  -i              Keep REPL open after executing script/command
  -d              Show disassembled bytecode (debug)
  --version       Show version information
  --help          Show help message
```

## REPL Commands

| Command | Description |
|---------|-------------|
| `help` or `?` | Show available commands |
| `quit` | Exit the REPL |
| `exit` | Exit the REPL |
| `history` | Show command history |
| `clear` | Clear the screen |
| `vars` | Show defined variables |

## REPL Key Bindings

| Key | Action |
|-----|--------|
| `Ctrl+R` | Search command history |
| `Ctrl+L` | Clear screen |
| `Ctrl+O` | Toggle expression result display |
| `Ctrl+D` | Toggle bytecode disassembly panel |
| `Ctrl+U` | Clear current line |
| `Ctrl+K` | Kill to end of line |
| `Ctrl+C` | Interrupt execution or quit |
| `Tab` | Autocomplete |
| `Up/Down` | Navigate history |
| `Mouse wheel` | Scroll output |

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

### VFS File Operations

```vega
# Write a file
write("/hello.txt", "Hello, World!")

# Read it back
content = read("/hello.txt")
println(content)

# List directory
files = readdir("/")
for f in files {
    println(f)
}

# Get file metadata
meta = stat("/hello.txt")
println("Size: " + string(meta["size"]))
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
| REPL | `pkg/repl` | TUI-based interactive shell |

## License

MIT License - see LICENSE file for details.
