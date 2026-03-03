package slot

// TypeTag identifies the binary encoding format for an allocable value.
type TypeTag byte

const (
	TagVoid    TypeTag = 0
	TagShort   TypeTag = 1 // int16,   2 bytes
	TagInteger TypeTag = 2 // int32,   4 bytes
	TagLong    TypeTag = 3 // int64,   8 bytes
	TagFloat   TypeTag = 4 // float32, 4 bytes
	TagDecimal TypeTag = 5 // float64, 8 bytes
	TagBoolean TypeTag = 6 // bool,    1 byte
	TagByte    TypeTag = 7 // uint8,   1 byte
	TagSlice   TypeTag = 8 // []byte,  bounded capacity (SizeForTag returns 0; capacity is always external)

	// TagAny is a sentinel used in ParameterLayoutDescriptor.Tag to indicate
	// that a parameter accepts a value of any type. It is never used for
	// allocation or bitmask operations.
	TagAny TypeTag = 0xFF
)

var tagByName = map[string]TypeTag{
	"short":   TagShort,
	"int16":   TagShort,
	"int":     TagInteger,
	"int32":   TagInteger,
	"long":    TagLong,
	"int64":   TagLong,
	"float":   TagFloat,
	"decimal": TagDecimal,
	"bool":    TagBoolean,
	"byte":    TagByte,
	"string":  TagSlice,
	"any":     TagAny,
	"void":    TagVoid,
}

// TagForName resolves a type name string to its TypeTag.
func TagForName(name string) (TypeTag, bool) {
	if tag, ok := tagByName[name]; ok {
		return tag, true
	}
	return 0, false
}

// NameForTag
func NameForTag(tag TypeTag) (string, bool) {
	switch tag {
	case TagVoid:
		return "void", true
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
	case TagSlice:
		return "slice", true
	case TagAny:
		return "any", true
	default:
		return "", false
	}
}

// MaskForTag returns a bitmask with the bit for the given tag set.
// Tags 1–8 map to bits 0–7. TagAny returns 0xFF (all concrete types).
func MaskForTag(tag TypeTag) byte {
	if tag == TagAny {
		return 0xFF
	}
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
// Returns 0 for TagSlice — capacity is always provided externally.
func SizeForTag(tag TypeTag) int {
	switch tag {
	case TagVoid:
		return 0
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
	case TagSlice:
		return 0 // capacity is always provided externally; SizeForTag is not meaningful for slices
	case TagAny:
		return 0
	default:
		return 0
	}
}
