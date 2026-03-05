package cli

import (
	"fmt"
	"os"
	"strings"

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
			uri := "ephemeral://"
			if len(args) == 1 {
				uri = strings.TrimSpace(args[0])
			}

			fs, err := createVirtualFileSystem(uri)
			if err != nil {
				return err
			}

			disasm, _ := cmd.Flags().GetBool("disasm")

			vm := vm.NewVM(fs)
			vm.Stdin(os.Stdin)
			vm.Stdout(os.Stdout)
			vm.Stderr(os.Stderr)

			defer fs.Shutdown(cmd.Context())
			return repl.RunTUI(vm, disasm)
		},
	}

	cmd.Flags().BoolP("disasm", "d", false, "Show disassembled bytecode (debug)")
	cmd.Version = fmt.Sprintf("%s.%s", info.Version, info.Commit)

	return cmd
}
