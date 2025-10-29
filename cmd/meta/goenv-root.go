package meta

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"
	"github.com/go-nv/goenv/internal/config"
	"github.com/spf13/cobra"
)

var goenvRootCmd = &cobra.Command{
	Use:          "root",
	Short:        "Display the root directory where versions are installed",
	Long:         "Print the value of GOENV_ROOT, the directory where Go versions are installed",
	Args:         cobra.NoArgs,
	RunE:         runGoenvRoot,
	SilenceUsage: true,
}

func init() {
	cmdpkg.RootCmd.AddCommand(goenvRootCmd)
}

func runGoenvRoot(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	fmt.Fprintln(cmd.OutOrStdout(), cfg.Root)
	return nil
}
