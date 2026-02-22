package value

import "fmt"

// RawValue holds the raw bytes of a stencil (struct/tuple) value so it can
// be transported through the expression stack when passing structs as function
// arguments. It is never used as a general-purpose variable type.
type RawValue struct {
	Data []byte
}

func (r *RawValue) Type() string   { return "raw" }
func (r *RawValue) String() string { return fmt.Sprintf("<raw %d bytes>", len(r.Data)) }
