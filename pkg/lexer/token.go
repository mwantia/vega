package lexer

type TokenType string

const (
	// Special tokens
	ILLEGAL TokenType = "ILLEGAL"
	EOF     TokenType = "EOF"

	IDENT   TokenType = "IDENT"
	BYTE    TokenType = "BYTE"
	SHORT   TokenType = "SHORT"
	INTEGER TokenType = "INTEGER"
	LONG    TokenType = "LONG"
	FLOAT   TokenType = "FLOAT"
	DECIMAL TokenType = "DECIMAL"
	CHAR    TokenType = "CHAR"
	STRING  TokenType = "STRING"
	SLICE   TokenType = "SLICE"

	// Start of interpolated string
	INTERP_START TokenType = "INTERP_START"
	// String part of interpolation
	INTERP_PART TokenType = "INTERP_PART"
	// Expression placeholder in interpolation
	INTERP_EXPR TokenType = "INTERP_EXPR"
	// End of interpolated string
	INTERP_END TokenType = "INTERP_END"

	ASSIGN   TokenType = "="
	PLUS     TokenType = "+"
	MINUS    TokenType = "-"
	ASTERISK TokenType = "*"
	SLASH    TokenType = "/"
	PERCENT  TokenType = "%"
	BANG     TokenType = "!"
	PIPE     TokenType = "|"

	EQUAL     TokenType = "=="
	NOT_EQUAL TokenType = "!="
	LT        TokenType = "<"
	GT        TokenType = ">"
	LTE       TokenType = "<="
	GTE       TokenType = ">="

	AND TokenType = "&&"
	OR  TokenType = "||"

	COMMA     TokenType = ","
	COLON     TokenType = ":"
	DOT       TokenType = "."
	LPAREN    TokenType = "("
	RPAREN    TokenType = ")"
	LBRACKET  TokenType = "["
	RBRACKET  TokenType = "]"
	LBRACE    TokenType = "{"
	RBRACE    TokenType = "}"
	NEWLINE   TokenType = "NL"
	SEMICOLON TokenType = ";"

	TRUE     TokenType = "TRUE"
	FALSE    TokenType = "FALSE"
	NIL      TokenType = "NIL"
	IF       TokenType = "IF"
	ELSE     TokenType = "ELSE"
	FOR      TokenType = "FOR"
	WHILE    TokenType = "WHILE"
	IN       TokenType = "IN"
	FN       TokenType = "FN"
	RETURN   TokenType = "RETURN"
	BREAK    TokenType = "BREAK"
	CONTINUE TokenType = "CONTINUE"
	ALLOC    TokenType = "ALLOC"
	FREE     TokenType = "FREE"
	STRUCT   TokenType = "STRUCT"
)

var (
	EmptyPosition = TokenPosition{
		Line:   0,
		Column: 0,
		Offset: 0,
	}
)

type TokenPosition struct {
	Line   int
	Column int
	Offset int
}

// Token represents a single token in the source.
type Token struct {
	Type     TokenType
	Literal  string
	Position TokenPosition
}

func (t Token) IsNewline() bool {
	switch t.Type {
	case NEWLINE:
		return true
	}
	return false
}

// IsKeyword returns true if the token type is a keyword.
func (t Token) IsKeyword() bool {
	switch t.Type {
	case TRUE, FALSE, NIL, IF, ELSE, FOR, WHILE, IN, FN, RETURN, BREAK, CONTINUE, ALLOC, FREE, STRUCT:
		return true
	}
	return false
}

// IsOperator returns true if the token type is an operator.
func (t Token) IsOperator() bool {
	switch t.Type {
	case ASSIGN, PLUS, MINUS, ASTERISK, SLASH, PERCENT, BANG, EQUAL, NOT_EQUAL, LT, GT, LTE, GTE, AND, OR:
		return true
	}
	return false
}

// IsLiteral returns true if the token type is a literal.
func (t Token) IsLiteral() bool {
	switch t.Type {
	case BYTE, SHORT, INTEGER, LONG, FLOAT, DECIMAL, CHAR, STRING, TRUE, FALSE, NIL:
		return true
	}
	return false
}

var keywords = map[string]TokenType{
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
	"alloc":    ALLOC,
	"free":     FREE,
	"struct":   STRUCT,
}

func LookupIdent(ident string) TokenType {
	if tok, ok := keywords[ident]; ok {
		return tok
	}
	return IDENT
}
