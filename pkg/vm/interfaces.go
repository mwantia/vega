package vm

import (
	"context"
	"io"

	"github.com/mwantia/vega/pkg/compiler"
)

// VirtualMachine
type VirtualMachine interface {
	// Stdin returns and/or sets the currently used reader for stdin input messages.
	Stdin(io.Reader) io.Reader

	// Stdout returns and/or sets the currently used writer to output stdout messages.
	Stdout(io.Writer) io.Writer

	// Stderr returns and/or sets the currently used writer to output stderr messages.
	Stderr(io.Writer) io.Writer

	// Run executes bytecode and returns the exit code.
	Run(context.Context, *compiler.ByteCode) (int, error)
}
