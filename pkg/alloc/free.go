package alloc

import "fmt"

// FreeListAllocator manages a []byte backing store with a free list for allocation and deallocation.
// When capacity is 0 the buffer grows automatically; when capacity > 0 it is a hard limit.
type FreeListAllocator struct {
	buffer       []byte
	freedRegions []Region // sorted by offset
	capacity     int      // hard limit (0 = unlimited)
}

// NewFreeListAllocator creates a FreeListAllocator with the given byte capacity.
// If capacity is 0 the allocator starts at defaultInitialSize and grows as needed.
// The entire initial buffer starts as one free block.
func NewFreeListAllocator(capacity int) Allocator {
	return &FreeListAllocator{
		buffer: make([]byte, capacity),
		freedRegions: []Region{
			{
				Offset: 0,
				Size:   capacity,
			},
		},
		capacity: capacity,
	}
}

// Alloc finds the first free block that fits size bytes (first-fit), splits if larger,
// and returns the offset. If no block fits and capacity is 0 the buffer is grown first.
func (a *FreeListAllocator) Alloc(size int) (int, error) {
	if off, ok := a.TryAlloc(size); ok {
		return off, nil
	}
	if a.capacity != 0 {
		return 0, fmt.Errorf("out of memory: need %d bytes, have %d free (limit: %d)", size, a.FreeSpace(), a.capacity)
	}
	a.GrowSpace(size)
	if off, ok := a.TryAlloc(size); ok {
		return off, nil
	}
	return 0, fmt.Errorf("out of memory: growth failed for %d bytes", size)
}

func (a *FreeListAllocator) TryAlloc(size int) (int, bool) {
	for i, block := range a.freedRegions {
		if block.Size >= size {
			offset := block.Offset
			if block.Size == size {
				a.freedRegions = append(a.freedRegions[:i], a.freedRegions[i+1:]...)
			} else {
				a.freedRegions[i] = Region{
					Offset: block.Offset + size,
					Size:   block.Size - size,
				}
			}
			return offset, true
		}
	}
	return 0, false
}

// grow expands the buffer so that at least needed contiguous bytes are added.
// The new region is inserted into the free list and coalesced with its left neighbour
// if adjacent.
func (a *FreeListAllocator) GrowSpace(needed int) {
	oldSize := len(a.buffer)
	newSize := max(oldSize*2, oldSize+needed)
	newBuf := make([]byte, newSize)
	copy(newBuf, a.buffer)
	a.buffer = newBuf

	// Append the new free region (highest offset — always at the end of the list).
	newRegion := Region{
		Offset: oldSize,
		Size:   newSize - oldSize,
	}
	insertIdx := len(a.freedRegions)
	a.freedRegions = append(a.freedRegions, newRegion)

	// Coalesce with left neighbour if adjacent.
	if insertIdx > 0 {
		left := a.freedRegions[insertIdx-1]
		if left.Offset+left.Size == oldSize {
			a.freedRegions[insertIdx-1].Size += newSize - oldSize
			a.freedRegions = a.freedRegions[:insertIdx]
		}
	}
}

// Free returns the region at [offset, offset+size] back to the free list.
// The memory is zeroed and coalesced with adjacent free blocks.
func (a *FreeListAllocator) Free(offset, size int) {
	for i := offset; i < offset+size; i++ {
		a.buffer[i] = 0
	}

	newBlock := Region{
		Offset: offset,
		Size:   size,
	}

	insertIdx := 0
	for insertIdx < len(a.freedRegions) && a.freedRegions[insertIdx].Offset < offset {
		insertIdx++
	}

	a.freedRegions = append(a.freedRegions, Region{})
	copy(a.freedRegions[insertIdx+1:], a.freedRegions[insertIdx:])
	a.freedRegions[insertIdx] = newBlock

	// Coalesce with right neighbor.
	if insertIdx+1 < len(a.freedRegions) {
		right := a.freedRegions[insertIdx+1]
		if a.freedRegions[insertIdx].Offset+a.freedRegions[insertIdx].Size == right.Offset {
			a.freedRegions[insertIdx].Size += right.Size
			a.freedRegions = append(a.freedRegions[:insertIdx+1], a.freedRegions[insertIdx+2:]...)
		}
	}

	// Coalesce with left neighbor.
	if insertIdx > 0 {
		left := a.freedRegions[insertIdx-1]
		if left.Offset+left.Size == a.freedRegions[insertIdx].Offset {
			a.freedRegions[insertIdx-1].Size += a.freedRegions[insertIdx].Size
			a.freedRegions = append(a.freedRegions[:insertIdx], a.freedRegions[insertIdx+1:]...)
		}
	}
}

// Slice returns a writable view of the buffer at [offset, offset+size).
func (a *FreeListAllocator) Slice(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Write copies data into the buffer at the given offset.
func (a *FreeListAllocator) Write(offset int, data []byte) {
	copy(a.buffer[offset:], data)
}

// Read returns a slice view of the buffer at [offset, offset+size).
func (a *FreeListAllocator) Read(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Capacity returns the hard limit in bytes (0 = unlimited).
func (a *FreeListAllocator) Capacity() int {
	return a.capacity
}

// Size returns the current actual size of the backing buffer in bytes.
func (a *FreeListAllocator) Size() int {
	return len(a.buffer)
}

// FreeSpace returns the total number of free bytes across all free blocks.
func (a *FreeListAllocator) FreeSpace() int {
	total := 0
	for _, block := range a.freedRegions {
		total += block.Size
	}
	return total
}

// Snapshot returns a deep copy of the current allocator state.
func (a *FreeListAllocator) Snapshot() (Snapshot, error) {
	buf := make([]byte, len(a.buffer))
	copy(buf, a.buffer)
	regions := make([]Region, len(a.freedRegions))
	copy(regions, a.freedRegions)
	return Snapshot{
		Buffer:       buf,
		FreedRegions: regions,
		Capacity:     a.capacity,
	}, nil
}

// Restore overwrites the allocator's state with the given snapshot.
// The backing buffer is resized if the snapshot's buffer has a different length.
func (a *FreeListAllocator) Restore(snap Snapshot) error {
	if len(snap.Buffer) != len(a.buffer) {
		a.buffer = make([]byte, len(snap.Buffer))
	}
	copy(a.buffer, snap.Buffer)
	a.freedRegions = make([]Region, len(snap.FreedRegions))
	copy(a.freedRegions, snap.FreedRegions)
	a.capacity = snap.Capacity
	return nil
}
