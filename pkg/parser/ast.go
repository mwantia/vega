package parser

import (
	"strings"

	"github.com/mwantia/vega/pkg/lexer"
)

type AST interface {
	Node

	Statements() []Statement
}

type Program struct {
	AST

	statements []Statement
}

func NewProgram() *Program {
	return &Program{
		statements: make([]Statement, 0),
	}
}

// Literal implements Node.
func (p *Program) Literal() string {
	if len(p.statements) > 0 {
		return p.statements[0].Literal()
	}
	return ""
}

// Position implements Node.
func (p *Program) Position() lexer.TokenPosition {
	if len(p.statements) > 0 {
		return p.statements[0].Position()
	}
	return lexer.EmptyPosition
}

// String implements Node.
func (p *Program) String() string {
	var out strings.Builder
	for _, statement := range p.statements {
		s := statement.String()
		out.WriteString(s)
	}
	return out.String()
}

// Statements implements AST.
func (p *Program) Statements() []Statement {
	return p.statements
}

var _ AST = (*Program)(nil)
