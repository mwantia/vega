package value

import (
	"fmt"
	"strconv"
	"strings"
)

type String struct {
	Value string
}

var _ Value = (*String)(nil)
var _ Comparable = (*String)(nil)
var _ Methodable = (*String)(nil)

func NewString(s string) *String {
	return &String{Value: s}
}

func (s *String) Type() string {
	return TypeString
}

func (s *String) String() string {
	return s.Value
}

func (s *String) Boolean() bool {
	return len(s.Value) > 0
}

func (s *String) Equal(other Value) bool {
	if o, ok := other.(*String); ok {
		return s.Value == o.Value
	}
	return false
}

func (v *String) Method(name string, args []Value) (Value, error) {
	switch name {
	case "length":
		i := int64(len(v.Value))
		return NewInteger(i), nil
	case "upper":
		s := strings.ToUpper(v.Value)
		return NewString(s), nil
	case "lower":
		s := strings.ToLower(v.Value)
		return NewString(s), nil
	case "trim":
		s := strings.TrimSpace(v.Value)
		return NewString(s), nil
	case "split":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		sep, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("method '%s' first argument must be 'string', got '%s'", name, args[0].Type())
		}
		parts := strings.Split(v.Value, sep.Value)
		elements := make([]Value, len(parts))
		for i, p := range parts {
			elements[i] = NewString(p)
		}
		return NewArray(elements), nil
	case "contains":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		sub, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("contains() first argument must be string for string search")
		}
		return NewBoolean(strings.Contains(v.Value, sub.Value)), nil
	case "startswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		prefix, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("method '%s' first argument must be 'string', got '%s'", name, args[0].Type())
		}
		return NewBoolean(strings.HasPrefix(v.Value, prefix.Value)), nil
	case "endswith":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		suffix, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("method '%s' first argument must be 'string', got '%s'", name, args[0].Type())
		}
		return NewBoolean(strings.HasSuffix(v.Value, suffix.Value)), nil
	case "replace":
		if len(args) != 2 {
			return nil, fmt.Errorf("method '%s' expects 2 arguments, got %d", name, len(args))
		}
		old, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("method '%s' first argument must be 'string', got '%s'", name, args[0].Type())
		}
		new, ok := args[1].(*String)
		if !ok {
			return nil, fmt.Errorf("method '%s' second argument must be 'string', got '%s'", name, args[1].Type())
		}
		return NewString(strings.ReplaceAll(v.Value, old.Value, new.Value)), nil
	case "index":
		if len(args) != 1 {
			return nil, fmt.Errorf("method '%s' expects 1 arguments, got %d", name, len(args))
		}
		sub, ok := args[0].(*String)
		if !ok {
			return nil, fmt.Errorf("method '%s' first argument must be 'string', got '%s'", name, args[0].Type())
		}
		return NewInteger(int64(strings.Index(v.Value, sub.Value))), nil
	}

	return nil, fmt.Errorf("unknown method-map: '%s'", name)
}

func (s *String) Compare(other Value) (int, bool) {
	if o, ok := other.(*String); ok {
		return strings.Compare(s.Value, o.Value), true
	}
	return 0, false
}

func (s *String) ParseToSize() (int64, error) {
	const (
		B  = 1
		KB = 1024
		MB = KB * 1024
		GB = MB * 1024
	)

	value := strings.TrimSpace(strings.ToLower(s.Value))
	if value == "" {
		return 0, fmt.Errorf("empty size string")
	}

	// Find where the number ends and unit begins
	i := 0
	for i < len(value) && (value[i] >= '0' && value[i] <= '9' || value[i] == '.') {
		i++
	}

	if i == 0 {
		return 0, fmt.Errorf("no numeric value in size string: %s", value)
	}

	numberStr := value[:i]
	unit := strings.TrimSpace(value[i:])

	// Parse the numeric value
	f, err := strconv.ParseFloat(numberStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in size string: %s", value)
	}

	// Determine multiplier based on unit
	var multiplier int64
	switch unit {
	case "", "b":
		multiplier = B
	case "k", "kb":
		multiplier = KB
	case "m", "mb":
		multiplier = MB
	case "g", "gb":
		multiplier = GB
	default:
		return 0, fmt.Errorf("unknown size unit: %s (supported: B, KB, MB, GB)", unit)
	}

	return int64(f * float64(multiplier)), nil
}
