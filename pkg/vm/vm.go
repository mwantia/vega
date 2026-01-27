// Package vm implements the Vega virtual machine.
package vm

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

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
	ctx, cancel := context.WithCancel(context.Background())
	vm := &VirtualMachine{
		stack:      make([]value.Value, StackSize),
		sp:         0,
		globals:    make(map[string]value.Value),
		builtins:   make(map[string]BuiltinFunc),
		ctx:        ctx,
		cancel:     cancel,
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

// SetContext sets the VM's context for cancellation and timeouts.
// This replaces any existing context and creates a new cancellable child context.
func (vm *VirtualMachine) SetContext(ctx context.Context) {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Cancel any existing context
	if vm.cancel != nil {
		vm.cancel()
	}

	// Create new cancellable context from the provided parent
	vm.ctx, vm.cancel = context.WithCancel(ctx)
}

// Context returns the VM's current context.
// This is safe to call from builtin functions.
func (vm *VirtualMachine) Context() context.Context {
	vm.mu.RLock()
	defer vm.mu.RUnlock()

	if vm.ctx == nil {
		return context.Background()
	}
	return vm.ctx
}

// Cancel cancels the VM's context, which will interrupt any running operations.
func (vm *VirtualMachine) Cancel() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	if vm.cancel != nil {
		vm.cancel()
	}
}

// Shutdown gracefully shuts down the VM by cancelling its context.
// This should be called when the VM is no longer needed.
func (vm *VirtualMachine) Shutdown() {
	vm.Cancel()
}

// Reset resets the VM state and creates a fresh context.
// This is useful for running multiple scripts in sequence.
func (vm *VirtualMachine) Reset() {
	vm.mu.Lock()
	defer vm.mu.Unlock()

	// Cancel any existing context
	if vm.cancel != nil {
		vm.cancel()
	}

	// Create fresh context
	vm.ctx, vm.cancel = context.WithCancel(context.Background())

	// Reset execution state
	vm.sp = 0
	vm.frameIndex = 0
	vm.iterators = make(map[string]value.Iterator)
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
		if intVal, ok := vm.stack[vm.sp-1].(*value.Integer); ok {
			return int(intVal.Value), nil
		}
	}
	return 0, nil
}

// execute is the main execution loop.
func (vm *VirtualMachine) execute() error {
	for {
		// Check for context cancellation
		select {
		case <-vm.Context().Done():
			return vm.Context().Err()
		default:
			// Continue execution
		}

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
			if ls, ok := l.(*value.String); ok {
				return value.NewString(ls.Value + vm.toString(r)), nil
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
		vm.push(value.NewBoolean(!val.Boolean()))

	case compiler.OpEq:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewBoolean(left.Equal(right)))

	case compiler.OpNotEq:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewBoolean(!left.Equal(right)))

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
		vm.push(value.NewBoolean(left.Boolean() && right.Boolean()))

	case compiler.OpOr:
		right := vm.pop()
		left := vm.pop()
		vm.push(value.NewBoolean(left.Boolean() || right.Boolean()))

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
			}{key.(*value.String).Value, val}
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
		vm.push(value.NewBoolean(fn(cmp)))
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
	case *value.Metadata:
		// Get metadata field (e.g., meta.key, meta.size, meta.isDir)
		return o.GetField(name)
	case *value.Map:
		if v, ok := o.Get(name); ok {
			return v, nil
		}
		return value.Nil, nil
	case *value.Array:
		if name == "length" {
			return value.NewInteger(int64(o.Len())), nil
		}
	case *value.String:
		if name == "length" {
			return value.NewInteger(int64(len(o.Value))), nil
		}
	case *value.Stream:
		// Stream properties
		switch name {
		case "name":
			return value.NewString(o.Name()), nil
		case "closed":
			return value.NewBoolean(o.IsClosed()), nil
		case "canRead":
			return value.NewBoolean(o.CanRead()), nil
		case "canWrite":
			return value.NewBoolean(o.CanWrite()), nil
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
	// prepare method name
	methodName := strings.ToLower(strings.TrimSpace(name))
	if methodName == "" {
		return fmt.Errorf("empty method name defined: '%s'", name)
	}

	var result value.Value
	// Execute general methods that all values support
	// or that are supported based on interfaces (e.g. Comparable)
	switch methodName {
	case "string":
		s := obj.String()
		result = value.NewString(s)
	case "type":
		s := obj.Type()
		result = value.NewType(s)
	case "boolean":
		b := obj.Boolean()
		result = value.NewBoolean(b)
	case "equal":
		if len(args) != 1 {
			return fmt.Errorf("method 'equal' expects 1 argument, got %d", len(args))
		}
		b := obj.Equal(args[0])
		result = value.NewBoolean(b)
	case "compare":
		if len(args) != 1 {
			return fmt.Errorf("method 'compare' expects 1 argument, got %d", len(args))
		}
		if compare, ok1 := obj.(value.Comparable); ok1 {
			if i, ok2 := compare.Compare(args[0]); ok2 {
				result = value.NewInteger(int64(i))
			}
		}
	default:
		if method, ok := obj.(value.Methodable); ok {
			var err error
			if result, err = method.Method(methodName, args); err != nil {
				return fmt.Errorf("failed to call method '%s': %v", method, err)
			}
		} else {
			return fmt.Errorf("unknown method call: '%s'", name)
		}
	}

	vm.push(result)
	return nil
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
