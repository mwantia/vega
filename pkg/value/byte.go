package value

import "strconv"

// ByteValue wraps uint8. Size = 1 byte.
// The view slice points into the alloc buffer — no data is owned.
type ByteValue struct {
	view []byte
}

func NewByte(view []byte) *ByteValue {
	return &ByteValue{
		view: view,
	}
}

func (v *ByteValue) Type() string {
	return "byte"
}

func (v *ByteValue) String() string {
	return strconv.Itoa(int(v.view[0]))
}

func (v *ByteValue) Size() byte {
	return 1
}

func (v *ByteValue) Data() byte {
	return v.view[0]
}

func (v *ByteValue) View() []byte {
	return v.view
}

var _ Allocable = (*ByteValue)(nil)
