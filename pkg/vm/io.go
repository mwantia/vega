package vm

import (
	"bufio"
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/value"
)

func newBuiltinPrintFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.String()
	}
	fmt.Fprint(vm.stdout, strings.Join(parts, " "))
	return value.Nil, nil
}

func newBuiltinPrintlnFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.String()
	}
	fmt.Fprintln(vm.stdout, strings.Join(parts, " "))
	return value.Nil, nil
}

func newBuiltinInputFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	// Print prompt if provided
	if len(args) > 0 {
		fmt.Fprint(vm.stdout, args[0].String())
	}

	reader := bufio.NewReader(vm.stdin)
	line, err := reader.ReadString('\n')
	if err != nil {
		return value.Nil, nil
	}
	return value.NewString(strings.TrimSuffix(line, "\n")), nil
}
