# Language Comparison

How Vega's memory model differs from other languages and why.

---

## The Core Idea

Most scripting languages store variables as pointers to heap-allocated objects managed by a garbage collector. Vega stores allocable variables as raw bytes inside a flat `[]byte` buffer managed by a free list allocator. Variable names are erased at compile time — the runtime operates on slot IDs and byte offsets.

This is unusual for a scripting language. It's a design you'd expect from a systems language or an embedded VM, not a language with Python-like syntax.

---

## Comparison Matrix

| Feature | Vega | Python | Lua | C | Rust | WebAssembly |
|---------|------|--------|-----|---|------|-------------|
| Variable storage | Byte array | Heap objects | Heap objects | Stack/heap | Stack/heap | Linear memory |
| Memory management | Explicit `free()` + block scoping | GC (reference counting + cycle collector) | GC (incremental mark-and-sweep) | Manual `malloc`/`free` | Ownership + borrow checker | Host-managed |
| Type of variables | Constrained by bitmask (fixed or union) | Dynamic, any reassignment | Dynamic, any reassignment | Static, declared | Static, declared | Static, declared |
| Name resolution | Compile-time (slot IDs) | Runtime (dict lookup) | Runtime (table lookup) | Compile-time (stack offsets) | Compile-time (stack offsets) | Compile-time (local indices) |
| Fragmentation handling | Free list coalescing | N/A (GC compacts) | N/A (GC compacts) | System allocator (replaceable) | System allocator (replaceable) | N/A (linear growth) |
| OOM behavior | Runtime error | MemoryError exception | Lua panic | Undefined / segfault | Panic or Result | Trap |

---

## What Makes Vega Different

### 1. Explicit Memory Budgets in a Scripting Language

In Python, Lua, JavaScript — you declare variables freely and the runtime figures out memory. In C and Rust, you think about memory but work with types and the compiler decides sizes.

Vega is unique: **the programmer declares a byte budget, and the runtime allocates within it**.

```
alloc 64 {
    x = 42;       # 4 bytes (int32)
    y = 3.14;     # 8 bytes (float64)
    z = true;     # 1 byte (bool)
}                  # Total: 13 bytes of 64 used
```

This is not a type annotation system. It's a memory region with a hard limit. If your variables exceed the budget, you get a runtime error. This forces the programmer to think about how much data they're working with — which is exactly the right constraint for a VFS orchestration DSL where scripts should be small and short-lived.

### 2. Variables Are Byte Offsets, Not Objects

In Python, `x = 42` creates a `PyObject` on the heap with a reference count, type pointer, and value field. In Lua, `x = 42` creates a tagged value in a table. In both cases, the variable name exists at runtime as a string key in a dictionary or table.

In Vega, `x = 42` means:
1. The compiler assigns slot ID 0 with tag `TagInteger`.
2. At runtime, the allocator reserves 4 bytes at some offset.
3. The integer `42` is encoded as 4 little-endian bytes and copied into the buffer.
4. The name "x" does not exist at runtime. Only `slot=0` exists.

This is the same approach as WebAssembly's local variables or a register allocator in a compiled language — but applied to a scripting language with dynamic syntax.

### 3. Explicit Deallocation Without Ownership or Borrowing

Rust gives you safe memory management through ownership and borrowing, enforced at compile time. C gives you `malloc`/`free` with no safety net.

Vega's `free()` sits between these:
- Like C, deallocation is explicit and immediate.
- Unlike C, the runtime detects use-after-free and double-free (as compile errors when possible, as runtime errors otherwise).
- Unlike Rust, there's no borrow checker or lifetime annotations.

```
alloc 64 {
    x = 42;
    free(x);     # Bytes returned to the free list
    x;           # Compile error: undefined variable 'x'
}
```

The compiler removes freed variables from the symbol table, so most dangling references are caught at compile time. This is simpler than Rust's borrow checker but catches the most common error class.

### 4. Type Tags and Bitmasks Instead of Object Headers

In a GC'd language, every value carries a type tag (or is a pointer to a typed object). The tag is stored alongside the data.

In Vega, type information is stored **in the slot table, not with the data**. Each slot has a `Mask` (allowed types) and a `Tag` (current variant). The byte buffer contains only raw data — no headers, no vtable pointers, no reference counts. A 4-byte integer occupies exactly 4 bytes.

This means:
- No per-value overhead. An `int32` is 4 bytes, not 16+ bytes like a Python int.
- No pointer chasing. Values are contiguous in the byte buffer.
- Type checking happens at the slot level via bitmask, not the value level.
- Union types (`int|bool`) cost no extra runtime overhead — just a wider mask and `max(size)` allocation.

### 5. Pointer Aliases for Manual Memory Layout

Most languages don't let you directly control where variables sit in memory relative to each other. C gives you `union` and pointer arithmetic but no bounds checking. Rust has `unsafe` blocks for raw pointer manipulation.

Vega's pointer aliases (`*type(offset)`) provide deliberate overlapping views with mandatory bounds checking:

```
alloc 8 {
    x = 42           # int at offset 0
    y = *short(0)    # view the first 2 bytes as a short
    z = *int(0)      # view the same 4 bytes as an int (same as x)
}
```

This is similar to C's `union` but more flexible — aliases can point anywhere in the buffer, not just to the start of a struct. And unlike C, out-of-bounds pointers are caught at runtime.

### 6. No Garbage Collector

Vega has zero GC overhead for named variables. Memory is:
- **Allocated** by the free list when a variable is first assigned.
- **Freed** explicitly by `free()` or implicitly when the `alloc` block exits.
- **Reused** immediately — freed bytes are available for the next allocation.

The expression stack still uses Go's `[]Value` slice (and therefore Go's GC), but this is for transient temporaries only. Named variables — the persistent state of the program — are entirely GC-free.

---

## Comparison with WebAssembly

Vega's memory model is closest to WebAssembly's linear memory:

| Aspect | Vega | WebAssembly |
|--------|------|-------------|
| Memory region | `alloc N { ... }` creates N bytes | Module declares initial/max memory pages |
| Variable access | Slot ID → offset lookup → byte read | Local index → typed value (separate from linear memory) |
| Deallocation | `free()` returns bytes to free list | No built-in allocator; must implement in user code |
| Growth | Fixed at block entry | `memory.grow` adds pages |
| Safety | Use-after-free detection | Bounds-checked memory access |

The key difference: WebAssembly separates locals (typed, stack-allocated by the VM) from linear memory (raw bytes, manually managed). Vega merges them — named variables **are** the raw bytes.

---

## Comparison with C's Stack

C functions allocate local variables on the call stack with known offsets determined at compile time:

```c
void example() {
    int x = 42;      // [rbp-4]
    double y = 3.14;  // [rbp-12]
}
```

Vega's `alloc` block is similar — variables get byte offsets determined by the allocator. But Vega adds:
- **Dynamic allocation** within the block (variables can be created at any point, not just at function entry).
- **Explicit deallocation** via `free()` (C stack variables live until the function returns).
- **Reuse** of freed space (C cannot reclaim stack space mid-function).

---

## Why This Design

Vega is a VFS orchestration DSL, not a general-purpose language. Scripts are:
- **Short-lived** — run once, exit.
- **Small** — tens of lines, not thousands.
- **Data movers** — they read, transform, and write data through VFS, not build data structures.

The byte-array model enforces these constraints:
- Fixed budgets prevent runaway memory growth.
- No GC means predictable performance (no pauses, no overhead).
- Explicit `free()` encourages scripts to release resources promptly.
- Type-tag encoding is compact — a VFS script that moves a few integers and booleans through a pipeline doesn't need the overhead of a full object system.

This is the wrong model for building a web framework. It's the right model for `read file → transform bytes → write file`.
