package parser

import (
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/mwantia/vega/pkg/lexer"
)

type Parser struct {
}

func NewParser() *Parser {
	return &Parser{}
}

func (p *Parser) MakeProgram(buffer lexer.TokenBuffer) (AST, error) {
	program := NewProgram()

	for !buffer.EndReached() {
		// Skip any newlines
		buffer.SkipAny(lexer.NEWLINE)
		if buffer.EndReached() {
			break
		}

		statement, err := p.makeStatement(buffer)
		if err != nil {
			return nil, fmt.Errorf("failed to parse statement: %v", err)
		}
		if statement != nil {
			program.statements = append(program.statements, statement)
		}
	}

	return program, nil
}

func (p *Parser) makeStatement(b lexer.TokenBuffer) (Statement, error) {
	current := b.Current()
	switch current.Type {
	case lexer.IF:
		return p.makeIfStatement(b)
	case lexer.FOR:
		return p.makeForStatement(b)
	case lexer.WHILE:
		return p.makeWhileStatement(b)
	case lexer.FN:
		return p.makeFunctionStatement(b)
	case lexer.RETURN:
		return p.makeReturnStatement(b)
	case lexer.BREAK:
		return p.makeBreakStatement(b)
	case lexer.CONTINUE:
		return p.makeContinueStatement(b)
	case lexer.ALLOC:
		return p.makeAllocStatement(b)
	case lexer.FREE:
		return p.makeFreeStatement(b)
	case lexer.STRUCT:
		return p.makeStructStatement(b)
	default:
		return p.makeExpressionOrAssignment(b)
	}
}

func (p *Parser) makeIfStatement(b lexer.TokenBuffer) (*IfStatement, error) {
	statement := &IfStatement{
		Token: b.Current(),
	}
	// Consume 'if' token
	b.Read()

	condition, err := p.makeExpression(b, LOWEST)
	if err != nil {
		return nil, fmt.Errorf("empty expression defined for 'if': %v", err)
	}
	statement.Condition = condition

	if !b.MatchAny(false, lexer.LBRACE) {
		return nil, fmt.Errorf("expected '{', but received '%s'", b.Current().Literal)
	}

	consequence, err := p.makeBlockStatement(b)
	if err != nil {
		return nil, err
	}
	statement.Consequence = consequence
	// Match and consume 'else' token
	if b.MatchAny(true, lexer.ELSE) {
		if !b.MatchAny(false, lexer.LBRACE) {
			return nil, fmt.Errorf("expected '{', but received '%s'", b.Current().Literal)
		}
		alternative, err := p.makeBlockStatement(b)
		if err != nil {
			return nil, fmt.Errorf("empty statement defined for 'else': %v", err)
		}
		statement.Alternative = alternative
	}
	return statement, nil
}

func (p *Parser) makeForStatement(b lexer.TokenBuffer) (*ForStatement, error) {
	statement := &ForStatement{
		Token: b.Current(),
	}
	b.Peek()

	if !b.MatchAny(false, lexer.IDENT) {
		return nil, fmt.Errorf("expected identifier after 'for', but received '%s'", b.Current().Literal)
	}

	token := b.Current()
	statement.Variable = &IdentifierExpression{
		Token: token,
		Value: token.Literal,
	}
	b.Read()

	if !b.MatchAny(false, lexer.IN) {
		return nil, fmt.Errorf("expected 'in', but received '%s'", b.Current().Literal)
	}

	b.Read()
	iterable, err := p.makeExpression(b, LOWEST)
	if err != nil {
		return nil, fmt.Errorf("failed to make iterable expression: %v", err)
	}
	statement.Iterable = iterable

	if !b.MatchAny(false, lexer.LBRACE) {
		return nil, fmt.Errorf("expected '{', but received '%s'", b.Current().Literal)
	}

	b.Read()
	body, err := p.makeBlockStatement(b)
	if err != nil {
		return nil, fmt.Errorf("failed to make block statement: %v", err)
	}
	statement.Body = body
	return statement, nil
}

func (p *Parser) makeWhileStatement(b lexer.TokenBuffer) (*WhileStatement, error) {
	statement := &WhileStatement{
		Token: b.Current(),
	}
	b.Read()

	condition, err := p.makeExpression(b, LOWEST)
	if err != nil {
		return nil, fmt.Errorf("failed to make condition: %v", err)
	}
	statement.Condition = condition

	if !b.MatchAny(false, lexer.LBRACE) {
		return nil, fmt.Errorf("expected '{', but received '%s'", b.Current().Literal)
	}

	b.Read()
	body, err := p.makeBlockStatement(b)
	if err != nil {
		return nil, fmt.Errorf("failed to make block statement: %v", err)
	}
	statement.Body = body
	return statement, nil
}

func (p *Parser) makeFunctionStatement(b lexer.TokenBuffer) (*FunctionStatement, error) {
	statement := &FunctionStatement{
		Token: b.Current(),
	}
	b.Read()

	if !b.MatchAny(false, lexer.IDENT) {
		return nil, fmt.Errorf("expected identifier after 'fn', but received '%s'", b.Current().Literal)
	}

	token := b.Current()
	statement.Name = &IdentifierExpression{
		Token: token,
		Value: b.Current().Literal,
	}
	b.Read()

	if !b.MatchAny(false, lexer.LPAREN) {
		return nil, fmt.Errorf("expected '(', but received '%s'", b.Current().Literal)
	}

	b.Read()
	params, err := p.makeParameterList(b)
	if err != nil {
		return nil, fmt.Errorf("failed to make parameter list: %v", err)
	}
	statement.Parameters = params
	if !b.MatchAny(false, lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')', but received '%s'", b.Current().Literal)
	}

	b.Read()
	if !b.MatchAny(false, lexer.LBRACE) {
		return nil, fmt.Errorf("expected '{', but received '%s'", b.Current().Literal)
	}

	b.Read()
	body, err := p.makeBlockStatement(b)
	if err != nil {
		return nil, fmt.Errorf("failed to make block statement: %v", err)
	}
	statement.Body = body
	return statement, nil
}

func (p *Parser) makeParameterList(b lexer.TokenBuffer) ([]*DeclarationExpression, error) {
	params := make([]*DeclarationExpression, 0)

	if b.MatchAny(true, lexer.RPAREN) {
		return params, nil
	}

	param, err := p.makeDeclarationExpression(b)
	if err != nil {
		return nil, fmt.Errorf("failed to make declaration expression: %v", err)
	}

	params = append(params, param)
	for b.MatchAny(true, lexer.COMMA) {
		param, err = p.makeDeclarationExpression(b)
		if err != nil {
			return nil, fmt.Errorf("failed to make declaration expression: %v", err)
		}
		params = append(params, param)
	}

	return params, nil
}

func (p *Parser) makeDeclarationExpression(b lexer.TokenBuffer) (*DeclarationExpression, error) {
	token := b.Current()
	expr := &DeclarationExpression{
		Token:       token,
		Value:       token.Literal,
		Constraints: make([]Expression, 0),
	}

	if b.MatchAny(true, lexer.COLON) {
		constraint, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("failed to make expression constraint: %v", err)
		}
		expr.Constraints = append(expr.Constraints, constraint)
		for b.MatchAny(true, lexer.PIPE) {
			constraint, err = p.makeExpression(b, LOWEST)
			if err != nil {
				return nil, fmt.Errorf("failed to make expression constraint: %v", err)
			}
			expr.Constraints = append(expr.Constraints, constraint)
		}
	}

	return expr, nil
}

func (p *Parser) makeBlockStatement(b lexer.TokenBuffer) (*BlockStatement, error) {
	block := &BlockStatement{
		Token:      b.Current(),
		Statements: make([]Statement, 0),
	}

	b.SkipAny(lexer.NEWLINE)
	for !b.MatchAny(true, lexer.RBRACE) && !b.EndReached() {
		statement, err := p.makeStatement(b)
		if err != nil {
			return nil, err
		}
		if statement != nil {
			block.Statements = append(block.Statements, statement)
		}
		b.SkipAny(lexer.NEWLINE)
	}

	return block, nil
}

func (p *Parser) makeReturnStatement(b lexer.TokenBuffer) (*ReturnStatement, error) {
	statement := &ReturnStatement{
		Token: b.Current(),
	}
	b.Read()
	// Optional return value
	if !b.MatchAny(true, lexer.NEWLINE, lexer.RBRACE) && !b.EndReached() {
		value, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("empty expression defined for 'return': %v", err)
		}
		statement.Value = value
	}
	b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
	return statement, nil
}

func (p *Parser) makeBreakStatement(b lexer.TokenBuffer) (*BreakStatement, error) {
	statement := &BreakStatement{
		Token: b.Current(),
	}
	b.Read()
	b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
	return statement, nil
}

func (p *Parser) makeContinueStatement(b lexer.TokenBuffer) (*ContinueStatement, error) {
	statement := &ContinueStatement{
		Token: b.Current(),
	}
	b.Read()
	b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
	return statement, nil
}

func (p *Parser) makeAllocStatement(b lexer.TokenBuffer) (*AllocStatement, error) {
	statement := &AllocStatement{
		Token: b.Current(),
	}
	b.Read()

	size, err := p.makeExpression(b, LOWEST)
	if err != nil {
		return nil, fmt.Errorf("expected size expression after 'alloc': %v", err)
	}
	statement.Size = size

	if !b.MatchAny(false, lexer.LBRACE) {
		return nil, fmt.Errorf("expected '{' after alloc size, but received '%s'", b.Current().Literal)
	}

	b.Read()
	body, err := p.makeBlockStatement(b)
	if err != nil {
		return nil, fmt.Errorf("failed to parse alloc body: %v", err)
	}
	statement.Body = body
	return statement, nil
}

func (p *Parser) makeFreeStatement(b lexer.TokenBuffer) (*FreeStatement, error) {
	statement := &FreeStatement{
		Token: b.Current(),
	}
	b.Read() // consume 'free'

	if !b.MatchAny(true, lexer.LPAREN) {
		return nil, fmt.Errorf("expected '(' after 'free', but received '%s'", b.Current().Literal)
	}

	if !b.MatchAny(false, lexer.IDENT) {
		return nil, fmt.Errorf("expected identifier in 'free(...)', but received '%s'", b.Current().Literal)
	}

	token := b.Current()
	statement.Name = &IdentifierExpression{
		Token: token,
		Value: token.Literal,
	}
	b.Read() // consume identifier

	if !b.MatchAny(true, lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')' after identifier in 'free(...)', but received '%s'", b.Current().Literal)
	}

	b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
	return statement, nil
}

func (p *Parser) makeStructExpression(b lexer.TokenBuffer, token lexer.Token, name string) (*StructExpression, error) {
	expr := &StructExpression{
		Token:  token,
		Name:   name,
		Fields: make(map[string]Expression),
		Order:  make([]string, 0),
	}

	b.SkipAny(lexer.NEWLINE)
	for !b.MatchAny(false, lexer.RBRACE) && !b.EndReached() {
		if !b.MatchAny(false, lexer.IDENT) {
			return nil, fmt.Errorf("expected field name in struct literal, but received '%s'", b.Current().Literal)
		}
		fieldName := b.Current().Literal
		b.Read() // consume field name

		if !b.MatchAny(true, lexer.ASSIGN) {
			return nil, fmt.Errorf("expected '=' after field name '%s', but received '%s'", fieldName, b.Current().Literal)
		}

		val, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("failed to parse value for field '%s': %v", fieldName, err)
		}

		expr.Fields[fieldName] = val
		expr.Order = append(expr.Order, fieldName)

		// Consume comma or newline separators
		b.MatchAny(true, lexer.COMMA)
		b.SkipAny(lexer.NEWLINE)
	}

	if !b.MatchAny(true, lexer.RBRACE) {
		return nil, fmt.Errorf("expected '}' to close struct literal, but received '%s'", b.Current().Literal)
	}

	return expr, nil
}

func (p *Parser) makeStructStatement(b lexer.TokenBuffer) (*StructStatement, error) {
	statement := &StructStatement{
		Token: b.Current(),
	}
	b.Read() // consume 'struct'

	if !b.MatchAny(false, lexer.IDENT) {
		return nil, fmt.Errorf("expected struct name after 'struct', but received '%s'", b.Current().Literal)
	}
	statement.Name = b.Current().Literal
	b.Read() // consume name

	if !b.MatchAny(true, lexer.LBRACE) {
		return nil, fmt.Errorf("expected '{' after struct name, but received '%s'", b.Current().Literal)
	}

	b.SkipAny(lexer.NEWLINE)
	for !b.MatchAny(false, lexer.RBRACE) && !b.EndReached() {
		if !b.MatchAny(false, lexer.IDENT) {
			return nil, fmt.Errorf("expected field name in struct, but received '%s'", b.Current().Literal)
		}
		fieldName := b.Current().Literal
		b.Read() // consume field name

		if !b.MatchAny(true, lexer.COLON) {
			return nil, fmt.Errorf("expected ':' after field name '%s', but received '%s'", fieldName, b.Current().Literal)
		}

		if !b.MatchAny(false, lexer.IDENT) {
			return nil, fmt.Errorf("expected type name for field '%s', but received '%s'", fieldName, b.Current().Literal)
		}
		typeName := b.Current().Literal
		b.Read() // consume type name

		statement.Fields = append(statement.Fields, StructField{
			Name: fieldName,
			Type: typeName,
		})

		// Consume comma or newline separators
		b.MatchAny(true, lexer.COMMA)
		b.SkipAny(lexer.NEWLINE)
	}

	if !b.MatchAny(true, lexer.RBRACE) {
		return nil, fmt.Errorf("expected '}' to close struct definition, but received '%s'", b.Current().Literal)
	}

	b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
	return statement, nil
}

func (p *Parser) makeArgumentList(b lexer.TokenBuffer) ([]Expression, error) {
	args := make([]Expression, 0)

	if b.MatchAny(true, lexer.RPAREN) {
		return args, nil
	}

	arg, err := p.makeExpression(b, LOWEST)
	if err != nil {
		return nil, fmt.Errorf("failed to parse argument: %v", err)
	}
	args = append(args, arg)

	for b.MatchAny(true, lexer.COMMA) {
		arg, err = p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("failed to parse argument: %v", err)
		}
		args = append(args, arg)
	}

	if !b.MatchAny(true, lexer.RPAREN) {
		return nil, fmt.Errorf("expected ')' after arguments, but received '%s'", b.Current().Literal)
	}

	return args, nil
}

func (p *Parser) makeExpression(b lexer.TokenBuffer, precedence int) (Expression, error) {
	left, err := p.makePrefixExpression(b)
	if err != nil {
		return nil, err
	}

	for !b.EndReached() && precedence < GetTokenPrecedence(b.Current()) {
		left, err = p.makeInfixExpression(b, left)
		if err != nil {
			return nil, err
		}
	}

	return left, nil
}

func (p *Parser) makePrefixExpression(b lexer.TokenBuffer) (Expression, error) {
	token := b.Current()
	switch token.Type {
	case lexer.IDENT:
		identifier := &IdentifierExpression{
			Token: token,
			Value: token.Literal,
		}
		b.Read()

		// If followed by '{', parse as struct literal: name { field = expr, ... }
		if b.MatchAny(true, lexer.LBRACE) {
			return p.makeStructExpression(b, token, identifier.Value)
		}

		return identifier, nil
	case lexer.BYTE:
		v, err := strconv.ParseUint(token.Literal, 0, 8)
		if err != nil {
			return nil, fmt.Errorf("failed to parse byte: %v", err)
		}
		b.Read()
		return &ByteExpression{Token: token, Value: byte(v)}, nil
	case lexer.SHORT:
		v, err := strconv.ParseInt(token.Literal, 10, 16)
		if err != nil {
			return nil, fmt.Errorf("failed to parse short: %v", err)
		}
		b.Read()
		return &ShortExpression{Token: token, Value: int16(v)}, nil
	case lexer.INTEGER:
		v, err := strconv.ParseInt(token.Literal, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse integer: %v", err)
		}
		b.Read()
		return &IntegerExpression{Token: token, Value: int32(v)}, nil
	case lexer.LONG:
		v, err := strconv.ParseInt(token.Literal, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse long: %v", err)
		}
		b.Read()
		return &LongExpression{Token: token, Value: v}, nil
	case lexer.FLOAT:
		v, err := strconv.ParseFloat(token.Literal, 32)
		if err != nil {
			return nil, fmt.Errorf("failed to parse float: %v", err)
		}
		b.Read()
		return &FloatExpression{Token: token, Value: float32(v)}, nil
	case lexer.DECIMAL:
		v, err := strconv.ParseFloat(token.Literal, 64)
		if err != nil {
			return nil, fmt.Errorf("failed to parse decimal: %v", err)
		}
		b.Read()
		return &DecimalExpression{Token: token, Value: v}, nil
	case lexer.CHAR:
		ch, _ := utf8.DecodeRuneInString(token.Literal)
		b.Read()
		return &CharExpression{Token: token, Value: ch}, nil
	case lexer.STRING:
		str := &StringExpression{
			Token: token,
			Value: token.Literal,
		}
		b.Read()
		return str, nil
	case lexer.INTERP_START:
		// We keep the old token stored in 'token'
		b.Read()

		parts := make([]Expression, 0)
		// TODO
		return &InterpolatedExpression{
			Token: token,
			Parts: parts,
		}, nil
	case lexer.TRUE, lexer.FALSE:
		boolean := &BooleanExpression{
			Token: token,
			Value: token.Type == lexer.TRUE,
		}
		b.Read()
		return boolean, nil
	case lexer.NIL:
		n := &NilExpression{
			Token: token,
		}
		b.Read()
		return n, nil
	case lexer.ASTERISK:
		b.Read() // consume '*'
		if !b.MatchAny(false, lexer.IDENT) {
			return nil, fmt.Errorf("expected type name after '*', but received '%s'", b.Current().Literal)
		}
		typeName := b.Current().Literal
		b.Read() // consume type name
		if !b.MatchAny(true, lexer.LPAREN) {
			return nil, fmt.Errorf("expected '(' after type name in pointer, but received '%s'", b.Current().Literal)
		}
		offset, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("failed to parse pointer offset: %v", err)
		}
		if !b.MatchAny(true, lexer.RPAREN) {
			return nil, fmt.Errorf("expected ')' after pointer offset, but received '%s'", b.Current().Literal)
		}
		return &PointerExpression{
			Token:    token,
			TypeName: typeName,
			Offset:   offset,
		}, nil
	case lexer.MINUS, lexer.BANG:
		prefix := &PrefixExpression{
			Token:    token,
			Operator: token.Literal,
		}
		b.Read()
		right, err := p.makeExpression(b, UNARY)
		if err != nil {
			return nil, fmt.Errorf("failed to make expression for 'unary': %v", err)
		}
		prefix.Right = right
		return prefix, nil
	case lexer.LPAREN:
		b.Read()

		expr, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("failed to make group expression: %v", err)
		}

		// If a comma follows, this is a tuple expression
		if b.MatchAny(true, lexer.COMMA) {
			elements := []Expression{expr}
			for {
				if b.MatchAny(false, lexer.RPAREN) {
					break
				}
				elem, err := p.makeExpression(b, LOWEST)
				if err != nil {
					return nil, fmt.Errorf("failed to parse tuple element: %v", err)
				}
				elements = append(elements, elem)
				if !b.MatchAny(true, lexer.COMMA) {
					break
				}
			}
			if !b.MatchAny(true, lexer.RPAREN) {
				return nil, fmt.Errorf("expected ')' to close tuple, but received '%s'", b.Current().Literal)
			}
			return &TupleExpression{
				Token:    token,
				Elements: elements,
			}, nil
		}

		if !b.MatchAny(false, lexer.RPAREN) {
			return nil, fmt.Errorf("expected ')', but received '%s'", b.Current().Literal)
		}

		return &GroupedExpression{
			Token: token,
			Expr:  expr,
		}, nil
	case lexer.LBRACKET:
		arr := &ArrayExpression{
			Token: token,
		}
		b.Read()
		arr.Elements = nil // TODO
		return arr, nil
	case lexer.LBRACE:
		return nil, nil
	}

	b.Read()
	return nil, nil
}

func (p *Parser) makeInfixExpression(b lexer.TokenBuffer, left Expression) (Expression, error) {
	token := b.Current()
	switch token.Type {
	case lexer.LPAREN:
		b.Read() // consume '('
		args, err := p.makeArgumentList(b)
		if err != nil {
			return nil, fmt.Errorf("failed to parse call arguments: %v", err)
		}
		return &CallExpression{
			Token:     token,
			Function:  left,
			Arguments: args,
		}, nil
	case lexer.DOT:
		b.Read() // consume '.'
		if !b.MatchAny(false, lexer.IDENT, lexer.INTEGER) {
			return nil, fmt.Errorf("expected field name after '.', but received '%s'", b.Current().Literal)
		}
		attrToken := b.Current()
		attr := &IdentifierExpression{
			Token: attrToken,
			Value: attrToken.Literal,
		}
		b.Read() // consume attribute name

		// Check if this is a method call: obj.method(...)
		if b.MatchAny(true, lexer.LPAREN) {
			args, err := p.makeArgumentList(b)
			if err != nil {
				return nil, fmt.Errorf("failed to parse method arguments: %v", err)
			}
			return &MethodCallExpression{
				Token:     token,
				Object:    left,
				Method:    attr,
				Arguments: args,
			}, nil
		}

		return &AttributeExpression{
			Token:     token,
			Object:    left,
			Attribute: attr,
		}, nil
	default:
		return nil, fmt.Errorf("unexpected infix operator '%s'", token.Literal)
	}
}

func (p *Parser) makeExpressionOrAssignment(b lexer.TokenBuffer) (Statement, error) {
	expr, err := p.makeExpression(b, LOWEST)
	if err != nil {
		return nil, err
	}
	token := b.Current()

	// Check for typed assignment: ident: type|type = expr
	if ident, ok := expr.(*IdentifierExpression); ok && b.MatchAny(true, lexer.COLON) {
		constraints := make([]Expression, 0)
		constraint, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("expected type constraint after ':': %v", err)
		}
		constraints = append(constraints, constraint)
		for b.MatchAny(true, lexer.PIPE) {
			constraint, err = p.makeExpression(b, LOWEST)
			if err != nil {
				return nil, fmt.Errorf("expected type constraint after '|': %v", err)
			}
			constraints = append(constraints, constraint)
		}

		if !b.MatchAny(true, lexer.ASSIGN) {
			return nil, fmt.Errorf("expected '=' after type constraints, but received '%s'", b.Current().Literal)
		}

		value, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("expected expression after '=': %v", err)
		}

		b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
		return &AssignmentStatement{
			Token:       token,
			Name:        ident,
			Constraints: constraints,
			Value:       value,
		}, nil
	}

	// Match and consume '=' token
	if b.MatchAny(true, lexer.ASSIGN) {
		value, err := p.makeExpression(b, LOWEST)
		if err != nil {
			return nil, fmt.Errorf("expected expression during ASSIGN after '=': %v", err)
		}

		// Discard statement: _ = expr
		if ident, ok := expr.(*IdentifierExpression); ok && ident.Value == "_" {
			b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
			return &DiscardStatement{
				Token: token,
				Value: value,
			}, nil
		}

		switch left := expr.(type) {
		case *IdentifierExpression:
			b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
			return &AssignmentStatement{
				Token: token,
				Name:  left,
				Value: value,
			}, nil
		case *IndexExpression:
			b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
			return &IndexAssignmentStatement{
				Token: token,
				Left:  left,
				Value: value,
			}, nil
		}
		return nil, fmt.Errorf("invalid assignment target defined")
	}

	// Call expression used as a statement: fn(args)
	if call, ok := expr.(*CallExpression); ok {
		b.MatchAny(true, lexer.NEWLINE, lexer.SEMICOLON)
		return &CallStatement{
			Token:     call.Token,
			Function:  call.Function,
			Arguments: call.Arguments,
		}, nil
	}

	// Bare expression statements are not allowed
	return nil, fmt.Errorf("bare expression statements are not allowed at line %d; use assignment, discard '_ = ...', or function call",
		expr.Position().Line)
}
