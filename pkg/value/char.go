package value

// CharValue wraps a single Unicode code point (rune).
// It is a plain Value — it is NOT Allocable and has no TypeTag.
// Char literals in scripts are lowered to int32.
type CharValue struct {
	data rune
}

func NewChar(r rune) *CharValue {
	return &CharValue{data: r}
}

func (v *CharValue) Type() string   { return "char" }
func (v *CharValue) String() string { return string(v.data) }
func (v *CharValue) Data() rune     { return v.data }

var _ Value = (*CharValue)(nil)
