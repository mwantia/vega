package vm

import (
	"context"
	"fmt"

	"github.com/mwantia/vega/pkg/alloc"
	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/value"
)

type SlotEntry struct {
	Offset  int
	Size    int
	Tag     value.TypeTag
	Mask    byte
	Alive   bool
	Alias   bool // true = manually positioned pointer, not allocator-owned
	Stencil bool // true = stencil-based allocation (struct/tuple)
}

type Runtime struct {
	Frames []*CallFrame
	Index  int

	exprStack *ExprStack
	allocator *alloc.Allocator
	slots     []SlotEntry
	native    *Native
}

type CallFrame struct {
	ByteCode           *compiler.ByteCode
	InstructionPointer int
	BasePointer        int
}

func (r *Runtime) ExecuteFrames(ctx context.Context) error {
	for {
		select {
		// Check for context cancellation
		case <-ctx.Done():
			return ctx.Err()
		default:
			// Continue execution
		}

		frame := r.IndexedFrame()
		if frame.InstructionPointer >= len(frame.ByteCode.Instructions) {
			// End of bytecode reached
			if r.Index == 0 {
				return nil
			}
			// Return from function without explicit return
			r.Index--
			continue
		}

		instr := frame.ByteCode.Instructions[frame.InstructionPointer]
		frame.InstructionPointer++

		if err := r.ExecuteInstruction(instr, frame); err != nil {
			return fmt.Errorf("line %d: %w", instr.SourceLine, err)
		}
	}
}

func (r *Runtime) ExecuteInstruction(instr compiler.Instruction, frame *CallFrame) error {
	switch instr.Operation {
	case compiler.OpStackALLOC:
		if err := r.allocStack(instr.Argument); err != nil {
			return fmt.Errorf("instr 'OpStackALLOC': %w", err)
		}

	case compiler.OpStackFREE:
		r.freeStack()

	case compiler.OpLoadCONST:
		c := frame.ByteCode.Constants[instr.Argument]
		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpLoadCONST': undefined stack")
		}
		// Wrap the constant's data slice as a view-based value.
		// The value reads directly from the constants table — no copy.
		val, err := value.Wrap(c.Tag, c.Data)
		if err != nil {
			return fmt.Errorf("instr 'OpLoadCONST': %w", err)
		}
		r.exprStack.Push(val)

	case compiler.OpStackPOP:
		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpStackPOP': undefined stack")
		}
		if _, err := r.exprStack.Pop(); err != nil {
			return fmt.Errorf("instr 'OpStackPOP': %w", err)
		}

	case compiler.OpVarALLOC:
		slotID := instr.Argument
		mask := instr.Extra
		size := value.MaxSizeForMask(mask)

		if r.allocator == nil {
			return fmt.Errorf("instr 'OpVarALLOC': no allocator active")
		}

		offset, err := r.allocator.Alloc(size)
		if err != nil {
			return fmt.Errorf("instr 'OpVarALLOC': %w", err)
		}

		// Grow slot table if needed
		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset: offset,
			Size:   size,
			Tag:    0, // uninitialized until first store
			Mask:   mask,
			Alive:  true,
		}

	case compiler.OpVarSTORE:
		slotID := instr.Argument
		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarSTORE': slot %d is not alive", slotID)
		}

		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpVarSTORE': undefined stack")
		}

		val, err := r.exprStack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpVarSTORE': %w", err)
		}

		alloc, ok := val.(value.Allocable)
		if !ok {
			return fmt.Errorf("instr 'OpVarSTORE': value is not allocable")
		}

		tag := value.TagFor(alloc)
		slot := &r.slots[slotID]
		if !value.TagInMask(tag, slot.Mask) {
			return fmt.Errorf("instr 'OpVarSTORE': type mismatch: slot mask %08b does not allow tag %d", slot.Mask, tag)
		}
		slot.Tag = tag

		// Copy the value's backing bytes into the alloc buffer.
		// This is the one copy point: from constant/temporary → alloc buffer.
		dest := r.allocator.Slice(slot.Offset, slot.Size)
		src := alloc.View()
		n := copy(dest, src)
		for i := n; i < len(dest); i++ {
			dest[i] = 0
		}

	case compiler.OpVarLOAD:
		slotID := instr.Argument
		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarLOAD': use after free on slot %d", slotID)
		}

		slot := r.slots[slotID]
		if slot.Tag == 0 {
			return fmt.Errorf("instr 'OpVarLOAD': slot %d is uninitialized", slotID)
		}

		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpVarLOAD': undefined stack")
		}

		// Create a view-based value that points directly into the alloc buffer.
		// No copy — the value reads from the allocator's memory.
		typeSize := value.SizeForTag(slot.Tag)
		view := r.allocator.Slice(slot.Offset, typeSize)
		val, err := value.Wrap(slot.Tag, view)
		if err != nil {
			return fmt.Errorf("instr 'OpVarLOAD': %w", err)
		}
		r.exprStack.Push(val)

	case compiler.OpVarFREE:
		slotID := instr.Argument
		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarFREE': double free on slot %d", slotID)
		}

		if r.slots[slotID].Alias {
			return fmt.Errorf("instr 'OpVarFREE': cannot free pointer alias on slot %d", slotID)
		}

		slot := r.slots[slotID]
		r.allocator.Free(slot.Offset, slot.Size)
		r.slots[slotID].Alive = false

	case compiler.OpVarPTR:
		slotID := instr.Argument
		tag := value.TypeTag(instr.Extra)
		size := value.SizeForTag(tag)

		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpVarPTR': undefined stack")
		}

		val, err := r.exprStack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpVarPTR': %w", err)
		}

		alloc, ok := val.(value.Allocable)
		if !ok {
			return fmt.Errorf("instr 'OpVarPTR': offset value is not allocable")
		}

		offset, err := value.ToInt(alloc)
		if err != nil {
			return fmt.Errorf("instr 'OpVarPTR': %w", err)
		}

		if r.allocator == nil {
			return fmt.Errorf("instr 'OpVarPTR': no allocator active")
		}

		if offset < 0 || offset+size > r.allocator.Capacity() {
			return fmt.Errorf("instr 'OpVarPTR': pointer out of bounds (offset=%d, size=%d, capacity=%d)", offset, size, r.allocator.Capacity())
		}

		// Grow slot table if needed
		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset: offset,
			Size:   size,
			Tag:    tag,
			Mask:   value.MaskForTag(tag),
			Alive:  true,
			Alias:  true,
		}

	case compiler.OpStencilALLOC:
		slotID := instr.Argument
		totalSize := instr.Offset

		if r.allocator == nil {
			return fmt.Errorf("instr 'OpStencilALLOC': no allocator active")
		}

		offset, err := r.allocator.Alloc(totalSize)
		if err != nil {
			return fmt.Errorf("instr 'OpStencilALLOC': %w", err)
		}

		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset:  offset,
			Size:    totalSize,
			Tag:     0,
			Mask:    0,
			Alive:   true,
			Stencil: true,
		}

	case compiler.OpFieldSTORE:
		slotID := instr.Argument
		fieldOffset := instr.Offset
		tag := value.TypeTag(instr.Extra)

		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpFieldSTORE': slot %d is not alive", slotID)
		}
		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpFieldSTORE': undefined stack")
		}

		val, err := r.exprStack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpFieldSTORE': %w", err)
		}

		alloc, ok := val.(value.Allocable)
		if !ok {
			return fmt.Errorf("instr 'OpFieldSTORE': value is not allocable")
		}

		actualTag := value.TagFor(alloc)
		if actualTag != tag {
			return fmt.Errorf("instr 'OpFieldSTORE': type mismatch: expected tag %d, got %d", tag, actualTag)
		}

		slot := r.slots[slotID]
		fieldSize := value.SizeForTag(tag)
		dest := r.allocator.Slice(slot.Offset+fieldOffset, fieldSize)
		src := alloc.View()
		copy(dest, src)

	case compiler.OpFieldLOAD:
		slotID := instr.Argument
		fieldOffset := instr.Offset
		tag := value.TypeTag(instr.Extra)

		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpFieldLOAD': slot %d is not alive", slotID)
		}
		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpFieldLOAD': undefined stack")
		}

		slot := r.slots[slotID]
		fieldSize := value.SizeForTag(tag)
		view := r.allocator.Slice(slot.Offset+fieldOffset, fieldSize)
		val, err := value.Wrap(tag, view)
		if err != nil {
			return fmt.Errorf("instr 'OpFieldLOAD': %w", err)
		}
		r.exprStack.Push(val)

	case compiler.OpCallNAT:
		name := instr.Name
		argc := instr.Argument

		fn, ok := lookupNative(name)
		if !ok {
			return fmt.Errorf("instr 'OpCallNAT': unknown native function '%s'", name)
		}

		args := make([]value.Value, argc)
		for i := argc - 1; i >= 0; i-- {
			val, err := r.exprStack.Pop()
			if err != nil {
				return fmt.Errorf("instr 'OpCallNAT': %w", err)
			}
			args[i] = val
		}

		if err := fn(r.native, args); err != nil {
			return fmt.Errorf("instr 'OpCallNAT' ('%s'): %w", name, err)
		}
	}

	return nil
}

func (r *Runtime) IndexedFrame() *CallFrame {
	return r.Frames[r.Index]
}

func (r *Runtime) allocStack(size int) error {
	if r.exprStack != nil {
		return fmt.Errorf("stack already allocated")
	}
	r.exprStack = &ExprStack{}
	r.allocator = alloc.NewAllocator(size)
	r.slots = make([]SlotEntry, 0)
	return nil
}

func (r *Runtime) freeStack() {
	r.exprStack = nil
	r.allocator = nil
	r.slots = nil
}
