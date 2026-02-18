package value

import (
	"encoding/binary"
	"strconv"
)

// ShortValue wraps int16. Size = 2 bytes.
// The view slice points into the alloc buffer — no data is owned.
type ShortValue struct {
	view []byte
}

func NewShort(view []byte) *ShortValue {
	return &ShortValue{
		view: view,
	}
}

func (v *ShortValue) Type() string {
	return "short"
}

func (v *ShortValue) String() string {
	return strconv.FormatInt(int64(v.Data()), 10)
}

func (v *ShortValue) Size() byte {
	return 2
}

func (v *ShortValue) Data() int16 {
	return int16(binary.LittleEndian.Uint16(v.view))
}

func (v *ShortValue) View() []byte {
	return v.view
}

var _ Allocable = (*ShortValue)(nil)
