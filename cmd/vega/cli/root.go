package cli

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/repl"
	"github.com/mwantia/vega/pkg/vm"
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

			fs, err := createVirtualFileSystem(uri)
			if err != nil {
				return err
			}

			interactive, _ := cmd.Flags().GetBool("interactive")
			disasm, _ := cmd.Flags().GetBool("disasm")

			var bytecode *compiler.ByteCode

			file, _ := cmd.Flags().GetString("file")
			if len(file) > 0 {
				buf, err := os.ReadFile(file)
				if err != nil {
					return fmt.Errorf("failed to read file: %v", err)
				}

				if compiler.HasValidMagic(buf[:4]) {
					bytecode = &compiler.ByteCode{}
					if err := bytecode.Deserialize(buf); err != nil {
						return fmt.Errorf("failed to deserialize vgc file: %v", err)
					}
				} else {
					content := string(buf)
					if bytecode, err = compileFileContent(content); err != nil {
						return fmt.Errorf("failed to compile content: %v", err)
					}
				}
			} else {
				if command, _ := cmd.Flags().GetString("command"); command != "" {
					bytecode, err = compileFileContent(command)
					if err != nil {
						return fmt.Errorf("failed to compile content: %v", err)
					}
				}
			}

			vm := vm.NewVM(fs)
			vm.Stdin(os.Stdin)
			vm.Stdout(os.Stdout)
			vm.Stderr(os.Stderr)

			if bytecode != nil {
				if disasm {
					fmt.Println(bytecode.Disassemble())
					fmt.Println("=== Execution ===")
				}

				if _, err := vm.Run(ctx, bytecode); err != nil {
					return fmt.Errorf("runtime error: %w", err)
				}

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
	cmd.Flags().StringP("file", "f", "", "Execute a Vega file")
	cmd.Flags().BoolP("disasm", "d", false, "Show disassembled bytecode (debug)")
	cmd.Flags().BoolP("trace", "t", false, "Enable execution tracing (shown on error)")
	// Set version used by './vega version'
	cmd.Version = fmt.Sprintf("%s.%s", info.Version, info.Commit)

	return cmd
}
