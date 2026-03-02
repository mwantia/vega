package extension

import (
	"strings"

	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/value"
)

func init() {
	// --- Members ---

	descriptor.Global.RegisterMember(value.TagSlice, &descriptor.MemberDescriptor{
		Name:      "length",
		Readonly:  true,
		ReturnTag: value.TagInteger,
		Getter: func(inst value.Value) (value.Value, error) {
			return value.NewInteger(value.EncodeInteger(inst.(*value.Slice).Length())), nil
		},
	})

	descriptor.Global.RegisterMember(value.TagSlice, &descriptor.MemberDescriptor{
		Name:      "capacity",
		Readonly:  true,
		ReturnTag: value.TagInteger,
		Getter: func(inst value.Value) (value.Value, error) {
			return value.NewInteger(value.EncodeInteger(inst.(*value.Slice).Capacity())), nil
		},
	})

	// --- Methods ---

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name:      "upper",
		Params:    []descriptor.ParameterLayoutDescriptor{},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(strings.ToUpper(inst.String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name:      "lower",
		Params:    []descriptor.ParameterLayoutDescriptor{},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(strings.ToLower(inst.String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name:      "trim",
		Params:    []descriptor.ParameterLayoutDescriptor{},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(strings.TrimSpace(inst.String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "contains",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "sub",
				Tag:      value.TagSlice,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagBoolean,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewBoolean(value.EncodeBoolean(strings.Contains(inst.String(), args[0].String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "startswith",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "prefix",
				Tag:      value.TagSlice,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagBoolean,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewBoolean(value.EncodeBoolean(strings.HasPrefix(inst.String(), args[0].String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "endswith",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "suffix",
				Tag:      value.TagSlice,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagBoolean,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewBoolean(value.EncodeBoolean(strings.HasSuffix(inst.String(), args[0].String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "index",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "sub",
				Tag:      value.TagSlice,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagInteger,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewInteger(value.EncodeInteger(strings.Index(inst.String(), args[0].String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "trimprefix",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "prefix",
				Tag:      value.TagSlice,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(strings.TrimPrefix(inst.String(), args[0].String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "trimsuffix",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "suffix",
				Tag:      value.TagSlice,
				Position: 0,
				Required: true,
			},
		},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(strings.TrimSuffix(inst.String(), args[0].String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "replace",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "old",
				Tag:      value.TagSlice,
				Position: 0,
				Required: true,
			},
			{
				Name:     "new",
				Tag:      value.TagSlice,
				Position: 1,
				Required: true,
			},
		},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			return value.NewSlice([]byte(strings.ReplaceAll(inst.String(), args[0].String(), args[1].String()))), nil
		},
	})

	descriptor.Global.RegisterMethod(value.TagSlice, &descriptor.MethodDescriptor{
		Name: "slice",
		Params: []descriptor.ParameterLayoutDescriptor{
			{
				Name:     "from",
				Tag:      value.TagInteger,
				Position: 0,
				Required: true,
			},
			{
				Name:     "length",
				Tag:      value.TagInteger,
				Position: 1,
				Required: false,
			},
		},
		ReturnTag: value.TagSlice,
		Run: func(inst value.Value, session descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			s := inst.(*value.Slice)
			from := args[0].(*value.Integer).Data()
			if args[1] == nil {
				return s.SubSlice(from, s.Length()-from)
			}
			return s.SubSlice(from, args[1].(*value.Integer).Data())
		},
	})
}
