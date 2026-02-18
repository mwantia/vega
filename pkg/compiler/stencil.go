package compiler

import (
	"fmt"

	"github.com/mwantia/vega/pkg/value"
)

// FieldLayout describes a single field within a stencil.
type FieldLayout struct {
	Name   string
	Offset int           // cumulative byte offset within the stencil
	Tag    value.TypeTag // type tag for this field
}

// Stencil is a compile-time layout recipe for packing primitive fields
// contiguously in the byte buffer. At runtime a struct is just bytes
// at known offsets — the stencil captures the layout.
type Stencil struct {
	Name      string
	Fields    []FieldLayout
	TotalSize int
}

// StencilField describes a field for use with RegisterStencil.
type StencilField struct {
	Name string
	Tag  value.TypeTag
}

// Field creates a StencilField for use with RegisterStencil.
func Field(name string, tag value.TypeTag) StencilField {
	return StencilField{Name: name, Tag: tag}
}

// RegisterStencil registers a named stencil on the compiler so it is
// available to scripts without a `struct` declaration. Offsets and
// total size are computed automatically from the field list.
//
//	c := compiler.NewCompiler()
//	c.RegisterStencil("record",
//	    compiler.Field("id", value.TagInteger),
//	    compiler.Field("active", value.TagBoolean),
//	    compiler.Field("score", value.TagFloat),
//	)
func (c *Compiler) RegisterStencil(name string, fields ...StencilField) error {
	if name == "" {
		return fmt.Errorf("stencil name cannot be empty")
	}
	stencil := &Stencil{
		Name:   name,
		Fields: make([]FieldLayout, 0, len(fields)),
	}
	offset := 0
	for _, f := range fields {
		size := value.SizeForTag(f.Tag)
		if size == 0 {
			return fmt.Errorf("stencil '%s': unknown type tag %d for field '%s'", name, f.Tag, f.Name)
		}
		stencil.Fields = append(stencil.Fields, FieldLayout{
			Name:   f.Name,
			Offset: offset,
			Tag:    f.Tag,
		})
		offset += size
	}
	stencil.TotalSize = offset
	c.stencils[name] = stencil
	return nil
}

// LookupStencil returns a registered stencil by name, or nil.
func (c *Compiler) LookupStencil(name string) *Stencil {
	return c.stencils[name]
}

// LookupField returns the FieldLayout for the named field, or false.
func (s *Stencil) LookupField(name string) (FieldLayout, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return FieldLayout{}, false
}

// LookupIndex returns the FieldLayout for the given positional index, or false.
func (s *Stencil) LookupIndex(idx int) (FieldLayout, bool) {
	if idx < 0 || idx >= len(s.Fields) {
		return FieldLayout{}, false
	}
	return s.Fields[idx], true
}
