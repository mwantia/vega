package vm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"sync"

	"github.com/mwantia/vega/pkg/alloc"
	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vfs"
	"github.com/mwantia/vfs/mount"
)

const (
	MaxFrames = 256

	// DefaultAllocSize is the capacity of the global allocator in bytes (1 MiB).
	DefaultAllocSize = 1024 * 1024
)

// vmSession holds state that persists across Run() calls in REPL mode.
type vmSession struct {
	allocator *alloc.Allocator
	slots     []SlotEntry
	funcs     map[string]*compiler.FunctionDef
}

type VM struct {
	mu      sync.RWMutex
	fs      vfs.VirtualFileSystem
	session *vmSession

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

// StartSession implements VirtualMachine.
func (v *VM) StartSession() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.session = &vmSession{
		allocator: alloc.NewAllocator(DefaultAllocSize),
		slots:     make([]SlotEntry, 0),
		funcs:     make(map[string]*compiler.FunctionDef),
	}
}

// ResetSession implements VirtualMachine.
func (v *VM) ResetSession() {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.session = nil
}

// Run implements VirtualMachine.
func (v *VM) Run(ctx context.Context, bytecode *compiler.ByteCode) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	var a *alloc.Allocator
	var s []SlotEntry

	if v.session != nil {
		// Inject previously defined functions so this bytecode can call them.
		for name, fn := range v.session.funcs {
			if _, exists := bytecode.Functions[name]; !exists {
				bytecode.Functions[name] = fn
			}
		}
		a = v.session.allocator
		s = v.session.slots
	} else {
		a = alloc.NewAllocator(DefaultAllocSize)
		s = make([]SlotEntry, 0)
	}

	runtime := &Runtime{
		Frames:    make([]*CallFrame, MaxFrames),
		Index:     0,
		exprStack: &ExprStack{},
		allocator: a,
		slots:     s,
		native: &Native{
			Stdin:  v.stdin,
			Stdout: v.stdout,
			Stderr: v.stderr,
			FS:     v.fs,
		},
		userFuncs: bytecode.Functions,
	}
	runtime.Frames[0] = &CallFrame{
		ByteCode: bytecode,
	}

	if err := runtime.ExecuteFrames(ctx); err != nil {
		return 1, fmt.Errorf("runtime execution failed: %w", err)
	}

	if v.session != nil {
		// Persist updated slots and any newly defined functions.
		v.session.slots = runtime.slots
		for name, fn := range bytecode.Functions {
			v.session.funcs[name] = fn
		}
	}

	return 0, nil
}

// Snapshot implements VirtualMachine.
func (v *VM) Snapshot() *alloc.AllocSnapshot {
	v.mu.RLock()
	defer v.mu.RUnlock()
	if v.session == nil {
		return nil
	}
	snap := v.session.allocator.Snapshot()
	return &snap
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
