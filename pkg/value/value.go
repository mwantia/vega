package value

// Value represents a low-level runtime value.
type Value interface {
	// Type returns the type name for this value.
	Type() string

	// String returns a presentable string representation of this value.
	String() string
}

// --- String value ---

// --- Nil ---

// NilValue is a singleton sentinel. Not allocable, not a slice.
type NilValue struct{}

func (v *NilValue) Type() string   { return "nil" }
func (v *NilValue) String() string { return "nil" }

var _ Value = (*NilValue)(nil)

// Nil is the package-level nil singleton.
var Nil = &NilValue{}
