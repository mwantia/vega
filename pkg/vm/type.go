package vm

import (
	"fmt"
	"strconv"

	"github.com/mwantia/vega/pkg/value"
)

func newBuiltinTypeFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("type() expects 1 argument, got %d", len(args))
	}
	return value.NewString(args[0].Type()), nil
}

func newBuiltinStringFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("string() expects 1 argument, got %d", len(args))
	}
	return value.NewString(args[0].String()), nil
}

func newBuiltinIntFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("int() expects 1 argument, got %d", len(args))
	}
	switch v := args[0].(type) {
	case *value.IntValue:
		return v, nil
	case *value.FloatValue:
		return value.NewInt(int64(v.Val)), nil
	case *value.StringValue:
		i, err := strconv.ParseInt(v.Val, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int", v.Val)
		}
		return value.NewInt(i), nil
	case *value.BoolValue:
		if v.Val {
			return value.NewInt(1), nil
		}
		return value.NewInt(0), nil
	default:
		return nil, fmt.Errorf("cannot convert %s to int", v.Type())
	}
}

func newBuiltinFloatFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("float() expects 1 argument, got %d", len(args))
	}
	switch v := args[0].(type) {
	case *value.FloatValue:
		return v, nil
	case *value.IntValue:
		return value.NewFloat(float64(v.Val)), nil
	case *value.StringValue:
		f, err := strconv.ParseFloat(v.Val, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float", v.Val)
		}
		return value.NewFloat(f), nil
	default:
		return nil, fmt.Errorf("cannot convert %s to float", v.Type())
	}
}

func newBuiltinBoolFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("bool() expects 1 argument, got %d", len(args))
	}
	return value.NewBool(args[0].Boolean()), nil
}
