# Peephole Optimization & CFG + Dataflow
### Grounded in Vega's actual instruction set

---

## Part 1 — Peephole Optimization

### What it is

A peephole optimizer takes the flat `[]Instruction` slice that the compiler emits and makes a single pass (sometimes multiple passes until stable) over it, looking at a small window of consecutive instructions — typically 2 to 4 — and replacing known wasteful patterns with shorter or cheaper equivalents.

It requires no knowledge of the program's control flow. It has no understanding of what came before the window or what comes after. That's both its strength (simple to implement) and its ceiling (it can't eliminate redundancy across basic blocks).

The canonical structure is a **rewrite rule**:

```
pattern  →  replacement  (condition)
```

You match the left side against the instruction window. If it matches and the optional condition holds, you replace the window with the right side. Then you slide the window forward (or sometimes backward one step to allow chaining) and repeat.

---

### Where to insert the pass

```
Compiler.Compile(AST) → ByteCode
                              ↓
                    Peephole.Optimize(ByteCode) → ByteCode   ← insert here
                              ↓
                    VM.Run(ByteCode)
```

It operates on a `*ByteCode` and returns a new (or mutated) `*ByteCode`. It only needs to see `Instructions []Instruction` and `Constants []Constant`. It must not touch `Names[]` or `Functions` — those are handled separately (each `FunctionDefinition.ByteCode` should be peepholed independently, same pass).

---

### Concrete Vega-specific rules

**Rule 1 — Constant-fold pure arithmetic at compile time**

The most impactful peephole rule. When two `LOAD_CONST` instructions are followed by a binary arithmetic opcode, and both constants are numeric, compute the result immediately and replace the three instructions with a single `LOAD_CONST` for the result.

```
Before (3 instructions):
  LOAD_CONST index=A      (e.g. tag=int, data=0x0A000000 → 10)
  LOAD_CONST index=B      (e.g. tag=int, data=0x05000000 → 5)
  ADD

After (1 instruction):
  LOAD_CONST index=C      (new constant: tag=int, data=0x0F000000 → 15)
```

Condition: both constants have numeric tags (`TagShort`, `TagInteger`, `TagLong`, `TagFloat`, `TagDecimal`). The result tag follows normal promotion rules (int+int=int, int+decimal=decimal, etc.).

This applies to all pure binary ops: ADD, SUB, MUL, DIV, MOD, and also comparison ops (EQ, NEQ, LT, GT, LE, GE) where the result becomes a `TagBoolean` constant.

**Rule 2 — Eliminate LOAD_CONST / VAR_STORE into a combined init**

This is the most Vega-specific rule. The naïve emission pattern for `x = 42` is:

```
Before (3 instructions):
  VAR_ALLOC  slot=0 mask=00000010
  LOAD_CONST index=0
  VAR_STORE  slot=0
```

This pushes the constant onto the expression stack purely so `VAR_STORE` can pop it. The constant value is already known at compile time — it's sitting in `Constants[0]`. You can introduce a new opcode `VAR_INIT` that folds all three into one:

```
After (1 instruction):
  VAR_INIT slot=0 mask=00000010 const=0
```

`VAR_INIT` semantics: allocate slot, read directly from `Constants[const].Data`, write to allocator. No push, no pop. This bypasses the expression stack entirely for literal initialization.

Alternatively, if you don't want a new opcode, the peephole can simply reorder: it replaces the three instructions with two — `LOAD_CONST` first, `VAR_ALLOC+VAR_STORE` fused into a single instruction that consumes a constant index directly, not from the stack.

Condition: `VAR_ALLOC` followed immediately by `LOAD_CONST` for a literal constant, followed immediately by `VAR_STORE` to the same slot, with no other instruction in between and no branch target landing in the middle.

**Rule 3 — Dead store elimination (VAR_STORE immediately followed by VAR_FREE)**

If a slot is stored to and then freed on the very next instruction with no intervening use:

```
Before:
  VAR_STORE  slot=3
  VAR_FREE   slot=3

After:
  STACK_POP        (discard the value that would have been stored)
```

The store was pointless — the slot is freed before anyone reads it. The value was on the expression stack, so we just discard it.

Condition: no instruction between the STORE and FREE references slot 3, and nothing jumps into the gap. (The second condition requires knowing jump targets — see the CFG section for how to track this.)

**Rule 4 — Redundant VAR_LOAD / VAR_STORE to same slot**

```
Before:
  VAR_LOAD  slot=2
  VAR_STORE slot=2

After:
  (eliminated entirely — STACK_POP if the load result would remain)
```

Reading a slot and immediately writing the same value back is a no-op. This can appear when the compiler naïvely emits intermediate steps for augmented assignments.

**Rule 5 — STACK_POP after a no-result instruction**

Some instructions push nothing but the emission code defensively appends `STACK_POP` after every expression statement:

```
Before:
  VAR_FREE  slot=4
  STACK_POP

After:
  VAR_FREE  slot=4
```

`VAR_FREE` doesn't push anything. Popping after it is an error (would underflow) or is dead code from a defensive emit pattern. The peephole verifies the instruction is non-pushing and removes the `STACK_POP`.

**Rule 6 — Double negation / double NOT**

```
Before:
  NOT
  NOT

After:
  (eliminated)
```

Two consecutive boolean inversions cancel. Applies equally to any pure involution (NEG NEG → eliminated).

**Rule 7 — Jump to next instruction**

```
Before:
  JMP → addr=5
  [instruction at address 5]

After:
  [instruction at address 5]
```

A forward jump that lands on the immediately following instruction does nothing. Generated by some `if`/`else` patterns when the else branch is empty. Remove the jump. This requires tracking absolute addresses — after removing the instruction you must rewrite all jump targets that pointed past it.

---

### The address-rewriting problem

Every time you delete or insert instructions, the absolute addresses stored in jump targets become stale. This is the main implementation complexity of peephole work.

The standard approach: after applying all rewrite rules, build a **relocation table** — a map from old instruction index to new instruction index — and rewrite all jump `Argument` fields through it in a single final pass.

```go
// After collecting all rewrites:
newAddr := make([]int, len(old))   // old index → new index
cursor := 0
for i, keep := range keepFlags {
    newAddr[i] = cursor
    if keep { cursor++ }
}
// Then patch jumps:
for i := range newInstructions {
    if isJump(newInstructions[i].Operation) {
        oldTarget := newInstructions[i].Argument
        newInstructions[i].Argument = newAddr[oldTarget]
    }
}
```

---

### Running multiple passes

A single pass may not converge. Rule 1 (constant folding) may create a new `LOAD_CONST / LOAD_CONST / OP` pattern that Rule 1 can fold again. The standard practice: run until no rules fire, or cap at a fixed number of iterations (CPython caps at 10). In practice, 2–3 passes reach fixpoint for typical code.

---

## Part 2 — Control Flow Graph (CFG) Construction

### What it is and why you need it

A peephole optimizer can only see a small window of linear instructions. It cannot answer questions like:

- Is this variable's value used on *any* path after this point?
- Is this entire block of code unreachable?
- If I eliminate this store, will any later instruction still expect the value?

To answer those questions you need to model the program's control flow explicitly as a graph. The CFG is that model.

**A basic block** is a maximal sequence of instructions with the following properties:
- **Single entry**: control can only enter at the first instruction.
- **Single exit**: control leaves only at the last instruction.
- No internal jumps in or out.

This means: every jump target starts a new block. Every instruction that transfers control (jump, conditional jump, return, call that can raise) ends a block.

**The CFG** is a directed graph where nodes are basic blocks and edges represent possible control flow transitions (block A → block B means execution can reach B's entry from A's exit).

---

### Step 1 — Identify block boundaries (leaders)

A **leader** is any instruction that begins a basic block. An instruction is a leader if:

1. It is the first instruction of the function.
2. It is the target of any jump instruction.
3. It is the instruction immediately following a jump or return.

```
Instruction stream (with hypothetical jump opcodes):
  0: VAR_ALLOC slot=0 mask=00000010        ← leader (rule 1: first instruction)
  1: LOAD_CONST index=0
  2: VAR_STORE slot=0
  3: VAR_LOAD slot=0
  4: LOAD_CONST index=1
  5: CMP_LT                                 (comparison, pushes bool)
  6: JMP_IF_FALSE → 10                      (conditional jump ends block)
  7: LOAD_CONST index=2                    ← leader (rule 3: follows a jump)
  8: VAR_STORE slot=0
  9: JMP → 12                               (unconditional jump ends block)
 10: LOAD_CONST index=3                    ← leader (rule 2: target of jump at 6)
 11: VAR_STORE slot=0
 12: CALL_NAT print argc=1                 ← leader (rule 2: target of jump at 9)
 13: STACK_POP
 14: RETURN void
```

Leaders: {0, 7, 10, 12}.

---

### Step 2 — Build blocks

Each block spans from its leader to the instruction immediately before the next leader.

```
Block 0  [0..6]:  VAR_ALLOC, LOAD_CONST, VAR_STORE, VAR_LOAD, LOAD_CONST, CMP_LT, JMP_IF_FALSE→10
Block 1  [7..9]:  LOAD_CONST, VAR_STORE, JMP→12
Block 2  [10..11]: LOAD_CONST, VAR_STORE
Block 3  [12..14]: CALL_NAT, STACK_POP, RETURN
```

---

### Step 3 — Add edges

For each block, examine its last instruction:

- Unconditional jump (`JMP → target`): add edge to the block containing `target`.
- Conditional jump (`JMP_IF_FALSE → target`): add edge to the block containing `target` **and** to the immediately following block (fall-through).
- `RETURN`, `VAR_FREE` that terminates: no successors.
- Any other instruction: fall-through edge to the next block.

```
Block 0 → {Block 1 (fall-through), Block 2 (JMP_IF_FALSE target)}
Block 1 → {Block 3 (JMP target)}
Block 2 → {Block 3 (fall-through)}
Block 3 → {}  (RETURN)
```

Drawn as a graph:

```
        ┌─────────┐
        │ Block 0 │
        │ (cmp x) │
        └────┬────┘
      true   │  false
      ┌──────┘  └──────┐
      ↓                ↓
 ┌─────────┐      ┌─────────┐
 │ Block 1 │      │ Block 2 │
 │ x = 0   │      │ x = 1   │
 └────┬────┘      └────┬────┘
      └────────┬────────┘
               ↓
          ┌─────────┐
          │ Block 3 │
          │ print   │
          └─────────┘
```

---

### The Go structures

```go
type BasicBlock struct {
    ID           int
    Instructions []Instruction  // the slice of instructions in this block
    StartAddr    int            // index of first instruction in original stream
    Succs        []*BasicBlock  // successor blocks (control flow out)
    Preds        []*BasicBlock  // predecessor blocks (control flow in)
}

type CFG struct {
    Blocks []*BasicBlock
    Entry  *BasicBlock
    Exit   *BasicBlock   // synthetic exit node (all RETURN blocks point here)
}

func BuildCFG(code *ByteCode) *CFG { ... }
```

When you're done with all passes and want to re-emit flat bytecode:

```go
func (cfg *CFG) Flatten() *ByteCode { ... }
```

`Flatten` linearises the blocks in some valid topological order, recomputes instruction addresses, and patches all jump targets.

---

### One important Vega-specific note on jumps

Looking at the current `pkg/compiler` instruction set, explicit jump opcodes (`JMP`, `JMP_IF_TRUE`, `JMP_IF_FALSE`) are not yet listed in the documented opcode table — they're present in `old/pkg` but haven't been added to the new instruction set yet (control flow is parsed but not compiled). When you add them, every jump opcode needs to be classified as either:

- **Unconditional**: `JMP` — exactly one successor (the target)
- **Conditional**: `JMP_IF_FALSE`, `JMP_IF_TRUE` — exactly two successors (target + fall-through)
- **Multi-way** (future): `JUMP_TABLE` for match/switch — N successors

The `LoopStack` in `ByteCode` already records break/continue patch addresses — those become the back-edges and break-exit edges in the CFG when you build it.

---

## Part 3 — Dataflow Analysis on the CFG

### What it is

Dataflow analysis propagates facts about program state through the CFG edges. Each analysis defines:

- What **facts** are tracked (e.g. "slot 2 definitely holds constant value 42")
- A **transfer function** for each block (how does execution of the block change the facts?)
- A **meet operator** for merging facts from multiple predecessors
- A **direction** (forward: facts flow entry→exit; backward: facts flow exit→entry)

The algorithm then iterates over all blocks until no facts change (fixpoint). For most analyses on small programs (scripts, not compilers), fixpoint is reached in 2–5 iterations.

---

### Analysis 1 — Reaching Definitions (forward)

**What it tracks**: for each point in the program, which slot-store instructions might have been the last write to each slot?

**Why it matters**: if only one definition reaches a use, and that definition is a constant, you can replace the load with a constant push (constant propagation). If no definition reaches a use, the slot is used uninitialised (a bug to report at compile time).

**Facts**: `in[B]` and `out[B]` are sets of (slot, instruction_address) pairs — "slot S was last written at address A and that write reaches this point."

**Transfer function** for block B:
```
out[B] = GEN[B] ∪ (in[B] − KILL[B])
```

- `GEN[B]`: the set of definitions produced within B (the last write to each slot in B)
- `KILL[B]`: all definitions from elsewhere that write to slots written in B (they're overwritten)

**Meet operator** (at block entry, merging from all predecessors):
```
in[B] = ∪ out[P]   for all predecessors P of B
```

Using union because a definition "reaches" B if it reaches B along *any* path.

**Example with Vega slots**:

```
Block 0:
  VAR_ALLOC slot=0
  LOAD_CONST 10
  VAR_STORE  slot=0        ← def(slot=0, addr=2)
  LOAD_CONST 5
  VAR_STORE  slot=1        ← def(slot=1, addr=4)
  JMP_IF_FALSE → Block 2

GEN[Block 0] = {(slot=0, addr=2), (slot=1, addr=4)}
KILL[Block 0] = {}  (no prior defs killed)
```

After fixpoint, if `in[use_site]` for slot=0 contains exactly one definition `(slot=0, addr=2)` and that definition is a `LOAD_CONST`, the slot's value is a constant — eligible for constant propagation.

---

### Analysis 2 — Live Variable Analysis (backward)

**What it tracks**: at each point, which slots *will be read* on some future execution path?

**Why it matters**: if a slot is not live after a `VAR_STORE`, the store is dead — nothing will ever read the written value. You can eliminate both the store and the `VAR_ALLOC + VAR_FREE` that surround it.

This is the analysis that makes dead store elimination rigorous. The peephole Rule 3 earlier was approximate (it only caught immediate STORE+FREE pairs). Liveness catches dead stores with arbitrarily many instructions in between.

**Direction: backward** — you propagate facts from block exits toward entries, because liveness is about future use.

**Facts**: `in[B]` and `out[B]` are sets of slot IDs that are live entering/exiting B.

**Transfer function**:
```
in[B] = USE[B] ∪ (out[B] − DEF[B])
```

- `USE[B]`: slots read in B before being written (upward-exposed uses)
- `DEF[B]`: slots written in B (they kill liveness coming from above)

**Meet operator** (backward: at block exit, merging successors' entries):
```
out[B] = ∪ in[S]   for all successors S of B
```

**Example**:

```
Block 0:
  VAR_STORE slot=0         ← writes slot=0
  VAR_STORE slot=1         ← writes slot=1
  JMP_IF_FALSE → Block 2

Block 1 (then-branch):
  VAR_LOAD  slot=0         ← reads slot=0
  CALL_NAT print argc=1

Block 2 (else-branch):
  VAR_LOAD  slot=1         ← reads slot=1
  CALL_NAT print argc=1
```

Working backward:
- `out[Block 0] = in[Block 1] ∪ in[Block 2] = {slot=0} ∪ {slot=1} = {slot=0, slot=1}`
- Both stores in Block 0 are live → neither is dead.

Now consider adding a write to slot=0 that is never read:

```
Block 0:
  VAR_STORE slot=0         ← writes slot=0 (but Block 1 and 2 never use it)
  VAR_STORE slot=2
  JMP_IF_FALSE → Block 2
```

If neither Block 1 nor Block 2 reads slot=0:
- `out[Block 0]` does not contain slot=0
- `in[Block 0]` does not contain slot=0 (it's not in USE[Block 0] either)
- Therefore the `VAR_STORE slot=0` is dead → can be eliminated along with its `VAR_ALLOC` and any `LOAD_CONST` that fed into it.

---

### Analysis 3 — Constant Propagation (forward)

Builds on reaching definitions. Instead of tracking just *which* definition reaches a use, it tracks *what constant value* the slot holds, if known.

**Facts**: a map `{slot → ConstantValue | ⊤ | ⊥}` where:
- `⊤` (top) = unknown (multiple conflicting definitions reach this point)
- `⊥` (bottom) = never written (uninitialised, not yet relevant)
- `ConstantValue` = a specific constant, meaning *every path* that reaches here left this value in the slot

**Meet operator**: if two predecessors both know slot=0 is `42`, it's still `42`. If one knows `42` and another knows `17`, the result is `⊤` (unknown — must load from allocator at runtime).

```
meet(42, 42) = 42
meet(42, 17) = ⊤
meet(42,  ⊤) = ⊤
meet( ⊥, 42) = 42
```

**Transfer function** for a `VAR_STORE slot=S` in block B:
- If the value on the expression stack was just pushed by `LOAD_CONST index=K`, record `slot=S → Constants[K]` in the outgoing fact map.
- Otherwise record `slot=S → ⊤`.

**Transfer function** for a `VAR_LOAD slot=S`:
- If the current fact for slot=S is a known constant, **replace** this `VAR_LOAD` with `LOAD_CONST index=K` (adding K to the constant pool if needed).
- If it's `⊤`, leave the load as-is.

This is the clean dataflow formulation of constant propagation. The peephole version only handles the trivial case where the constant is assigned and used in the same window. Dataflow handles it across any number of instructions and blocks.

---

### Analysis 4 — Dead Code Elimination

A basic block is **unreachable** if it has no predecessors in the CFG (except the entry block). It can be removed entirely.

This is the simplest dataflow-style analysis — it's just a graph reachability traversal:

```go
func (cfg *CFG) RemoveUnreachable() {
    reachable := make(map[int]bool)
    var visit func(*BasicBlock)
    visit = func(b *BasicBlock) {
        if reachable[b.ID] { return }
        reachable[b.ID] = true
        for _, s := range b.Succs { visit(s) }
    }
    visit(cfg.Entry)
    cfg.Blocks = filter(cfg.Blocks, func(b *BasicBlock) bool {
        return reachable[b.ID]
    })
}
```

Common source of unreachable blocks in Vega: code after a `RETURN`, or the else-branch of `if true { ... }` after constant folding resolves the condition.

---

## Part 4 — Putting It All Together: the Optimization Pipeline

The full optimization sequence, ordered by dependency:

```
ByteCode (raw, from compiler)
        │
        ▼
1. Peephole pass (constant folding, dead stores, jump-to-next)
        │     ↑
        │     └── loop until stable (usually 2–3 iterations)
        ▼
2. CFG construction (BuildCFG)
        │
        ▼
3. Unreachable block elimination
        │
        ▼
4. Reaching definitions analysis → constant propagation
        │
        ▼
5. Liveness analysis → dead store elimination
        │
        ▼
6. Peephole pass again (cleans up what dataflow exposed)
        │
        ▼
7. CFG.Flatten() → ByteCode (optimized, ready for VM or serialization)
```

Stages 3–5 require the CFG. Stages 1 and 6 don't — they operate on the flat instruction stream. Running peephole at the start reduces the size of the CFG; running it again at the end cleans up any new patterns exposed by the dataflow passes.

For Vega's scope (scripts over VFS, not performance-critical inner loops), **stages 1 and 3 alone give the most practical benefit** for the least implementation work. Stage 4 (constant propagation) is the next meaningful step. Stage 5 (full liveness-based dead store elimination) is most valuable for generated code patterns, not hand-written scripts.

---

## Summary: What Vega Can Implement Incrementally

| Stage | Complexity | Requires CFG | Biggest win for Vega |
|-------|-----------|-------------|----------------------|
| Peephole: constant folding | Low | No | Eliminates pointless arithmetic in slot sizing |
| Peephole: dead jump elimination | Low | No (but needs jump targets map) | Cleans up if/else emission |
| Peephole: VAR_ALLOC+LOAD_CONST+VAR_STORE fusion | Low | No | Reduces instruction count for every literal assignment |
| CFG construction | Medium | — | Prerequisite for everything below |
| Unreachable block elimination | Low (once CFG exists) | Yes | Removes dead branches after constant fold |
| Constant propagation | Medium | Yes | Eliminates VAR_LOAD for loop-invariant slots |
| Liveness / dead store elimination | Medium-High | Yes | Removes temporary slots that never escape |