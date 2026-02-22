package compiler

import (
	"encoding/binary"
	"fmt"
	"math"

	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/value"
)

type SymbolInfo struct {
	SlotID  int
	Tag     value.TypeTag
	Mask    byte
	Stencil *Stencil // non-nil for struct/tuple variables
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

func (st *SymbolTable) Remove(name string) {
	delete(st.symbols, name)
}

type Compiler struct {
	scope    *SymbolTable // nil outside alloc blocks
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
		name := s.Name.Value

		// Check if RHS is a struct literal expression
		if structExpr, ok := s.Value.(*parser.StructExpression); ok {
			return c.compileStructAssignment(b, s, structExpr)
		}

		// Check if RHS is a tuple literal expression
		if tupleExpr, ok := s.Value.(*parser.TupleExpression); ok {
			return c.compileTupleAssignment(b, s, tupleExpr)
		}

		// Check if RHS is a pointer alias expression
		if ptrExpr, ok := s.Value.(*parser.PointerExpression); ok {
			// Compile the offset expression (pushes offset onto expr stack)
			if err := c.compileExpression(b, ptrExpr.Offset); err != nil {
				return fmt.Errorf("failed to compile pointer offset: %v", err)
			}

			tag, ok := value.TagForName(ptrExpr.TypeName)
			if !ok {
				return fmt.Errorf("unknown type name '%s' in pointer", ptrExpr.TypeName)
			}

			if _, exists := c.scope.Lookup(name); !exists {
				mask := value.MaskForTag(tag)
				c.scope.Define(name, tag, mask)
			}

			info, _ := c.scope.Lookup(name)
			b.EmitArgExtra(OpVarPTR, info.SlotID, byte(tag), s.Position().Line)
			return nil
		}

		// Compile RHS expression (pushes value onto expression stack)
		if err := c.compileExpression(b, s.Value); err != nil {
			return fmt.Errorf("failed to compile assignment value: %v", err)
		}

		// Detect string assignments — strings use dynamic allocation via
		// OpStrSTORE, bypassing the mask-based OpVarALLOC path entirely.
		rhsTag, _ := c.inferTypeTag(s.Value)
		if rhsTag == value.TagString {
			if _, exists := c.scope.Lookup(name); !exists {
				c.scope.Define(name, value.TagString, 0)
			}
			info, _ := c.scope.Lookup(name)
			b.EmitArg(OpStrSTORE, info.SlotID, s.Position().Line)
			return nil
		}

		if _, exists := c.scope.Lookup(name); !exists {
			var mask byte
			var inferredTag value.TypeTag

			if len(s.Constraints) > 0 {
				// Typed assignment — resolve constraint mask
				var err error
				mask, err = c.resolveConstraintMask(s.Constraints)
				if err != nil {
					return fmt.Errorf("type constraint for '%s': %v", name, err)
				}
				// Infer the initial tag from the RHS for the symbol table
				inferredTag, _ = c.inferTypeTag(s.Value)
			} else {
				// Untyped assignment — infer type and build single-type mask
				tag, err := c.inferTypeTag(s.Value)
				if err != nil {
					return fmt.Errorf("cannot infer type for '%s': %v", name, err)
				}
				inferredTag = tag
				mask = value.MaskForTag(tag)
			}

			info := c.scope.Define(name, inferredTag, mask)
			b.EmitArgExtra(OpVarALLOC, info.SlotID, mask, s.Position().Line)
		}

		info, _ := c.scope.Lookup(name)
		b.EmitArg(OpVarSTORE, info.SlotID, s.Position().Line)

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
			tag, ok := value.TagForName(f.Type)
			if !ok {
				return fmt.Errorf("struct '%s': unknown type '%s' for field '%s'", s.Name, f.Type, f.Name)
			}
			stencil.Fields = append(stencil.Fields, FieldLayout{
				Name:   f.Name,
				Offset: offset,
				Tag:    tag,
			})
			offset += value.SizeForTag(tag)
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
			Functions:    b.Functions,
		}

		// Enter function scope — parameters become variables in this scope.
		oldScope := c.scope
		c.scope = newSymbolTable()

		// For each parameter: pull from the pending-args buffer, allocate a slot,
		// and store the value.  Primitive and struct params use different sequences.
		for i, param := range params {
			if param.Stencil != nil {
				// Struct parameter: allocate a stencil slot and bulk-copy the raw bytes.
				// OpLoadArgStencil: Argument=pending index, Extra=slot ID, Offset=total size.
				info := c.scope.Define(param.Name, 0, 0)
				sym := c.scope.symbols[param.Name]
				sym.Stencil = param.Stencil
				c.scope.symbols[param.Name] = sym
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

		// Restore the outer scope and register the compiled function.
		c.scope = oldScope
		b.Functions[s.Name.Value] = &FunctionDef{
			Name:     s.Name.Value,
			ByteCode: fnCode,
			Params:   params,
		}

	case *parser.ReturnStatement:
		if s.Value != nil {
			if err := c.compileExpression(b, s.Value); err != nil {
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
			if err := c.compileExpression(b, arg); err != nil {
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

func (c *Compiler) compileStructAssignment(b *ByteCode, s *parser.AssignmentStatement, structExpr *parser.StructExpression) error {
	stencil, ok := c.stencils[structExpr.Name]
	if !ok {
		return fmt.Errorf("undefined struct type '%s'", structExpr.Name)
	}

	name := s.Name.Value

	if _, exists := c.scope.Lookup(name); !exists {
		info := c.scope.Define(name, 0, 0)
		// Store stencil reference in the symbol
		sym := c.scope.symbols[name]
		sym.Stencil = stencil
		c.scope.symbols[name] = sym
		// Emit stencil alloc: Argument=slotID, Offset=totalSize
		b.EmitField(OpStencilALLOC, info.SlotID, stencil.TotalSize, 0, s.Position().Line)
	}

	info, _ := c.scope.Lookup(name)

	// Compile and store each field
	for _, fieldName := range structExpr.Order {
		fieldExpr := structExpr.Fields[fieldName]
		field, ok := stencil.LookupField(fieldName)
		if !ok {
			return fmt.Errorf("struct '%s' has no field '%s'", structExpr.Name, fieldName)
		}

		if err := c.compileExpression(b, fieldExpr); err != nil {
			return fmt.Errorf("failed to compile struct field '%s': %v", fieldName, err)
		}

		b.EmitField(OpFieldSTORE, info.SlotID, field.Offset, byte(field.Tag), s.Position().Line)
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

	name := s.Name.Value

	if _, exists := c.scope.Lookup(name); !exists {
		info := c.scope.Define(name, 0, 0)
		sym := c.scope.symbols[name]
		sym.Stencil = stencil
		c.scope.symbols[name] = sym
		b.EmitField(OpStencilALLOC, info.SlotID, stencil.TotalSize, 0, s.Position().Line)
	}

	info, _ := c.scope.Lookup(name)

	// Compile and store each element
	for i, elem := range tupleExpr.Elements {
		if err := c.compileExpression(b, elem); err != nil {
			return fmt.Errorf("failed to compile tuple element %d: %v", i, err)
		}
		field := stencil.Fields[i]
		b.EmitField(OpFieldSTORE, info.SlotID, field.Offset, byte(field.Tag), s.Position().Line)
	}

	return nil
}

func (c *Compiler) compileExpression(b *ByteCode, expr parser.Expression) error {
	switch e := expr.(type) {
	case *parser.ByteExpression:
		data := []byte{e.Value}
		constIdx := b.AddConstant(Constant{Tag: value.TagByte, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.ShortExpression:
		data := make([]byte, 2)
		binary.LittleEndian.PutUint16(data, uint16(e.Value))
		constIdx := b.AddConstant(Constant{Tag: value.TagShort, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.IntegerExpression:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(e.Value))
		constIdx := b.AddConstant(Constant{Tag: value.TagInteger, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.LongExpression:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, uint64(e.Value))
		constIdx := b.AddConstant(Constant{Tag: value.TagLong, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.FloatExpression:
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, math.Float32bits(e.Value))
		constIdx := b.AddConstant(Constant{Tag: value.TagFloat, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.DecimalExpression:
		data := make([]byte, 8)
		binary.LittleEndian.PutUint64(data, math.Float64bits(e.Value))
		constIdx := b.AddConstant(Constant{Tag: value.TagDecimal, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.CharExpression:
		// char is not a distinct allocable type; lower the rune literal to int32.
		data := make([]byte, 4)
		binary.LittleEndian.PutUint32(data, uint32(e.Value))
		constIdx := b.AddConstant(Constant{Tag: value.TagInteger, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.BooleanExpression:
		data := []byte{0}
		if e.Value {
			data[0] = 1
		}
		constIdx := b.AddConstant(Constant{Tag: value.TagBoolean, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.StringExpression:
		data := []byte(e.Value)
		constIdx := b.AddConstant(Constant{Tag: value.TagString, Data: data})
		b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
	case *parser.NilExpression:
		return fmt.Errorf("nil literals are not allocable")
	case *parser.IdentifierExpression:
		info, exists := c.scope.Lookup(e.Value)
		if !exists {
			return fmt.Errorf("undefined variable '%s'", e.Value)
		}
		if info.Stencil != nil {
			// Struct variable — push a snapshot of its raw bytes onto the expr stack
			// so it can be passed as a function argument via OpCallFN.
			b.EmitField(OpVarLoadRaw, info.SlotID, info.Stencil.TotalSize, 0, e.Position().Line)
		} else {
			b.EmitArg(OpVarLOAD, info.SlotID, e.Position().Line)
		}
	case *parser.PointerExpression:
		// Inline pointer dereference used as an expression: *type(offset)
		// Compile the offset, then emit OpPtrLOAD so the VM reads directly
		// from the allocator at that offset and pushes the value.
		tag, ok := value.TagForName(e.TypeName)
		if !ok {
			return fmt.Errorf("unknown type name '%s' in pointer expression", e.TypeName)
		}
		if err := c.compileExpression(b, e.Offset); err != nil {
			return fmt.Errorf("failed to compile pointer offset: %v", err)
		}
		b.EmitArgExtra(OpPtrLOAD, 0, byte(tag), e.Position().Line)
	case *parser.AttributeExpression:
		// Field access on a struct/tuple: obj.field or obj.0
		ident, ok := e.Object.(*parser.IdentifierExpression)
		if !ok {
			return fmt.Errorf("field access requires an identifier, got %T", e.Object)
		}
		info, exists := c.scope.Lookup(ident.Value)
		if !exists {
			return fmt.Errorf("undefined variable '%s'", ident.Value)
		}
		if info.Stencil == nil {
			return fmt.Errorf("variable '%s' is not a struct or tuple", ident.Value)
		}
		fieldName := e.Attribute.Value
		field, ok := info.Stencil.LookupField(fieldName)
		if !ok {
			return fmt.Errorf("struct '%s' has no field '%s'", info.Stencil.Name, fieldName)
		}
		b.EmitField(OpFieldLOAD, info.SlotID, field.Offset, byte(field.Tag), e.Position().Line)
	default:
		return fmt.Errorf("unknown expression type: %T", e)
	}
	return nil
}

func (c *Compiler) resolveConstraintMask(constraints []parser.Expression) (byte, error) {
	var mask byte
	for _, constraint := range constraints {
		ident, ok := constraint.(*parser.IdentifierExpression)
		if !ok {
			return 0, fmt.Errorf("type constraint must be an identifier, got %T", constraint)
		}
		tag, ok := value.TagForName(ident.Value)
		if !ok {
			return 0, fmt.Errorf("unknown type name '%s'", ident.Value)
		}
		mask |= value.MaskForTag(tag)
	}
	return mask, nil
}

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
	case *parser.CharExpression:
		return value.TagInteger, nil // char literals are lowered to int32
	case *parser.StringExpression:
		return value.TagString, nil
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
