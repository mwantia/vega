// Package token defines the token types and keywords for the Vega language.
package token

// Type represents a token type.
type Type string

// Token types
const (
	// Special tokens
	ILLEGAL Type = "ILLEGAL"
	EOF     Type = "EOF"

	// Literals
	IDENT  Type = "IDENT"  // identifiers: x, foo, vfs
	INT    Type = "INT"    // 42
	FLOAT  Type = "FLOAT"  // 3.14
	STRING Type = "STRING" // "hello"

	// String interpolation
	INTERP_START Type = "INTERP_START" // Start of interpolated string
	INTERP_PART  Type = "INTERP_PART"  // String part of interpolation
	INTERP_EXPR  Type = "INTERP_EXPR"  // Expression placeholder in interpolation
	INTERP_END   Type = "INTERP_END"   // End of interpolated string

	// Operators
	ASSIGN   Type = "="
	PLUS     Type = "+"
	MINUS    Type = "-"
	ASTERISK Type = "*"
	SLASH    Type = "/"
	PERCENT  Type = "%"
	BANG     Type = "!"

	// Comparison
	EQ     Type = "=="
	NOT_EQ Type = "!="
	LT     Type = "<"
	GT     Type = ">"
	LTE    Type = "<="
	GTE    Type = ">="

	// Logical
	AND Type = "&&"
	OR  Type = "||"

	// Pipe
	PIPE Type = "|"

	// Delimiters
	COMMA     Type = ","
	COLON     Type = ":"
	DOT       Type = "."
	LPAREN    Type = "("
	RPAREN    Type = ")"
	LBRACKET  Type = "["
	RBRACKET  Type = "]"
	LBRACE    Type = "{"
	RBRACE    Type = "}"
	NEWLINE   Type = "NEWLINE"
	SEMICOLON Type = ";"

	// Keywords
	TRUE     Type = "TRUE"
	FALSE    Type = "FALSE"
	NIL      Type = "NIL"
	IF       Type = "IF"
	ELSE     Type = "ELSE"
	FOR      Type = "FOR"
	WHILE    Type = "WHILE"
	IN       Type = "IN"
	FN       Type = "FN"
	RETURN   Type = "RETURN"
	BREAK    Type = "BREAK"
	CONTINUE Type = "CONTINUE"
)

// keywords maps keyword strings to their token types.
var keywords = map[string]Type{
	"true":     TRUE,
	"false":    FALSE,
	"nil":      NIL,
	"if":       IF,
	"else":     ELSE,
	"for":      FOR,
	"while":    WHILE,
	"in":       IN,
	"fn":       FN,
	"return":   RETURN,
	"break":    BREAK,
	"continue": CONTINUE,
}

// LookupIdent checks if an identifier is a keyword and returns the appropriate type.
func LookupIdent(ident string) Type {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}

// Position represents a source position with line and column.
type Position struct {
	Line   int // 1-based line number
	Column int // 1-based column number
	Offset int // 0-based byte offset
}

// Token represents a single token in the source.
type Token struct {
	Type    Type
	Literal string
	Pos     Position
}

// String returns a string representation of the token.
func (t Token) String() string {
	return "Token{Type: " + string(t.Type) + ", Literal: " + t.Literal + "}"
}

// IsKeyword returns true if the token type is a keyword.
func (t Token) IsKeyword() bool {
	switch t.Type {
	case TRUE, FALSE, NIL, IF, ELSE, FOR, WHILE, IN, FN, RETURN, BREAK, CONTINUE:
		return true
	}
	return false
}

// IsOperator returns true if the token type is an operator.
func (t Token) IsOperator() bool {
	switch t.Type {
	case ASSIGN, PLUS, MINUS, ASTERISK, SLASH, PERCENT, BANG,
		EQ, NOT_EQ, LT, GT, LTE, GTE, AND, OR, PIPE:
		return true
	}
	return false
}

// IsLiteral returns true if the token type is a literal.
func (t Token) IsLiteral() bool {
	switch t.Type {
	case INT, FLOAT, STRING, TRUE, FALSE, NIL:
		return true
	}
	return false
}
