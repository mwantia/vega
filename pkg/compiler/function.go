package compiler

import "github.com/mwantia/vega/pkg/value"

// ParamDef describes a single function parameter: its name, slot ID within
// the function's alloc buffer, and its type constraints.
// For primitive parameters Tag/Mask are set; for struct parameters Stencil is set.
type ParamDef struct {
	Name    string
	SlotID  int
	Tag     value.TypeTag
	Mask    byte
	Stencil *Stencil // non-nil for struct-typed parameters
}

// FunctionDef holds everything the runtime needs to call a user-defined
// (Vega-implemented) function: its compiled bytecode, parameter metadata,
// and the total frame size required to host parameters and local variables.
type FunctionDef struct {
	Name      string
	ByteCode  *ByteCode
	Params    []ParamDef
	FrameSize int // total alloc size for parameters + overhead
}
