package compiler

import (
	"testing"

	"github.com/mwantia/vega/old/pkg/lexer"
	"github.com/mwantia/vega/old/pkg/parser"
	"github.com/mwantia/vega/old/pkg/value"
)

func compile(t *testing.T, input string) *Bytecode {
	t.Helper()

	l := lexer.New(input)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	p := parser.New(tokens)
	program, err := p.Parse()
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	c := New()
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	return bytecode
}

func TestIntegerLiteral(t *testing.T) {
	bc := compile(t, "42")

	// Should have one constant: 42
	if len(bc.Constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(bc.Constants))
	}

	intVal, ok := bc.Constants[0].(*value.Integer)
	if !ok {
		t.Fatalf("expected Integer, got %T", bc.Constants[0])
	}
	if intVal.Value != 42 {
		t.Errorf("expected 42, got %d", intVal.Value)
	}

	// Should have: LOAD_CONST 0, POP
	if len(bc.Instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(bc.Instructions))
	}

	if bc.Instructions[0].Op != OpLoadConst {
		t.Errorf("expected LOAD_CONST, got %s", bc.Instructions[0].Op)
	}
	if bc.Instructions[1].Op != OpPop {
		t.Errorf("expected POP, got %s", bc.Instructions[1].Op)
	}
}

func TestStringLiteral(t *testing.T) {
	bc := compile(t, `"hello"`)

	if len(bc.Constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(bc.Constants))
	}

	strVal, ok := bc.Constants[0].(*value.String)
	if !ok {
		t.Fatalf("expected String, got %T", bc.Constants[0])
	}
	if strVal.Value != "hello" {
		t.Errorf("expected 'hello', got %q", strVal.Value)
	}
}

func TestBooleanLiteral(t *testing.T) {
	bc := compile(t, "true")

	if len(bc.Constants) != 1 {
		t.Fatalf("expected 1 constant, got %d", len(bc.Constants))
	}

	boolVal, ok := bc.Constants[0].(*value.Boolean)
	if !ok {
		t.Fatalf("expected Boolean, got %T", bc.Constants[0])
	}
	if !boolVal.Value {
		t.Error("expected true")
	}
}

func TestAssignment(t *testing.T) {
	bc := compile(t, "x = 42")

	// Should have: LOAD_CONST 0, STORE_VAR "x"
	if len(bc.Instructions) != 2 {
		t.Fatalf("expected 2 instructions, got %d", len(bc.Instructions))
	}

	if bc.Instructions[0].Op != OpLoadConst {
		t.Errorf("expected LOAD_CONST, got %s", bc.Instructions[0].Op)
	}
	if bc.Instructions[1].Op != OpStoreVar {
		t.Errorf("expected STORE_VAR, got %s", bc.Instructions[1].Op)
	}
	if bc.Instructions[1].Name != "x" {
		t.Errorf("expected 'x', got %q", bc.Instructions[1].Name)
	}
}

func TestBinaryExpression(t *testing.T) {
	bc := compile(t, "1 + 2")

	// Constants: 1, 2
	if len(bc.Constants) != 2 {
		t.Fatalf("expected 2 constants, got %d", len(bc.Constants))
	}

	// Instructions: LOAD_CONST 0, LOAD_CONST 1, ADD, POP
	if len(bc.Instructions) != 4 {
		t.Fatalf("expected 4 instructions, got %d", len(bc.Instructions))
	}

	if bc.Instructions[2].Op != OpAdd {
		t.Errorf("expected ADD, got %s", bc.Instructions[2].Op)
	}
}

func TestComparisonExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected OpCode
	}{
		{"1 == 2", OpEq},
		{"1 != 2", OpNotEq},
		{"1 < 2", OpLt},
		{"1 <= 2", OpLte},
		{"1 > 2", OpGt},
		{"1 >= 2", OpGte},
	}

	for _, tt := range tests {
		bc := compile(t, tt.input)

		// Find the comparison opcode
		found := false
		for _, instr := range bc.Instructions {
			if instr.Op == tt.expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("input %q: expected %s opcode", tt.input, tt.expected)
		}
	}
}

func TestPrefixExpression(t *testing.T) {
	tests := []struct {
		input    string
		expected OpCode
	}{
		{"-5", OpNeg},
		{"!true", OpNot},
	}

	for _, tt := range tests {
		bc := compile(t, tt.input)

		found := false
		for _, instr := range bc.Instructions {
			if instr.Op == tt.expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("input %q: expected %s opcode", tt.input, tt.expected)
		}
	}
}

func TestFunctionCall(t *testing.T) {
	bc := compile(t, "print(42)")

	// Should have CALL instruction
	found := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpCall {
			found = true
			if instr.Name != "print" {
				t.Errorf("expected function name 'print', got %q", instr.Name)
			}
			if instr.Arg != 1 {
				t.Errorf("expected 1 argument, got %d", instr.Arg)
			}
			break
		}
	}
	if !found {
		t.Error("expected CALL instruction")
	}
}

func TestMethodCall(t *testing.T) {
	bc := compile(t, "vfs.stat(path)")

	// Should have: LOAD_VAR "vfs", LOAD_VAR "path", CALL_METHOD "stat"
	foundLoadVfs := false
	foundCallMethod := false

	for _, instr := range bc.Instructions {
		if instr.Op == OpLoadVar && instr.Name == "vfs" {
			foundLoadVfs = true
		}
		if instr.Op == OpCallMethod && instr.Name == "stat" {
			foundCallMethod = true
			if instr.Arg != 1 {
				t.Errorf("expected 1 argument, got %d", instr.Arg)
			}
		}
	}

	if !foundLoadVfs {
		t.Error("expected LOAD_VAR 'vfs'")
	}
	if !foundCallMethod {
		t.Error("expected CALL_METHOD 'stat'")
	}
}

func TestArrayLiteral(t *testing.T) {
	bc := compile(t, "[1, 2, 3]")

	// Should have BUILD_ARRAY with count 3
	found := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpBuildArray {
			found = true
			if instr.Arg != 3 {
				t.Errorf("expected 3 elements, got %d", instr.Arg)
			}
			break
		}
	}
	if !found {
		t.Error("expected BUILD_ARRAY instruction")
	}
}

func TestMapLiteral(t *testing.T) {
	bc := compile(t, `{name: "alice", age: 30}`)

	// Should have BUILD_MAP with count 2
	found := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpBuildMap {
			found = true
			if instr.Arg != 2 {
				t.Errorf("expected 2 pairs, got %d", instr.Arg)
			}
			break
		}
	}
	if !found {
		t.Error("expected BUILD_MAP instruction")
	}
}

func TestIndexExpression(t *testing.T) {
	bc := compile(t, "arr[0]")

	found := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpIndex {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected INDEX instruction")
	}
}

func TestIfStatement(t *testing.T) {
	bc := compile(t, "if true { x = 1 }")

	// Should have JMP_IF_FALSE
	found := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpJmpIfFalse {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected JMP_IF_FALSE instruction")
	}
}

func TestIfElseStatement(t *testing.T) {
	bc := compile(t, "if true { x = 1 } else { x = 2 }")

	// Should have JMP_IF_FALSE and JMP (for else skip)
	foundJmpIfFalse := false
	foundJmp := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpJmpIfFalse {
			foundJmpIfFalse = true
		}
		if instr.Op == OpJmp {
			foundJmp = true
		}
	}
	if !foundJmpIfFalse {
		t.Error("expected JMP_IF_FALSE instruction")
	}
	if !foundJmp {
		t.Error("expected JMP instruction")
	}
}

func TestWhileLoop(t *testing.T) {
	bc := compile(t, "while x < 10 { x = x + 1 }")

	// Should have JMP_IF_FALSE and JMP (back to start)
	foundJmpIfFalse := false
	foundJmp := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpJmpIfFalse {
			foundJmpIfFalse = true
		}
		if instr.Op == OpJmp {
			foundJmp = true
		}
	}
	if !foundJmpIfFalse {
		t.Error("expected JMP_IF_FALSE instruction")
	}
	if !foundJmp {
		t.Error("expected JMP instruction")
	}
}

func TestForLoop(t *testing.T) {
	bc := compile(t, "for i in [1, 2, 3] { print(i) }")

	// Should have ITER_INIT and ITER_NEXT
	foundIterInit := false
	foundIterNext := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpIterInit {
			foundIterInit = true
		}
		if instr.Op == OpIterNext {
			foundIterNext = true
		}
	}
	if !foundIterInit {
		t.Error("expected ITER_INIT instruction")
	}
	if !foundIterNext {
		t.Error("expected ITER_NEXT instruction")
	}
}

func TestAndExpression(t *testing.T) {
	bc := compile(t, "true && false")

	// Should have DUP, JMP_IF_FALSE (short-circuit)
	foundDup := false
	foundJmpIfFalse := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpDup {
			foundDup = true
		}
		if instr.Op == OpJmpIfFalse {
			foundJmpIfFalse = true
		}
	}
	if !foundDup {
		t.Error("expected DUP instruction for short-circuit")
	}
	if !foundJmpIfFalse {
		t.Error("expected JMP_IF_FALSE instruction for short-circuit")
	}
}

func TestOrExpression(t *testing.T) {
	bc := compile(t, "true || false")

	// Should have DUP, JMP_IF_TRUE (short-circuit)
	foundDup := false
	foundJmpIfTrue := false
	for _, instr := range bc.Instructions {
		if instr.Op == OpDup {
			foundDup = true
		}
		if instr.Op == OpJmpIfTrue {
			foundJmpIfTrue = true
		}
	}
	if !foundDup {
		t.Error("expected DUP instruction for short-circuit")
	}
	if !foundJmpIfTrue {
		t.Error("expected JMP_IF_TRUE instruction for short-circuit")
	}
}

func TestFunctionDefinition(t *testing.T) {
	bc := compile(t, "fn add(a, b) { return a + b }")

	// The function should be stored as a constant
	foundFunc := false
	for _, c := range bc.Constants {
		if _, ok := c.(*Function); ok {
			foundFunc = true
			break
		}
	}
	if !foundFunc {
		t.Error("expected function in constants")
	}
}

func TestReturnStatement(t *testing.T) {
	bc := compile(t, "fn foo() { return 42 }")

	// Find the function and check it has RETURN
	for _, c := range bc.Constants {
		if fn, ok := c.(*Function); ok {
			foundReturn := false
			for _, instr := range fn.Bytecode.Instructions {
				if instr.Op == OpReturn {
					foundReturn = true
					break
				}
			}
			if !foundReturn {
				t.Error("expected RETURN in function bytecode")
			}
		}
	}
}

func TestDisassemble(t *testing.T) {
	bc := compile(t, "x = 1 + 2")

	disasm := bc.Disassemble()
	if disasm == "" {
		t.Error("disassemble returned empty string")
	}

	// Check it contains expected parts
	if !contains(disasm, "LOAD_CONST") {
		t.Error("disassembly should contain LOAD_CONST")
	}
	if !contains(disasm, "ADD") {
		t.Error("disassembly should contain ADD")
	}
	if !contains(disasm, "STORE_VAR") {
		t.Error("disassembly should contain STORE_VAR")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
