package value

import (
	"encoding/binary"
	"strconv"
)

// IntegerValue wraps int32. Size = 4 bytes.
// The view slice points into the alloc buffer — no data is owned.
type IntegerValue struct {
	view []byte
}

func NewInteger(view []byte) *IntegerValue {
	return &IntegerValue{
		view: view,
	}
}

func (v *IntegerValue) Type() string {
	return "int"
}

func (v *IntegerValue) String() string {
	dat := binary.LittleEndian.Uint32(v.view)
	return strconv.FormatUint(uint64(dat), 10)
}

func (v *IntegerValue) Size() byte {
	return 4
}

func (v *IntegerValue) Data() int32 {
	return int32(binary.LittleEndian.Uint32(v.view))
}

func (v *IntegerValue) View() []byte {
	return v.view
}

var _ Allocable = (*IntegerValue)(nil)
