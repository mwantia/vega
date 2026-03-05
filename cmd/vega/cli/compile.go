package cli

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"
)

func NewCompileCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "compile <input> <output>",
		Short: "Vega - Virtual Execution & Graph Abstraction",
		Long: `Vega is a lightweight scripting language and runtime for VFS operations.

It provides a programmable interface between the host OS filesystem and
VFS-mounted storage backends (SQLite, S3, PostgreSQL, ephemeral, etc.)`,
		Args: cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			inPath := strings.TrimSpace(args[0])
			outPath := strings.TrimSpace(args[1])

			debug, _ := cmd.Flags().GetBool("debug")
			compress, _ := cmd.Flags().GetBool("compress")

			buf, err := os.ReadFile(inPath)
			if err != nil {
				return fmt.Errorf("failed to read input file: %w", err)
			}

			content := string(buf)
			bytecode, err := compileFileContent(content)
			if err != nil {
				return fmt.Errorf("failed to compile content: %v", err)
			}

			ser, err := bytecode.Serialize(debug, compress)
			if err != nil {
				return err
			}

			f, err := os.Create(outPath)
			if err != nil {
				return err
			}
			defer f.Close()

			_, err = f.Write(ser)
			return err
		},
	}

	cmd.Flags().BoolP("debug", "d", false, "Keep open after executing (default is 'false')")
	cmd.Flags().BoolP("compress", "c", false, "Keep open after executing (default is 'false')")

	return cmd
}
