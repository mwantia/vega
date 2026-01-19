package compiler

import (
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/value"
)

// Instruction represents a single bytecode instruction.
type Instruction struct {
	Op   OpCode
	Arg  int    // Numeric argument (index, count, offset)
	Name string // String argument (variable name, function name)
	Line int    // Source line number for error reporting
}

func (i Instruction) String() string {
	switch i.Op {
	case OpLoadConst:
		return fmt.Sprintf("%s %d", i.Op, i.Arg)
	case OpLoadVar, OpStoreVar, OpLoadAttr:
		return fmt.Sprintf("%s %q", i.Op, i.Name)
	case OpCall, OpCallMethod:
		return fmt.Sprintf("%s %q (%d args)", i.Op, i.Name, i.Arg)
	case OpJmp, OpJmpIfFalse, OpJmpIfTrue:
		return fmt.Sprintf("%s %d", i.Op, i.Arg)
	case OpBuildArray, OpBuildMap:
		return fmt.Sprintf("%s %d", i.Op, i.Arg)
	case OpIterInit, OpIterNext:
		return fmt.Sprintf("%s %q (jump: %d)", i.Op, i.Name, i.Arg)
	default:
		return i.Op.String()
	}
}

// Bytecode represents compiled bytecode for a program or function.
type Bytecode struct {
	Instructions []Instruction
	Constants    []value.Value
	// LoopStack tracks nested loops for break/continue
	loopStack []loopInfo
}

type loopInfo struct {
	startAddr    int   // Address of loop start (for continue)
	breakAddrs   []int // Addresses of break jumps to patch
	continueAddrs []int // Addresses of continue jumps to patch
}

// NewBytecode creates a new empty Bytecode.
func NewBytecode() *Bytecode {
	return &Bytecode{
		Instructions: make([]Instruction, 0),
		Constants:    make([]value.Value, 0),
	}
}

// Emit adds an instruction and returns its address.
func (b *Bytecode) Emit(op OpCode, line int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Op:   op,
		Line: line,
	})
	return addr
}

// EmitArg adds an instruction with a numeric argument.
func (b *Bytecode) EmitArg(op OpCode, arg int, line int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Op:   op,
		Arg:  arg,
		Line: line,
	})
	return addr
}

// EmitName adds an instruction with a string argument.
func (b *Bytecode) EmitName(op OpCode, name string, line int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Op:   op,
		Name: name,
		Line: line,
	})
	return addr
}

// EmitNameArg adds an instruction with both string and numeric arguments.
func (b *Bytecode) EmitNameArg(op OpCode, name string, arg int, line int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Op:   op,
		Name: name,
		Arg:  arg,
		Line: line,
	})
	return addr
}

// AddConstant adds a constant to the pool and returns its index.
func (b *Bytecode) AddConstant(v value.Value) int {
	// Check if constant already exists
	for i, c := range b.Constants {
		if c.Equal(v) {
			return i
		}
	}
	idx := len(b.Constants)
	b.Constants = append(b.Constants, v)
	return idx
}

// CurrentAddr returns the address of the next instruction to be emitted.
func (b *Bytecode) CurrentAddr() int {
	return len(b.Instructions)
}

// PatchJump patches a jump instruction at addr to jump to the current address.
func (b *Bytecode) PatchJump(addr int) {
	b.Instructions[addr].Arg = b.CurrentAddr()
}

// PatchJumpTo patches a jump instruction at addr to jump to target.
func (b *Bytecode) PatchJumpTo(addr, target int) {
	b.Instructions[addr].Arg = target
}

// PushLoop starts tracking a new loop.
func (b *Bytecode) PushLoop(startAddr int) {
	b.loopStack = append(b.loopStack, loopInfo{
		startAddr:    startAddr,
		breakAddrs:   make([]int, 0),
		continueAddrs: make([]int, 0),
	})
}

// PopLoop ends loop tracking and patches break/continue jumps.
func (b *Bytecode) PopLoop() {
	if len(b.loopStack) == 0 {
		return
	}
	loop := b.loopStack[len(b.loopStack)-1]
	b.loopStack = b.loopStack[:len(b.loopStack)-1]

	// Patch all break jumps to current address (after loop)
	for _, addr := range loop.breakAddrs {
		b.PatchJump(addr)
	}

	// Continue jumps are patched when emitted (to loop start)
}

// AddBreak adds a break jump address to be patched later.
func (b *Bytecode) AddBreak(addr int) {
	if len(b.loopStack) > 0 {
		b.loopStack[len(b.loopStack)-1].breakAddrs = append(
			b.loopStack[len(b.loopStack)-1].breakAddrs, addr)
	}
}

// GetLoopStart returns the start address of the current loop.
func (b *Bytecode) GetLoopStart() int {
	if len(b.loopStack) > 0 {
		return b.loopStack[len(b.loopStack)-1].startAddr
	}
	return -1
}

// InLoop returns true if we're inside a loop.
func (b *Bytecode) InLoop() bool {
	return len(b.loopStack) > 0
}

// Disassemble returns a string representation of the bytecode.
func (b *Bytecode) Disassemble() string {
	var sb strings.Builder

	sb.WriteString("=== Constants ===\n")
	for i, c := range b.Constants {
		sb.WriteString(fmt.Sprintf("%4d: %s (%s)\n", i, c.String(), c.Type()))
	}

	sb.WriteString("\n=== Instructions ===\n")
	for i, instr := range b.Instructions {
		sb.WriteString(fmt.Sprintf("%4d: %s\n", i, instr.String()))
	}

	return sb.String()
}
