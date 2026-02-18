package parser

import (
	"github.com/mwantia/vega/pkg/lexer"
)

// Precedence levels for the Pratt parser.
const (
	LOWEST     int = iota
	ASSIGNMENT     // =
	OR             // ||
	AND            // &&
	EQUALITY       // == !=
	COMPARISON     // < > <= >=
	TERM           // + -
	FACTOR         // * / %
	UNARY          // -x !x
	CALL           // fn() obj.method() obj[index]
	PRIMARY        // literals, identifiers
)

// precedences maps token types to their precedence levels.
// Note: ASSIGN is not included here - assignment is handled at statement level
var precedences = map[lexer.TokenType]int{
	lexer.OR:        OR,
	lexer.AND:       AND,
	lexer.EQUAL:     EQUALITY,
	lexer.NOT_EQUAL: EQUALITY,
	lexer.LT:        COMPARISON,
	lexer.GT:        COMPARISON,
	lexer.LTE:       COMPARISON,
	lexer.GTE:       COMPARISON,
	lexer.PLUS:      TERM,
	lexer.MINUS:     TERM,
	lexer.ASTERISK:  FACTOR,
	lexer.SLASH:     FACTOR,
	lexer.PERCENT:   FACTOR,
	lexer.LPAREN:    CALL,
	lexer.LBRACKET:  CALL,
	lexer.DOT:       CALL,
}

// GetPrecedence returns the precedence of a token type.
func GetTokenPrecedence(t lexer.Token) int {
	if p, ok := precedences[t.Type]; ok {
		return p
	}
	return LOWEST
}
