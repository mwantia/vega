package vm

import (
	"context"
	"io"
	"sync"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/value"
	libvfs "github.com/mwantia/vfs"
)

// VirtualMachine is the Vega virtual machine.
type VirtualMachine struct {
	stack    []value.Value
	sp       int // Stack pointer
	globals  map[string]value.Value
	builtins map[string]BuiltinFunc

	// Context for cancellation and timeouts
	ctx    context.Context
	cancel context.CancelFunc
	mu     sync.RWMutex // Protects context access

	// VFS integration
	vfs libvfs.VirtualFileSystem

	// Call frames for function calls
	frames     []*CallFrame
	frameIndex int

	// I/O streams
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	// Iteration state (for for-loops)
	iterators map[string]value.Iterator

	// Execution trace (for debugging)
	traceEnabled bool
	trace        []TraceEntry
}

// TraceEntry records a single executed instruction for debugging.
type TraceEntry struct {
	FrameIndex int                  // Call frame depth
	IP         int                  // Instruction pointer before execution
	Instr      compiler.Instruction // The instruction that was executed
	StackTop   string               // String representation of top-of-stack after execution
	StackDepth int                  // Stack pointer after execution
}

// CallFrame represents a function call frame.
type CallFrame struct {
	bytecode *compiler.Bytecode
	ip       int                    // Instruction pointer
	bp       int                    // Base pointer (stack base for this frame)
	locals   map[string]value.Value // Local variables
}
