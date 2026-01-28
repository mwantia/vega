// Package compiler implements the bytecode compiler for Vega.
package compiler

// OpCode represents a bytecode operation.
type OpCode byte

const (
	// Stack operations
	OpPop OpCode = iota
	OpDup

	// Constants and variables
	OpLoadConst
	OpLoadVar
	OpStoreVar
	OpLoadMember

	// Arithmetic and logic
	OpAdd
	OpSub
	OpMul
	OpDiv
	OpMod
	OpNeg
	OpNot

	// Comparison
	OpEq
	OpNotEq
	OpLt
	OpLte
	OpGt
	OpGte

	// Logical
	OpAnd
	OpOr

	// Jump operations
	OpJmp
	OpJmpIfFalse
	OpJmpIfTrue

	// Function operations
	OpCall
	OpCallMethod
	OpReturn

	// Collection operations
	OpBuildArray
	OpBuildMap
	OpIndex
	OpSetIndex

	// Iteration
	OpIterInit
	OpIterNext

	// String operations
	OpConcat

	// Pipe operations
	OpPipe
)

var opNames = map[OpCode]string{
	OpPop:        "POP",
	OpDup:        "DUP",
	OpLoadConst:  "LOAD_CONST",
	OpLoadVar:    "LOAD_VAR",
	OpStoreVar:   "STORE_VAR",
	OpLoadMember: "LOAD_MEMB",
	OpAdd:        "ADD",
	OpSub:        "SUB",
	OpMul:        "MUL",
	OpDiv:        "DIV",
	OpMod:        "MOD",
	OpNeg:        "NEG",
	OpNot:        "NOT",
	OpEq:         "EQ",
	OpNotEq:      "NOT_EQ",
	OpLt:         "LT",
	OpLte:        "LTE",
	OpGt:         "GT",
	OpGte:        "GTE",
	OpAnd:        "AND",
	OpOr:         "OR",
	OpJmp:        "JMP",
	OpJmpIfFalse: "JMP_IF_FALSE",
	OpJmpIfTrue:  "JMP_IF_TRUE",
	OpCall:       "CALL",
	OpCallMethod: "CALL_METHOD",
	OpReturn:     "RETURN",
	OpBuildArray: "BUILD_ARRAY",
	OpBuildMap:   "BUILD_MAP",
	OpIndex:      "INDEX",
	OpSetIndex:   "SET_INDEX",
	OpIterInit:   "ITER_INIT",
	OpIterNext:   "ITER_NEXT",
	OpConcat:     "CONCAT",
	OpPipe:       "PIPE",
}

func (op OpCode) String() string {
	if name, ok := opNames[op]; ok {
		return name
	}
	return "UNKNOWN"
}
