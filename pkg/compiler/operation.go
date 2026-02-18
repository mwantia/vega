package compiler

type OperationCode byte

const (
	OpStackPOP OperationCode = iota

	OpStackALLOC
	OpStackFREE
	OpLoadCONST

	OpVarALLOC // allocate slot in byte buffer (arg: slot ID, extra: type mask)
	OpVarSTORE // pop expr stack, copy bytes into slot (arg: slot ID)
	OpVarLOAD  // copy bytes from slot, push to expr stack (arg: slot ID)
	OpVarFREE  // return slot memory to free list (arg: slot ID)
	OpVarPTR   // create alias slot at explicit offset (arg: slot ID, extra: type tag)

	OpStencilALLOC // allocate stencil-sized slot (arg: slot ID, offset: total size)
	OpFieldSTORE   // pop expr stack, copy into field (arg: slot ID, offset: field byte offset, extra: type tag)
	OpFieldLOAD    // load field from slot (arg: slot ID, offset: field byte offset, extra: type tag)

	OpCallNAT // call a registered native (Go) function (name: function name, arg: argument count)
)

var operationNames = map[OperationCode]string{
	OpStackPOP:   "STACK_POP",
	OpStackALLOC: "STACK_ALLOC",
	OpStackFREE:  "STACK_FREE",
	OpLoadCONST:  "LOAD_CONST",

	OpVarALLOC: "VAR_ALLOC",
	OpVarSTORE: "VAR_STORE",
	OpVarLOAD:  "VAR_LOAD",
	OpVarFREE:  "VAR_FREE",
	OpVarPTR:   "VAR_PTR",

	OpStencilALLOC: "STENCIL_ALLOC",
	OpFieldSTORE:   "FIELD_STORE",
	OpFieldLOAD:    "FIELD_LOAD",

	OpCallNAT: "CALL_NAT",
}

func (op OperationCode) String() string {
	if name, ok := operationNames[op]; ok {
		return name
	}
	return "UNKNOWN"
}
