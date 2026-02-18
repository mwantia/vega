package value

import "fmt"

// ToInt extracts an integer offset from an Allocable value.
// Supports byte, short, int, and long types.
func ToInt(a Allocable) (int, error) {
	switch v := a.(type) {
	case *ByteValue:
		return int(v.Data()), nil
	case *ShortValue:
		return int(v.Data()), nil
	case *IntegerValue:
		return int(v.Data()), nil
	case *LongValue:
		return int(v.Data()), nil
	default:
		return 0, fmt.Errorf("cannot convert %s to integer offset", a.Type())
	}
}

// Wrap creates an Allocable value that views the given byte slice.
// The returned value does not copy data — it reads and writes through the slice directly.
// The caller must ensure the slice remains valid for the lifetime of the value.
func Wrap(tag TypeTag, view []byte) (Allocable, error) {
	switch tag {
	case TagByte:
		return NewByte(view), nil
	case TagShort:
		return NewShort(view), nil
	case TagInteger:
		return NewInteger(view), nil
	case TagLong:
		return NewLong(view), nil
	case TagFloat:
		return NewFloat(view), nil
	case TagDecimal:
		return NewDecimal(view), nil
	case TagBoolean:
		return NewBoolean(view), nil
	case TagChar:
		return NewChar(view), nil
	default:
		return nil, fmt.Errorf("unknown type tag: %d", tag)
	}
}
