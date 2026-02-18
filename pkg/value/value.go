package value

// Value represents a low-level runtime value.
type Value interface {
	// Type returns the type name for this value.
	Type() string

	// String returns a presentable string representation of this value.
	String() string
}

// --- Slice value ---

// StringSlice holds a reference to string data in static VM memory (constants table).
// Implements Value only for now; Slice interface will be added when indexing/iteration is needed.
type StringSlice struct {
	data string
}

func NewString(v string) *StringSlice {
	return &StringSlice{data: v}
}

func (v *StringSlice) Type() string   { return "string" }
func (v *StringSlice) String() string { return v.data }
func (v *StringSlice) Data() string   { return v.data }

var _ Value = (*StringSlice)(nil)

// --- Nil ---

// NilValue is a singleton sentinel. Not allocable, not a slice.
type NilValue struct{}

func (v *NilValue) Type() string   { return "nil" }
func (v *NilValue) String() string { return "nil" }

var _ Value = (*NilValue)(nil)

// Nil is the package-level nil singleton.
var Nil = &NilValue{}
