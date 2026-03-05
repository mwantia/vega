package compiler

import (
	"bytes"
	"compress/zlib"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/mwantia/vega/pkg/slot"
)

// .vgc binary format constants.
//
// Layout:
//
//	HEADER (16 bytes):
//	  [0:4]  magic:      0x56 0x45 0x47 0x41  ("VEGA", little-endian uint32)
//	  [4:6]  version:    uint16 LE  (increment on breaking bytecode changes)
//	  [6:8]  flags:      uint16 LE  (bit 0: debug section present, bit 1: stencils section present, bit 2: payload compressed with zlib)
//	  [8:12] sourceCRC:  uint32 LE  (CRC32 of source .vega file, or 0 if unknown)
//	  [12:16] totalSize: uint32 LE  (total file length for basic integrity check)
//
//	SECTION TABLE (immediately after header):
//	  uint16  section count
//	  per section:
//	    byte    section ID
//	    uint32  offset from start of file
//	    uint32  byte length
//
//	SECTION 0 — CONSTANTS:
//	  uint32  count
//	  per constant:
//	    byte    TypeTag
//	    uint16  data length
//	    N bytes raw value (little-endian, same encoding as slot.StackSlot.Data)
//
//	SECTION 1 — NAMES (interned string table):
//	  uint32  count
//	  per name:
//	    uint16  byte length
//	    N bytes UTF-8
//
//	SECTION 2 — INSTRUCTIONS (fixed 10 bytes each):
//	  uint32  count
//	  per instruction:
//	    byte    OperationCode
//	    [3]byte Argument  (unsigned int24 LE; max 16 777 215)
//	    [3]byte Offset    (unsigned int24 LE)
//	    byte    Extra
//	    uint16  Size
//
//	SECTION 3 — DEBUG INFO (present when flags bit 0 is set):
//	  uint32  count  (must equal instruction count)
//	  per instruction:
//	    uint16  source line number
//
//	SECTION 4 — FUNCTIONS:
//	  uint32  count
//	  per function:
//	    uint16  name byte length + N bytes UTF-8
//	    uint32  FrameSize
//	    byte    param count
//	    per param:
//	      uint16  name byte length + N bytes UTF-8
//	      uint16  SlotID
//	      byte    TypeTag
//	      byte    Mask
//	      byte    has_stencil (0 = no, 1 = yes)
//	      if has_stencil:
//	        uint16  stencil name byte length + N bytes UTF-8
//	    embedded ByteCode (length-prefixed sub-sections):
//	      uint32  constants payload length + payload
//	      uint32  names payload length    + payload
//	      uint32  instrs payload length   + payload
//	      if debug: uint32 debug payload length + payload
//
//	SECTION 5 — STENCILS (present when flags bit 1 is set):
//	  uint32  count
//	  per stencil:
//	    uint16  name byte length + N bytes UTF-8
//	    uint32  TotalSize
//	    byte    field count
//	    per field:
//	      uint16  name byte length + N bytes UTF-8
//	      uint32  Offset
//	      byte    TypeTag
//	      uint32  Capacity
const (
	vgcMagic   = uint32(0x41474556) // little-endian "VEGA"
	vgcVersion = uint16(1)

	vgcFlagDebug      = uint16(1 << 0)
	vgcFlagStencils   = uint16(1 << 1)
	vgcFlagCompressed = uint16(1 << 2) // everything after the 16-byte header is a zlib stream

	vgcSectConstants    = byte(0)
	vgcSectNames        = byte(1)
	vgcSectInstructions = byte(2)
	vgcSectDebug        = byte(3)
	vgcSectFunctions    = byte(4)
	vgcSectStencils     = byte(5)
)

func HasValidMagic(buf []byte) bool {
	if len(buf) < 4 {
		return false
	}

	magic := binary.LittleEndian.Uint32(buf)
	return magic == vgcMagic
}

// Serialize encodes b into an uncompressed .vgc byte slice.
func (b *ByteCode) Serialize(debug, compress bool) ([]byte, error) {
	constData, err := encodeConstants(b.Constants)
	if err != nil {
		return nil, fmt.Errorf("serialize constants: %w", err)
	}
	namesData, err := encodeNames(b.Names)
	if err != nil {
		return nil, fmt.Errorf("serialize names: %w", err)
	}
	instrData, err := encodeInstructions(b.Instructions)
	if err != nil {
		return nil, fmt.Errorf("serialize instructions: %w", err)
	}

	stencils := collectStencils(b)
	stencData, err := encodeStencils(stencils)
	if err != nil {
		return nil, fmt.Errorf("serialize stencils: %w", err)
	}
	funcData, err := encodeFunctions(b, debug)
	if err != nil {
		return nil, fmt.Errorf("serialize functions: %w", err)
	}

	flags := uint16(0)
	if debug {
		flags |= vgcFlagDebug
	}
	if len(stencils) > 0 {
		flags |= vgcFlagStencils
	}
	if compress {
		flags |= vgcFlagCompressed
	}

	type section struct {
		id   byte
		data []byte
	}
	sects := []section{
		{vgcSectConstants, constData},
		{vgcSectNames, namesData},
		{vgcSectInstructions, instrData},
	}
	if debug {
		dbgData, err := encodeDebug(b.Instructions)
		if err != nil {
			return nil, fmt.Errorf("serialize debug: %w", err)
		}
		sects = append(sects, section{vgcSectDebug, dbgData})
	}
	sects = append(sects, section{vgcSectFunctions, funcData})
	if len(stencils) > 0 {
		sects = append(sects, section{vgcSectStencils, stencData})
	}

	// Header: 16 bytes. Section table: 2 + 9*N bytes (uint16 + N*(1+4+4)).
	headerSz := 16
	tableSz := 2 + 9*len(sects)
	dataStart := uint32(headerSz + tableSz)

	var w vgcWriter
	// Header.
	w.writeU32(vgcMagic)
	w.writeU16(vgcVersion)
	w.writeU16(flags)
	w.writeU32(0) // sourceCRC — unknown at compile time; caller may patch [8:12] if desired
	w.writeU32(0) // totalSize — patched below

	// Section table.
	w.writeU16(uint16(len(sects)))
	offset := dataStart
	for _, s := range sects {
		w.writeByte(s.id)
		w.writeU32(offset)
		w.writeU32(uint32(len(s.data)))
		offset += uint32(len(s.data))
	}

	// Section payloads.
	for _, s := range sects {
		w.Write(s.data)
	}

	result := w.Bytes()

	if compress {
		// Compress everything after the 16-byte header as a single zlib stream.
		var zbuf bytes.Buffer
		zw, _ := zlib.NewWriterLevel(&zbuf, zlib.BestCompression)
		zw.Write(result[16:])
		zw.Close()

		final := make([]byte, 16+zbuf.Len())
		copy(final[:16], result[:16])
		copy(final[16:], zbuf.Bytes())
		binary.LittleEndian.PutUint32(final[12:16], uint32(len(final)))
		return final, nil
	}

	// Patch totalSize.
	binary.LittleEndian.PutUint32(result[12:16], uint32(len(result)))
	return result, nil
}

func (b *ByteCode) Deserialize(buf []byte) error {
	if len(buf) < 16 {
		return fmt.Errorf("vgc: file too short (%d bytes)", len(buf))
	}

	r := &vgcReader{data: buf}

	magic, _ := r.readU32()
	if magic != vgcMagic {
		return fmt.Errorf("vgc: invalid magic %08x (expected VEGA)", magic)
	}
	version, _ := r.readU16()
	if version != vgcVersion {
		return fmt.Errorf("vgc: unsupported version %d (expected %d)", version, vgcVersion)
	}
	flags, _ := r.readU16()
	_, _ = r.readU32() // sourceCRC (not verified here)
	totalSize, _ := r.readU32()
	if uint32(len(buf)) != totalSize {
		return fmt.Errorf("vgc: size mismatch (header says %d, got %d)", totalSize, len(buf))
	}

	// If the payload is compressed, decompress buf[16:] and recurse with the
	// reconstructed uncompressed buffer. The header is rewritten with the
	// compressed flag cleared and the totalSize updated so the recursive call
	// passes its own size check.
	if flags&vgcFlagCompressed != 0 {
		zr, err := zlib.NewReader(bytes.NewReader(buf[16:]))
		if err != nil {
			return fmt.Errorf("vgc: zlib open: %w", err)
		}
		payload, err := io.ReadAll(zr)
		zr.Close()
		if err != nil {
			return fmt.Errorf("vgc: zlib decompress: %w", err)
		}
		full := make([]byte, 16+len(payload))
		copy(full[:16], buf[:16])
		binary.LittleEndian.PutUint16(full[6:8], flags&^vgcFlagCompressed) // clear compressed flag
		binary.LittleEndian.PutUint32(full[12:16], uint32(len(full)))      // update totalSize
		copy(full[16:], payload)
		return b.Deserialize(full)
	}

	hasDebug := flags&vgcFlagDebug != 0

	// Section table.
	sectCount, err := r.readU16()
	if err != nil {
		return fmt.Errorf("vgc: truncated section table")
	}
	type sectEntry struct {
		id             byte
		offset, length uint32
	}
	entries := make([]sectEntry, sectCount)
	for i := range entries {
		id, _ := r.readByte()
		off, _ := r.readU32()
		ln, err := r.readU32()
		if err != nil {
			return fmt.Errorf("vgc: truncated section table entry %d", i)
		}
		entries[i] = sectEntry{id, off, ln}
	}

	// Index sections by ID.
	sectMap := make(map[byte][]byte, len(entries))
	for _, e := range entries {
		end := e.offset + e.length
		if uint32(len(buf)) < end {
			return fmt.Errorf("vgc: section %d out of bounds (offset=%d length=%d)", e.id, e.offset, e.length)
		}
		sectMap[e.id] = buf[e.offset:end]
	}

	// SECTION 0 — Constants.
	if data, ok := sectMap[vgcSectConstants]; ok {
		b.Constants, err = decodeConstants(data)
		if err != nil {
			return fmt.Errorf("vgc constants: %w", err)
		}
	}

	// SECTION 1 — Names.
	if data, ok := sectMap[vgcSectNames]; ok {
		b.Names, err = decodeNames(data)
		if err != nil {
			return fmt.Errorf("vgc names: %w", err)
		}
	}

	// SECTION 2 — Instructions.
	if data, ok := sectMap[vgcSectInstructions]; ok {
		b.Instructions, err = decodeInstructions(data)
		if err != nil {
			return fmt.Errorf("vgc instructions: %w", err)
		}
	}

	// SECTION 3 — Debug (patch SourceLine into already-decoded instructions).
	if hasDebug {
		if data, ok := sectMap[vgcSectDebug]; ok {
			if err := decodeDebug(data, b.Instructions); err != nil {
				return fmt.Errorf("vgc debug: %w", err)
			}
		}
	}

	// SECTION 5 — Stencils (must come before Functions).
	stencilsByName := map[string]*StencilDefinition{}
	if data, ok := sectMap[vgcSectStencils]; ok {
		stencils, err := decodeStencils(data)
		if err != nil {
			return fmt.Errorf("vgc stencils: %w", err)
		}
		for _, s := range stencils {
			stencilsByName[s.Name] = s
		}
	}

	// SECTION 4 — Functions.
	if data, ok := sectMap[vgcSectFunctions]; ok {
		b.Functions, err = decodeFunctions(data, stencilsByName, hasDebug)
		if err != nil {
			return fmt.Errorf("vgc functions: %w", err)
		}
	}

	return nil
}

// -- Section encoders ---------------------------------------------------------

func encodeConstants(consts []Constant) ([]byte, error) {
	var w vgcWriter
	w.writeU32(uint32(len(consts)))
	for _, c := range consts {
		if len(c.Data) > 0xFFFF {
			return nil, fmt.Errorf("constant data too large (%d bytes)", len(c.Data))
		}
		w.writeByte(byte(c.Tag))
		w.writeU16(uint16(len(c.Data)))
		w.Write(c.Data)
	}
	return w.Bytes(), nil
}

func encodeNames(names []string) ([]byte, error) {
	var w vgcWriter
	w.writeU32(uint32(len(names)))
	for _, n := range names {
		if err := w.writeString(n); err != nil {
			return nil, err
		}
	}
	return w.Bytes(), nil
}

func encodeInstructions(instrs []Instruction) ([]byte, error) {
	var w vgcWriter
	w.writeU32(uint32(len(instrs)))
	for i, instr := range instrs {
		w.writeByte(byte(instr.Operation))
		if err := w.writeU24(instr.Argument); err != nil {
			return nil, fmt.Errorf("instruction %d argument: %w", i, err)
		}
		if err := w.writeU24(instr.Offset); err != nil {
			return nil, fmt.Errorf("instruction %d offset: %w", i, err)
		}
		w.writeByte(instr.Extra)
		w.writeU16(uint16(instr.Size))
	}
	return w.Bytes(), nil
}

func encodeDebug(instrs []Instruction) ([]byte, error) {
	var w vgcWriter
	w.writeU32(uint32(len(instrs)))
	for _, instr := range instrs {
		w.writeU16(uint16(instr.SourceLine))
	}
	return w.Bytes(), nil
}

// collectStencils returns all unique StencilDefinitions referenced by function
// params, in deterministic (name-sorted) order.
func collectStencils(b *ByteCode) []*StencilDefinition {
	seen := map[string]*StencilDefinition{}
	for _, fn := range b.Functions {
		for _, p := range fn.Params {
			if p.Stencil != nil {
				seen[p.Stencil.Name] = p.Stencil
			}
		}
	}
	result := make([]*StencilDefinition, 0, len(seen))
	for _, s := range seen {
		result = append(result, s)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

func encodeStencils(stencils []*StencilDefinition) ([]byte, error) {
	var w vgcWriter
	w.writeU32(uint32(len(stencils)))
	for _, s := range stencils {
		if err := w.writeString(s.Name); err != nil {
			return nil, fmt.Errorf("stencil %q name: %w", s.Name, err)
		}
		w.writeU32(uint32(s.TotalSize))
		if len(s.Fields) > 0xFF {
			return nil, fmt.Errorf("stencil %q: too many fields (%d)", s.Name, len(s.Fields))
		}
		w.writeByte(byte(len(s.Fields)))
		for _, f := range s.Fields {
			if err := w.writeString(f.Name); err != nil {
				return nil, fmt.Errorf("stencil %q field %q: %w", s.Name, f.Name, err)
			}
			w.writeU32(uint32(f.Offset))
			w.writeByte(byte(f.Tag))
			w.writeU32(uint32(f.Capacity))
		}
	}
	return w.Bytes(), nil
}

func encodeFunctions(b *ByteCode, includeDebug bool) ([]byte, error) {
	// Deterministic order.
	names := make([]string, 0, len(b.Functions))
	for name := range b.Functions {
		names = append(names, name)
	}
	sort.Strings(names)

	var w vgcWriter
	w.writeU32(uint32(len(names)))
	for _, name := range names {
		fn := b.Functions[name]

		if err := w.writeString(fn.Name); err != nil {
			return nil, fmt.Errorf("function %q name: %w", fn.Name, err)
		}
		w.writeU32(uint32(fn.FrameSize))

		if len(fn.Params) > 0xFF {
			return nil, fmt.Errorf("function %q: too many params (%d)", fn.Name, len(fn.Params))
		}
		w.writeByte(byte(len(fn.Params)))
		for _, p := range fn.Params {
			if err := w.writeString(p.Name); err != nil {
				return nil, fmt.Errorf("function %q param %q: %w", fn.Name, p.Name, err)
			}
			w.writeU16(uint16(p.SlotID))
			w.writeByte(byte(p.Tag))
			w.writeByte(p.Mask)
			if p.Stencil != nil {
				w.writeByte(1)
				if err := w.writeString(p.Stencil.Name); err != nil {
					return nil, fmt.Errorf("function %q param %q stencil name: %w", fn.Name, p.Name, err)
				}
			} else {
				w.writeByte(0)
			}
		}

		// Embedded ByteCode — functions do not carry sub-functions.
		if err := writeEmbeddedBytecode(&w, fn.ByteCode, includeDebug); err != nil {
			return nil, fmt.Errorf("function %q bytecode: %w", fn.Name, err)
		}
	}
	return w.Bytes(), nil
}

// writeEmbeddedBytecode serialises the three (or four with debug) sub-sections
// of a function's ByteCode, each prefixed by a uint32 payload length.
// The Functions map is intentionally not serialised here — function bytecodes
// share the top-level Functions map and are not recursively nested.
func writeEmbeddedBytecode(w *vgcWriter, bc *ByteCode, includeDebug bool) error {
	constData, err := encodeConstants(bc.Constants)
	if err != nil {
		return err
	}
	namesData, err := encodeNames(bc.Names)
	if err != nil {
		return err
	}
	instrData, err := encodeInstructions(bc.Instructions)
	if err != nil {
		return err
	}

	w.writeU32(uint32(len(constData)))
	w.Write(constData)
	w.writeU32(uint32(len(namesData)))
	w.Write(namesData)
	w.writeU32(uint32(len(instrData)))
	w.Write(instrData)

	if includeDebug {
		dbgData, err := encodeDebug(bc.Instructions)
		if err != nil {
			return err
		}
		w.writeU32(uint32(len(dbgData)))
		w.Write(dbgData)
	}
	return nil
}

// -- Section decoders ---------------------------------------------------------

func decodeConstants(data []byte) ([]Constant, error) {
	r := &vgcReader{data: data}
	count, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated count")
	}
	consts := make([]Constant, count)
	for i := range consts {
		tag, err := r.readByte()
		if err != nil {
			return nil, fmt.Errorf("constant %d: truncated tag", i)
		}
		dlen, err := r.readU16()
		if err != nil {
			return nil, fmt.Errorf("constant %d: truncated length", i)
		}
		cdata, err := r.readBytes(int(dlen))
		if err != nil {
			return nil, fmt.Errorf("constant %d: truncated data", i)
		}
		consts[i] = Constant{Tag: slot.TypeTag(tag), Data: cdata}
	}
	return consts, nil
}

func decodeNames(data []byte) ([]string, error) {
	r := &vgcReader{data: data}
	count, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated count")
	}
	names := make([]string, count)
	for i := range names {
		s, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("name %d: %w", i, err)
		}
		names[i] = s
	}
	return names, nil
}

func decodeInstructions(data []byte) ([]Instruction, error) {
	r := &vgcReader{data: data}
	count, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated count")
	}
	instrs := make([]Instruction, count)
	for i := range instrs {
		op, err := r.readByte()
		if err != nil {
			return nil, fmt.Errorf("instruction %d: truncated", i)
		}
		arg, err := r.readU24()
		if err != nil {
			return nil, fmt.Errorf("instruction %d argument: %w", i, err)
		}
		off, err := r.readU24()
		if err != nil {
			return nil, fmt.Errorf("instruction %d offset: %w", i, err)
		}
		extra, err := r.readByte()
		if err != nil {
			return nil, fmt.Errorf("instruction %d extra: %w", i, err)
		}
		size, err := r.readU16()
		if err != nil {
			return nil, fmt.Errorf("instruction %d size: %w", i, err)
		}
		instrs[i] = Instruction{
			Operation: OperationCode(op),
			Argument:  arg,
			Offset:    off,
			Extra:     extra,
			Size:      int(size),
		}
	}
	return instrs, nil
}

func decodeDebug(data []byte, instrs []Instruction) error {
	r := &vgcReader{data: data}
	count, err := r.readU32()
	if err != nil {
		return fmt.Errorf("truncated count")
	}
	if int(count) != len(instrs) {
		return fmt.Errorf("debug count mismatch: got %d, want %d", count, len(instrs))
	}
	for i := range instrs {
		line, err := r.readU16()
		if err != nil {
			return fmt.Errorf("entry %d: truncated", i)
		}
		instrs[i].SourceLine = int(line)
	}
	return nil
}

func decodeStencils(data []byte) ([]*StencilDefinition, error) {
	r := &vgcReader{data: data}
	count, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated count")
	}
	stencils := make([]*StencilDefinition, count)
	for i := range stencils {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("stencil %d name: %w", i, err)
		}
		totalSize, err := r.readU32()
		if err != nil {
			return nil, fmt.Errorf("stencil %q: truncated total size", name)
		}
		fieldCount, err := r.readByte()
		if err != nil {
			return nil, fmt.Errorf("stencil %q: truncated field count", name)
		}
		fields := make([]StencilFieldLayout, fieldCount)
		for j := range fields {
			fname, err := r.readString()
			if err != nil {
				return nil, fmt.Errorf("stencil %q field %d name: %w", name, j, err)
			}
			foff, err := r.readU32()
			if err != nil {
				return nil, fmt.Errorf("stencil %q field %q: truncated offset", name, fname)
			}
			ftag, err := r.readByte()
			if err != nil {
				return nil, fmt.Errorf("stencil %q field %q: truncated tag", name, fname)
			}
			fcap, err := r.readU32()
			if err != nil {
				return nil, fmt.Errorf("stencil %q field %q: truncated capacity", name, fname)
			}
			fields[j] = StencilFieldLayout{
				Name:     fname,
				Offset:   int(foff),
				Tag:      slot.TypeTag(ftag),
				Capacity: int(fcap),
			}
		}
		stencils[i] = &StencilDefinition{
			Name:      name,
			Fields:    fields,
			TotalSize: int(totalSize),
		}
	}
	return stencils, nil
}

func decodeFunctions(data []byte, stencilsByName map[string]*StencilDefinition, hasDebug bool) (map[string]*FunctionDefinition, error) {
	r := &vgcReader{data: data}
	count, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated count")
	}
	fns := make(map[string]*FunctionDefinition, count)
	for i := range count {
		name, err := r.readString()
		if err != nil {
			return nil, fmt.Errorf("function %d name: %w", i, err)
		}
		frameSize, err := r.readU32()
		if err != nil {
			return nil, fmt.Errorf("function %q: truncated frame size", name)
		}
		paramCount, err := r.readByte()
		if err != nil {
			return nil, fmt.Errorf("function %q: truncated param count", name)
		}
		params := make([]ParamDefinition, paramCount)
		for j := range params {
			pname, err := r.readString()
			if err != nil {
				return nil, fmt.Errorf("function %q param %d name: %w", name, j, err)
			}
			slotID, err := r.readU16()
			if err != nil {
				return nil, fmt.Errorf("function %q param %q: truncated slot ID", name, pname)
			}
			ptag, err := r.readByte()
			if err != nil {
				return nil, fmt.Errorf("function %q param %q: truncated tag", name, pname)
			}
			mask, err := r.readByte()
			if err != nil {
				return nil, fmt.Errorf("function %q param %q: truncated mask", name, pname)
			}
			hasStencil, err := r.readByte()
			if err != nil {
				return nil, fmt.Errorf("function %q param %q: truncated stencil flag", name, pname)
			}
			var stencil *StencilDefinition
			if hasStencil != 0 {
				sname, err := r.readString()
				if err != nil {
					return nil, fmt.Errorf("function %q param %q: truncated stencil name", name, pname)
				}
				stencil = stencilsByName[sname] // nil if stencils section was absent
			}
			params[j] = ParamDefinition{
				Name:    pname,
				SlotID:  int(slotID),
				Tag:     slot.TypeTag(ptag),
				Mask:    mask,
				Stencil: stencil,
			}
		}

		bc, err := decodeEmbeddedBytecode(r, hasDebug)
		if err != nil {
			return nil, fmt.Errorf("function %q bytecode: %w", name, err)
		}
		fns[name] = &FunctionDefinition{
			Name:      name,
			ByteCode:  bc,
			Params:    params,
			FrameSize: int(frameSize),
		}
	}
	return fns, nil
}

func decodeEmbeddedBytecode(r *vgcReader, hasDebug bool) (*ByteCode, error) {
	bc := &ByteCode{}

	constLen, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated constants length")
	}
	constData, err := r.readBytes(int(constLen))
	if err != nil {
		return nil, fmt.Errorf("truncated constants data")
	}
	bc.Constants, err = decodeConstants(constData)
	if err != nil {
		return nil, fmt.Errorf("constants: %w", err)
	}

	namesLen, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated names length")
	}
	namesData, err := r.readBytes(int(namesLen))
	if err != nil {
		return nil, fmt.Errorf("truncated names data")
	}
	bc.Names, err = decodeNames(namesData)
	if err != nil {
		return nil, fmt.Errorf("names: %w", err)
	}

	instrLen, err := r.readU32()
	if err != nil {
		return nil, fmt.Errorf("truncated instructions length")
	}
	instrData, err := r.readBytes(int(instrLen))
	if err != nil {
		return nil, fmt.Errorf("truncated instructions data")
	}
	bc.Instructions, err = decodeInstructions(instrData)
	if err != nil {
		return nil, fmt.Errorf("instructions: %w", err)
	}

	if hasDebug {
		dbgLen, err := r.readU32()
		if err != nil {
			return nil, fmt.Errorf("truncated debug length")
		}
		dbgData, err := r.readBytes(int(dbgLen))
		if err != nil {
			return nil, fmt.Errorf("truncated debug data")
		}
		if err := decodeDebug(dbgData, bc.Instructions); err != nil {
			return nil, fmt.Errorf("debug: %w", err)
		}
	}

	return bc, nil
}

type vgcWriter struct{ bytes.Buffer }

func (w *vgcWriter) writeByte(b byte) { w.Buffer.WriteByte(b) }

func (w *vgcWriter) writeU16(v uint16) {
	var b [2]byte
	binary.LittleEndian.PutUint16(b[:], v)
	w.Write(b[:])
}

func (w *vgcWriter) writeU32(v uint32) {
	var b [4]byte
	binary.LittleEndian.PutUint32(b[:], v)
	w.Write(b[:])
}

func (w *vgcWriter) writeU24(v int) error {
	if v < 0 || v > 0xFFFFFF {
		return fmt.Errorf("value %d out of uint24 range [0, 16777215]", v)
	}
	w.Buffer.WriteByte(byte(v))
	w.Buffer.WriteByte(byte(v >> 8))
	w.Buffer.WriteByte(byte(v >> 16))
	return nil
}

func (w *vgcWriter) writeString(s string) error {
	b := []byte(s)
	if len(b) > 0xFFFF {
		return fmt.Errorf("string too long (%d bytes)", len(b))
	}
	w.writeU16(uint16(len(b)))
	w.Write(b)
	return nil
}

type vgcReader struct {
	data []byte
	pos  int
}

func (r *vgcReader) readByte() (byte, error) {
	if r.pos >= len(r.data) {
		return 0, fmt.Errorf("unexpected EOF")
	}
	b := r.data[r.pos]
	r.pos++
	return b, nil
}

func (r *vgcReader) readBytes(n int) ([]byte, error) {
	if r.pos+n > len(r.data) {
		return nil, fmt.Errorf("unexpected EOF (need %d, have %d)", n, len(r.data)-r.pos)
	}
	b := make([]byte, n)
	copy(b, r.data[r.pos:r.pos+n])
	r.pos += n
	return b, nil
}

func (r *vgcReader) readU16() (uint16, error) {
	b, err := r.readBytes(2)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint16(b), nil
}

func (r *vgcReader) readU32() (uint32, error) {
	b, err := r.readBytes(4)
	if err != nil {
		return 0, err
	}
	return binary.LittleEndian.Uint32(b), nil
}

func (r *vgcReader) readU24() (int, error) {
	b, err := r.readBytes(3)
	if err != nil {
		return 0, err
	}
	return int(b[0]) | int(b[1])<<8 | int(b[2])<<16, nil
}

func (r *vgcReader) readString() (string, error) {
	slen, err := r.readU16()
	if err != nil {
		return "", err
	}
	b, err := r.readBytes(int(slen))
	if err != nil {
		return "", err
	}
	return string(b), nil
}
