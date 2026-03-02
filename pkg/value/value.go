package value

// Value represents a low-level runtime value.
type Value interface {
	// Type returns the type name for this value.
	Type() string

	// String returns a presentable string representation of this value.
	String() string
}

// Nil is a singleton sentinel. Not allocable, not a slice.
type Nil struct{}

func (v *Nil) Type() string {
	return "nil"
}

func (v *Nil) String() string {
	return "nil"
}

var _ Value = (*Nil)(nil)

// Nil is the package-level nil singleton.
var NilSingleton = &Nil{}
