package value

import (
	"bytes"
	"fmt"
)

// SliceValue is a bounded, contiguous region of bytes backed by the VM allocator
// (or the Go heap for constants). It is the runtime representation of all slice
// types: string<N>, byte<N>, and any future parameterised slice types.
//
// Memory layout: exactly Capacity bytes in the allocator. Content ends at the
// first '\0' byte OR at the capacity boundary, whichever comes first. All bytes
// beyond the content are zero-padded.
//
// Implements Allocable (stored in slots) and Slice (indexed, iterable, sub-sliceable).
type SliceValue struct {
	view []byte // full capacity region; content ends at first '\0' or len(view)
}

// NewSlice creates a SliceValue that owns a copy of data.
// Use this for constants and literals where no allocator slot owns the memory.
func NewSlice(data []byte) *SliceValue {
	b := make([]byte, len(data))
	copy(b, data)
	return &SliceValue{view: b}
}

// NewSliceView creates a SliceValue that directly references the given byte
// slice without copying. Use this when the slice is already owned by the caller
// (e.g. a sub-region of the VM allocator).
func NewSliceView(view []byte) *SliceValue {
	return &SliceValue{view: view}
}

func (v *SliceValue) Type() string { return "slice" }

// String returns the effective content as a Go string (bytes up to first '\0').
func (v *SliceValue) String() string {
	n := bytes.IndexByte(v.view, 0)
	if n < 0 {
		return string(v.view)
	}
	return string(v.view[:n])
}

// Size returns 0 — capacity is always external (stored in SlotEntry.Capacity).
func (v *SliceValue) Size() byte { return 0 }

// View returns the full capacity backing slice. For allocator-backed values this
// is a direct sub-slice of the allocator buffer — not a copy.
func (v *SliceValue) View() []byte { return v.view }

var _ Allocable = (*SliceValue)(nil)

// effectiveLen returns the number of usable bytes: up to the first '\0' or the
// full capacity if no '\0' is present.
func (v *SliceValue) effectiveLen() int {
	n := bytes.IndexByte(v.view, 0)
	if n < 0 {
		return len(v.view)
	}
	return n
}

// Length returns the effective byte count (content length, not capacity).
func (v *SliceValue) Length() int {
	return v.effectiveLen()
}

// Index returns the byte at the given position as a *ByteValue.
func (v *SliceValue) Index(key Value) (Value, error) {
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
func (v *SliceValue) SetIndex(_ Value, _ Value) error {
	return fmt.Errorf("slice elements are not directly assignable")
}

// Iterator returns an iterator that yields each byte as a *ByteValue.
func (v *SliceValue) Iterator() Iterator {
	return &sliceIterator{data: v.view[:v.effectiveLen()]}
}

// Slice returns a sub-slice view covering bytes [from, from+length).
// The returned *SliceValue shares the same backing bytes — no copy.
func (v *SliceValue) Slice(from, length int) (Slice, error) {
	n := v.effectiveLen()
	if from < 0 || length < 0 || from+length > n {
		return nil, fmt.Errorf("slice [%d:%d] out of range (length %d)", from, from+length, n)
	}
	return NewSliceView(v.view[from : from+length]), nil
}

// Alloc returns a new *SliceValue that owns a copy of the effective content.
// Used when a slice value must outlive the current allocator region.
func (v *SliceValue) Alloc() (Allocable, error) {
	n := v.effectiveLen()
	cp := make([]byte, n)
	copy(cp, v.view[:n])
	return NewSliceView(cp), nil
}

var _ Slice = (*SliceValue)(nil)

// --- iterator ---

type sliceIterator struct {
	data []byte
	pos  int
}

func (it *sliceIterator) Next() bool {
	return it.pos < len(it.data)
}

func (it *sliceIterator) Value() Value {
	b := it.data[it.pos]
	it.pos++
	return NewByte([]byte{b})
}

// --- helpers ---

// indexFromValue extracts an integer index from a Value.
func indexFromValue(key Value) (int, error) {
	switch k := key.(type) {
	case *ByteValue:
		return int(k.Data()), nil
	case *ShortValue:
		return int(k.Data()), nil
	case *IntegerValue:
		return int(k.Data()), nil
	case *LongValue:
		return int(k.Data()), nil
	default:
		return 0, fmt.Errorf("index must be an integer, got %s", key.Type())
	}
}
