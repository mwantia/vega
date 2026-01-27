package vm

import "github.com/mwantia/vega/pkg/value"

func (vm *VirtualMachine) NewStdinStream() *value.Stream {
	return value.NewInputStream("stdin", vm.stdin)
}

func newBuiltinStdinFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	return vm.NewStdinStream(), nil
}

func (vm *VirtualMachine) NewStdoutStream() *value.Stream {
	return value.NewOutputStream("stdout", vm.stdout)
}

func newBuiltinStdoutFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	return vm.NewStdoutStream(), nil
}

func (vm *VirtualMachine) NewStderrStream() *value.Stream {
	return value.NewOutputStream("stderr", vm.stderr)
}

func newBuiltinStderrFunction(vm *VirtualMachine, args []value.Value) (value.Value, error) {
	return vm.NewStderrStream(), nil
}
