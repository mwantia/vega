 Plan: Add byte and char Primitive Types

 Context

 The byte-array memory model is implemented with 6 allocable primitives (short, int, long, float, decimal, bool). We decided on a final budget of 8 primitive types to fit a bitmask byte for future union type support. This plan adds the last two: byte (uint8, 1 byte) and char (rune, 4 bytes).

 Literal syntax:
 - byte: 42b (suffix) and 0xFF (hex prefix). Range 0–255.
 - char: 'A' (single quotes, like C/Go/Rust).

 ---
 Step 1: Value types

 File: new/pkg/value/value.go

 Add ByteValue and CharValue following the existing pattern:

 // ByteValue wraps uint8. Size = 1 byte.
 type ByteValue struct {
     data   byte
     offset byte
 }
 func NewByte(v byte) *ByteValue
 func (v *ByteValue) Type() string   { return "byte" }
 func (v *ByteValue) String() string { return strconv.Itoa(int(v.data)) }
 func (v *ByteValue) Size() byte     { return 1 }
 func (v *ByteValue) Offset() byte   { return v.offset }
 func (v *ByteValue) Data() byte     { return v.data }
 var _ Allocable = (*ByteValue)(nil)

 // CharValue wraps rune (int32). Size = 4 bytes.
 type CharValue struct {
     data   rune
     offset byte
 }
 func NewChar(v rune) *CharValue
 func (v *CharValue) Type() string   { return "char" }
 func (v *CharValue) String() string { return string(v.data) }
 func (v *CharValue) Size() byte     { return 4 }
 func (v *CharValue) Offset() byte   { return v.offset }
 func (v *CharValue) Data() rune     { return v.data }
 var _ Allocable = (*CharValue)(nil)

 Step 2: Type tags

 File: new/pkg/value/typetag.go

 Add two new tags and update TagFor and SizeForTag:

 TagByte TypeTag = 7  // uint8, 1 byte
 TagChar TypeTag = 8  // rune, 4 bytes

 - TagFor: add *ByteValue → TagByte, *CharValue → TagChar
 - SizeForTag: add TagByte → 1, TagChar → 4

 Step 3: Encoding

File: new/pkg/value/encoding.go

 Add cases to Encode and Decode:

 - ByteValue: dst[0] = v.data / NewByte(src[0]) — single byte, no endianness needed
 - CharValue: binary.LittleEndian.PutUint32(dst, uint32(v.data)) / NewChar(rune(binary.LittleEndian.Uint32(src))) — same pattern as IntegerValue

 Step 4: Lexer

 File: new/pkg/lexer/token.go

 - Add BYTE TokenType = "BYTE" and CHAR TokenType = "CHAR" constants
 - Add BYTE and CHAR to IsLiteral()

 File: new/pkg/lexer/helpers.go

 - Add IsHexDigit(ch rune) bool — returns true for 0-9, a-f, A-F

 File: new/pkg/lexer/lexer.go

 Two changes:

 1. readNumberToken — add 'b' suffix case (same pattern as 's'/'l'):
 case 'b':
     l.ReadChar()
     return Token{Type: BYTE, Literal: literal, Position: pos}
 2. Hex literals — at the start of readNumberToken, check if the first digit is 0 and peek is x/X. If so, consume 0x, read hex digits, always emit BYTE token:
 if l.current == '0' && (l.PeekChar() == 'x' || l.PeekChar() == 'X') {
     l.ReadChar() // consume '0'
     l.ReadChar() // consume 'x'
     startPos = l.position
     for IsHexDigit(l.current) { l.ReadChar() }
     literal := l.text[startPos:l.position]
     return Token{Type: BYTE, Literal: "0x" + literal, Position: pos}
 }
 2. The literal stores the full 0xFF form for later parsing.
 3. Char literals — add case '\'' in Next(), implement ReadCharToken:
   - Consume opening '
   - Read one character (handle \n, \t, \\, \' escapes)
   - Expect closing '
   - Return Token{Type: CHAR, Literal: string(ch)}

 Step 5: Parser

 File: new/pkg/parser/expression.go

 Add two new AST nodes:

 type ByteExpression struct {
     Token lexer.Token
     Value byte
 }
// Expression interface methods + String() returns literal + "b"

 type CharExpression struct {
     Token lexer.Token
     Value rune
 }
 // Expression interface methods + String() returns "'" + string(Value) + "'"

 File: new/pkg/parser/parser.go

 Add two cases in makePrefixExpression:

 - case lexer.BYTE: Parse literal. If starts with 0x, use strconv.ParseUint(literal, 0, 8). Otherwise strconv.ParseUint(literal, 10, 8). Return ByteExpression.
 - case lexer.CHAR: The literal is already the character string from the lexer. Get the rune with utf8.DecodeRuneInString. Return CharExpression.

 Step 6: Compiler

 File: new/pkg/compiler/compiler.go

 Add to compileExpression:
 case *parser.ByteExpression:
     constIdx := b.AddConstant(value.NewByte(e.Value))
     b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)
 case *parser.CharExpression:
     constIdx := b.AddConstant(value.NewChar(e.Value))
     b.EmitArg(OpLoadCONST, constIdx, e.Position().Line)

 Add to inferTypeTag:
 case *parser.ByteExpression:
     return value.TagByte, nil
 case *parser.CharExpression:
     return value.TagChar, nil

 Step 7: Tests

 File: new/pkg/value/encoding_test.go

 Add to TestEncodeDecodeRoundTrip:
 {"byte zero", value.NewByte(0), value.TagByte, "0"},
 {"byte max", value.NewByte(255), value.TagByte, "255"},
 {"byte mid", value.NewByte(42), value.TagByte, "42"},
 {"char ascii", value.NewChar('A'), value.TagChar, "A"},
 {"char unicode", value.NewChar('€'), value.TagChar, "€"},
 {"char emoji", value.NewChar('🎉'), value.TagChar, "🎉"},

 Add to TestTagFor and TestSizeForTag.

 File: new/pkg/compiler/compiler_test.go

 Add to Factories:
 "literal-byte":    func() string { return `alloc 64 { 42b }` },
 "literal-byte-hex": func() string { return `alloc 64 { 0xFF }` },
 "literal-char":    func() string { return `alloc 64 { 'A' }` },
 "assign-byte":     func() string { return `alloc 64 { x = 42b; x }` },
 "assign-char":     func() string { return `alloc 64 { x = 'A'; x }` },

 ---
 Files Modified (all under new/pkg/)

 ┌───────────────────────────┬───────────────────────────────────────────────────────────┐
 │           File            │                          Change                           │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ value/value.go            │ Add ByteValue, CharValue                                  │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ value/typetag.go          │ Add TagByte=7, TagChar=8, update TagFor/SizeForTag        │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ value/encoding.go         │ Add encode/decode cases for byte and char                 │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ value/encoding_test.go    │ Add round-trip tests                                      │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ lexer/token.go            │ Add BYTE, CHAR tokens, update IsLiteral                   │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ lexer/helpers.go          │ Add IsHexDigit                                            │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ lexer/lexer.go            │ Add b suffix, hex 0x prefix, single-quote ' char literals │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ parser/expression.go      │ Add ByteExpression, CharExpression                        │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ parser/parser.go          │ Add cases in makePrefixExpression                         │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ compiler/compiler.go      │ Add cases in compileExpression and inferTypeTag           │
 ├───────────────────────────┼───────────────────────────────────────────────────────────┤
 │ compiler/compiler_test.go │ Add literal and assignment tests                          │
 └───────────────────────────┴───────────────────────────────────────────────────────────┘

 Verification

 # Value encoding round-trips
 go test -v ./new/pkg/value/...

 # Full compiler pipeline (all existing + new tests)
 go test -v ./new/pkg/compiler/...

 # Full suite
 go test ./new/...

 All existing tests must continue to pass unchanged.