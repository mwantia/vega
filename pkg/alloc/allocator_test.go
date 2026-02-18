package alloc_test

import (
	"testing"

	"github.com/mwantia/vega/pkg/alloc"
)

func TestCapacity(t *testing.T) {
	a := alloc.NewAllocator(32)
	if a.Capacity() != 32 {
		t.Errorf("Capacity() = %d, want 32", a.Capacity())
	}

	b := alloc.NewAllocator(1024)
	if b.Capacity() != 1024 {
		t.Errorf("Capacity() = %d, want 1024", b.Capacity())
	}
}

func TestSequentialAlloc(t *testing.T) {
	a := alloc.NewAllocator(16)

	off1, err := a.Alloc(4)
	if err != nil {
		t.Fatalf("alloc 4: %v", err)
	}
	if off1 != 0 {
		t.Errorf("first alloc offset = %d, want 0", off1)
	}

	off2, err := a.Alloc(4)
	if err != nil {
		t.Fatalf("alloc 4: %v", err)
	}
	if off2 != 4 {
		t.Errorf("second alloc offset = %d, want 4", off2)
	}

	off3, err := a.Alloc(8)
	if err != nil {
		t.Fatalf("alloc 8: %v", err)
	}
	if off3 != 8 {
		t.Errorf("third alloc offset = %d, want 8", off3)
	}

	if a.FreeSpace() != 0 {
		t.Errorf("free space = %d, want 0", a.FreeSpace())
	}
}

func TestOutOfMemory(t *testing.T) {
	a := alloc.NewAllocator(8)

	_, err := a.Alloc(4)
	if err != nil {
		t.Fatalf("alloc 4: %v", err)
	}

	_, err = a.Alloc(8)
	if err == nil {
		t.Fatal("expected OOM error, got nil")
	}
}

func TestFreeAndReuse(t *testing.T) {
	a := alloc.NewAllocator(8)

	off1, _ := a.Alloc(4)
	off2, _ := a.Alloc(4)

	// Free first block
	a.Free(off1, 4)
	if a.FreeSpace() != 4 {
		t.Errorf("free space after free = %d, want 4", a.FreeSpace())
	}

	// Reuse freed space
	off3, err := a.Alloc(4)
	if err != nil {
		t.Fatalf("realloc: %v", err)
	}
	if off3 != off1 {
		t.Errorf("reused offset = %d, want %d", off3, off1)
	}
	_ = off2
}

func TestCoalesceRight(t *testing.T) {
	a := alloc.NewAllocator(16)

	off1, _ := a.Alloc(4) // [0..4)
	off2, _ := a.Alloc(4) // [4..8)
	_, _ = a.Alloc(4)     // [8..12)

	// Free first, then second — they should coalesce
	a.Free(off1, 4)
	a.Free(off2, 4)

	// Should be able to allocate 8 contiguous bytes at offset 0
	off, err := a.Alloc(8)
	if err != nil {
		t.Fatalf("alloc 8 after coalesce: %v", err)
	}
	if off != 0 {
		t.Errorf("coalesced offset = %d, want 0", off)
	}
}

func TestCoalesceLeft(t *testing.T) {
	a := alloc.NewAllocator(16)

	off1, _ := a.Alloc(4) // [0..4)
	off2, _ := a.Alloc(4) // [4..8)
	_, _ = a.Alloc(4)     // [8..12)

	// Free second, then first — they should coalesce
	a.Free(off2, 4)
	a.Free(off1, 4)

	// Should be able to allocate 8 contiguous bytes at offset 0
	off, err := a.Alloc(8)
	if err != nil {
		t.Fatalf("alloc 8 after coalesce: %v", err)
	}
	if off != 0 {
		t.Errorf("coalesced offset = %d, want 0", off)
	}
}

func TestCoalesceBoth(t *testing.T) {
	a := alloc.NewAllocator(12)

	off1, _ := a.Alloc(4) // [0..4)
	off2, _ := a.Alloc(4) // [4..8)
	off3, _ := a.Alloc(4) // [8..12)

	// Free first and third, then middle — all three should coalesce
	a.Free(off1, 4)
	a.Free(off3, 4)
	a.Free(off2, 4)

	if a.FreeSpace() != 12 {
		t.Errorf("free space = %d, want 12", a.FreeSpace())
	}

	// Should be able to allocate full 12 bytes
	off, err := a.Alloc(12)
	if err != nil {
		t.Fatalf("alloc 12 after full coalesce: %v", err)
	}
	if off != 0 {
		t.Errorf("coalesced offset = %d, want 0", off)
	}
}

func TestFirstFitWithHoles(t *testing.T) {
	a := alloc.NewAllocator(16)

	a.Alloc(4)            // [0..4)
	off2, _ := a.Alloc(4) // [4..8)
	a.Alloc(4)            // [8..12)
	a.Alloc(4)            // [12..16)

	// Free the middle block to create a hole
	a.Free(off2, 4)

	// Allocate 2 bytes — should fit in the hole at offset 4
	off, err := a.Alloc(2)
	if err != nil {
		t.Fatalf("alloc 2: %v", err)
	}
	if off != 4 {
		t.Errorf("first-fit offset = %d, want 4", off)
	}

	// Allocate 2 more bytes — should use remaining hole at offset 6
	off, err = a.Alloc(2)
	if err != nil {
		t.Fatalf("alloc 2: %v", err)
	}
	if off != 6 {
		t.Errorf("second first-fit offset = %d, want 6", off)
	}
}

func TestWriteAndRead(t *testing.T) {
	a := alloc.NewAllocator(16)
	off, _ := a.Alloc(4)

	data := []byte{0xDE, 0xAD, 0xBE, 0xEF}
	a.Write(off, data)

	got := a.Read(off, 4)
	for i, b := range got {
		if b != data[i] {
			t.Errorf("byte[%d] = 0x%02X, want 0x%02X", i, b, data[i])
		}
	}
}

func TestFreeZerosMemory(t *testing.T) {
	a := alloc.NewAllocator(8)
	off, _ := a.Alloc(4)

	a.Write(off, []byte{0xFF, 0xFF, 0xFF, 0xFF})
	a.Free(off, 4)

	// Re-allocate and check memory is zeroed
	off2, _ := a.Alloc(4)
	got := a.Read(off2, 4)
	for i, b := range got {
		if b != 0 {
			t.Errorf("byte[%d] = 0x%02X after free, want 0x00", i, b)
		}
	}
}
