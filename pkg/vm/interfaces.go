package vm

import (
	"context"
	"io"

	"github.com/mwantia/vega/pkg/alloc"
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

	// ResetSession clears all persistent session state, returning the VM to
	// stateless mode (each Run() call gets a fresh allocator and empty slots).
	ResetSession() error

	// Snapshot returns the SnapshotManager for the active session's allocator,
	// or nil if no session is active or the allocator does not support snapshots.
	// The manager persists for the lifetime of the session; callers should call
	// Take() after each Run() to record incremental deltas.
	SnapshotManager() (*alloc.SnapshotManager, error)
}
