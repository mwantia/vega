// Package value defines the runtime value types for Vega.
package value

import "fmt"

// TypeNamespace is the type name for namespace values.
const TypeNamespace = "namespace"

// NamespaceValue represents a namespace like sys or vfs.
// It provides attribute access for values and method dispatch.
type NamespaceValue struct {
	name    string
	members map[string]Value
	methods map[string]NamespaceMethod
}

// NamespaceMethod is a function that can be called on a namespace.
type NamespaceMethod func(args []Value) (Value, error)

// NewNamespace creates a new namespace with the given name.
func NewNamespace(name string) *NamespaceValue {
	return &NamespaceValue{
		name:    name,
		members: make(map[string]Value),
		methods: make(map[string]NamespaceMethod),
	}
}

// Type returns "namespace".
func (n *NamespaceValue) Type() string {
	return TypeNamespace
}

// String returns a string representation.
func (n *NamespaceValue) String() string {
	return fmt.Sprintf("<namespace:%s>", n.name)
}

// Boolean returns true (namespaces are always truthy).
func (n *NamespaceValue) Boolean() bool {
	return true
}

// Equal compares two namespaces by identity.
func (n *NamespaceValue) Equal(other Value) bool {
	if o, ok := other.(*NamespaceValue); ok {
		return n == o
	}
	return false
}

func (v *NamespaceValue) Method(name string, args []Value) (Value, error) {
	if method, ok := v.GetMethod(name); ok {
		return method(args)
	}
	return nil, fmt.Errorf("namespace '%s' has no method '%s'", v.name, name)
}

// Name returns the namespace name.
func (n *NamespaceValue) Name() string {
	return n.name
}

// Set sets a member value.
func (n *NamespaceValue) Set(name string, val Value) {
	n.members[name] = val
}

// Get gets a member value.
func (n *NamespaceValue) Get(name string) (Value, bool) {
	val, ok := n.members[name]
	return val, ok
}

// SetMethod registers a method on the namespace.
func (n *NamespaceValue) SetMethod(name string, fn NamespaceMethod) {
	n.methods[name] = fn
}

// GetMethod gets a method by name.
func (n *NamespaceValue) GetMethod(name string) (NamespaceMethod, bool) {
	fn, ok := n.methods[name]
	return fn, ok
}

// HasMember returns true if the namespace has a member with the given name.
func (n *NamespaceValue) HasMember(name string) bool {
	_, hasMember := n.members[name]
	_, hasMethod := n.methods[name]
	return hasMember || hasMethod
}

// Members returns all member names.
func (n *NamespaceValue) Members() []string {
	names := make([]string, 0, len(n.members))
	for name := range n.members {
		names = append(names, name)
	}
	return names
}

// Methods returns all method names.
func (n *NamespaceValue) Methods() []string {
	names := make([]string, 0, len(n.methods))
	for name := range n.methods {
		names = append(names, name)
	}
	return names
}
