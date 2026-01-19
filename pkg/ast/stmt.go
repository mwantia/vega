package ast

import (
	"strings"

	"github.com/mwantia/vega/pkg/token"
)

// ExpressionStatement wraps an expression as a statement.
type ExpressionStatement struct {
	Token      token.Token // the first token of the expression
	Expression Expression
}

func (es *ExpressionStatement) statementNode()       {}
func (es *ExpressionStatement) TokenLiteral() string { return es.Token.Literal }
func (es *ExpressionStatement) Pos() token.Position  { return es.Token.Pos }
func (es *ExpressionStatement) String() string {
	if es.Expression != nil {
		return es.Expression.String()
	}
	return ""
}

// AssignmentStatement represents an assignment like x = 42.
type AssignmentStatement struct {
	Token token.Token // the '=' token
	Name  *Identifier
	Value Expression
}

func (as *AssignmentStatement) statementNode()       {}
func (as *AssignmentStatement) TokenLiteral() string { return as.Token.Literal }
func (as *AssignmentStatement) Pos() token.Position  { return as.Name.Pos() }
func (as *AssignmentStatement) String() string {
	return as.Name.String() + " = " + as.Value.String()
}

// IndexAssignmentStatement represents an index assignment like arr[0] = 42.
type IndexAssignmentStatement struct {
	Token token.Token // the '=' token
	Left  *IndexExpression
	Value Expression
}

func (ias *IndexAssignmentStatement) statementNode()       {}
func (ias *IndexAssignmentStatement) TokenLiteral() string { return ias.Token.Literal }
func (ias *IndexAssignmentStatement) Pos() token.Position  { return ias.Left.Pos() }
func (ias *IndexAssignmentStatement) String() string {
	return ias.Left.String() + " = " + ias.Value.String()
}

// BlockStatement represents a block of statements { ... }.
type BlockStatement struct {
	Token      token.Token // the '{' token
	Statements []Statement
}

func (bs *BlockStatement) statementNode()       {}
func (bs *BlockStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BlockStatement) Pos() token.Position  { return bs.Token.Pos }
func (bs *BlockStatement) String() string {
	var out strings.Builder
	out.WriteString("{\n")
	for _, s := range bs.Statements {
		out.WriteString("  ")
		out.WriteString(s.String())
		out.WriteString("\n")
	}
	out.WriteString("}")
	return out.String()
}

// IfStatement represents an if/else statement.
type IfStatement struct {
	Token       token.Token     // the 'if' token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement // optional else block
}

func (is *IfStatement) statementNode()       {}
func (is *IfStatement) TokenLiteral() string { return is.Token.Literal }
func (is *IfStatement) Pos() token.Position  { return is.Token.Pos }
func (is *IfStatement) String() string {
	var out strings.Builder
	out.WriteString("if ")
	out.WriteString(is.Condition.String())
	out.WriteString(" ")
	out.WriteString(is.Consequence.String())
	if is.Alternative != nil {
		out.WriteString(" else ")
		out.WriteString(is.Alternative.String())
	}
	return out.String()
}

// ForStatement represents a for loop like "for i in items { ... }".
type ForStatement struct {
	Token    token.Token     // the 'for' token
	Variable *Identifier     // loop variable
	Iterable Expression      // the expression being iterated
	Body     *BlockStatement
}

func (fs *ForStatement) statementNode()       {}
func (fs *ForStatement) TokenLiteral() string { return fs.Token.Literal }
func (fs *ForStatement) Pos() token.Position  { return fs.Token.Pos }
func (fs *ForStatement) String() string {
	var out strings.Builder
	out.WriteString("for ")
	out.WriteString(fs.Variable.String())
	out.WriteString(" in ")
	out.WriteString(fs.Iterable.String())
	out.WriteString(" ")
	out.WriteString(fs.Body.String())
	return out.String()
}

// WhileStatement represents a while loop.
type WhileStatement struct {
	Token     token.Token // the 'while' token
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) statementNode()       {}
func (ws *WhileStatement) TokenLiteral() string { return ws.Token.Literal }
func (ws *WhileStatement) Pos() token.Position  { return ws.Token.Pos }
func (ws *WhileStatement) String() string {
	var out strings.Builder
	out.WriteString("while ")
	out.WriteString(ws.Condition.String())
	out.WriteString(" ")
	out.WriteString(ws.Body.String())
	return out.String()
}

// FunctionDefinition represents a function definition like "fn add(a, b) { ... }".
type FunctionDefinition struct {
	Token      token.Token     // the 'fn' token
	Name       *Identifier
	Parameters []*Identifier
	Body       *BlockStatement
}

func (fd *FunctionDefinition) statementNode()       {}
func (fd *FunctionDefinition) TokenLiteral() string { return fd.Token.Literal }
func (fd *FunctionDefinition) Pos() token.Position  { return fd.Token.Pos }
func (fd *FunctionDefinition) String() string {
	var out strings.Builder
	out.WriteString("fn ")
	out.WriteString(fd.Name.String())
	out.WriteString("(")
	params := make([]string, len(fd.Parameters))
	for i, p := range fd.Parameters {
		params[i] = p.String()
	}
	out.WriteString(strings.Join(params, ", "))
	out.WriteString(") ")
	out.WriteString(fd.Body.String())
	return out.String()
}

// ReturnStatement represents a return statement.
type ReturnStatement struct {
	Token token.Token // the 'return' token
	Value Expression  // optional return value
}

func (rs *ReturnStatement) statementNode()       {}
func (rs *ReturnStatement) TokenLiteral() string { return rs.Token.Literal }
func (rs *ReturnStatement) Pos() token.Position  { return rs.Token.Pos }
func (rs *ReturnStatement) String() string {
	if rs.Value != nil {
		return "return " + rs.Value.String()
	}
	return "return"
}

// BreakStatement represents a break statement.
type BreakStatement struct {
	Token token.Token
}

func (bs *BreakStatement) statementNode()       {}
func (bs *BreakStatement) TokenLiteral() string { return bs.Token.Literal }
func (bs *BreakStatement) Pos() token.Position  { return bs.Token.Pos }
func (bs *BreakStatement) String() string       { return "break" }

// ContinueStatement represents a continue statement.
type ContinueStatement struct {
	Token token.Token
}

func (cs *ContinueStatement) statementNode()       {}
func (cs *ContinueStatement) TokenLiteral() string { return cs.Token.Literal }
func (cs *ContinueStatement) Pos() token.Position  { return cs.Token.Pos }
func (cs *ContinueStatement) String() string       { return "continue" }
