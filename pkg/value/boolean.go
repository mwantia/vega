package value

// BooleanValue wraps bool. Size = 1 byte.
// The view slice points into the alloc buffer — no data is owned.
type BooleanValue struct {
	view []byte
}

func NewBoolean(view []byte) *BooleanValue {
	return &BooleanValue{
		view: view,
	}
}

func (v *BooleanValue) Type() string {
	return "boolean"
}

func (v *BooleanValue) String() string {
	if v.Data() {
		return "true"
	}

	return "false"
}

func (v *BooleanValue) Size() byte {
	return 1
}

func (v *BooleanValue) Data() bool {
	return v.view[0] != 0
}

func (v *BooleanValue) View() []byte {
	return v.view
}

var _ Allocable = (*BooleanValue)(nil)
