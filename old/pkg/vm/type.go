package vm

import (
	"fmt"
	"strconv"

	"github.com/mwantia/vega/old/pkg/value"
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

func newBuiltinIntegerFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("integer() expects 1 argument, got %d", len(args))
	}
	switch v := args[0].(type) {
	case *value.Integer:
		return v, nil
	case *value.Float:
		return value.NewInteger(int(v.Value)), nil
	case *value.String:
		i, err := strconv.ParseInt(v.Value, 10, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to int", v.Value)
		}
		return value.NewInteger(int(i)), nil
	case *value.Boolean:
		if v.Value {
			return value.NewInteger(1), nil
		}
		return value.NewInteger(0), nil
	default:
		return nil, fmt.Errorf("cannot convert %s to int", v.Type())
	}
}

func newBuiltinFloatFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("float() expects 1 argument, got %d", len(args))
	}
	switch v := args[0].(type) {
	case *value.Float:
		return v, nil
	case *value.Integer:
		return value.NewFloat(float64(v.Value)), nil
	case *value.String:
		f, err := strconv.ParseFloat(v.Value, 64)
		if err != nil {
			return nil, fmt.Errorf("cannot convert %q to float", v.Value)
		}
		return value.NewFloat(f), nil
	default:
		return nil, fmt.Errorf("cannot convert %s to float", v.Type())
	}
}

func newBuiltinBooleanFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) != 1 {
		return nil, fmt.Errorf("boolean() expects 1 argument, got %d", len(args))
	}
	return value.NewBoolean(args[0].Boolean()), nil
}
