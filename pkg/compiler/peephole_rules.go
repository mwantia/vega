package compiler

import (
	"encoding/binary"
	"math"

	"github.com/mwantia/vega/pkg/slot"
)

// isNumericTag returns true for tags that support constant-folding arithmetic.
func isNumericTag(tag slot.TypeTag) bool {
	switch tag {
	case slot.TagByte, slot.TagShort, slot.TagInteger, slot.TagLong,
		slot.TagFloat, slot.TagDecimal:
		return true
	}
	return false
}

// isBinaryArithOp returns true if op is a pure binary arithmetic opcode.
func isBinaryArithOp(op OperationCode) bool {
	switch op {
	case OpBinAdd, OpBinSub, OpBinMul, OpBinDiv, OpBinMod:
		return true
	}
	return false
}

// isBinaryCmpOp returns true if op is a comparison opcode.
func isBinaryCmpOp(op OperationCode) bool {
	switch op {
	case OpCmpEQ, OpCmpNE, OpCmpLT, OpCmpLE, OpCmpGT, OpCmpGE:
		return true
	}
	return false
}

// isPushingInstruction returns false for opcodes that leave the expression
// stack unchanged or pop from it without pushing. Used by RulePopAfterNonPush.
func isPushingInstruction(op OperationCode) bool {
	switch op {
	case OpVarALLOC, OpVarSTORE, OpVarFREE, OpVarINIT,
		OpStencilALLOC, OpSliceALLOC,
		OpFieldSTORE,
		OpReturn:
		return false
	}
	return true
}

// foldArith computes the result of a binary arithmetic opcode on two
// same-typed constant data slices. Returns (resultTag, resultData, ok).
// Returns ok=false if the operation is not supported or would divide by zero.
func foldArith(op OperationCode, tag slot.TypeTag, aData, bData []byte) (slot.TypeTag, []byte, bool) {
	result := make([]byte, len(aData))

	switch tag {
	case slot.TagByte:
		a, b := aData[0], bData[0]
		switch op {
		case OpBinAdd:
			result[0] = a + b
		case OpBinSub:
			result[0] = a - b
		case OpBinMul:
			result[0] = a * b
		case OpBinDiv:
			if b == 0 {
				return 0, nil, false
			}
			result[0] = a / b
		case OpBinMod:
			if b == 0 {
				return 0, nil, false
			}
			result[0] = a % b
		default:
			return 0, nil, false
		}
		return tag, result, true

	case slot.TagShort:
		a := int16(binary.LittleEndian.Uint16(aData))
		b := int16(binary.LittleEndian.Uint16(bData))
		var v int16
		switch op {
		case OpBinAdd:
			v = a + b
		case OpBinSub:
			v = a - b
		case OpBinMul:
			v = a * b
		case OpBinDiv:
			if b == 0 {
				return 0, nil, false
			}
			v = a / b
		case OpBinMod:
			if b == 0 {
				return 0, nil, false
			}
			v = a % b
		default:
			return 0, nil, false
		}
		binary.LittleEndian.PutUint16(result, uint16(v))
		return tag, result, true

	case slot.TagInteger:
		a := int32(binary.LittleEndian.Uint32(aData))
		b := int32(binary.LittleEndian.Uint32(bData))
		var v int32
		switch op {
		case OpBinAdd:
			v = a + b
		case OpBinSub:
			v = a - b
		case OpBinMul:
			v = a * b
		case OpBinDiv:
			if b == 0 {
				return 0, nil, false
			}
			v = a / b
		case OpBinMod:
			if b == 0 {
				return 0, nil, false
			}
			v = a % b
		default:
			return 0, nil, false
		}
		binary.LittleEndian.PutUint32(result, uint32(v))
		return tag, result, true

	case slot.TagLong:
		a := int64(binary.LittleEndian.Uint64(aData))
		b := int64(binary.LittleEndian.Uint64(bData))
		var v int64
		switch op {
		case OpBinAdd:
			v = a + b
		case OpBinSub:
			v = a - b
		case OpBinMul:
			v = a * b
		case OpBinDiv:
			if b == 0 {
				return 0, nil, false
			}
			v = a / b
		case OpBinMod:
			if b == 0 {
				return 0, nil, false
			}
			v = a % b
		default:
			return 0, nil, false
		}
		binary.LittleEndian.PutUint64(result, uint64(v))
		return tag, result, true

	case slot.TagFloat:
		a := math.Float32frombits(binary.LittleEndian.Uint32(aData))
		b := math.Float32frombits(binary.LittleEndian.Uint32(bData))
		var v float32
		switch op {
		case OpBinAdd:
			v = a + b
		case OpBinSub:
			v = a - b
		case OpBinMul:
			v = a * b
		case OpBinDiv:
			v = a / b
		default:
			return 0, nil, false // no float mod
		}
		binary.LittleEndian.PutUint32(result, math.Float32bits(v))
		return tag, result, true

	case slot.TagDecimal:
		a := math.Float64frombits(binary.LittleEndian.Uint64(aData))
		b := math.Float64frombits(binary.LittleEndian.Uint64(bData))
		var v float64
		switch op {
		case OpBinAdd:
			v = a + b
		case OpBinSub:
			v = a - b
		case OpBinMul:
			v = a * b
		case OpBinDiv:
			v = a / b
		default:
			return 0, nil, false
		}
		binary.LittleEndian.PutUint64(result, math.Float64bits(v))
		return tag, result, true
	}

	return 0, nil, false
}

// foldCmp computes a comparison result and returns -1, 0, or 1.
// Returns ok=false for unsupported tags.
func foldCmp(tag slot.TypeTag, aData, bData []byte) (int, bool) {
	cmp := func(less, equal bool) int {
		if less {
			return -1
		}
		if equal {
			return 0
		}
		return 1
	}

	switch tag {
	case slot.TagByte:
		a, b := aData[0], bData[0]
		return cmp(a < b, a == b), true
	case slot.TagShort:
		a := int16(binary.LittleEndian.Uint16(aData))
		b := int16(binary.LittleEndian.Uint16(bData))
		return cmp(a < b, a == b), true
	case slot.TagInteger:
		a := int32(binary.LittleEndian.Uint32(aData))
		b := int32(binary.LittleEndian.Uint32(bData))
		return cmp(a < b, a == b), true
	case slot.TagLong:
		a := int64(binary.LittleEndian.Uint64(aData))
		b := int64(binary.LittleEndian.Uint64(bData))
		return cmp(a < b, a == b), true
	case slot.TagFloat:
		a := math.Float32frombits(binary.LittleEndian.Uint32(aData))
		b := math.Float32frombits(binary.LittleEndian.Uint32(bData))
		return cmp(a < b, a == b), true
	case slot.TagDecimal:
		a := math.Float64frombits(binary.LittleEndian.Uint64(aData))
		b := math.Float64frombits(binary.LittleEndian.Uint64(bData))
		return cmp(a < b, a == b), true
	case slot.TagBoolean:
		a := aData[0] != 0
		b := bData[0] != 0
		if a == b {
			return 0, true
		}
		if !a && b {
			return -1, true
		}
		return 1, true
	}
	return 0, false
}

// --- Rule 1: Constant-fold pure arithmetic and comparison ---
//
// LOAD_CONST A, LOAD_CONST B, <binop/cmpop>  →  LOAD_CONST <result>
//
// Both constants must be numeric. The result constant is added to the pool
// (AddConstant deduplicates automatically).

type RuleConstantFold struct{}

func (RuleConstantFold) Name() string       { return "constant-fold" }
func (RuleConstantFold) WindowSize() int    { return 3 }

func (RuleConstantFold) Rewrite(w []Instruction, code *ByteCode) ([]Instruction, bool) {
	if w[0].Operation != OpLoadCONST || w[1].Operation != OpLoadCONST {
		return nil, false
	}
	op := w[2].Operation
	if !isBinaryArithOp(op) && !isBinaryCmpOp(op) {
		return nil, false
	}

	ca := code.Constants[w[0].Argument]
	cb := code.Constants[w[1].Argument]

	if ca.Tag != cb.Tag {
		return nil, false
	}
	if !isNumericTag(ca.Tag) {
		return nil, false
	}

	var resultTag slot.TypeTag
	var resultData []byte

	if isBinaryArithOp(op) {
		var ok bool
		resultTag, resultData, ok = foldArith(op, ca.Tag, ca.Data, cb.Data)
		if !ok {
			return nil, false
		}
	} else {
		c, ok := foldCmp(ca.Tag, ca.Data, cb.Data)
		if !ok {
			return nil, false
		}
		resultTag = slot.TagBoolean
		var bv byte
		switch op {
		case OpCmpEQ:
			if c == 0 {
				bv = 1
			}
		case OpCmpNE:
			if c != 0 {
				bv = 1
			}
		case OpCmpLT:
			if c < 0 {
				bv = 1
			}
		case OpCmpLE:
			if c <= 0 {
				bv = 1
			}
		case OpCmpGT:
			if c > 0 {
				bv = 1
			}
		case OpCmpGE:
			if c >= 0 {
				bv = 1
			}
		}
		resultData = []byte{bv}
	}

	idx := code.AddConstant(Constant{Tag: resultTag, Data: resultData})
	return []Instruction{{
		Operation:  OpLoadCONST,
		Argument:   idx,
		SourceLine: w[0].SourceLine,
	}}, true
}

// --- Rule 2: VAR_INIT fusion ---
//
// LOAD_CONST index=K (scalar), VAR_ALLOC slot=S mask=M, VAR_STORE slot=S
//   →  VAR_INIT slot=S mask=M const=K
//
// The Vega compiler emits the RHS expression first, then VAR_ALLOC, then
// VAR_STORE. When the RHS is a single scalar LOAD_CONST we can fuse all
// three into VAR_INIT, eliminating the expression-stack round-trip.
// Only applies to scalar (non-slice) constants.

type RuleVarInit struct{}

func (RuleVarInit) Name() string    { return "var-init" }
func (RuleVarInit) WindowSize() int { return 3 }

func (RuleVarInit) Rewrite(w []Instruction, code *ByteCode) ([]Instruction, bool) {
	if w[0].Operation != OpLoadCONST {
		return nil, false
	}
	if w[1].Operation != OpVarALLOC {
		return nil, false
	}
	if w[2].Operation != OpVarSTORE {
		return nil, false
	}
	slotID := w[1].Argument
	if w[2].Argument != slotID {
		return nil, false
	}
	// Only fold scalar constants — slice variables use OpSliceALLOC, not OpVarALLOC.
	c := code.Constants[w[0].Argument]
	if c.Tag == slot.TagSlice {
		return nil, false
	}
	return []Instruction{{
		Operation:  OpVarINIT,
		Argument:   slotID,
		Extra:      w[1].Extra,    // type mask from VAR_ALLOC
		Offset:     w[0].Argument, // constant index from LOAD_CONST
		SourceLine: w[1].SourceLine,
	}}, true
}

// --- Rule 3: Dead store elimination ---
//
// VAR_STORE slot=S, VAR_FREE slot=S  →  STACK_POP
//
// The store is pointless if the slot is freed before anyone reads it.
// The value was on the expression stack, so we just discard it.

type RuleDeadStore struct{}

func (RuleDeadStore) Name() string    { return "dead-store" }
func (RuleDeadStore) WindowSize() int { return 2 }

func (RuleDeadStore) Rewrite(w []Instruction, code *ByteCode) ([]Instruction, bool) {
	if w[0].Operation != OpVarSTORE || w[1].Operation != OpVarFREE {
		return nil, false
	}
	if w[0].Argument != w[1].Argument {
		return nil, false
	}
	return []Instruction{{
		Operation:  OpStackPOP,
		SourceLine: w[0].SourceLine,
	}}, true
}

// --- Rule 4: Redundant VAR_LOAD / VAR_STORE to same slot ---
//
// VAR_LOAD slot=S, VAR_STORE slot=S  →  (eliminated)
//
// Loading a value from a slot and immediately writing the same value back
// is a net no-op: the stack depth is unchanged, the slot value is unchanged.

type RuleRedundantLoadStore struct{}

func (RuleRedundantLoadStore) Name() string    { return "redundant-load-store" }
func (RuleRedundantLoadStore) WindowSize() int { return 2 }

func (RuleRedundantLoadStore) Rewrite(w []Instruction, code *ByteCode) ([]Instruction, bool) {
	if w[0].Operation != OpVarLOAD || w[1].Operation != OpVarSTORE {
		return nil, false
	}
	if w[0].Argument != w[1].Argument {
		return nil, false
	}
	return []Instruction{}, true
}

// --- Rule 5: STACK_POP after a non-pushing instruction ---
//
// <non-push-instr>, STACK_POP  →  <non-push-instr>
//
// Instructions that don't push to the expression stack should never be
// followed by STACK_POP (it would underflow or silently discard an
// unrelated value). Remove the dead POP.

type RulePopAfterNonPush struct{}

func (RulePopAfterNonPush) Name() string    { return "pop-after-non-push" }
func (RulePopAfterNonPush) WindowSize() int { return 2 }

func (RulePopAfterNonPush) Rewrite(w []Instruction, code *ByteCode) ([]Instruction, bool) {
	if w[1].Operation != OpStackPOP {
		return nil, false
	}
	if isPushingInstruction(w[0].Operation) {
		return nil, false
	}
	return []Instruction{w[0]}, true
}

// --- Rule 6a: Double NOT elimination ---
//
// LOG_NOT, LOG_NOT  →  (eliminated)
//
// Two consecutive boolean inversions cancel each other out.

type RuleDoubleNot struct{}

func (RuleDoubleNot) Name() string    { return "double-not" }
func (RuleDoubleNot) WindowSize() int { return 2 }

func (RuleDoubleNot) Rewrite(w []Instruction, code *ByteCode) ([]Instruction, bool) {
	if w[0].Operation != OpLogNot || w[1].Operation != OpLogNot {
		return nil, false
	}
	return []Instruction{}, true
}

// --- Rule 6b: Double NEG elimination ---
//
// UN_NEG, UN_NEG  →  (eliminated)
//
// Two consecutive unary negations cancel each other out.

type RuleDoubleNeg struct{}

func (RuleDoubleNeg) Name() string    { return "double-neg" }
func (RuleDoubleNeg) WindowSize() int { return 2 }

func (RuleDoubleNeg) Rewrite(w []Instruction, code *ByteCode) ([]Instruction, bool) {
	if w[0].Operation != OpUnNeg || w[1].Operation != OpUnNeg {
		return nil, false
	}
	return []Instruction{}, true
}

// --- Rule 7: Jump-to-next-instruction elimination (stub) ---
//
// JMP → addr=N+1 (where N+1 is the immediately following instruction)  →  (eliminated)
//
// NOTE: This rule is not yet active because jump opcodes (JMP, JMP_IF_TRUE,
// JMP_IF_FALSE) have not been added to the instruction set. It is registered
// here as documentation and will become functional once jump opcodes land.
//
// When enabled, it also requires the relocation pass in runPass to be
// implemented so that removing an instruction shifts all downstream targets.
