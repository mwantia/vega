package vm

import (
	"fmt"

	"github.com/mwantia/vega/pkg/slot"
)

// Stack holds temporary values during expression evaluation.
// Values on this stack are self-contained StackSlots — scalars live inline
// with zero heap allocation on the hot path.
type Stack struct {
	data []slot.StackSlot
}

// Push appends a slot onto the stack.
func (s *Stack) Push(slot slot.StackSlot) {
	s.data = append(s.data, slot)
}

// Pop removes and returns the top slot from the stack.
func (s *Stack) Pop() (slot.StackSlot, error) {
	n := len(s.data)
	if n == 0 {
		return slot.StackSlot{}, fmt.Errorf("stack underflow")
	}

	slot := s.data[n-1]
	s.data = s.data[:n-1]
	return slot, nil
}

// Peek returns the top slot without removing it.
func (s *Stack) Peek() (slot.StackSlot, error) {
	n := len(s.data)
	if n == 0 {
		return slot.StackSlot{}, fmt.Errorf("stack underflow")
	}

	return s.data[n-1], nil
}

// Len returns the number of slots on the stack.
func (s *Stack) Len() int {
	return len(s.data)
}

// Reset clears the stack.
func (s *Stack) Reset() {
	s.data = s.data[:0]
}
