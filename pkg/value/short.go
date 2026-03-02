package value

import (
	"encoding/binary"
	"strconv"
)

// Short wraps int16. Size = 2 bytes.
// The view slice points into the alloc buffer — no data is owned.
type Short struct {
	view []byte
}

func NewShort(view []byte) *Short {
	return &Short{
		view: view,
	}
}

func (v *Short) Type() string {
	return "short"
}

func (v *Short) String() string {
	return strconv.FormatInt(int64(v.Data()), 10)
}

func (v *Short) Size() byte {
	return 2
}

func (v *Short) Data() int16 {
	return int16(binary.LittleEndian.Uint16(v.view))
}

func (v *Short) View() []byte {
	return v.view
}

var _ Allocatable = (*Short)(nil)
