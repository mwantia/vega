package parser

import (
	"strings"

	"github.com/mwantia/vega/pkg/lexer"
)

type CallStatement struct {
	BaseStatement
	Function  Expression
	Arguments []Expression
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
	BaseStatement
	Value Expression // must be a CallExpression
}

func (ds *DiscardStatement) String() string { return "_ = " + ds.Value.String() }

var _ Statement = (*DiscardStatement)(nil)

type AssignmentStatement struct {
	BaseStatement
	Name        *IdentifierExpression
	Constraints []Expression
	Value       Expression
}

func (as *AssignmentStatement) Position() lexer.TokenPosition { return as.Name.Position() }

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
	BaseStatement
	Left  *IndexExpression
	Value Expression
}

func (ias *IndexAssignmentStatement) Position() lexer.TokenPosition { return ias.Left.Position() }

func (ias *IndexAssignmentStatement) String() string {
	return ias.Left.String() + " = " + ias.Value.String()
}

var _ Statement = (*IndexAssignmentStatement)(nil)

type BlockStatement struct {
	BaseStatement
	Statements []Statement
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
	BaseStatement
	Condition   Expression
	Consequence *BlockStatement
	Alternative *BlockStatement
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
	BaseStatement
	Variable *IdentifierExpression
	Iterable Expression
	Body     *BlockStatement
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
	BaseStatement
	Condition Expression
	Body      *BlockStatement
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
	BaseStatement
	Name       *IdentifierExpression
	Parameters []*DeclarationExpression
	Body       *BlockStatement
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
	BaseStatement
	Value Expression
}

func (rs *ReturnStatement) String() string {
	if rs.Value != nil {
		return "return " + rs.Value.String()
	}
	return "return"
}

var _ Statement = (*ReturnStatement)(nil)

type BreakStatement struct {
	BaseStatement
}

func (*BreakStatement) String() string { return "break" }

var _ Statement = (*BreakStatement)(nil)

type ContinueStatement struct {
	BaseStatement
}

func (*ContinueStatement) String() string { return "continue" }

var _ Statement = (*ContinueStatement)(nil)

type FreeStatement struct {
	BaseStatement
	Name *IdentifierExpression
}

func (fs *FreeStatement) String() string { return "free(" + fs.Name.String() + ")" }

var _ Statement = (*FreeStatement)(nil)

type StructField struct {
	Name     string
	Type     string // type name (e.g. "int", "bool", "string", "byte")
	Capacity int    // >0 for parameterised slice fields (e.g. string<10> → Capacity=10)
}

type StructStatement struct {
	BaseStatement
	Name   string
	Fields []StructField
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
