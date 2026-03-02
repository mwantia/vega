package value

import (
	"encoding/binary"
	"strconv"
)

// Integer wraps int32. Size = 4 bytes.
// The view slice points into the alloc buffer — no data is owned.
type Integer struct {
	view []byte
}

func NewInteger(view []byte) *Integer {
	return &Integer{
		view: view,
	}
}

func (v *Integer) Type() string {
	return "int"
}

func (v *Integer) String() string {
	dat := binary.LittleEndian.Uint32(v.view)
	return strconv.FormatUint(uint64(dat), 10)
}

func (v *Integer) Size() byte {
	return 4
}

func (v *Integer) Data() int {
	return int(binary.LittleEndian.Uint32(v.view))
}

func (v *Integer) View() []byte {
	return v.view
}

var _ Allocatable = (*Integer)(nil)
