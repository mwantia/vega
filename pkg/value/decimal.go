package value

import (
	"encoding/binary"
	"math"
	"strconv"
)

// DecimalValue wraps float64. Size = 8 bytes.
// The view slice points into the alloc buffer — no data is owned.
type DecimalValue struct {
	view []byte
}

func NewDecimal(view []byte) *DecimalValue {
	return &DecimalValue{
		view: view,
	}
}

func (v *DecimalValue) Type() string {
	return "decimal"
}

func (v *DecimalValue) String() string {
	return strconv.FormatFloat(v.Data(), 'f', -1, 64)
}

func (v *DecimalValue) Size() byte {
	return 8
}

func (v *DecimalValue) Data() float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(v.view))
}

func (v *DecimalValue) View() []byte {
	return v.view
}

var _ Allocable = (*DecimalValue)(nil)
