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
	LoopStack    []LoopStack
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
			fmt.Fprintf(&sb, "%4d: %s\n", i, n.String())
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

func (b *ByteCode) EmitName(operation OperationCode, name string, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		Name:       name,
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

func (b *ByteCode) EmitNameArg(operation OperationCode, name string, arg int, sourceLine int) int {
	addr := len(b.Instructions)
	b.Instructions = append(b.Instructions, Instruction{
		Operation:  operation,
		Name:       name,
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
