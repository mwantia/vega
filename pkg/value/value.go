package value

// Value represents a low-level runtime value.
type Value interface {
	// Type returns the type name for this value.
	Type() string

	// String returns a presentable string representation of this value.
	String() string
}

// --- String value ---

// StringValue is a view into a []byte region that holds UTF-8 string data.
// The backing bytes may live in the VM allocator (for variables) or on the
// Go heap (for constants loaded from the constant pool). It implements both
// Allocable (so it can be stored/loaded from slots) and Slice (so it supports
// indexing, iteration, and sub-slicing).
type StringValue struct {
	view []byte
}

// NewString creates a StringValue whose backing bytes are a copy of s.
// Use this for constants and literals where no allocator slot owns the memory.
func NewString(s string) *StringValue {
	b := make([]byte, len(s))
	copy(b, s)
	return &StringValue{view: b}
}

// NewStringView creates a StringValue that directly references the given byte
// slice without copying. Use this when the slice is already owned by the caller
// (e.g. a sub-slice of the VM allocator).
func NewStringView(view []byte) *StringValue {
	return &StringValue{view: view}
}

func (v *StringValue) Type() string   { return "string" }
func (v *StringValue) String() string { return string(v.view) }

// Size returns 0, signalling variable length. The actual byte count is always
// available via len(View()) or from the SlotEntry.Size in the runtime.
func (v *StringValue) Size() byte { return 0 }

func (v *StringValue) View() []byte { return v.view }

var _ Allocable = (*StringValue)(nil)

// --- Nil ---

// NilValue is a singleton sentinel. Not allocable, not a slice.
type NilValue struct{}

func (v *NilValue) Type() string   { return "nil" }
func (v *NilValue) String() string { return "nil" }

var _ Value = (*NilValue)(nil)

// Nil is the package-level nil singleton.
var Nil = &NilValue{}
