package compiler

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/value"
)

type SymbolInfo struct {
	SlotID   int
	Tag      value.TypeTag
	Mask     byte
	Capacity int      // byte capacity for TagSlice variables (0 for scalar types)
	Stencil  *Stencil // non-nil for struct/tuple variables
}

type SymbolTable struct {
	symbols  map[string]SymbolInfo
	nextSlot int
}

func newSymbolTable() *SymbolTable {
	return &SymbolTable{
		symbols:  make(map[string]SymbolInfo),
		nextSlot: 0,
	}
}

func (st *SymbolTable) Lookup(name string) (SymbolInfo, bool) {
	info, ok := st.symbols[name]
	return info, ok
}

func (st *SymbolTable) Define(name string, tag value.TypeTag, mask byte) SymbolInfo {
	info := SymbolInfo{
		SlotID: st.nextSlot,
		Tag:    tag,
		Mask:   mask,
	}
	st.symbols[name] = info
	st.nextSlot++
	return info
}

func (st *SymbolTable) DefineSlice(name string, capacity int) SymbolInfo {
	info := SymbolInfo{
		SlotID:   st.nextSlot,
		Tag:      value.TagSlice,
		Mask:     value.MaskForTag(value.TagSlice),
		Capacity: capacity,
	}
	st.symbols[name] = info
	st.nextSlot++
	return info
}

func (st *SymbolTable) DefineStencil(name string, stencil *Stencil) SymbolInfo {
	info := SymbolInfo{SlotID: st.nextSlot, Stencil: stencil}
	st.symbols[name] = info
	st.nextSlot++
	return info
}

func (st *SymbolTable) Remove(name string) {
	delete(st.symbols, name)
}

// Preallocated boolean constants — shared across all compilations to avoid
// per-literal []byte allocations (AddConstant deduplicates by value).
var (
	boolConstTrue  = Constant{Tag: value.TagBoolean, Data: []byte{1}}
	boolConstFalse = Constant{Tag: value.TagBoolean, Data: []byte{0}}
)

type Compiler struct {
	scope    *SymbolTable
	stencils map[string]*Stencil
}

func NewCompiler() *Compiler {
	return &Compiler{
		stencils: make(map[string]*Stencil),
	}
}

func (c *Compiler) Compile(ast parser.AST) (*ByteCode, error) {
	byteCode := &ByteCode{
		Instructions: make([]Instruction, 0),
		Constants:    make([]Constant, 0),
		Names:        make([]string, 0),
		LoopStack:    nil,
		Functions:    make(map[string]*FunctionDef),
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
		stencil := &Stencil{
			Name:   s.Name,
			Fields: make([]FieldLayout, 0, len(s.Fields)),
		}
		offset := 0
		for _, f := range s.Fields {
			if f.Capacity > 0 {
				// Parameterised slice field: string<N> or byte<N>.
				stencil.Fields = append(stencil.Fields, FieldLayout{
					Name:     f.Name,
					Offset:   offset,
					Tag:      value.TagSlice,
					Capacity: f.Capacity,
				})
				offset += f.Capacity
			} else {
				tag, ok := value.TagForName(f.Type)
				if !ok {
					return fmt.Errorf("struct '%s': unknown type '%s' for field '%s'", s.Name, f.Type, f.Name)
				}
				size := value.SizeForTag(tag)
				if size == 0 {
					return fmt.Errorf("struct '%s': type '%s' for field '%s' requires an explicit capacity (use %s<N>)", s.Name, f.Type, f.Name, f.Type)
				}
				stencil.Fields = append(stencil.Fields, FieldLayout{
					Name:   f.Name,
					Offset: offset,
					Tag:    tag,
				})
				offset += size
			}
		}
		stencil.TotalSize = offset
		c.stencils[s.Name] = stencil

	case *parser.FunctionStatement:
		params := make([]ParamDef, len(s.Parameters))

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
			if tag, ok := value.TagForName(ident.Value); ok {
				// Primitive type parameter (int, long, float, …)
				mask := value.MaskForTag(tag)
				params[i] = ParamDef{Name: p.Value, Tag: tag, Mask: mask}
			} else if stencil, ok := c.stencils[ident.Value]; ok {
				// Struct type parameter
				params[i] = ParamDef{Name: p.Value, Stencil: stencil}
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
		oldScope := c.scope
		c.scope = newSymbolTable()

		// For each parameter: pull from the pending-args buffer, allocate a slot,
		// and store the value. Primitive and struct params use different sequences.
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
				frameSize += value.MaxSizeForMask(instr.Extra)
			case OpSliceALLOC, OpStencilALLOC, OpLoadArgStencil:
				frameSize += instr.Offset
			}
		}

		// Restore the outer scope and register the compiled function.
		c.scope = oldScope
		b.Functions[s.Name.Value] = &FunctionDef{
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
		ident, ok := s.Function.(*parser.IdentifierExpression)
		if !ok {
			return fmt.Errorf("only identifier function calls are supported, got %T", s.Function)
		}
		for i, arg := range s.Arguments {
			if _, err := c.compileExpression(b, arg); err != nil {
				return fmt.Errorf("argument %d of call to '%s': %v", i, ident.Value, err)
			}
		}
		// Prefer user-defined functions over native ones at compile time.
		if _, isUserFunc := b.Functions[ident.Value]; isUserFunc {
			b.EmitNameArg(OpCallFN, ident.Value, len(s.Arguments), s.Position().Line)
		} else {
			b.EmitNameArg(OpCallNAT, ident.Value, len(s.Arguments), s.Position().Line)
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

	tag, ok := value.TagForName(ptrExpr.TypeName)
	if !ok {
		return fmt.Errorf("unknown type name '%s' in pointer", ptrExpr.TypeName)
	}

	name := s.Name.Value
	if _, exists := c.scope.Lookup(name); !exists {
		mask := value.MaskForTag(tag)
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
		c.emitConst(b, value.TagSlice, []byte(strExpr.Value), s.Position().Line)
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
	if rhsTag == value.TagSlice {
		return fmt.Errorf("string assignment to '%s' requires an explicit capacity: use `%s: string<N> = ...`", name, name)
	}

	if _, exists := c.scope.Lookup(name); !exists {
		var mask byte
		var inferredTag value.TypeTag

		if len(s.Constraints) > 0 {
			mask, err = c.resolveConstraintMask(s.Constraints)
			if err != nil {
				return fmt.Errorf("type constraint for '%s': %v", name, err)
			}
			inferredTag = rhsTag
		} else {
			inferredTag = rhsTag
			mask = value.MaskForTag(rhsTag)
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
	if rhsTag != value.TagSlice {
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

		if field.Tag == value.TagSlice {
			b.EmitFieldSize(OpFieldSTORE, info.SlotID, field.Offset, byte(field.Tag), field.Capacity, s.Position().Line)
		} else {
			b.EmitField(OpFieldSTORE, info.SlotID, field.Offset, byte(field.Tag), s.Position().Line)
		}
	}

	return nil
}

func (c *Compiler) compileTupleAssignment(b *ByteCode, s *parser.AssignmentStatement, tupleExpr *parser.TupleExpression) error {
	// Build an anonymous stencil from the element types
	stencil := &Stencil{
		Name:   "",
		Fields: make([]FieldLayout, 0, len(tupleExpr.Elements)),
	}
	offset := 0
	for i, elem := range tupleExpr.Elements {
		tag, err := c.inferTypeTag(elem)
		if err != nil {
			return fmt.Errorf("tuple element %d: %v", i, err)
		}
		stencil.Fields = append(stencil.Fields, FieldLayout{
			Name:   fmt.Sprintf("%d", i),
			Offset: offset,
			Tag:    tag,
		})
		offset += value.SizeForTag(tag)
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
func (c *Compiler) emitStencilInit(b *ByteCode, name string, stencil *Stencil, line int) SymbolInfo {
	if _, exists := c.scope.Lookup(name); !exists {
		info := c.scope.DefineStencil(name, stencil)
		b.EmitField(OpStencilALLOC, info.SlotID, stencil.TotalSize, 0, line)
	}
	info, _ := c.scope.Lookup(name)
	return info
}

// emitConst adds a typed constant to the pool and emits OpLoadCONST.
func (c *Compiler) emitConst(b *ByteCode, tag value.TypeTag, data []byte, line int) {
	idx := b.AddConstant(Constant{Tag: tag, Data: data})
	b.EmitArg(OpLoadCONST, idx, line)
}

func (c *Compiler) compileExpression(b *ByteCode, expr parser.Expression) (value.TypeTag, error) {
	switch e := expr.(type) {
	case *parser.ByteExpression:
		c.emitConst(b, value.TagByte, []byte{e.Value}, e.Position().Line)
		return value.TagByte, nil
	case *parser.ShortExpression:
		data := make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(e.Value))
		c.emitConst(b, value.TagShort, data, e.Position().Line)
		return value.TagShort, nil
	case *parser.IntegerExpression:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(e.Value))
		c.emitConst(b, value.TagInteger, data, e.Position().Line)
		return value.TagInteger, nil
	case *parser.LongExpression:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(e.Value))
		c.emitConst(b, value.TagLong, data, e.Position().Line)
		return value.TagLong, nil
	case *parser.FloatExpression:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, math.Float32bits(e.Value))
		c.emitConst(b, value.TagFloat, data, e.Position().Line)
		return value.TagFloat, nil
	case *parser.DecimalExpression:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, math.Float64bits(e.Value))
		c.emitConst(b, value.TagDecimal, data, e.Position().Line)
		return value.TagDecimal, nil
	case *parser.BooleanExpression:
		bc := boolConstFalse
		if e.Value {
			bc = boolConstTrue
		}
		c.emitConst(b, bc.Tag, bc.Data, e.Position().Line)
		return value.TagBoolean, nil
	case *parser.StringExpression:
		c.emitConst(b, value.TagSlice, []byte(e.Value), e.Position().Line)
		return value.TagSlice, nil
	case *parser.InterpolatedExpression:
		for _, part := range e.Parts {
			if _, err := c.compileExpression(b, part); err != nil {
				return 0, fmt.Errorf("interpolated string part: %w", err)
			}
		}
		b.EmitArg(OpBuildSTRING, len(e.Parts), e.Position().Line)
		return value.TagSlice, nil
	case *parser.NilExpression:
		return 0, fmt.Errorf("nil literals are not allocable")
	case *parser.IdentifierExpression:
		info, exists := c.scope.Lookup(e.Value)
		if !exists {
			return 0, fmt.Errorf("undefined variable '%s'", e.Value)
		}
		if info.Stencil != nil {
			// Struct variable — push a snapshot of its raw bytes onto the expr stack
			// so it can be passed as a function argument via OpCallFN.
			b.EmitField(OpVarLoadRaw, info.SlotID, info.Stencil.TotalSize, 0, e.Position().Line)
		} else {
			b.EmitArg(OpVarLOAD, info.SlotID, e.Position().Line)
		}
		return info.Tag, nil
	case *parser.PointerExpression:
		// Inline pointer dereference used as an expression: *type(offset)
		// Compile the offset, then emit OpPtrLOAD so the VM reads directly
		// from the allocator at that offset and pushes the value.
		tag, ok := value.TagForName(e.TypeName)
		if !ok {
			return 0, fmt.Errorf("unknown type name '%s' in pointer expression", e.TypeName)
		}
		if _, err := c.compileExpression(b, e.Offset); err != nil {
			return 0, fmt.Errorf("failed to compile pointer offset: %v", err)
		}
		b.EmitArgExtra(OpPtrLOAD, 0, byte(tag), e.Position().Line)
		return tag, nil
	case *parser.AttributeExpression:
		// Field access on a struct/tuple: obj.field or obj.0
		ident, ok := e.Object.(*parser.IdentifierExpression)
		if !ok {
			return 0, fmt.Errorf("field access requires an identifier, got %T", e.Object)
		}
		info, exists := c.scope.Lookup(ident.Value)
		if !exists {
			return 0, fmt.Errorf("undefined variable '%s'", ident.Value)
		}
		if info.Stencil == nil {
			return 0, fmt.Errorf("variable '%s' is not a struct or tuple", ident.Value)
		}
		fieldName := e.Attribute.Value
		field, ok := info.Stencil.LookupField(fieldName)
		if !ok {
			return 0, fmt.Errorf("struct '%s' has no field '%s'", info.Stencil.Name, fieldName)
		}
		if field.Tag == value.TagSlice {
			b.EmitFieldSize(OpFieldLOAD, info.SlotID, field.Offset, byte(field.Tag), field.Capacity, e.Position().Line)
		} else {
			b.EmitField(OpFieldLOAD, info.SlotID, field.Offset, byte(field.Tag), e.Position().Line)
		}
		return field.Tag, nil
	default:
		return 0, fmt.Errorf("unknown expression type: %T", e)
	}
}

func (c *Compiler) resolveConstraintMask(constraints []parser.Expression) (byte, error) {
	var mask byte
	for _, constraint := range constraints {
		switch ct := constraint.(type) {
		case *parser.IdentifierExpression:
			tag, ok := value.TagForName(ct.Value)
			if !ok {
				return 0, fmt.Errorf("unknown type name '%s'", ct.Value)
			}
			mask |= value.MaskForTag(tag)
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
func (c *Compiler) inferTypeTag(expr parser.Expression) (value.TypeTag, error) {
	switch expr := expr.(type) {
	case *parser.ByteExpression:
		return value.TagByte, nil
	case *parser.ShortExpression:
		return value.TagShort, nil
	case *parser.IntegerExpression:
		return value.TagInteger, nil
	case *parser.LongExpression:
		return value.TagLong, nil
	case *parser.FloatExpression:
		return value.TagFloat, nil
	case *parser.DecimalExpression:
		return value.TagDecimal, nil
	case *parser.BooleanExpression:
		return value.TagBoolean, nil
	case *parser.StringExpression:
		return value.TagSlice, nil // string literals are slice-typed; capacity must be declared
	case *parser.InterpolatedExpression:
		return value.TagSlice, nil
	case *parser.PointerExpression:
		tag, ok := value.TagForName(expr.TypeName)
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
	default:
		return 0, fmt.Errorf("cannot infer type from expression %T", expr)
	}
}
