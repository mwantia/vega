package value

import (
	"fmt"
	"math"
	"strconv"
)

// Float represents a floating-point value.
type Float struct {
	Value float64
}

var _ Value = (*Float)(nil)
var _ Numeric = (*Float)(nil)
var _ Comparable = (*Float)(nil)
var _ Methodable = (*Float)(nil)

func NewFloat(f float64) *Float {
	return &Float{Value: f}
}

func (f *Float) Type() string {
	return TypeFloat
}

func (f *Float) String() string {
	return strconv.FormatFloat(f.Value, 'f', -1, 64)
}

func (f *Float) Boolean() bool {
	return f.Value != 0
}

func (f *Float) Equal(other Value) bool {
	switch o := other.(type) {
	case *Float:
		return f.Value == o.Value
	case *Integer:
		return f.Value == float64(o.Value)
	}
	return false
}

func (v *Float) Method(name string, args []Value) (Value, error) {
	return nil, fmt.Errorf("unknown method call")
}

func (f *Float) Compare(other Value) (int, bool) {
	var ov float64
	switch o := other.(type) {
	case *Float:
		ov = o.Value
	case *Integer:
		ov = float64(o.Value)
	default:
		return 0, false
	}

	if f.Value < ov {
		return -1, true
	} else if f.Value > ov {
		return 1, true
	}
	return 0, true
}

func (f *Float) Add(other Value) (Value, error) {
	switch o := other.(type) {
	case *Float:
		return NewFloat(f.Value + o.Value), nil
	case *Integer:
		return NewFloat(f.Value + float64(o.Value)), nil
	}
	return nil, fmt.Errorf("cannot add %s to float", other.Type())
}

func (f *Float) Sub(other Value) (Value, error) {
	switch o := other.(type) {
	case *Float:
		return NewFloat(f.Value - o.Value), nil
	case *Integer:
		return NewFloat(f.Value - float64(o.Value)), nil
	}
	return nil, fmt.Errorf("cannot subtract %s from float", other.Type())
}

func (f *Float) Mul(other Value) (Value, error) {
	switch o := other.(type) {
	case *Float:
		return NewFloat(f.Value * o.Value), nil
	case *Integer:
		return NewFloat(f.Value * float64(o.Value)), nil
	}
	return nil, fmt.Errorf("cannot multiply float by %s", other.Type())
}

func (f *Float) Div(other Value) (Value, error) {
	switch o := other.(type) {
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(f.Value / o.Value), nil
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(f.Value / float64(o.Value)), nil
	}
	return nil, fmt.Errorf("cannot divide float by %s", other.Type())
}

func (f *Float) Mod(other Value) (Value, error) {
	switch o := other.(type) {
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(f.Value, o.Value)), nil
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(f.Value, float64(o.Value))), nil
	}
	return nil, fmt.Errorf("cannot modulo float by %s", other.Type())
}

func (f *Float) Neg() (Value, error) {
	return NewFloat(-f.Value), nil
}
