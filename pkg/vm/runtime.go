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
	allocator *alloc.Allocator // global allocator shared across all scopes
	slots     []SlotEntry
	native    *Native

	// pendingArgs holds arguments passed to the currently-being-entered user
	// function. They are consumed by OpLoadArg instructions in the function
	// prologue and cleared once the call is fully set up.
	pendingArgs []value.Value

	// userFuncs is the table of compiled user-defined functions populated from
	// the program's ByteCode.Functions before execution begins.
	userFuncs map[string]*compiler.FunctionDef
}

type CallFrame struct {
	ByteCode           *compiler.ByteCode
	InstructionPointer int
	BasePointer        int

	// ExprCall is true when this frame was entered from an expression context
	// (i.e. the return value is expected on the caller's stack).
	ExprCall bool

	// Saved caller state — restored when this frame returns.
	savedExprStack *ExprStack
	savedSlots     []SlotEntry
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
			if r.Index == 0 {
				return nil
			}
			// Implicit void return — clean up and restore caller state.
			if err := r.doReturn(frame, false, nil); err != nil {
				return err
			}
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
		// For strings (variable length) use slot.Size; for fixed types use SizeForTag.
		var view []byte
		if slot.Tag == value.TagString {
			view = r.allocator.Slice(slot.Offset, slot.Size)
		} else {
			view = r.allocator.Slice(slot.Offset, value.SizeForTag(slot.Tag))
		}
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

	case compiler.OpStrSTORE:
		slotID := instr.Argument

		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpStrSTORE': undefined stack")
		}
		if r.allocator == nil {
			return fmt.Errorf("instr 'OpStrSTORE': no allocator active")
		}

		val, err := r.exprStack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpStrSTORE': %w", err)
		}

		sv, ok := val.(*value.StringValue)
		if !ok {
			return fmt.Errorf("instr 'OpStrSTORE': expected string value, got %T", val)
		}

		need := len(sv.View())

		// Grow slot table if needed.
		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}

		slot := &r.slots[slotID]
		if !slot.Alive {
			// First assignment — allocate fresh.
			offset, err := r.allocator.Alloc(need)
			if err != nil {
				return fmt.Errorf("instr 'OpStrSTORE': %w", err)
			}
			slot.Offset = offset
			slot.Size = need
			slot.Tag = value.TagString
			slot.Alive = true
		} else if slot.Size != need {
			// Reassignment with different length — free old, alloc new.
			r.allocator.Free(slot.Offset, slot.Size)
			offset, err := r.allocator.Alloc(need)
			if err != nil {
				return fmt.Errorf("instr 'OpStrSTORE': realloc: %w", err)
			}
			slot.Offset = offset
			slot.Size = need
		}

		copy(r.allocator.Slice(slot.Offset, slot.Size), sv.View())

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

	case compiler.OpCallFN:
		name := instr.Name
		argc := instr.Argument

		fn, ok := r.userFuncs[name]
		if !ok {
			return fmt.Errorf("instr 'OpCallFN': unknown user function '%s'", name)
		}
		if argc != len(fn.Params) {
			return fmt.Errorf("instr 'OpCallFN': '%s' expects %d argument(s), got %d",
				name, len(fn.Params), argc)
		}

		// Pop arguments left-to-right from the current expr stack.
		args := make([]value.Value, argc)
		for i := argc - 1; i >= 0; i-- {
			val, err := r.exprStack.Pop()
			if err != nil {
				return fmt.Errorf("instr 'OpCallFN' ('%s'): %w", name, err)
			}
			args[i] = val
		}

		// Save the caller's expression stack and slot table inside the current
		// frame so that doReturn can restore them when the function exits.
		// The global allocator is shared and never saved/restored.
		currentFrame := r.IndexedFrame()
		currentFrame.savedExprStack = r.exprStack
		currentFrame.savedSlots = r.slots

		// Initialize fresh expression stack and slot table for the callee.
		r.exprStack = &ExprStack{}
		r.slots = make([]SlotEntry, 0)
		r.pendingArgs = args

		// Push new call frame.
		r.Index++
		if r.Index >= MaxFrames {
			return fmt.Errorf("instr 'OpCallFN': call stack overflow")
		}
		r.Frames[r.Index] = &CallFrame{
			ByteCode: fn.ByteCode,
			ExprCall: false, // statement context — return value is discarded
		}

	case compiler.OpReturn:
		if r.Index == 0 {
			// Returning from the top-level program is a no-op; execution will
			// stop naturally when IP reaches the end of the main bytecode.
			return nil
		}

		hasValue := instr.Extra == 1
		var retVal value.Value
		if hasValue && r.exprStack != nil {
			val, err := r.exprStack.Pop()
			if err != nil {
				return fmt.Errorf("instr 'OpReturn': %w", err)
			}
			retVal = val
		}

		if err := r.doReturn(r.IndexedFrame(), hasValue, retVal); err != nil {
			return fmt.Errorf("instr 'OpReturn': %w", err)
		}

	case compiler.OpLoadArg:
		idx := instr.Argument
		if idx < 0 || idx >= len(r.pendingArgs) {
			return fmt.Errorf("instr 'OpLoadArg': index %d out of range (have %d pending)",
				idx, len(r.pendingArgs))
		}
		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpLoadArg': expr stack not initialised")
		}
		r.exprStack.Push(r.pendingArgs[idx])

	case compiler.OpVarLoadRaw:
		slotID := instr.Argument
		size := instr.Offset

		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarLoadRaw': slot %d is not alive", slotID)
		}
		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpVarLoadRaw': expr stack not initialised")
		}

		slot := r.slots[slotID]
		data := make([]byte, size)
		copy(data, r.allocator.Slice(slot.Offset, size))
		r.exprStack.Push(&value.RawValue{Data: data})

	case compiler.OpPtrLOAD:
		tag := value.TypeTag(instr.Extra)
		size := value.SizeForTag(tag)

		if r.exprStack == nil {
			return fmt.Errorf("instr 'OpPtrLOAD': undefined stack")
		}
		if r.allocator == nil {
			return fmt.Errorf("instr 'OpPtrLOAD': no allocator active")
		}

		val, err := r.exprStack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpPtrLOAD': %w", err)
		}

		alloc, ok := val.(value.Allocable)
		if !ok {
			return fmt.Errorf("instr 'OpPtrLOAD': offset value is not allocable")
		}

		offset, err := value.ToInt(alloc)
		if err != nil {
			return fmt.Errorf("instr 'OpPtrLOAD': %w", err)
		}

		if offset < 0 || offset+size > r.allocator.Capacity() {
			return fmt.Errorf("instr 'OpPtrLOAD': pointer out of bounds (offset=%d, size=%d, capacity=%d)", offset, size, r.allocator.Capacity())
		}

		view := r.allocator.Slice(offset, size)
		loaded, err := value.Wrap(tag, view)
		if err != nil {
			return fmt.Errorf("instr 'OpPtrLOAD': %w", err)
		}
		r.exprStack.Push(loaded)

	case compiler.OpLoadArgStencil:
		argIdx := instr.Argument
		slotID := int(instr.Extra)
		size := instr.Offset

		if argIdx < 0 || argIdx >= len(r.pendingArgs) {
			return fmt.Errorf("instr 'OpLoadArgStencil': index %d out of range (have %d pending)",
				argIdx, len(r.pendingArgs))
		}
		if r.allocator == nil {
			return fmt.Errorf("instr 'OpLoadArgStencil': no allocator active")
		}

		raw, ok := r.pendingArgs[argIdx].(*value.RawValue)
		if !ok {
			return fmt.Errorf("instr 'OpLoadArgStencil': expected RawValue for struct arg, got %T",
				r.pendingArgs[argIdx])
		}

		offset, err := r.allocator.Alloc(size)
		if err != nil {
			return fmt.Errorf("instr 'OpLoadArgStencil': %w", err)
		}

		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset:  offset,
			Size:    size,
			Alive:   true,
			Stencil: true,
		}

		copy(r.allocator.Slice(offset, size), raw.Data)
	}

	return nil
}

// doReturn tears down the current function frame and restores the caller's
// runtime state. If hasValue is true and the frame was entered from an
// expression context, retVal is pushed onto the restored caller stack.
//
// Before freeing the function's slots, the return value is materialized
// (copied) so that it remains valid after the backing memory is returned
// to the global allocator.
func (r *Runtime) doReturn(frame *CallFrame, hasValue bool, retVal value.Value) error {
	// Materialize the return value before freeing slots, so it doesn't point
	// into memory that is about to be returned to the allocator.
	var materializedRet value.Value
	if hasValue && retVal != nil {
		if a, ok := retVal.(value.Allocable); ok {
			src := a.View()
			data := make([]byte, len(src))
			copy(data, src)
			wrapped, err := value.Wrap(value.TagFor(a), data)
			if err == nil {
				materializedRet = wrapped
			} else {
				materializedRet = retVal // fallback: keep the view (may be stale)
			}
		} else {
			materializedRet = retVal
		}
	}

	// Free all alive, non-alias slots from this function scope back to the
	// global allocator. Alias slots are not owned by the allocator.
	for _, slot := range r.slots {
		if slot.Alive && !slot.Alias {
			r.allocator.Free(slot.Offset, slot.Size)
		}
	}

	// Step back to the caller's frame and restore its saved state.
	r.Index--
	callerFrame := r.Frames[r.Index]
	r.exprStack = callerFrame.savedExprStack
	r.slots = callerFrame.savedSlots
	callerFrame.savedExprStack = nil
	callerFrame.savedSlots = nil

	// Push return value only when the call site expected one.
	if hasValue && frame.ExprCall && r.exprStack != nil {
		r.exprStack.Push(materializedRet)
	}

	return nil
}

func (r *Runtime) IndexedFrame() *CallFrame {
	return r.Frames[r.Index]
}
