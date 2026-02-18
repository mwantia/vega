package parser

import (
	"testing"

	"github.com/mwantia/vega/old/pkg/ast"
	"github.com/mwantia/vega/old/pkg/lexer"
)

func parse(t *testing.T, input string) *ast.Program {
	l := lexer.New(input)
	tokens, err := l.Tokenize()
	if err != nil {
		t.Fatalf("lexer error: %v", err)
	}

	p := New(tokens)
	program, err := p.Parse()
	if err != nil {
		t.Fatalf("parser error: %v", err)
	}

	return program
}

func TestIntegerLiteral(t *testing.T) {
	input := "42"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ExpressionStatement)
	if !ok {
		t.Fatalf("expected ExpressionStatement, got %T", program.Statements[0])
	}

	lit, ok := stmt.Expression.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", stmt.Expression)
	}

	if lit.Value != 42 {
		t.Errorf("expected value 42, got %d", lit.Value)
	}
}

func TestFloatLiteral(t *testing.T) {
	input := "3.14"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	lit, ok := stmt.Expression.(*ast.FloatLiteral)
	if !ok {
		t.Fatalf("expected FloatLiteral, got %T", stmt.Expression)
	}

	if lit.Value != 3.14 {
		t.Errorf("expected value 3.14, got %f", lit.Value)
	}
}

func TestStringLiteral(t *testing.T) {
	input := `"hello world"`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	lit, ok := stmt.Expression.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", stmt.Expression)
	}

	if lit.Value != "hello world" {
		t.Errorf("expected 'hello world', got %q", lit.Value)
	}
}

func TestBooleanLiteral(t *testing.T) {
	tests := []struct {
		input    string
		expected bool
	}{
		{"true", true},
		{"false", false},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		lit, ok := stmt.Expression.(*ast.BooleanLiteral)
		if !ok {
			t.Fatalf("expected BooleanLiteral, got %T", stmt.Expression)
		}

		if lit.Value != tt.expected {
			t.Errorf("expected %v, got %v", tt.expected, lit.Value)
		}
	}
}

func TestNilLiteral(t *testing.T) {
	input := "nil"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	_, ok := stmt.Expression.(*ast.NilLiteral)
	if !ok {
		t.Fatalf("expected NilLiteral, got %T", stmt.Expression)
	}
}

func TestIdentifier(t *testing.T) {
	input := "foobar"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	ident, ok := stmt.Expression.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", stmt.Expression)
	}

	if ident.Value != "foobar" {
		t.Errorf("expected 'foobar', got %q", ident.Value)
	}
}

func TestAssignment(t *testing.T) {
	input := "x = 42"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.AssignmentStatement)
	if !ok {
		t.Fatalf("expected AssignmentStatement, got %T", program.Statements[0])
	}

	if stmt.Name.Value != "x" {
		t.Errorf("expected 'x', got %q", stmt.Name.Value)
	}

	lit, ok := stmt.Value.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", stmt.Value)
	}

	if lit.Value != 42 {
		t.Errorf("expected 42, got %d", lit.Value)
	}
}

func TestPrefixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		operator string
		value    int
	}{
		{"-5", "-", 5},
		{"!5", "!", 5},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		prefix, ok := stmt.Expression.(*ast.PrefixExpression)
		if !ok {
			t.Fatalf("expected PrefixExpression, got %T", stmt.Expression)
		}

		if prefix.Operator != tt.operator {
			t.Errorf("expected operator %q, got %q", tt.operator, prefix.Operator)
		}

		lit, ok := prefix.Right.(*ast.IntegerLiteral)
		if !ok {
			t.Fatalf("expected IntegerLiteral, got %T", prefix.Right)
		}

		if lit.Value != tt.value {
			t.Errorf("expected %d, got %d", tt.value, lit.Value)
		}
	}
}

func TestInfixExpressions(t *testing.T) {
	tests := []struct {
		input    string
		left     int
		operator string
		right    int
	}{
		{"5 + 5", 5, "+", 5},
		{"5 - 5", 5, "-", 5},
		{"5 * 5", 5, "*", 5},
		{"5 / 5", 5, "/", 5},
		{"5 % 5", 5, "%", 5},
		{"5 > 5", 5, ">", 5},
		{"5 < 5", 5, "<", 5},
		{"5 == 5", 5, "==", 5},
		{"5 != 5", 5, "!=", 5},
		{"5 >= 5", 5, ">=", 5},
		{"5 <= 5", 5, "<=", 5},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		infix, ok := stmt.Expression.(*ast.InfixExpression)
		if !ok {
			t.Fatalf("expected InfixExpression for %q, got %T", tt.input, stmt.Expression)
		}

		testIntegerLiteral(t, infix.Left, tt.left)

		if infix.Operator != tt.operator {
			t.Errorf("expected operator %q, got %q", tt.operator, infix.Operator)
		}

		testIntegerLiteral(t, infix.Right, tt.right)
	}
}

func TestOperatorPrecedence(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"1 + 2 * 3", "(1 + (2 * 3))"},
		{"1 * 2 + 3", "((1 * 2) + 3)"},
		{"1 + 2 + 3", "((1 + 2) + 3)"},
		{"-1 + 2", "((-1) + 2)"},
		{"!true == false", "((!true) == false)"},
		{"1 < 2 == true", "((1 < 2) == true)"},
		{"a + b * c + d / e - f", "(((a + (b * c)) + (d / e)) - f)"},
		{"1 + (2 + 3) + 4", "((1 + ((2 + 3))) + 4)"},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		actual := stmt.Expression.String()

		if actual != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, actual)
		}
	}
}

func TestCallExpression(t *testing.T) {
	input := "add(1, 2 * 3, 4 + 5)"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call, ok := stmt.Expression.(*ast.CallExpression)
	if !ok {
		t.Fatalf("expected CallExpression, got %T", stmt.Expression)
	}

	ident, ok := call.Function.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", call.Function)
	}

	if ident.Value != "add" {
		t.Errorf("expected 'add', got %q", ident.Value)
	}

	if len(call.Arguments) != 3 {
		t.Fatalf("expected 3 arguments, got %d", len(call.Arguments))
	}
}

func TestMethodCallExpression(t *testing.T) {
	input := "vfs.stat(path)"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	call, ok := stmt.Expression.(*ast.MethodCallExpression)
	if !ok {
		t.Fatalf("expected MethodCallExpression, got %T", stmt.Expression)
	}

	obj, ok := call.Object.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier for object, got %T", call.Object)
	}

	if obj.Value != "vfs" {
		t.Errorf("expected 'vfs', got %q", obj.Value)
	}

	if call.Method.Value != "stat" {
		t.Errorf("expected 'stat', got %q", call.Method.Value)
	}

	if len(call.Arguments) != 1 {
		t.Fatalf("expected 1 argument, got %d", len(call.Arguments))
	}
}

func TestIndexExpression(t *testing.T) {
	input := "arr[0]"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	index, ok := stmt.Expression.(*ast.IndexExpression)
	if !ok {
		t.Fatalf("expected IndexExpression, got %T", stmt.Expression)
	}

	ident, ok := index.Left.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", index.Left)
	}

	if ident.Value != "arr" {
		t.Errorf("expected 'arr', got %q", ident.Value)
	}

	lit, ok := index.Index.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", index.Index)
	}

	if lit.Value != 0 {
		t.Errorf("expected 0, got %d", lit.Value)
	}
}

func TestArrayLiteral(t *testing.T) {
	input := "[1, 2, 3]"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	arr, ok := stmt.Expression.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", stmt.Expression)
	}

	if len(arr.Elements) != 3 {
		t.Fatalf("expected 3 elements, got %d", len(arr.Elements))
	}

	testIntegerLiteral(t, arr.Elements[0], 1)
	testIntegerLiteral(t, arr.Elements[1], 2)
	testIntegerLiteral(t, arr.Elements[2], 3)
}

func TestMapLiteral(t *testing.T) {
	input := `{name: "alice", age: 30}`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	m, ok := stmt.Expression.(*ast.MapLiteral)
	if !ok {
		t.Fatalf("expected MapLiteral, got %T", stmt.Expression)
	}

	if len(m.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(m.Pairs))
	}

	nameVal, ok := m.Pairs["name"]
	if !ok {
		t.Fatal("expected 'name' key")
	}

	str, ok := nameVal.(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", nameVal)
	}

	if str.Value != "alice" {
		t.Errorf("expected 'alice', got %q", str.Value)
	}
}

func TestIfStatement(t *testing.T) {
	input := `if x > 10 { y = 1 }`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("expected IfStatement, got %T", program.Statements[0])
	}

	infix, ok := stmt.Condition.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", stmt.Condition)
	}

	if infix.Operator != ">" {
		t.Errorf("expected '>', got %q", infix.Operator)
	}

	if len(stmt.Consequence.Statements) != 1 {
		t.Fatalf("expected 1 consequence statement, got %d", len(stmt.Consequence.Statements))
	}

	if stmt.Alternative != nil {
		t.Error("expected no alternative")
	}
}

func TestIfElseStatement(t *testing.T) {
	input := `if x > 10 { y = 1 } else { y = 2 }`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.IfStatement)
	if !ok {
		t.Fatalf("expected IfStatement, got %T", program.Statements[0])
	}

	if stmt.Alternative == nil {
		t.Fatal("expected alternative")
	}

	if len(stmt.Alternative.Statements) != 1 {
		t.Fatalf("expected 1 alternative statement, got %d", len(stmt.Alternative.Statements))
	}
}

func TestForStatement(t *testing.T) {
	input := `for i in [1, 2, 3] { print(i) }`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("expected ForStatement, got %T", program.Statements[0])
	}

	if stmt.Variable.Value != "i" {
		t.Errorf("expected 'i', got %q", stmt.Variable.Value)
	}

	_, ok = stmt.Iterable.(*ast.ArrayLiteral)
	if !ok {
		t.Fatalf("expected ArrayLiteral, got %T", stmt.Iterable)
	}

	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(stmt.Body.Statements))
	}
}

func TestWhileStatement(t *testing.T) {
	input := `while x < 10 { x = x + 1 }`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt, ok := program.Statements[0].(*ast.WhileStatement)
	if !ok {
		t.Fatalf("expected WhileStatement, got %T", program.Statements[0])
	}

	infix, ok := stmt.Condition.(*ast.InfixExpression)
	if !ok {
		t.Fatalf("expected InfixExpression, got %T", stmt.Condition)
	}

	if infix.Operator != "<" {
		t.Errorf("expected '<', got %q", infix.Operator)
	}

	if len(stmt.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(stmt.Body.Statements))
	}
}

func TestFunctionDefinition(t *testing.T) {
	input := `fn add(a, b) { return a + b }`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	fn, ok := program.Statements[0].(*ast.FunctionDefinition)
	if !ok {
		t.Fatalf("expected FunctionDefinition, got %T", program.Statements[0])
	}

	if fn.Name.Value != "add" {
		t.Errorf("expected 'add', got %q", fn.Name.Value)
	}

	if len(fn.Parameters) != 2 {
		t.Fatalf("expected 2 parameters, got %d", len(fn.Parameters))
	}

	if fn.Parameters[0].Value != "a" {
		t.Errorf("expected 'a', got %q", fn.Parameters[0].Value)
	}

	if fn.Parameters[1].Value != "b" {
		t.Errorf("expected 'b', got %q", fn.Parameters[1].Value)
	}

	if len(fn.Body.Statements) != 1 {
		t.Fatalf("expected 1 body statement, got %d", len(fn.Body.Statements))
	}
}

func TestReturnStatement(t *testing.T) {
	tests := []struct {
		input    string
		hasValue bool
	}{
		{"return 42", true},
		{"return", false},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		ret, ok := program.Statements[0].(*ast.ReturnStatement)
		if !ok {
			t.Fatalf("expected ReturnStatement, got %T", program.Statements[0])
		}

		if tt.hasValue && ret.Value == nil {
			t.Error("expected return value")
		}

		if !tt.hasValue && ret.Value != nil {
			t.Error("expected no return value")
		}
	}
}

func TestBreakContinue(t *testing.T) {
	input := `
for i in [1, 2, 3] {
    if i == 2 {
        break
    }
    continue
}
`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	forStmt, ok := program.Statements[0].(*ast.ForStatement)
	if !ok {
		t.Fatalf("expected ForStatement, got %T", program.Statements[0])
	}

	if len(forStmt.Body.Statements) != 2 {
		t.Fatalf("expected 2 body statements, got %d", len(forStmt.Body.Statements))
	}
}

func TestInterpolatedString(t *testing.T) {
	input := `"Hello ${name}!"`
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	interp, ok := stmt.Expression.(*ast.InterpolatedString)
	if !ok {
		t.Fatalf("expected InterpolatedString, got %T", stmt.Expression)
	}

	if len(interp.Parts) != 3 {
		t.Fatalf("expected 3 parts, got %d", len(interp.Parts))
	}

	// First part should be "Hello "
	part1, ok := interp.Parts[0].(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", interp.Parts[0])
	}
	if part1.Value != "Hello " {
		t.Errorf("expected 'Hello ', got %q", part1.Value)
	}

	// Second part should be identifier 'name'
	part2, ok := interp.Parts[1].(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", interp.Parts[1])
	}
	if part2.Value != "name" {
		t.Errorf("expected 'name', got %q", part2.Value)
	}

	// Third part should be "!"
	part3, ok := interp.Parts[2].(*ast.StringLiteral)
	if !ok {
		t.Fatalf("expected StringLiteral, got %T", interp.Parts[2])
	}
	if part3.Value != "!" {
		t.Errorf("expected '!', got %q", part3.Value)
	}
}

func TestAttributeExpression(t *testing.T) {
	input := "obj.name"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	attr, ok := stmt.Expression.(*ast.AttributeExpression)
	if !ok {
		t.Fatalf("expected AttributeExpression, got %T", stmt.Expression)
	}

	obj, ok := attr.Object.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier, got %T", attr.Object)
	}

	if obj.Value != "obj" {
		t.Errorf("expected 'obj', got %q", obj.Value)
	}

	if attr.Attribute.Value != "name" {
		t.Errorf("expected 'name', got %q", attr.Attribute.Value)
	}
}

func TestLogicalOperators(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"a && b", "(a && b)"},
		{"a || b", "(a || b)"},
		{"a && b || c", "((a && b) || c)"},
		{"a || b && c", "(a || (b && c))"},
	}

	for _, tt := range tests {
		program := parse(t, tt.input)

		if len(program.Statements) != 1 {
			t.Fatalf("expected 1 statement, got %d", len(program.Statements))
		}

		stmt := program.Statements[0].(*ast.ExpressionStatement)
		actual := stmt.Expression.String()

		if actual != tt.expected {
			t.Errorf("input %q: expected %q, got %q", tt.input, tt.expected, actual)
		}
	}
}

func TestPipeExpression(t *testing.T) {
	input := "a | b"
	program := parse(t, input)

	if len(program.Statements) != 1 {
		t.Fatalf("expected 1 statement, got %d", len(program.Statements))
	}

	stmt := program.Statements[0].(*ast.ExpressionStatement)
	pipe, ok := stmt.Expression.(*ast.PipeExpression)
	if !ok {
		t.Fatalf("expected PipeExpression, got %T", stmt.Expression)
	}

	left, ok := pipe.Left.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier for left, got %T", pipe.Left)
	}

	if left.Value != "a" {
		t.Errorf("expected 'a', got %q", left.Value)
	}

	right, ok := pipe.Right.(*ast.Identifier)
	if !ok {
		t.Fatalf("expected Identifier for right, got %T", pipe.Right)
	}

	if right.Value != "b" {
		t.Errorf("expected 'b', got %q", right.Value)
	}
}

func testIntegerLiteral(t *testing.T, expr ast.Expression, expected int) {
	t.Helper()

	lit, ok := expr.(*ast.IntegerLiteral)
	if !ok {
		t.Fatalf("expected IntegerLiteral, got %T", expr)
	}

	if lit.Value != expected {
		t.Errorf("expected %d, got %d", expected, lit.Value)
	}
}
