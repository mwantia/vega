package extension

import (
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/slot"
)

func init() {
	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name:      "print",
		Params:    nil, // variadic: accepts any number of any-typed arguments
		ReturnTag: slot.TagVoid,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			var out strings.Builder
			for _, arg := range args {
				out.WriteString(arg.Format())
			}
			_, err := fmt.Fprintln(session.Stdout(), out.String())
			return slot.VoidSlot, err
		},
	})

	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "string",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "value", Tag: slot.TagAny, Position: 0, Required: true},
		},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewString(args[0].Format()), nil
		},
	})

	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "type",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "value", Tag: slot.TagAny, Position: 0, Required: true},
		},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			name, _ := slot.NameForTag(args[0].Tag)
			return slot.NewString(name), nil
		},
	})
}
