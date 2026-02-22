package value

import (
	"fmt"
	"unicode/utf8"
)

// Length returns the number of Unicode code points (runes) in the string.
func (v *StringValue) Length() int {
	return utf8.RuneCount(v.view)
}

// Index returns the rune at the given code-point index as a *CharValue.
// Negative indices and out-of-range accesses are rejected.
func (v *StringValue) Index(key Value) (Value, error) {
	idx, err := indexFromValue(key)
	if err != nil {
		return nil, fmt.Errorf("string index: %w", err)
	}

	n := utf8.RuneCount(v.view)
	if idx < 0 || idx >= n {
		return nil, fmt.Errorf("string index %d out of range [0, %d)", idx, n)
	}

	// Decode the idx-th rune.
	b := v.view
	for i := 0; i < idx; i++ {
		_, sz := utf8.DecodeRune(b)
		b = b[sz:]
	}
	r, _ := utf8.DecodeRune(b)
	return NewChar(r), nil
}

// SetIndex is rejected — strings are immutable after allocation.
func (v *StringValue) SetIndex(_ Value, _ Value) error {
	return fmt.Errorf("strings are immutable")
}

// Iterator returns an iterator that yields each rune as a *CharValue.
func (v *StringValue) Iterator() Iterator {
	return &stringIterator{data: v.view}
}

// Slice returns a sub-string view covering runes [from, from+length).
// The returned *StringValue shares the same backing bytes — no copy.
func (v *StringValue) Slice(from, length int) (Slice, error) {
	total := utf8.RuneCount(v.view)
	if from < 0 || length < 0 || from+length > total {
		return nil, fmt.Errorf("string slice [%d:%d] out of range (length %d)", from, from+length, total)
	}

	// Advance to byte offset of 'from'.
	b := v.view
	for i := 0; i < from; i++ {
		_, sz := utf8.DecodeRune(b)
		b = b[sz:]
	}
	// Measure byte length of 'length' runes.
	end := b
	for i := 0; i < length; i++ {
		_, sz := utf8.DecodeRune(end)
		end = end[sz:]
	}
	return NewStringView(b[:len(b)-len(end)]), nil
}

// Alloc returns a new *StringValue that owns a copy of the backing bytes.
// This is used when the caller needs a string that outlives the current view
// (e.g. for transport through the expression stack between call frames).
func (v *StringValue) Alloc() (Allocable, error) {
	cp := make([]byte, len(v.view))
	copy(cp, v.view)
	return NewStringView(cp), nil
}

var _ Slice = (*StringValue)(nil)

// --- iterator ---

type stringIterator struct {
	data []byte
	cur  rune
}

func (it *stringIterator) Next() bool {
	if len(it.data) == 0 {
		return false
	}
	r, sz := utf8.DecodeRune(it.data)
	it.cur = r
	it.data = it.data[sz:]
	return true
}

func (it *stringIterator) Value() Value {
	return NewChar(it.cur)
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
