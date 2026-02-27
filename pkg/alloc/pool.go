package alloc

import "fmt"

// PoolAllocator manages fixed-size slots backed by a single byte buffer.
// Allocation is O(n) in the number of slots; free is O(1).
// When capacity is 0 the pool grows automatically; when capacity > 0 it is a hard limit.
type PoolAllocator struct {
	buffer   []byte
	slotSize int
	numSlots int
	used     []bool
	capacity int // hard limit in bytes (0 = unlimited)
}

// NewPoolAllocator creates a PoolAllocator with the given capacity and slot size.
// If capacity is 0 the pool starts at defaultInitialSlots slots and grows as needed.
// A positive capacity is rounded down to the nearest multiple of slotSize.
func NewPoolAllocator(capacity, slotSize int) Allocator {
	if slotSize <= 0 {
		slotSize = 1
	}
	var numSlots int
	if capacity > 0 {
		numSlots = capacity / slotSize
	} else {
		numSlots = 16
	}
	return &PoolAllocator{
		buffer:   make([]byte, numSlots*slotSize),
		slotSize: slotSize,
		numSlots: numSlots,
		used:     make([]bool, numSlots),
		capacity: capacity,
	}
}

// Alloc finds the first free slot and returns its byte offset.
// Returns an error if size exceeds slotSize or no slots are available and the limit is reached.
func (a *PoolAllocator) Alloc(size int) (int, error) {
	if size > a.slotSize {
		return 0, fmt.Errorf("pool alloc: requested %d bytes exceeds slot size %d", size, a.slotSize)
	}
	for i := 0; i < a.numSlots; i++ {
		if !a.used[i] {
			a.used[i] = true
			return i * a.slotSize, nil
		}
	}
	if a.capacity != 0 {
		return 0, fmt.Errorf("out of memory: no free slots (slot size=%d, slots=%d, limit=%d)", a.slotSize, a.numSlots, a.capacity)
	}
	firstNew := a.numSlots
	a.Growspace()
	a.used[firstNew] = true
	return firstNew * a.slotSize, nil
}

// grow doubles the number of slots.
func (a *PoolAllocator) Growspace() {
	oldSlots := a.numSlots
	newSlots := oldSlots * 2
	if newSlots == 0 {
		newSlots = 16
	}
	newBuf := make([]byte, newSlots*a.slotSize)
	copy(newBuf, a.buffer)
	newUsed := make([]bool, newSlots)
	copy(newUsed, a.used)
	a.buffer = newBuf
	a.used = newUsed
	a.numSlots = newSlots
}

// Free marks the slot containing offset as unused and zeroes its memory.
func (a *PoolAllocator) Free(offset, size int) {
	slot := offset / a.slotSize
	if slot < 0 || slot >= a.numSlots {
		return
	}
	start := slot * a.slotSize
	end := start + a.slotSize
	for i := start; i < end; i++ {
		a.buffer[i] = 0
	}
	a.used[slot] = false
}

// Slice returns a writable view of the buffer at [offset, offset+size).
func (a *PoolAllocator) Slice(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Write copies data into the buffer at the given offset.
func (a *PoolAllocator) Write(offset int, data []byte) {
	copy(a.buffer[offset:], data)
}

// Read returns a slice view of the buffer at [offset, offset+size).
func (a *PoolAllocator) Read(offset, size int) []byte {
	return a.buffer[offset : offset+size]
}

// Capacity returns the hard limit in bytes (0 = unlimited).
func (a *PoolAllocator) Capacity() int {
	return a.capacity
}

// Size returns the current actual size of the backing buffer in bytes.
func (a *PoolAllocator) Size() int {
	return len(a.buffer)
}

// FreeSpace returns the total free bytes across unused slots.
func (a *PoolAllocator) FreeSpace() int {
	free := 0
	for _, u := range a.used {
		if !u {
			free += a.slotSize
		}
	}
	return free
}
