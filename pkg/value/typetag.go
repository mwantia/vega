package value

// TypeTag identifies the binary encoding format for an allocable value.
type TypeTag byte

const (
	TagShort   TypeTag = 1 // int16, 2 bytes
	TagInteger TypeTag = 2 // int32, 4 bytes
	TagLong    TypeTag = 3 // int64, 8 bytes
	TagFloat   TypeTag = 4 // float32, 4 bytes
	TagDecimal TypeTag = 5 // float64, 8 bytes
	TagBoolean TypeTag = 6 // bool, 1 byte
	TagByte    TypeTag = 7 // uint8, 1 byte
	TagChar    TypeTag = 8 // rune, 4 bytes
)

// TagFor returns the TypeTag for an Allocable value.
func TagFor(a Allocable) TypeTag {
	switch a.(type) {
	case *ShortValue:
		return TagShort
	case *IntegerValue:
		return TagInteger
	case *LongValue:
		return TagLong
	case *FloatValue:
		return TagFloat
	case *DecimalValue:
		return TagDecimal
	case *BooleanValue:
		return TagBoolean
	case *ByteValue:
		return TagByte
	case *CharValue:
		return TagChar
	default:
		return 0
	}
}

// TagForName resolves a type name string to its TypeTag.
func TagForName(name string) (TypeTag, bool) {
	switch name {
	case "short":
		return TagShort, true
	case "int":
		return TagInteger, true
	case "long":
		return TagLong, true
	case "float":
		return TagFloat, true
	case "decimal":
		return TagDecimal, true
	case "bool":
		return TagBoolean, true
	case "byte":
		return TagByte, true
	case "char":
		return TagChar, true
	default:
		return 0, false
	}
}

func NameForTag(tag TypeTag) (string, bool) {
	switch tag {
	case TagShort:
		return "short", true
	case TagInteger:
		return "int", true
	case TagLong:
		return "long", true
	case TagFloat:
		return "float", true
	case TagDecimal:
		return "decimal", true
	case TagBoolean:
		return "boolean", true
	case TagByte:
		return "byte", true
	case TagChar:
		return "char", true
	default:
		return "", false
	}
}

// MaskForTag returns a bitmask with the bit for the given tag set.
// Tags 1–8 map to bits 0–7.
func MaskForTag(tag TypeTag) byte {
	if tag < 1 || tag > 8 {
		return 0
	}
	return 1 << (tag - 1)
}

// TagInMask checks whether the given tag is present in the bitmask.
func TagInMask(tag TypeTag, mask byte) bool {
	return MaskForTag(tag)&mask != 0
}

// MaxSizeForMask returns the maximum byte size across all tags in the mask.
func MaxSizeForMask(mask byte) int {
	maxSize := 0
	for t := TypeTag(1); t <= 8; t++ {
		if mask&(1<<(t-1)) != 0 {
			if s := SizeForTag(t); s > maxSize {
				maxSize = s
			}
		}
	}
	return maxSize
}

// SizeForTag returns the byte size for the given TypeTag.
func SizeForTag(tag TypeTag) int {
	switch tag {
	case TagShort:
		return 2
	case TagInteger:
		return 4
	case TagLong:
		return 8
	case TagFloat:
		return 4
	case TagDecimal:
		return 8
	case TagBoolean:
		return 1
	case TagByte:
		return 1
	case TagChar:
		return 4
	default:
		return 0
	}
}
