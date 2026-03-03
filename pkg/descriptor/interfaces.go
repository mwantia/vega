package descriptor

import (
	"context"
	"io"

	"github.com/mwantia/vega/pkg/slot"
	"github.com/mwantia/vfs"
)

type DescriptorRegister interface {
	RegisterMethod(slot.TypeTag, *MethodDescriptor) error

	RegisterStatic(*MethodDescriptor) error

	RegisterMember(slot.TypeTag, *MemberDescriptor) error

	RegisterStencil(*StencilDescriptor) error

	LookupMethod(slot.TypeTag, string) (*MethodDescriptor, bool)

	LookupStatic(string) (*MethodDescriptor, bool)

	LookupMember(slot.TypeTag, string) (*MemberDescriptor, bool)

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
