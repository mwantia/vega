package value

import (
	"fmt"
	"math"
	"strconv"
)

// Short represents a 16-bit integer value.
type Short struct {
	Value int16
}

var _ Value = (*Short)(nil)
var _ Numeric = (*Short)(nil)
var _ Comparable = (*Short)(nil)
var _ Methodable = (*Short)(nil)

func NewShort(i int16) *Short {
	return &Short{Value: i}
}

func (s *Short) Type() string {
	return TypeShort
}

func (s *Short) String() string {
	return strconv.FormatInt(int64(s.Value), 10)
}

func (s *Short) Boolean() bool {
	return s.Value != 0
}

func (s *Short) Equal(other Value) bool {
	switch o := other.(type) {
	case *Short:
		return s.Value == o.Value
	case *Integer:
		return int(s.Value) == o.Value
	case *Long:
		return int64(s.Value) == o.Value
	case *Float:
		return float64(s.Value) == o.Value
	}
	return false
}

func (s *Short) Method(name string, args []Value) (Value, error) {
	return nil, fmt.Errorf("unknown method call")
}

func (s *Short) Compare(other Value) (int, bool) {
	switch o := other.(type) {
	case *Short:
		if s.Value < o.Value {
			return -1, true
		} else if s.Value > o.Value {
			return 1, true
		}
		return 0, true
	case *Integer:
		sv := int(s.Value)
		if sv < o.Value {
			return -1, true
		} else if sv > o.Value {
			return 1, true
		}
		return 0, true
	case *Long:
		sv := int64(s.Value)
		if sv < o.Value {
			return -1, true
		} else if sv > o.Value {
			return 1, true
		}
		return 0, true
	case *Float:
		fv := float64(s.Value)
		if fv < o.Value {
			return -1, true
		} else if fv > o.Value {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func (s *Short) Add(other Value) (Value, error) {
	switch o := other.(type) {
	case *Short:
		return NewShort(s.Value + o.Value), nil
	case *Integer:
		return NewInteger(int(s.Value) + o.Value), nil
	case *Long:
		return NewLong(int64(s.Value) + o.Value), nil
	case *Float:
		return NewFloat(float64(s.Value) + o.Value), nil
	}
	return nil, fmt.Errorf("cannot add %s to short", other.Type())
}

func (s *Short) Sub(other Value) (Value, error) {
	switch o := other.(type) {
	case *Short:
		return NewShort(s.Value - o.Value), nil
	case *Integer:
		return NewInteger(int(s.Value) - o.Value), nil
	case *Long:
		return NewLong(int64(s.Value) - o.Value), nil
	case *Float:
		return NewFloat(float64(s.Value) - o.Value), nil
	}
	return nil, fmt.Errorf("cannot subtract %s from short", other.Type())
}

func (s *Short) Mul(other Value) (Value, error) {
	switch o := other.(type) {
	case *Short:
		return NewShort(s.Value * o.Value), nil
	case *Integer:
		return NewInteger(int(s.Value) * o.Value), nil
	case *Long:
		return NewLong(int64(s.Value) * o.Value), nil
	case *Float:
		return NewFloat(float64(s.Value) * o.Value), nil
	}
	return nil, fmt.Errorf("cannot multiply short by %s", other.Type())
}

func (s *Short) Div(other Value) (Value, error) {
	switch o := other.(type) {
	case *Short:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewShort(s.Value / o.Value), nil
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewInteger(int(s.Value) / o.Value), nil
	case *Long:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewLong(int64(s.Value) / o.Value), nil
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(float64(s.Value) / o.Value), nil
	}
	return nil, fmt.Errorf("cannot divide short by %s", other.Type())
}

func (s *Short) Mod(other Value) (Value, error) {
	switch o := other.(type) {
	case *Short:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewShort(s.Value % o.Value), nil
	case *Integer:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewInteger(int(s.Value) % o.Value), nil
	case *Long:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewLong(int64(s.Value) % o.Value), nil
	case *Float:
		if o.Value == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(float64(s.Value), o.Value)), nil
	}
	return nil, fmt.Errorf("cannot modulo short by %s", other.Type())
}

func (s *Short) Neg() (Value, error) {
	return NewShort(-s.Value), nil
}
