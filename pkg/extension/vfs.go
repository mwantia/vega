package extension

import (
	"fmt"

	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/slot"
)

func init() {
	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "read",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "path", Tag: slot.TagSlice, Position: 0, Required: true},
			{Name: "offset", Tag: slot.TagLong, Position: 1, Required: true},
			{Name: "size", Tag: slot.TagLong, Position: 2, Required: true},
		},
		ReturnTag: slot.TagSlice,
		Run: func(v slot.StackSlot, ds descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			if ds.VFS() == nil {
				return slot.VoidSlot, fmt.Errorf("no VFS attached to current session")
			}

			path := args[0].AsString()
			offset, err := args[1].AsInt()
			if err != nil {
				return slot.VoidSlot, err
			}
			size, err := args[2].AsInt()
			if err != nil {
				return slot.VoidSlot, err
			}

			data, err := ds.VFS().ReadFile(ds.Context(), path, int64(offset), int64(size))
			if err != nil {
				return slot.VoidSlot, fmt.Errorf("read failed: %v", err)
			}

			return slot.NewBytes(data), nil
		},
	})

	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "exists",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "path", Tag: slot.TagSlice, Required: true},
		},
		ReturnTag: slot.TagBoolean,
		Run: func(v slot.StackSlot, ds descriptor.DescriptorSession, args []slot.StackSlot) (slot.StackSlot, error) {
			if ds.VFS() == nil {
				return slot.VoidSlot, fmt.Errorf("no VFS attached to current session")
			}

			path := args[0].AsString()
			exists, _ := ds.VFS().LookupMetadata(ds.Context(), path)
			return slot.NewBool(exists), nil
		},
	})
}
