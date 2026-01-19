package vm

import (
	"fmt"

	"github.com/mwantia/vega/pkg/value"
)

func newBuiltinLenFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("len() expects 1 argument, got %d", len(args))
	}
	switch v := args[0].(type) {
	case *value.StringValue:
		return value.NewInt(int64(len(v.Val))), nil
	case *value.ArrayValue:
		return value.NewInt(int64(len(v.Elements))), nil
	case *value.MapValue:
		return value.NewInt(int64(len(v.Pairs))), nil
	default:
		return nil, fmt.Errorf("len() not supported for %s", v.Type())
	}
}

func newBuiltinPushFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 2 {
		return nil, fmt.Errorf("push() expects 2 arguments, got %d", len(args))
	}
	arr, ok := args[0].(*value.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("push() first argument must be array, got %s", args[0].Type())
	}
	arr.Push(args[1])
	return value.Nil, nil
}

func newBuiltinPopFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("pop() expects 1 argument, got %d", len(args))
	}
	arr, ok := args[0].(*value.ArrayValue)
	if !ok {
		return nil, fmt.Errorf("pop() argument must be array, got %s", args[0].Type())
	}
	return arr.Pop()
}

func newBuiltinKeysFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("keys() expects 1 argument, got %d", len(args))
	}
	m, ok := args[0].(*value.MapValue)
	if !ok {
		return nil, fmt.Errorf("keys() argument must be map, got %s", args[0].Type())
	}
	keys := m.Keys()
	elements := make([]value.Value, len(keys))
	for i, k := range keys {
		elements[i] = value.NewString(k)
	}
	return value.NewArray(elements), nil
}
