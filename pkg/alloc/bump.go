package alloc

import "fmt"

// BumpAllocator is a linear allocator that advances a cursor through a buffer.
// Free is a no-op — memory is never reclaimed individually.
// When capacity is 0 the buffer grows automatically; when capacity > 0 it is a hard limit.
type BumpAllocator struct {
	buffer   []byte
	cursor   int
	capacity int // hard limit (0 = unlimited)
}

// NewBumpAllocator creates a BumpAllocator with the given byte capacity.
// If capacity is 0 the allocator starts at defaultInitialSize and grows as needed.
func NewBumpAllocator(capacity int) Allocator {
	return &BumpAllocator{
		buffer:   make([]byte, capacity),
		capacity: capacity,
	}
}

// Alloc advances the cursor by size and returns the previous cursor position.
// If the buffer would overflow and capacity is 0 the buffer is grown first.
func (a *BumpAllocator) Alloc(size int) (int, error) {
	if a.cursor+size <= len(a.buffer) {
		offset := a.cursor
		a.cursor += size
		return offset, nil
	}
	if a.capacity != 0 {
		return 0, fmt.Errorf("out of memory: need %d bytes, have %d free (limit: %d)", size, a.FreeSpace(), a.capacity)
	}
	a.GrowSpace(size)
	offset := a.cursor
	a.cursor += size
	return offset, nil
}

// grow expands the buffer so that at least needed bytes are available after the cursor.
func (a *BumpAllocator) GrowSpace(needed int) {
	oldSize := len(a.buffer)
	newSize := max(oldSize*2, a.cursor+needed)
	newBuf := make([]byte, newSize)
	copy(newBuf, a.buffer)
	a.buffer = newBuf
}

// Free is a no-op for BumpAllocator — memory is not reclaimed.
func (a *BumpAllocator) Free(offset, size int) {}

// Slice returns a writable view of the buffer at [offset, offset+size).
func (a *BumpAllocator) Slice(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Write copies data into the buffer at the given offset.
func (a *BumpAllocator) Write(offset int, data []byte) {
	copy(a.buffer[offset:], data)
}

// Read returns a slice view of the buffer at [offset, offset+size).
func (a *BumpAllocator) Read(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Capacity returns the hard limit in bytes (0 = unlimited).
func (a *BumpAllocator) Capacity() int {
	return a.capacity
}

// Size returns the current actual size of the backing buffer in bytes.
func (a *BumpAllocator) Size() int {
	return len(a.buffer)
}

// FreeSpace returns the number of bytes remaining after the cursor.
func (a *BumpAllocator) FreeSpace() int {
	return len(a.buffer) - a.cursor
}
