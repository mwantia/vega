// Package value defines the runtime value types for Vega.
package value

// Type constants for value types.
const (
	TypeString = "string"
	TypeInt    = "int"
	TypeFloat  = "float"
	TypeBool   = "bool"
	TypeNil    = "nil"
	TypeArray  = "array"
	TypeMap    = "map"
	TypeStream = "stream"
)

// Value represents a runtime value in Vega.
type Value interface {
	// Type returns the type name of this value.
	Type() string

	// String returns a string representation of this value.
	String() string

	// Boolean returns the truthy/falsy evaluation of this value.
	Boolean() bool

	// Equal returns true if this value equals another value.
	Equal(other Value) bool
}

// Comparable values can be compared with < > <= >=
type Comparable interface {
	Value
	Compare(other Value) (int, bool)
}

// Numeric values support arithmetic operations
type Numeric interface {
	Value
	Add(other Value) (Value, error)
	Sub(other Value) (Value, error)
	Mul(other Value) (Value, error)
	Div(other Value) (Value, error)
	Mod(other Value) (Value, error)
	Neg() Value
}

// Indexable values can be accessed with []
type Indexable interface {
	Value
	Index(key Value) (Value, error)
	SetIndex(key Value, val Value) error
}

// Iterable values can be used in for loops
type Iterable interface {
	Value
	Iterator() Iterator
}

// Iterator allows iterating over iterable values.
type Iterator interface {
	// Next advances the iterator and returns true if there's another value.
	Next() bool

	// Value returns the current value.
	Value() Value
}
