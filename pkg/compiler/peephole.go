package compiler

const peepholeMaxPasses = 10

// PeepholeRule defines a single rewrite rule for the peephole optimizer.
// Rules examine a fixed-size window of consecutive instructions and may
// replace it with a shorter (or absent) sequence.
type PeepholeRule interface {
	// Name returns a human-readable identifier used in diagnostics.
	Name() string

	// WindowSize is the number of consecutive instructions this rule examines.
	WindowSize() int

	// Rewrite attempts to apply the rule to window. On match it returns the
	// replacement instructions and true. On no match it returns nil, false.
	// The replacement may be empty (instructions eliminated) or shorter than
	// the window (instructions folded).
	Rewrite(window []Instruction, code *ByteCode) ([]Instruction, bool)
}

// Peephole applies a set of PeepholeRules to a flat instruction stream.
// Rules are tried in registration order; the first match wins per window position.
// Multiple passes are run until no rule fires or peepholeMaxPasses is reached.
type Peephole struct {
	rules []PeepholeRule
}

// NewPeephole returns an empty optimizer. Use Register to add rules.
func NewPeephole() *Peephole { return &Peephole{} }

// Register appends a rule. Rules are evaluated in registration order.
func (p *Peephole) Register(r PeepholeRule) {
	p.rules = append(p.rules, r)
}

// DefaultPeephole returns a Peephole with all standard rules registered.
func DefaultPeephole() *Peephole {
	ph := NewPeephole()
	ph.Register(RuleConstantFold{})
	ph.Register(RuleVarInit{})
	ph.Register(RuleDoubleNot{})
	ph.Register(RuleDoubleNeg{})
	ph.Register(RuleDeadStore{})
	ph.Register(RuleRedundantLoadStore{})
	ph.Register(RulePopAfterNonPush{})
	return ph
}

// Optimize returns a new *ByteCode whose instructions have been rewritten by
// all registered rules. The top-level instruction stream and each
// FunctionDefinition's ByteCode are optimized independently.
func (p *Peephole) Optimize(code *ByteCode) *ByteCode {
	out := &ByteCode{
		Instructions: p.runUntilStable(code, code.Instructions),
		Constants:    code.Constants,
		Names:        code.Names,
		LoopStack:    code.LoopStack,
		Functions:    make(map[string]*FunctionDefinition, len(code.Functions)),
	}
	for name, fn := range code.Functions {
		newFn := *fn
		newFn.ByteCode = &ByteCode{
			Instructions: p.runUntilStable(fn.ByteCode, fn.ByteCode.Instructions),
			Constants:    fn.ByteCode.Constants,
			Names:        fn.ByteCode.Names,
			LoopStack:    fn.ByteCode.LoopStack,
			Functions:    fn.ByteCode.Functions,
		}
		out.Functions[name] = &newFn
	}
	return out
}

func (p *Peephole) runUntilStable(code *ByteCode, instrs []Instruction) []Instruction {
	cur := instrs
	for range peepholeMaxPasses {
		next, changed := p.runPass(code, cur)
		if !changed {
			break
		}
		cur = next
	}
	return cur
}

func (p *Peephole) runPass(code *ByteCode, instrs []Instruction) ([]Instruction, bool) {
	jumpTargets := collectJumpTargets(instrs)
	result := make([]Instruction, 0, len(instrs))
	changed := false

	i := 0
	for i < len(instrs) {
		matched := false
		for _, rule := range p.rules {
			win := rule.WindowSize()
			if i+win > len(instrs) {
				continue
			}
			// Don't rewrite across a jump-target boundary: an instruction at
			// position j>0 within the window might be a branch destination, so
			// removing or merging it would corrupt control flow.
			safeWindow := true
			for j := 1; j < win; j++ {
				if jumpTargets[i+j] {
					safeWindow = false
					break
				}
			}
			if !safeWindow {
				continue
			}
			if replacement, ok := rule.Rewrite(instrs[i:i+win], code); ok {
				result = append(result, replacement...)
				i += win
				changed = true
				matched = true
				break
			}
		}
		if !matched {
			result = append(result, instrs[i])
			i++
		}
	}

	// TODO: patch jump Argument fields through a relocation table once jump
	// opcodes (JMP, JMP_IF_TRUE, JMP_IF_FALSE) are added to the instruction set.
	// Currently there are no jumps, so no relocation is needed.

	return result, changed
}

// collectJumpTargets returns the set of instruction indices that are the
// targets of any jump instruction. Any such index inside a rule's window
// (at position > 0) blocks the rule from firing.
func collectJumpTargets(instrs []Instruction) map[int]bool {
	targets := make(map[int]bool)
	// TODO: when JMP / JMP_IF_TRUE / JMP_IF_FALSE opcodes are introduced,
	// iterate instrs here and mark targets[instr.Argument] = true for each jump.
	_ = instrs
	return targets
}
