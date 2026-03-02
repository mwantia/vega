package extension

import (
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/value"
)

func init() {
	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name:      "print",
		Params:    nil, // variadic: accepts any number of any-typed arguments
		ReturnTag: value.TagVoid,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			var out strings.Builder
			for _, arg := range args {
				s := arg.String()
				out.WriteString(s)
			}
			_, err := fmt.Fprintln(session.Stdout(), out.String())
			return nil, err
		},
	})

	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "string",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "value",
				Tag:      value.TagAny,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(args[0].String())), nil
		},
	})

	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "type",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "value",
				Tag:      value.TagAny,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(args[0].Type())), nil
		},
	})
}
