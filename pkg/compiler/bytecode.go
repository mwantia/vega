package compiler

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/value"
)

// Constant holds a pre-encoded value for the constant pool.
type Constant struct {
	Tag  value.TypeTag
	Data []byte
}

type ByteCode struct {
	Instructions []Instruction
	Constants    []Constant
	Names        []string             // interned function names for OpCallNAT/OpCallFN
	LoopStack    []LoopStack
	Functions    map[string]*FunctionDef // user-defined functions compiled from this program
}

type LoopStack struct {
	StartAddress    int
	BreakAdress     []int
	ContinueAddress []int
}

func (b *ByteCode) Disassemble() string {
	var sb strings.Builder
	if len(b.Constants) > 0 {
		sb.WriteString("=== Constants ===\n")
		for i, c := range b.Constants {
			if name, ok := value.NameForTag(c.Tag); ok {
				hex := hex.EncodeToString(c.Data)
				fmt.Fprintf(&sb, "%4d: %s (%s)\n", i, hex, name)
			}
		}
	}

	if len(b.Instructions) > 0 {
		sb.WriteString("\n=== Instructions ===\n")
		for i, n := range b.Instructions {
			if n.Operation == OpCallNAT || n.Operation == OpCallFN {
				fmt.Fprintf(&sb, "%4d: %s %s argc=%d\n", i, n.Operation, b.Names[n.Offset], n.Argument)
			} else {
				fmt.Fprintf(&sb, "%4d: %s\n", i, n.String())
			}
		}
	}

	for name, fn := range b.Functions {
		fmt.Fprintf(&sb, "\n=== Function: %s ===\n", name)
		// Print the function body directly — do NOT call fn.ByteCode.Disassemble()
		// because function bytecodes share the top-level Functions map, which would
		// cause infinite recursion.
		if len(fn.ByteCode.Constants) > 0 {
			sb.WriteString("=== Constants ===\n")
			for i, c := range fn.ByteCode.Constants {
				if cname, ok := value.NameForTag(c.Tag); ok {
					fmt.Fprintf(&sb, "%4d: %s (%s)\n", i, hex.EncodeToString(c.Data), cname)
				}
			}
		}
		if len(fn.ByteCode.Instructions) > 0 {
			sb.WriteString("=== Instructions ===\n")
			for i, n := range fn.ByteCode.Instructions {
				if n.Operation == OpCallNAT || n.Operation == OpCallFN {
					fmt.Fprintf(&sb, "%4d: %s %s argc=%d\n", i, n.Operation, fn.ByteCode.Names[n.Offset], n.Argument)
				} else {
					fmt.Fprintf(&sb, "%4d: %s\n", i, n.String())
				}
			}
		}
	}

	return sb.String()
}

func (b *ByteCode) Emit(operation OperationCode, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		SourceLine: sourceLine,
	})
	return addr
}

func (b *ByteCode) EmitArg(operation OperationCode, arg int, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		Argument:   arg,
		SourceLine: sourceLine,
	})
	return addr
}

func (b *ByteCode) EmitArgExtra(operation OperationCode, arg int, extra byte, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		Argument:   arg,
		Extra:      extra,
		SourceLine: sourceLine,
	})
	return addr
}

func (b *ByteCode) EmitField(operation OperationCode, arg int, offset int, extra byte, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		Argument:   arg,
		Offset:     offset,
		Extra:      extra,
		SourceLine: sourceLine,
	})
	return addr
}

// EmitFieldSize is like EmitField but also sets Size — used for slice fields
// in OpFieldSTORE/OpFieldLOAD where the field width is the slice capacity, not
// derivable from SizeForTag.
func (b *ByteCode) EmitFieldSize(operation OperationCode, arg int, offset int, extra byte, size int, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		Argument:   arg,
		Offset:     offset,
		Extra:      extra,
		Size:       size,
		SourceLine: sourceLine,
	})
	return addr
}

// internName appends name to the Names table if not already present and
// returns its index. Used by EmitNameArg to store call target names.
func (b *ByteCode) internName(name string) int {
	for i, n := range b.Names {
		if n == name {
			return i
		}
	}
	idx := len(b.Names)
	b.Names = append(b.Names, name)
	return idx
}

func (b *ByteCode) EmitNameArg(operation OperationCode, name string, arg int, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		Offset:     b.internName(name),
		Argument:   arg,
		SourceLine: sourceLine,
	})
	return addr
}

// AddConstant adds a pre-encoded constant to the pool. Deduplicates by Tag + Data.
func (b *ByteCode) AddConstant(c Constant) int {
	for i, existing := range b.Constants {
		if existing.Tag == c.Tag && bytes.Equal(existing.Data, c.Data) {
			return i
		}
	}
	idx := len(b.Constants)
	b.Constants = append(b.Constants, c)
	return idx
}

func (b *ByteCode) CurrentAddr() int {
	return len(b.Instructions)
}

func (b *ByteCode) PatchJump(addr int) {
	b.Instructions[addr].Argument = b.CurrentAddr()
}

func (b *ByteCode) PatchJumpTo(addr, target int) {
	b.Instructions[addr].Argument = target
}

func (b *ByteCode) PushLoop(startAddr int) {
	b.LoopStack = append(b.LoopStack, LoopStack{
		StartAddress:    startAddr,
		BreakAdress:     make([]int, 0),
		ContinueAddress: make([]int, 0),
	})
}

func (b *ByteCode) PopLoop() {
	if len(b.LoopStack) == 0 {
		return
	}
	loop := b.LoopStack[len(b.LoopStack)-1]
	b.LoopStack = b.LoopStack[:len(b.LoopStack)-1]
	// Patch all break jumps to current address (after loop)
	for _, addr := range loop.BreakAdress {
		b.PatchJump(addr)
	}
	// Continue jumps are patched when emitted (to loop start)
}

func (b *ByteCode) AddBreak(addr int) {
	if len(b.LoopStack) > 0 {
		b.LoopStack[len(b.LoopStack)-1].BreakAdress = append(
			b.LoopStack[len(b.LoopStack)-1].BreakAdress, addr)
	}
}

func (b *ByteCode) GetLoopStart() int {
	if len(b.LoopStack) > 0 {
		return b.LoopStack[len(b.LoopStack)-1].StartAddress
	}
	return -1
}

func (b *ByteCode) InLoop() bool {
	return len(b.LoopStack) > 0
}
