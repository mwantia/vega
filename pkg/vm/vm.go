package vm

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"sync"

	"github.com/mwantia/vega/pkg/alloc"
	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vfs"
	"github.com/mwantia/vfs/mount"
)

const (
	MaxFrames = 256
)

// Session holds state that persists across Run() calls in REPL mode.
type Session struct {
	allocator alloc.Allocator
	slots     []SlotEntry
	funcs     map[string]*compiler.FunctionDef
	snapshots *alloc.SnapshotManager // nil when allocator does not implement SnapshotTarget
}

type VM struct {
	mu      sync.RWMutex
	fs      vfs.VirtualFileSystem
	session *Session

	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer
}

var _ VirtualMachine = (*VM)(nil)

func NewVM(fs vfs.VirtualFileSystem) VirtualMachine {
	allocator := alloc.NewFreeListAllocator(0)
	var snapshots *alloc.SnapshotManager
	if target, ok := allocator.(alloc.Snapshottable); ok {
		snapshots, _ = alloc.NewSnapshotManager(target)
	}

	return &VM{
		fs: fs,
		session: &Session{
			allocator: allocator,
			snapshots: snapshots,
			slots:     make([]SlotEntry, 0),
			funcs:     make(map[string]*compiler.FunctionDef),
		},

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

	return NewVM(fs), nil
}

// ResetSession implements VirtualMachine.
func (v *VM) ResetSession() error {
	v.mu.Lock()
	defer v.mu.Unlock()

	allocator := alloc.NewFreeListAllocator(0)
	var snapshots *alloc.SnapshotManager
	if target, ok := allocator.(alloc.Snapshottable); ok {
		snapshots, _ = alloc.NewSnapshotManager(target)
	}

	v.session = &Session{
		allocator: allocator,
		snapshots: snapshots,
		slots:     make([]SlotEntry, 0),
		funcs:     make(map[string]*compiler.FunctionDef),
	}

	return nil
}

// Run implements VirtualMachine.
func (v *VM) Run(ctx context.Context, bytecode *compiler.ByteCode) (int, error) {
	v.mu.Lock()
	defer v.mu.Unlock()

	// Inject previously defined functions so this bytecode can call them.
	for name, fn := range v.session.funcs {
		if _, exists := bytecode.Functions[name]; !exists {
			bytecode.Functions[name] = fn
		}
	}

	runtime := &Runtime{
		Frames:    make([]*CallFrame, MaxFrames),
		Index:     0,
		exprStack: &ExprStack{},
		allocator: v.session.allocator,
		slots:     v.session.slots,
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

	// Persist updated slots and any newly defined functions.
	v.session.slots = runtime.slots
	maps.Copy(v.session.funcs, bytecode.Functions)

	return 0, nil
}

// Snapshot implements VirtualMachine.
func (v *VM) SnapshotManager() (*alloc.SnapshotManager, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if v.session == nil {
		return nil, fmt.Errorf("no active session exists")
	}

	return v.session.snapshots, nil
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
