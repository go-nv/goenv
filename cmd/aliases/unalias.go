package aliases

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/helptext"
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
	if err := cmdutil.ValidateExactArgs(args, 1, "name"); err != nil {
		return fmt.Errorf("usage: goenv unalias <name>")
	}

	_, mgr := cmdutil.SetupContext()

	name := args[0]
	if err := mgr.DeleteAlias(name); err != nil {
		return err
	}

	// Silent success (consistent with other goenv commands)
	return nil
}
