package lexer

import (
	"fmt"
	"io"
	"unicode/utf8"
)

type Lexer struct {
	text     string
	position int
	index    int
	current  rune
	line     int
	column   int
}

func NewLexer(s string) (*Lexer, error) {
	lexer := &Lexer{
		text:    s,
		line:    1,
		current: 0,
	}
	if lexer.ReadChar() {
		return lexer, nil
	}
	return nil, io.EOF
}

func (l *Lexer) Tokenize() (TokenBuffer, error) {
	var tokens []Token

	for {
		token, err := l.Next()
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, token)
		if token.Type == EOF {
			break
		}
	}

	buffer := &lexerTokenBuffer{
		tokens:   tokens,
		position: 0,
	}
	if len(tokens) > 0 {
		buffer.current = tokens[0]
		buffer.position = 1
	}
	if len(tokens) > 1 {
		buffer.peek = tokens[1]
		buffer.position = 2
	}

	return buffer, nil
}

func (l *Lexer) ReadChar() bool {
	if l.index >= len(l.text) {
		l.current = 0 // EOF
		l.position = l.index
		return false
	} else {
		current, width := utf8.DecodeRuneInString(l.text[l.index:])
		l.current = current
		l.position = l.index
		l.index += width
	}
	l.column++
	if l.current == '\n' {
		l.line++
		l.column = 0
	}
	return true
}

func (l *Lexer) PeekChar() rune {
	if l.index >= len(l.text) {
		return 0 // EOF
	}
	peek, _ := utf8.DecodeRuneInString(l.text[l.index:])
	return peek
}

func (l *Lexer) Position() TokenPosition {
	return TokenPosition{
		Line:   l.line,
		Column: l.column,
		Offset: l.position,
	}
}

func (l *Lexer) Next() (Token, error) {
	var token Token

	l.SkipWhitespaceAndComments()

	pos := l.Position()
	switch l.current {
	case '=':
		if l.PeekChar() == '=' {
			l.ReadChar()
			token = Token{
				Type:     EQUAL,
				Literal:  "==",
				Position: pos,
			}
		} else {
			token = Token{
				Type:     ASSIGN,
				Literal:  "=",
				Position: pos,
			}
		}
	case '+':
		token = Token{
			Type:     PLUS,
			Literal:  "+",
			Position: pos,
		}
	case '-':
		token = Token{
			Type:     MINUS,
			Literal:  "-",
			Position: pos,
		}
	case '*':
		token = Token{
			Type:     ASTERISK,
			Literal:  "*",
			Position: pos,
		}
	case '|':
		token = Token{
			Type:     PIPE,
			Literal:  "|",
			Position: pos,
		}
	case '/':
		token = Token{
			Type:     SLASH,
			Literal:  "/",
			Position: pos,
		}
	case '%':
		token = Token{
			Type:     PERCENT,
			Literal:  "%",
			Position: pos,
		}
	case '!':
		if l.PeekChar() == '=' {
			l.ReadChar()
			token = Token{
				Type:     NOT_EQUAL,
				Literal:  "!=",
				Position: pos,
			}
		} else {
			token = Token{
				Type:     BANG,
				Literal:  "!",
				Position: pos,
			}
		}
	case '<':
		if l.PeekChar() == '=' {
			l.ReadChar()
			token = Token{
				Type:     LTE,
				Literal:  "<=",
				Position: pos,
			}
		} else {
			token = Token{
				Type:     LT,
				Literal:  "<",
				Position: pos,
			}
		}
	case '>':
		if l.PeekChar() == '=' {
			l.ReadChar()
			token = Token{
				Type:     GTE,
				Literal:  ">=",
				Position: pos,
			}
		} else {
			token = Token{
				Type:     GT,
				Literal:  ">",
				Position: pos,
			}
		}
	case '&':
		if l.PeekChar() == '&' {
			l.ReadChar()
			token = Token{
				Type:     AND,
				Literal:  "&&",
				Position: pos,
			}
		} else {
			return token, fmt.Errorf("unexpected character '&', expected '&&'")
		}
	case ',':
		token = Token{
			Type:     COMMA,
			Literal:  ",",
			Position: pos,
		}
	case ':':
		token = Token{
			Type:     COLON,
			Literal:  ":",
			Position: pos,
		}
	case '.':
		token = Token{
			Type:     DOT,
			Literal:  ".",
			Position: pos,
		}
	case '(':
		token = Token{
			Type:     LPAREN,
			Literal:  "(",
			Position: pos,
		}
	case ')':
		token = Token{
			Type:     RPAREN,
			Literal:  ")",
			Position: pos,
		}
	case '[':
		token = Token{
			Type:     LBRACKET,
			Literal:  "[",
			Position: pos,
		}
	case ']':
		token = Token{
			Type:     RBRACKET,
			Literal:  "]",
			Position: pos,
		}
	case '{':
		token = Token{
			Type:     LBRACE,
			Literal:  "{",
			Position: pos,
		}
	case '}':
		token = Token{
			Type:     RBRACE,
			Literal:  "}",
			Position: pos,
		}
	case ';':
		token = Token{
			Type:     SEMICOLON,
			Literal:  ";",
			Position: pos,
		}
	case '\n':
		token = Token{
			Type:     NEWLINE,
			Literal:  "\n",
			Position: pos,
		}
	case '\'':
		return l.readCharToken(pos)
	case '"':
		return l.ReadStringToken(pos), nil
	case 0:
		token = Token{
			Type:     EOF,
			Literal:  "",
			Position: pos,
		}
	default:
		if IsLetter(l.current) {
			ident := l.ReadIdentifier()
			tokType := LookupIdent(ident)
			return Token{
				Type:     tokType,
				Literal:  ident,
				Position: pos,
			}, nil
		} else if IsDigit(l.current) {
			return l.readNumberToken(pos), nil
		} else {
			return token, fmt.Errorf("unexpected character: %v", string(l.current))
		}
	}

	l.ReadChar()
	return token, nil
}

func (l *Lexer) SkipWhitespaceAndComments() bool {
	for {
		// Skip spaces and tabs
		for l.current == ' ' || l.current == '\t' || l.current == '\r' {
			if !l.ReadChar() {
				return false
			}
		}

		// Skip comments
		if l.current == '#' {
			for l.current != '\n' && l.current != 0 {
				if !l.ReadChar() {
					return false
				}
			}
			continue
		}

		break
	}
	return true
}

func (l *Lexer) ReadIdentifier() string {
	pos := l.position
	for IsLetter(l.current) || IsDigit(l.current) || l.current == '_' {
		l.ReadChar()
	}
	return l.text[pos:l.position]
}

// readNumberToken reads a numeric literal with optional type suffix.
// Integer suffixes: s (short/int16), l (long/int64), default is int (int32).
// Decimal suffixes: f (float/float32), default is decimal (float64).
func (l *Lexer) readNumberToken(pos TokenPosition) Token {
	startPos := l.position
	isDecimal := false

	// Check for hex literal: 0x or 0X
	if l.current == '0' && (l.PeekChar() == 'x' || l.PeekChar() == 'X') {
		l.ReadChar() // consume '0'
		l.ReadChar() // consume 'x'/'X'
		hexStart := l.position
		for IsHexDigit(l.current) {
			l.ReadChar()
		}
		hexLiteral := l.text[hexStart:l.position]
		return Token{
			Type:     BYTE,
			Literal:  "0x" + hexLiteral,
			Position: pos,
		}
	}

	for IsDigit(l.current) {
		l.ReadChar()
	}

	// Check for decimal point
	if l.current == '.' && IsDigit(l.PeekChar()) {
		isDecimal = true
		l.ReadChar() // consume '.'
		for IsDigit(l.current) {
			l.ReadChar()
		}
	}

	literal := l.text[startPos:l.position]

	// Check for type suffix
	if isDecimal {
		if l.current == 'f' {
			l.ReadChar() // consume suffix
			return Token{
				Type:     FLOAT,
				Literal:  literal,
				Position: pos,
			}
		}
		return Token{
			Type:     DECIMAL,
			Literal:  literal,
			Position: pos,
		}
	}

	switch l.current {
	case 'b':
		l.ReadChar() // consume suffix
		return Token{
			Type:     BYTE,
			Literal:  literal,
			Position: pos,
		}
	case 's':
		l.ReadChar() // consume suffix
		return Token{
			Type:     SHORT,
			Literal:  literal,
			Position: pos,
		}
	case 'l':
		l.ReadChar() // consume suffix
		return Token{
			Type:     LONG,
			Literal:  literal,
			Position: pos,
		}
	default:
		return Token{
			Type:     INTEGER,
			Literal:  literal,
			Position: pos,
		}
	}
}

func (l *Lexer) readCharToken(pos TokenPosition) (Token, error) {
	l.ReadChar() // consume opening '\''

	var ch rune
	if l.current == '\\' {
		l.ReadChar() // consume '\\'
		switch l.current {
		case 'n':
			ch = '\n'
		case 't':
			ch = '\t'
		case 'r':
			ch = '\r'
		case '\\':
			ch = '\\'
		case '\'':
			ch = '\''
		default:
			return Token{}, fmt.Errorf("unknown char escape: \\%c", l.current)
		}
	} else if l.current == '\'' || l.current == 0 {
		return Token{}, fmt.Errorf("empty char literal")
	} else {
		ch = l.current
	}

	l.ReadChar() // advance past the character

	if l.current != '\'' {
		return Token{}, fmt.Errorf("unterminated char literal, expected closing '")
	}
	l.ReadChar() // consume closing '\''

	return Token{
		Type:     CHAR,
		Literal:  string(ch),
		Position: pos,
	}, nil
}

func (l *Lexer) ReadStringToken(pos TokenPosition) Token {
	l.ReadChar()

	var result []rune
	hasInterpolation := false

	for {
		if l.current == 0 {
			return Token{
				Type:     ILLEGAL,
				Literal:  string(result),
				Position: pos,
			}
		}

		if l.current == '"' {
			l.ReadChar()
			break
		}

		if l.current == '\\' {
			l.ReadChar()
			switch l.current {
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
				result = append(result, l.current)
			}
			l.ReadChar()
			continue
		}

		if l.current == '$' && l.PeekChar() == '{' {
			hasInterpolation = true
		}

		result = append(result, l.current)
		l.ReadChar()
	}

	if hasInterpolation {
		return Token{
			Type:     INTERP_START,
			Literal:  string(result),
			Position: pos,
		}
	}

	return Token{
		Type:     STRING,
		Literal:  string(result),
		Position: pos,
	}
}
