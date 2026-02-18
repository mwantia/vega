package vm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vfs"
	"github.com/mwantia/vfs/mount"
)

const (
	MaxFrames = 256
)

type VM struct {
	mu sync.RWMutex
	fs vfs.VirtualFileSystem

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

var _ VirtualMachine = (*VM)(nil)

func NewVM(fs vfs.VirtualFileSystem) VirtualMachine {
	return &VM{
		fs: fs,

		stdin:  bytes.NewBuffer(nil),
		stdout: io.Discard,
		stderr: io.Discard,
	}
}

func NewEphemeralVM() (VirtualMachine, error) {
	ctx := context.Background()
	steps, err := mount.IdentifyMountSteps(ctx, "ephemeral://")
	if err != nil {
		return nil, err
	}

	fs, err := vfs.NewVirtualFileSystem()
	if err != nil {
		return nil, err
	}
	if err := fs.Mount(ctx, "/", steps...); err != nil {
		return nil, err
	}

	return &VM{
		fs: fs,

		stdin:  bytes.NewBuffer(nil),
		stdout: io.Discard,
		stderr: io.Discard,
	}, nil
}

// Run implements VirtualMachine.
func (v *VM) Run(ctx context.Context, bytecode *compiler.ByteCode) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	// Create initial call frame
	runtime := &Runtime{
		Frames: make([]*CallFrame, MaxFrames),
		Index:  0,
		native: &Native{
			Stdin:  v.stdin,
			Stdout: v.stdout,
			Stderr: v.stderr,
			FS:     v.fs,
		},
	}
	runtime.Frames[0] = &CallFrame{
		ByteCode: bytecode,
	}

	if err := runtime.ExecuteFrames(ctx); err != nil {
		return 1, fmt.Errorf("runtime execution failed: %w", err)
	}

	return 0, nil
}

// Stdin implements VirtualMachine.
func (v *VM) Stdin(stdin io.Reader) io.Reader {
	if stdin != nil {
		v.mu.RLock()
		defer v.mu.RUnlock()
		v.stdin = stdin
	}
	return v.stdin
}

// Stdout implements VirtualMachine.
func (v *VM) Stdout(stdout io.Writer) io.Writer {
	if stdout != nil {
		v.mu.RLock()
		defer v.mu.RUnlock()
		v.stdout = stdout
	}
	return v.stdout
}

// Stderr implements VirtualMachine.
func (v *VM) Stderr(stderr io.Writer) io.Writer {
	if stderr != nil {
		v.mu.RLock()
		defer v.mu.RUnlock()
		v.stderr = stderr
	}
	return v.stderr
}
