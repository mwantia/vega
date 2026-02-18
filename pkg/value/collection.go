package value

// Indexable values can be accessed with [<index>].
type Indexable interface {
	Value
	// Index
	Index(key Value) (Value, error)

	// SetIndex
	SetIndex(key Value, val Value) error
}

// Iterable values can be used in for loops.
type Iterable interface {
	Value
	// Iterator returns an iterator for iterating over all values.
	Iterator() Iterator
}

// Iterator allows iterating over iterable values.
type Iterator interface {
	// Next advances the iterator and returns true if there's another value.
	Next() bool

	// Value returns the current value.
	Value() Value
}
