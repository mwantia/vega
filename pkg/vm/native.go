package vm

import (
	"fmt"
	"io"
	"strings"

	"github.com/mwantia/vega/pkg/value"
	"github.com/mwantia/vfs"
)

// Native holds the host-side resources available to native functions.
type Native struct {
	Stdin  io.Reader
	Stdout io.Writer
	Stderr io.Writer
	FS     vfs.VirtualFileSystem
}

// NativeFunc is the signature for all native (Go-implemented) functions
// callable from Vega via OpCallNAT. Native functions are void — they do not
// push a return value onto the expression stack.
type NativeFunc func(nat *Native, args []value.Value) error

var nativeRegistry = map[string]NativeFunc{}

// RegisterNative registers a native function under the given name.
// Call this before running any bytecode that references the function.
func RegisterNative(name string, fn NativeFunc) {
	nativeRegistry[name] = fn
}

// lookupNative returns the native function registered under name, if any.
func lookupNative(name string) (NativeFunc, bool) {
	fn, ok := nativeRegistry[name]
	return fn, ok
}

func init() {
	RegisterNative("print", nativePrint)
	RegisterNative("type", nativeType)
}

func nativePrint(nat *Native, args []value.Value) error {
	parts := make([]string, len(args))
	for i, arg := range args {
		parts[i] = arg.String()
	}
	_, err := fmt.Fprintln(nat.Stdout, strings.Join(parts, " "))
	return err
}

func nativeType(nat *Native, args []value.Value) error {
	if len(args) != 1 {
		return fmt.Errorf("type expects 1 argument, got %d", len(args))
	}
	_, err := fmt.Fprintf(nat.Stdout, "(%s)\n", args[0].Type())
	return err
}
