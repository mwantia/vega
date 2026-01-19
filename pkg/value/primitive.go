package value

import (
	"fmt"
	"math"
	"strconv"
)

// StringValue represents a string value.
type StringValue struct {
	Val string
}

var _ Value = (*StringValue)(nil)
var _ Comparable = (*StringValue)(nil)

func NewString(s string) *StringValue {
	return &StringValue{Val: s}
}

func (s *StringValue) Type() string   { return TypeString }
func (s *StringValue) String() string { return s.Val }
func (s *StringValue) Boolean() bool  { return len(s.Val) > 0 }

func (s *StringValue) Equal(other Value) bool {
	if o, ok := other.(*StringValue); ok {
		return s.Val == o.Val
	}
	return false
}

func (s *StringValue) Compare(other Value) (int, bool) {
	if o, ok := other.(*StringValue); ok {
		if s.Val < o.Val {
			return -1, true
		} else if s.Val > o.Val {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

// IntValue represents an integer value.
type IntValue struct {
	Val int64
}

var _ Value = (*IntValue)(nil)
var _ Numeric = (*IntValue)(nil)
var _ Comparable = (*IntValue)(nil)

func NewInt(i int64) *IntValue {
	return &IntValue{Val: i}
}

func (i *IntValue) Type() string   { return TypeInt }
func (i *IntValue) String() string { return strconv.FormatInt(i.Val, 10) }
func (i *IntValue) Boolean() bool  { return i.Val != 0 }

func (i *IntValue) Equal(other Value) bool {
	switch o := other.(type) {
	case *IntValue:
		return i.Val == o.Val
	case *FloatValue:
		return float64(i.Val) == o.Val
	}
	return false
}

func (i *IntValue) Compare(other Value) (int, bool) {
	switch o := other.(type) {
	case *IntValue:
		if i.Val < o.Val {
			return -1, true
		} else if i.Val > o.Val {
			return 1, true
		}
		return 0, true
	case *FloatValue:
		fv := float64(i.Val)
		if fv < o.Val {
			return -1, true
		} else if fv > o.Val {
			return 1, true
		}
		return 0, true
	}
	return 0, false
}

func (i *IntValue) Add(other Value) (Value, error) {
	switch o := other.(type) {
	case *IntValue:
		return NewInt(i.Val + o.Val), nil
	case *FloatValue:
		return NewFloat(float64(i.Val) + o.Val), nil
	}
	return nil, fmt.Errorf("cannot add %s to int", other.Type())
}

func (i *IntValue) Sub(other Value) (Value, error) {
	switch o := other.(type) {
	case *IntValue:
		return NewInt(i.Val - o.Val), nil
	case *FloatValue:
		return NewFloat(float64(i.Val) - o.Val), nil
	}
	return nil, fmt.Errorf("cannot subtract %s from int", other.Type())
}

func (i *IntValue) Mul(other Value) (Value, error) {
	switch o := other.(type) {
	case *IntValue:
		return NewInt(i.Val * o.Val), nil
	case *FloatValue:
		return NewFloat(float64(i.Val) * o.Val), nil
	}
	return nil, fmt.Errorf("cannot multiply int by %s", other.Type())
}

func (i *IntValue) Div(other Value) (Value, error) {
	switch o := other.(type) {
	case *IntValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewInt(i.Val / o.Val), nil
	case *FloatValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(float64(i.Val) / o.Val), nil
	}
	return nil, fmt.Errorf("cannot divide int by %s", other.Type())
}

func (i *IntValue) Mod(other Value) (Value, error) {
	switch o := other.(type) {
	case *IntValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewInt(i.Val % o.Val), nil
	case *FloatValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(float64(i.Val), o.Val)), nil
	}
	return nil, fmt.Errorf("cannot modulo int by %s", other.Type())
}

func (i *IntValue) Neg() Value {
	return NewInt(-i.Val)
}

// FloatValue represents a floating-point value.
type FloatValue struct {
	Val float64
}

var _ Value = (*FloatValue)(nil)
var _ Numeric = (*FloatValue)(nil)
var _ Comparable = (*FloatValue)(nil)

func NewFloat(f float64) *FloatValue {
	return &FloatValue{Val: f}
}

func (f *FloatValue) Type() string   { return TypeFloat }
func (f *FloatValue) String() string { return strconv.FormatFloat(f.Val, 'f', -1, 64) }
func (f *FloatValue) Boolean() bool  { return f.Val != 0 }

func (f *FloatValue) Equal(other Value) bool {
	switch o := other.(type) {
	case *FloatValue:
		return f.Val == o.Val
	case *IntValue:
		return f.Val == float64(o.Val)
	}
	return false
}

func (f *FloatValue) Compare(other Value) (int, bool) {
	var ov float64
	switch o := other.(type) {
	case *FloatValue:
		ov = o.Val
	case *IntValue:
		ov = float64(o.Val)
	default:
		return 0, false
	}

	if f.Val < ov {
		return -1, true
	} else if f.Val > ov {
		return 1, true
	}
	return 0, true
}

func (f *FloatValue) Add(other Value) (Value, error) {
	switch o := other.(type) {
	case *FloatValue:
		return NewFloat(f.Val + o.Val), nil
	case *IntValue:
		return NewFloat(f.Val + float64(o.Val)), nil
	}
	return nil, fmt.Errorf("cannot add %s to float", other.Type())
}

func (f *FloatValue) Sub(other Value) (Value, error) {
	switch o := other.(type) {
	case *FloatValue:
		return NewFloat(f.Val - o.Val), nil
	case *IntValue:
		return NewFloat(f.Val - float64(o.Val)), nil
	}
	return nil, fmt.Errorf("cannot subtract %s from float", other.Type())
}

func (f *FloatValue) Mul(other Value) (Value, error) {
	switch o := other.(type) {
	case *FloatValue:
		return NewFloat(f.Val * o.Val), nil
	case *IntValue:
		return NewFloat(f.Val * float64(o.Val)), nil
	}
	return nil, fmt.Errorf("cannot multiply float by %s", other.Type())
}

func (f *FloatValue) Div(other Value) (Value, error) {
	switch o := other.(type) {
	case *FloatValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(f.Val / o.Val), nil
	case *IntValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("division by zero")
		}
		return NewFloat(f.Val / float64(o.Val)), nil
	}
	return nil, fmt.Errorf("cannot divide float by %s", other.Type())
}

func (f *FloatValue) Mod(other Value) (Value, error) {
	switch o := other.(type) {
	case *FloatValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(f.Val, o.Val)), nil
	case *IntValue:
		if o.Val == 0 {
			return nil, fmt.Errorf("modulo by zero")
		}
		return NewFloat(math.Mod(f.Val, float64(o.Val))), nil
	}
	return nil, fmt.Errorf("cannot modulo float by %s", other.Type())
}

func (f *FloatValue) Neg() Value {
	return NewFloat(-f.Val)
}

// BoolValue represents a boolean value.
type BoolValue struct {
	Val bool
}

var _ Value = (*BoolValue)(nil)

// True and False are singleton boolean values.
var (
	True  = &BoolValue{Val: true}
	False = &BoolValue{Val: false}
)

func NewBool(b bool) *BoolValue {
	if b {
		return True
	}
	return False
}

func (b *BoolValue) Type() string { return TypeBool }

func (b *BoolValue) String() string {
	if b.Val {
		return "true"
	}
	return "false"
}

func (b *BoolValue) Boolean() bool { return b.Val }

func (b *BoolValue) Equal(other Value) bool {
	if o, ok := other.(*BoolValue); ok {
		return b.Val == o.Val
	}
	return false
}

// NilValue represents the nil value.
type NilValue struct{}

var _ Value = (*NilValue)(nil)

// Nil is the singleton nil value.
var Nil = &NilValue{}

func NewNil() *NilValue {
	return Nil
}

func (n *NilValue) Type() string    { return TypeNil }
func (n *NilValue) String() string  { return "nil" }
func (n *NilValue) Boolean() bool   { return false }
func (n *NilValue) Equal(other Value) bool {
	_, ok := other.(*NilValue)
	return ok
}
