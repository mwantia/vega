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

	OpCall   // call a function by name (name: function name, arg: argument count, extra: 0=statement 1=expression)
	OpReturn // return from user function (extra: 0=void, 1=has return value)
	OpLoadArg // push a pending primitive argument onto the expr stack (arg: argument index)

	OpVarLoadRaw     // push stencil slot bytes as a Raw (arg: slot ID, offset: total size)
	OpLoadArgStencil // restore a stencil arg from pending args into a new slot (arg: pending index, extra: slot ID, offset: total size)

	OpPtrLOAD // pop offset from expr stack, read tag-typed value from allocator, push result (extra: type tag)

	OpVarINIT // alloc slot + store constant directly, no stack interaction (arg: slot ID, extra: type mask, offset: const index)

	OpBuildSTRING // pop N values, call String() on each, concat, push heap SliceValue (arg: N)

	OpGetMember  // pop value, get named member via Memberable interface (offset: name index in Names[])
	OpCallMethod // pop argc args then pop object, call method via Methodable interface (offset: name index in Names[], arg: argc)

	// Arithmetic — operate directly on inline StackSlot bytes (zero heap alloc).
	OpBinAdd // pop b, pop a, push a+b (same tag)
	OpBinSub // pop b, pop a, push a-b
	OpBinMul // pop b, pop a, push a*b
	OpBinDiv // pop b, pop a, push a/b
	OpBinMod // pop b, pop a, push a%b (integer types only)
	OpUnNeg  // pop a, push -a

	// Comparison — result is always TagBoolean.
	OpCmpEQ // pop b, pop a, push (a==b)
	OpCmpNE // pop b, pop a, push (a!=b)
	OpCmpLT // pop b, pop a, push (a<b)
	OpCmpLE // pop b, pop a, push (a<=b)
	OpCmpGT // pop b, pop a, push (a>b)
	OpCmpGE // pop b, pop a, push (a>=b)

	// Logical — operands must be TagBoolean.
	OpLogAnd // pop b, pop a, push (a&&b)
	OpLogOr  // pop b, pop a, push (a||b)
	OpLogNot // pop a, push (!a)
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

	OpCall:   "CALL",
	OpReturn: "RETURN",
	OpLoadArg: "LOAD_ARG",

	OpVarLoadRaw:     "VAR_LOAD_RAW",
	OpLoadArgStencil: "LOAD_ARG_STENCIL",

	OpPtrLOAD: "PTR_LOAD",

	OpVarINIT: "VAR_INIT",

	OpBuildSTRING: "BUILD_STRING",

	OpGetMember:  "GET_MEMBER",
	OpCallMethod: "CALL_METHOD",

	OpBinAdd: "BIN_ADD",
	OpBinSub: "BIN_SUB",
	OpBinMul: "BIN_MUL",
	OpBinDiv: "BIN_DIV",
	OpBinMod: "BIN_MOD",
	OpUnNeg:  "UN_NEG",

	OpCmpEQ: "CMP_EQ",
	OpCmpNE: "CMP_NE",
	OpCmpLT: "CMP_LT",
	OpCmpLE: "CMP_LE",
	OpCmpGT: "CMP_GT",
	OpCmpGE: "CMP_GE",

	OpLogAnd: "LOG_AND",
	OpLogOr:  "LOG_OR",
	OpLogNot: "LOG_NOT",
}

func (op OperationCode) String() string {
	if name, ok := operationNames[op]; ok {
		return name
	}
	return "UNKNOWN"
}
