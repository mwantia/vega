package value

import (
	"fmt"
	"strings"
)

// Map represents a map value.
type Map struct {
	Pairs map[string]Value
	Order []string // Maintains insertion order
}

var _ Value = (*Map)(nil)
var _ Indexable = (*Map)(nil)
var _ Iterable = (*Map)(nil)
var _ Methodable = (*Map)(nil)

func NewMap() *Map {
	return &Map{
		Pairs: make(map[string]Value),
		Order: make([]string, 0),
	}
}

func NewMapWithPairs(pairs map[string]Value, order []string) *Map {
	return &Map{Pairs: pairs, Order: order}
}

func (m *Map) Type() string {
	return TypeMap
}

func (m *Map) String() string {
	parts := make([]string, 0, len(m.Pairs))
	for _, k := range m.Order {
		if v, ok := m.Pairs[k]; ok {
			parts = append(parts, fmt.Sprintf("%s: %s", k, v.String()))
		}
	}
	return "{" + strings.Join(parts, ", ") + "}"
}

func (m *Map) Boolean() bool {
	return len(m.Pairs) > 0
}

func (m *Map) Equal(other Value) bool {
	o, ok := other.(*Map)
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

func (v *Map) Method(name string, args []Value) (Value, error) {
	switch name {
	case "length":
		i := v.Length()
		return NewInteger(i), nil
	case "keys":
		keys := v.Keys()
		elements := make([]Value, len(keys))
		for i, k := range keys {
			elements[i] = NewString(k)
		}
		return NewArray(elements), nil
	}
	return nil, fmt.Errorf("not yet implemented")
}

func (m *Map) Index(key Value) (Value, error) {
	k, ok := key.(*String)
	if !ok {
		return nil, fmt.Errorf("map key must be string, got %s", key.Type())
	}
	if v, exists := m.Pairs[k.Value]; exists {
		return v, nil
	}
	return Nil, nil
}

func (m *Map) SetIndex(key Value, val Value) error {
	k, ok := key.(*String)
	if !ok {
		return fmt.Errorf("map key must be string, got %s", key.Type())
	}
	if _, exists := m.Pairs[k.Value]; !exists {
		m.Order = append(m.Order, k.Value)
	}
	m.Pairs[k.Value] = val
	return nil
}

func (m *Map) Iterator() Iterator {
	return &mapIterator{m: m, pos: -1}
}

func (m *Map) Get(key string) (Value, bool) {
	v, ok := m.Pairs[key]
	return v, ok
}

func (m *Map) Set(key string, val Value) {
	if _, exists := m.Pairs[key]; !exists {
		m.Order = append(m.Order, key)
	}
	m.Pairs[key] = val
}

func (m *Map) Length() int {
	return len(m.Pairs)
}

func (m *Map) Keys() []string {
	return m.Order
}

type mapIterator struct {
	m   *Map
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
