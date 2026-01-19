package lexer

import (
	"testing"

	"github.com/mwantia/vega/pkg/token"
)

func TestNextToken(t *testing.T) {
	input := `x = 42
path = "/data"
count = 3.14
enabled = true
nothing = nil

# This is a comment
if count > 10 {
    print("large")
} else {
    print("small")
}

for item in data {
    print(item)
}

while x < 100 {
    x = x + 1
}

fn add(a, b) {
    return a + b
}

result = vfs.stat(path)
arr = [1, 2, 3]
map = {name: "alice", age: 30}
`

	tests := []struct {
		expectedType    token.Type
		expectedLiteral string
	}{
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "42"},
		{token.NEWLINE, "\n"},

		{token.IDENT, "path"},
		{token.ASSIGN, "="},
		{token.STRING, "/data"},
		{token.NEWLINE, "\n"},

		{token.IDENT, "count"},
		{token.ASSIGN, "="},
		{token.FLOAT, "3.14"},
		{token.NEWLINE, "\n"},

		{token.IDENT, "enabled"},
		{token.ASSIGN, "="},
		{token.TRUE, "true"},
		{token.NEWLINE, "\n"},

		{token.IDENT, "nothing"},
		{token.ASSIGN, "="},
		{token.NIL, "nil"},
		{token.NEWLINE, "\n"},
		{token.NEWLINE, "\n"}, // blank line

		// After comment
		{token.NEWLINE, "\n"},

		{token.IF, "if"},
		{token.IDENT, "count"},
		{token.GT, ">"},
		{token.INT, "10"},
		{token.LBRACE, "{"},
		{token.NEWLINE, "\n"},
		{token.IDENT, "print"},
		{token.LPAREN, "("},
		{token.STRING, "large"},
		{token.RPAREN, ")"},
		{token.NEWLINE, "\n"},
		{token.RBRACE, "}"},
		{token.ELSE, "else"},
		{token.LBRACE, "{"},
		{token.NEWLINE, "\n"},
		{token.IDENT, "print"},
		{token.LPAREN, "("},
		{token.STRING, "small"},
		{token.RPAREN, ")"},
		{token.NEWLINE, "\n"},
		{token.RBRACE, "}"},
		{token.NEWLINE, "\n"},
		{token.NEWLINE, "\n"},

		// for loop
		{token.FOR, "for"},
		{token.IDENT, "item"},
		{token.IN, "in"},
		{token.IDENT, "data"},
		{token.LBRACE, "{"},
		{token.NEWLINE, "\n"},
		{token.IDENT, "print"},
		{token.LPAREN, "("},
		{token.IDENT, "item"},
		{token.RPAREN, ")"},
		{token.NEWLINE, "\n"},
		{token.RBRACE, "}"},
		{token.NEWLINE, "\n"},
		{token.NEWLINE, "\n"},

		// while loop
		{token.WHILE, "while"},
		{token.IDENT, "x"},
		{token.LT, "<"},
		{token.INT, "100"},
		{token.LBRACE, "{"},
		{token.NEWLINE, "\n"},
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.IDENT, "x"},
		{token.PLUS, "+"},
		{token.INT, "1"},
		{token.NEWLINE, "\n"},
		{token.RBRACE, "}"},
		{token.NEWLINE, "\n"},
		{token.NEWLINE, "\n"},

		// function definition
		{token.FN, "fn"},
		{token.IDENT, "add"},
		{token.LPAREN, "("},
		{token.IDENT, "a"},
		{token.COMMA, ","},
		{token.IDENT, "b"},
		{token.RPAREN, ")"},
		{token.LBRACE, "{"},
		{token.NEWLINE, "\n"},
		{token.RETURN, "return"},
		{token.IDENT, "a"},
		{token.PLUS, "+"},
		{token.IDENT, "b"},
		{token.NEWLINE, "\n"},
		{token.RBRACE, "}"},
		{token.NEWLINE, "\n"},
		{token.NEWLINE, "\n"},

		// method call
		{token.IDENT, "result"},
		{token.ASSIGN, "="},
		{token.IDENT, "vfs"},
		{token.DOT, "."},
		{token.IDENT, "stat"},
		{token.LPAREN, "("},
		{token.IDENT, "path"},
		{token.RPAREN, ")"},
		{token.NEWLINE, "\n"},

		// array literal
		{token.IDENT, "arr"},
		{token.ASSIGN, "="},
		{token.LBRACKET, "["},
		{token.INT, "1"},
		{token.COMMA, ","},
		{token.INT, "2"},
		{token.COMMA, ","},
		{token.INT, "3"},
		{token.RBRACKET, "]"},
		{token.NEWLINE, "\n"},

		// map literal
		{token.IDENT, "map"},
		{token.ASSIGN, "="},
		{token.LBRACE, "{"},
		{token.IDENT, "name"},
		{token.COLON, ":"},
		{token.STRING, "alice"},
		{token.COMMA, ","},
		{token.IDENT, "age"},
		{token.COLON, ":"},
		{token.INT, "30"},
		{token.RBRACE, "}"},
		{token.NEWLINE, "\n"},

		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q (literal=%q)",
				i, tt.expectedType, tok.Type, tok.Literal)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestOperators(t *testing.T) {
	input := `+ - * / % = == != < > <= >= && || ! |`

	tests := []struct {
		expectedType    token.Type
		expectedLiteral string
	}{
		{token.PLUS, "+"},
		{token.MINUS, "-"},
		{token.ASTERISK, "*"},
		{token.SLASH, "/"},
		{token.PERCENT, "%"},
		{token.ASSIGN, "="},
		{token.EQ, "=="},
		{token.NOT_EQ, "!="},
		{token.LT, "<"},
		{token.GT, ">"},
		{token.LTE, "<="},
		{token.GTE, ">="},
		{token.AND, "&&"},
		{token.OR, "||"},
		{token.BANG, "!"},
		{token.PIPE, "|"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestStringLiterals(t *testing.T) {
	tests := []struct {
		input           string
		expectedType    token.Type
		expectedLiteral string
	}{
		{`"hello"`, token.STRING, "hello"},
		{`"hello world"`, token.STRING, "hello world"},
		{`""`, token.STRING, ""},
		{`"hello\nworld"`, token.STRING, "hello\nworld"},
		{`"hello\tworld"`, token.STRING, "hello\tworld"},
		{`"hello\"world"`, token.STRING, "hello\"world"},
		{`"hello\\world"`, token.STRING, "hello\\world"},
		{`"path/to/file"`, token.STRING, "path/to/file"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("input %q: expected type %q, got %q", tt.input, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestStringInterpolation(t *testing.T) {
	input := `"Hello ${name}!"`
	l := New(input)
	tok := l.NextToken()

	// String with interpolation should be marked as INTERP_START
	if tok.Type != token.INTERP_START {
		t.Fatalf("expected INTERP_START, got %q", tok.Type)
	}

	if tok.Literal != "Hello ${name}!" {
		t.Fatalf("expected literal 'Hello ${name}!', got %q", tok.Literal)
	}
}

func TestNumbers(t *testing.T) {
	tests := []struct {
		input           string
		expectedType    token.Type
		expectedLiteral string
	}{
		{"42", token.INT, "42"},
		{"0", token.INT, "0"},
		{"12345", token.INT, "12345"},
		{"3.14", token.FLOAT, "3.14"},
		{"0.5", token.FLOAT, "0.5"},
		{"100.0", token.FLOAT, "100.0"},
	}

	for _, tt := range tests {
		l := New(tt.input)
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Errorf("input %q: expected type %q, got %q", tt.input, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Errorf("input %q: expected literal %q, got %q", tt.input, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestKeywords(t *testing.T) {
	input := `true false nil if else for while in fn return break continue`

	tests := []struct {
		expectedType    token.Type
		expectedLiteral string
	}{
		{token.TRUE, "true"},
		{token.FALSE, "false"},
		{token.NIL, "nil"},
		{token.IF, "if"},
		{token.ELSE, "else"},
		{token.FOR, "for"},
		{token.WHILE, "while"},
		{token.IN, "in"},
		{token.FN, "fn"},
		{token.RETURN, "return"},
		{token.BREAK, "break"},
		{token.CONTINUE, "continue"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestDelimiters(t *testing.T) {
	input := `( ) [ ] { } , : .`

	tests := []struct {
		expectedType    token.Type
		expectedLiteral string
	}{
		{token.LPAREN, "("},
		{token.RPAREN, ")"},
		{token.LBRACKET, "["},
		{token.RBRACKET, "]"},
		{token.LBRACE, "{"},
		{token.RBRACE, "}"},
		{token.COMMA, ","},
		{token.COLON, ":"},
		{token.DOT, "."},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestComments(t *testing.T) {
	input := `x = 42 # this is a comment
y = 43`

	tests := []struct {
		expectedType    token.Type
		expectedLiteral string
	}{
		{token.IDENT, "x"},
		{token.ASSIGN, "="},
		{token.INT, "42"},
		{token.NEWLINE, "\n"},
		{token.IDENT, "y"},
		{token.ASSIGN, "="},
		{token.INT, "43"},
		{token.EOF, ""},
	}

	l := New(input)

	for i, tt := range tests {
		tok := l.NextToken()

		if tok.Type != tt.expectedType {
			t.Fatalf("tests[%d] - tokentype wrong. expected=%q, got=%q",
				i, tt.expectedType, tok.Type)
		}

		if tok.Literal != tt.expectedLiteral {
			t.Fatalf("tests[%d] - literal wrong. expected=%q, got=%q",
				i, tt.expectedLiteral, tok.Literal)
		}
	}
}

func TestPositionTracking(t *testing.T) {
	input := `x = 42
y = 100`

	l := New(input)

	// First token: x
	tok := l.NextToken()
	if tok.Pos.Line != 1 || tok.Pos.Column != 1 {
		t.Errorf("token 'x': expected line 1, column 1, got line %d, column %d",
			tok.Pos.Line, tok.Pos.Column)
	}

	// Skip to second line's first token
	l.NextToken() // =
	l.NextToken() // 42
	l.NextToken() // \n

	// Token on second line: y
	tok = l.NextToken()
	if tok.Pos.Line != 2 || tok.Pos.Column != 1 {
		t.Errorf("token 'y': expected line 2, column 1, got line %d, column %d",
			tok.Pos.Line, tok.Pos.Column)
	}
}

func TestTokenize(t *testing.T) {
	input := `x = 1 + 2`

	l := New(input)
	tokens, err := l.Tokenize()

	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := []token.Type{
		token.IDENT,
		token.ASSIGN,
		token.INT,
		token.PLUS,
		token.INT,
		token.EOF,
	}

	if len(tokens) != len(expected) {
		t.Fatalf("expected %d tokens, got %d", len(expected), len(tokens))
	}

	for i, tt := range expected {
		if tokens[i].Type != tt {
			t.Errorf("tokens[%d]: expected %q, got %q", i, tt, tokens[i].Type)
		}
	}
}

func TestUnclosedString(t *testing.T) {
	input := `"unclosed string`

	l := New(input)
	l.Tokenize()

	if !l.Errors().HasErrors() {
		t.Fatal("expected error for unclosed string")
	}
}

func TestEscapeSequences(t *testing.T) {
	input := `"hello\nworld\t!\r\"\\\$"`

	l := New(input)
	tok := l.NextToken()

	expected := "hello\nworld\t!\r\"\\$"
	if tok.Literal != expected {
		t.Errorf("expected %q, got %q", expected, tok.Literal)
	}
}
