package value

import (
	"encoding/binary"
	"strconv"
)

// LongValue wraps int64. Size = 8 bytes.
// The view slice points into the alloc buffer — no data is owned.
type LongValue struct {
	view []byte
}

func NewLong(view []byte) *LongValue {
	return &LongValue{
		view: view,
	}
}

func (v *LongValue) Type() string {
	return "long"
}

func (v *LongValue) String() string {
	return strconv.FormatInt(v.Data(), 10)
}

func (v *LongValue) Size() byte {
	return 8
}

func (v *LongValue) Data() int64 {
	return int64(binary.LittleEndian.Uint64(v.view))
}

func (v *LongValue) View() []byte {
	return v.view
}

var _ Allocable = (*LongValue)(nil)
