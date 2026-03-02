package value

// Methodable
type Methodable interface {
	Value
	// Method calls an extended method for this value.
	Method(name string, args []Value) (Value, error)
}

// Memberable
type Memberable interface {
	Value
	// GetMember returns an extended member value for this value.
	GetMember(name string) (Value, error)

	// SetMember sets the value of an extended member for this value.
	SetMember(name string, val Value) (bool, error)
}

// Allocatable
type Allocatable interface {
	Value
	// Size returns the known size of the value type that needs to be allocated.
	Size() byte
	// View returns the backing byte slice. For values loaded from the alloc buffer,
	// this is a sub-slice of the allocator's memory — not a copy.
	View() []byte
}

// Comparable
type Comparable interface {
	Allocatable
	// Compare returns an integer comparing two comparable values.
	Compare(other Comparable) (int, error)
}

// Numeric
type Numeric interface {
	Allocatable
	// Add
	Add(other Numeric) (Numeric, error)

	// Sub
	Sub(other Numeric) (Numeric, error)

	// Mul
	Mul(other Numeric) (Numeric, error)

	// Div
	Div(other Numeric) (Numeric, error)

	// Mod
	Mod(other Numeric) (Numeric, error)

	// Neg
	Neg() (Numeric, error)
}
