package value

// Boolean represents a boolean value.
type Boolean struct {
	Value bool
}

var _ Value = (*Boolean)(nil)

// True and False are singleton boolean values.
var (
	True  = &Boolean{Value: true}
	False = &Boolean{Value: false}
)

func NewBoolean(b bool) *Boolean {
	if b {
		return True
	}
	return False
}

func (b *Boolean) Type() string { return TypeBool }

func (b *Boolean) String() string {
	if b.Value {
		return "true"
	}
	return "false"
}

func (b *Boolean) Boolean() bool {
	return b.Value
}

func (b *Boolean) Equal(other Value) bool {
	if o, ok := other.(*Boolean); ok {
		return b.Value == o.Value
	}
	return false
}
