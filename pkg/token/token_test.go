package token

import "testing"

func TestLookupIdent(t *testing.T) {
	tests := []struct {
		input    string
		expected Type
	}{
		{"true", TRUE},
		{"false", FALSE},
		{"nil", NIL},
		{"if", IF},
		{"else", ELSE},
		{"for", FOR},
		{"while", WHILE},
		{"in", IN},
		{"fn", FN},
		{"return", RETURN},
		{"break", BREAK},
		{"continue", CONTINUE},
		{"foo", IDENT},
		{"bar", IDENT},
		{"vfs", IDENT},
		{"x", IDENT},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := LookupIdent(tt.input)
			if got != tt.expected {
				t.Errorf("LookupIdent(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestTokenIsKeyword(t *testing.T) {
	tests := []struct {
		tokenType Type
		expected  bool
	}{
		{TRUE, true},
		{FALSE, true},
		{NIL, true},
		{IF, true},
		{ELSE, true},
		{FOR, true},
		{WHILE, true},
		{IN, true},
		{FN, true},
		{RETURN, true},
		{BREAK, true},
		{CONTINUE, true},
		{IDENT, false},
		{INT, false},
		{STRING, false},
		{PLUS, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.tokenType), func(t *testing.T) {
			tok := Token{Type: tt.tokenType}
			if got := tok.IsKeyword(); got != tt.expected {
				t.Errorf("Token{Type: %v}.IsKeyword() = %v, want %v", tt.tokenType, got, tt.expected)
			}
		})
	}
}

func TestTokenIsOperator(t *testing.T) {
	tests := []struct {
		tokenType Type
		expected  bool
	}{
		{ASSIGN, true},
		{PLUS, true},
		{MINUS, true},
		{ASTERISK, true},
		{SLASH, true},
		{PERCENT, true},
		{BANG, true},
		{EQ, true},
		{NOT_EQ, true},
		{LT, true},
		{GT, true},
		{LTE, true},
		{GTE, true},
		{AND, true},
		{OR, true},
		{PIPE, true},
		{IDENT, false},
		{INT, false},
		{IF, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.tokenType), func(t *testing.T) {
			tok := Token{Type: tt.tokenType}
			if got := tok.IsOperator(); got != tt.expected {
				t.Errorf("Token{Type: %v}.IsOperator() = %v, want %v", tt.tokenType, got, tt.expected)
			}
		})
	}
}

func TestTokenIsLiteral(t *testing.T) {
	tests := []struct {
		tokenType Type
		expected  bool
	}{
		{INT, true},
		{FLOAT, true},
		{STRING, true},
		{TRUE, true},
		{FALSE, true},
		{NIL, true},
		{IDENT, false},
		{PLUS, false},
		{IF, false},
	}

	for _, tt := range tests {
		t.Run(string(tt.tokenType), func(t *testing.T) {
			tok := Token{Type: tt.tokenType}
			if got := tok.IsLiteral(); got != tt.expected {
				t.Errorf("Token{Type: %v}.IsLiteral() = %v, want %v", tt.tokenType, got, tt.expected)
			}
		})
	}
}
