package compiler

type OperationCode byte

const (
	OpStackPOP OperationCode = iota

	OpLoadCONST

	OpVarALLOC // allocate slot in byte buffer (arg: slot ID, extra: type mask)
	OpVarSTORE // pop expr stack, copy bytes into slot (arg: slot ID)
	OpVarLOAD  // copy bytes from slot, push to expr stack (arg: slot ID)
	OpVarFREE  // return slot memory to free list (arg: slot ID)
	OpVarPTR   // create alias slot at explicit offset (arg: slot ID, extra: type tag)

	OpStencilALLOC // allocate stencil-sized slot (arg: slot ID, offset: total size)
	OpSliceALLOC   // allocate a bounded slice slot (arg: slot ID, offset: capacity in bytes)
	OpFieldSTORE   // pop expr stack, copy into field (arg: slot ID, offset: field byte offset, extra: type tag, size: field size)
	OpFieldLOAD    // load field from slot (arg: slot ID, offset: field byte offset, extra: type tag, size: field size)

	OpCallNAT // call a registered native (Go) function (name: function name, arg: argument count)
	OpCallFN  // call a user-defined (Vega) function (name: function name, arg: argument count)
	OpReturn  // return from user function (extra: 0=void, 1=has return value)
	OpLoadArg // push a pending primitive argument onto the expr stack (arg: argument index)

	OpVarLoadRaw     // push stencil slot bytes as a RawValue (arg: slot ID, offset: total size)
	OpLoadArgStencil // restore a stencil arg from pending args into a new slot (arg: pending index, extra: slot ID, offset: total size)

	OpPtrLOAD // pop offset from expr stack, read tag-typed value from allocator, push result (extra: type tag)

	OpBuildSTRING // pop N values, call String() on each, concat, push heap SliceValue (arg: N)
)

var operationNames = map[OperationCode]string{
	OpStackPOP:  "STACK_POP",
	OpLoadCONST: "LOAD_CONST",

	OpVarALLOC: "VAR_ALLOC",
	OpVarSTORE: "VAR_STORE",
	OpVarLOAD:  "VAR_LOAD",
	OpVarFREE:  "VAR_FREE",
	OpVarPTR:   "VAR_PTR",

	OpStencilALLOC: "STENCIL_ALLOC",
	OpSliceALLOC:   "SLICE_ALLOC",
	OpFieldSTORE:   "FIELD_STORE",
	OpFieldLOAD:    "FIELD_LOAD",

	OpCallNAT: "CALL_NAT",
	OpCallFN:  "CALL_FN",
	OpReturn:  "RETURN",
	OpLoadArg: "LOAD_ARG",

	OpVarLoadRaw:     "VAR_LOAD_RAW",
	OpLoadArgStencil: "LOAD_ARG_STENCIL",

	OpPtrLOAD: "PTR_LOAD",

	OpBuildSTRING: "BUILD_STRING",
}

func (op OperationCode) String() string {
	if name, ok := operationNames[op]; ok {
		return name
	}
	return "UNKNOWN"
}
