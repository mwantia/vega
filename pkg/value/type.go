package value

import "strings"

type Type struct {
	Value string
}

var _ Value = (*Type)(nil)
var _ Comparable = (*Type)(nil)

func NewType(s string) *Type {
	return &Type{Value: s}
}

func (v *Type) Type() string {
	return TypeType
}

func (v *Type) String() string {
	return v.Value
}

func (v *Type) Boolean() bool {
	return len(v.Value) > 0
}

func (v *Type) Equal(other Value) bool {
	if o, ok := other.(*Type); ok {
		return v.Value == o.Value
	}
	return false
}

func (s *Type) Compare(other Value) (int, bool) {
	if o, ok := other.(*Type); ok {
		return strings.Compare(s.Value, o.Value), true
	}
	return 0, false
}
