package compiler

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/slot"
)

// Preallocated boolean constants — shared across all compilations to avoid
// per-literal []byte allocations (AddConstant deduplicates by value).
var (
	boolConstTrue  = Constant{Tag: slot.TagBoolean, Data: []byte{1}}
	boolConstFalse = Constant{Tag: slot.TagBoolean, Data: []byte{0}}
)

type Compiler struct {
	scope    *SymbolTable
	stencils map[string]*StencilDefinition
}

func NewCompiler() *Compiler {
	return &Compiler{
		stencils: make(map[string]*StencilDefinition),
	}
}

func (c *Compiler) Compile(ast parser.AST) (*ByteCode, error) {
	byteCode := &ByteCode{
		Instructions: make([]Instruction, 0),
		Constants:    make([]Constant, 0),
		Names:        make([]string, 0),
		LoopStack:    nil,
		Functions:    make(map[string]*FunctionDefinition),
	}

	statements := ast.Statements()
	if len(statements) == 0 {
		return nil, fmt.Errorf("invalid program defined: expected statements")
	}

	// Initialize the top-level scope. Every variable lives in this scope
	// (or in a per-function scope) and is backed by the global allocator.
	// When reusing the same Compiler across REPL commands, preserve the
	// existing scope so variable name→slot-ID assignments survive.
	if c.scope == nil {
		c.scope = newSymbolTable()
	}

	for _, stmt := range statements {
		if err := c.compileStatement(byteCode, stmt); err != nil {
			return nil, fmt.Errorf("failed to compile statement '%s': %v", stmt.String(), err)
		}
	}

	return byteCode, nil
}

func (c *Compiler) compileStatement(b *ByteCode, statement parser.Statement) error {
	switch s := statement.(type) {
	case *parser.AssignmentStatement:
		switch v := s.Value.(type) {
		case *parser.StructExpression:
			return c.compileStructAssignment(b, s, v)
		case *parser.TupleExpression:
			return c.compileTupleAssignment(b, s, v)
		case *parser.PointerExpression:
			return c.compilePointerAssignment(b, s, v)
		default:
			return c.compileScalarAssignment(b, s)
		}

	case *parser.FreeStatement:
		name := s.Name.Value
		info, exists := c.scope.Lookup(name)
		if !exists {
			return fmt.Errorf("free: undefined variable '%s'", name)
		}

		b.EmitArg(OpVarFREE, info.SlotID, s.Position().Line)
		c.scope.Remove(name)

	case *parser.StructStatement:
		// Build a stencil from the field declarations and register it.
		// This is pure compile-time data — no bytecode emitted.
		stencil := &StencilDefinition{
			Name:   s.Name,
			Fields: make([]StencilFieldLayout, 0, len(s.Fields)),
		}
		offset := 0
		for _, f := range s.Fields {
			if f.Capacity > 0 {
				// Parameterised slice field: string<N> or byte<N>.
				stencil.Fields = append(stencil.Fields, StencilFieldLayout{
					Name:     f.Name,
					Offset:   offset,
					Tag:      slot.TagSlice,
					Capacity: f.Capacity,
				})
				offset += f.Capacity
			} else {
				tag, ok := slot.TagForName(f.Type)
				if !ok {
					return fmt.Errorf("struct '%s': unknown type '%s' for field '%s'", s.Name, f.Type, f.Name)
				}
				size := slot.SizeForTag(tag)
				if size == 0 {
					return fmt.Errorf("struct '%s': type '%s' for field '%s' requires an explicit capacity (use %s<N>)", s.Name, f.Type, f.Name, f.Type)
				}
				stencil.Fields = append(stencil.Fields, StencilFieldLayout{
					Name:   f.Name,
					Offset: offset,
					Tag:    tag,
				})
				offset += size
			}
		}
		stencil.TotalSize = offset
		c.stencils[s.Name] = stencil

		// Also register into the global descriptor registry so the stencil is
		// visible to external code and to method lookups on stencil types.
		descFields := make([]descriptor.FieldLayoutDescriptor, len(stencil.Fields))
		for i, f := range stencil.Fields {
			descFields[i] = descriptor.FieldLayoutDescriptor{
				Name:     f.Name,
				Tag:      f.Tag,
				Offset:   f.Offset,
				Capacity: f.Capacity,
			}
		}
		descriptor.Global.RegisterStencil(&descriptor.StencilDescriptor{
			Name:   s.Name,
			Fields: descFields,
			Size:   stencil.TotalSize,
		})

	case *parser.FunctionStatement:
		params := make([]ParamDefinition, len(s.Parameters))

		for i, p := range s.Parameters {
			if len(p.Constraints) != 1 {
				return fmt.Errorf("function '%s': parameter '%s' must have exactly one type constraint",
					s.Name.Value, p.Value)
			}
			ident, ok := p.Constraints[0].(*parser.IdentifierExpression)
			if !ok {
				return fmt.Errorf("function '%s': parameter '%s' type must be an identifier",
					s.Name.Value, p.Value)
			}
			if tag, ok := slot.TagForName(ident.Value); ok {
				// Primitive type parameter (int, long, float, …)
				mask := slot.MaskForTag(tag)
				params[i] = ParamDefinition{Name: p.Value, Tag: tag, Mask: mask}
			} else if stencil, ok := c.stencils[ident.Value]; ok {
				// Struct type parameter
				params[i] = ParamDefinition{Name: p.Value, Stencil: stencil}
			} else {
				return fmt.Errorf("function '%s': parameter '%s': unknown type '%s'",
					s.Name.Value, p.Value, ident.Value)
			}
		}

		// Compile the function body into a separate ByteCode.
		// Functions share the top-level function table so they can call each other.
		fnCode := &ByteCode{
			Instructions: make([]Instruction, 0),
			Constants:    make([]Constant, 0),
			Names:        make([]string, 0),
			Functions:    b.Functions,
		}

		// Enter function scope — parameters become variables in this scope.
		// Use defer so the outer scope is always restored, even on compile error.
		oldScope := c.scope
		c.scope = newSymbolTable()
		defer func() { c.scope = oldScope }()

		// For each parameter: pull from the pending-args buffer, allocate a slot,
		// and store the slot. Primitive and struct params use different sequences.
		for i, param := range params {
			if param.Stencil != nil {
				// Struct parameter: allocate a stencil slot and bulk-copy the raw bytes.
				// OpLoadArgStencil: Argument=pending index, Extra=slot ID, Offset=total size.
				info := c.scope.DefineStencil(param.Name, param.Stencil)
				params[i].SlotID = info.SlotID
				fnCode.EmitField(OpLoadArgStencil, i, param.Stencil.TotalSize, byte(info.SlotID), s.Position().Line)
			} else {
				// Primitive parameter: push from pending args, allocate slot, store.
				fnCode.EmitArg(OpLoadArg, i, s.Position().Line)
				info := c.scope.Define(param.Name, param.Tag, param.Mask)
				params[i].SlotID = info.SlotID
				fnCode.EmitArgExtra(OpVarALLOC, info.SlotID, param.Mask, s.Position().Line)
				fnCode.EmitArg(OpVarSTORE, info.SlotID, s.Position().Line)
			}
		}

		// Compile body statements inside the function scope.
		for _, stmt := range s.Body.Statements {
			if err := c.compileStatement(fnCode, stmt); err != nil {
				return fmt.Errorf("in function '%s': %w", s.Name.Value, err)
			}
		}

		// Implicit void return at the end of the function.
		fnCode.Emit(OpReturn, s.Position().Line)

		// Compute the total fixed-size frame allocation by scanning emitted
		// instructions. This is the sum of all slot allocation sizes.
		// OpSliceALLOC uses Offset as capacity; OpStencilALLOC and
		// OpLoadArgStencil use Offset as total size; OpVarALLOC derives
		// size from the type mask.
		frameSize := 0
		for _, instr := range fnCode.Instructions {
			switch instr.Operation {
			case OpVarALLOC:
				frameSize += slot.MaxSizeForMask(instr.Extra)
			case OpSliceALLOC, OpStencilALLOC, OpLoadArgStencil:
				frameSize += instr.Offset
			}
		}

		b.Functions[s.Name.Value] = &FunctionDefinition{
			Name:      s.Name.Value,
			ByteCode:  fnCode,
			Params:    params,
			FrameSize: frameSize,
		}

	case *parser.ReturnStatement:
		if s.Value != nil {
			if _, err := c.compileExpression(b, s.Value); err != nil {
				return fmt.Errorf("return value: %w", err)
			}
			// extra=1 signals that a return value is on the expr stack.
			b.EmitArgExtra(OpReturn, 0, 1, s.Position().Line)
		} else {
			b.Emit(OpReturn, s.Position().Line)
		}

	case *parser.CallStatement:
		switch fn := s.Function.(type) {
		case *parser.IdentifierExpression:
			for i, arg := range s.Arguments {
				if _, err := c.compileExpression(b, arg); err != nil {
					return fmt.Errorf("argument %d of call to '%s': %v", i, fn.Value, err)
				}
			}
			b.EmitNameArg(OpCall, fn.Value, len(s.Arguments), s.Position().Line)
		case *parser.MethodCallExpression:
			// Method call as a standalone statement — result is discarded.
			if _, err := c.compileExpression(b, fn); err != nil {
				return fmt.Errorf("method call statement: %w", err)
			}
			b.Emit(OpStackPOP, s.Position().Line)
		default:
			return fmt.Errorf("unsupported call target: %T", s.Function)
		}

	case *parser.DiscardStatement:
		return fmt.Errorf("discard statements not yet implemented")

	default:
		return fmt.Errorf("unknown statement type: %T", statement)
	}
	return nil
}

func (c *Compiler) compilePointerAssignment(b *ByteCode, s *parser.AssignmentStatement, ptrExpr *parser.PointerExpression) error {
	if _, err := c.compileExpression(b, ptrExpr.Offset); err != nil {
		return fmt.Errorf("failed to compile pointer offset: %v", err)
	}

	tag, ok := slot.TagForName(ptrExpr.TypeName)
	if !ok {
		return fmt.Errorf("unknown type name '%s' in pointer", ptrExpr.TypeName)
	}

	name := s.Name.Value
	if _, exists := c.scope.Lookup(name); !exists {
		mask := slot.MaskForTag(tag)
		c.scope.Define(name, tag, mask)
	}

	info, _ := c.scope.Lookup(name)
	b.EmitArgExtra(OpVarPTR, info.SlotID, byte(tag), s.Position().Line)
	return nil
}

func (c *Compiler) compileScalarAssignment(b *ByteCode, s *parser.AssignmentStatement) error {
	name := s.Name.Value

	// Check for explicit slice type constraint before compiling the RHS,
	// so we can detect and route slice assignments early.
	if len(s.Constraints) == 1 {
		if st, ok := s.Constraints[0].(*parser.SliceTypeExpression); ok {
			return c.compileSliceAssignment(b, s, st)
		}
	}

	// For a string literal with no explicit constraint, auto-infer capacity
	// from the literal length — the compiler knows the exact byte count.
	if strExpr, ok := s.Value.(*parser.StringExpression); ok && len(s.Constraints) == 0 {
		capacity := len(strExpr.Value)
		if capacity == 0 {
			capacity = 1
		}
		if _, exists := c.scope.Lookup(name); !exists {
			info := c.scope.DefineSlice(name, capacity)
			b.EmitField(OpSliceALLOC, info.SlotID, capacity, 0, s.Position().Line)
		}
		// Emit the constant and store into the (possibly pre-existing) slot.
		c.emitConst(b, slot.TagSlice, []byte(strExpr.Value), s.Position().Line)
		info, _ := c.scope.Lookup(name)
		b.EmitArg(OpVarSTORE, info.SlotID, s.Position().Line)
		return nil
	}

	rhsTag, err := c.compileExpression(b, s.Value)
	if err != nil {
		return fmt.Errorf("failed to compile assignment value: %v", err)
	}

	// Non-literal slice values (e.g. BUILD_STRING result) require an explicit
	// capacity declaration so the allocator knows how large a slot to reserve.
	if rhsTag == slot.TagSlice {
		return fmt.Errorf("string assignment to '%s' requires an explicit capacity: use `%s: string<N> = ...`", name, name)
	}

	if _, exists := c.scope.Lookup(name); !exists {
		var mask byte
		var inferredTag slot.TypeTag

		if len(s.Constraints) > 0 {
			mask, err = c.resolveConstraintMask(s.Constraints)
			if err != nil {
				return fmt.Errorf("type constraint for '%s': %v", name, err)
			}
			inferredTag = rhsTag
		} else {
			inferredTag = rhsTag
			mask = slot.MaskForTag(rhsTag)
		}

		info := c.scope.Define(name, inferredTag, mask)
		b.EmitArgExtra(OpVarALLOC, info.SlotID, mask, s.Position().Line)
	}

	info, _ := c.scope.Lookup(name)
	b.EmitArg(OpVarSTORE, info.SlotID, s.Position().Line)
	return nil
}

// compileSliceAssignment handles `name: string<N> = expr` and `name: byte<N> = expr`.
// The RHS must evaluate to TagSlice. On first use the variable is allocated as a
// fixed-capacity slot of N bytes; on re-assignment the existing slot is reused.
func (c *Compiler) compileSliceAssignment(b *ByteCode, s *parser.AssignmentStatement, st *parser.SliceTypeExpression) error {
	name := s.Name.Value

	rhsTag, err := c.compileExpression(b, s.Value)
	if err != nil {
		return fmt.Errorf("failed to compile slice assignment value: %v", err)
	}
	if rhsTag != slot.TagSlice {
		return fmt.Errorf("type mismatch: '%s' declared as %s<%d> but right-hand side is not a slice value", name, st.TypeName, st.Capacity)
	}

	if _, exists := c.scope.Lookup(name); !exists {
		info := c.scope.DefineSlice(name, st.Capacity)
		// OpSliceALLOC: Argument=slotID, Offset=capacity
		b.EmitField(OpSliceALLOC, info.SlotID, st.Capacity, 0, s.Position().Line)
	}

	info, _ := c.scope.Lookup(name)
	b.EmitArg(OpVarSTORE, info.SlotID, s.Position().Line)
	return nil
}

func (c *Compiler) compileStructAssignment(b *ByteCode, s *parser.AssignmentStatement, structExpr *parser.StructExpression) error {
	stencil, ok := c.stencils[structExpr.Name]
	if !ok {
		return fmt.Errorf("undefined struct type '%s'", structExpr.Name)
	}

	info := c.emitStencilInit(b, s.Name.Value, stencil, s.Position().Line)

	// Compile and store each field
	for _, fieldName := range structExpr.Order {
		fieldExpr := structExpr.Fields[fieldName]
		field, ok := stencil.LookupField(fieldName)
		if !ok {
			return fmt.Errorf("struct '%s' has no field '%s'", structExpr.Name, fieldName)
		}

		if _, err := c.compileExpression(b, fieldExpr); err != nil {
			return fmt.Errorf("failed to compile struct field '%s': %v", fieldName, err)
		}

		if field.Tag == slot.TagSlice {
			b.EmitFieldSize(OpFieldSTORE, info.SlotID, field.Offset, byte(field.Tag), field.Capacity, s.Position().Line)
		} else {
			b.EmitField(OpFieldSTORE, info.SlotID, field.Offset, byte(field.Tag), s.Position().Line)
		}
	}

	return nil
}

func (c *Compiler) compileTupleAssignment(b *ByteCode, s *parser.AssignmentStatement, tupleExpr *parser.TupleExpression) error {
	// Build an anonymous stencil from the element types
	stencil := &StencilDefinition{
		Name:   "",
		Fields: make([]StencilFieldLayout, 0, len(tupleExpr.Elements)),
	}
	offset := 0
	for i, elem := range tupleExpr.Elements {
		tag, err := c.inferTypeTag(elem)
		if err != nil {
			return fmt.Errorf("tuple element %d: %v", i, err)
		}
		stencil.Fields = append(stencil.Fields, StencilFieldLayout{
			Name:   fmt.Sprintf("%d", i),
			Offset: offset,
			Tag:    tag,
		})
		offset += slot.SizeForTag(tag)
	}
	stencil.TotalSize = offset

	info := c.emitStencilInit(b, s.Name.Value, stencil, s.Position().Line)

	// Compile and store each element
	for i, elem := range tupleExpr.Elements {
		if _, err := c.compileExpression(b, elem); err != nil {
			return fmt.Errorf("failed to compile tuple element %d: %v", i, err)
		}
		field := stencil.Fields[i]
		b.EmitField(OpFieldSTORE, info.SlotID, field.Offset, byte(field.Tag), s.Position().Line)
	}

	return nil
}

// emitStencilInit allocates a stencil slot on first use and returns its SymbolInfo.
// If the variable already exists in scope (re-assignment), the existing slot is reused.
func (c *Compiler) emitStencilInit(b *ByteCode, name string, stencil *StencilDefinition, line int) SymbolInfo {
	if _, exists := c.scope.Lookup(name); !exists {
		info := c.scope.DefineStencil(name, stencil)
		b.EmitField(OpStencilALLOC, info.SlotID, stencil.TotalSize, 0, line)
	}
	info, _ := c.scope.Lookup(name)
	return info
}

// emitConst adds a typed constant to the pool and emits OpLoadCONST.
func (c *Compiler) emitConst(b *ByteCode, tag slot.TypeTag, data []byte, line int) {
	idx := b.AddConstant(Constant{Tag: tag, Data: data})
	b.EmitArg(OpLoadCONST, idx, line)
}

func (c *Compiler) compileExpression(b *ByteCode, expr parser.Expression) (slot.TypeTag, error) {
	switch e := expr.(type) {
	case *parser.ByteExpression:
		c.emitConst(b, slot.TagByte, []byte{e.Value}, e.Position().Line)
		return slot.TagByte, nil
	case *parser.ShortExpression:
		data := make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(e.Value))
		c.emitConst(b, slot.TagShort, data, e.Position().Line)
		return slot.TagShort, nil
	case *parser.IntegerExpression:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(e.Value))
		c.emitConst(b, slot.TagInteger, data, e.Position().Line)
		return slot.TagInteger, nil
	case *parser.LongExpression:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(e.Value))
		c.emitConst(b, slot.TagLong, data, e.Position().Line)
		return slot.TagLong, nil
	case *parser.FloatExpression:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, math.Float32bits(e.Value))
		c.emitConst(b, slot.TagFloat, data, e.Position().Line)
		return slot.TagFloat, nil
	case *parser.DecimalExpression:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, math.Float64bits(e.Value))
		c.emitConst(b, slot.TagDecimal, data, e.Position().Line)
		return slot.TagDecimal, nil
	case *parser.BooleanExpression:
		bc := boolConstFalse
		if e.Value {
			bc = boolConstTrue
		}
		c.emitConst(b, bc.Tag, bc.Data, e.Position().Line)
		return slot.TagBoolean, nil
	case *parser.StringExpression:
		c.emitConst(b, slot.TagSlice, []byte(e.Value), e.Position().Line)
		return slot.TagSlice, nil
	case *parser.InterpolatedExpression:
		for _, part := range e.Parts {
			if _, err := c.compileExpression(b, part); err != nil {
				return 0, fmt.Errorf("interpolated string part: %w", err)
			}
		}
		b.EmitArg(OpBuildSTRING, len(e.Parts), e.Position().Line)
		return slot.TagSlice, nil
	case *parser.NilExpression:
		return 0, fmt.Errorf("nil literals are not allocable")
	case *parser.IdentifierExpression:
		info, exists := c.scope.Lookup(e.Value)
		if !exists {
			return 0, fmt.Errorf("undefined variable '%s'", e.Value)
		}
		if info.Stencil != nil {
			// Struct variable — push a snapshot of its raw bytes onto the expr stack
			// so it can be passed as a function argument via OpCall.
			b.EmitField(OpVarLoadRaw, info.SlotID, info.Stencil.TotalSize, 0, e.Position().Line)
		} else {
			b.EmitArg(OpVarLOAD, info.SlotID, e.Position().Line)
		}
		return info.Tag, nil
	case *parser.PointerExpression:
		// Inline pointer dereference used as an expression: *type(offset)
		// Compile the offset, then emit OpPtrLOAD so the VM reads directly
		// from the allocator at that offset and pushes the slot.
		tag, ok := slot.TagForName(e.TypeName)
		if !ok {
			return 0, fmt.Errorf("unknown type name '%s' in pointer expression", e.TypeName)
		}
		if _, err := c.compileExpression(b, e.Offset); err != nil {
			return 0, fmt.Errorf("failed to compile pointer offset: %v", err)
		}
		b.EmitArgExtra(OpPtrLOAD, 0, byte(tag), e.Position().Line)
		return tag, nil
	case *parser.AttributeExpression:
		// If the object is a known stencil (struct/tuple) variable, use static
		// compile-time field dispatch via OpFieldLOAD.
		if ident, ok := e.Object.(*parser.IdentifierExpression); ok {
			if info, exists := c.scope.Lookup(ident.Value); exists && info.Stencil != nil {
				fieldName := e.Attribute.Value
				field, ok := info.Stencil.LookupField(fieldName)
				if !ok {
					return 0, fmt.Errorf("struct '%s' has no field '%s'", info.Stencil.Name, fieldName)
				}
				if field.Tag == slot.TagSlice {
					b.EmitFieldSize(OpFieldLOAD, info.SlotID, field.Offset, byte(field.Tag), field.Capacity, e.Position().Line)
				} else {
					b.EmitField(OpFieldLOAD, info.SlotID, field.Offset, byte(field.Tag), e.Position().Line)
				}
				return field.Tag, nil
			}
		}
		// Generic fallback: compile the object and dispatch at runtime via the
		// descriptor registry.
		recvTag, err := c.compileExpression(b, e.Object)
		if err != nil {
			return 0, fmt.Errorf("member access object: %w", err)
		}
		b.EmitNameArg(OpGetMember, e.Attribute.Value, 0, e.Position().Line)
		// Use the descriptor registry to infer the return type at compile time.
		if desc, ok := descriptor.Global.LookupMember(recvTag, e.Attribute.Value); ok {
			return desc.ReturnTag, nil
		}
		return 0, nil

	case *parser.MethodCallExpression:
		// Compile the receiver object first, then arguments, then dispatch at
		// runtime via the descriptor registry.
		recvTag, err := c.compileExpression(b, e.Object)
		if err != nil {
			return 0, fmt.Errorf("method call receiver: %w", err)
		}
		desc, hasDesc := descriptor.Global.LookupMethod(recvTag, e.Method.Value)
		args := e.Arguments
		if hasDesc {
			if args, err = reorderArgs(desc.Params, args); err != nil {
				return 0, fmt.Errorf("method '%s': %w", e.Method.Value, err)
			}
		}
		for i, arg := range args {
			if _, err := c.compileExpression(b, arg); err != nil {
				return 0, fmt.Errorf("method '%s' argument %d: %w", e.Method.Value, i, err)
			}
		}
		b.EmitNameArg(OpCallMethod, e.Method.Value, len(args), e.Position().Line)
		if hasDesc {
			return desc.ReturnTag, nil
		}
		return 0, nil

	case *parser.CallExpression:
		fn, ok := e.Function.(*parser.IdentifierExpression)
		if !ok {
			return 0, fmt.Errorf("call expression: unsupported callee type %T", e.Function)
		}
		desc, hasDesc := descriptor.Global.LookupStatic(fn.Value)
		args := e.Arguments
		if hasDesc {
			var err error
			if args, err = reorderArgs(desc.Params, args); err != nil {
				return 0, fmt.Errorf("call '%s': %w", fn.Value, err)
			}
		}
		for i, arg := range args {
			if _, err := c.compileExpression(b, arg); err != nil {
				return 0, fmt.Errorf("argument %d of call to '%s': %w", i, fn.Value, err)
			}
		}
		b.EmitNameArgExtra(OpCall, fn.Value, len(args), 1, e.Position().Line)
		if hasDesc {
			return desc.ReturnTag, nil
		}
		return 0, nil

	case *parser.GroupedExpression:
		// Parenthesized expression — just compile the inner expression.
		return c.compileExpression(b, e.Expr)

	case *parser.InfixExpression:
		leftTag, err := c.compileExpression(b, e.Left)
		if err != nil {
			return 0, fmt.Errorf("infix left: %w", err)
		}
		if _, err := c.compileExpression(b, e.Right); err != nil {
			return 0, fmt.Errorf("infix right: %w", err)
		}
		line := e.Position().Line
		switch e.Operator {
		case "+":
			b.Emit(OpBinAdd, line)
			return leftTag, nil
		case "-":
			b.Emit(OpBinSub, line)
			return leftTag, nil
		case "*":
			b.Emit(OpBinMul, line)
			return leftTag, nil
		case "/":
			b.Emit(OpBinDiv, line)
			return leftTag, nil
		case "%":
			b.Emit(OpBinMod, line)
			return leftTag, nil
		case "==":
			b.Emit(OpCmpEQ, line)
			return slot.TagBoolean, nil
		case "!=":
			b.Emit(OpCmpNE, line)
			return slot.TagBoolean, nil
		case "<":
			b.Emit(OpCmpLT, line)
			return slot.TagBoolean, nil
		case "<=":
			b.Emit(OpCmpLE, line)
			return slot.TagBoolean, nil
		case ">":
			b.Emit(OpCmpGT, line)
			return slot.TagBoolean, nil
		case ">=":
			b.Emit(OpCmpGE, line)
			return slot.TagBoolean, nil
		case "&&":
			b.Emit(OpLogAnd, line)
			return slot.TagBoolean, nil
		case "||":
			b.Emit(OpLogOr, line)
			return slot.TagBoolean, nil
		default:
			return 0, fmt.Errorf("unknown infix operator '%s'", e.Operator)
		}

	case *parser.PrefixExpression:
		tag, err := c.compileExpression(b, e.Right)
		if err != nil {
			return 0, fmt.Errorf("prefix operand: %w", err)
		}
		line := e.Position().Line
		switch e.Operator {
		case "-":
			b.Emit(OpUnNeg, line)
			return tag, nil
		case "!":
			b.Emit(OpLogNot, line)
			return slot.TagBoolean, nil
		default:
			return 0, fmt.Errorf("unknown prefix operator '%s'", e.Operator)
		}

	default:
		return 0, fmt.Errorf("unknown expression type: %T", e)
	}
}

func (c *Compiler) resolveConstraintMask(constraints []parser.Expression) (byte, error) {
	var mask byte
	for _, constraint := range constraints {
		switch ct := constraint.(type) {
		case *parser.IdentifierExpression:
			tag, ok := slot.TagForName(ct.Value)
			if !ok {
				return 0, fmt.Errorf("unknown type name '%s'", ct.Value)
			}
			mask |= slot.MaskForTag(tag)
		case *parser.SliceTypeExpression:
			return 0, fmt.Errorf("slice type '%s<%d>' cannot be used in a union type constraint", ct.TypeName, ct.Capacity)
		default:
			return 0, fmt.Errorf("type constraint must be a type name, got %T", constraint)
		}
	}
	return mask, nil
}

// inferTypeTag determines the TypeTag of an expression without emitting bytecode.
// It is called only from compileTupleAssignment for stencil pre-scanning, where
// type information must be known before any code is emitted.
func (c *Compiler) inferTypeTag(expr parser.Expression) (slot.TypeTag, error) {
	switch expr := expr.(type) {
	case *parser.ByteExpression:
		return slot.TagByte, nil
	case *parser.ShortExpression:
		return slot.TagShort, nil
	case *parser.IntegerExpression:
		return slot.TagInteger, nil
	case *parser.LongExpression:
		return slot.TagLong, nil
	case *parser.FloatExpression:
		return slot.TagFloat, nil
	case *parser.DecimalExpression:
		return slot.TagDecimal, nil
	case *parser.BooleanExpression:
		return slot.TagBoolean, nil
	case *parser.StringExpression:
		return slot.TagSlice, nil // string literals are slice-typed; capacity must be declared
	case *parser.InterpolatedExpression:
		return slot.TagSlice, nil
	case *parser.PointerExpression:
		tag, ok := slot.TagForName(expr.TypeName)
		if !ok {
			return 0, fmt.Errorf("unknown type name '%s' in pointer", expr.TypeName)
		}
		return tag, nil
	case *parser.IdentifierExpression:
		if info, ok := c.scope.Lookup(expr.Value); ok {
			return info.Tag, nil
		}
		return 0, fmt.Errorf("cannot infer type from undefined variable '%s'", expr.Value)
	case *parser.AttributeExpression:
		ident, ok := expr.Object.(*parser.IdentifierExpression)
		if !ok {
			return 0, fmt.Errorf("cannot infer type from non-identifier attribute access")
		}
		if info, ok := c.scope.Lookup(ident.Value); ok && info.Stencil != nil {
			if field, ok := info.Stencil.LookupField(expr.Attribute.Value); ok {
				return field.Tag, nil
			}
			return 0, fmt.Errorf("struct '%s' has no field '%s'", info.Stencil.Name, expr.Attribute.Value)
		}
		return 0, fmt.Errorf("cannot infer type from attribute expression")
	case *parser.GroupedExpression:
		return c.inferTypeTag(expr.Expr)
	case *parser.MethodCallExpression:
		return 0, fmt.Errorf("cannot infer type from method call (use explicit type annotation)")
	case *parser.InfixExpression:
		switch expr.Operator {
		case "==", "!=", "<", "<=", ">", ">=", "&&", "||":
			return slot.TagBoolean, nil
		default:
			return c.inferTypeTag(expr.Left)
		}
	case *parser.PrefixExpression:
		if expr.Operator == "!" {
			return slot.TagBoolean, nil
		}
		return c.inferTypeTag(expr.Right)
	default:
		return 0, fmt.Errorf("cannot infer type from expression %T", expr)
	}
}

// reorderArgs resolves named arguments in rawArgs to positional order using
// the given param descriptors. If no named arguments are present the original
// slice is returned unchanged. Positional and named arguments may be mixed:
// positional args fill slots left-to-right, named args fill their declared
// position. Trailing absent optional params are trimmed from the result.
func reorderArgs(params []descriptor.ParameterLayoutDescriptor, rawArgs []parser.Expression) ([]parser.Expression, error) {
	hasNamed := false
	for _, a := range rawArgs {
		if _, ok := a.(*parser.NamedArgumentExpression); ok {
			hasNamed = true
			break
		}
	}
	if !hasNamed {
		return rawArgs, nil
	}
	if len(params) == 0 {
		return nil, fmt.Errorf("function accepts no parameters; named arguments are not allowed")
	}

	result := make([]parser.Expression, len(params))
	nextPos := 0
	for _, a := range rawArgs {
		if named, ok := a.(*parser.NamedArgumentExpression); ok {
			pos := -1
			for _, p := range params {
				if p.Name == named.Name {
					pos = p.Position
					break
				}
			}
			if pos < 0 {
				return nil, fmt.Errorf("unknown named argument '%s'", named.Name)
			}
			if result[pos] != nil {
				return nil, fmt.Errorf("argument '%s' provided more than once", named.Name)
			}
			result[pos] = named.Value
		} else {
			for nextPos < len(result) && result[nextPos] != nil {
				nextPos++
			}
			if nextPos >= len(result) {
				return nil, fmt.Errorf("too many positional arguments")
			}
			result[nextPos] = a
			nextPos++
		}
	}

	// Trim trailing nils (absent optional params) and validate no holes remain.
	end := len(result)
	for end > 0 && result[end-1] == nil {
		end--
	}
	for i := 0; i < end; i++ {
		if result[i] == nil {
			return nil, fmt.Errorf("missing argument for parameter '%s'", params[i].Name)
		}
	}
	return result[:end], nil
}
