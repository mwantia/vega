package parser

import (
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/lexer"
)

type IdentifierExpression struct {
	Token lexer.Token
	Value string
}

// Expression implements Expression.
func (*IdentifierExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (i *IdentifierExpression) Literal() string {
	return i.Token.Literal
}

// Position implements Expression.
func (i *IdentifierExpression) Position() lexer.TokenPosition {
	return i.Token.Position
}

// String implements Expression.
func (i *IdentifierExpression) String() string {
	return i.Value
}

type DeclarationExpression struct {
	Token       lexer.Token
	Value       string
	Constraints []Expression
}

// Expression implements Expression.
func (*DeclarationExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (d *DeclarationExpression) Literal() string {
	return d.Token.Literal
}

// Position implements Expression.
func (d *DeclarationExpression) Position() lexer.TokenPosition {
	return d.Token.Position
}

// String implements Expression.
func (d *DeclarationExpression) String() string {
	var constraints []string
	for _, c := range d.Constraints {
		constraints = append(constraints, c.String())
	}
	return d.Value + ": " + strings.Join(constraints, "| ")
}

var _ Expression = (*IdentifierExpression)(nil)

type ByteExpression struct {
	Token lexer.Token
	Value byte
}

func (*ByteExpression) Expression()                     {}
func (b *ByteExpression) Literal() string               { return b.Token.Literal }
func (b *ByteExpression) Position() lexer.TokenPosition { return b.Token.Position }
func (b *ByteExpression) String() string                { return b.Literal() + "b" }

var _ Expression = (*ByteExpression)(nil)

type ShortExpression struct {
	Token lexer.Token
	Value int16
}

func (*ShortExpression) Expression()                     {}
func (s *ShortExpression) Literal() string               { return s.Token.Literal }
func (s *ShortExpression) Position() lexer.TokenPosition { return s.Token.Position }
func (s *ShortExpression) String() string                { return s.Literal() + "s" }

var _ Expression = (*ShortExpression)(nil)

type IntegerExpression struct {
	Token lexer.Token
	Value int32
}

func (*IntegerExpression) Expression()                     {}
func (i *IntegerExpression) Literal() string               { return i.Token.Literal }
func (i *IntegerExpression) Position() lexer.TokenPosition { return i.Token.Position }
func (i *IntegerExpression) String() string                { return i.Literal() }

var _ Expression = (*IntegerExpression)(nil)

type LongExpression struct {
	Token lexer.Token
	Value int64
}

func (*LongExpression) Expression()                     {}
func (l *LongExpression) Literal() string               { return l.Token.Literal }
func (l *LongExpression) Position() lexer.TokenPosition { return l.Token.Position }
func (l *LongExpression) String() string                { return l.Literal() + "l" }

var _ Expression = (*LongExpression)(nil)

type FloatExpression struct {
	Token lexer.Token
	Value float32
}

func (*FloatExpression) Expression()                     {}
func (f *FloatExpression) Literal() string               { return f.Token.Literal }
func (f *FloatExpression) Position() lexer.TokenPosition { return f.Token.Position }
func (f *FloatExpression) String() string                { return f.Literal() + "f" }

var _ Expression = (*FloatExpression)(nil)

type DecimalExpression struct {
	Token lexer.Token
	Value float64
}

func (*DecimalExpression) Expression()                     {}
func (f *DecimalExpression) Literal() string               { return f.Token.Literal }
func (f *DecimalExpression) Position() lexer.TokenPosition { return f.Token.Position }
func (f *DecimalExpression) String() string                { return f.Literal() }

var _ Expression = (*DecimalExpression)(nil)

type CharExpression struct {
	Token lexer.Token
	Value rune
}

func (*CharExpression) Expression()                     {}
func (c *CharExpression) Literal() string               { return c.Token.Literal }
func (c *CharExpression) Position() lexer.TokenPosition { return c.Token.Position }
func (c *CharExpression) String() string                { return "'" + string(c.Value) + "'" }

var _ Expression = (*CharExpression)(nil)

type StringExpression struct {
	Token lexer.Token
	Value string
}

// Expression implements Expression.
func (*StringExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (s *StringExpression) Literal() string {
	return s.Token.Literal
}

// Position implements Expression.
func (s *StringExpression) Position() lexer.TokenPosition {
	return s.Token.Position
}

// String implements Expression.
func (s *StringExpression) String() string {
	return fmt.Sprintf("%q", s.Value)
}

var _ Expression = (*StringExpression)(nil)

type InterpolatedExpression struct {
	Token lexer.Token
	Parts []Expression
}

// Expression implements Expression.
func (*InterpolatedExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (i *InterpolatedExpression) Literal() string {
	return i.Token.Literal
}

// Position implements Expression.
func (i *InterpolatedExpression) Position() lexer.TokenPosition {
	return i.Token.Position
}

// String implements Expression.
func (i *InterpolatedExpression) String() string {
	var parts []string
	for _, p := range i.Parts {
		parts = append(parts, p.String())
	}
	return "\"" + strings.Join(parts, "") + "\""
}

var _ Expression = (*InterpolatedExpression)(nil)

type BooleanExpression struct {
	Token lexer.Token
	Value bool
}

// Expression implements Expression.
func (*BooleanExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (b *BooleanExpression) Literal() string {
	return b.Token.Literal
}

// Position implements Expression.
func (b *BooleanExpression) Position() lexer.TokenPosition {
	return b.Token.Position
}

// String implements Expression.
func (b *BooleanExpression) String() string {
	return b.Literal()
}

var _ Expression = (*BooleanExpression)(nil)

type NilExpression struct {
	Token lexer.Token
}

// Expression implements Expression.
func (*NilExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (n *NilExpression) Literal() string {
	return n.Token.Literal
}

// Position implements Expression.
func (n *NilExpression) Position() lexer.TokenPosition {
	return n.Token.Position
}

// String implements Expression.
func (n *NilExpression) String() string {
	return "nil"
}

var _ Expression = (*NilExpression)(nil)

type ArrayExpression struct {
	Token    lexer.Token
	Elements []Expression
}

// Expression implements Expression.
func (*ArrayExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (a *ArrayExpression) Literal() string {
	return a.Token.Literal
}

// Position implements Expression.
func (a *ArrayExpression) Position() lexer.TokenPosition {
	return a.Token.Position
}

// String implements Expression.
func (a *ArrayExpression) String() string {
	var elements []string
	for _, e := range a.Elements {
		elements = append(elements, e.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

var _ Expression = (*ArrayExpression)(nil)

type MapExpression struct {
	Token lexer.Token
	Pairs map[string]Expression
	Order []string
}

// Expression implements Expression.
func (*MapExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (m *MapExpression) Literal() string {
	return m.Token.Literal
}

// Position implements Expression.
func (m *MapExpression) Position() lexer.TokenPosition {
	return m.Token.Position
}

// String implements Expression.
func (m *MapExpression) String() string {
	var pairs []string
	for _, key := range m.Order {
		if val, ok := m.Pairs[key]; ok {
			pairs = append(pairs, key+": "+val.String())
		}
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

var _ Expression = (*MapExpression)(nil)

type PrefixExpression struct {
	Token    lexer.Token
	Operator string
	Right    Expression
}

// Expression implements Expression.
func (*PrefixExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (p *PrefixExpression) Literal() string {
	return p.Token.Literal
}

// Position implements Expression.
func (p *PrefixExpression) Position() lexer.TokenPosition {
	return p.Token.Position
}

// String implements Expression.
func (p *PrefixExpression) String() string {
	return "(" + p.Operator + p.Right.String() + ")"
}

var _ Expression = (*PrefixExpression)(nil)

type InfixExpression struct {
	Token    lexer.Token
	Left     Expression
	Operator string
	Right    Expression
}

// Expression implements Expression.
func (*InfixExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (i *InfixExpression) Literal() string {
	return i.Token.Literal
}

// Position implements Expression.
func (i *InfixExpression) Position() lexer.TokenPosition {
	return i.Token.Position
}

// String implements Expression.
func (i *InfixExpression) String() string {
	return "(" + i.Left.String() + " " + i.Operator + " " + i.Right.String() + ")"
}

var _ Expression = (*InfixExpression)(nil)

type CallExpression struct {
	Token     lexer.Token
	Function  Expression
	Arguments []Expression
}

// Expression implements Expression.
func (*CallExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (c *CallExpression) Literal() string {
	return c.Token.Literal
}

// Position implements Expression.
func (c *CallExpression) Position() lexer.TokenPosition {
	return c.Token.Position
}

// String implements Expression.
func (c *CallExpression) String() string {
	var args []string
	for _, a := range c.Arguments {
		args = append(args, a.String())
	}
	return c.Function.String() + "(" + strings.Join(args, ", ") + ")"
}

var _ Expression = (*CallExpression)(nil)

type IndexExpression struct {
	Token lexer.Token
	Left  Expression
	Index Expression
}

// Expression implements Expression.
func (*IndexExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (i *IndexExpression) Literal() string {
	return i.Token.Literal
}

// Position implements Expression.
func (i *IndexExpression) Position() lexer.TokenPosition {
	return i.Token.Position
}

// String implements Expression.
func (i *IndexExpression) String() string {
	return "(" + i.Left.String() + "[" + i.Index.String() + "])"
}

var _ Expression = (*IndexExpression)(nil)

type AttributeExpression struct {
	Token     lexer.Token
	Object    Expression
	Attribute *IdentifierExpression
}

// Expression implements Expression.
func (*AttributeExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (a *AttributeExpression) Literal() string {
	return a.Token.Literal
}

// Position implements Expression.
func (a *AttributeExpression) Position() lexer.TokenPosition {
	return a.Token.Position
}

// String implements Expression.
func (a *AttributeExpression) String() string {
	return "(" + a.Object.String() + "." + a.Attribute.String() + ")"
}

var _ Expression = (*AttributeExpression)(nil)

type MethodCallExpression struct {
	Token     lexer.Token
	Object    Expression
	Method    *IdentifierExpression
	Arguments []Expression
}

// Expression implements Expression.
func (*MethodCallExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (m *MethodCallExpression) Literal() string {
	return m.Token.Literal
}

// Position implements Expression.
func (m *MethodCallExpression) Position() lexer.TokenPosition {
	return m.Token.Position
}

// String implements Expression.
func (m *MethodCallExpression) String() string {
	var args []string
	for _, a := range m.Arguments {
		args = append(args, a.String())
	}
	return m.Object.String() + "." + m.Method.String() + "(" + strings.Join(args, ", ") + ")"
}

var _ Expression = (*MethodCallExpression)(nil)

type GroupedExpression struct {
	Token lexer.Token
	Expr  Expression
}

// Expression implements Expression.
func (*GroupedExpression) Expression() {
	// Marker method.
}

// Literal implements Expression.
func (g *GroupedExpression) Literal() string {
	return g.Token.Literal
}

// Position implements Expression.
func (g *GroupedExpression) Position() lexer.TokenPosition {
	return g.Token.Position
}

// String implements Expression.
func (g *GroupedExpression) String() string {
	return "(" + g.Expr.String() + ")"
}

var _ Expression = (*GroupedExpression)(nil)

type PointerExpression struct {
	Token    lexer.Token
	TypeName string
	Offset   Expression
}

func (*PointerExpression) Expression() {}

func (p *PointerExpression) Literal() string {
	return p.Token.Literal
}

func (p *PointerExpression) Position() lexer.TokenPosition {
	return p.Token.Position
}

func (p *PointerExpression) String() string {
	return "*" + p.TypeName + "(" + p.Offset.String() + ")"
}

var _ Expression = (*PointerExpression)(nil)

// StructExpression represents a struct initialization: name { field = expr, ... }
type StructExpression struct {
	Token  lexer.Token
	Name   string
	Fields map[string]Expression
	Order  []string // field insertion order
}

func (*StructExpression) Expression() {}

func (s *StructExpression) Literal() string {
	return s.Token.Literal
}

func (s *StructExpression) Position() lexer.TokenPosition {
	return s.Token.Position
}

func (s *StructExpression) String() string {
	var out strings.Builder
	out.WriteString(s.Name)
	out.WriteString(" { ")
	for i, key := range s.Order {
		if i > 0 {
			out.WriteString(", ")
		}
		out.WriteString(key)
		out.WriteString(" = ")
		if val, ok := s.Fields[key]; ok {
			out.WriteString(val.String())
		}
	}
	out.WriteString(" }")
	return out.String()
}

var _ Expression = (*StructExpression)(nil)

// TupleExpression represents a tuple literal: (expr, expr, ...)
type TupleExpression struct {
	Token    lexer.Token
	Elements []Expression
}

func (*TupleExpression) Expression() {}

func (t *TupleExpression) Literal() string {
	return t.Token.Literal
}

func (t *TupleExpression) Position() lexer.TokenPosition {
	return t.Token.Position
}

func (t *TupleExpression) String() string {
	var parts []string
	for _, e := range t.Elements {
		parts = append(parts, e.String())
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

var _ Expression = (*TupleExpression)(nil)
