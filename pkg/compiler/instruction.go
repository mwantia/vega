package compiler

import "fmt"

type Instruction struct {
	Operation  OperationCode
	Argument   int    // Numeric argument (index, count or offset)
	Offset     int    // Field byte offset (for OpFieldLOAD/OpFieldSTORE)
	Name       string // String argument (variable name or function name)
	Extra      byte   // Extra byte (carries type tag for OpVarALLOC)
	SourceLine int    // Source line number for error reporting
}

func (i *Instruction) String() string {
	switch i.Operation {
	case OpLoadCONST:
		return fmt.Sprintf("%s index=%d", i.Operation, i.Argument)
	case OpVarALLOC:
		return fmt.Sprintf("%s slot=%d mask=%08b", i.Operation, i.Argument, i.Extra)
	case OpVarPTR:
		return fmt.Sprintf("%s slot=%d tag=%d", i.Operation, i.Argument, i.Extra)
	case OpVarSTORE, OpVarLOAD, OpVarFREE:
		return fmt.Sprintf("%s slot=%d", i.Operation, i.Argument)
	case OpStencilALLOC:
		return fmt.Sprintf("%s slot=%d size=%d", i.Operation, i.Argument, i.Offset)
	case OpFieldSTORE, OpFieldLOAD:
		return fmt.Sprintf("%s slot=%d offset=%d tag=%d", i.Operation, i.Argument, i.Offset, i.Extra)
	case OpCallNAT:
		return fmt.Sprintf("%s %s argc=%d", i.Operation, i.Name, i.Argument)
	case OpCallFN:
		return fmt.Sprintf("%s %s argc=%d", i.Operation, i.Name, i.Argument)
	case OpReturn:
		if i.Extra == 1 {
			return fmt.Sprintf("%s value", i.Operation)
		}
		return fmt.Sprintf("%s void", i.Operation)
	case OpLoadArg:
		return fmt.Sprintf("%s index=%d", i.Operation, i.Argument)
	case OpVarLoadRaw:
		return fmt.Sprintf("%s slot=%d size=%d", i.Operation, i.Argument, i.Offset)
	case OpLoadArgStencil:
		return fmt.Sprintf("%s index=%d slot=%d size=%d", i.Operation, i.Argument, i.Extra, i.Offset)
	case OpStrSTORE:
		return fmt.Sprintf("%s slot=%d", i.Operation, i.Argument)
	}

	return i.Operation.String()
}
