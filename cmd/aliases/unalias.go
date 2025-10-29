package aliases

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var unaliasCmd = &cobra.Command{
	Use:   "unalias <name>",
	Short: "Remove a version alias",
	Long: `Remove a version alias.

Examples:
  goenv unalias stable  # Remove the 'stable' alias`,
	RunE: runUnalias,
}

func init() {
	cmdpkg.RootCmd.AddCommand(unaliasCmd)
	helptext.SetCommandHelp(unaliasCmd)
}

func runUnalias(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: goenv unalias <name>")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	name := args[0]
	if err := mgr.DeleteAlias(name); err != nil {
		return err
	}

	// Silent success (consistent with other goenv commands)
	return nil
}
