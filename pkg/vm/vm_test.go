package vm

import (
	"bytes"
	"testing"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/lexer"
	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/value"
)

func runVirtualMachine(t *testing.T, input string) *VirtualMachine {
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

	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	vm := NewVirtualMachine()
	_, err = vm.Run(bytecode)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	return vm
}

func runVirtualMachineWithOutput(t *testing.T, input string) (*VirtualMachine, string) {
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

	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	var buf bytes.Buffer
	vm := NewVirtualMachine()
	vm.SetStdout(&buf)
	_, err = vm.Run(bytecode)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	return vm, buf.String()
}

func TestIntegerArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"x = 1 + 2", 3},
		{"x = 5 - 3", 2},
		{"x = 4 * 3", 12},
		{"x = 10 / 2", 5},
		{"x = 10 % 3", 1},
		{"x = -5", -5},
		{"x = 1 + 2 * 3", 7},
		{"x = (1 + 2) * 3", 9},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		intVal, ok := val.(*value.IntValue)
		if !ok {
			t.Fatalf("input %q: expected IntValue, got %T", tt.input, val)
		}
		if intVal.Val != tt.expected {
			t.Errorf("input %q: expected %d, got %d", tt.input, tt.expected, intVal.Val)
		}
	}
}

func TestFloatArithmetic(t *testing.T) {
	tests := []struct {
		input    string
		expected float64
	}{
		{"x = 1.5 + 2.5", 4.0},
		{"x = 5.0 - 3.0", 2.0},
		{"x = 2.0 * 3.0", 6.0},
		{"x = 10.0 / 4.0", 2.5},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		floatVal, ok := val.(*value.FloatValue)
		if !ok {
			t.Fatalf("input %q: expected FloatValue, got %T", tt.input, val)
		}
		if floatVal.Val != tt.expected {
			t.Errorf("input %q: expected %f, got %f", tt.input, tt.expected, floatVal.Val)
		}
	}
}

func TestComparison(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"x = 1 == 1", true},
		{"x = 1 == 2", false},
		{"x = 1 != 2", true},
		{"x = 1 != 1", false},
		{"x = 1 < 2", true},
		{"x = 2 < 1", false},
		{"x = 1 <= 1", true},
		{"x = 1 <= 2", true},
		{"x = 2 > 1", true},
		{"x = 1 > 2", false},
		{"x = 1 >= 1", true},
		{"x = 2 >= 1", true},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		boolVal, ok := val.(*value.BoolValue)
		if !ok {
			t.Fatalf("input %q: expected BoolValue, got %T", tt.input, val)
		}
		if boolVal.Val != tt.expected {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.expected, boolVal.Val)
		}
	}
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"x = true && true", true},
		{"x = true && false", false},
		{"x = false && true", false},
		{"x = false || true", true},
		{"x = true || false", true},
		{"x = false || false", false},
		{"x = !true", false},
		{"x = !false", true},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		boolVal, ok := val.(*value.BoolValue)
		if !ok {
			t.Fatalf("input %q: expected BoolValue, got %T", tt.input, val)
		}
		if boolVal.Val != tt.expected {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.expected, boolVal.Val)
		}
	}
}

func TestStringConcatenation(t *testing.T) {
	vm := runVirtualMachine(t, `x = "hello" + " " + "world"`)
	val, ok := vm.GetGlobal("x")
	if !ok {
		t.Fatal("variable 'x' not found")
	}
	strVal, ok := val.(*value.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", val)
	}
	if strVal.Val != "hello world" {
		t.Errorf("expected 'hello world', got %q", strVal.Val)
	}
}

func TestIfStatement(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{"x = 0\nif true { x = 1 }", 1},
		{"x = 0\nif false { x = 1 }", 0},
		{"x = 0\nif true { x = 1 } else { x = 2 }", 1},
		{"x = 0\nif false { x = 1 } else { x = 2 }", 2},
		{"x = 0\nif 1 < 2 { x = 1 }", 1},
		{"x = 0\nif 2 < 1 { x = 1 }", 0},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		intVal, ok := val.(*value.IntValue)
		if !ok {
			t.Fatalf("input %q: expected IntValue, got %T", tt.input, val)
		}
		if intVal.Val != tt.expected {
			t.Errorf("input %q: expected %d, got %d", tt.input, tt.expected, intVal.Val)
		}
	}
}

func TestWhileLoop(t *testing.T) {
	vm := runVirtualMachine(t, `
		x = 0
		while x < 5 {
			x = x + 1
		}
	`)
	val, ok := vm.GetGlobal("x")
	if !ok {
		t.Fatal("variable 'x' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	if intVal.Val != 5 {
		t.Errorf("expected 5, got %d", intVal.Val)
	}
}

func TestForLoop(t *testing.T) {
	vm := runVirtualMachine(t, `
		sum = 0
		for i in [1, 2, 3, 4, 5] {
			sum = sum + i
		}
	`)
	val, ok := vm.GetGlobal("sum")
	if !ok {
		t.Fatal("variable 'sum' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	if intVal.Val != 15 {
		t.Errorf("expected 15, got %d", intVal.Val)
	}
}

func TestArray(t *testing.T) {
	vm := runVirtualMachine(t, `
		arr = [1, 2, 3]
		x = arr[0]
		y = arr[1]
		z = arr[2]
	`)

	tests := []struct {
		name     string
		expected int64
	}{
		{"x", 1},
		{"y", 2},
		{"z", 3},
	}

	for _, tt := range tests {
		val, ok := vm.GetGlobal(tt.name)
		if !ok {
			t.Fatalf("variable '%s' not found", tt.name)
		}
		intVal, ok := val.(*value.IntValue)
		if !ok {
			t.Fatalf("expected IntValue for %s, got %T", tt.name, val)
		}
		if intVal.Val != tt.expected {
			t.Errorf("%s: expected %d, got %d", tt.name, tt.expected, intVal.Val)
		}
	}
}

func TestMap(t *testing.T) {
	vm := runVirtualMachine(t, `
		m = {name: "alice", age: 30}
		x = m["name"]
		y = m["age"]
	`)

	val, ok := vm.GetGlobal("x")
	if !ok {
		t.Fatal("variable 'x' not found")
	}
	strVal, ok := val.(*value.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", val)
	}
	if strVal.Val != "alice" {
		t.Errorf("expected 'alice', got %q", strVal.Val)
	}

	val, ok = vm.GetGlobal("y")
	if !ok {
		t.Fatal("variable 'y' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	if intVal.Val != 30 {
		t.Errorf("expected 30, got %d", intVal.Val)
	}
}

func TestFunctionCall(t *testing.T) {
	_, output := runVirtualMachineWithOutput(t, `println("hello", "world")`)
	if output != "hello world\n" {
		t.Errorf("expected 'hello world\\n', got %q", output)
	}
}

func TestBuiltinLen(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
	}{
		{`x = len("hello")`, 5},
		{`x = len([1, 2, 3])`, 3},
		{`x = len({a: 1, b: 2})`, 2},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		intVal, ok := val.(*value.IntValue)
		if !ok {
			t.Fatalf("input %q: expected IntValue, got %T", tt.input, val)
		}
		if intVal.Val != tt.expected {
			t.Errorf("input %q: expected %d, got %d", tt.input, tt.expected, intVal.Val)
		}
	}
}

func TestBuiltinStringFunctions(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{`x = upper("hello")`, "HELLO"},
		{`x = lower("HELLO")`, "hello"},
		{`x = trim("  hello  ")`, "hello"},
		{`x = replace("hello", "l", "L")`, "heLLo"},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		strVal, ok := val.(*value.StringValue)
		if !ok {
			t.Fatalf("input %q: expected StringValue, got %T", tt.input, val)
		}
		if strVal.Val != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, strVal.Val)
		}
	}
}

func TestBuiltinSplitJoin(t *testing.T) {
	vm := runVirtualMachine(t, `
		parts = split("a,b,c", ",")
		result = join(parts, "-")
	`)

	val, ok := vm.GetGlobal("result")
	if !ok {
		t.Fatal("variable 'result' not found")
	}
	strVal, ok := val.(*value.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", val)
	}
	if strVal.Val != "a-b-c" {
		t.Errorf("expected 'a-b-c', got %q", strVal.Val)
	}
}

func TestBuiltinContains(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{`x = contains("hello", "ell")`, true},
		{`x = contains("hello", "xyz")`, false},
		{`x = contains([1, 2, 3], 2)`, true},
		{`x = contains([1, 2, 3], 5)`, false},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		boolVal, ok := val.(*value.BoolValue)
		if !ok {
			t.Fatalf("input %q: expected BoolValue, got %T", tt.input, val)
		}
		if boolVal.Val != tt.expected {
			t.Errorf("input %q: expected %v, got %v", tt.input, tt.expected, boolVal.Val)
		}
	}
}

func TestBuiltinRange(t *testing.T) {
	vm := runVirtualMachine(t, `
		sum = 0
		for i in range(5) {
			sum = sum + i
		}
	`)

	val, ok := vm.GetGlobal("sum")
	if !ok {
		t.Fatal("variable 'sum' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	// 0 + 1 + 2 + 3 + 4 = 10
	if intVal.Val != 10 {
		t.Errorf("expected 10, got %d", intVal.Val)
	}
}

func TestUserDefinedFunction(t *testing.T) {
	vm := runVirtualMachine(t, `
		fn add(a, b) {
			return a + b
		}
		result = add(3, 4)
	`)

	val, ok := vm.GetGlobal("result")
	if !ok {
		t.Fatal("variable 'result' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	if intVal.Val != 7 {
		t.Errorf("expected 7, got %d", intVal.Val)
	}
}

func TestRecursiveFunction(t *testing.T) {
	vm := runVirtualMachine(t, `
		fn factorial(n) {
			if n <= 1 {
				return 1
			}
			return n * factorial(n - 1)
		}
		result = factorial(5)
	`)

	val, ok := vm.GetGlobal("result")
	if !ok {
		t.Fatal("variable 'result' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	// 5! = 120
	if intVal.Val != 120 {
		t.Errorf("expected 120, got %d", intVal.Val)
	}
}

func TestTypeConversion(t *testing.T) {
	tests := []struct {
		input    string
		check    func(value.Value) bool
		expected string
	}{
		{`x = int("42")`, func(v value.Value) bool {
			iv, ok := v.(*value.IntValue)
			return ok && iv.Val == 42
		}, "42"},
		{`x = float("3.14")`, func(v value.Value) bool {
			fv, ok := v.(*value.FloatValue)
			return ok && fv.Val == 3.14
		}, "3.14"},
		{`x = string(42)`, func(v value.Value) bool {
			sv, ok := v.(*value.StringValue)
			return ok && sv.Val == "42"
		}, "42"},
		{`x = bool(1)`, func(v value.Value) bool {
			bv, ok := v.(*value.BoolValue)
			return ok && bv.Val == true
		}, "true"},
		{`x = bool(0)`, func(v value.Value) bool {
			bv, ok := v.(*value.BoolValue)
			return ok && bv.Val == false
		}, "false"},
	}

	for _, tt := range tests {
		vm := runVirtualMachine(t, tt.input)
		val, ok := vm.GetGlobal("x")
		if !ok {
			t.Fatalf("input %q: variable 'x' not found", tt.input)
		}
		if !tt.check(val) {
			t.Errorf("input %q: unexpected value %v", tt.input, val)
		}
	}
}

func TestArrayMethods(t *testing.T) {
	vm := runVirtualMachine(t, `
		arr = [1, 2, 3]
		push(arr, 4)
		last = pop(arr)
		length = len(arr)
	`)

	// Check last popped value
	val, ok := vm.GetGlobal("last")
	if !ok {
		t.Fatal("variable 'last' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	if intVal.Val != 4 {
		t.Errorf("expected last=4, got %d", intVal.Val)
	}

	// Check length after pop
	val, ok = vm.GetGlobal("length")
	if !ok {
		t.Fatal("variable 'length' not found")
	}
	intVal, ok = val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	if intVal.Val != 3 {
		t.Errorf("expected length=3, got %d", intVal.Val)
	}
}

func TestShortCircuitEvaluation(t *testing.T) {
	// Test that && short-circuits
	vm := runVirtualMachine(t, `
		called = false
		fn setTrue() {
			called = true
			return true
		}
		result = false && setTrue()
	`)

	val, ok := vm.GetGlobal("called")
	if !ok {
		t.Fatal("variable 'called' not found")
	}
	boolVal, ok := val.(*value.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", val)
	}
	if boolVal.Val != false {
		t.Error("expected 'called' to be false (short-circuit)")
	}

	// Test that || short-circuits
	vm = runVirtualMachine(t, `
		called = false
		fn setTrue() {
			called = true
			return true
		}
		result = true || setTrue()
	`)

	val, ok = vm.GetGlobal("called")
	if !ok {
		t.Fatal("variable 'called' not found")
	}
	boolVal, ok = val.(*value.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", val)
	}
	if boolVal.Val != false {
		t.Error("expected 'called' to be false (short-circuit)")
	}
}

func TestNestedLoops(t *testing.T) {
	vm := runVirtualMachine(t, `
		result = 0
		for i in [1, 2, 3] {
			for j in [10, 20] {
				result = result + i * j
			}
		}
	`)

	val, ok := vm.GetGlobal("result")
	if !ok {
		t.Fatal("variable 'result' not found")
	}
	intVal, ok := val.(*value.IntValue)
	if !ok {
		t.Fatalf("expected IntValue, got %T", val)
	}
	// (1*10 + 1*20) + (2*10 + 2*20) + (3*10 + 3*20) = 30 + 60 + 90 = 180
	if intVal.Val != 180 {
		t.Errorf("expected 180, got %d", intVal.Val)
	}
}

func TestSysStdoutWrite(t *testing.T) {
	var buf bytes.Buffer
	vm := NewVirtualMachine()
	vm.SetStdout(&buf)

	// Compile and run: sys.stdout.write("hello")
	input := `stdout().write("hello")`

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

	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	_, err = vm.Run(bytecode)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	if buf.String() != "hello" {
		t.Errorf("expected 'hello', got %q", buf.String())
	}
}

func TestSysStdoutWriteln(t *testing.T) {
	var buf bytes.Buffer
	vm := NewVirtualMachine()
	vm.SetStdout(&buf)

	input := `stdout().writeln("world")`

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

	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	_, err = vm.Run(bytecode)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	if buf.String() != "world\n" {
		t.Errorf("expected 'world\\n', got %q", buf.String())
	}
}

func TestSysStdinRead(t *testing.T) {
	input := `data = stdin().readln()`

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

	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	vm := NewVirtualMachine()
	vm.SetStdin(bytes.NewBufferString("test input\n"))

	_, err = vm.Run(bytecode)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	val, ok := vm.GetGlobal("data")
	if !ok {
		t.Fatal("variable 'data' not found")
	}

	strVal, ok := val.(*value.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", val)
	}

	if strVal.Val != "test input" {
		t.Errorf("expected 'test input', got %q", strVal.Val)
	}
}

func TestStreamProperties(t *testing.T) {
	input := `
		closed = stdin().isClosed()
		name = stdin().name
	`

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

	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		t.Fatalf("compiler error: %v", err)
	}

	vm := NewVirtualMachine()
	vm.SetStdin(bytes.NewBufferString(""))

	_, err = vm.Run(bytecode)
	if err != nil {
		t.Fatalf("runtime error: %v", err)
	}

	// Check closed property (should be false initially)
	val, ok := vm.GetGlobal("closed")
	if !ok {
		t.Fatal("variable 'closed' not found")
	}
	boolVal, ok := val.(*value.BoolValue)
	if !ok {
		t.Fatalf("expected BoolValue, got %T", val)
	}
	if boolVal.Val != false {
		t.Error("expected stdin.isClosed() to be false")
	}

	// Check name property
	val, ok = vm.GetGlobal("name")
	if !ok {
		t.Fatal("variable 'name' not found")
	}
	strVal, ok := val.(*value.StringValue)
	if !ok {
		t.Fatalf("expected StringValue, got %T", val)
	}
	if strVal.Val != "stdin" {
		t.Errorf("expected 'stdin', got %q", strVal.Val)
	}
}
