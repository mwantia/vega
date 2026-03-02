package parser

import (
	"fmt"
	"strings"
)

type IdentifierExpression struct {
	BaseExpression
	Value string
}

func (i *IdentifierExpression) String() string { return i.Value }

var _ Expression = (*IdentifierExpression)(nil)

type DeclarationExpression struct {
	BaseExpression
	Value       string
	Constraints []Expression
}

func (d *DeclarationExpression) String() string {
	var constraints []string
	for _, c := range d.Constraints {
		constraints = append(constraints, c.String())
	}
	return d.Value + ": " + strings.Join(constraints, "| ")
}

var _ Expression = (*DeclarationExpression)(nil)

type ByteExpression struct {
	BaseExpression
	Value byte
}

func (b *ByteExpression) String() string { return b.Literal() + "b" }

var _ Expression = (*ByteExpression)(nil)

type ShortExpression struct {
	BaseExpression
	Value int16
}

func (s *ShortExpression) String() string { return s.Literal() + "s" }

var _ Expression = (*ShortExpression)(nil)

type IntegerExpression struct {
	BaseExpression
	Value int32
}

func (i *IntegerExpression) String() string { return i.Literal() }

var _ Expression = (*IntegerExpression)(nil)

type LongExpression struct {
	BaseExpression
	Value int64
}

func (l *LongExpression) String() string { return l.Literal() + "l" }

var _ Expression = (*LongExpression)(nil)

type FloatExpression struct {
	BaseExpression
	Value float32
}

func (f *FloatExpression) String() string { return f.Literal() + "f" }

var _ Expression = (*FloatExpression)(nil)

type DecimalExpression struct {
	BaseExpression
	Value float64
}

func (f *DecimalExpression) String() string { return f.Literal() }

var _ Expression = (*DecimalExpression)(nil)

type StringExpression struct {
	BaseExpression
	Value string
}

func (s *StringExpression) String() string { return fmt.Sprintf("%q", s.Value) }

var _ Expression = (*StringExpression)(nil)

type InterpolatedExpression struct {
	BaseExpression
	Parts []Expression
}

func (i *InterpolatedExpression) String() string {
	var parts []string
	for _, p := range i.Parts {
		parts = append(parts, p.String())
	}
	return "\"" + strings.Join(parts, "") + "\""
}

var _ Expression = (*InterpolatedExpression)(nil)

type BooleanExpression struct {
	BaseExpression
	Value bool
}

func (b *BooleanExpression) String() string { return b.Literal() }

var _ Expression = (*BooleanExpression)(nil)

type NilExpression struct {
	BaseExpression
}

func (*NilExpression) String() string { return "nil" }

var _ Expression = (*NilExpression)(nil)

type ArrayExpression struct {
	BaseExpression
	Elements []Expression
}

func (a *ArrayExpression) String() string {
	var elements []string
	for _, e := range a.Elements {
		elements = append(elements, e.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

var _ Expression = (*ArrayExpression)(nil)

type MapExpression struct {
	BaseExpression
	Pairs map[string]Expression
	Order []string
}

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
	BaseExpression
	Operator string
	Right    Expression
}

func (p *PrefixExpression) String() string {
	return "(" + p.Operator + p.Right.String() + ")"
}

var _ Expression = (*PrefixExpression)(nil)

type InfixExpression struct {
	BaseExpression
	Left     Expression
	Operator string
	Right    Expression
}

func (i *InfixExpression) String() string {
	return "(" + i.Left.String() + " " + i.Operator + " " + i.Right.String() + ")"
}

var _ Expression = (*InfixExpression)(nil)

type CallExpression struct {
	BaseExpression
	Function  Expression
	Arguments []Expression
}

func (c *CallExpression) String() string {
	var args []string
	for _, a := range c.Arguments {
		args = append(args, a.String())
	}
	return c.Function.String() + "(" + strings.Join(args, ", ") + ")"
}

var _ Expression = (*CallExpression)(nil)

type NamedArgumentExpression struct {
	BaseExpression
	Name  string
	Value Expression
}

func (n *NamedArgumentExpression) String() string { return n.Name + ": " + n.Value.String() }

var _ Expression = (*NamedArgumentExpression)(nil)

type IndexExpression struct {
	BaseExpression
	Left  Expression
	Index Expression
}

func (i *IndexExpression) String() string {
	return "(" + i.Left.String() + "[" + i.Index.String() + "])"
}

var _ Expression = (*IndexExpression)(nil)

type AttributeExpression struct {
	BaseExpression
	Object    Expression
	Attribute *IdentifierExpression
}

func (a *AttributeExpression) String() string {
	return "(" + a.Object.String() + "." + a.Attribute.String() + ")"
}

var _ Expression = (*AttributeExpression)(nil)

type MethodCallExpression struct {
	BaseExpression
	Object    Expression
	Method    *IdentifierExpression
	Arguments []Expression
}

func (m *MethodCallExpression) String() string {
	var args []string
	for _, a := range m.Arguments {
		args = append(args, a.String())
	}
	return m.Object.String() + "." + m.Method.String() + "(" + strings.Join(args, ", ") + ")"
}

var _ Expression = (*MethodCallExpression)(nil)

type GroupedExpression struct {
	BaseExpression
	Expr Expression
}

func (g *GroupedExpression) String() string { return "(" + g.Expr.String() + ")" }

var _ Expression = (*GroupedExpression)(nil)

type PointerExpression struct {
	BaseExpression
	TypeName string
	Offset   Expression
}

func (p *PointerExpression) String() string {
	return "*" + p.TypeName + "(" + p.Offset.String() + ")"
}

var _ Expression = (*PointerExpression)(nil)

type StructExpression struct {
	BaseExpression
	Name   string
	Fields map[string]Expression
	Order  []string // field insertion order
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

type SliceTypeExpression struct {
	BaseExpression
	TypeName string // "string", "byte", etc.
	Capacity int    // declared capacity in bytes (always > 0)
}

func (s *SliceTypeExpression) String() string {
	return fmt.Sprintf("%s<%d>", s.TypeName, s.Capacity)
}

var _ Expression = (*SliceTypeExpression)(nil)

type TupleExpression struct {
	BaseExpression
	Elements []Expression
}

func (t *TupleExpression) String() string {
	var parts []string
	for _, e := range t.Elements {
		parts = append(parts, e.String())
	}
	return "(" + strings.Join(parts, ", ") + ")"
}

var _ Expression = (*TupleExpression)(nil)
