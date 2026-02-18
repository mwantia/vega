package value

// Slice
type Slice interface {
	Indexable
	Iterable

	// Length
	Length() int

	// Slice returns a new slice at a specified index for a specified length
	Slice(int, int) (Slice, error)

	// Alloc copies the content of this slice into an allocated value.
	Alloc() (Allocable, error)
}
