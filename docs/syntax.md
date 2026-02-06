# Vega Language Reference

Vega is a dynamically-typed scripting language with Python-like syntax, designed for filesystem automation through VFS (Virtual Filesystem) integration. It compiles to stack-based bytecode and runs on a virtual machine.

## Table of Contents

- [Syntax Rework](#syntax-rework)
- [Syntax Basics](#syntax-basics)
- [Data Types](#data-types)
- [Variables](#variables)
- [Operators](#operators)
- [Control Flow](#control-flow)
- [Functions](#functions)
- [Strings](#strings)
- [Arrays](#arrays)
- [Maps](#maps)
- [Streams](#streams)
- [Metadata](#metadata)
- [Time](#time)
- [Built-in Functions](#built-in-functions)
- [VFS Functions](#vfs-functions)
- [Pipe Operator](#pipe-operator)
- [Truthy and Falsy Values](#truthy-and-falsy-values)
- [Error Handling](#error-handling)
- [Complete Examples](#complete-examples)

---

## Syntax Rework

## Methods and Calls

| FullName         | Alias        | Parameters                       | Return Types | Description                                                      |
|------------------|--------------|----------------------------------|--------------|------------------------------------------------------------------|
| `vfs.mount()`    | `mount()`    | `"path"`, `"uri"`                | `result`     | Attaches a filesystem handler at the specified path              |
| `vfs.umount()`   | `umount()`   | `"path"`                         | `result`     | Removes the filesystem handler at the specified path             |
| `vfs.open()`     | `open()`     | `"path"`, `<mode>`, `[flags]`    | `handler`    | Opens a new filehandler with specified access mode flag          |
| `vfs.close()`    | `close()`    | `"path"`                         | `result`     | Closes all open handlers associated with the specified path      |
| `vfs.create()`   | `create()`   | `"path"`, `<flags>`              | `result`     | Create a file/folder for the specified path and return result    |
| `vfs.read()`     | `read()`     | `"path"`, `<offset>`, `<size>`   | `buffer`     | Reads size bytes from the file at path starting at offset        |
| `vfs.write()`    | `write()`    | `"path"`, `<offset>`, `<buffer>` | `integer`    | Writes data to the file at path starting at offset               |
| `vfs.sync()`     | `sync()`     | `"path"`                         | `result`     | Synchronizes all buffers and commits for the specified path      |
| `vfs.lookup()`   | `lookup()`   | `"path"`                         | `metadata`   | Returns file information/metadata for the given path             |
| `vfs.exists()`   | `exists()`   | `"path"`                         | `boolean`    | Checks if a file or directory exists at the given path           |
| `vfs.rename()`   | `rename()`   | `"old_path"`, `"new_path"`       | `result`     | Moves or renames a file or directory from onePath to another     |
| `vfs.remove()`   | `remove`     | `"path"`, `[force]`              | `result`     | Removes a file or directory at the specified path                |
| `vfs.list()`     | `list()`     | `"path"`                         | `array`      | Returns a list of file metadata in the directory at path         |
| `vfs.getattr()`  | `getattr()`  | `"path"`, `"key"`                | `string`     | Read file metadata for the specified path and key                |
| `vfs.setattr()`  | `getattr()`  | `"path"`, `"key"`, `<value>`     | `result`     | Writes/Updates file metadata for the specified path and key      |
| `vfs.listattr()` | `listattr()` | `"path"`                         | `map`        | List all file attribute map for the specified path               |

## New extended types

| Type        | Description                                 | Name          | Examples                                                            |
|-------------|---------------------------------------------|---------------|---------------------------------------------------------------------|
| `metadata`  | Metadata handler for transaction operations | `"metadata"`  | `meta.set("content-type", "text/html")`, `meta.commit() (result)`   |
| `slice`    | Current solution for data like `[]byte`     | `"slice"`    | `slice.seek(0) (boolean)`, `slice.wtext("Hello World") (integer)` |
| `result`    | Replacement for `error` or `exception`      | `"result"`    | `result.success (boolean)`, `result.error (string)`                 |
| `handler`   | I/O filehandler for stream operations       | `"handler"`   | `file.canwrite (boolean)`, `file.read(512) (integer)`               | 

### Metadata

#### Properties

| Name        | Type      | Description                       |
|-------------|-----------|-----------------------------------|
| id          | `string`  | Unique identifier for metadata    |
| key         | `string`  | Relative key within the service   |
| mode        | `integer` | Unix-style mode and permissions   |
| size        | `long`    | Size in bytes (0 for directories) |
| accesstime  | `time`    |                                   |
| modifytime  | `time`    |                                   |
| createtime  | `time`    |                                   |
| uid         | `long`    | User ownership identity           |
| gid         | `long`    | Group ownership identity          |
| contenttype | `string`  | Content MIME type                 |
| attributes  | `map`     | Extended attributes               |
| etag        | `string`  | Entity tag hash                   |

#### Methods

| Name   | Parameters         | Return Type | Description |
|--------|--------------------|-------------|-------------|
| get    | `string`           | `string`    |             |
| set    | `string`, `string` |             |             |
| remove | `string`           | `result`    |             |
| list   |                    | `map`       |             |
| commit |                    | `result`    |             |
| revert |                    | `result`    |             |

### Slice

#### Properties

| Name        | Type      | Description                       |
|-------------|-----------|-----------------------------------|
| id          | `string`  | Unique identifier for metadata    |
| key         | `string`  | Relative key within the service   |
| mode        | `integer` | Unix-style mode and permissions   |
| size        | `long`    | Size in bytes (0 for directories) |
| accesstime  | `time`    |                                   |
| modifytime  | `time`    |                                   |
| createtime  | `time`    |                                   |
| uid         | `long`    | User ownership identity           |
| gid         | `long`    | Group ownership identity          |
| contenttype | `string`  | Content MIME type                 |
| attributes  | `map`     | Extended attributes               |
| etag        | `string`  | Entity tag hash                   |

#### Methods

| Name   | Parameters         | Return Type | Description |
|--------|--------------------|-------------|-------------|
| read   | `integer`          | `slice`     |             |
| write  | `string`, `string` |             |             |
| seek   | `string`           | `result`    |             |
| copy   |                    | `map`       |             |

### Result

#### Properties

| Name        | Type      | Description                       |
|-------------|-----------|-----------------------------------|
| id          | `string`  | Unique identifier for metadata    |
| key         | `string`  | Relative key within the service   |
| mode        | `integer` | Unix-style mode and permissions   |
| size        | `long`    | Size in bytes (0 for directories) |
| accesstime  | `time`    |                                   |
| modifytime  | `time`    |                                   |
| createtime  | `time`    |                                   |
| uid         | `long`    | User ownership identity           |
| gid         | `long`    | Group ownership identity          |
| contenttype | `string`  | Content MIME type                 |
| attributes  | `map`     | Extended attributes               |
| etag        | `string`  | Entity tag hash                   |

#### Methods

| Name   | Parameters         | Return Type | Description |
|--------|--------------------|-------------|-------------|
| get    | `string`           | `string`    |             |
| set    | `string`, `string` |             |             |
| remove | `string`           | `result`    |             |
| list   |                    | `map`       |             |
| commit |                    | `result`    |             |
| revert |                    | `result`    |             |

### Handler

#### Properties

| Name        | Type      | Description                       |
|-------------|-----------|-----------------------------------|
| id          | `string`  | Unique identifier for metadata    |
| key         | `string`  | Relative key within the service   |
| mode        | `integer` | Unix-style mode and permissions   |
| size        | `long`    | Size in bytes (0 for directories) |
| accesstime  | `time`    |                                   |
| modifytime  | `time`    |                                   |
| createtime  | `time`    |                                   |
| uid         | `long`    | User ownership identity           |
| gid         | `long`    | Group ownership identity          |
| contenttype | `string`  | Content MIME type                 |
| attributes  | `map`     | Extended attributes               |
| etag        | `string`  | Entity tag hash                   |

#### Methods

| Name   | Parameters         | Return Type | Description |
|--------|--------------------|-------------|-------------|
| get    | `string`           | `string`    |             |
| set    | `string`, `string` |             |             |
| remove | `string`           | `result`    |             |
| list   |                    | `map`       |             |
| commit |                    | `result`    |             |
| revert |                    | `result`    |             |

## Syntax Basics

### Comments

Single-line comments start with `#`:

```
# This is a comment
x = 42  # Inline comment
```

### Statement Separators

Statements are separated by newlines or semicolons:

```
x = 1
y = 2

# Or on one line:
x = 1; y = 2
```

### Blocks

Code blocks are delimited by curly braces `{}`:

```
if x > 0 {
    println("positive")
}
```

### Identifiers

Identifiers start with a letter or underscore, followed by letters, digits, or underscores:

```
x
my_variable
_private
count2
```

---

## Data Types

| Type       | Description               | Literal Examples              | Type Name (from `type()`) |
|------------|---------------------------|-------------------------------|---------------------------|
| `string`   | Text                      | `"hello"`, `"line\n"`         | `"string"`                |
| `integer`  | 32-bit signed integer     | `42`, `0`, `-5`               | `"integer"`               |
| `short`    | 16-bit signed integer     | (created internally)          | `"short"`                 |
| `long`     | 64-bit signed integer     | (created internally)          | `"long"`                  |
| `float`    | 64-bit floating point     | `3.14`, `0.5`, `-1.7`        | `"float"`                 |
| `boolean`  | True or false             | `true`, `false`               | `"boolean"`               |
| `nil`      | Null/absence of value     | `nil`                         | `"nil"`                   |
| `array`    | Ordered collection        | `[1, 2, 3]`, `[]`            | `"array"`                 |
| `map`      | Key-value pairs           | `{name: "alice", age: 30}`    | `"map"`                   |
| `stream`   | I/O stream                | (created via `open()`, etc.)  | `"stream"`                |
| `metadata` | File metadata             | (created via `stat()`, etc.)  | `"metadata"`              |
| `time`     | Point in time             | (created via `now()`)         | `"time"`                  |

---

## Variables

Variables are dynamically typed. Assignment creates or updates a variable:

```
x = 42
name = "alice"
items = [1, 2, 3]
```

Top-level variables are global. Variables inside functions are local to that function's call frame. There is no block-level scoping -- `if`, `while`, and `for` blocks share the enclosing function's (or global) scope.

```
x = "global"

fn example() {
    y = "local"     # Only accessible inside this function
    println(x)      # Can read global variables
}
```

**Important caveat -- flat function scope**: All variables within a function share a single flat scope. This means recursive calls overwrite the same locals. Avoid recursion when the function relies on local state across the recursive call. Use iterative approaches with an explicit stack instead. See the [Scoping Limitations](#scoping-limitations) section for details.

Variable lookup order:
1. Current function's locals (if inside a function)
2. Global variables
3. Error if not found

---

## Operators

### Arithmetic

| Operator | Description    | Example       | Notes                              |
|----------|----------------|---------------|------------------------------------|
| `+`      | Add            | `3 + 4`       | Also concatenates strings          |
| `-`      | Subtract       | `10 - 3`      | Also unary negation: `-x`          |
| `*`      | Multiply       | `2 * 5`       |                                    |
| `/`      | Divide         | `10 / 3`      | Integer division for integers      |
| `%`      | Modulo         | `10 % 3`      | Remainder after division           |

Integer + Float operations always produce a Float result.

### Comparison

| Operator | Description        | Example     |
|----------|--------------------|-------------|
| `==`     | Equal              | `x == 5`    |
| `!=`     | Not equal          | `x != 5`    |
| `<`      | Less than          | `x < 10`    |
| `>`      | Greater than       | `x > 0`     |
| `<=`     | Less than/equal    | `x <= 100`  |
| `>=`     | Greater than/equal | `x >= 0`    |

### Logical

| Operator | Description | Example          | Notes              |
|----------|-------------|------------------|--------------------|
| `&&`     | Logical AND | `a && b`         | Short-circuit      |
| `\|\|`   | Logical OR  | `a \|\| b`       | Short-circuit      |
| `!`      | Logical NOT | `!condition`     | Unary prefix       |

Short-circuit: `&&` stops if left side is falsy; `||` stops if left side is truthy.

### Other

| Operator | Description    | Example           |
|----------|----------------|--------------------|
| `\|`     | Pipe           | `value \| func()`  |
| `[]`     | Index          | `arr[0]`, `m["k"]` |
| `.`      | Member/method  | `obj.name`, `s.upper()` |

### Operator Precedence (lowest to highest)

1. `|` (pipe)
2. `||` (logical OR)
3. `&&` (logical AND)
4. `==`, `!=` (equality)
5. `<`, `>`, `<=`, `>=` (comparison)
6. `+`, `-` (addition, subtraction)
7. `*`, `/`, `%` (multiplication, division, modulo)
8. `-x`, `!x` (unary negation, logical NOT)
9. `()`, `.method()`, `[index]` (call, method, index)

---

## Control Flow

### If / Else

```
if x > 0 {
    println("positive")
}

if x > 0 {
    println("positive")
} else {
    println("non-positive")
}
```

Nested if/else for multiple branches:

```
if x > 100 {
    println("large")
} else {
    if x > 10 {
        println("medium")
    } else {
        println("small")
    }
}
```

### While Loop

```
x = 0
while x < 10 {
    println(x)
    x = x + 1
}
```

### For Loop

Iterates over iterable values (arrays, maps, ranges):

```
# Iterate over array
for item in [1, 2, 3] {
    println(item)
}

# Iterate over range
for i in range(5) {
    println(i)    # Prints 0, 1, 2, 3, 4
}

# Iterate over map (yields keys as strings)
m = {name: "alice", age: 30}
for key in m {
    println(key)         # "name", "age"
    println(m[key])      # "alice", 30
}
```

### Break and Continue

```
for i in range(100) {
    if i == 5 {
        break       # Exit the loop
    }
    if i % 2 == 0 {
        continue    # Skip to next iteration
    }
    println(i)      # Prints 1, 3
}
```

---

## Functions

### Definition

```
fn greet(name) {
    println("Hello, ${name}!")
}

fn add(a, b) {
    return a + b
}
```

### Calling

```
greet("Alice")
result = add(3, 4)    # result = 7
```

### Return Values

Functions return `nil` by default. Use `return` to return a value:

```
fn square(x) {
    return x * x
}

fn nothing() {
    # Implicitly returns nil
}
```

### Recursion

```
fn factorial(n) {
    if n <= 1 {
        return 1
    }
    return n * factorial(n - 1)
}

println(factorial(5))  # 120
```

### First-Class Functions

Functions can be stored in variables and passed as values:

```
fn apply(f, x) {
    return f(x)
}

fn double(x) {
    return x * 2
}

result = apply(double, 5)  # 10
```

---

## Strings

### Literals

Strings use double quotes with escape sequences:

```
plain = "hello world"
escaped = "line1\nline2"
tab = "col1\tcol2"
quote = "she said \"hi\""
backslash = "path\\to\\file"
dollar = "price: \${99}"
```

**Supported escape sequences**: `\n` (newline), `\t` (tab), `\r` (carriage return), `\\` (backslash), `\"` (quote), `\$` (literal dollar sign)

### String Interpolation

Embed expressions inside strings with `${}`:

```
name = "Alice"
age = 30
println("Name: ${name}, Age: ${age}")
println("Sum: ${1 + 2}")
println("Upper: ${name.upper()}")
```

### Concatenation

```
greeting = "Hello" + " " + "World"
```

### String Methods

| Method                        | Returns   | Description                                  |
|-------------------------------|-----------|----------------------------------------------|
| `s.length()`                  | integer   | Number of characters                         |
| `s.upper()`                   | string    | Uppercase copy                               |
| `s.lower()`                   | string    | Lowercase copy                               |
| `s.trim()`                    | string    | Remove leading/trailing whitespace            |
| `s.split(separator)`          | array     | Split into array of strings                  |
| `s.contains(substring)`       | boolean   | Check if substring exists                    |
| `s.startswith(prefix)`        | boolean   | Check if starts with prefix                  |
| `s.endswith(suffix)`          | boolean   | Check if ends with suffix                    |
| `s.replace(old, new)`         | string    | Replace all occurrences of `old` with `new`  |
| `s.index(substring)`          | integer   | Index of first occurrence (-1 if not found)  |
| `s.trimprefix(prefix)`        | string    | Remove prefix if present                     |
| `s.trimsuffix(suffix)`        | string    | Remove suffix if present                     |

```
text = "Hello, World!"
println(text.length())          # 13
println(text.upper())           # HELLO, WORLD!
println(text.lower())           # hello, world!
println(text.contains("World")) # true
println(text.split(", "))       # [Hello, World!]
println(text.replace("World", "Vega"))  # Hello, Vega!
println(text.index("World"))    # 7
```

---

## Arrays

### Creation

```
empty = []
numbers = [1, 2, 3, 4, 5]
mixed = ["hello", 42, true, nil]
```

### Indexing

0-based indexing. Out-of-bounds access produces an error.

```
arr = [10, 20, 30]
first = arr[0]     # 10
last = arr[2]      # 30
arr[1] = 99        # [10, 99, 30]
```

### Array Methods

| Method                  | Returns   | Description                                      |
|-------------------------|-----------|--------------------------------------------------|
| `a.length()`            | integer   | Number of elements                               |
| `a.push(value)`         | nil       | Append element to end (mutates array)            |
| `a.pop()`               | value     | Remove and return last element (mutates array)   |
| `a.join(separator)`     | string    | Join elements as strings with separator          |
| `a.contains(value)`     | boolean   | Check if value exists in array                   |
| `a.index(value)`        | integer   | Index of first occurrence (-1 if not found)      |

```
arr = [1, 2, 3]
arr.push(4)                 # arr is now [1, 2, 3, 4]
last = arr.pop()            # last = 4, arr is [1, 2, 3]
println(arr.length())       # 3
println(arr.join(", "))     # "1, 2, 3"
println(arr.contains(2))   # true
println(arr.index(3))      # 2
```

### Iteration

```
for item in [10, 20, 30] {
    println(item)
}
```

---

## Maps

### Creation

Map keys are identifiers (unquoted) in literals but are stored as strings:

```
empty = {}
person = {name: "Alice", age: 30, active: true}
```

### Indexing

Access values with string keys. Returns `nil` for missing keys.

```
m = {name: "Alice", age: 30}
println(m["name"])     # Alice
println(m["missing"])  # nil

m["email"] = "alice@example.com"  # Add new key
m["age"] = 31                     # Update existing key
```

### Map Methods

| Method          | Returns   | Description                          |
|-----------------|-----------|--------------------------------------|
| `m.length()`    | integer   | Number of key-value pairs            |
| `m.keys()`      | array     | Array of all keys (insertion order)  |

```
m = {x: 1, y: 2, z: 3}
println(m.length())    # 3
println(m.keys())      # [x, y, z]
```

### Iteration

Iterating over a map yields keys as strings:

```
m = {name: "Alice", age: 30}
for key in m {
    println("${key}: ${m[key]}")
}
# Output:
# name: Alice
# age: 30
```

Maps maintain insertion order.

---

## Streams

Streams represent I/O channels (files, stdin/stdout/stderr). They are created via `open()`, `stdin()`, `stdout()`, or `stderr()`.

### Members (read-only)

Access with dot notation:

| Member      | Type    | Description                    |
|-------------|---------|--------------------------------|
| `.name`     | string  | Stream name                    |
| `.canread`  | boolean | Whether stream supports reading |
| `.canwrite` | boolean | Whether stream supports writing |
| `.closed`   | boolean | Whether stream is closed       |

### Stream Methods

| Method                   | Returns   | Description                                    |
|--------------------------|-----------|------------------------------------------------|
| `s.read()`               | string    | Read all available data                        |
| `s.readln()`             | string    | Read single line (without newline)             |
| `s.readn(n)`             | string    | Read exactly `n` bytes                         |
| `s.write(data)`          | integer   | Write data, returns bytes written              |
| `s.writeln(data)`        | integer   | Write data followed by newline                 |
| `s.copy(dest)`           | long      | Copy all data to destination stream            |
| `s.flush()`              | nil       | Flush buffered data                            |
| `s.close()`              | nil       | Close the stream                               |

```
# Read from stdin
stream = stdin()
line = stream.readln()

# Write to a file
file = open("/path/to/file.txt", "w")
file.writeln("Hello, World!")
file.close()

# Copy stdin to a file
input = stdin()
output = open("/data/output.txt", "w")
input.copy(output)
output.close()
```

### Boolean Evaluation

A stream evaluates to `true` if open, `false` if closed.

---

## Metadata

Metadata objects represent file system entry information. Created by `stat()` or returned from `readdir()`.

### Members (dot access)

| Member         | Type    | Description                    | Writable |
|----------------|---------|--------------------------------|----------|
| `.id`          | string  | File identifier                | yes      |
| `.key`         | string  | File path/key                  | yes      |
| `.mode`        | integer | File mode/permissions          | yes      |
| `.size`        | long    | File size in bytes             | yes      |
| `.accesstime`  | string  | Last access time (RFC3339)     | no       |
| `.modifytime`  | string  | Last modify time (RFC3339)     | no       |
| `.createtime`  | string  | Creation time (RFC3339)        | no       |
| `.uid`         | long    | User ID                        | yes      |
| `.gid`         | long    | Group ID                       | yes      |
| `.contentType` | string  | MIME content type              | yes      |
| `.etag`        | string  | Entity tag                     | yes      |

### Indexed Access (bracket notation)

Bracket notation provides additional derived fields:

| Field            | Type    | Description                    |
|------------------|---------|--------------------------------|
| `["filetype"]`   | string  | Type of file                   |
| `["isdir"]`      | boolean | Is a directory                 |
| `["isfile"]`     | boolean | Is a regular file              |
| `["ismount"]`    | boolean | Is a mount point               |
| `["issymlink"]`  | boolean | Is a symbolic link             |
| `["attributes"]` | map     | Extended attributes            |

All member fields are also accessible via bracket notation.

```
meta = stat("/myfile.txt")
println(meta.key)            # /myfile.txt
println(meta.size)           # 1234
println(meta["isdir"])       # false
println(meta["isfile"])      # true
println(meta.modifytime)     # 2024-01-15T10:30:00Z
```

---

## Time

Time values represent a point in time. Created via `now()`.

### Members (read-only)

| Member       | Type    | Description                   |
|--------------|---------|-------------------------------|
| `.year`      | integer | Year                          |
| `.month`     | integer | Month (1-12)                  |
| `.day`       | integer | Day of month                  |
| `.hour`      | integer | Hour (0-23)                   |
| `.minute`    | integer | Minute (0-59)                 |
| `.second`    | integer | Second (0-59)                 |
| `.weekday`   | integer | Day of week (0=Sunday)        |
| `.yearday`   | integer | Day of year                   |
| `.unix`      | integer | Unix timestamp (seconds)      |
| `.unixmilli` | integer | Unix timestamp (milliseconds) |
| `.unixnano`  | integer | Unix timestamp (nanoseconds)  |

### Time Methods

| Method               | Returns | Description                        |
|----------------------|---------|------------------------------------|
| `t.format(layout)`   | string  | Format time using Go layout string |
| `t.utc()`            | time    | Convert to UTC                     |
| `t.local()`          | time    | Convert to local timezone          |

### Time Arithmetic

```
t = now()
later = t + 3600        # Add 3600 seconds (1 hour)
earlier = t - 60        # Subtract 60 seconds

t1 = now()
t2 = now()
diff = t2 - t1          # Difference in seconds (float)
```

### Format Layouts

Uses Go time format layouts. Reference time is `Mon Jan 2 15:04:05 MST 2006`:

```
t = now()
println(t.format("2006-01-02"))           # 2024-06-15
println(t.format("15:04:05"))             # 14:30:00
println(t.format("2006-01-02-15-04-05"))  # 2024-06-15-14-30-00
```

### Comparison

```
t1 = now()
t2 = now()
if t2 > t1 {
    println("time has passed")
}
```

---

## Built-in Functions

### I/O

| Function              | Returns | Description                                   |
|-----------------------|---------|-----------------------------------------------|
| `print(...values)`    | nil     | Print values separated by spaces (no newline) |
| `println(...values)`  | nil     | Print values separated by spaces with newline |
| `input([prompt])`     | string  | Read line from stdin, optional prompt         |

```
print("Hello", "World")    # Hello World (no newline)
println("Hello", "World")  # Hello World\n
name = input("Name: ")     # Prints "Name: ", reads line
```

### Stream Constructors

| Function     | Returns | Description                    |
|--------------|---------|--------------------------------|
| `stdin()`    | stream  | Read-only standard input       |
| `stdout()`   | stream  | Write-only standard output     |
| `stderr()`   | stream  | Write-only standard error      |

### Type Inspection and Conversion

| Function         | Returns | Description                                      |
|------------------|---------|--------------------------------------------------|
| `type(value)`    | string  | Get the type name of a value                     |
| `string(value)`  | string  | Convert to string                                |
| `integer(value)` | integer | Convert to integer (truncates floats, parses strings) |
| `float(value)`   | float   | Convert to float (parses strings)                |
| `boolean(value)` | boolean | Convert to boolean (truthy/falsy)                |

```
println(type(42))           # integer
println(type("hello"))      # string
println(type([1, 2]))       # array

println(integer("42"))      # 42
println(integer(3.14))      # 3
println(integer(true))      # 1
println(integer(false))     # 0

println(float("3.14"))      # 3.14
println(float(42))          # 42

println(string(42))         # "42"
println(boolean(0))         # false
println(boolean(1))         # true
```

### Utility

| Function                     | Returns | Description                               |
|------------------------------|---------|-------------------------------------------|
| `now()`                      | time    | Get current time                          |
| `range(n)`                   | array   | Array `[0, 1, ..., n-1]`                 |
| `range(start, end)`          | array   | Array `[start, start+1, ..., end-1]`     |
| `assert(condition, [message])` | nil   | Error if condition is false               |

```
println(range(5))       # [0, 1, 2, 3, 4]
println(range(2, 5))    # [2, 3, 4]

assert(1 + 1 == 2)
assert(x > 0, "x must be positive")
```

---

## VFS Functions

These functions require a VFS (Virtual Filesystem) to be attached to the VM. They operate on VFS paths.

### File Reading and Writing

| Function                          | Returns | Description                      |
|-----------------------------------|---------|----------------------------------|
| `read(path)`                      | string  | Read entire file content         |
| `read(path, offset, size)`        | string  | Read `size` bytes from `offset`  |
| `write(path, data)`              | long    | Write data to file, returns bytes written |
| `write(path, data, offset)`     | long    | Write at specific offset         |

```
content = read("/data/config.txt")
write("/data/output.txt", "Hello, World!")

# Read 100 bytes starting at position 50
chunk = read("/data/large.bin", 50, 100)

# Write at offset
write("/data/file.txt", "inserted", 10)
```

### File Streams

| Function               | Returns | Description                          |
|------------------------|---------|--------------------------------------|
| `open(path)`           | stream  | Open file for reading (default)      |
| `open(path, mode)`     | stream  | Open file with specified mode        |

**File modes**:

| Mode        | Description                          |
|-------------|--------------------------------------|
| `"r"`       | Read only                            |
| `"w"`       | Write, create file, truncate if exists |
| `"a"`       | Append, create file if not exists    |
| `"rw"`, `"r+"` | Read and write, create if not exists |
| `"wx"`      | Write, create, fail if file exists   |

```
# Read a file via stream
f = open("/data/file.txt", "r")
content = f.read()
f.close()

# Write to a file via stream
f = open("/data/output.txt", "w")
f.writeln("Line 1")
f.writeln("Line 2")
f.close()

# Append to a file
f = open("/data/log.txt", "a")
f.writeln("New log entry")
f.close()
```

### File Metadata

| Function         | Returns  | Description                        |
|------------------|----------|------------------------------------|
| `stat(path)`     | metadata | Get file/directory metadata        |
| `lookup(path)`   | boolean  | Check if path exists               |

```
if lookup("/data/config.txt") {
    meta = stat("/data/config.txt")
    println("Size: ${meta.size}")
    println("Modified: ${meta.modifytime}")
}
```

### Directory Operations

| Function                  | Returns | Description                               |
|---------------------------|---------|-------------------------------------------|
| `readdir(path)`           | array   | Array of metadata objects for entries     |
| `createdir(path)`         | nil     | Create directory                          |
| `remdir(path)`            | nil     | Remove empty directory                    |
| `remdir(path, true)`      | nil     | Remove directory recursively              |

```
createdir("/data/newdir")

entries = readdir("/data")
for entry in entries {
    println("${entry.key} - ${entry.size} bytes")
}

remdir("/data/newdir")          # Must be empty
remdir("/data/olddir", true)    # Force recursive delete
```

### File Management

| Function                   | Returns | Description                 |
|----------------------------|---------|-----------------------------|
| `unlink(path)`             | nil     | Delete a file               |
| `rename(oldpath, newpath)` | nil     | Rename or move file/directory |

```
unlink("/data/temp.txt")
rename("/data/old.txt", "/data/new.txt")
```

### Hashing

| Function                              | Returns | Description                           |
|---------------------------------------|---------|---------------------------------------|
| `etag(path)`                          | string  | Calculate MD5-based ETag for file     |
| `etag(path, chunk_size)`              | string  | ETag with custom chunk size (e.g., "8MB") |
| `etag(path, chunk_size, use_cached)`  | string  | Use cached ETag from metadata if available |

### Execution

| Function             | Returns | Description                    |
|----------------------|---------|--------------------------------|
| `exec(command, ...)` | integer | Execute VFS command, returns exit code |

---

## Pipe Operator

The pipe operator `|` passes the left-hand value to the right-hand expression:

```
value | function()
```

---

## Truthy and Falsy Values

Only two values are **falsy**:
- `false`
- `nil`

Everything else is **truthy**, including:
- `0` (zero integer) is truthy: `boolean(0)` returns `false`, but `0` as a value is truthy
- `""` (empty string) evaluates based on length: empty string is falsy
- `[]` (empty array) is falsy
- `{}` (empty map) is falsy

Specific type rules:
- **String**: truthy if length > 0
- **Integer/Float**: truthy if not zero
- **Array**: truthy if not empty
- **Map**: truthy if not empty
- **Stream**: truthy if open (not closed)
- **Metadata**: truthy if key is not empty
- **Time**: truthy if not zero time

---

## Error Handling

Vega does not have try/catch. Runtime errors terminate script execution with an error message including line and column information.

Error types:
- **SyntaxError** - Invalid token (e.g., unclosed string)
- **ParseError** - Invalid grammar (e.g., missing brace)
- **CompileError** - Invalid bytecode generation
- **TypeError** - Type mismatch at runtime (e.g., adding string to integer)
- **RuntimeError** - General runtime failure (e.g., division by zero, index out of bounds)

Use `assert()` to validate conditions:

```
assert(x > 0, "x must be positive")
assert(type(name) == "string", "name must be a string")
```

Use `lookup()` to check file existence before operations:

```
if lookup("/data/file.txt") {
    content = read("/data/file.txt")
} else {
    println("File not found")
}
```

---

## Scoping Limitations

Vega currently has a **flat function scope** -- all variables within a function share a single locals map per call frame. There is no block scoping and no per-call-frame isolation during recursion.

### Recursion breaks local state

When a function calls itself recursively, the inner call **overwrites** the outer call's local variables because all invocations at the same frame depth share global state for their locals:

```
# BROKEN - do not use this pattern
fn walk(path, result) {
    entries = readdir(path)          # Sets global 'entries'
    for entry in entries {           # Iterates over 'entries'
        if entry["isdir"] {
            walk("/" + entry.key, result)  # Overwrites 'entries' and 'entry'!
        }
    }
    # After recursive return, 'entries' now points to inner call's data
    # The for-loop iterator is invalid -> runtime error
}
```

### Workaround: iterative approach with explicit stack

Replace recursion with a `while` loop and a stack array. Use index-based iteration instead of `for-in` to avoid iterator corruption:

```
# CORRECT - iterative with explicit stack
fn walk(start, result) {
    dirs = [start]
    while dirs.length() > 0 {
        dir = dirs.pop()
        entries = readdir(dir)
        idx = 0
        while idx < entries.length() {
            e = entries[idx]
            if e["isdir"] {
                dirs.push("/" + e.key)
            } else {
                result.push(e)
            }
            idx = idx + 1
        }
    }
}
```

### For-loop variable name collisions across functions

`for` loop iterators are stored in a single global map keyed by variable name. If a `for` loop body calls a function that also uses a `for` loop with the **same variable name**, the called function's loop destroys the caller's iterator:

```
# BROKEN - 'for i' in pad_right collides with 'for i' in main
fn pad_right(str, width) {
    result = string(str)
    for i in range(0, width - result.length()) {
        result = " " + result
    }
    return result
}

fn main() {
    for i in range(10) {
        println(pad_right(i, 4))  # CRASH: pad_right's 'for i' destroys this 'for i'
    }
}
```

**Workaround**: Use `while` loops with manual counters when the loop body calls functions that may use `for` loops:

```
fn main() {
    n = 0
    while n < 10 {
        println(pad_right(n, 4))  # Safe - no iterator to corrupt
        n = n + 1
    }
}
```

### Loop variables leak into enclosing scope

Variables declared in `for` or `while` blocks are visible after the loop ends:

```
for i in range(5) {
    last = i
}
println(last)  # 4 -- 'last' persists
println(i)     # 4 -- loop variable persists
```

---

## Complete Examples

### FizzBuzz

```
fn fizzBuzz() {
    for i in range(1, 101) {
        if i % 15 == 0 {
            print("FizzBuzz")
        } else {
            if i % 3 == 0 {
                print("Fizz")
            } else {
                if i % 5 == 0 {
                    print("Buzz")
                } else {
                    print(i)
                }
            }
        }
        print(",")
    }
    println()
}

fizzBuzz()
```

### Directory Listing (ls-like)

```
# Format file size to human-readable format
fn format_size(size) {
    if size < 1024 {
        return "${size}B"
    }
    if size < 1048576 {
        kb = size / 1024
        return "${kb}K"
    }
    if size < 1073741824 {
        mb = size / 1048576
        return "${mb}M"
    }
    gb = size / 1073741824
    return "${gb}G"
}

fn list(path) {
    entries = readdir(path)
    println("total ${entries.length()}")

    for entry in entries {
        size_str = format_size(entry.size)
        name = entry.key
        filetype = entry["filetype"]
        println("${filetype}  ${size_str}  ${name}")
    }
}

list("/data")
```

### File Upload (stdin to VFS)

```
stream = stdin()

timestamp = now().format("2006-01-02-15-04-05")
file = open("/uploads/${timestamp}.txt", "w")

stream.copy(file)
file.close()

println("Uploaded to /uploads/${timestamp}.txt")
```

### String Processing

```
text = "  Hello, World! Welcome to Vega.  "

# Trim and split
cleaned = text.trim()
words = cleaned.split(" ")

println("Word count: ${words.length()}")

# Filter and transform
for word in words {
    if word.length() > 4 {
        println(word.upper())
    }
}

# Search and replace
result = cleaned.replace("World", "Universe")
println(result)

# Check contents
if cleaned.contains("Vega") {
    println("Found Vega!")
}
```

### Working with Maps

```
# Build a frequency counter
words = ["apple", "banana", "apple", "cherry", "banana", "apple"]
counts = {}

for word in words {
    if counts[word] == nil {
        counts[word] = 0
    }
    counts[word] = counts[word] + 1
}

# Print results
for key in counts {
    println("${key}: ${counts[key]}")
}
```

### Recursive File Tree

```
fn print_tree(path, indent) {
    entries = readdir(path)
    for entry in entries {
        name = entry.key.trimprefix(path).trimprefix("/")
        if entry["isdir"] {
            println("${indent}[DIR] ${name}/")
            print_tree("${path}/${name}", indent + "  ")
        } else {
            size = format_size(entry.size)
            println("${indent}${name} (${size})")
        }
    }
}

fn format_size(size) {
    if size < 1024 {
        return "${size}B"
    }
    return "${size / 1024}K"
}

print_tree("/data", "")
```

### Time Operations

```
start = now()
println("Started at: ${start.format("15:04:05")}")
println("Year: ${start.year}, Month: ${start.month}, Day: ${start.day}")

# UTC conversion
utc_time = start.utc()
println("UTC: ${utc_time.format("2006-01-02T15:04:05Z")}")

# Unix timestamp
println("Unix: ${start.unix}")
```
