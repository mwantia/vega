package vm

import (
	"bytes"

	"github.com/mwantia/vega/pkg/slot"
)

// slotString returns the string representation of a StackSlot value.
// Used by OpBuildSTRING to stringify each interpolated part.
// For allocator-backed TagSlice slots it resolves the backing bytes via the
// allocator; all other types delegate to slot.Format().
func (r *Runtime) slotString(s slot.StackSlot) string {
	if s.Tag == slot.TagSlice {
		var view []byte
		if s.HeapVal != nil {
			view = s.HeapVal
		} else {
			ref := s.GetSliceRef()
			view = r.allocator.Slice(int(ref.Offset), int(ref.Length))
		}
		n := bytes.IndexByte(view, 0)
		if n >= 0 {
			return string(view[:n])
		}
		return string(view)
	}
	return s.Format()
}

// materializeSlot ensures that a TagSlice slot's content is backed by a heap
// allocation (HeapVal) rather than an allocator reference. All other slot
// types are returned unchanged. Call this before passing a slot across the
// native function boundary so that native closures always receive stable bytes.
func (r *Runtime) materializeSlot(s slot.StackSlot) slot.StackSlot {
	if s.Tag == slot.TagSlice && s.HeapVal == nil {
		ref := s.GetSliceRef()
		data := make([]byte, ref.Length)
		copy(data, r.allocator.Slice(int(ref.Offset), int(ref.Length)))
		s.HeapVal = data
		s.Data = [8]byte{}
	}
	return s
}
