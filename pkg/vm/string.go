package vm

import (
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/value"
)

func newBuiltinUpperFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("upper() expects 1 argument, got %d", len(args))
	}
	s, ok := args[0].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("upper() argument must be string, got %s", args[0].Type())
	}
	return value.NewString(strings.ToUpper(s.Val)), nil
}

func newBuiltinLowerFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("lower() expects 1 argument, got %d", len(args))
	}
	s, ok := args[0].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("lower() argument must be string, got %s", args[0].Type())
	}
	return value.NewString(strings.ToLower(s.Val)), nil
}

func newBuiltinTrimFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("trim() expects 1 argument, got %d", len(args))
	}
	s, ok := args[0].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("trim() argument must be string, got %s", args[0].Type())
	}
	return value.NewString(strings.TrimSpace(s.Val)), nil
}

func newBuiltinSplitFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("split() expects 2 arguments, got %d", len(args))
	}
	s, ok := args[0].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("split() first argument must be string, got %s", args[0].Type())
	}
	sep, ok := args[1].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("split() second argument must be string, got %s", args[1].Type())
	}
	parts := strings.Split(s.Val, sep.Val)
	elements := make([]value.Value, len(parts))
	for i, p := range parts {
		elements[i] = value.NewString(p)
	}
	return value.NewArray(elements), nil
}

func newBuiltinJoinFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("join() expects 2 arguments, got %d", len(args))
	}
	arr, ok := args[0].(*value.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("join() first argument must be array, got %s", args[0].Type())
	}
	sep, ok := args[1].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("join() second argument must be string, got %s", args[1].Type())
	}
	parts := make([]string, len(arr.Elements))
	for i, e := range arr.Elements {
		parts[i] = e.String()
	}
	return value.NewString(strings.Join(parts, sep.Val)), nil
}

func newBuiltinContainsFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("contains() expects 2 arguments, got %d", len(args))
	}
	switch v := args[0].(type) {
	case *value.StringValue:
		substr, ok := args[1].(*value.StringValue)
		if !ok {
			return nil, fmt.Errorf("contains() second argument must be string for string search")
		}
		return value.NewBool(strings.Contains(v.Val, substr.Val)), nil
	case *value.ArrayValue:
		for _, e := range v.Elements {
			if e.Equal(args[1]) {
				return value.True, nil
			}
		}
		return value.False, nil
	default:
		return nil, fmt.Errorf("contains() first argument must be string or array, got %s", v.Type())
	}
}

func newBuiltinStartsWithFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("startswith() expects 2 arguments, got %d", len(args))
	}
	s, ok := args[0].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("startswith() first argument must be string, got %s", args[0].Type())
	}
	prefix, ok := args[1].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("startswith() second argument must be string, got %s", args[1].Type())
	}
	return value.NewBool(strings.HasPrefix(s.Val, prefix.Val)), nil
}

func newBuiltinEndsWithFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("endswith() expects 2 arguments, got %d", len(args))
	}
	s, ok := args[0].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("endswith() first argument must be string, got %s", args[0].Type())
	}
	suffix, ok := args[1].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("endswith() second argument must be string, got %s", args[1].Type())
	}
	return value.NewBool(strings.HasSuffix(s.Val, suffix.Val)), nil
}

func newBuiltinReplaceFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 3 {
		return nil, fmt.Errorf("replace() expects 3 arguments, got %d", len(args))
	}
	s, ok := args[0].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("replace() first argument must be string, got %s", args[0].Type())
	}
	old, ok := args[1].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("replace() second argument must be string, got %s", args[1].Type())
	}
	new, ok := args[2].(*value.StringValue)
	if !ok {
		return nil, fmt.Errorf("replace() third argument must be string, got %s", args[2].Type())
	}
	return value.NewString(strings.ReplaceAll(s.Val, old.Val, new.Val)), nil
}

func newBuiltinIndexFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("index() expects 2 arguments, got %d", len(args))
	}
	switch v := args[0].(type) {
	case *value.StringValue:
		substr, ok := args[1].(*value.StringValue)
		if !ok {
			return nil, fmt.Errorf("index() second argument must be string for string search")
		}
		return value.NewInt(int64(strings.Index(v.Val, substr.Val))), nil
	case *value.ArrayValue:
		for i, e := range v.Elements {
			if e.Equal(args[1]) {
				return value.NewInt(int64(i)), nil
			}
		}
		return value.NewInt(-1), nil
	default:
		return nil, fmt.Errorf("index() first argument must be string or array, got %s", v.Type())
	}
}
