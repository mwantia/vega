package value

import "strconv"

// Byte wraps uint8. Size = 1 byte.
// The view slice points into the alloc buffer — no data is owned.
type Byte struct {
	view []byte
}

func NewByte(view []byte) *Byte {
	return &Byte{
		view: view,
	}
}

func (v *Byte) Type() string {
	return "byte"
}

func (v *Byte) String() string {
	return strconv.Itoa(int(v.view[0]))
}

func (v *Byte) Size() byte {
	return 1
}

func (v *Byte) Data() byte {
	return v.view[0]
}

func (v *Byte) View() []byte {
	return v.view
}

var _ Allocatable = (*Byte)(nil)
