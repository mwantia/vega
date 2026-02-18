package lexer

import (
	"slices"
)

type TokenBuffer interface {
	Current() Token

	EndReached() bool

	MatchAny(bool, ...TokenType) bool

	Read() (Token, bool)

	SkipAny(...TokenType)

	Peek() Token
}

type lexerTokenBuffer struct {
	tokens []Token

	position int
	current  Token
	peek     Token
}

// Current implements TokenBuffer.
func (l *lexerTokenBuffer) Current() Token {
	return l.current
}

// EndReached implements TokenBuffer.
func (l *lexerTokenBuffer) EndReached() bool {
	return l.current.Type == EOF
}

// MatchAny implements TokenBuffer.
func (l *lexerTokenBuffer) MatchAny(consume bool, types ...TokenType) bool {
	if slices.Contains(types, l.current.Type) {
		if consume {
			l.Read()
		}
		return true
	}
	return false
}

// Read implements TokenBuffer.
func (l *lexerTokenBuffer) Read() (Token, bool) {
	l.current = l.peek
	if l.position < len(l.tokens) {
		l.peek = l.tokens[l.position]
	} else {
		l.peek = Token{
			Type: EOF,
		}
	}
	l.position++
	return l.current, true
}

// SkipAny implements TokenBuffer.
func (l *lexerTokenBuffer) SkipAny(types ...TokenType) {
	for slices.Contains(types, l.current.Type) {
		l.Read()
	}
}

// Peek implements TokenBuffer.
func (l *lexerTokenBuffer) Peek() Token {
	return l.peek
}

var _ TokenBuffer = (*lexerTokenBuffer)(nil)
