package vm

import (
	"fmt"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/slot"
)

type Operation interface {
	Name() string
	Execute(compiler.Instruction, *Runtime) error
}

var OperationCodes = map[compiler.OperationCode]Operation{
	compiler.OpLoadCONST: OpLoadCONST{},
	compiler.OpStackPOP:  OpStackPOP{},
	compiler.OpVarALLOC:  OpVarALLOC{},
	compiler.OpVarINIT:   OpVarINIT{},
}

type OpDummy struct{}

func (OpDummy) Name() string {
	return ""
}

func (OpDummy) Execute(i compiler.Instruction, r *Runtime) error {
	return nil
}

type OpLoadCONST struct{}

func (OpLoadCONST) Name() string {
	return "OpLoadCONST"
}

func (OpLoadCONST) Execute(i compiler.Instruction, r *Runtime) error {
	c := r.IndexedFrame().ByteCode.Constants[i.Argument]
	if r.stack == nil {
		return fmt.Errorf("undefined stack")
	}

	var s slot.StackSlot
	s.Tag = c.Tag
	if c.Tag == slot.TagSlice {
		// String/slice constants live in the ByteCode constants table.
		// Store a reference to the slice (no copy) — HeapVal points to
		// the constants table backing, which outlives any stack slot.
		s.HeapVal = c.Data
	} else {
		size := slot.SizeForTag(c.Tag)
		copy(s.Data[:size], c.Data)
	}
	r.stack.Push(s)
	return nil
}

type OpStackPOP struct{}

func (OpStackPOP) Name() string {
	return "OpStackPOP"
}

func (OpStackPOP) Execute(i compiler.Instruction, r *Runtime) error {
	if r.stack == nil {
		return fmt.Errorf("undefined stack")
	}
	if _, err := r.stack.Pop(); err != nil {
		return err
	}
	return nil
}

// OpVarINIT combines VAR_ALLOC + LOAD_CONST + VAR_STORE into a single instruction.
// Instruction fields: Argument=slotID, Extra=typeMask, Offset=constIndex.
// The constant must be scalar (non-slice). No expression stack interaction.
type OpVarINIT struct{}

func (OpVarINIT) Name() string { return "OpVarINIT" }

func (OpVarINIT) Execute(i compiler.Instruction, r *Runtime) error {
	// Step 1: allocate the slot exactly as VAR_ALLOC would.
	if err := (OpVarALLOC{}).Execute(i, r); err != nil {
		return fmt.Errorf("OpVarINIT (alloc): %w", err)
	}
	// Step 2: copy the constant bytes directly into the allocated region.
	c := r.IndexedFrame().ByteCode.Constants[i.Offset]
	if c.Tag == slot.TagSlice {
		return fmt.Errorf("OpVarINIT: cannot use slice constant (index %d) — use OpSliceALLOC+OpVarSTORE", i.Offset)
	}
	sl := &r.slots[i.Argument]
	sl.Tag = c.Tag
	size := slot.SizeForTag(c.Tag)
	if size > 0 {
		dest := r.allocator.Slice(sl.Offset, sl.Capacity)
		copy(dest, c.Data[:size])
	}
	return nil
}

type OpVarALLOC struct{}

func (OpVarALLOC) Name() string {
	return "OpVarALLOC"
}

func (OpVarALLOC) Execute(i compiler.Instruction, r *Runtime) error {
	size := slot.MaxSizeForMask(i.Extra)
	if r.allocator == nil {
		return fmt.Errorf("no allocator active")
	}

	var offset int
	var inRegion bool

	frame := r.IndexedFrame()
	if frame.RegionSize > 0 {
		// Bump-allocate within the frame's pre-claimed contiguous region.
		if frame.bumpPtr+size > frame.RegionOffset+frame.RegionSize {
			return fmt.Errorf("frame region exhausted (bump=%d size=%d region=[%d,%d))", frame.bumpPtr, size, frame.RegionOffset, frame.RegionOffset+frame.RegionSize)
		}
		offset = frame.bumpPtr
		frame.bumpPtr += size
		inRegion = true
	} else {
		var err error
		offset, err = r.allocator.Alloc(size)
		if err != nil {
			return err
		}
	}

	// Grow slot table if needed
	for len(r.slots) <= i.Argument {
		r.slots = append(r.slots, SlotEntry{})
	}
	r.slots[i.Argument] = SlotEntry{
		Offset:   offset,
		Capacity: size,
		Tag:      0, // uninitialized until first store
		Mask:     i.Extra,
		Alive:    true,
		InRegion: inRegion,
	}
	return nil
}
