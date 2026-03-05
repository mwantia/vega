package cli

import (
	"context"
	"fmt"
	"time"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/lexer"
	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vfs"
	"github.com/mwantia/vfs/mount"
	"github.com/mwantia/vfs/mount/builder"
)

func createVirtualFileSystem(uri string) (vfs.VirtualFileSystem, error) {
	fs, err := vfs.NewVirtualFileSystem()
	if err != nil {
		return nil, fmt.Errorf("failed to create VFS: %w", err)
	}

	ctx, canc := context.WithTimeout(context.Background(), time.Second*5)
	defer canc()

	steps, err := mount.IdentifyMountSteps(ctx, uri)
	if err != nil {
		return nil, fmt.Errorf("failed to identify mount for '%s': %w", uri, err)
	}

	steps = append(steps, builder.AsCascading())
	if err := fs.Mount(ctx, "/", steps...); err != nil {
		return nil, fmt.Errorf("failed to mount root: %w", err)
	}

	return fs, nil
}

func compileFileContent(t string) (*compiler.ByteCode, error) {
	l, err := lexer.NewLexer(t)
	if err != nil {
		return nil, fmt.Errorf("syntax error: %w", err)
	}
	buffer, err := l.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("syntax error: %w", err)
	}

	p := parser.NewParser()
	program, err := p.MakeProgram(buffer)
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	c := compiler.NewCompiler()
	bytecode, err := c.Compile(program)
	if err != nil {
		return nil, fmt.Errorf("compile error: %w", err)
	}

	return bytecode, nil
}
