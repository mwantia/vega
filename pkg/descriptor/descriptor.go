package descriptor

import (
	"fmt"

	"github.com/mwantia/vega/pkg/value"
)

// FieldLayoutDescriptor describes one field in a registered struct type.
type FieldLayoutDescriptor struct {
	Name     string
	Tag      value.TypeTag
	Offset   int
	Capacity int // > 0 for slice fields
}

// StencilDescriptor describes a registered struct type available to the
// compiler. It replaces the compiler-internal Stencil and can be registered
// by external code via RegisterStencil.
type StencilDescriptor struct {
	Name    string
	Fields  []FieldLayoutDescriptor
	Methods []MethodDescriptor
	Size    int // total byte width
}

// LookupField returns the FieldLayoutDescriptor for the named field, or false.
func (s *StencilDescriptor) LookupField(name string) (FieldLayoutDescriptor, bool) {
	for _, f := range s.Fields {
		if f.Name == name {
			return f, true
		}
	}
	return FieldLayoutDescriptor{}, false
}

// LookupIndex returns the FieldLayoutDescriptor at the given positional index.
func (s *StencilDescriptor) LookupIndex(idx int) (FieldLayoutDescriptor, bool) {
	if idx < 0 || idx >= len(s.Fields) {
		return FieldLayoutDescriptor{}, false
	}
	return s.Fields[idx], true
}

// LookupMethod returns the MethodDescriptor registered on this stencil.
func (s *StencilDescriptor) LookupMethod(name string) (MethodDescriptor, bool) {
	for _, m := range s.Methods {
		if m.Name == name {
			return m, true
		}
	}
	return MethodDescriptor{}, false
}

// ParameterLayoutDescriptor describes a single parameter in a method or native
// function signature. Optional parameters (Required=false) must trail all
// required ones.
type ParameterLayoutDescriptor struct {
	Name     string
	Tag      value.TypeTag
	Position int
	Default  value.Value // non-nil for optional parameters
	Required bool
}

// MethodDescriptorRuntime
type MethodDescriptorRuntime func(value.Value, DescriptorSession, []value.Value) (value.Value, error)

// MethodDescriptor describes a method callable on a value of a given TypeTag.
// ReturnTag is used by the compiler for type inference at the call site;
// 0 means void (no assignable result).
//
// Params semantics:
//   - nil  → variadic / unvalidated: ValidateArgs is a no-op, Run receives args as-is
//   - []   → zero parameters: any supplied argument is an error
//   - [...] → validated against the listed parameters (count + type)
type MethodDescriptor struct {
	Name      string
	Params    []ParameterLayoutDescriptor
	ReturnTag value.TypeTag
	Run       MethodDescriptorRuntime
}

// ValidateArgs validates args against the descriptor's Params, fills in
// Default values for absent optional parameters, and returns a normalised
// slice of exactly len(Params) entries (nil where an optional was omitted
// and has no Default). When Params is nil the args are returned unchanged.
func (m *MethodDescriptor) ValidateArgs(args []value.Value) ([]value.Value, error) {
	if m.Params == nil {
		return args, nil
	}

	required := 0
	for _, p := range m.Params {
		if p.Required {
			required++
		}
	}
	if len(args) < required {
		return nil, fmt.Errorf("'%s' expects at least %d argument(s), got %d", m.Name, required, len(args))
	}
	if len(args) > len(m.Params) {
		return nil, fmt.Errorf("'%s' expects at most %d argument(s), got %d", m.Name, len(m.Params), len(args))
	}

	result := make([]value.Value, len(m.Params))
	for i, p := range m.Params {
		if i < len(args) {
			arg := args[i]
			if p.Tag != value.TagAny {
				if alloc, ok := arg.(value.Allocatable); ok {
					if got := value.TagFor(alloc); got != p.Tag {
						wantName, _ := value.NameForTag(p.Tag)
						return nil, fmt.Errorf("'%s' parameter '%s': expected %s, got %s",
							m.Name, p.Name, wantName, arg.Type())
					}
				}
			}
			result[i] = arg
		} else {
			result[i] = p.Default // nil for absent optional with no Default set
		}
	}
	return result, nil
}

// MemberDescriptorGetter
type MemberDescriptorGetter func(value.Value) (value.Value, error)

// MemberDescriptorSetter
type MemberDescriptorSetter func(value.Value, value.Value) error

// MemberDescriptor describes a named property readable (and optionally
// writable) on a value of a given TypeTag.
type MemberDescriptor struct {
	Name      string
	Readonly  bool
	ReturnTag value.TypeTag
	Getter    MemberDescriptorGetter
	Setter    MemberDescriptorSetter
}
