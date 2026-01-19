package value

import (
	"fmt"
	"strings"
)

// ArrayValue represents an array value.
type ArrayValue struct {
	Elements []Value
}

var _ Value = (*ArrayValue)(nil)
var _ Indexable = (*ArrayValue)(nil)
var _ Iterable = (*ArrayValue)(nil)

func NewArray(elements []Value) *ArrayValue {
	return &ArrayValue{Elements: elements}
}

func (a *ArrayValue) Type() string { return TypeArray }

func (a *ArrayValue) String() string {
	parts := make([]string, len(a.Elements))
	for i, e := range a.Elements {
		parts[i] = e.String()
	}
	return "[" + strings.Join(parts, ", ") + "]"
}

func (a *ArrayValue) Boolean() bool { return len(a.Elements) > 0 }

func (a *ArrayValue) Equal(other Value) bool {
	o, ok := other.(*ArrayValue)
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

func (a *ArrayValue) Index(key Value) (Value, error) {
	idx, ok := key.(*IntValue)
	if !ok {
		return nil, fmt.Errorf("array index must be int, got %s", key.Type())
	}
	i := int(idx.Val)
	if i < 0 || i >= len(a.Elements) {
		return nil, fmt.Errorf("array index out of bounds: %d (length %d)", i, len(a.Elements))
	}
	return a.Elements[i], nil
}

func (a *ArrayValue) SetIndex(key Value, val Value) error {
	idx, ok := key.(*IntValue)
	if !ok {
		return fmt.Errorf("array index must be int, got %s", key.Type())
	}
	i := int(idx.Val)
	if i < 0 || i >= len(a.Elements) {
		return fmt.Errorf("array index out of bounds: %d (length %d)", i, len(a.Elements))
	}
	a.Elements[i] = val
	return nil
}

func (a *ArrayValue) Iterator() Iterator {
	return &arrayIterator{arr: a, pos: -1}
}

func (a *ArrayValue) Len() int {
	return len(a.Elements)
}

func (a *ArrayValue) Push(v Value) {
	a.Elements = append(a.Elements, v)
}

func (a *ArrayValue) Pop() (Value, error) {
	if len(a.Elements) == 0 {
		return nil, fmt.Errorf("cannot pop from empty array")
	}
	v := a.Elements[len(a.Elements)-1]
	a.Elements = a.Elements[:len(a.Elements)-1]
	return v, nil
}

type arrayIterator struct {
	arr *ArrayValue
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

// MapValue represents a map value.
type MapValue struct {
	Pairs map[string]Value
	Order []string // Maintains insertion order
}

var _ Value = (*MapValue)(nil)
var _ Indexable = (*MapValue)(nil)
var _ Iterable = (*MapValue)(nil)

func NewMap() *MapValue {
	return &MapValue{
		Pairs: make(map[string]Value),
		Order: make([]string, 0),
	}
}

func NewMapWithPairs(pairs map[string]Value, order []string) *MapValue {
	return &MapValue{Pairs: pairs, Order: order}
}

func (m *MapValue) Type() string { return TypeMap }

func (m *MapValue) String() string {
	parts := make([]string, 0, len(m.Pairs))
	for _, k := range m.Order {
		if v, ok := m.Pairs[k]; ok {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v.String()))
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (m *MapValue) Boolean() bool { return len(m.Pairs) > 0 }

func (m *MapValue) Equal(other Value) bool {
	o, ok := other.(*MapValue)
	if !ok || len(m.Pairs) != len(o.Pairs) {
		return false
	}
	for k, v := range m.Pairs {
		ov, exists := o.Pairs[k]
		if !exists || !v.Equal(ov) {
			return false
		}
	}
	return true
}

func (m *MapValue) Index(key Value) (Value, error) {
	k, ok := key.(*StringValue)
	if !ok {
		return nil, fmt.Errorf("map key must be string, got %s", key.Type())
	}
	if v, exists := m.Pairs[k.Val]; exists {
		return v, nil
	}
	return Nil, nil
}

func (m *MapValue) SetIndex(key Value, val Value) error {
	k, ok := key.(*StringValue)
	if !ok {
		return fmt.Errorf("map key must be string, got %s", key.Type())
	}
	if _, exists := m.Pairs[k.Val]; !exists {
		m.Order = append(m.Order, k.Val)
	}
	m.Pairs[k.Val] = val
	return nil
}

func (m *MapValue) Iterator() Iterator {
	return &mapIterator{m: m, pos: -1}
}

func (m *MapValue) Get(key string) (Value, bool) {
	v, ok := m.Pairs[key]
	return v, ok
}

func (m *MapValue) Set(key string, val Value) {
	if _, exists := m.Pairs[key]; !exists {
		m.Order = append(m.Order, key)
	}
	m.Pairs[key] = val
}

func (m *MapValue) Len() int {
	return len(m.Pairs)
}

func (m *MapValue) Keys() []string {
	return m.Order
}

type mapIterator struct {
	m   *MapValue
	pos int
}

func (it *mapIterator) Next() bool {
	it.pos++
	return it.pos < len(it.m.Order)
}

func (it *mapIterator) Value() Value {
	if it.pos < 0 || it.pos >= len(it.m.Order) {
		return Nil
	}
	return NewString(it.m.Order[it.pos])
}
