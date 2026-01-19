// Command vega is the Vega scripting language interpreter.
package main

import (
	"fmt"
	"os"

	"github.com/mwantia/vega/cmd/vega/cli"

	// Import VFS drivers
	_ "github.com/mwantia/vfs/mount/service/consul"
	_ "github.com/mwantia/vfs/mount/service/ephemeral"
	_ "github.com/mwantia/vfs/mount/service/file"
	_ "github.com/mwantia/vfs/mount/service/postgres"
	_ "github.com/mwantia/vfs/mount/service/s3"
	_ "github.com/mwantia/vfs/mount/service/sqlite"
)

var (
	version = "0.0.1-dev"
	commit  = "main"
)

func main() {
	root := cli.NewRootCommand(cli.VersionInfo{
		Version: version,
		Commit:  commit,
	})
	root.AddCommand(cli.NewVersionCommand())

	if err := root.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
