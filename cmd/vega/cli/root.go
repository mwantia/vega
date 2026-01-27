package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/lexer"
	"github.com/mwantia/vega/pkg/parser"
	"github.com/mwantia/vega/pkg/repl"
	"github.com/mwantia/vega/pkg/value"
	"github.com/mwantia/vega/pkg/vm"
	"github.com/mwantia/vfs"
	"github.com/mwantia/vfs/mount"
	"github.com/mwantia/vfs/mount/builder"
	"github.com/spf13/cobra"
)

func NewRootCommand(info VersionInfo) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "vega <uri>",
		Short: "Vega - Virtual Execution & Graph Abstraction",
		Long: `Vega is a lightweight scripting language and runtime for VFS operations.

It provides a programmable interface between the host OS filesystem and
VFS-mounted storage backends (SQLite, S3, PostgreSQL, ephemeral, etc.)`,
		SilenceErrors: true,
		SilenceUsage:  true,
		Args:          cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			if ctx == nil {
				ctx = context.Background()
			}

			uri := "ephemeral://"
			if len(args) == 1 {
				uri = strings.TrimSpace(args[0])
			}

			fs, err := vfs.NewVirtualFileSystem()
			if err != nil {
				return fmt.Errorf("failed to create VFS: %w", err)
			}

			steps, err := mount.IdentifyMountSteps(ctx, uri)
			if err != nil {
				return fmt.Errorf("failed to identify mount for '%s': %w", uri, err)
			}

			steps = append(steps, builder.AsCascading())
			if err := fs.Mount(ctx, "/", steps...); err != nil {
				return fmt.Errorf("failed to mount root: %w", err)
			}

			interactive, _ := cmd.Flags().GetBool("interactive")
			disasm, _ := cmd.Flags().GetBool("disasm")

			var bytecode *compiler.Bytecode

			if script, _ := cmd.Flags().GetString("script"); script != "" {
				content, err := os.ReadFile(script)
				if err != nil {
					return fmt.Errorf("failed to read file: %w", err)
				}

				bytecode, err = compile(string(content))
				if err != nil {
					return err
				}
			}

			if command, _ := cmd.Flags().GetString("command"); command != "" {
				bytecode, err = compile(command)
				if err != nil {
					return err
				}
			}

			vm := createVM(ctx, fs)
			defer vm.Shutdown()

			if bytecode != nil {
				if disasm {
					fmt.Println(bytecode.Disassemble())
					fmt.Println("--- Execution ---")
				}

				exitCode, err := vm.Run(bytecode)
				if err != nil {
					return fmt.Errorf("runtime error: %w", err)
				}

				vm.SetGlobal("exitcode", value.NewInteger(int64(exitCode)))
				// If interactive is 'false' (default), close immediately to avoid running vega
				if !interactive {
					return fs.Shutdown(ctx)
				}
			}

			defer fs.Shutdown(ctx)
			return repl.RunTUI(vm, disasm)
		},
	}

	// Existing flags
	cmd.Flags().BoolP("interactive", "i", false, "Keep open after executing (default is 'false')")
	cmd.Flags().StringP("command", "c", "", "Execute a single Vega command")
	cmd.Flags().StringP("script", "s", "", "Execute a Vega script file")
	cmd.Flags().BoolP("disasm", "d", false, "Show disassembled bytecode (debug)")
	// Set version used by './vega version'
	cmd.Version = fmt.Sprintf("%s.%s", info.Version, info.Commit)

	return cmd
}

func compile(input string) (*compiler.Bytecode, error) {
	l := lexer.New(input)
	tokens, err := l.Tokenize()
	if err != nil {
		return nil, fmt.Errorf("syntax error: %w", err)
	}

	p := parser.New(tokens)
	program, err := p.Parse()
	if err != nil {
		return nil, fmt.Errorf("parse error: %w", err)
	}

	c := compiler.New()
	bytecode, err := c.Compile(program)
	if err != nil {
		return nil, fmt.Errorf("compile error: %w", err)
	}

	return bytecode, nil
}

func createVM(ctx context.Context, fs vfs.VirtualFileSystem) *vm.VirtualMachine {
	v := vm.NewVirtualMachine()
	v.SetContext(ctx)
	v.SetVFS(fs)
	v.SetStdout(os.Stdout)
	v.SetStderr(os.Stderr)
	v.SetStdin(os.Stdin)

	return v
}
