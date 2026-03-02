package value

import (
	"encoding/binary"
	"strconv"
)

// Long wraps int64. Size = 8 bytes.
// The view slice points into the alloc buffer — no data is owned.
type Long struct {
	view []byte
}

func NewLong(view []byte) *Long {
	return &Long{
		view: view,
	}
}

func (v *Long) Type() string {
	return "long"
}

func (v *Long) String() string {
	return strconv.FormatInt(v.Data(), 10)
}

func (v *Long) Size() byte {
	return 8
}

func (v *Long) Data() int64 {
	return int64(binary.LittleEndian.Uint64(v.view))
}

func (v *Long) View() []byte {
	return v.view
}

var _ Allocatable = (*Long)(nil)
