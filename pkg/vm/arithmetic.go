package vm

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/mwantia/vega/pkg/slot"
)

// addSlots computes a + b. Both slots must have the same tag.
func addSlots(a, b slot.StackSlot) (slot.StackSlot, error) {
	if a.Tag != b.Tag {
		return slot.StackSlot{}, fmt.Errorf("type mismatch: cannot add tag %d and tag %d", a.Tag, b.Tag)
	}
	var s slot.StackSlot
	s.Tag = a.Tag
	switch a.Tag {
	case slot.TagByte:
		s.Data[0] = a.Data[0] + b.Data[0]
	case slot.TagShort:
		av := int16(binary.LittleEndian.Uint16(a.Data[:2]))
		bv := int16(binary.LittleEndian.Uint16(b.Data[:2]))
		binary.LittleEndian.PutUint16(s.Data[:2], uint16(av+bv))
	case slot.TagInteger:
		av := int32(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := int32(binary.LittleEndian.Uint32(b.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], uint32(av+bv))
	case slot.TagLong:
		av := int64(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := int64(binary.LittleEndian.Uint64(b.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], uint64(av+bv))
	case slot.TagFloat:
		av := math.Float32frombits(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := math.Float32frombits(binary.LittleEndian.Uint32(b.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], math.Float32bits(av+bv))
	case slot.TagDecimal:
		av := math.Float64frombits(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := math.Float64frombits(binary.LittleEndian.Uint64(b.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], math.Float64bits(av+bv))
	default:
		return slot.StackSlot{}, fmt.Errorf("arithmetic not supported for tag %d", a.Tag)
	}
	return s, nil
}

// subSlots computes a - b.
func subSlots(a, b slot.StackSlot) (slot.StackSlot, error) {
	if a.Tag != b.Tag {
		return slot.StackSlot{}, fmt.Errorf("type mismatch: cannot subtract tag %d and tag %d", a.Tag, b.Tag)
	}
	var s slot.StackSlot
	s.Tag = a.Tag
	switch a.Tag {
	case slot.TagByte:
		s.Data[0] = a.Data[0] - b.Data[0]
	case slot.TagShort:
		av := int16(binary.LittleEndian.Uint16(a.Data[:2]))
		bv := int16(binary.LittleEndian.Uint16(b.Data[:2]))
		binary.LittleEndian.PutUint16(s.Data[:2], uint16(av-bv))
	case slot.TagInteger:
		av := int32(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := int32(binary.LittleEndian.Uint32(b.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], uint32(av-bv))
	case slot.TagLong:
		av := int64(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := int64(binary.LittleEndian.Uint64(b.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], uint64(av-bv))
	case slot.TagFloat:
		av := math.Float32frombits(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := math.Float32frombits(binary.LittleEndian.Uint32(b.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], math.Float32bits(av-bv))
	case slot.TagDecimal:
		av := math.Float64frombits(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := math.Float64frombits(binary.LittleEndian.Uint64(b.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], math.Float64bits(av-bv))
	default:
		return slot.StackSlot{}, fmt.Errorf("arithmetic not supported for tag %d", a.Tag)
	}
	return s, nil
}

// mulSlots computes a * b.
func mulSlots(a, b slot.StackSlot) (slot.StackSlot, error) {
	if a.Tag != b.Tag {
		return slot.StackSlot{}, fmt.Errorf("type mismatch: cannot multiply tag %d and tag %d", a.Tag, b.Tag)
	}
	var s slot.StackSlot
	s.Tag = a.Tag
	switch a.Tag {
	case slot.TagByte:
		s.Data[0] = a.Data[0] * b.Data[0]
	case slot.TagShort:
		av := int16(binary.LittleEndian.Uint16(a.Data[:2]))
		bv := int16(binary.LittleEndian.Uint16(b.Data[:2]))
		binary.LittleEndian.PutUint16(s.Data[:2], uint16(av*bv))
	case slot.TagInteger:
		av := int32(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := int32(binary.LittleEndian.Uint32(b.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], uint32(av*bv))
	case slot.TagLong:
		av := int64(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := int64(binary.LittleEndian.Uint64(b.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], uint64(av*bv))
	case slot.TagFloat:
		av := math.Float32frombits(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := math.Float32frombits(binary.LittleEndian.Uint32(b.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], math.Float32bits(av*bv))
	case slot.TagDecimal:
		av := math.Float64frombits(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := math.Float64frombits(binary.LittleEndian.Uint64(b.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], math.Float64bits(av*bv))
	default:
		return slot.StackSlot{}, fmt.Errorf("arithmetic not supported for tag %d", a.Tag)
	}
	return s, nil
}

// divSlots computes a / b. Returns an error on integer divide-by-zero.
func divSlots(a, b slot.StackSlot) (slot.StackSlot, error) {
	if a.Tag != b.Tag {
		return slot.StackSlot{}, fmt.Errorf("type mismatch: cannot divide tag %d by tag %d", a.Tag, b.Tag)
	}
	var s slot.StackSlot
	s.Tag = a.Tag
	switch a.Tag {
	case slot.TagByte:
		if b.Data[0] == 0 {
			return slot.StackSlot{}, fmt.Errorf("division by zero")
		}
		s.Data[0] = a.Data[0] / b.Data[0]
	case slot.TagShort:
		bv := int16(binary.LittleEndian.Uint16(b.Data[:2]))
		if bv == 0 {
			return slot.StackSlot{}, fmt.Errorf("division by zero")
		}
		av := int16(binary.LittleEndian.Uint16(a.Data[:2]))
		binary.LittleEndian.PutUint16(s.Data[:2], uint16(av/bv))
	case slot.TagInteger:
		bv := int32(binary.LittleEndian.Uint32(b.Data[:4]))
		if bv == 0 {
			return slot.StackSlot{}, fmt.Errorf("division by zero")
		}
		av := int32(binary.LittleEndian.Uint32(a.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], uint32(av/bv))
	case slot.TagLong:
		bv := int64(binary.LittleEndian.Uint64(b.Data[:8]))
		if bv == 0 {
			return slot.StackSlot{}, fmt.Errorf("division by zero")
		}
		av := int64(binary.LittleEndian.Uint64(a.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], uint64(av/bv))
	case slot.TagFloat:
		av := math.Float32frombits(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := math.Float32frombits(binary.LittleEndian.Uint32(b.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], math.Float32bits(av/bv))
	case slot.TagDecimal:
		av := math.Float64frombits(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := math.Float64frombits(binary.LittleEndian.Uint64(b.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], math.Float64bits(av/bv))
	default:
		return slot.StackSlot{}, fmt.Errorf("arithmetic not supported for tag %d", a.Tag)
	}
	return s, nil
}

// modSlots computes a % b. Only valid for integer types.
func modSlots(a, b slot.StackSlot) (slot.StackSlot, error) {
	if a.Tag != b.Tag {
		return slot.StackSlot{}, fmt.Errorf("type mismatch: cannot mod tag %d by tag %d", a.Tag, b.Tag)
	}
	var s slot.StackSlot
	s.Tag = a.Tag
	switch a.Tag {
	case slot.TagByte:
		if b.Data[0] == 0 {
			return slot.StackSlot{}, fmt.Errorf("modulo by zero")
		}
		s.Data[0] = a.Data[0] % b.Data[0]
	case slot.TagShort:
		bv := int16(binary.LittleEndian.Uint16(b.Data[:2]))
		if bv == 0 {
			return slot.StackSlot{}, fmt.Errorf("modulo by zero")
		}
		av := int16(binary.LittleEndian.Uint16(a.Data[:2]))
		binary.LittleEndian.PutUint16(s.Data[:2], uint16(av%bv))
	case slot.TagInteger:
		bv := int32(binary.LittleEndian.Uint32(b.Data[:4]))
		if bv == 0 {
			return slot.StackSlot{}, fmt.Errorf("modulo by zero")
		}
		av := int32(binary.LittleEndian.Uint32(a.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], uint32(av%bv))
	case slot.TagLong:
		bv := int64(binary.LittleEndian.Uint64(b.Data[:8]))
		if bv == 0 {
			return slot.StackSlot{}, fmt.Errorf("modulo by zero")
		}
		av := int64(binary.LittleEndian.Uint64(a.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], uint64(av%bv))
	default:
		return slot.StackSlot{}, fmt.Errorf("modulo not supported for tag %d", a.Tag)
	}
	return s, nil
}

// negSlot computes -a (unary negation).
func negSlot(a slot.StackSlot) (slot.StackSlot, error) {
	var s slot.StackSlot
	s.Tag = a.Tag
	switch a.Tag {
	case slot.TagShort:
		v := int16(binary.LittleEndian.Uint16(a.Data[:2]))
		binary.LittleEndian.PutUint16(s.Data[:2], uint16(-v))
	case slot.TagInteger:
		v := int32(binary.LittleEndian.Uint32(a.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], uint32(-v))
	case slot.TagLong:
		v := int64(binary.LittleEndian.Uint64(a.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], uint64(-v))
	case slot.TagFloat:
		v := math.Float32frombits(binary.LittleEndian.Uint32(a.Data[:4]))
		binary.LittleEndian.PutUint32(s.Data[:4], math.Float32bits(-v))
	case slot.TagDecimal:
		v := math.Float64frombits(binary.LittleEndian.Uint64(a.Data[:8]))
		binary.LittleEndian.PutUint64(s.Data[:8], math.Float64bits(-v))
	default:
		return slot.StackSlot{}, fmt.Errorf("negation not supported for tag %d", a.Tag)
	}
	return s, nil
}

// cmpSlots compares two same-typed slot.StackSlots and returns -1, 0, or +1.
func cmpSlots(a, b slot.StackSlot) (int, error) {
	if a.Tag != b.Tag {
		return 0, fmt.Errorf("type mismatch: cannot compare tag %d and tag %d", a.Tag, b.Tag)
	}
	switch a.Tag {
	case slot.TagByte:
		if a.Data[0] < b.Data[0] {
			return -1, nil
		}
		if a.Data[0] > b.Data[0] {
			return 1, nil
		}
		return 0, nil
	case slot.TagShort:
		av := int16(binary.LittleEndian.Uint16(a.Data[:2]))
		bv := int16(binary.LittleEndian.Uint16(b.Data[:2]))
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case slot.TagInteger:
		av := int32(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := int32(binary.LittleEndian.Uint32(b.Data[:4]))
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case slot.TagLong:
		av := int64(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := int64(binary.LittleEndian.Uint64(b.Data[:8]))
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case slot.TagFloat:
		av := math.Float32frombits(binary.LittleEndian.Uint32(a.Data[:4]))
		bv := math.Float32frombits(binary.LittleEndian.Uint32(b.Data[:4]))
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case slot.TagDecimal:
		av := math.Float64frombits(binary.LittleEndian.Uint64(a.Data[:8]))
		bv := math.Float64frombits(binary.LittleEndian.Uint64(b.Data[:8]))
		if av < bv {
			return -1, nil
		}
		if av > bv {
			return 1, nil
		}
		return 0, nil
	case slot.TagBoolean:
		ab := a.Data[0] != 0
		bb := b.Data[0] != 0
		if ab == bb {
			return 0, nil
		}
		if !ab && bb {
			return -1, nil
		}
		return 1, nil
	default:
		return 0, fmt.Errorf("comparison not supported for tag %d", a.Tag)
	}
}
