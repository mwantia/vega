// Package vm implements the Vega virtual machine.
package vm

import (
	"fmt"
	"io"
	"os"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/value"
	libvfs "github.com/mwantia/vfs"
)

const (
	StackSize = 1024
	MaxFrames = 256
)

// New creates a new VM.
func NewVirtualMachine() *VirtualMachine {
	vm := &VirtualMachine{
		stack:      make([]value.Value, StackSize),
		sp:         0,
		globals:    make(map[string]value.Value),
		builtins:   make(map[string]BuiltinFunc),
		frames:     make([]*CallFrame, MaxFrames),
		frameIndex: 0,
		stdout:     os.Stdout,
		stderr:     os.Stderr,
		stdin:      os.Stdin,
		iterators:  make(map[string]value.Iterator),
	}
	vm.registerBuiltinFunctions()

	return vm
}

// SetVFS attaches a VFS implementation to the virtual machine.
func (vm *VirtualMachine) SetVFS(fs libvfs.VirtualFileSystem) {
	vm.vfs = fs
}

// GetVFS returns the attached VFS, or nil if none is attached.
func (vm *VirtualMachine) GetVFS() libvfs.VirtualFileSystem {
	return vm.vfs
}

// SetStdin sets the stdin reader.
func (vm *VirtualMachine) SetStdin(r io.Reader) error {
	if r == nil {
		return fmt.Errorf("failed: empty reader as 'stdin' is not allowed")
	}

	vm.stdin = r
	return nil
}

// SetStdout sets the stdout writer.
func (vm *VirtualMachine) SetStdout(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("failed: empty writer as 'stdout' is not allowed")
	}

	vm.stdout = w
	return nil
}

// SetStderr sets the stderr writer.
func (vm *VirtualMachine) SetStderr(w io.Writer) error {
	if w == nil {
		return fmt.Errorf("failed: empty writer as 'stderr' is not allowed")
	}

	vm.stderr = w
	return nil
}

// Run executes bytecode and returns the exit code.
func (vm *VirtualMachine) Run(bytecode *compiler.Bytecode) (int, error) {
	// Create initial frame
	frame := &CallFrame{
		bytecode: bytecode,
		ip:       0,
		bp:       0,
		locals:   make(map[string]value.Value),
	}
	vm.frames[0] = frame
	vm.frameIndex = 0

	// Execute
	err := vm.execute()
	if err != nil {
		return 1, err
	}

	// Return exit code from stack if available
	if vm.sp > 0 {
		if intVal, ok := vm.stack[vm.sp-1].(*value.IntValue); ok {
			return int(intVal.Val), nil
		}
	}
	return 0, nil
}

// execute is the main execution loop.
func (vm *VirtualMachine) execute() error {
	for {
		frame := vm.frames[vm.frameIndex]
		if frame.ip >= len(frame.bytecode.Instructions) {
			// End of bytecode
			if vm.frameIndex == 0 {
				return nil
			}
			// Return from function without explicit return
			vm.frameIndex--
			continue
		}

		instr := frame.bytecode.Instructions[frame.ip]
		frame.ip++

		err := vm.executeInstruction(instr, frame)
		if err != nil {
			if err == errReturn {
				if vm.frameIndex == 0 {
					return nil
				}
				vm.frameIndex--
				continue
			}
			return fmt.Errorf("line %d: %w", instr.Line, err)
		}
	}
}

var errReturn = fmt.Errorf("return")

func (vm *VirtualMachine) executeInstruction(instr compiler.Instruction, frame *CallFrame) error {
	switch instr.Op {
	case compiler.OpLoadConst:
		vm.push(frame.bytecode.Constants[instr.Arg])

	case compiler.OpLoadVar:
		val, ok := frame.locals[instr.Name]
		if !ok {
			val, ok = vm.globals[instr.Name]
			if !ok {
				return fmt.Errorf("undefined variable: %s", instr.Name)
			}
		}
		vm.push(val)

	case compiler.OpStoreVar:
		val := vm.pop()
		// Store in locals if we're in a function, otherwise globals
		if vm.frameIndex > 0 {
			frame.locals[instr.Name] = val
		} else {
			vm.globals[instr.Name] = val
		}

	case compiler.OpLoadAttr:
		obj := vm.pop()
		val, err := vm.getAttribute(obj, instr.Name)
		if err != nil {
			return err
		}
		vm.push(val)

	case compiler.OpPop:
		vm.pop()

	case compiler.OpDup:
		val := vm.peek()
		vm.push(val)

	case compiler.OpAdd:
		return vm.binaryOp(func(l, r value.Value) (value.Value, error) {
			// Handle string concatenation
			if ls, ok := l.(*value.StringValue); ok {
				return value.NewString(ls.Val + vm.toString(r)), nil
			}
			if ln, ok := l.(value.Numeric); ok {
				return ln.Add(r)
			}
			return nil, fmt.Errorf("cannot add %s and %s", l.Type(), r.Type())
		})

	case compiler.OpSub:
		return vm.numericOp(func(l, r value.Numeric) (value.Value, error) {
			return l.Sub(r.(value.Value))
		})

	case compiler.OpMul:
		return vm.numericOp(func(l, r value.Numeric) (value.Value, error) {
			return l.Mul(r.(value.Value))
		})

	case compiler.OpDiv:
		return vm.numericOp(func(l, r value.Numeric) (value.Value, error) {
			return l.Div(r.(value.Value))
		})

	case compiler.OpMod:
		return vm.numericOp(func(l, r value.Numeric) (value.Value, error) {
			return l.Mod(r.(value.Value))
		})

	case compiler.OpNeg:
		val := vm.pop()
		if n, ok := val.(value.Numeric); ok {
			vm.push(n.Neg())
		} else {
			return fmt.Errorf("cannot negate %s", val.Type())
		}

	case compiler.OpNot:
		val := vm.pop()
		vm.push(value.NewBool(!val.Boolean()))

	case compiler.OpEq:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewBool(left.Equal(right)))

	case compiler.OpNotEq:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewBool(!left.Equal(right)))

	case compiler.OpLt:
		return vm.compareOp(func(cmp int) bool { return cmp < 0 })

	case compiler.OpLte:
		return vm.compareOp(func(cmp int) bool { return cmp <= 0 })

	case compiler.OpGt:
		return vm.compareOp(func(cmp int) bool { return cmp > 0 })

	case compiler.OpGte:
		return vm.compareOp(func(cmp int) bool { return cmp >= 0 })

	case compiler.OpAnd:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewBool(left.Boolean() && right.Boolean()))

	case compiler.OpOr:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewBool(left.Boolean() || right.Boolean()))

	case compiler.OpJmp:
		frame.ip = instr.Arg

	case compiler.OpJmpIfFalse:
		cond := vm.pop()
		if !cond.Boolean() {
			frame.ip = instr.Arg
		}

	case compiler.OpJmpIfTrue:
		cond := vm.pop()
		if cond.Boolean() {
			frame.ip = instr.Arg
		}

	case compiler.OpCall:
		return vm.callFunction(instr.Name, instr.Arg)

	case compiler.OpCallMethod:
		return vm.callMethod(instr.Name, instr.Arg)

	case compiler.OpReturn:
		// Return value is on top of stack
		return errReturn

	case compiler.OpBuildArray:
		elements := make([]value.Value, instr.Arg)
		for i := instr.Arg - 1; i >= 0; i-- {
			elements[i] = vm.pop()
		}
		vm.push(value.NewArray(elements))

	case compiler.OpBuildMap:
		m := value.NewMap()
		// Pop key-value pairs (pushed in order, so pop in reverse)
		pairs := make([]struct {
			key string
			val value.Value
		}, instr.Arg)
		for i := instr.Arg - 1; i >= 0; i-- {
			val := vm.pop()
			key := vm.pop()
			pairs[i] = struct {
				key string
				val value.Value
			}{key.(*value.StringValue).Val, val}
		}
		for _, p := range pairs {
			m.Set(p.key, p.val)
		}
		vm.push(m)

	case compiler.OpIndex:
		index := vm.pop()
		obj := vm.pop()
		if indexable, ok := obj.(value.Indexable); ok {
			val, err := indexable.Index(index)
			if err != nil {
				return err
			}
			vm.push(val)
		} else {
			return fmt.Errorf("cannot index %s", obj.Type())
		}

	case compiler.OpSetIndex:
		val := vm.pop()
		index := vm.pop()
		obj := vm.pop()
		if indexable, ok := obj.(value.Indexable); ok {
			if err := indexable.SetIndex(index, val); err != nil {
				return err
			}
		} else {
			return fmt.Errorf("cannot index %s", obj.Type())
		}

	case compiler.OpIterInit:
		obj := vm.pop()
		if iterable, ok := obj.(value.Iterable); ok {
			vm.iterators[instr.Name] = iterable.Iterator()
		} else {
			return fmt.Errorf("cannot iterate over %s", obj.Type())
		}

	case compiler.OpIterNext:
		iter, ok := vm.iterators[instr.Name]
		if !ok {
			return fmt.Errorf("iterator not initialized: %s", instr.Name)
		}
		if iter.Next() {
			// Store current value in loop variable
			if vm.frameIndex > 0 {
				frame.locals[instr.Name] = iter.Value()
			} else {
				vm.globals[instr.Name] = iter.Value()
			}
		} else {
			// Iteration complete, jump to end
			delete(vm.iterators, instr.Name)
			frame.ip = instr.Arg
		}

	case compiler.OpConcat:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewString(vm.toString(left) + vm.toString(right)))

	case compiler.OpPipe:
		// For now, pipe just passes through (will be expanded for stream handling)
		// The right side should have already been evaluated with left as context
		// This is a placeholder - full pipe support requires stream infrastructure

	default:
		return fmt.Errorf("unknown opcode: %s", instr.Op)
	}

	return nil
}

// Stack operations

func (vm *VirtualMachine) push(v value.Value) {
	if vm.sp >= StackSize {
		panic("stack overflow")
	}
	vm.stack[vm.sp] = v
	vm.sp++
}

func (vm *VirtualMachine) pop() value.Value {
	if vm.sp == 0 {
		panic("stack underflow")
	}
	vm.sp--
	return vm.stack[vm.sp]
}

func (vm *VirtualMachine) peek() value.Value {
	if vm.sp == 0 {
		return value.Nil
	}
	return vm.stack[vm.sp-1]
}

// Helper functions

func (vm *VirtualMachine) binaryOp(fn func(l, r value.Value) (value.Value, error)) error {
	right := vm.pop()
	left := vm.pop()
	result, err := fn(left, right)
	if err != nil {
		return err
	}
	vm.push(result)
	return nil
}

func (vm *VirtualMachine) numericOp(fn func(l, r value.Numeric) (value.Value, error)) error {
	right := vm.pop()
	left := vm.pop()
	ln, lok := left.(value.Numeric)
	rn, rok := right.(value.Numeric)
	if !lok || !rok {
		return fmt.Errorf("cannot perform arithmetic on %s and %s", left.Type(), right.Type())
	}
	result, err := fn(ln, rn)
	if err != nil {
		return err
	}
	vm.push(result)
	return nil
}

func (vm *VirtualMachine) compareOp(fn func(int) bool) error {
	right := vm.pop()
	left := vm.pop()
	if lc, ok := left.(value.Comparable); ok {
		cmp, ok := lc.Compare(right)
		if !ok {
			return fmt.Errorf("cannot compare %s and %s", left.Type(), right.Type())
		}
		vm.push(value.NewBool(fn(cmp)))
		return nil
	}
	return fmt.Errorf("cannot compare %s", left.Type())
}

func (vm *VirtualMachine) toString(v value.Value) string {
	return v.String()
}

func (vm *VirtualMachine) getAttribute(obj value.Value, name string) (value.Value, error) {
	switch o := obj.(type) {
	case *value.NamespaceValue:
		// Get member from namespace (e.g., sys.stdin)
		if v, ok := o.Get(name); ok {
			return v, nil
		}
		return nil, fmt.Errorf("namespace '%s' has no member '%s'", o.Name(), name)
	case *value.MapValue:
		if v, ok := o.Get(name); ok {
			return v, nil
		}
		return value.Nil, nil
	case *value.ArrayValue:
		if name == "length" {
			return value.NewInt(int64(o.Len())), nil
		}
	case *value.StringValue:
		if name == "length" {
			return value.NewInt(int64(len(o.Val))), nil
		}
	case *value.StreamValue:
		// Stream properties
		switch name {
		case "name":
			return value.NewString(o.Name()), nil
		case "closed":
			return value.NewBool(o.IsClosed()), nil
		case "canRead":
			return value.NewBool(o.CanRead()), nil
		case "canWrite":
			return value.NewBool(o.CanWrite()), nil
		}
	}
	return nil, fmt.Errorf("cannot get attribute '%s' from %s", name, obj.Type())
}

func (vm *VirtualMachine) callFunction(name string, argCount int) error {
	// Check built-ins first
	if builtin, ok := vm.builtins[name]; ok {
		args := make([]value.Value, argCount)
		for i := argCount - 1; i >= 0; i-- {
			args[i] = vm.pop()
		}
		result, err := builtin(vm, args)
		if err != nil {
			return err
		}
		if result != nil {
			vm.push(result)
		} else {
			vm.push(value.Nil)
		}
		return nil
	}

	// Check user-defined functions
	var fn *compiler.FunctionValue
	if val, ok := vm.globals[name]; ok {
		fn, ok = val.(*compiler.FunctionValue)
		if !ok {
			return fmt.Errorf("'%s' is not a function", name)
		}
	} else {
		return fmt.Errorf("undefined function: %s", name)
	}

	if argCount != len(fn.Parameters) {
		return fmt.Errorf("function '%s' expects %d arguments, got %d", name, len(fn.Parameters), argCount)
	}

	// Create new frame
	if vm.frameIndex >= MaxFrames-1 {
		return fmt.Errorf("stack overflow: too many nested function calls")
	}

	newFrame := &CallFrame{
		bytecode: fn.Bytecode,
		ip:       0,
		bp:       vm.sp - argCount,
		locals:   make(map[string]value.Value),
	}

	// Pop arguments and assign to parameters
	args := make([]value.Value, argCount)
	for i := argCount - 1; i >= 0; i-- {
		args[i] = vm.pop()
	}
	for i, param := range fn.Parameters {
		newFrame.locals[param] = args[i]
	}

	vm.frameIndex++
	vm.frames[vm.frameIndex] = newFrame

	return nil
}

func (vm *VirtualMachine) callMethod(name string, argCount int) error {
	// Pop arguments first
	args := make([]value.Value, argCount)
	for i := argCount - 1; i >= 0; i-- {
		args[i] = vm.pop()
	}
	// Then pop object
	obj := vm.pop()

	result, err := vm.invokeMethod(obj, name, args)
	if err != nil {
		return err
	}
	vm.push(result)
	return nil
}

func (vm *VirtualMachine) invokeMethod(obj value.Value, name string, args []value.Value) (value.Value, error) {
	switch o := obj.(type) {
	case *value.StreamValue:
		return vm.streamMethod(o, name, args)
	case *value.NamespaceValue:
		return vm.namespaceMethod(o, name, args)
	case *value.ArrayValue:
		return vm.arrayMethod(o, name, args)
	case *value.StringValue:
		return vm.stringMethod(o, name, args)
	case *value.MapValue:
		return vm.mapMethod(o, name, args)
	}
	return nil, fmt.Errorf("cannot call method '%s' on %s", name, obj.Type())
}

func (vm *VirtualMachine) arrayMethod(arr *value.ArrayValue, name string, args []value.Value) (value.Value, error) {
	switch name {
	case "push":
		if len(args) != 1 {
			return nil, fmt.Errorf("push expects 1 argument")
		}
		arr.Push(args[0])
		return value.Nil, nil
	case "pop":
		return arr.Pop()
	case "length":
		return value.NewInt(int64(arr.Len())), nil
	default:
		return nil, fmt.Errorf("unknown array method: %s", name)
	}
}

func (vm *VirtualMachine) stringMethod(str *value.StringValue, name string, args []value.Value) (value.Value, error) {
	switch name {
	case "length":
		return value.NewInt(int64(len(str.Val))), nil
	case "upper":
		return value.NewString(toUpper(str.Val)), nil
	case "lower":
		return value.NewString(toLower(str.Val)), nil
	default:
		return nil, fmt.Errorf("unknown string method: %s", name)
	}
}

func (vm *VirtualMachine) mapMethod(m *value.MapValue, name string, args []value.Value) (value.Value, error) {
	switch name {
	case "keys":
		keys := m.Keys()
		elements := make([]value.Value, len(keys))
		for i, k := range keys {
			elements[i] = value.NewString(k)
		}
		return value.NewArray(elements), nil
	case "length":
		return value.NewInt(int64(m.Len())), nil
	default:
		return nil, fmt.Errorf("unknown map method: %s", name)
	}
}

func (vm *VirtualMachine) streamMethod(s *value.StreamValue, name string, args []value.Value) (value.Value, error) {
	switch name {
	case "read":
		// read() - read all available data
		return s.Read()

	case "readln":
		// readln() - read one line
		return s.ReadLine()

	case "readn":
		// readn(n) - read n bytes
		if len(args) != 1 {
			return nil, fmt.Errorf("readn expects 1 argument, got %d", len(args))
		}
		n, ok := args[0].(*value.IntValue)
		if !ok {
			return nil, fmt.Errorf("readn expects integer argument, got %s", args[0].Type())
		}
		return s.ReadN(int(n.Val))

	case "write":
		// write(data) - write data to stream
		if len(args) != 1 {
			return nil, fmt.Errorf("write expects 1 argument, got %d", len(args))
		}
		return s.Write(args[0])

	case "writeln":
		// writeln(data) - write data followed by newline
		if len(args) != 1 {
			return nil, fmt.Errorf("writeln expects 1 argument, got %d", len(args))
		}
		return s.WriteLine(args[0])

	case "close":
		// close() - close the stream
		if err := s.Close(); err != nil {
			return nil, err
		}
		return value.Nil, nil

	case "isClosed":
		// isClosed() - check if stream is closed
		return value.NewBool(s.IsClosed()), nil

	case "flush":
		// flush() - flush buffered data
		if err := s.Flush(); err != nil {
			return nil, err
		}
		return value.Nil, nil

	case "copy":
		// copy(dest) - copy all data from this stream to dest stream
		if len(args) != 1 {
			return nil, fmt.Errorf("copy expects 1 argument (dest stream), got %d", len(args))
		}
		dest, ok := args[0].(*value.StreamValue)
		if !ok {
			return nil, fmt.Errorf("copy expects stream argument, got %s", args[0].Type())
		}
		if !s.CanRead() {
			return nil, fmt.Errorf("source stream is not readable")
		}
		if !dest.CanWrite() {
			return nil, fmt.Errorf("destination stream is not writable")
		}
		n, err := io.Copy(dest.Writer(), s.Reader())
		if err != nil {
			return nil, fmt.Errorf("copy failed: %w", err)
		}
		return value.NewInt(n), nil

	default:
		return nil, fmt.Errorf("unknown stream method: %s", name)
	}
}

func (vm *VirtualMachine) namespaceMethod(ns *value.NamespaceValue, name string, args []value.Value) (value.Value, error) {
	// Check if namespace has a registered method
	if method, ok := ns.GetMethod(name); ok {
		return method(args)
	}
	return nil, fmt.Errorf("namespace '%s' has no method '%s'", ns.Name(), name)
}

// Simple string case conversion (ASCII only for now)
func toUpper(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'a' && c <= 'z' {
			b[i] = c - 32
		}
	}
	return string(b)
}

func toLower(s string) string {
	b := []byte(s)
	for i, c := range b {
		if c >= 'A' && c <= 'Z' {
			b[i] = c + 32
		}
	}
	return string(b)
}

// GetGlobal returns a global variable value.
func (vm *VirtualMachine) GetGlobal(name string) (value.Value, bool) {
	v, ok := vm.globals[name]
	return v, ok
}

// SetGlobal sets a global variable.
func (vm *VirtualMachine) SetGlobal(name string, val value.Value) {
	vm.globals[name] = val
}

// StackTop returns the top value on the stack without popping.
func (vm *VirtualMachine) StackTop() value.Value {
	if vm.sp == 0 {
		return nil
	}
	return vm.stack[vm.sp-1]
}

// LastPopped returns what would have been the last popped value.
func (vm *VirtualMachine) LastPopped() value.Value {
	return vm.stack[vm.sp]
}
