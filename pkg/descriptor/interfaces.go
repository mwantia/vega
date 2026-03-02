package descriptor

import (
	"context"
	"io"

	"github.com/mwantia/vega/pkg/value"
	"github.com/mwantia/vfs"
)

type DescriptorRegister interface {
	RegisterMethod(value.TypeTag, *MethodDescriptor) error

	RegisterStatic(*MethodDescriptor) error

	RegisterMember(value.TypeTag, *MemberDescriptor) error

	RegisterStencil(*StencilDescriptor) error

	LookupMethod(value.TypeTag, string) (*MethodDescriptor, bool)

	LookupStatic(string) (*MethodDescriptor, bool)

	LookupMember(value.TypeTag, string) (*MemberDescriptor, bool)

	LookupStencil(string) (*StencilDescriptor, bool)
}

type DescriptorSession interface {
	FprintfStdout(format string, a ...any) error

	FprintfStderr(format string, a ...any) error

	Stdin() io.Reader

	Stdout() io.Writer

	Stderr() io.Writer

	Context() context.Context

	VFS() vfs.VirtualFileSystem
}
