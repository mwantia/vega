package vm

import (
	"fmt"

	"github.com/mwantia/vega/pkg/value"
)

// ExprStack holds temporary values during expression evaluation.
// Values on this stack are view-based — they reference data in the alloc buffer
// or the constants table, not independent copies.
type ExprStack struct {
	data []value.Value
}

// Push appends a value onto the stack.
func (s *ExprStack) Push(val value.Value) {
	s.data = append(s.data, val)
}

// Pop removes and returns the top value from the stack.
func (s *ExprStack) Pop() (value.Value, error) {
	n := len(s.data)
	if n == 0 {
		return nil, fmt.Errorf("stack underflow")
	}

	val := s.data[n-1]
	s.data = s.data[:n-1]
	return val, nil
}

// Peek returns the top value without removing it.
func (s *ExprStack) Peek() (value.Value, error) {
	n := len(s.data)
	if n == 0 {
		return nil, fmt.Errorf("stack underflow")
	}

	return s.data[n-1], nil
}

// Len returns the number of values on the stack.
func (s *ExprStack) Len() int {
	return len(s.data)
}

// Reset clears the stack.
func (s *ExprStack) Reset() {
	s.data = s.data[:0]
}
