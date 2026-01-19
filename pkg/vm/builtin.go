package vm

import (
	"github.com/mwantia/vega/pkg/value"
)

// BuiltinFunc is the signature for built-in functions.
type BuiltinFunc func(vm *VirtualMachine, args []value.Value) (value.Value, error)

func (vm *VirtualMachine) registerBuiltinFunctions() {
	// Stream Functions
	vm.builtins["stdin"] = newBuiltinStdinFunction
	vm.builtins["stdout"] = newBuiltinStdoutFunction
	vm.builtins["stderr"] = newBuiltinStderrFunction
	// I/O Functions
	vm.builtins["print"] = newBuiltinPrintFunction
	vm.builtins["println"] = newBuiltinPrintlnFunction
	vm.builtins["input"] = newBuiltinInputFunction
	// Type-Declaration Functions
	vm.builtins["type"] = newBuiltinTypeFunction
	vm.builtins["string"] = newBuiltinStringFunction
	vm.builtins["int"] = newBuiltinIntFunction
	vm.builtins["float"] = newBuiltinFloatFunction
	vm.builtins["bool"] = newBuiltinBoolFunction
	// Collection Functions
	vm.builtins["len"] = newBuiltinLenFunction
	vm.builtins["push"] = newBuiltinPushFunction
	vm.builtins["pop"] = newBuiltinPopFunction
	vm.builtins["keys"] = newBuiltinKeysFunction
	// String Functions
	vm.builtins["upper"] = newBuiltinUpperFunction
	vm.builtins["lower"] = newBuiltinLowerFunction
	vm.builtins["trim"] = newBuiltinTrimFunction
	vm.builtins["split"] = newBuiltinSplitFunction
	vm.builtins["join"] = newBuiltinJoinFunction
	vm.builtins["contains"] = newBuiltinContainsFunction
	vm.builtins["startswith"] = newBuiltinStartsWithFunction
	vm.builtins["endswith"] = newBuiltinEndsWithFunction
	vm.builtins["replace"] = newBuiltinReplaceFunction
	vm.builtins["index"] = newBuiltinIndexFunction
	// Utility Functions
	vm.builtins["range"] = newBuiltinRangeFunction
	vm.builtins["assert"] = newBuiltinAssertFunction
	// VFS Functions
	vm.builtins["read"] = newBuiltinReadFunction
	vm.builtins["write"] = newBuiltinWriteFunction
	vm.builtins["stat"] = newBuiltinStatFunction
	vm.builtins["lookup"] = newBuiltinLookupFunction
	vm.builtins["readdir"] = newBuiltinReaddirFunction
	vm.builtins["createdir"] = newBuiltinCreatedirFunction
	vm.builtins["remdir"] = newBuiltinRemdirFunction
	vm.builtins["unlink"] = newBuiltinUnlinkFunction
	vm.builtins["rename"] = newBuiltinRenameFunction
	// VFS Stream/Exec Functions
	vm.builtins["open"] = newBuiltinOpenFunction
	vm.builtins["exec"] = newBuiltinExecFunction
	vm.builtins["sexec"] = newBuiltinSexecFunction
	vm.builtins["capture"] = newBuiltinCaptureFunction
}
