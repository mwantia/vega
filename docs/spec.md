# Vega × VFS Specification

**Status:** Stable design baseline  
**Audience:** Vega/VFS maintainers, driver authors  
**Scope:** Vega execution model, memory semantics, VFS contract  
**Non-goals:** General-purpose scripting, memory safety, long-lived data processing

---

## 1. Purpose

Vega is a **borrowed-memory orchestration DSL** designed exclusively to control
and coordinate operations within `vfs`.

Vega scripts:
- move data *through* VFS
- do not own or persist data
- are short-running and synchronous
- are failure-oriented and explicit

Vega is not a general-purpose programming language.

---

## 2. Execution Model

- Single-threaded
- Synchronous
- Non-reentrant
- No background or daemon semantics
- Script execution ends on completion or hard error

---

## 3. Memory Model

### 3.1 Borrowed Memory (Core Rule)

All data read from VFS is represented as **borrowed memory**.

There is no concept of owned data in Vega.

---

### 3.2 `slice` Type

A `slice` is a borrowed, non-owning view into memory owned by:
- a handler
- a stream
- static VM memory (string literals only)

Properties:
- zero-copy
- non-owning
- immutable by default
- invalidated by owner lifecycle events

Prohibited:
- storing in arrays or maps
- returning from functions
- storing in globals
- use after invalidation

Violations result in a **hard VM abort**.

---

### 3.3 Lifetime & Invalidation

A `slice` is valid **only while its owner is alive**, unless invalidated earlier.

Invalidation occurs when:
- owning handler closes
- owning stream closes
- metadata transaction commits
- metadata transaction aborts
- hard VM error occurs

After invalidation, all access is undefined and aborts execution.

---

### 3.4 Error Paths

On hard error:
- all slices are immediately invalidated
- execution aborts
- no recovery semantics are defined

---

## 4. Text & Literals

String literals are:
- `slice`s
- backed by static VM memory
- immutable
- valid for entire script execution

---

## 5. Error Model

### 5.1 `result` Type

Side-effecting operations return a `result`:

```
result.success : boolean
result.error : string
```

### 5.2 Failure Semantics

- Failed `result` may be inspected
- Non-`result` failures cause hard VM abort
- No exception handling exists

---

## 6. VFS Interaction Layers

1. Administrative
2. Atomic data access
3. Stateful handlers
4. Metadata transactions

Each layer has independent semantics.

---

## 7. VFS API Contract

### 7.1 Administrative Operations

```
mount(path, uri) -> result
umount(path) -> result
close(path) -> result # force-close all handlers
sync(path) -> result
```

`close(path)`:
- forcefully closes all handlers for the path
- invalidates all derived slices
- may interrupt in-flight I/O

---

### 7.2 Atomic Data Operations

```
read(path, offset?, size?) -> slice
write(path, offset?, slice) -> integer
```

Properties:
- stateless
- atomic
- no consistency guarantee with open handlers
- backend-optimized when possible

---

### 7.3 Stateful Handlers

```
h = open(path, flags)
```

Handler operations:

```
h.read(size?) -> slice
h.write(slice) -> integer
h.seek(offset) -> result
h.sync() -> result
h.close() -> result
```

Closing a handler invalidates all slices it produced.

---

### 7.4 Metadata Operations

```
meta = lookup(path)

meta.get(key) -> value
meta.set(key, value)
meta.list() -> map
meta.commit() -> result
meta.refresh() -> result
```


Metadata operations are transactional.
`commit()` is atomic and invalidates all related slices.

---

## 8. Consistency & Atomicity

### 8.1 Consistency

- Reads reflect last write in same handler
- Cross-handler consistency is backend-defined
- Metadata consistency is backend-defined

### 8.2 Atomicity (Script Perspective)

| Operation         | Atomic |
|------------------|--------|
| read             | yes    |
| write            | yes    |
| rename           | no     |
| remove           | no     |
| metadata commit  | yes    |
| directory list   | no     |

Partial reads/writes are expected.

---

## 9. Concurrency Rules

- Multiple readers: allowed
- Single writer: allowed
- Multiple writers: forbidden
- Read/write concurrency: backend-defined

Violations may return errors or abort execution.

---

## 10. Driver Contract

### 10.1 Memory Rules

Drivers:
- may return borrowed memory
- may pool and reuse memory
- must not reuse memory until owner invalidation
- must not assume slice persistence

### 10.2 Handler Rules

Drivers may:
- implement optimized streaming handlers
- bypass atomic read/write paths

Drivers must:
- declare capabilities accurately
- enforce constraints
- invalidate slices on lifecycle events

---

## 11. Non-Goals

Vega does not provide:
- memory safety guarantees
- owned data types
- GC-based lifetime extension
- general-purpose language features

Correctness is the responsibility of the script author.

2. Go Interface Examples (Contract-Aligned)

These are illustrative, not prescriptive.

```go
// Slice represents borrowed memory.
// Lifetime is bound to the owning object.
type Slice interface {
	Bytes() []byte // may become invalid after owner invalidation
	Len() int
}
```

```go
// Handler represents a stateful I/O object.
type Handler interface {
	Read(size int64) (Slice, error)
	Write(data Slice) (int64, error)
	Seek(offset int64) error
	Sync() error
	Close() error
}
```

```go
// Atomic object storage operations.
type ObjectStorage interface {
	ReadAt(path string, offset, size int64) (Slice, error)
	WriteAt(path string, offset int64, data Slice) (int64, error)
}
```

```go
// Metadata transaction.
type Metadata interface {
	Get(key string) (any, error)
	Set(key string, value any) error
	List() (map[string]any, error)
	Commit() error
	Refresh() error
}
```

```go
// Driver-facing contract.
type Driver interface {
	Name() string
	Capabilities() DriverCapabilities

	OpenHandler(path string, flags OpenFlags) (Handler, error)
	ObjectStorage() ObjectStorage
	Metadata() Metadata
}
```