package value

import (
	"encoding/binary"
	"math"
	"strconv"
)

// Decimal wraps float64. Size = 8 bytes.
// The view slice points into the alloc buffer — no data is owned.
type Decimal struct {
	view []byte
}

func NewDecimal(view []byte) *Decimal {
	return &Decimal{
		view: view,
	}
}

func (v *Decimal) Type() string {
	return "decimal"
}

func (v *Decimal) String() string {
	return strconv.FormatFloat(v.Data(), 'f', -1, 64)
}

func (v *Decimal) Size() byte {
	return 8
}

func (v *Decimal) Data() float64 {
	return math.Float64frombits(binary.LittleEndian.Uint64(v.view))
}

func (v *Decimal) View() []byte {
	return v.view
}

var _ Allocatable = (*Decimal)(nil)
