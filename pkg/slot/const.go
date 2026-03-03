package slot

import (
	"encoding/binary"
	"math"
)

// NewBool creates a TagBoolean StackSlot.
func NewBool(v bool) StackSlot {
	var s StackSlot
	s.Tag = TagBoolean
	if v {
		s.Data[0] = 1
	}
	return s
}

// NewByte creates a TagByte StackSlot.
func NewByte(v byte) StackSlot {
	var s StackSlot
	s.Tag = TagByte
	s.Data[0] = v
	return s
}

// NewShort creates a TagShort StackSlot.
func NewShort(v int16) StackSlot {
	var s StackSlot
	s.Tag = TagShort
	binary.LittleEndian.PutUint16(s.Data[:2], uint16(v))
	return s
}

// NewInt creates a TagInteger StackSlot.
func NewInt(v int) StackSlot {
	var s StackSlot
	s.Tag = TagInteger
	binary.LittleEndian.PutUint32(s.Data[:4], uint32(int32(v)))
	return s
}

// NewLong creates a TagLong StackSlot.
func NewLong(v int64) StackSlot {
	var s StackSlot
	s.Tag = TagLong
	binary.LittleEndian.PutUint64(s.Data[:8], uint64(v))
	return s
}

// NewFloat creates a TagFloat StackSlot.
func NewFloat(v float32) StackSlot {
	var s StackSlot
	s.Tag = TagFloat
	binary.LittleEndian.PutUint32(s.Data[:4], math.Float32bits(v))
	return s
}

// NewDecimal creates a TagDecimal StackSlot.
func NewDecimal(v float64) StackSlot {
	var s StackSlot
	s.Tag = TagDecimal
	binary.LittleEndian.PutUint64(s.Data[:8], math.Float64bits(v))
	return s
}

// NewString creates a heap-backed TagSlice StackSlot from a string.
func NewString(s string) StackSlot {
	return StackSlot{Tag: TagSlice, HeapVal: []byte(s)}
}

// NewBytes creates a heap-backed TagSlice StackSlot from a byte slice (copied).
func NewBytes(b []byte) StackSlot {
	data := make([]byte, len(b))
	copy(data, b)
	return StackSlot{Tag: TagSlice, HeapVal: data}
}
