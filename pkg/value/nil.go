package value

// NilValue represents the nil value.
type NilValue struct{}

var _ Value = (*NilValue)(nil)

// Nil is the singleton nil value.
var Nil = &NilValue{}

func NewNil() *NilValue {
	return Nil
}

func (n *NilValue) Type() string {
	return TypeNil
}

func (n *NilValue) String() string {
	return "nil"
}

func (n *NilValue) Boolean() bool {
	return false
}

func (n *NilValue) Equal(other Value) bool {
	_, ok := other.(*NilValue)
	return ok
}
