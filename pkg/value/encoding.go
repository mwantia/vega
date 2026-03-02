package value

import (
	"encoding/binary"
	"fmt"
)

func EncodeInteger(n int) []byte {
	data := make([]byte, 4)
	binary.LittleEndian.PutUint32(data, uint32(n))
	return data
}

func EncodeBoolean(b bool) []byte {
	if b {
		return []byte{1}
	}
	return []byte{0}
}

func Encode(a any) ([]byte, error) {
	switch v := a.(type) {
	case bool:
		if v {
			return []byte{1}, nil
		}
		return []byte{0}, nil
	}
	return nil, fmt.Errorf("failed to encode: unknown valuetype")
}

// ToInt extracts an integer offset from an Allocatable value.
// Supports byte, short, int, and long types.
func ToInt(a Allocatable) (int, error) {
	switch v := a.(type) {
	case *Byte:
		return int(v.Data()), nil
	case *Short:
		return int(v.Data()), nil
	case *Integer:
		return int(v.Data()), nil
	case *Long:
		return int(v.Data()), nil
	default:
		return 0, fmt.Errorf("cannot convert %s to integer offset", a.Type())
	}
}

// Wrap creates an Allocatable value that views the given byte slice.
// The returned value does not copy data — it reads and writes through the slice directly.
// The caller must ensure the slice remains valid for the lifetime of the value.
func Wrap(tag TypeTag, view []byte) (Allocatable, error) {
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
	case TagSlice:
		return NewSliceView(view), nil
	default:
		return nil, fmt.Errorf("unknown type tag: %d", tag)
	}
}
