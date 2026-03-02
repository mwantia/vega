package vm

import (
	"context"
	"fmt"
	"io"

	"github.com/mwantia/vfs"
)

type RuntimeSession struct {
	stdin  io.Reader
	stdout io.Writer
	stderr io.Writer

	ctx context.Context
	vfs vfs.VirtualFileSystem
}

func (rs *RuntimeSession) FprintfStdout(format string, a ...any) error {
	_, err := fmt.Fprintf(rs.stdout, format, a...)
	return err
}

func (rs *RuntimeSession) FprintfStderr(format string, a ...any) error {
	_, err := fmt.Fprintf(rs.stderr, format, a...)
	return err
}

func (rs *RuntimeSession) Stdin() io.Reader {
	return rs.stdin
}

func (rs *RuntimeSession) Stdout() io.Writer {
	return rs.stdout
}

func (rs *RuntimeSession) Stderr() io.Writer {
	return rs.stderr
}

func (rs *RuntimeSession) Context() context.Context {
	return rs.ctx
}

func (rs *RuntimeSession) VFS() vfs.VirtualFileSystem {
	return rs.vfs
}
