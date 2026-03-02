package value

// Boolean wraps bool. Size = 1 byte.
// The view slice points into the alloc buffer — no data is owned.
type Boolean struct {
	view []byte
}

func NewBoolean(view []byte) *Boolean {
	return &Boolean{
		view: view,
	}
}

func (v *Boolean) Type() string {
	return "boolean"
}

func (v *Boolean) String() string {
	if v.Data() {
		return "true"
	}

	return "false"
}

func (v *Boolean) Size() byte {
	return 1
}

func (v *Boolean) Data() bool {
	return v.view[0] != 0
}

func (v *Boolean) View() []byte {
	return v.view
}

var _ Allocatable = (*Boolean)(nil)
