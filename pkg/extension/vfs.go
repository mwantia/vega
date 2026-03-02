package extension

import (
	"fmt"

	"github.com/mwantia/vega/pkg/descriptor"
	"github.com/mwantia/vega/pkg/value"
)

func init() {
	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "read",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "path", Tag: value.TagSlice, Position: 0, Required: true},
			{Name: "offset", Tag: value.TagLong, Position: 1, Required: true},
			{Name: "size", Tag: value.TagLong, Position: 2, Required: true},
		},
		ReturnTag: value.TagSlice,
		Run: func(v value.Value, ds descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			if ds.VFS() == nil {
				return nil, fmt.Errorf("no VFS attached to current session")
			}

			path := args[0].String()
			offset := args[1].(*value.Long).Data()
			size := args[2].(*value.Long).Data()

			data, err := ds.VFS().ReadFile(ds.Context(), path, offset, size)
			if err != nil {
				return nil, fmt.Errorf("read failed: %v", err)
			}

			return value.NewSlice(data), nil
		},
	})
	descriptor.Global.RegisterStatic(&descriptor.MethodDescriptor{
		Name: "exists",
		Params: []descriptor.ParameterLayoutDescriptor{
			{Name: "path", Tag: value.TagSlice, Required: true},
		},
		ReturnTag: value.TagBoolean,
		Run: func(v value.Value, ds descriptor.DescriptorSession, args []value.Value) (value.Value, error) {
			if ds.VFS() == nil {
				return nil, fmt.Errorf("no VFS attached to current session")
			}

			path := args[0].String()
			exists, _ := ds.VFS().LookupMetadata(ds.Context(), path)

			data, err := value.Encode(exists)
			if err != nil {
				return nil, fmt.Errorf("exists failed: %v", err)
			}
			return value.NewBoolean(data), nil
		},
	})
}
