// Package ast defines the Abstract Syntax Tree for the Vega language.
package ast

import "github.com/mwantia/vega/old/pkg/token"

// Node represents any node in the AST.
type Node interface {
	// TokenLiteral returns the literal value of the token that produced this node.
	TokenLiteral() string
	// Pos returns the position in the source where this node starts.
	Pos() token.Position
	// String returns a string representation for debugging.
	String() string
}

// Statement represents a statement node in the AST.
type Statement interface {
	Node
	statementNode()
}

// Expression represents an expression node in the AST.
type Expression interface {
	Node
	expressionNode()
}

// Program is the root node of the AST representing an entire Vega program.
type Program struct {
	Statements []Statement
}

// TokenLiteral returns the token literal of the first statement.
func (p *Program) TokenLiteral() string {
	if len(p.Statements) > 0 {
		return p.Statements[0].TokenLiteral()
	}
	return ""
}

// Pos returns the position of the first statement.
func (p *Program) Pos() token.Position {
	if len(p.Statements) > 0 {
		return p.Statements[0].Pos()
	}
	return token.Position{}
}

// String returns a string representation of the program.
func (p *Program) String() string {
	var out string
	for _, s := range p.Statements {
		out += s.String()
	}
	return out
}
