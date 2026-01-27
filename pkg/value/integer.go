package value

import (
	"fmt"
	"math"
	"strconv"
)

// Integer represents an integer value.
type Integer struct {
	Value int64
}

var _ Value = (*Integer)(nil)
var _ Numeric = (*Integer)(nil)
var _ Comparable = (*Integer)(nil)
var _ Methodable = (*Integer)(nil)

func NewInteger(i int64) *Integer {
	return &Integer{Value: i}
}

func (i *Integer) Type() string {
	return TypeInt
}

func (i *Integer) String() string {
	return strconv.FormatInt(i.Value, 10)
}

func (i *Integer) Boolean() bool {
	return i.Value != 0
}

func (i *Integer) Equal(other Value) bool {
	switch o := other.(type) {
	case *Integer:
		return i.Value == o.Value
	case *Float:
		return float64(i.Value) == o.Value
	}
	return false
}

func (v *Integer) Method(name string, args []Value) (Value, error) {
	return nil, fmt.Errorf("unknown method call")
}

func (i *Integer) Compare(other Value) (int, bool) {
	switch o := other.(type) {
	case *Integer:
		if i.Value < o.Value {
			return -1, true
		} else if i.Value > o.Value {
			return 1, true
		}
		return 0, true
	case *Float:
		fv := float64(i.Value)
		if fv < o.Value {
			return -1, true
		} else if fv > o.Value {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func (i *Integer) Add(other Value) (Value, error) {
	switch o := other.(type) {
	case *Integer:
		return NewInteger(i.Value + o.Value), nil
	case *Float:
		return NewFloat(float64(i.Value) + o.Value), nil
	}
	return nil, fmt.Errorf("cannot add %s to int", other.Type())
}

func (i *Integer) Sub(other Value) (Value, error) {
	switch o := other.(type) {
	case *Integer:
		return NewInteger(i.Value - o.Value), nil
	case *Float:
		return NewFloat(float64(i.Value) - o.Value), nil
	}
	return nil, fmt.Errorf("cannot subtract %s from int", other.Type())
}

func (i *Integer) Mul(other Value) (Value, error) {
	switch o := other.(type) {
	case *Integer:
		return NewInteger(i.Value * o.Value), nil
	case *Float:
		return NewFloat(float64(i.Value) * o.Value), nil
	}
	return nil, fmt.Errorf("cannot multiply int by %s", other.Type())
}

func (i *Integer) Div(other Value) (Value, error) {
	switch o := other.(type) {
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewInteger(i.Value / o.Value), nil
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(float64(i.Value) / o.Value), nil
	}
	return nil, fmt.Errorf("cannot divide int by %s", other.Type())
}

func (i *Integer) Mod(other Value) (Value, error) {
	switch o := other.(type) {
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewInteger(i.Value % o.Value), nil
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(float64(i.Value), o.Value)), nil
	}
	return nil, fmt.Errorf("cannot modulo int by %s", other.Type())
}

func (i *Integer) Neg() Value {
	return NewInteger(-i.Value)
}
