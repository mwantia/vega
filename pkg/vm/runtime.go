package vm

import (
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/alloc"
	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/slot"
)

type SlotEntry struct {
	Offset      int
	Capacity    int // bytes currently accessible in the allocator for this slot
	DeclaredCap int // declared capacity from string<N>; non-zero for TagSlice slots
	Tag         slot.TypeTag
	Mask        byte
	Alive       bool
	Alias       bool // true = manually positioned pointer or slice view, not allocator-owned
	Stencil     bool // true = stencil-based allocation (struct/tuple)
	InRegion    bool // true = bump-allocated inside the enclosing frame's contiguous region
}

type Runtime struct {
	Frames []*CallFrame
	Index  int

	stack     *Stack
	allocator alloc.Allocator // global allocator shared across all scopes
	slots     []SlotEntry
	session   *RuntimeSession

	// pendingArgs holds arguments passed to the currently-being-entered user
	// function. They are consumed by OpLoadArg/OpLoadArgStencil in the function
	// prologue and cleared once the call is fully set up.
	pendingArgs []slot.StackSlot

	// userFuncs is the table of compiled user-defined functions populated from
	// the program's ByteCode.Functions before execution begins.
	userFuncs map[string]*compiler.FunctionDefinition
}

type CallFrame struct {
	ByteCode           *compiler.ByteCode
	InstructionPointer int
	BasePointer        int

	// ExprCall is true when this frame was entered from an expression context
	// (i.e. the return value is expected on the caller's stack).
	ExprCall bool

	// Frame region: a single contiguous block of the global allocator claimed
	// at frame entry for all fixed-size locals. Released atomically at frame
	// exit. RegionSize==0 means no region (top-level frame, or a function
	// whose locals are all strings).
	RegionOffset int
	RegionSize   int
	bumpPtr      int // next free byte within [RegionOffset, RegionOffset+RegionSize)

	// Saved caller state — restored when this frame returns.
	savedStack *Stack
	savedSlots []SlotEntry
}

func (r *Runtime) ExecuteInstruction(instr compiler.Instruction, frame *CallFrame) error {
	switch instr.Operation {
	case compiler.OpVarSTORE:
		slotID := instr.Argument
		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarSTORE': slot %d is not alive", slotID)
		}
		if r.stack == nil {
			return fmt.Errorf("instr 'OpVarSTORE': undefined stack")
		}

		s, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpVarSTORE': %w", err)
		}

		tag := s.Tag
		sl := &r.slots[slotID]
		if !slot.TagInMask(tag, sl.Mask) {
			return fmt.Errorf("instr 'OpVarSTORE': type mismatch: slot mask %08b does not allow tag %d", sl.Mask, tag)
		}
		sl.Tag = tag

		if tag == slot.TagSlice {
			var srcOff int
			var srcView []byte
			if s.HeapVal != nil {
				srcView = s.HeapVal
				srcOff = -1
			} else {
				ref := s.GetSliceRef()
				srcOff = int(ref.Offset)
				srcView = r.allocator.Slice(srcOff, int(ref.Length))
			}

			if srcOff >= 0 && !sl.InRegion {
				// Source lives in the allocator — make an alias, no copy.
				if !sl.Alias {
					r.allocator.Free(sl.Offset, sl.Capacity)
				}
				sl.Offset = srcOff
				sl.Capacity = len(srcView)
				sl.Alias = true
			} else {
				// Source is heap/constant or in a frame region — must copy.
				if sl.Alias {
					newOff, err := r.allocator.Alloc(sl.DeclaredCap)
					if err != nil {
						return fmt.Errorf("instr 'OpVarSTORE': %w", err)
					}
					sl.Offset = newOff
					sl.Capacity = sl.DeclaredCap
					sl.Alias = false
				}
				if len(srcView) > sl.Capacity {
					return fmt.Errorf("instr 'OpVarSTORE': slice value (%d bytes) exceeds slot capacity (%d bytes)", len(srcView), sl.Capacity)
				}
				dest := r.allocator.Slice(sl.Offset, sl.Capacity)
				n := copy(dest, srcView)
				for i := n; i < len(dest); i++ {
					dest[i] = 0
				}
			}
			return nil
		}

		// Non-slice: copy inline bytes into the allocator buffer.
		size := slot.SizeForTag(tag)
		dest := r.allocator.Slice(sl.Offset, sl.Capacity)
		n := copy(dest, s.Data[:size])
		for i := n; i < len(dest); i++ {
			dest[i] = 0
		}

	case compiler.OpVarLOAD:
		slotID := instr.Argument
		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarLOAD': use after free on slot %d", slotID)
		}

		sl := r.slots[slotID]
		if sl.Tag == 0 {
			return fmt.Errorf("instr 'OpVarLOAD': slot %d is uninitialized", slotID)
		}
		if r.stack == nil {
			return fmt.Errorf("instr 'OpVarLOAD': undefined stack")
		}

		var s slot.StackSlot
		s.Tag = sl.Tag
		if sl.Tag == slot.TagSlice {
			// Allocator-backed slice: encode as SliceRef (no heap alloc).
			s.PutSliceRef(slot.SliceRef{
				Offset: uint32(sl.Offset),
				Length: uint32(sl.Capacity),
			})
		} else {
			size := slot.SizeForTag(sl.Tag)
			copy(s.Data[:size], r.allocator.Slice(sl.Offset, size))
		}
		r.stack.Push(s)

	case compiler.OpVarFREE:
		slotID := instr.Argument
		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarFREE': double free on slot %d", slotID)
		}

		sl := r.slots[slotID]
		if sl.Alias {
			if sl.Tag != slot.TagSlice {
				return fmt.Errorf("instr 'OpVarFREE': cannot free pointer alias on slot %d", slotID)
			}
			r.slots[slotID].Alive = false
			return nil
		}
		if !sl.InRegion {
			r.allocator.Free(sl.Offset, sl.Capacity)
		}
		r.slots[slotID].Alive = false

	case compiler.OpSliceALLOC:
		slotID := instr.Argument
		capacity := instr.Offset

		if r.allocator == nil {
			return fmt.Errorf("instr 'OpSliceALLOC': no allocator active")
		}

		var offset int
		var inRegion bool
		currentFrame := r.IndexedFrame()
		if currentFrame.RegionSize > 0 {
			if currentFrame.bumpPtr+capacity > currentFrame.RegionOffset+currentFrame.RegionSize {
				return fmt.Errorf("instr 'OpSliceALLOC': frame region exhausted (bump=%d capacity=%d region=[%d,%d))",
					currentFrame.bumpPtr, capacity, currentFrame.RegionOffset, currentFrame.RegionOffset+currentFrame.RegionSize)
			}
			offset = currentFrame.bumpPtr
			currentFrame.bumpPtr += capacity
			inRegion = true
		} else {
			var err error
			offset, err = r.allocator.Alloc(capacity)
			if err != nil {
				return fmt.Errorf("instr 'OpSliceALLOC': %w", err)
			}
		}

		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset:      offset,
			Capacity:    capacity,
			DeclaredCap: capacity,
			Tag:         0,
			Mask:        slot.MaskForTag(slot.TagSlice),
			Alive:       true,
			InRegion:    inRegion,
		}

	case compiler.OpVarPTR:
		slotID := instr.Argument
		tag := slot.TypeTag(instr.Extra)
		size := slot.SizeForTag(tag)

		if r.stack == nil {
			return fmt.Errorf("instr 'OpVarPTR': undefined stack")
		}

		s, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpVarPTR': %w", err)
		}

		offset, err := s.AsInt()
		if err != nil {
			return fmt.Errorf("instr 'OpVarPTR': %w", err)
		}

		if r.allocator == nil {
			return fmt.Errorf("instr 'OpVarPTR': no allocator active")
		}

		if offset < 0 || offset+size > r.allocator.Size() {
			return fmt.Errorf("instr 'OpVarPTR': pointer out of bounds (offset=%d, size=%d, allocSize=%d)", offset, size, r.allocator.Size())
		}

		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset:   offset,
			Capacity: size,
			Tag:      tag,
			Mask:     slot.MaskForTag(tag),
			Alive:    true,
			Alias:    true,
		}

	case compiler.OpStencilALLOC:
		slotID := instr.Argument
		totalSize := instr.Offset

		if r.allocator == nil {
			return fmt.Errorf("instr 'OpStencilALLOC': no allocator active")
		}

		var offset int
		var inRegion bool
		currentFrame := r.IndexedFrame()
		if currentFrame.RegionSize > 0 {
			if currentFrame.bumpPtr+totalSize > currentFrame.RegionOffset+currentFrame.RegionSize {
				return fmt.Errorf("instr 'OpStencilALLOC': frame region exhausted")
			}
			offset = currentFrame.bumpPtr
			currentFrame.bumpPtr += totalSize
			inRegion = true
		} else {
			var err error
			offset, err = r.allocator.Alloc(totalSize)
			if err != nil {
				return fmt.Errorf("instr 'OpStencilALLOC': %w", err)
			}
		}

		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset:   offset,
			Capacity: totalSize,
			Tag:      0,
			Mask:     0,
			Alive:    true,
			Stencil:  true,
			InRegion: inRegion,
		}

	case compiler.OpFieldSTORE:
		slotID := instr.Argument
		fieldOffset := instr.Offset
		tag := slot.TypeTag(instr.Extra)

		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpFieldSTORE': slot %d is not alive", slotID)
		}
		if r.stack == nil {
			return fmt.Errorf("instr 'OpFieldSTORE': undefined stack")
		}

		s, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpFieldSTORE': %w", err)
		}

		if s.Tag != tag {
			return fmt.Errorf("instr 'OpFieldSTORE': type mismatch: expected tag %d, got %d", tag, s.Tag)
		}

		sl := r.slots[slotID]
		fieldSize := fieldSizeFor(tag, instr.Size)

		var src []byte
		if tag == slot.TagSlice {
			if s.HeapVal != nil {
				src = s.HeapVal
			} else {
				ref := s.GetSliceRef()
				src = r.allocator.Slice(int(ref.Offset), int(ref.Length))
			}
			if len(src) > fieldSize {
				return fmt.Errorf("instr 'OpFieldSTORE': slice value (%d bytes) exceeds field capacity (%d bytes)", len(src), fieldSize)
			}
		} else {
			src = s.Data[:fieldSize]
		}

		dest := r.allocator.Slice(sl.Offset+fieldOffset, fieldSize)
		n := copy(dest, src)
		for i := n; i < len(dest); i++ {
			dest[i] = 0
		}

	case compiler.OpFieldLOAD:
		slotID := instr.Argument
		fieldOffset := instr.Offset
		tag := slot.TypeTag(instr.Extra)

		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpFieldLOAD': slot %d is not alive", slotID)
		}
		if r.stack == nil {
			return fmt.Errorf("instr 'OpFieldLOAD': undefined stack")
		}

		sl := r.slots[slotID]
		fieldSize := fieldSizeFor(tag, instr.Size)

		var s slot.StackSlot
		s.Tag = tag
		if tag == slot.TagSlice {
			s.PutSliceRef(slot.SliceRef{
				Offset: uint32(sl.Offset + fieldOffset),
				Length: uint32(fieldSize),
			})
		} else {
			view := r.allocator.Slice(sl.Offset+fieldOffset, fieldSize)
			copy(s.Data[:fieldSize], view)
		}
		r.stack.Push(s)

	case compiler.OpCall:
		name := frame.ByteCode.Names[instr.Offset]
		argc := instr.Argument
		exprCall := instr.Extra == 1

		// Pop arguments left-to-right from the current expr stack.
		args := make([]slot.StackSlot, argc)
		for i := argc - 1; i >= 0; i-- {
			s, err := r.stack.Pop()
			if err != nil {
				return fmt.Errorf("instr 'OpCall' ('%s'): %w", name, err)
			}
			args[i] = s
		}

		// User-defined functions take priority over native functions.
		if fn, ok := r.userFuncs[name]; ok {
			if argc != len(fn.Params) {
				return fmt.Errorf("instr 'OpCall': '%s' expects %d argument(s), got %d",
					name, len(fn.Params), argc)
			}

			// Save the caller's expression stack and slot table.
			currentFrame := r.IndexedFrame()
			currentFrame.savedStack = r.stack
			currentFrame.savedSlots = r.slots

			// Initialize fresh state for the callee.
			r.stack = &Stack{}
			r.slots = make([]SlotEntry, 0)
			r.pendingArgs = args

			r.Index++
			if r.Index >= MaxFrames {
				return fmt.Errorf("instr 'OpCall': call stack overflow")
			}

			var regionOffset, regionSize, bumpPtr int
			if fn.FrameSize > 0 {
				off, err := r.allocator.Alloc(fn.FrameSize)
				if err != nil {
					r.Index--
					return fmt.Errorf("instr 'OpCall' ('%s'): frame region: %w", name, err)
				}
				regionOffset = off
				regionSize = fn.FrameSize
				bumpPtr = off
			}

			r.Frames[r.Index] = &CallFrame{
				ByteCode:     fn.ByteCode,
				ExprCall:     exprCall,
				RegionOffset: regionOffset,
				RegionSize:   regionSize,
				bumpPtr:      bumpPtr,
			}
			break
		}

		// Fall back to native (static) functions.
		desc, ok := descriptor.Global.LookupStatic(name)
		if !ok {
			return fmt.Errorf("instr 'OpCall': unknown function '%s'", name)
		}

		// Materialize any TagSlice args before passing to native code.
		for i := range args {
			args[i] = r.materializeSlot(args[i])
		}

		args, err := desc.ValidateArgs(args)
		if err != nil {
			return fmt.Errorf("instr 'OpCall' ('%s'): %w", name, err)
		}

		result, err := desc.Run(slot.VoidSlot, r.session, args)
		if err != nil {
			return fmt.Errorf("instr 'OpCall' ('%s'): %w", name, err)
		}
		if exprCall && desc.ReturnTag != slot.TagVoid {
			r.stack.Push(result)
		}

	case compiler.OpReturn:
		if r.Index == 0 {
			return nil
		}

		hasValue := instr.Extra == 1
		var retSlot slot.StackSlot
		if hasValue && r.stack != nil {
			var err error
			retSlot, err = r.stack.Pop()
			if err != nil {
				return fmt.Errorf("instr 'OpReturn': %w", err)
			}
		}

		if err := r.doReturn(r.IndexedFrame(), hasValue, retSlot); err != nil {
			return fmt.Errorf("instr 'OpReturn': %w", err)
		}

	case compiler.OpLoadArg:
		idx := instr.Argument
		if idx < 0 || idx >= len(r.pendingArgs) {
			return fmt.Errorf("instr 'OpLoadArg': index %d out of range (have %d pending)",
				idx, len(r.pendingArgs))
		}
		if r.stack == nil {
			return fmt.Errorf("instr 'OpLoadArg': expr stack not initialised")
		}
		r.stack.Push(r.pendingArgs[idx])

	case compiler.OpVarLoadRaw:
		slotID := instr.Argument
		size := instr.Offset

		if slotID >= len(r.slots) || !r.slots[slotID].Alive {
			return fmt.Errorf("instr 'OpVarLoadRaw': slot %d is not alive", slotID)
		}
		if r.stack == nil {
			return fmt.Errorf("instr 'OpVarLoadRaw': expr stack not initialised")
		}

		sl := r.slots[slotID]
		// Push a stencil ref — no heap copy on the fast path.
		var s slot.StackSlot
		s.Tag = slot.TagStencilRef
		s.PutSliceRef(slot.SliceRef{
			Offset: uint32(sl.Offset),
			Length: uint32(size),
		})
		r.stack.Push(s)

	case compiler.OpPtrLOAD:
		tag := slot.TypeTag(instr.Extra)
		size := slot.SizeForTag(tag)

		if r.stack == nil {
			return fmt.Errorf("instr 'OpPtrLOAD': undefined stack")
		}
		if r.allocator == nil {
			return fmt.Errorf("instr 'OpPtrLOAD': no allocator active")
		}

		s, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpPtrLOAD': %w", err)
		}

		offset, err := s.AsInt()
		if err != nil {
			return fmt.Errorf("instr 'OpPtrLOAD': %w", err)
		}

		if offset < 0 || offset+size > r.allocator.Size() {
			return fmt.Errorf("instr 'OpPtrLOAD': pointer out of bounds (offset=%d, size=%d, allocSize=%d)", offset, size, r.allocator.Size())
		}

		view := r.allocator.Slice(offset, size)
		var result slot.StackSlot
		result.Tag = tag
		copy(result.Data[:size], view)
		r.stack.Push(result)

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

		pending := r.pendingArgs[argIdx]
		if pending.Tag != slot.TagStencilRef {
			return fmt.Errorf("instr 'OpLoadArgStencil': expected stencil ref for struct arg, got tag %d",
				pending.Tag)
		}
		ref := pending.GetSliceRef()

		var offset int
		var inRegion bool
		currentFrame := r.IndexedFrame()
		if currentFrame.RegionSize > 0 {
			if currentFrame.bumpPtr+size > currentFrame.RegionOffset+currentFrame.RegionSize {
				return fmt.Errorf("instr 'OpLoadArgStencil': frame region exhausted")
			}
			offset = currentFrame.bumpPtr
			currentFrame.bumpPtr += size
			inRegion = true
		} else {
			var err error
			offset, err = r.allocator.Alloc(size)
			if err != nil {
				return fmt.Errorf("instr 'OpLoadArgStencil': %w", err)
			}
		}

		for len(r.slots) <= slotID {
			r.slots = append(r.slots, SlotEntry{})
		}
		r.slots[slotID] = SlotEntry{
			Offset:   offset,
			Capacity: size,
			Alive:    true,
			Stencil:  true,
			InRegion: inRegion,
		}

		// Copy from caller's allocator region (still alive — freed only at doReturn).
		copy(r.allocator.Slice(offset, size), r.allocator.Slice(int(ref.Offset), int(ref.Length)))

	case compiler.OpBuildSTRING:
		n := instr.Argument
		if r.stack == nil {
			return fmt.Errorf("instr 'OpBuildSTRING': undefined stack")
		}
		parts := make([]string, n)
		for i := n - 1; i >= 0; i-- {
			s, err := r.stack.Pop()
			if err != nil {
				return fmt.Errorf("instr 'OpBuildSTRING': stack underflow")
			}
			parts[i] = r.slotString(s)
		}
		result := strings.Join(parts, "")
		r.stack.Push(slot.NewString(result))

	case compiler.OpGetMember:
		name := frame.ByteCode.Names[instr.Offset]
		if r.stack == nil {
			return fmt.Errorf("instr 'OpGetMember': undefined stack")
		}
		s, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpGetMember': %w", err)
		}

		tag := s.Tag
		desc, ok := descriptor.Global.LookupMember(tag, name)
		if !ok {
			return fmt.Errorf("instr 'OpGetMember': value of tag %d has no member '%s'", tag, name)
		}

		result, err := desc.Getter(r.materializeSlot(s))
		if err != nil {
			return fmt.Errorf("instr 'OpGetMember' ('%s'): %w", name, err)
		}
		r.stack.Push(result)

	case compiler.OpCallMethod:
		name := frame.ByteCode.Names[instr.Offset]
		argc := instr.Argument
		if r.stack == nil {
			return fmt.Errorf("instr 'OpCallMethod': undefined stack")
		}

		argSlots := make([]slot.StackSlot, argc)
		for i := argc - 1; i >= 0; i-- {
			s, err := r.stack.Pop()
			if err != nil {
				return fmt.Errorf("instr 'OpCallMethod': %w", err)
			}
			argSlots[i] = s
		}

		objSlot, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCallMethod': %w", err)
		}

		tag := objSlot.Tag
		desc, ok := descriptor.Global.LookupMethod(tag, name)
		if !ok {
			return fmt.Errorf("instr 'OpCallMethod': value of tag %d has no method '%s'", tag, name)
		}

		// Materialize receiver and any TagSlice args before native boundary.
		objSlot = r.materializeSlot(objSlot)
		for i := range argSlots {
			argSlots[i] = r.materializeSlot(argSlots[i])
		}

		argSlots, err = desc.ValidateArgs(argSlots)
		if err != nil {
			return fmt.Errorf("instr 'OpCallMethod' ('%s'): %w", name, err)
		}

		result, err := desc.Run(objSlot, r.session, argSlots)
		if err != nil {
			return fmt.Errorf("instr 'OpCallMethod' ('%s'): %w", name, err)
		}

		// Push result (VoidSlot for void methods) so OpStackPOP in statement context always succeeds.
		r.stack.Push(result)

	case compiler.OpBinAdd:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinAdd': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinAdd': %w", err)
		}
		result, err := addSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpBinAdd': %w", err)
		}
		r.stack.Push(result)

	case compiler.OpBinSub:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinSub': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinSub': %w", err)
		}
		result, err := subSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpBinSub': %w", err)
		}
		r.stack.Push(result)

	case compiler.OpBinMul:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinMul': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinMul': %w", err)
		}
		result, err := mulSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpBinMul': %w", err)
		}
		r.stack.Push(result)

	case compiler.OpBinDiv:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinDiv': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinDiv': %w", err)
		}
		result, err := divSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpBinDiv': %w", err)
		}
		r.stack.Push(result)

	case compiler.OpBinMod:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinMod': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpBinMod': %w", err)
		}
		result, err := modSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpBinMod': %w", err)
		}
		r.stack.Push(result)

	case compiler.OpUnNeg:
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpUnNeg': %w", err)
		}
		result, err := negSlot(a)
		if err != nil {
			return fmt.Errorf("instr 'OpUnNeg': %w", err)
		}
		r.stack.Push(result)

	case compiler.OpCmpEQ:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpEQ': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpEQ': %w", err)
		}
		cmp, err := cmpSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpCmpEQ': %w", err)
		}
		r.stack.Push(slot.NewBool(cmp == 0))

	case compiler.OpCmpNE:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpNE': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpNE': %w", err)
		}
		cmp, err := cmpSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpCmpNE': %w", err)
		}
		r.stack.Push(slot.NewBool(cmp != 0))

	case compiler.OpCmpLT:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpLT': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpLT': %w", err)
		}
		cmp, err := cmpSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpCmpLT': %w", err)
		}
		r.stack.Push(slot.NewBool(cmp < 0))

	case compiler.OpCmpLE:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpLE': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpLE': %w", err)
		}
		cmp, err := cmpSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpCmpLE': %w", err)
		}
		r.stack.Push(slot.NewBool(cmp <= 0))

	case compiler.OpCmpGT:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpGT': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpGT': %w", err)
		}
		cmp, err := cmpSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpCmpGT': %w", err)
		}
		r.stack.Push(slot.NewBool(cmp > 0))

	case compiler.OpCmpGE:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpGE': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpCmpGE': %w", err)
		}
		cmp, err := cmpSlots(a, b)
		if err != nil {
			return fmt.Errorf("instr 'OpCmpGE': %w", err)
		}
		r.stack.Push(slot.NewBool(cmp >= 0))

	case compiler.OpLogAnd:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpLogAnd': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpLogAnd': %w", err)
		}
		r.stack.Push(slot.NewBool(a.Data[0] != 0 && b.Data[0] != 0))

	case compiler.OpLogOr:
		b, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpLogOr': %w", err)
		}
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpLogOr': %w", err)
		}
		r.stack.Push(slot.NewBool(a.Data[0] != 0 || b.Data[0] != 0))

	case compiler.OpLogNot:
		a, err := r.stack.Pop()
		if err != nil {
			return fmt.Errorf("instr 'OpLogNot': %w", err)
		}
		r.stack.Push(slot.NewBool(a.Data[0] == 0))
	}

	return nil
}

// doReturn tears down the current function frame and restores the caller's
// runtime state. If hasValue is true and the frame was entered from an
// expression context, retSlot is pushed onto the restored caller stack.
//
// Allocator-backed slice return values are materialized (copied to heap)
// before the frame's slots are freed so they remain valid.
func (r *Runtime) doReturn(frame *CallFrame, hasValue bool, retSlot slot.StackSlot) error {
	// Materialize allocator-backed slices/stencil refs before freeing slots.
	if hasValue {
		if retSlot.Tag == slot.TagSlice && retSlot.HeapVal == nil {
			ref := retSlot.GetSliceRef()
			data := make([]byte, ref.Length)
			copy(data, r.allocator.Slice(int(ref.Offset), int(ref.Length)))
			retSlot.HeapVal = data
		} else if retSlot.Tag == slot.TagStencilRef {
			ref := retSlot.GetSliceRef()
			data := make([]byte, ref.Length)
			copy(data, r.allocator.Slice(int(ref.Offset), int(ref.Length)))
			retSlot.HeapVal = data
		}
	}
	// Free slots that are NOT part of the frame region individually.
	for _, slot := range r.slots {
		if slot.Alive && !slot.Alias && !slot.InRegion {
			r.allocator.Free(slot.Offset, slot.Capacity)
		}
	}
	// Release the entire frame region atomically.
	if frame.RegionSize > 0 {
		r.allocator.Free(frame.RegionOffset, frame.RegionSize)
	}
	// Step back to the caller's frame and restore its saved state.
	r.Index--
	callerFrame := r.Frames[r.Index]
	r.stack = callerFrame.savedStack
	r.slots = callerFrame.savedSlots
	callerFrame.savedStack = nil
	callerFrame.savedSlots = nil
	// Push return value only when the call site expected one.
	if hasValue && frame.ExprCall && r.stack != nil {
		r.stack.Push(retSlot)
	}

	return nil
}

func (r *Runtime) IndexedFrame() *CallFrame {
	return r.Frames[r.Index]
}

// fieldSizeFor returns the byte width of a struct field.
// For scalar fields the size is derived from the type tag; for TagSlice fields
// the compiler encodes the declared capacity in the instruction's Size field.
func fieldSizeFor(tag slot.TypeTag, instrSize int) int {
	if tag == slot.TagSlice {
		return instrSize
	}
	return slot.SizeForTag(tag)
}
