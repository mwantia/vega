package value

import (
	"fmt"
	"strings"
)

// Array represents an array value.
type Array struct {
	Elements []Value
}

var _ Value = (*Array)(nil)
var _ Indexable = (*Array)(nil)
var _ Iterable = (*Array)(nil)
var _ Methodable = (*Array)(nil)

func NewArray(elements []Value) *Array {
	return &Array{Elements: elements}
}

func (a *Array) Type() string {
	return TypeArray
}

func (a *Array) String() string {
	parts := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		parts[i] = e.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (a *Array) Boolean() bool {
	return len(a.Elements) > 0
}

func (a *Array) Equal(other Value) bool {
	o, ok := other.(*Array)
	if !ok || len(a.Elements) != len(o.Elements) {
		return false
	}
	for i, e := range a.Elements {
		if !e.Equal(o.Elements[i]) {
			return false
		}
	}
	return true
}

func (v *Array) Method(name string, args []Value) (Value, error) {
	switch name {
	case "length":
		i := v.Length()
		return NewInteger(i), nil
	case "push":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 argument, got %d", name, len(args))
		}
		v.Push(args[0])
		return Nil, nil
	case "pop":
		return v.Pop()
	case "join":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		sep, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("method '%s' first argument must be 'string', got '%s'", name, args[1].Type())
		}
		parts := make([]string, len(v.Elements))
		for i, e := range v.Elements {
			parts[i] = e.String()
		}
		return NewString(strings.Join(parts, sep.Value)), nil
	case "contains":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		other := args[0]
		for _, e := range v.Elements {
			if e.Equal(other) {
				return True, nil
			}
		}
		return False, nil
	case "index":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		other := args[0]
		for i, e := range v.Elements {
			if e.Equal(other) {
				return NewInteger(i), nil
			}
		}
		return NewInteger(-1), nil
	}
	return nil, fmt.Errorf("unknown method call")
}

func (a *Array) Index(key Value) (Value, error) {
	idx, ok := key.(*Integer)
	if !ok {
		return nil, fmt.Errorf("array index must be int, got %s", key.Type())
	}
	i := int(idx.Value)
	if i < 0 || i >= len(a.Elements) {
		return nil, fmt.Errorf("array index out of bounds: %d (length %d)", i, len(a.Elements))
	}
	return a.Elements[i], nil
}

func (a *Array) SetIndex(key Value, val Value) error {
	idx, ok := key.(*Integer)
	if !ok {
		return fmt.Errorf("array index must be int, got %s", key.Type())
	}
	i := int(idx.Value)
	if i < 0 || i >= len(a.Elements) {
		return fmt.Errorf("array index out of bounds: %d (length %d)", i, len(a.Elements))
	}
	a.Elements[i] = val
	return nil
}

func (a *Array) Iterator() Iterator {
	return &arrayIterator{arr: a, pos: -1}
}

func (a *Array) Length() int {
	return len(a.Elements)
}

func (a *Array) Push(v Value) {
	a.Elements = append(a.Elements, v)
}

func (a *Array) Pop() (Value, error) {
	if len(a.Elements) == 0 {
		return nil, fmt.Errorf("cannot pop from empty array")
	}
	v := a.Elements[len(a.Elements)-1]
	a.Elements = a.Elements[:len(a.Elements)-1]
	return v, nil
}

type arrayIterator struct {
	arr *Array
	pos int
}

func (it *arrayIterator) Next() bool {
	it.pos++
	return it.pos < len(it.arr.Elements)
}

func (it *arrayIterator) Value() Value {
	if it.pos < 0 || it.pos >= len(it.arr.Elements) {
		return Nil
	}
	return it.arr.Elements[it.pos]
}
