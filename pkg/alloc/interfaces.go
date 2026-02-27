package alloc

// Allocator is the interface for all memory allocation strategies used by the VM.
type Allocator interface {
	Alloc(size int) (int, error)
	Free(offset, size int)
	Slice(offset, size int) []byte
	Write(offset int, data []byte)
	Read(offset, size int) []byte
	// Capacity returns the hard limit in bytes (0 = unlimited, grows dynamically).
	Capacity() int
	// Size returns the current actual size of the backing buffer in bytes.
	Size() int
	FreeSpace() int
}

// Snapshottable is implemented by allocators that support
// both taking and restoring point-in-time snapshots.
type Snapshottable interface {
	Snapshot() (Snapshot, error)
	Restore(Snapshot) error
}

// Snapshot is a point-in-time deep copy of the allocator state.
// Capacity holds the hard limit (0 = unlimited); the actual buffer
// size is implicit in len(Buffer).
type Snapshot struct {
	Buffer       []byte
	FreedRegions []Region
	Capacity     int
}

type Region struct {
	Offset int
	Size   int
}
