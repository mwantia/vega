package value

import (
	"fmt"
	"math"
	"strconv"
)

// Long represents a 64-bit integer value.
type Long struct {
	Value int64
}

var _ Value = (*Long)(nil)
var _ Numeric = (*Long)(nil)
var _ Comparable = (*Long)(nil)
var _ Methodable = (*Long)(nil)

func NewLong(i int64) *Long {
	return &Long{Value: i}
}

func (l *Long) Type() string {
	return TypeLong
}

func (l *Long) String() string {
	return strconv.FormatInt(l.Value, 10)
}

func (l *Long) Boolean() bool {
	return l.Value != 0
}

func (l *Long) Equal(other Value) bool {
	switch o := other.(type) {
	case *Long:
		return l.Value == o.Value
	case *Integer:
		return l.Value == int64(o.Value)
	case *Short:
		return l.Value == int64(o.Value)
	case *Float:
		return float64(l.Value) == o.Value
	}
	return false
}

func (l *Long) Method(name string, args []Value) (Value, error) {
	return nil, fmt.Errorf("unknown method call")
}

func (l *Long) Compare(other Value) (int, bool) {
	switch o := other.(type) {
	case *Long:
		if l.Value < o.Value {
			return -1, true
		} else if l.Value > o.Value {
			return 1, true
		}
		return 0, true
	case *Integer:
		ov := int64(o.Value)
		if l.Value < ov {
			return -1, true
		} else if l.Value > ov {
			return 1, true
		}
		return 0, true
	case *Short:
		ov := int64(o.Value)
		if l.Value < ov {
			return -1, true
		} else if l.Value > ov {
			return 1, true
		}
		return 0, true
	case *Float:
		fv := float64(l.Value)
		if fv < o.Value {
			return -1, true
		} else if fv > o.Value {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func (l *Long) Add(other Value) (Value, error) {
	switch o := other.(type) {
	case *Long:
		return NewLong(l.Value + o.Value), nil
	case *Integer:
		return NewLong(l.Value + int64(o.Value)), nil
	case *Short:
		return NewLong(l.Value + int64(o.Value)), nil
	case *Float:
		return NewFloat(float64(l.Value) + o.Value), nil
	}
	return nil, fmt.Errorf("cannot add %s to long", other.Type())
}

func (l *Long) Sub(other Value) (Value, error) {
	switch o := other.(type) {
	case *Long:
		return NewLong(l.Value - o.Value), nil
	case *Integer:
		return NewLong(l.Value - int64(o.Value)), nil
	case *Short:
		return NewLong(l.Value - int64(o.Value)), nil
	case *Float:
		return NewFloat(float64(l.Value) - o.Value), nil
	}
	return nil, fmt.Errorf("cannot subtract %s from long", other.Type())
}

func (l *Long) Mul(other Value) (Value, error) {
	switch o := other.(type) {
	case *Long:
		return NewLong(l.Value * o.Value), nil
	case *Integer:
		return NewLong(l.Value * int64(o.Value)), nil
	case *Short:
		return NewLong(l.Value * int64(o.Value)), nil
	case *Float:
		return NewFloat(float64(l.Value) * o.Value), nil
	}
	return nil, fmt.Errorf("cannot multiply long by %s", other.Type())
}

func (l *Long) Div(other Value) (Value, error) {
	switch o := other.(type) {
	case *Long:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewLong(l.Value / o.Value), nil
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewLong(l.Value / int64(o.Value)), nil
	case *Short:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewLong(l.Value / int64(o.Value)), nil
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(float64(l.Value) / o.Value), nil
	}
	return nil, fmt.Errorf("cannot divide long by %s", other.Type())
}

func (l *Long) Mod(other Value) (Value, error) {
	switch o := other.(type) {
	case *Long:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewLong(l.Value % o.Value), nil
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewLong(l.Value % int64(o.Value)), nil
	case *Short:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewLong(l.Value % int64(o.Value)), nil
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(float64(l.Value), o.Value)), nil
	}
	return nil, fmt.Errorf("cannot modulo long by %s", other.Type())
}

func (l *Long) Neg() (Value, error) {
	return NewLong(-l.Value), nil
}
