package parser

import (
	"strings"

	"github.com/mwantia/vega/pkg/lexer"
)

type CallStatement struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

func (cs *CallStatement) Statement() {}

func (cs *CallStatement) Literal() string {
	return cs.Token.Literal
}

func (cs *CallStatement) Position() lexer.TokenPosition {
	return cs.Token.Position
}

func (cs *CallStatement) String() string {
	var out strings.Builder
	out.WriteString(cs.Function.String())
	out.WriteString("(")
	for i, arg := range cs.Arguments {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(arg.String())
	}
	out.WriteString(")")
	return out.String()
}

var _ Statement = (*CallStatement)(nil)

type DiscardStatement struct {
	Token lexer.Token
	Value Expression // must be a CallExpression
}

func (ds *DiscardStatement) Statement() {}

func (ds *DiscardStatement) Literal() string {
	return ds.Token.Literal
}

func (ds *DiscardStatement) Position() lexer.TokenPosition {
	return ds.Token.Position
}

func (ds *DiscardStatement) String() string {
	return "_ = " + ds.Value.String()
}

var _ Statement = (*DiscardStatement)(nil)

type AssignmentStatement struct {
	Token       lexer.Token
	Name        *IdentifierExpression
	Constraints []Expression
	Value       Expression
}

func (as *AssignmentStatement) Statement() {

}

func (as *AssignmentStatement) Literal() string {
	return as.Token.Literal
}

func (as *AssignmentStatement) Position() lexer.TokenPosition {
	return as.Name.Position()
}

func (as *AssignmentStatement) String() string {
	if len(as.Constraints) > 0 {
		parts := make([]string, len(as.Constraints))
		for i, c := range as.Constraints {
			parts[i] = c.String()
		}
		return as.Name.String() + ": " + strings.Join(parts, "|") + " = " + as.Value.String()
	}
	return as.Name.String() + " = " + as.Value.String()
}

var _ Statement = (*AssignmentStatement)(nil)

type IndexAssignmentStatement struct {
	Token lexer.Token
	Left  *IndexExpression
	Value Expression
}

func (ias *IndexAssignmentStatement) Statement() {

}

func (ias *IndexAssignmentStatement) Literal() string {
	return ias.Token.Literal
}

func (ias *IndexAssignmentStatement) Position() lexer.TokenPosition {
	return ias.Left.Position()
}

func (ias *IndexAssignmentStatement) String() string {
	return ias.Left.String() + " = " + ias.Value.String()
}

var _ Statement = (*IndexAssignmentStatement)(nil)

type BlockStatement struct {
	Token      lexer.Token
	Statements []Statement
}

func (bs *BlockStatement) Statement() {

}

func (bs *BlockStatement) Literal() string {
	return bs.Token.Literal
}

func (bs *BlockStatement) Position() lexer.TokenPosition {
	return bs.Token.Position
}

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

var _ Statement = (*BlockStatement)(nil)

type IfStatement struct {
	Token       lexer.Token
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
}

func (is *IfStatement) Statement() {

}

func (is *IfStatement) Literal() string {
	return is.Token.Literal
}

func (is *IfStatement) Position() lexer.TokenPosition {
	return is.Token.Position
}

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

var _ Statement = (*IfStatement)(nil)

type ForStatement struct {
	Token    lexer.Token
	Variable *IdentifierExpression
	Iterable Expression
	Body     *BlockStatement
}

func (fs *ForStatement) Statement() {

}

func (fs *ForStatement) Literal() string {
	return fs.Token.Literal
}

func (fs *ForStatement) Position() lexer.TokenPosition {
	return fs.Token.Position
}

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

var _ Statement = (*ForStatement)(nil)

type WhileStatement struct {
	Token     lexer.Token
	Condition Expression
	Body      *BlockStatement
}

func (ws *WhileStatement) Statement() {

}

func (ws *WhileStatement) Literal() string {
	return ws.Token.Literal
}

func (ws *WhileStatement) Position() lexer.TokenPosition {
	return ws.Token.Position
}

func (ws *WhileStatement) String() string {
	var out strings.Builder
	out.WriteString("while ")
	out.WriteString(ws.Condition.String())
	out.WriteString(" ")
	out.WriteString(ws.Body.String())
	return out.String()
}

var _ Statement = (*WhileStatement)(nil)

type FunctionStatement struct {
	Token      lexer.Token
	Name       *IdentifierExpression
	Parameters []*DeclarationExpression
	Body       *BlockStatement
}

func (fd *FunctionStatement) Statement() {

}

func (fd *FunctionStatement) Literal() string {
	return fd.Token.Literal
}

func (fd *FunctionStatement) Position() lexer.TokenPosition {
	return fd.Token.Position
}

func (fd *FunctionStatement) String() string {
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

var _ Statement = (*FunctionStatement)(nil)

type ReturnStatement struct {
	Token lexer.Token
	Value Expression
}

func (rs *ReturnStatement) Statement() {

}

func (rs *ReturnStatement) Literal() string {
	return rs.Token.Literal
}

func (rs *ReturnStatement) Position() lexer.TokenPosition {
	return rs.Token.Position
}

func (rs *ReturnStatement) String() string {
	if rs.Value != nil {
		return "return " + rs.Value.String()
	}
	return "return"
}

var _ Statement = (*ReturnStatement)(nil)

// BreakStatement represents a break statement.
type BreakStatement struct {
	Token lexer.Token
}

func (bs *BreakStatement) Statement() {

}

func (bs *BreakStatement) Literal() string {
	return bs.Token.Literal
}

func (bs *BreakStatement) Position() lexer.TokenPosition {
	return bs.Token.Position
}

func (bs *BreakStatement) String() string {
	return "break"
}

var _ Statement = (*BreakStatement)(nil)

// ContinueStatement represents a continue statement.
type ContinueStatement struct {
	Token lexer.Token
}

func (cs *ContinueStatement) Statement() {

}

func (cs *ContinueStatement) Literal() string {
	return cs.Token.Literal
}

func (cs *ContinueStatement) Position() lexer.TokenPosition {
	return cs.Token.Position
}

func (cs *ContinueStatement) String() string {
	return "continue"
}

var _ Statement = (*ContinueStatement)(nil)

type AllocStatement struct {
	Token lexer.Token
	Size  Expression
	Body  *BlockStatement
}

func (as *AllocStatement) Statement() {

}

func (as *AllocStatement) Literal() string {
	return as.Token.Literal
}

func (as *AllocStatement) Position() lexer.TokenPosition {
	return as.Token.Position
}

func (as *AllocStatement) String() string {
	var out strings.Builder
	out.WriteString("alloc ")
	out.WriteString(as.Size.String())
	out.WriteString(" ")
	out.WriteString(as.Body.String())
	return out.String()
}

var _ Statement = (*AllocStatement)(nil)

type FreeStatement struct {
	Token lexer.Token
	Name  *IdentifierExpression
}

func (fs *FreeStatement) Statement() {

}

func (fs *FreeStatement) Literal() string {
	return fs.Token.Literal
}

func (fs *FreeStatement) Position() lexer.TokenPosition {
	return fs.Token.Position
}

func (fs *FreeStatement) String() string {
	return "free(" + fs.Name.String() + ")"
}

var _ Statement = (*FreeStatement)(nil)

// StructField represents a single field declaration inside a struct definition.
type StructField struct {
	Name string
	Type string // type name (e.g. "int", "bool")
}

// StructStatement represents a struct type definition: struct name { field: type, ... }
type StructStatement struct {
	Token  lexer.Token
	Name   string
	Fields []StructField
}

func (ss *StructStatement) Statement() {}

func (ss *StructStatement) Literal() string {
	return ss.Token.Literal
}

func (ss *StructStatement) Position() lexer.TokenPosition {
	return ss.Token.Position
}

func (ss *StructStatement) String() string {
	var out strings.Builder
	out.WriteString("struct ")
	out.WriteString(ss.Name)
	out.WriteString(" { ")
	for i, f := range ss.Fields {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(f.Name)
		out.WriteString(": ")
		out.WriteString(f.Type)
	}
	out.WriteString(" }")
	return out.String()
}

var _ Statement = (*StructStatement)(nil)
