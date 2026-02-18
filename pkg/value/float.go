package value

import (
	"encoding/binary"
	"math"
	"strconv"
)

// FloatValue wraps float32. Size = 4 bytes.
// The view slice points into the alloc buffer — no data is owned.
type FloatValue struct {
	view []byte
}

func NewFloat(view []byte) *FloatValue {
	return &FloatValue{
		view: view,
	}
}

func (v *FloatValue) Type() string {
	return "float"
}

func (v *FloatValue) String() string {
	return strconv.FormatFloat(float64(v.Data()), 'f', -1, 32)
}

func (v *FloatValue) Size() byte {
	return 4
}

func (v *FloatValue) Data() float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(v.view))
}

func (v *FloatValue) View() []byte {
	return v.view
}

var _ Allocable = (*FloatValue)(nil)
