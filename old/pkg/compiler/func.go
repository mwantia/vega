package compiler

import (
	"fmt"

	"github.com/mwantia/vega/old/pkg/value"
)

// Function represents a compiled function.
type Function struct {
	Name       string
	Parameters []string
	Bytecode   *Bytecode
}

func (f *Function) Type() string {
	return "func"
}

func (f *Function) String() string {
	return fmt.Sprintf("<fn %s>", f.Name)
}

func (f *Function) Boolean() bool {
	return true
}

func (f *Function) Equal(other value.Value) bool {
	if o, ok := other.(*Function); ok {
		return f.Name == o.Name
	}
	return false
}
