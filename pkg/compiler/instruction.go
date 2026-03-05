package compiler

import "fmt"

type Instruction struct {
	Operation  OperationCode
	Argument   int  // Numeric argument (index, count or offset)
	Offset     int  // Field byte offset (for OpFieldLOAD/OpFieldSTORE); name index for OpCall/OpCallMethod; capacity for OpSliceALLOC
	Extra      byte // Extra byte (type tag for OpVarALLOC/OpFieldSTORE/OpFieldLOAD; 0=stmt/1=expr for OpCall)
	Size       int  // Field byte size for OpFieldSTORE/OpFieldLOAD (needed for TagSlice fields whose SizeForTag returns 0)
	SourceLine int  // Source line number for error reporting
}

func (i *Instruction) String() string {
	switch i.Operation {
	case OpLoadCONST:
		return fmt.Sprintf("%s index=%d", i.Operation, i.Argument)
	case OpVarALLOC:
		return fmt.Sprintf("%s slot=%d mask=%08b", i.Operation, i.Argument, i.Extra)
	case OpVarINIT:
		return fmt.Sprintf("%s slot=%d mask=%08b const=%d", i.Operation, i.Argument, i.Extra, i.Offset)
	case OpVarPTR:
		return fmt.Sprintf("%s slot=%d tag=%d", i.Operation, i.Argument, i.Extra)
	case OpVarSTORE, OpVarLOAD, OpVarFREE:
		return fmt.Sprintf("%s slot=%d", i.Operation, i.Argument)
	case OpStencilALLOC:
		return fmt.Sprintf("%s slot=%d size=%d", i.Operation, i.Argument, i.Offset)
	case OpFieldSTORE, OpFieldLOAD:
		return fmt.Sprintf("%s slot=%d offset=%d tag=%d size=%d", i.Operation, i.Argument, i.Offset, i.Extra, i.Size)
	case OpSliceALLOC:
		return fmt.Sprintf("%s slot=%d capacity=%d", i.Operation, i.Argument, i.Offset)
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
	case OpBuildSTRING:
		return fmt.Sprintf("%s count=%d", i.Operation, i.Argument)
	}

	return i.Operation.String()
}
