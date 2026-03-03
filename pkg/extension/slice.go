package extension

import (
	"bytes"
	"fmt"
	"strings"

	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/slot"
)

func init() {
	// --- Members ---

	descriptor.Global.RegisterMember(slot.TagSlice, &descriptor.MemberDescriptor{
		Name:      "length",
		Readonly:  true,
		ReturnTag: slot.TagInteger,
		Getter: func(inst slot.StackSlot) (slot.StackSlot, error) {
			data := inst.Bytes()
			n := bytes.IndexByte(data, 0)
			if n < 0 {
				n = len(data)
			}
			return slot.NewInt(n), nil
		},
	})

	descriptor.Global.RegisterMember(slot.TagSlice, &descriptor.MemberDescriptor{
		Name:      "capacity",
		Readonly:  true,
		ReturnTag: slot.TagInteger,
		Getter: func(inst slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewInt(len(inst.Bytes())), nil
		},
	})

	// --- Methods ---

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name:      "upper",
		Params:    []descriptor.ParameterLayoutDescriptor{},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewString(strings.ToUpper(inst.AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name:      "lower",
		Params:    []descriptor.ParameterLayoutDescriptor{},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewString(strings.ToLower(inst.AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name:      "trim",
		Params:    []descriptor.ParameterLayoutDescriptor{},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewString(strings.TrimSpace(inst.AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "contains",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "sub", Tag: slot.TagSlice, Position: 0, Required: true},
		},
		ReturnTag: slot.TagBoolean,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewBool(strings.Contains(inst.AsString(), args[0].AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "startswith",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "prefix", Tag: slot.TagSlice, Position: 0, Required: true},
		},
		ReturnTag: slot.TagBoolean,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewBool(strings.HasPrefix(inst.AsString(), args[0].AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "endswith",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "suffix", Tag: slot.TagSlice, Position: 0, Required: true},
		},
		ReturnTag: slot.TagBoolean,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewBool(strings.HasSuffix(inst.AsString(), args[0].AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "index",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "sub", Tag: slot.TagSlice, Position: 0, Required: true},
		},
		ReturnTag: slot.TagInteger,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewInt(strings.Index(inst.AsString(), args[0].AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "trimprefix",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "prefix", Tag: slot.TagSlice, Position: 0, Required: true},
		},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewString(strings.TrimPrefix(inst.AsString(), args[0].AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "trimsuffix",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "suffix", Tag: slot.TagSlice, Position: 0, Required: true},
		},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewString(strings.TrimSuffix(inst.AsString(), args[0].AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "replace",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "old", Tag: slot.TagSlice, Position: 0, Required: true},
			{Name: "new", Tag: slot.TagSlice, Position: 1, Required: true},
		},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			return slot.NewString(strings.ReplaceAll(inst.AsString(), args[0].AsString(), args[1].AsString())), nil
		},
	})

	descriptor.Global.RegisterMethod(slot.TagSlice, &descriptor.MethodDescriptor{
		Name: "slice",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "from", Tag: slot.TagInteger, Position: 0, Required: true},
			{Name: "length", Tag: slot.TagInteger, Position: 1, Required: false},
		},
		ReturnTag: slot.TagSlice,
		Run: func(inst slot.StackSlot, session descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			data := inst.Bytes()
			n := bytes.IndexByte(data, 0)
			if n < 0 {
				n = len(data)
			}

			from, err := args[0].AsInt()
			if err != nil {
				return slot.VoidSlot, err
			}

			length := n - from
			if args[1].Tag != slot.TagVoid {
				length, err = args[1].AsInt()
				if err != nil {
					return slot.VoidSlot, err
				}
			}

			if from < 0 || length < 0 || from+length > n {
				return slot.VoidSlot, fmt.Errorf("slice [%d:%d] out of range (length %d)", from, from+length, n)
			}
			return slot.NewBytes(data[from : from+length]), nil
		},
	})
}
