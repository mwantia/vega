package compiler

import (
	"fmt"

	"github.com/mwantia/vega/errors"
	"github.com/mwantia/vega/pkg/ast"
	"github.com/mwantia/vega/pkg/value"
)

// Compiler compiles an AST to bytecode.
type Compiler struct {
	bytecode *Bytecode
	errors   errors.ErrorList
}

// New creates a new Compiler.
func New() *Compiler {
	return &Compiler{
		bytecode: NewBytecode(),
	}
}

// Compile compiles a program AST to bytecode.
func (c *Compiler) Compile(program *ast.Program) (*Bytecode, error) {
	for _, stmt := range program.Statements {
		c.compileStatement(stmt)
	}

	if c.errors.HasErrors() {
		return nil, c.errors
	}

	return c.bytecode, nil
}

// CompileExpression compiles a single expression (for REPL).
func (c *Compiler) CompileExpression(expr ast.Expression) (*Bytecode, error) {
	c.compileExpression(expr)

	if c.errors.HasErrors() {
		return nil, c.errors
	}

	return c.bytecode, nil
}

func (c *Compiler) addError(msg string, line, col int) {
	c.errors.Add(errors.NewCompileError(msg, line, col))
}

// compileStatement compiles a statement node.
func (c *Compiler) compileStatement(stmt ast.Statement) {
	switch s := stmt.(type) {
	case *ast.ExpressionStatement:
		c.compileExpression(s.Expression)
		// Pop the result since it's not used
		c.bytecode.Emit(OpPop, s.Pos().Line)

	case *ast.AssignmentStatement:
		c.compileAssignment(s)

	case *ast.IndexAssignmentStatement:
		c.compileIndexAssignment(s)

	case *ast.IfStatement:
		c.compileIf(s)

	case *ast.ForStatement:
		c.compileFor(s)

	case *ast.WhileStatement:
		c.compileWhile(s)

	case *ast.FunctionDefinition:
		c.compileFunction(s)

	case *ast.ReturnStatement:
		c.compileReturn(s)

	case *ast.BreakStatement:
		c.compileBreak(s)

	case *ast.ContinueStatement:
		c.compileContinue(s)

	case *ast.BlockStatement:
		c.compileBlock(s)

	default:
		c.addError(fmt.Sprintf("unknown statement type: %T", stmt), 0, 0)
	}
}

func (c *Compiler) compileAssignment(s *ast.AssignmentStatement) {
	c.compileExpression(s.Value)
	c.bytecode.EmitName(OpStoreVar, s.Name.Value, s.Pos().Line)
}

func (c *Compiler) compileIndexAssignment(s *ast.IndexAssignmentStatement) {
	// Push: object, index, value
	c.compileExpression(s.Left.Left)
	c.compileExpression(s.Left.Index)
	c.compileExpression(s.Value)
	c.bytecode.Emit(OpSetIndex, s.Pos().Line)
}

func (c *Compiler) compileIf(s *ast.IfStatement) {
	line := s.Pos().Line

	// Compile condition
	c.compileExpression(s.Condition)

	// Jump to else/end if false
	jumpIfFalseAddr := c.bytecode.EmitArg(OpJmpIfFalse, 0, line)

	// Compile consequence
	c.compileBlock(s.Consequence)

	if s.Alternative != nil {
		// Jump over else block
		jumpOverElseAddr := c.bytecode.EmitArg(OpJmp, 0, line)

		// Patch the conditional jump to here (else block)
		c.bytecode.PatchJump(jumpIfFalseAddr)

		// Compile alternative
		c.compileBlock(s.Alternative)

		// Patch the jump over else
		c.bytecode.PatchJump(jumpOverElseAddr)
	} else {
		// No else block, patch jump to here
		c.bytecode.PatchJump(jumpIfFalseAddr)
	}
}

func (c *Compiler) compileFor(s *ast.ForStatement) {
	line := s.Pos().Line
	varName := s.Variable.Value

	// Compile iterable expression
	c.compileExpression(s.Iterable)

	// Initialize iterator
	c.bytecode.EmitName(OpIterInit, varName, line)

	// Loop start
	loopStart := c.bytecode.CurrentAddr()
	c.bytecode.PushLoop(loopStart)

	// Get next value, jump to end if done
	iterNextAddr := c.bytecode.EmitNameArg(OpIterNext, varName, 0, line)

	// Compile body
	c.compileBlock(s.Body)

	// Jump back to loop start
	c.bytecode.EmitArg(OpJmp, loopStart, line)

	// Patch iterator next to jump here when done
	c.bytecode.PatchJump(iterNextAddr)

	// Pop loop and patch breaks
	c.bytecode.PopLoop()
}

func (c *Compiler) compileWhile(s *ast.WhileStatement) {
	line := s.Pos().Line

	// Loop start
	loopStart := c.bytecode.CurrentAddr()
	c.bytecode.PushLoop(loopStart)

	// Compile condition
	c.compileExpression(s.Condition)

	// Jump to end if false
	jumpIfFalseAddr := c.bytecode.EmitArg(OpJmpIfFalse, 0, line)

	// Compile body
	c.compileBlock(s.Body)

	// Jump back to condition
	c.bytecode.EmitArg(OpJmp, loopStart, line)

	// Patch conditional jump to here
	c.bytecode.PatchJump(jumpIfFalseAddr)

	// Pop loop and patch breaks
	c.bytecode.PopLoop()
}

func (c *Compiler) compileFunction(s *ast.FunctionDefinition) {
	// For now, we store function definitions as constants
	// The VM will handle registering them
	line := s.Pos().Line

	// Compile function body into separate bytecode
	funcCompiler := New()

	// Note: Parameters are handled by the VM when calling the function.
	// The VM pops arguments and assigns them to the frame's locals before
	// executing the function body, so we don't emit STORE_VAR here.

	// Compile function body
	funcCompiler.compileBlock(s.Body)

	// Ensure there's a return
	funcCompiler.bytecode.Emit(OpReturn, line)

	// Store function info as a constant
	fn := &Function{
		Name:       s.Name.Value,
		Parameters: make([]string, len(s.Parameters)),
		Bytecode:   funcCompiler.bytecode,
	}
	for i, p := range s.Parameters {
		fn.Parameters[i] = p.Value
	}

	constIdx := c.bytecode.AddConstant(fn)
	c.bytecode.EmitArg(OpLoadConst, constIdx, line)
	c.bytecode.EmitName(OpStoreVar, s.Name.Value, line)
}

func (c *Compiler) compileReturn(s *ast.ReturnStatement) {
	line := s.Pos().Line

	if s.Value != nil {
		c.compileExpression(s.Value)
	} else {
		// Return nil
		constIdx := c.bytecode.AddConstant(value.Nil)
		c.bytecode.EmitArg(OpLoadConst, constIdx, line)
	}

	c.bytecode.Emit(OpReturn, line)
}

func (c *Compiler) compileBreak(s *ast.BreakStatement) {
	if !c.bytecode.InLoop() {
		c.addError("break outside of loop", s.Pos().Line, s.Pos().Column)
		return
	}

	// Emit jump (will be patched when loop ends)
	addr := c.bytecode.EmitArg(OpJmp, 0, s.Pos().Line)
	c.bytecode.AddBreak(addr)
}

func (c *Compiler) compileContinue(s *ast.ContinueStatement) {
	if !c.bytecode.InLoop() {
		c.addError("continue outside of loop", s.Pos().Line, s.Pos().Column)
		return
	}

	// Jump to loop start
	loopStart := c.bytecode.GetLoopStart()
	c.bytecode.EmitArg(OpJmp, loopStart, s.Pos().Line)
}

func (c *Compiler) compileBlock(block *ast.BlockStatement) {
	for _, stmt := range block.Statements {
		c.compileStatement(stmt)
	}
}

// compileExpression compiles an expression node.
func (c *Compiler) compileExpression(expr ast.Expression) {
	switch e := expr.(type) {
	case *ast.IntegerLiteral:
		constIdx := c.bytecode.AddConstant(value.NewInteger(e.Value))
		c.bytecode.EmitArg(OpLoadConst, constIdx, e.Pos().Line)

	case *ast.FloatLiteral:
		constIdx := c.bytecode.AddConstant(value.NewFloat(e.Value))
		c.bytecode.EmitArg(OpLoadConst, constIdx, e.Pos().Line)

	case *ast.StringLiteral:
		constIdx := c.bytecode.AddConstant(value.NewString(e.Value))
		c.bytecode.EmitArg(OpLoadConst, constIdx, e.Pos().Line)

	case *ast.BooleanLiteral:
		constIdx := c.bytecode.AddConstant(value.NewBoolean(e.Value))
		c.bytecode.EmitArg(OpLoadConst, constIdx, e.Pos().Line)

	case *ast.NilLiteral:
		constIdx := c.bytecode.AddConstant(value.Nil)
		c.bytecode.EmitArg(OpLoadConst, constIdx, e.Pos().Line)

	case *ast.Identifier:
		c.bytecode.EmitName(OpLoadVar, e.Value, e.Pos().Line)

	case *ast.ArrayLiteral:
		c.compileArray(e)

	case *ast.MapLiteral:
		c.compileMap(e)

	case *ast.PrefixExpression:
		c.compilePrefix(e)

	case *ast.InfixExpression:
		c.compileInfix(e)

	case *ast.CallExpression:
		c.compileCall(e)

	case *ast.MethodCallExpression:
		c.compileMethodCall(e)

	case *ast.IndexExpression:
		c.compileIndex(e)

	case *ast.AttributeExpression:
		c.compileAttribute(e)

	case *ast.GroupedExpression:
		c.compileExpression(e.Expression)

	case *ast.InterpolatedString:
		c.compileInterpolatedString(e)

	case *ast.PipeExpression:
		c.compilePipe(e)

	default:
		c.addError(fmt.Sprintf("unknown expression type: %T", expr), 0, 0)
	}
}

func (c *Compiler) compileArray(e *ast.ArrayLiteral) {
	// Push all elements
	for _, elem := range e.Elements {
		c.compileExpression(elem)
	}
	// Build array from stack elements
	c.bytecode.EmitArg(OpBuildArray, len(e.Elements), e.Pos().Line)
}

func (c *Compiler) compileMap(e *ast.MapLiteral) {
	// Push key-value pairs in order
	for _, key := range e.Order {
		val := e.Pairs[key]
		// Push key as string constant
		constIdx := c.bytecode.AddConstant(value.NewString(key))
		c.bytecode.EmitArg(OpLoadConst, constIdx, e.Pos().Line)
		// Push value
		c.compileExpression(val)
	}
	// Build map from stack elements
	c.bytecode.EmitArg(OpBuildMap, len(e.Pairs), e.Pos().Line)
}

func (c *Compiler) compilePrefix(e *ast.PrefixExpression) {
	c.compileExpression(e.Right)

	switch e.Operator {
	case "-":
		c.bytecode.Emit(OpNeg, e.Pos().Line)
	case "!":
		c.bytecode.Emit(OpNot, e.Pos().Line)
	default:
		c.addError(fmt.Sprintf("unknown prefix operator: %s", e.Operator), e.Pos().Line, e.Pos().Column)
	}
}

func (c *Compiler) compileInfix(e *ast.InfixExpression) {
	// Handle short-circuit evaluation for && and ||
	if e.Operator == "&&" {
		c.compileAnd(e)
		return
	}
	if e.Operator == "||" {
		c.compileOr(e)
		return
	}

	// Compile left and right operands
	c.compileExpression(e.Left)
	c.compileExpression(e.Right)

	line := e.Pos().Line

	switch e.Operator {
	case "+":
		c.bytecode.Emit(OpAdd, line)
	case "-":
		c.bytecode.Emit(OpSub, line)
	case "*":
		c.bytecode.Emit(OpMul, line)
	case "/":
		c.bytecode.Emit(OpDiv, line)
	case "%":
		c.bytecode.Emit(OpMod, line)
	case "==":
		c.bytecode.Emit(OpEq, line)
	case "!=":
		c.bytecode.Emit(OpNotEq, line)
	case "<":
		c.bytecode.Emit(OpLt, line)
	case "<=":
		c.bytecode.Emit(OpLte, line)
	case ">":
		c.bytecode.Emit(OpGt, line)
	case ">=":
		c.bytecode.Emit(OpGte, line)
	default:
		c.addError(fmt.Sprintf("unknown infix operator: %s", e.Operator), line, e.Pos().Column)
	}
}

func (c *Compiler) compileAnd(e *ast.InfixExpression) {
	line := e.Pos().Line

	// Compile left side
	c.compileExpression(e.Left)

	// If false, short-circuit (jump to end, leave false on stack)
	c.bytecode.Emit(OpDup, line)
	jumpIfFalseAddr := c.bytecode.EmitArg(OpJmpIfFalse, 0, line)

	// Left was true, pop it and evaluate right
	c.bytecode.Emit(OpPop, line)
	c.compileExpression(e.Right)

	// Patch jump to here
	c.bytecode.PatchJump(jumpIfFalseAddr)
}

func (c *Compiler) compileOr(e *ast.InfixExpression) {
	line := e.Pos().Line

	// Compile left side
	c.compileExpression(e.Left)

	// If true, short-circuit (jump to end, leave true on stack)
	c.bytecode.Emit(OpDup, line)
	jumpIfTrueAddr := c.bytecode.EmitArg(OpJmpIfTrue, 0, line)

	// Left was false, pop it and evaluate right
	c.bytecode.Emit(OpPop, line)
	c.compileExpression(e.Right)

	// Patch jump to here
	c.bytecode.PatchJump(jumpIfTrueAddr)
}

func (c *Compiler) compileCall(e *ast.CallExpression) {
	line := e.Pos().Line

	// Get function name
	var funcName string
	if ident, ok := e.Function.(*ast.Identifier); ok {
		funcName = ident.Value
	} else {
		// For now, only support simple function calls
		c.addError("only simple function calls are supported", line, e.Pos().Column)
		return
	}

	// Push arguments
	for _, arg := range e.Arguments {
		c.compileExpression(arg)
	}

	// Call function
	c.bytecode.EmitNameArg(OpCall, funcName, len(e.Arguments), line)
}

func (c *Compiler) compileMethodCall(e *ast.MethodCallExpression) {
	line := e.Pos().Line

	// Push object
	c.compileExpression(e.Object)

	// Push arguments
	for _, arg := range e.Arguments {
		c.compileExpression(arg)
	}

	// Call method
	c.bytecode.EmitNameArg(OpCallMethod, e.Method.Value, len(e.Arguments), line)
}

func (c *Compiler) compileIndex(e *ast.IndexExpression) {
	c.compileExpression(e.Left)
	c.compileExpression(e.Index)
	c.bytecode.Emit(OpIndex, e.Pos().Line)
}

func (c *Compiler) compileAttribute(e *ast.AttributeExpression) {
	c.compileExpression(e.Object)
	c.bytecode.EmitName(OpLoadAttr, e.Attribute.Value, e.Pos().Line)
}

func (c *Compiler) compileInterpolatedString(e *ast.InterpolatedString) {
	line := e.Pos().Line

	if len(e.Parts) == 0 {
		// Empty string
		constIdx := c.bytecode.AddConstant(value.NewString(""))
		c.bytecode.EmitArg(OpLoadConst, constIdx, line)
		return
	}

	// Compile first part
	c.compileExpression(e.Parts[0])

	// Concatenate remaining parts
	for i := 1; i < len(e.Parts); i++ {
		c.compileExpression(e.Parts[i])
		c.bytecode.Emit(OpConcat, line)
	}
}

func (c *Compiler) compilePipe(e *ast.PipeExpression) {
	// Compile left side (produces a value)
	c.compileExpression(e.Left)

	// The pipe operator passes the left value as input to the right
	// For now, we emit a special pipe opcode that the VM will handle
	c.compileExpression(e.Right)
	c.bytecode.Emit(OpPipe, e.Pos().Line)
}
