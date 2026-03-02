package compiler

import "github.com/mwantia/vega/pkg/value"

// ParamDefinition describes a single function parameter: its name, slot ID within
// the function's alloc buffer, and its type constraints.
// For primitive parameters Tag/Mask are set; for struct parameters Stencil is set.
type ParamDefinition struct {
	Name    string
	SlotID  int
	Tag     value.TypeTag
	Mask    byte
	Stencil *StencilDefinition // non-nil for struct-typed parameters
}

// FunctionDefinition holds everything the runtime needs to call a user-defined
// (Vega-implemented) function: its compiled bytecode, parameter metadata,
// and the total frame size required to host parameters and local variables.
type FunctionDefinition struct {
	Name      string
	ByteCode  *ByteCode
	Params    []ParamDefinition
	FrameSize int // total alloc size for parameters + overhead
}

// StencilDefinition is a compile-time layout recipe for packing primitive fields
// contiguously in the byte buffer. At runtime a struct is just bytes
// at known offsets — the stencil captures the layout.
type StencilDefinition struct {
	Name      string
	Fields    []StencilFieldLayout
	TotalSize int
}

// FieldLayout describes a single field within a stencil.
type StencilFieldLayout struct {
	Name     string
	Offset   int           // cumulative byte offset within the stencil
	Tag      value.TypeTag // type tag for this field
	Capacity int           // byte capacity for TagSlice fields (0 for scalar fields)
}

// LookupField returns the FieldLayout for the named field, or false.
func (sd *StencilDefinition) LookupField(name string) (StencilFieldLayout, bool) {
	for _, f := range sd.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return StencilFieldLayout{}, false
}
