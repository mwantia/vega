package value

import (
	"encoding/binary"
	"math"
	"strconv"
)

// Float wraps float32. Size = 4 bytes.
// The view slice points into the alloc buffer — no data is owned.
type Float struct {
	view []byte
}

func NewFloat(view []byte) *Float {
	return &Float{
		view: view,
	}
}

func (v *Float) Type() string {
	return "float"
}

func (v *Float) String() string {
	return strconv.FormatFloat(float64(v.Data()), 'f', -1, 32)
}

func (v *Float) Size() byte {
	return 4
}

func (v *Float) Data() float32 {
	return math.Float32frombits(binary.LittleEndian.Uint32(v.view))
}

func (v *Float) View() []byte {
	return v.view
}

var _ Allocatable = (*Float)(nil)
