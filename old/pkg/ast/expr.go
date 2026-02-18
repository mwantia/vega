package ast

import (
	"fmt"
	"strings"

	"github.com/mwantia/vega/old/pkg/token"
)

// Identifier represents an identifier like "x" or "foo".
type Identifier struct {
	Token token.Token // the token.IDENT token
	Value string
}

func (i *Identifier) expressionNode()      {}
func (i *Identifier) TokenLiteral() string { return i.Token.Literal }
func (i *Identifier) Pos() token.Position  { return i.Token.Pos }
func (i *Identifier) String() string       { return i.Value }

// IntegerLiteral represents an integer like 42.
type IntegerLiteral struct {
	Token token.Token
	Value int
}

func (il *IntegerLiteral) expressionNode()      {}
func (il *IntegerLiteral) TokenLiteral() string { return il.Token.Literal }
func (il *IntegerLiteral) Pos() token.Position  { return il.Token.Pos }
func (il *IntegerLiteral) String() string       { return il.Token.Literal }

// FloatLiteral represents a floating-point number like 3.14.
type FloatLiteral struct {
	Token token.Token
	Value float64
}

func (fl *FloatLiteral) expressionNode()      {}
func (fl *FloatLiteral) TokenLiteral() string { return fl.Token.Literal }
func (fl *FloatLiteral) Pos() token.Position  { return fl.Token.Pos }
func (fl *FloatLiteral) String() string       { return fl.Token.Literal }

// StringLiteral represents a plain string literal like "hello".
type StringLiteral struct {
	Token token.Token
	Value string
}

func (sl *StringLiteral) expressionNode()      {}
func (sl *StringLiteral) TokenLiteral() string { return sl.Token.Literal }
func (sl *StringLiteral) Pos() token.Position  { return sl.Token.Pos }
func (sl *StringLiteral) String() string       { return fmt.Sprintf("%q", sl.Value) }

// InterpolatedString represents a string with embedded expressions like "Hello ${name}".
type InterpolatedString struct {
	Token token.Token  // the opening quote token
	Parts []Expression // alternating StringLiteral and other expressions
}

func (is *InterpolatedString) expressionNode()      {}
func (is *InterpolatedString) TokenLiteral() string { return is.Token.Literal }
func (is *InterpolatedString) Pos() token.Position  { return is.Token.Pos }
func (is *InterpolatedString) String() string {
	var parts []string
	for _, p := range is.Parts {
		parts = append(parts, p.String())
	}
	return "\"" + strings.Join(parts, "") + "\""
}

// BooleanLiteral represents true or false.
type BooleanLiteral struct {
	Token token.Token
	Value bool
}

func (bl *BooleanLiteral) expressionNode()      {}
func (bl *BooleanLiteral) TokenLiteral() string { return bl.Token.Literal }
func (bl *BooleanLiteral) Pos() token.Position  { return bl.Token.Pos }
func (bl *BooleanLiteral) String() string       { return bl.Token.Literal }

// NilLiteral represents the nil value.
type NilLiteral struct {
	Token token.Token
}

func (nl *NilLiteral) expressionNode()      {}
func (nl *NilLiteral) TokenLiteral() string { return nl.Token.Literal }
func (nl *NilLiteral) Pos() token.Position  { return nl.Token.Pos }
func (nl *NilLiteral) String() string       { return "nil" }

// ArrayLiteral represents an array like [1, 2, 3].
type ArrayLiteral struct {
	Token    token.Token // the '[' token
	Elements []Expression
}

func (al *ArrayLiteral) expressionNode()      {}
func (al *ArrayLiteral) TokenLiteral() string { return al.Token.Literal }
func (al *ArrayLiteral) Pos() token.Position  { return al.Token.Pos }
func (al *ArrayLiteral) String() string {
	var elements []string
	for _, e := range al.Elements {
		elements = append(elements, e.String())
	}
	return "[" + strings.Join(elements, ", ") + "]"
}

// MapLiteral represents a map like {name: "alice", age: 30}.
type MapLiteral struct {
	Token token.Token // the '{' token
	Pairs map[string]Expression
	// Order tracks insertion order for deterministic iteration
	Order []string
}

func (ml *MapLiteral) expressionNode()      {}
func (ml *MapLiteral) TokenLiteral() string { return ml.Token.Literal }
func (ml *MapLiteral) Pos() token.Position  { return ml.Token.Pos }
func (ml *MapLiteral) String() string {
	var pairs []string
	for _, key := range ml.Order {
		if val, ok := ml.Pairs[key]; ok {
			pairs = append(pairs, key+": "+val.String())
		}
	}
	return "{" + strings.Join(pairs, ", ") + "}"
}

// PrefixExpression represents a prefix operation like -x or !x.
type PrefixExpression struct {
	Token    token.Token // the prefix operator token, e.g. ! or -
	Operator string
	Right    Expression
}

func (pe *PrefixExpression) expressionNode()      {}
func (pe *PrefixExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PrefixExpression) Pos() token.Position  { return pe.Token.Pos }
func (pe *PrefixExpression) String() string {
	return "(" + pe.Operator + pe.Right.String() + ")"
}

// InfixExpression represents a binary operation like a + b.
type InfixExpression struct {
	Token    token.Token // the operator token, e.g. +
	Left     Expression
	Operator string
	Right    Expression
}

func (ie *InfixExpression) expressionNode()      {}
func (ie *InfixExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *InfixExpression) Pos() token.Position  { return ie.Token.Pos }
func (ie *InfixExpression) String() string {
	return "(" + ie.Left.String() + " " + ie.Operator + " " + ie.Right.String() + ")"
}

// CallExpression represents a function call like print("hello").
type CallExpression struct {
	Token     token.Token // the '(' token
	Function  Expression  // Identifier or function literal
	Arguments []Expression
}

func (ce *CallExpression) expressionNode()      {}
func (ce *CallExpression) TokenLiteral() string { return ce.Token.Literal }
func (ce *CallExpression) Pos() token.Position  { return ce.Token.Pos }
func (ce *CallExpression) String() string {
	var args []string
	for _, a := range ce.Arguments {
		args = append(args, a.String())
	}
	return ce.Function.String() + "(" + strings.Join(args, ", ") + ")"
}

// IndexExpression represents array/map access like arr[0] or map["key"].
type IndexExpression struct {
	Token token.Token // the '[' token
	Left  Expression  // the object being indexed
	Index Expression  // the index expression
}

func (ie *IndexExpression) expressionNode()      {}
func (ie *IndexExpression) TokenLiteral() string { return ie.Token.Literal }
func (ie *IndexExpression) Pos() token.Position  { return ie.Token.Pos }
func (ie *IndexExpression) String() string {
	return "(" + ie.Left.String() + "[" + ie.Index.String() + "])"
}

// AttributeExpression represents attribute access like obj.name.
type AttributeExpression struct {
	Token     token.Token // the '.' token
	Object    Expression
	Attribute *Identifier
}

func (ae *AttributeExpression) expressionNode()      {}
func (ae *AttributeExpression) TokenLiteral() string { return ae.Token.Literal }
func (ae *AttributeExpression) Pos() token.Position  { return ae.Token.Pos }
func (ae *AttributeExpression) String() string {
	return "(" + ae.Object.String() + "." + ae.Attribute.String() + ")"
}

// MethodCallExpression represents a method call like vfs.stat(path).
type MethodCallExpression struct {
	Token     token.Token // the '(' token after method name
	Object    Expression
	Method    *Identifier
	Arguments []Expression
}

func (mc *MethodCallExpression) expressionNode()      {}
func (mc *MethodCallExpression) TokenLiteral() string { return mc.Token.Literal }
func (mc *MethodCallExpression) Pos() token.Position  { return mc.Token.Pos }
func (mc *MethodCallExpression) String() string {
	var args []string
	for _, a := range mc.Arguments {
		args = append(args, a.String())
	}
	return mc.Object.String() + "." + mc.Method.String() + "(" + strings.Join(args, ", ") + ")"
}

// GroupedExpression represents a parenthesized expression like (a + b).
type GroupedExpression struct {
	Token      token.Token // the '(' token
	Expression Expression
}

func (ge *GroupedExpression) expressionNode()      {}
func (ge *GroupedExpression) TokenLiteral() string { return ge.Token.Literal }
func (ge *GroupedExpression) Pos() token.Position  { return ge.Token.Pos }
func (ge *GroupedExpression) String() string       { return "(" + ge.Expression.String() + ")" }

// PipeExpression represents a pipe operation like a | b.
type PipeExpression struct {
	Token token.Token
	Left  Expression
	Right Expression
}

func (pe *PipeExpression) expressionNode()      {}
func (pe *PipeExpression) TokenLiteral() string { return pe.Token.Literal }
func (pe *PipeExpression) Pos() token.Position  { return pe.Token.Pos }
func (pe *PipeExpression) String() string {
	return "(" + pe.Left.String() + " | " + pe.Right.String() + ")"
}
