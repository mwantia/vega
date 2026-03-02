package parser

import "github.com/mwantia/vega/pkg/lexer"

type Node interface {
	Literal() string

	Position() lexer.TokenPosition

	String() string
}

type Statement interface {
	Node
	// Statement marker method.
	Statement()
}

type Expression interface {
	Node
	// Expression marker method.
	Expression()
}

// BaseExpression holds the source token and satisfies Literal(), Position(), and
// Expression() for any expression node that embeds it.
type BaseExpression struct {
	Token lexer.Token
}

func (n BaseExpression) Literal() string               { return n.Token.Literal }
func (n BaseExpression) Position() lexer.TokenPosition { return n.Token.Position }
func (*BaseExpression) Expression()                    {}

// BaseStatement holds the source token and satisfies Literal(), Position(), and
// Statement() for any statement node that embeds it.
type BaseStatement struct {
	Token lexer.Token
}

func (n BaseStatement) Literal() string               { return n.Token.Literal }
func (n BaseStatement) Position() lexer.TokenPosition { return n.Token.Position }
func (*BaseStatement) Statement()                     {}
