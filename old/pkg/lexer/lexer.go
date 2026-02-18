// Package lexer provides tokenization for the Vega language.
package lexer

import (
	"unicode"
	"unicode/utf8"

	"github.com/mwantia/vega/errors"
	"github.com/mwantia/vega/old/pkg/token"
)

// Lexer tokenizes Vega source code.
type Lexer struct {
	input        string
	position     int  // current position in input (points to current char)
	readPosition int  // current reading position in input (after current char)
	ch           rune // current character under examination
	line         int  // current line number (1-indexed)
	column       int  // current column number (1-indexed)
	errors       errors.ErrorList
}

// New creates a new Lexer for the given input.
func New(input string) *Lexer {
	l := &Lexer{
		input:  input,
		line:   1,
		column: 0,
	}
	l.readChar()
	return l
}

// Errors returns any lexing errors encountered.
func (l *Lexer) Errors() errors.ErrorList {
	return l.errors
}

// readChar advances the lexer by one character.
func (l *Lexer) readChar() {
	if l.readPosition >= len(l.input) {
		l.ch = 0                    // EOF
		l.position = l.readPosition // Update position even at EOF for correct slicing
	} else {
		r, width := utf8.DecodeRuneInString(l.input[l.readPosition:])
		l.ch = r
		l.position = l.readPosition
		l.readPosition += width
	}
	l.column++
	if l.ch == '\n' {
		l.line++
		l.column = 0
	}
}

// peekChar returns the next character without advancing.
func (l *Lexer) peekChar() rune {
	if l.readPosition >= len(l.input) {
		return 0
	}
	r, _ := utf8.DecodeRuneInString(l.input[l.readPosition:])
	return r
}

// currentPos returns the current position.
func (l *Lexer) currentPos() token.Position {
	return token.Position{
		Line:   l.line,
		Column: l.column,
		Offset: l.position,
	}
}

// NextToken returns the next token from the input.
func (l *Lexer) NextToken() token.Token {
	var tok token.Token

	l.skipWhitespaceAndComments()

	pos := l.currentPos()

	switch l.ch {
	case '=':
		if l.peekChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.EQ, Literal: "==", Pos: pos}
		} else {
			tok = token.Token{Type: token.ASSIGN, Literal: "=", Pos: pos}
		}
	case '+':
		tok = token.Token{Type: token.PLUS, Literal: "+", Pos: pos}
	case '-':
		tok = token.Token{Type: token.MINUS, Literal: "-", Pos: pos}
	case '*':
		tok = token.Token{Type: token.ASTERISK, Literal: "*", Pos: pos}
	case '/':
		tok = token.Token{Type: token.SLASH, Literal: "/", Pos: pos}
	case '%':
		tok = token.Token{Type: token.PERCENT, Literal: "%", Pos: pos}
	case '!':
		if l.peekChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.NOT_EQ, Literal: "!=", Pos: pos}
		} else {
			tok = token.Token{Type: token.BANG, Literal: "!", Pos: pos}
		}
	case '<':
		if l.peekChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.LTE, Literal: "<=", Pos: pos}
		} else {
			tok = token.Token{Type: token.LT, Literal: "<", Pos: pos}
		}
	case '>':
		if l.peekChar() == '=' {
			l.readChar()
			tok = token.Token{Type: token.GTE, Literal: ">=", Pos: pos}
		} else {
			tok = token.Token{Type: token.GT, Literal: ">", Pos: pos}
		}
	case '&':
		if l.peekChar() == '&' {
			l.readChar()
			tok = token.Token{Type: token.AND, Literal: "&&", Pos: pos}
		} else {
			l.addError("unexpected character '&', expected '&&'")
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Pos: pos}
		}
	case '|':
		if l.peekChar() == '|' {
			l.readChar()
			tok = token.Token{Type: token.OR, Literal: "||", Pos: pos}
		} else {
			tok = token.Token{Type: token.PIPE, Literal: "|", Pos: pos}
		}
	case ',':
		tok = token.Token{Type: token.COMMA, Literal: ",", Pos: pos}
	case ':':
		tok = token.Token{Type: token.COLON, Literal: ":", Pos: pos}
	case '.':
		tok = token.Token{Type: token.DOT, Literal: ".", Pos: pos}
	case '(':
		tok = token.Token{Type: token.LPAREN, Literal: "(", Pos: pos}
	case ')':
		tok = token.Token{Type: token.RPAREN, Literal: ")", Pos: pos}
	case '[':
		tok = token.Token{Type: token.LBRACKET, Literal: "[", Pos: pos}
	case ']':
		tok = token.Token{Type: token.RBRACKET, Literal: "]", Pos: pos}
	case '{':
		tok = token.Token{Type: token.LBRACE, Literal: "{", Pos: pos}
	case '}':
		tok = token.Token{Type: token.RBRACE, Literal: "}", Pos: pos}
	case ';':
		tok = token.Token{Type: token.SEMICOLON, Literal: ";", Pos: pos}
	case '\n':
		tok = token.Token{Type: token.NEWLINE, Literal: "\n", Pos: pos}
	case '"':
		return l.readString(pos)
	case 0:
		tok = token.Token{Type: token.EOF, Literal: "", Pos: pos}
	default:
		if isLetter(l.ch) {
			ident := l.readIdentifier()
			tokType := token.LookupIdent(ident)
			return token.Token{Type: tokType, Literal: ident, Pos: pos}
		} else if isDigit(l.ch) {
			return l.readNumber(pos)
		} else {
			l.addError("unexpected character: " + string(l.ch))
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Pos: pos}
		}
	}

	l.readChar()
	return tok
}

// Tokenize returns all tokens from the input.
func (l *Lexer) Tokenize() ([]token.Token, error) {
	var tokens []token.Token

	for {
		tok := l.NextToken()
		tokens = append(tokens, tok)
		if tok.Type == token.EOF {
			break
		}
	}

	if l.errors.HasErrors() {
		return tokens, l.errors
	}
	return tokens, nil
}

// skipWhitespaceAndComments skips whitespace (except newlines) and comments.
func (l *Lexer) skipWhitespaceAndComments() {
	for {
		// Skip spaces and tabs
		for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' {
			l.readChar()
		}

		// Skip comments
		if l.ch == '#' {
			l.skipComment()
			continue
		}

		break
	}
}

// skipComment skips a comment until end of line.
func (l *Lexer) skipComment() {
	for l.ch != '\n' && l.ch != 0 {
		l.readChar()
	}
}

// readIdentifier reads an identifier.
func (l *Lexer) readIdentifier() string {
	position := l.position
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[position:l.position]
}

// readNumber reads an integer or float literal.
func (l *Lexer) readNumber(pos token.Position) token.Token {
	startPos := l.position
	isFloat := false

	for isDigit(l.ch) {
		l.readChar()
	}

	// Check for decimal point
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar() // consume '.'
		for isDigit(l.ch) {
			l.readChar()
		}
	}

	literal := l.input[startPos:l.position]
	if isFloat {
		return token.Token{Type: token.FLOAT, Literal: literal, Pos: pos}
	}
	return token.Token{Type: token.INT, Literal: literal, Pos: pos}
}

// readString reads a string literal, handling escape sequences and interpolation.
func (l *Lexer) readString(pos token.Position) token.Token {
	l.readChar() // consume opening quote

	var result []rune
	hasInterpolation := false

	for {
		if l.ch == 0 {
			l.addError("unclosed string literal")
			return token.Token{Type: token.ILLEGAL, Literal: string(result), Pos: pos}
		}

		if l.ch == '"' {
			l.readChar() // consume closing quote
			break
		}

		if l.ch == '\\' {
			// Escape sequence
			l.readChar()
			switch l.ch {
			case 'n':
				result = append(result, '\n')
			case 't':
				result = append(result, '\t')
			case 'r':
				result = append(result, '\r')
			case '\\':
				result = append(result, '\\')
			case '"':
				result = append(result, '"')
			case '$':
				result = append(result, '$')
			default:
				l.addError("invalid escape sequence: \\" + string(l.ch))
				result = append(result, l.ch)
			}
			l.readChar()
			continue
		}

		if l.ch == '$' && l.peekChar() == '{' {
			hasInterpolation = true
		}

		result = append(result, l.ch)
		l.readChar()
	}

	// If there's interpolation, use INTERP_START to signal it
	if hasInterpolation {
		return token.Token{Type: token.INTERP_START, Literal: string(result), Pos: pos}
	}

	return token.Token{Type: token.STRING, Literal: string(result), Pos: pos}
}

// addError adds a lexer error.
func (l *Lexer) addError(message string) {
	l.errors.Add(errors.NewSyntaxError(message, l.line, l.column))
}

// isLetter returns true if the rune is a letter or underscore.
func isLetter(ch rune) bool {
	return unicode.IsLetter(ch) || ch == '_'
}

// isDigit returns true if the rune is a digit.
func isDigit(ch rune) bool {
	return unicode.IsDigit(ch)
}
