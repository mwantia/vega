package parser

import "github.com/mwantia/vega/pkg/token"

// Precedence levels for the Pratt parser.
// Higher numbers bind tighter.
const (
	LOWEST      int = iota
	ASSIGNMENT      // =
	PIPE            // |
	OR              // ||
	AND             // &&
	EQUALITY        // == !=
	COMPARISON      // < > <= >=
	TERM            // + -
	FACTOR          // * / %
	UNARY           // -x !x
	CALL            // fn() obj.method() obj[index]
	PRIMARY         // literals, identifiers
)

// precedences maps token types to their precedence levels.
// Note: ASSIGN is not included here - assignment is handled at statement level
var precedences = map[token.Type]int{
	token.PIPE:     PIPE,
	token.OR:       OR,
	token.AND:      AND,
	token.EQ:       EQUALITY,
	token.NOT_EQ:   EQUALITY,
	token.LT:       COMPARISON,
	token.GT:       COMPARISON,
	token.LTE:      COMPARISON,
	token.GTE:      COMPARISON,
	token.PLUS:     TERM,
	token.MINUS:    TERM,
	token.ASTERISK: FACTOR,
	token.SLASH:    FACTOR,
	token.PERCENT:  FACTOR,
	token.LPAREN:   CALL,
	token.LBRACKET: CALL,
	token.DOT:      CALL,
}

// getPrecedence returns the precedence of a token type.
func getPrecedence(t token.Type) int {
	if p, ok := precedences[t]; ok {
		return p
	}
	return LOWEST
}
