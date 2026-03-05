package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/mwantia/vega/pkg/compiler"
	"github.com/mwantia/vega/pkg/vm"
	"github.com/spf13/cobra"
)

func NewRunCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run <path> <uri>",
		Short: "Vega - Virtual Execution & Graph Abstraction",
		Long: `Vega is a lightweight scripting language and runtime for VFS operations.

It provides a programmable interface between the host OS filesystem and
VFS-mounted storage backends (SQLite, S3, PostgreSQL, ephemeral, etc.)`,
		SilenceErrors: false,
		SilenceUsage:  true,
		Args:          cobra.MaximumNArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) < 1 {
				return fmt.Errorf("expected at least 1 argument, got %d", len(args))
			}

			file := strings.TrimSpace(args[0])
			uri := "ephemeral://"
			if len(args) > 1 {
				uri = strings.TrimSpace(args[1])
			}

			fs, err := createVirtualFileSystem(uri)
			if err != nil {
				return err
			}
			defer fs.Shutdown(cmd.Context())

			buf, err := os.ReadFile(file)
			if err != nil {
				return fmt.Errorf("failed to read file: %v", err)
			}

			var bytecode *compiler.ByteCode
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

			if bytecode == nil {
				return fmt.Errorf("no bytecode compiled as source")
			}

			if disasm, _ := cmd.Flags().GetBool("disasm"); disasm {
				fmt.Println(bytecode.Disassemble())
				fmt.Println("=== Execution ===")
			}

			vm := vm.NewVM(fs)
			vm.Stdin(os.Stdin)
			vm.Stdout(os.Stdout)
			vm.Stderr(os.Stderr)

			if _, err := vm.Run(cmd.Context(), bytecode); err != nil {
				return fmt.Errorf("runtime error: %w", err)
			}

			return nil
		},
	}

	cmd.Flags().BoolP("disasm", "d", false, "Show disassembled bytecode (debug)")

	return cmd
}
