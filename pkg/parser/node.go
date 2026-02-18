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
