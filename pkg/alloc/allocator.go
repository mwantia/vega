package alloc

import "fmt"

// FreeBlock represents a contiguous free region in the byte buffer.
type FreeBlock struct {
	Offset int
	Size   int
}

// Allocator manages a []byte backing store with a free list for allocation and deallocation.
type Allocator struct {
	buffer   []byte
	freeList []FreeBlock // sorted by offset
}

// NewAllocator creates an Allocator with the given byte capacity.
// The entire buffer starts as one free block.
func NewAllocator(capacity int) *Allocator {
	return &Allocator{
		buffer: make([]byte, capacity),
		freeList: []FreeBlock{
			{
				Offset: 0,
				Size:   capacity,
			},
		},
	}
}

// Alloc finds the first free block that can fit size bytes (first-fit),
// splits if larger, and returns the offset into the buffer.
func (a *Allocator) Alloc(size int) (int, error) {
	for i, block := range a.freeList {
		if block.Size >= size {
			offset := block.Offset
			if block.Size == size {
				// Exact fit — remove block
				a.freeList = append(a.freeList[:i], a.freeList[i+1:]...)
			} else {
				// Split — shrink block
				a.freeList[i] = FreeBlock{
					Offset: block.Offset + size,
					Size:   block.Size - size,
				}
			}
			return offset, nil
		}
	}
	return 0, fmt.Errorf("out of memory: need %d bytes, have %d free", size, a.FreeSpace())
}

// Free returns the region at [offset, offset+size) back to the free list.
// The memory is zeroed and coalesced with adjacent free blocks.
func (a *Allocator) Free(offset, size int) {
	// Zero the freed memory
	for i := offset; i < offset+size; i++ {
		a.buffer[i] = 0
	}

	newBlock := FreeBlock{
		Offset: offset,
		Size:   size,
	}

	// Find insertion point (sorted by offset)
	insertIdx := 0
	for insertIdx < len(a.freeList) && a.freeList[insertIdx].Offset < offset {
		insertIdx++
	}

	// Insert
	a.freeList = append(a.freeList, FreeBlock{})
	copy(a.freeList[insertIdx+1:], a.freeList[insertIdx:])
	a.freeList[insertIdx] = newBlock

	// Coalesce with right neighbor
	if insertIdx+1 < len(a.freeList) {
		right := a.freeList[insertIdx+1]
		if a.freeList[insertIdx].Offset+a.freeList[insertIdx].Size == right.Offset {
			a.freeList[insertIdx].Size += right.Size
			a.freeList = append(a.freeList[:insertIdx+1], a.freeList[insertIdx+2:]...)
		}
	}

	// Coalesce with left neighbor
	if insertIdx > 0 {
		left := a.freeList[insertIdx-1]
		if left.Offset+left.Size == a.freeList[insertIdx].Offset {
			a.freeList[insertIdx-1].Size += a.freeList[insertIdx].Size
			a.freeList = append(a.freeList[:insertIdx], a.freeList[insertIdx+1:]...)
		}
	}
}

// Slice returns a writable view of the buffer at [offset, offset+size).
func (a *Allocator) Slice(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Write copies data into the buffer at the given offset.
func (a *Allocator) Write(offset int, data []byte) {
	copy(a.buffer[offset:], data)
}

// Read returns a slice view of the buffer at [offset, offset+size).
func (a *Allocator) Read(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Capacity returns the total size of the backing buffer.
func (a *Allocator) Capacity() int {
	return len(a.buffer)
}

// FreeSpace returns the total number of free bytes across all free blocks.
func (a *Allocator) FreeSpace() int {
	total := 0
	for _, block := range a.freeList {
		total += block.Size
	}
	return total
}
