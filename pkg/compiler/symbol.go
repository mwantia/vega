package compiler

import "github.com/mwantia/vega/pkg/value"

type SymbolInfo struct {
	SlotID   int
	Tag      value.TypeTag
	Mask     byte
	Capacity int                // byte capacity for TagSlice variables (0 for scalar types)
	Stencil  *StencilDefinition // non-nil for struct/tuple variables
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

func (st *SymbolTable) DefineStencil(name string, stencil *StencilDefinition) SymbolInfo {
	info := SymbolInfo{
		SlotID:  st.nextSlot,
		Stencil: stencil,
	}
	st.symbols[name] = info
	st.nextSlot++
	return info
}

func (st *SymbolTable) Remove(name string) {
	delete(st.symbols, name)
}
