package slot

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
)

// TagStencilRef is an internal StackSlot tag for stencil (struct/tuple) allocator refs.
// It encodes SliceRef{Offset, Size} in Data — no heap copy on the fast path.
const TagStencilRef = TypeTag(0xFE)

// SliceRef encodes a region of the VM allocator buffer as a pair of uint32 values
// packed into the StackSlot.Data field.
type SliceRef struct {
	Offset uint32
	Length uint32
}

// StackSlot is the unit stored on the expression stack.
//
// Scalars (byte, bool, short, int, long, float, decimal):
//
//	Tag=<scalar tag>  Data[0:SizeForTag(Tag)] = value bytes  HeapVal=nil
//
// Allocator-backed slices (from OpVarLOAD, OpFieldLOAD):
//
//	Tag=TagSlice  Data=SliceRef{Offset, Length}  HeapVal=nil
//
// Heap-backed slices (from OpLoadCONST, OpBuildSTRING, native returns):
//
//	Tag=TagSlice  HeapVal=[]byte{...}
//
// Stencil refs (from OpVarLoadRaw — no heap copy):
//
//	Tag=TagStencilRef  Data=SliceRef{AllocOffset, Size}  HeapVal=nil
//
// Nil/Void:
//
//	Tag=TagVoid  everything else zero
type StackSlot struct {
	Tag     TypeTag
	Data    [8]byte
	HeapVal []byte // non-nil only for heap-backed slices
}

// VoidSlot is a reusable TagVoid slot (nil/void sentinel).
var VoidSlot = StackSlot{Tag: TagVoid}

// PutSliceRef encodes a SliceRef into Data[0:8] using little-endian layout.
func (s *StackSlot) PutSliceRef(ref SliceRef) {
	binary.LittleEndian.PutUint32(s.Data[0:4], ref.Offset)
	binary.LittleEndian.PutUint32(s.Data[4:8], ref.Length)
}

// GetSliceRef decodes a SliceRef from Data[0:8].
func (s StackSlot) GetSliceRef() SliceRef {
	return SliceRef{
		Offset: binary.LittleEndian.Uint32(s.Data[0:4]),
		Length: binary.LittleEndian.Uint32(s.Data[4:8]),
	}
}

// Bytes returns the HeapVal backing bytes for heap-backed TagSlice slots.
func (s StackSlot) Bytes() []byte {
	return s.HeapVal
}

// AsString returns the string content of a heap-backed TagSlice slot,
// truncated at the first null byte.
func (s StackSlot) AsString() string {
	if s.HeapVal == nil {
		return ""
	}
	n := bytes.IndexByte(s.HeapVal, 0)
	if n >= 0 {
		return string(s.HeapVal[:n])
	}
	return string(s.HeapVal)
}

// AsBool returns the boolean value of a TagBoolean slot.
func (s StackSlot) AsBool() bool {
	return s.Data[0] != 0
}

// AsInt extracts a signed integer from a scalar slot (TagByte, TagShort, TagInteger, TagLong).
func (s StackSlot) AsInt() (int, error) {
	switch s.Tag {
	case TagByte:
		return int(s.Data[0]), nil
	case TagShort:
		return int(int16(binary.LittleEndian.Uint16(s.Data[:2]))), nil
	case TagInteger:
		return int(int32(binary.LittleEndian.Uint32(s.Data[:4]))), nil
	case TagLong:
		return int(int64(binary.LittleEndian.Uint64(s.Data[:8]))), nil
	default:
		return 0, fmt.Errorf("cannot convert tag %d to integer offset", s.Tag)
	}
}

// Format returns a human-readable string representation of the slot value.
// Used by print() and string interpolation for all types.
func (s StackSlot) Format() string {
	switch s.Tag {
	case TagVoid:
		return "nil"
	case TagBoolean:
		if s.Data[0] != 0 {
			return "true"
		}
		return "false"
	case TagByte:
		return strconv.FormatUint(uint64(s.Data[0]), 10)
	case TagShort:
		return strconv.FormatInt(int64(int16(binary.LittleEndian.Uint16(s.Data[:2]))), 10)
	case TagInteger:
		return strconv.FormatInt(int64(int32(binary.LittleEndian.Uint32(s.Data[:4]))), 10)
	case TagLong:
		return strconv.FormatInt(int64(binary.LittleEndian.Uint64(s.Data[:8])), 10)
	case TagFloat:
		f := math.Float32frombits(binary.LittleEndian.Uint32(s.Data[:4]))
		return strconv.FormatFloat(float64(f), 'g', -1, 32)
	case TagDecimal:
		f := math.Float64frombits(binary.LittleEndian.Uint64(s.Data[:8]))
		return strconv.FormatFloat(f, 'g', -1, 64)
	case TagSlice:
		return s.AsString()
	case TagStencilRef:
		ref := s.GetSliceRef()
		return fmt.Sprintf("<struct %d bytes>", ref.Length)
	default:
		return fmt.Sprintf("<tag %d>", s.Tag)
	}
}
