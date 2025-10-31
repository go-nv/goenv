package aliases

import (
	"fmt"
	"sort"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var aliasCmd = &cobra.Command{
	Use:   "alias [name] [version]",
	Short: "Create or list version aliases",
	Long: `Create or list version aliases.

With no arguments, lists all defined aliases.
With name and version, creates or updates an alias.
With just a name, shows the version for that alias.

Examples:
  goenv alias              # List all aliases
  goenv alias stable 1.23.0  # Create alias 'stable' -> 1.23.0
  goenv alias dev latest     # Create alias 'dev' -> latest
  goenv alias stable         # Show what 'stable' points to`,
	RunE: runAlias,
}

func init() {
	cmdpkg.RootCmd.AddCommand(aliasCmd)
	helptext.SetCommandHelp(aliasCmd)
}

func runAlias(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	switch len(args) {
	case 0:
		// List all aliases
		return listAliases(cmd, mgr)
	case 1:
		// Show specific alias
		return showAlias(cmd, mgr, args[0])
	case 2:
		// Set alias
		return setAlias(cmd, mgr, args[0], args[1])
	default:
		return fmt.Errorf("usage: goenv alias [name] [version]")
	}
}

func listAliases(cmd *cobra.Command, mgr *manager.Manager) error {
	aliases, err := mgr.ListAliases()
	if err != nil {
		return err
	}

	if len(aliases) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No aliases defined")
		return nil
	}

	// Sort alias names for consistent output
	names := make([]string, 0, len(aliases))
	for name := range aliases {
		names = append(names, name)
	}
	sort.Strings(names)

	// Print aliases
	for _, name := range names {
		fmt.Fprintf(cmd.OutOrStdout(), "%s -> %s\n", name, aliases[name])
	}

	return nil
}

func showAlias(cmd *cobra.Command, mgr *manager.Manager, name string) error {
	aliases, err := mgr.ListAliases()
	if err != nil {
		return err
	}

	if version, exists := aliases[name]; exists {
		fmt.Fprintln(cmd.OutOrStdout(), version)
		return nil
	}

	return fmt.Errorf("alias '%s' not found", name)
}

func setAlias(cmd *cobra.Command, mgr *manager.Manager, name, version string) error {
	if err := mgr.SetAlias(name, version); err != nil {
		return err
	}

	// Silent success (consistent with other goenv commands)
	return nil
}
