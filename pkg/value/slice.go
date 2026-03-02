package value

import (
	"bytes"
	"fmt"
)

// Slice is a bounded, contiguous region of bytes backed by the VM allocator
// (or the Go heap for constants). It is the runtime representation of all slice
// types: string<N>, byte<N>, and any future parameterised slice types.
//
// Memory layout: exactly Capacity bytes in the allocator. Content ends at the
// first '\0' byte OR at the capacity boundary, whichever comes first. All bytes
// beyond the content are zero-padded.
//
// Implements Allocatable (stored in slots) and Slice (indexed, iterable, sub-sliceable).
type Slice struct {
	view     []byte // full capacity region; content ends at first '\0' or len(view)
	allocOff int    // byte offset into the VM allocator buffer; -1 if not allocator-backed
}

// NewSlice creates a Slice that owns a copy of data.
// Use this for constants and literals where no allocator slot owns the memory.
func NewSlice(data []byte) *Slice {
	b := make([]byte, len(data))
	copy(b, data)

	return &Slice{
		view:     b,
		allocOff: -1,
	}
}

// NewSliceView creates a Slice that directly references the given byte
// slice without copying. Use this when the slice is already owned by the caller
// (e.g. a sub-region of the VM allocator).
func NewSliceView(view []byte) *Slice {
	return &Slice{
		view:     view,
		allocOff: -1,
	}
}

// NewSliceAllocView creates a Slice backed by a region of the VM allocator
// at the given byte offset. The allocOff is propagated through SubSlice so
// that OpVarSTORE can detect view assignments and avoid copying.
func NewSliceAllocView(view []byte, offset int) *Slice {
	return &Slice{
		view:     view,
		allocOff: offset,
	}
}

func (v *Slice) Type() string { return "slice" }

// String returns the effective content as a Go string (bytes up to first '\0').
func (v *Slice) String() string {
	n := bytes.IndexByte(v.view, 0)
	if n < 0 {
		return string(v.view)
	}
	return string(v.view[:n])
}

// Size returns 0 — capacity is always external (stored in SlotEntry.Capacity).
func (v *Slice) Size() byte { return 0 }

// View returns the full capacity backing slice. For allocator-backed values this
// is a direct sub-slice of the allocator buffer — not a copy.
func (v *Slice) View() []byte { return v.view }

// AllocOffset returns the byte offset of this slice within the VM allocator
// buffer, or -1 if this slice is not backed by the allocator (e.g. a constant
// or a heap-allocated result from methods like upper() or replace()).
func (v *Slice) AllocOffset() int { return v.allocOff }

var _ Allocatable = (*Slice)(nil)

// effectiveLen returns the number of usable bytes: up to the first '\0' or the
// full capacity if no '\0' is present.
func (v *Slice) effectiveLen() int {
	n := bytes.IndexByte(v.view, 0)
	if n < 0 {
		return len(v.view)
	}
	return n
}

// Length returns the effective byte count (content length, not capacity).
func (v *Slice) Length() int {
	return v.effectiveLen()
}

// Capacity returns the total allocated byte capacity of the slice.
func (v *Slice) Capacity() int {
	return len(v.view)
}

// Index returns the byte at the given position as a *Byte.
func (v *Slice) Index(key Value) (Value, error) {
	idx, err := indexFromValue(key)
	if err != nil {
		return nil, fmt.Errorf("slice index: %w", err)
	}
	n := v.effectiveLen()
	if idx < 0 || idx >= n {
		return nil, fmt.Errorf("slice index %d out of range [0, %d)", idx, n)
	}
	return NewByte([]byte{v.view[idx]}), nil
}

// SetIndex is rejected — slices are immutable via direct indexing for now.
func (v *Slice) SetIndex(_ Value, _ Value) error {
	return fmt.Errorf("slice elements are not directly assignable")
}

// Iterator returns an iterator that yields each byte as a *Byte.
func (v *Slice) Iterator() Iterator {
	return &sliceIterator{
		data: v.view[:v.effectiveLen()],
	}
}

var _ Iterable = (*Slice)(nil)

// SubSlice returns a sub-slice view covering bytes [from, from+length).
// The returned *Slice shares the same backing bytes — no copy.
// If the receiver is allocator-backed, the allocator offset is propagated.
func (v *Slice) SubSlice(from, length int) (*Slice, error) {
	n := v.effectiveLen()
	if from < 0 || length < 0 || from+length > n {
		return nil, fmt.Errorf("slice [%d:%d] out of range (length %d)", from, from+length, n)
	}
	newOff := -1
	if v.allocOff >= 0 {
		newOff = v.allocOff + from
	}
	return &Slice{
		view:     v.view[from : from+length],
		allocOff: newOff,
	}, nil
}

// indexFromValue extracts an integer index from a Value.
func indexFromValue(key Value) (int, error) {
	switch k := key.(type) {
	case *Byte:
		return int(k.Data()), nil
	case *Short:
		return int(k.Data()), nil
	case *Integer:
		return int(k.Data()), nil
	case *Long:
		return int(k.Data()), nil
	default:
		return 0, fmt.Errorf("index must be an integer, got %s", key.Type())
	}
}
