package vm

import (
	"fmt"

	"github.com/mwantia/vega/pkg/value"
)

func newBuiltinRangeFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	switch len(args) {
	case 1:
		// range(n) -> [0, 1, ..., n-1]
		n, ok := args[0].(*value.IntValue)
		if !ok {
			return nil, fmt.Errorf("range() argument must be int, got %s", args[0].Type())
		}
		elements := make([]value.Value, n.Val)
		for i := int64(0); i < n.Val; i++ {
			elements[i] = value.NewInt(i)
		}
		return value.NewArray(elements), nil
	case 2:
		// range(start, end) -> [start, start+1, ..., end-1]
		start, ok := args[0].(*value.IntValue)
		if !ok {
			return nil, fmt.Errorf("range() first argument must be int, got %s", args[0].Type())
		}
		end, ok := args[1].(*value.IntValue)
		if !ok {
			return nil, fmt.Errorf("range() second argument must be int, got %s", args[1].Type())
		}
		if start.Val > end.Val {
			return value.NewArray([]value.Value{}), nil
		}
		elements := make([]value.Value, end.Val-start.Val)
		for i := start.Val; i < end.Val; i++ {
			elements[i-start.Val] = value.NewInt(i)
		}
		return value.NewArray(elements), nil
	default:
		return nil, fmt.Errorf("range() expects 1 or 2 arguments, got %d", len(args))
	}
}

func newBuiltinAssertFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	if len(args) < 1 || len(args) > 2 {
		return nil, fmt.Errorf("assert() expects 1 or 2 arguments, got %d", len(args))
	}
	if !args[0].Boolean() {
		msg := "assertion failed"
		if len(args) == 2 {
			msg = args[1].String()
		}
		return nil, fmt.Errorf("%s", msg)
	}
	return value.Nil, nil
}
