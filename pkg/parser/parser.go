// Package parser implements a recursive descent parser for Vega.
package parser

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/mwantia/vega/errors"
	"github.com/mwantia/vega/pkg/ast"
	"github.com/mwantia/vega/pkg/token"
)

// Parser parses tokens into an AST.
type Parser struct {
	tokens  []token.Token
	pos     int
	errors  errors.ErrorList
	current token.Token
	peek    token.Token
}

// New creates a new Parser.
func New(tokens []token.Token) *Parser {
	p := &Parser{
		tokens: tokens,
		pos:    0,
	}

	// Initialize current and peek
	if len(tokens) > 0 {
		p.current = tokens[0]
		p.pos = 1
	}
	if len(tokens) > 1 {
		p.peek = tokens[1]
		p.pos = 2
	}

	return p
}

// Errors returns any parsing errors.
func (p *Parser) Errors() errors.ErrorList {
	return p.errors
}

// Parse parses the tokens into an AST Program.
func (p *Parser) Parse() (*ast.Program, error) {
	program := &ast.Program{
		Statements: []ast.Statement{},
	}

	for !p.isAtEnd() {
		// Skip newlines at statement boundaries
		p.skipNewlines()
		if p.isAtEnd() {
			break
		}

		stmt := p.parseStatement()
		if stmt != nil {
			program.Statements = append(program.Statements, stmt)
		}
	}

	if p.errors.HasErrors() {
		return program, p.errors
	}
	return program, nil
}

// advance moves to the next token.
func (p *Parser) advance() token.Token {
	p.current = p.peek
	if p.pos < len(p.tokens) {
		p.peek = p.tokens[p.pos]
	} else {
		p.peek = token.Token{Type: token.EOF}
	}
	p.pos++
	return p.current
}

// isAtEnd returns true if we've reached the end of tokens.
func (p *Parser) isAtEnd() bool {
	return p.current.Type == token.EOF
}

// check returns true if the current token is of the given type.
func (p *Parser) check(t token.Type) bool {
	return p.current.Type == t
}

// checkPeek returns true if the peek token is of the given type.
func (p *Parser) checkPeek(t token.Type) bool {
	return p.peek.Type == t
}

// match advances if the current token matches any of the given types.
func (p *Parser) match(types ...token.Type) bool {
	for _, t := range types {
		if p.check(t) {
			p.advance()
			return true
		}
	}
	return false
}

// expect consumes the current token if it matches, otherwise adds an error.
func (p *Parser) expect(t token.Type) bool {
	if p.check(t) {
		p.advance()
		return true
	}
	p.addError(fmt.Sprintf("expected %s, got %s", t, p.current.Type))
	return false
}

// skipNewlines skips over newline tokens.
func (p *Parser) skipNewlines() {
	for p.check(token.NEWLINE) {
		p.advance()
	}
}

// addError adds a parse error.
func (p *Parser) addError(message string) {
	p.errors.Add(errors.NewParseError(message, p.current.Pos.Line, p.current.Pos.Column))
}

// parseStatement parses a statement.
func (p *Parser) parseStatement() ast.Statement {
	switch p.current.Type {
	case token.IF:
		return p.parseIfStatement()
	case token.FOR:
		return p.parseForStatement()
	case token.WHILE:
		return p.parseWhileStatement()
	case token.FN:
		return p.parseFunctionDefinition()
	case token.RETURN:
		return p.parseReturnStatement()
	case token.BREAK:
		return p.parseBreakStatement()
	case token.CONTINUE:
		return p.parseContinueStatement()
	default:
		return p.parseExpressionOrAssignment()
	}
}

// parseExpressionOrAssignment parses either an assignment or expression statement.
func (p *Parser) parseExpressionOrAssignment() ast.Statement {
	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	// Check for assignment
	if p.check(token.ASSIGN) {
		assignToken := p.current
		p.advance() // consume '='

		value := p.parseExpression(LOWEST)
		if value == nil {
			p.addError("expected expression after '='")
			return nil
		}

		// Check what we're assigning to
		switch left := expr.(type) {
		case *ast.Identifier:
			return &ast.AssignmentStatement{
				Token: assignToken,
				Name:  left,
				Value: value,
			}
		case *ast.IndexExpression:
			return &ast.IndexAssignmentStatement{
				Token: assignToken,
				Left:  left,
				Value: value,
			}
		default:
			p.addError("invalid assignment target")
			return nil
		}
	}

	// It's an expression statement
	stmt := &ast.ExpressionStatement{
		Token:      p.current,
		Expression: expr,
	}

	// Consume optional newline or semicolon
	p.match(token.NEWLINE, token.SEMICOLON)

	return stmt
}

// parseIfStatement parses an if statement.
func (p *Parser) parseIfStatement() *ast.IfStatement {
	stmt := &ast.IfStatement{Token: p.current}
	p.advance() // consume 'if'

	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		return nil
	}

	if !p.expect(token.LBRACE) {
		return nil
	}

	stmt.Consequence = p.parseBlockStatement()

	if p.check(token.ELSE) {
		p.advance() // consume 'else'

		if !p.expect(token.LBRACE) {
			return nil
		}

		stmt.Alternative = p.parseBlockStatement()
	}

	return stmt
}

// parseForStatement parses a for statement.
func (p *Parser) parseForStatement() *ast.ForStatement {
	stmt := &ast.ForStatement{Token: p.current}
	p.advance() // consume 'for'

	if !p.check(token.IDENT) {
		p.addError("expected identifier after 'for'")
		return nil
	}

	stmt.Variable = &ast.Identifier{
		Token: p.current,
		Value: p.current.Literal,
	}
	p.advance()

	if !p.expect(token.IN) {
		return nil
	}

	stmt.Iterable = p.parseExpression(LOWEST)
	if stmt.Iterable == nil {
		return nil
	}

	if !p.expect(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseWhileStatement parses a while statement.
func (p *Parser) parseWhileStatement() *ast.WhileStatement {
	stmt := &ast.WhileStatement{Token: p.current}
	p.advance() // consume 'while'

	stmt.Condition = p.parseExpression(LOWEST)
	if stmt.Condition == nil {
		return nil
	}

	if !p.expect(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseFunctionDefinition parses a function definition.
func (p *Parser) parseFunctionDefinition() *ast.FunctionDefinition {
	stmt := &ast.FunctionDefinition{Token: p.current}
	p.advance() // consume 'fn'

	if !p.check(token.IDENT) {
		p.addError("expected function name")
		return nil
	}

	stmt.Name = &ast.Identifier{
		Token: p.current,
		Value: p.current.Literal,
	}
	p.advance()

	if !p.expect(token.LPAREN) {
		return nil
	}

	stmt.Parameters = p.parseParameterList()

	if !p.expect(token.RPAREN) {
		return nil
	}

	if !p.expect(token.LBRACE) {
		return nil
	}

	stmt.Body = p.parseBlockStatement()

	return stmt
}

// parseParameterList parses function parameters.
func (p *Parser) parseParameterList() []*ast.Identifier {
	var params []*ast.Identifier

	if p.check(token.RPAREN) {
		return params
	}

	// First parameter
	if !p.check(token.IDENT) {
		p.addError("expected parameter name")
		return params
	}

	params = append(params, &ast.Identifier{
		Token: p.current,
		Value: p.current.Literal,
	})
	p.advance()

	// Additional parameters
	for p.check(token.COMMA) {
		p.advance() // consume ','

		if !p.check(token.IDENT) {
			p.addError("expected parameter name after ','")
			return params
		}

		params = append(params, &ast.Identifier{
			Token: p.current,
			Value: p.current.Literal,
		})
		p.advance()
	}

	return params
}

// parseReturnStatement parses a return statement.
func (p *Parser) parseReturnStatement() *ast.ReturnStatement {
	stmt := &ast.ReturnStatement{Token: p.current}
	p.advance() // consume 'return'

	// Optional return value
	if !p.check(token.NEWLINE) && !p.check(token.RBRACE) && !p.check(token.EOF) {
		stmt.Value = p.parseExpression(LOWEST)
	}

	p.match(token.NEWLINE, token.SEMICOLON)

	return stmt
}

// parseBreakStatement parses a break statement.
func (p *Parser) parseBreakStatement() *ast.BreakStatement {
	stmt := &ast.BreakStatement{Token: p.current}
	p.advance() // consume 'break'
	p.match(token.NEWLINE, token.SEMICOLON)
	return stmt
}

// parseContinueStatement parses a continue statement.
func (p *Parser) parseContinueStatement() *ast.ContinueStatement {
	stmt := &ast.ContinueStatement{Token: p.current}
	p.advance() // consume 'continue'
	p.match(token.NEWLINE, token.SEMICOLON)
	return stmt
}

// parseBlockStatement parses a block of statements.
func (p *Parser) parseBlockStatement() *ast.BlockStatement {
	block := &ast.BlockStatement{
		Token:      p.current,
		Statements: []ast.Statement{},
	}

	p.skipNewlines()

	for !p.check(token.RBRACE) && !p.isAtEnd() {
		stmt := p.parseStatement()
		if stmt != nil {
			block.Statements = append(block.Statements, stmt)
		}
		p.skipNewlines()
	}

	p.expect(token.RBRACE)

	return block
}

// parseExpression parses an expression using Pratt parsing.
func (p *Parser) parseExpression(precedence int) ast.Expression {
	// Parse prefix expression
	left := p.parsePrefixExpression()
	if left == nil {
		return nil
	}

	// Parse infix expressions while we have higher precedence
	for !p.isAtEnd() && precedence < getPrecedence(p.current.Type) {
		left = p.parseInfixExpression(left)
		if left == nil {
			return nil
		}
	}

	return left
}

// parsePrefixExpression parses a prefix expression.
func (p *Parser) parsePrefixExpression() ast.Expression {
	switch p.current.Type {
	case token.IDENT:
		return p.parseIdentifier()
	case token.INT:
		return p.parseIntegerLiteral()
	case token.FLOAT:
		return p.parseFloatLiteral()
	case token.STRING:
		return p.parseStringLiteral()
	case token.INTERP_START:
		return p.parseInterpolatedString()
	case token.TRUE, token.FALSE:
		return p.parseBooleanLiteral()
	case token.NIL:
		return p.parseNilLiteral()
	case token.MINUS, token.BANG:
		return p.parseUnaryExpression()
	case token.LPAREN:
		return p.parseGroupedExpression()
	case token.LBRACKET:
		return p.parseArrayLiteral()
	case token.LBRACE:
		return p.parseMapLiteral()
	default:
		p.addError(fmt.Sprintf("unexpected token: %s", p.current.Type))
		p.advance() // Skip the unexpected token to avoid infinite loops
		return nil
	}
}

// parseInfixExpression parses an infix expression.
func (p *Parser) parseInfixExpression(left ast.Expression) ast.Expression {
	switch p.current.Type {
	case token.PLUS, token.MINUS, token.ASTERISK, token.SLASH, token.PERCENT,
		token.EQ, token.NOT_EQ, token.LT, token.GT, token.LTE, token.GTE,
		token.AND, token.OR:
		return p.parseBinaryExpression(left)
	case token.PIPE:
		return p.parsePipeExpression(left)
	case token.LPAREN:
		return p.parseCallExpression(left)
	case token.LBRACKET:
		return p.parseIndexExpression(left)
	case token.DOT:
		return p.parseAttributeOrMethodCall(left)
	default:
		return left
	}
}

// parseIdentifier parses an identifier.
func (p *Parser) parseIdentifier() *ast.Identifier {
	ident := &ast.Identifier{
		Token: p.current,
		Value: p.current.Literal,
	}
	p.advance()
	return ident
}

// parseIntegerLiteral parses an integer literal.
func (p *Parser) parseIntegerLiteral() *ast.IntegerLiteral {
	lit := &ast.IntegerLiteral{Token: p.current}

	value, err := strconv.ParseInt(p.current.Literal, 10, 64)
	if err != nil {
		p.addError(fmt.Sprintf("could not parse %q as integer", p.current.Literal))
		return nil
	}

	lit.Value = value
	p.advance()
	return lit
}

// parseFloatLiteral parses a float literal.
func (p *Parser) parseFloatLiteral() *ast.FloatLiteral {
	lit := &ast.FloatLiteral{Token: p.current}

	value, err := strconv.ParseFloat(p.current.Literal, 64)
	if err != nil {
		p.addError(fmt.Sprintf("could not parse %q as float", p.current.Literal))
		return nil
	}

	lit.Value = value
	p.advance()
	return lit
}

// parseStringLiteral parses a simple string literal.
func (p *Parser) parseStringLiteral() *ast.StringLiteral {
	lit := &ast.StringLiteral{
		Token: p.current,
		Value: p.current.Literal,
	}
	p.advance()
	return lit
}

// parseInterpolatedString parses a string with interpolation.
func (p *Parser) parseInterpolatedString() ast.Expression {
	startToken := p.current
	content := p.current.Literal
	p.advance()

	// Parse the interpolated string content
	parts := p.parseInterpolationParts(content, startToken)

	// If there's only one part and it's a string, return it directly
	if len(parts) == 1 {
		if str, ok := parts[0].(*ast.StringLiteral); ok {
			return str
		}
	}

	return &ast.InterpolatedString{
		Token: startToken,
		Parts: parts,
	}
}

// parseInterpolationParts parses the parts of an interpolated string.
func (p *Parser) parseInterpolationParts(content string, tok token.Token) []ast.Expression {
	var parts []ast.Expression
	var current strings.Builder
	i := 0

	for i < len(content) {
		if i+1 < len(content) && content[i] == '$' && content[i+1] == '{' {
			// Save any accumulated string
			if current.Len() > 0 {
				parts = append(parts, &ast.StringLiteral{
					Token: tok,
					Value: current.String(),
				})
				current.Reset()
			}

			// Find the matching closing brace
			i += 2 // skip "${"
			start := i
			braceCount := 1
			for i < len(content) && braceCount > 0 {
				if content[i] == '{' {
					braceCount++
				} else if content[i] == '}' {
					braceCount--
				}
				if braceCount > 0 {
					i++
				}
			}

			if braceCount != 0 {
				p.addError("unclosed interpolation in string")
				break
			}

			// Parse the expression inside the interpolation
			exprStr := content[start:i]
			exprParser := New(tokenizeString(exprStr))
			expr := exprParser.parseExpression(LOWEST)
			if expr != nil {
				parts = append(parts, expr)
			}

			i++ // skip '}'
		} else {
			current.WriteRune(rune(content[i]))
			i++
		}
	}

	// Add any remaining string
	if current.Len() > 0 {
		parts = append(parts, &ast.StringLiteral{
			Token: tok,
			Value: current.String(),
		})
	}

	return parts
}

// tokenizeString is a helper to tokenize a string for interpolation parsing.
func tokenizeString(s string) []token.Token {
	// Use a simple lexer to tokenize the expression
	// This is imported from the lexer package
	lexer := newSimpleLexer(s)
	return lexer.tokenize()
}

// simpleLexer is a minimal lexer for parsing interpolated expressions.
type simpleLexer struct {
	input    string
	pos      int
	readPos  int
	ch       byte
	line     int
	column   int
}

func newSimpleLexer(input string) *simpleLexer {
	l := &simpleLexer{input: input, line: 1, column: 0}
	l.readChar()
	return l
}

func (l *simpleLexer) readChar() {
	if l.readPos >= len(l.input) {
		l.ch = 0
	} else {
		l.ch = l.input[l.readPos]
	}
	l.pos = l.readPos
	l.readPos++
	l.column++
}

func (l *simpleLexer) peekChar() byte {
	if l.readPos >= len(l.input) {
		return 0
	}
	return l.input[l.readPos]
}

func (l *simpleLexer) tokenize() []token.Token {
	var tokens []token.Token

	for l.ch != 0 {
		l.skipWhitespace()
		if l.ch == 0 {
			break
		}

		pos := token.Position{Line: l.line, Column: l.column, Offset: l.pos}
		var tok token.Token

		switch l.ch {
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
		case '(':
			tok = token.Token{Type: token.LPAREN, Literal: "(", Pos: pos}
		case ')':
			tok = token.Token{Type: token.RPAREN, Literal: ")", Pos: pos}
		case '[':
			tok = token.Token{Type: token.LBRACKET, Literal: "[", Pos: pos}
		case ']':
			tok = token.Token{Type: token.RBRACKET, Literal: "]", Pos: pos}
		case '.':
			tok = token.Token{Type: token.DOT, Literal: ".", Pos: pos}
		case ',':
			tok = token.Token{Type: token.COMMA, Literal: ",", Pos: pos}
		default:
			if isLetter(l.ch) {
				ident := l.readIdent()
				tokType := token.LookupIdent(ident)
				tokens = append(tokens, token.Token{Type: tokType, Literal: ident, Pos: pos})
				continue
			} else if isDigit(l.ch) {
				num, isFloat := l.readNumber()
				if isFloat {
					tokens = append(tokens, token.Token{Type: token.FLOAT, Literal: num, Pos: pos})
				} else {
					tokens = append(tokens, token.Token{Type: token.INT, Literal: num, Pos: pos})
				}
				continue
			}
			tok = token.Token{Type: token.ILLEGAL, Literal: string(l.ch), Pos: pos}
		}

		tokens = append(tokens, tok)
		l.readChar()
	}

	tokens = append(tokens, token.Token{Type: token.EOF, Pos: token.Position{Line: l.line, Column: l.column}})
	return tokens
}

func (l *simpleLexer) skipWhitespace() {
	for l.ch == ' ' || l.ch == '\t' || l.ch == '\r' || l.ch == '\n' {
		l.readChar()
	}
}

func (l *simpleLexer) readIdent() string {
	start := l.pos
	for isLetter(l.ch) || isDigit(l.ch) || l.ch == '_' {
		l.readChar()
	}
	return l.input[start:l.pos]
}

func (l *simpleLexer) readNumber() (string, bool) {
	start := l.pos
	isFloat := false
	for isDigit(l.ch) {
		l.readChar()
	}
	if l.ch == '.' && isDigit(l.peekChar()) {
		isFloat = true
		l.readChar()
		for isDigit(l.ch) {
			l.readChar()
		}
	}
	return l.input[start:l.pos], isFloat
}

func isLetter(ch byte) bool {
	return (ch >= 'a' && ch <= 'z') || (ch >= 'A' && ch <= 'Z') || ch == '_'
}

func isDigit(ch byte) bool {
	return ch >= '0' && ch <= '9'
}

// parseBooleanLiteral parses true or false.
func (p *Parser) parseBooleanLiteral() *ast.BooleanLiteral {
	lit := &ast.BooleanLiteral{
		Token: p.current,
		Value: p.current.Type == token.TRUE,
	}
	p.advance()
	return lit
}

// parseNilLiteral parses nil.
func (p *Parser) parseNilLiteral() *ast.NilLiteral {
	lit := &ast.NilLiteral{Token: p.current}
	p.advance()
	return lit
}

// parseUnaryExpression parses a unary expression like -x or !x.
func (p *Parser) parseUnaryExpression() *ast.PrefixExpression {
	expr := &ast.PrefixExpression{
		Token:    p.current,
		Operator: p.current.Literal,
	}
	p.advance()

	expr.Right = p.parseExpression(UNARY)
	return expr
}

// parseGroupedExpression parses a parenthesized expression.
func (p *Parser) parseGroupedExpression() ast.Expression {
	tok := p.current
	p.advance() // consume '('

	expr := p.parseExpression(LOWEST)
	if expr == nil {
		return nil
	}

	if !p.expect(token.RPAREN) {
		return nil
	}

	return &ast.GroupedExpression{
		Token:      tok,
		Expression: expr,
	}
}

// parseArrayLiteral parses an array literal.
func (p *Parser) parseArrayLiteral() *ast.ArrayLiteral {
	arr := &ast.ArrayLiteral{Token: p.current}
	p.advance() // consume '['

	arr.Elements = p.parseExpressionList(token.RBRACKET)

	return arr
}

// parseMapLiteral parses a map literal.
func (p *Parser) parseMapLiteral() *ast.MapLiteral {
	m := &ast.MapLiteral{
		Token: p.current,
		Pairs: make(map[string]ast.Expression),
		Order: []string{},
	}
	p.advance() // consume '{'

	p.skipNewlines()

	if p.check(token.RBRACE) {
		p.advance()
		return m
	}

	// Parse first key-value pair
	if !p.check(token.IDENT) && !p.check(token.STRING) {
		p.addError("expected identifier or string as map key")
		return nil
	}

	key := p.current.Literal
	if p.check(token.STRING) {
		key = p.current.Literal
	}
	p.advance()

	if !p.expect(token.COLON) {
		return nil
	}

	value := p.parseExpression(LOWEST)
	if value == nil {
		return nil
	}

	m.Pairs[key] = value
	m.Order = append(m.Order, key)

	// Parse additional pairs
	for p.check(token.COMMA) {
		p.advance() // consume ','
		p.skipNewlines()

		if p.check(token.RBRACE) {
			break
		}

		if !p.check(token.IDENT) && !p.check(token.STRING) {
			p.addError("expected identifier or string as map key")
			return nil
		}

		key = p.current.Literal
		p.advance()

		if !p.expect(token.COLON) {
			return nil
		}

		value = p.parseExpression(LOWEST)
		if value == nil {
			return nil
		}

		m.Pairs[key] = value
		m.Order = append(m.Order, key)
	}

	p.skipNewlines()
	p.expect(token.RBRACE)

	return m
}

// parseExpressionList parses a comma-separated list of expressions.
func (p *Parser) parseExpressionList(end token.Type) []ast.Expression {
	var list []ast.Expression

	p.skipNewlines()

	if p.check(end) {
		p.advance()
		return list
	}

	list = append(list, p.parseExpression(LOWEST))

	for p.check(token.COMMA) {
		p.advance() // consume ','
		p.skipNewlines()

		if p.check(end) {
			break
		}

		list = append(list, p.parseExpression(LOWEST))
	}

	p.skipNewlines()
	p.expect(end)

	return list
}

// parseBinaryExpression parses a binary expression.
func (p *Parser) parseBinaryExpression(left ast.Expression) ast.Expression {
	expr := &ast.InfixExpression{
		Token:    p.current,
		Left:     left,
		Operator: p.current.Literal,
	}

	prec := getPrecedence(p.current.Type)
	p.advance()

	expr.Right = p.parseExpression(prec)
	return expr
}

// parsePipeExpression parses a pipe expression.
func (p *Parser) parsePipeExpression(left ast.Expression) ast.Expression {
	expr := &ast.PipeExpression{
		Token: p.current,
		Left:  left,
	}

	p.advance() // consume '|'

	expr.Right = p.parseExpression(PIPE)
	return expr
}

// parseCallExpression parses a function call.
func (p *Parser) parseCallExpression(fn ast.Expression) *ast.CallExpression {
	call := &ast.CallExpression{
		Token:    p.current,
		Function: fn,
	}
	p.advance() // consume '('

	call.Arguments = p.parseExpressionList(token.RPAREN)

	return call
}

// parseIndexExpression parses an index expression.
func (p *Parser) parseIndexExpression(left ast.Expression) *ast.IndexExpression {
	expr := &ast.IndexExpression{
		Token: p.current,
		Left:  left,
	}
	p.advance() // consume '['

	expr.Index = p.parseExpression(LOWEST)

	p.expect(token.RBRACKET)

	return expr
}

// parseAttributeOrMethodCall parses attribute access or method call.
func (p *Parser) parseAttributeOrMethodCall(obj ast.Expression) ast.Expression {
	p.advance() // consume '.'

	if !p.check(token.IDENT) {
		p.addError("expected identifier after '.'")
		return nil
	}

	attr := &ast.Identifier{
		Token: p.current,
		Value: p.current.Literal,
	}
	p.advance()

	// Check if it's a method call
	if p.check(token.LPAREN) {
		call := &ast.MethodCallExpression{
			Token:  p.current,
			Object: obj,
			Method: attr,
		}
		p.advance() // consume '('

		call.Arguments = p.parseExpressionList(token.RPAREN)
		return call
	}

	// It's an attribute access
	return &ast.AttributeExpression{
		Token:     attr.Token,
		Object:    obj,
		Attribute: attr,
	}
}
