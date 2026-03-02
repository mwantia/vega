package value

import "fmt"

// Raw holds the raw bytes of a stencil (struct/tuple) value so it can
// be transported through the expression stack when passing structs as function
// arguments. It is never used as a general-purpose variable type.
type Raw struct {
	Data []byte
}

func (r *Raw) Type() string {
	return "raw"
}

func (r *Raw) String() string {
	return fmt.Sprintf("<raw %d bytes>", len(r.Data))
}
