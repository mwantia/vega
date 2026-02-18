package value

import "encoding/binary"

// CharValue wraps rune (int32). Size = 4 bytes.
// The view slice points into the alloc buffer — no data is owned.
type CharValue struct {
	view []byte
}

func NewChar(view []byte) *CharValue {
	return &CharValue{
		view: view,
	}
}

func (v *CharValue) Type() string {
	return "char"
}

func (v *CharValue) String() string {
	return string(v.Data())
}

func (v *CharValue) Size() byte {
	return 4
}

func (v *CharValue) Data() rune {
	return rune(binary.LittleEndian.Uint32(v.view))
}

func (v *CharValue) View() []byte {
	return v.view
}

var _ Allocable = (*CharValue)(nil)
