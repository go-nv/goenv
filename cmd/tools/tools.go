package tools

import (
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:     "tools",
	Short:   "Manage Go tools per version",
	GroupID: string(cmdpkg.GroupTools),
	Long: `Install and manage Go tools on a per-version basis.

This ensures tools are properly isolated per Go version and prevents
accidental global installations.

Tool Installation:
  install      Install tools for current or all Go versions
  uninstall    Remove tools from current or all Go versions
  list         List installed tools (supports --all)

Tool Management:
  status       View tool consistency across versions
  outdated     Check for tool updates
  update       Update installed tools to latest versions

Version Sync:
  sync-tools   Copy tools from one version to another
  default      Set default tool installation behavior

Examples:
  # Install across all versions
  goenv tools install gopls@latest --all

  # List tools everywhere
  goenv tools list --all

  # Check what's outdated
  goenv tools outdated

  # View consistency
  goenv tools status

  # Clean up
  goenv tools uninstall gopls --all`,
}

func init() {
	cmdpkg.RootCmd.AddCommand(toolsCmd)

	// Add subcommands (each defined in their own file)
	toolsCmd.AddCommand(installToolsCmd) // from install_tools.go
	toolsCmd.AddCommand(listToolsCmd)    // from list_tools.go
	toolsCmd.AddCommand(updateToolsCmd)  // from update_tools.go
	toolsCmd.AddCommand(syncToolsCmd)    // from sync_tools.go
	toolsCmd.AddCommand(defaultToolsCmd) // from default_tools.go

	// Add uninstall command
	cfg, _ := cmdutil.SetupContext()
	toolsCmd.AddCommand(NewUninstallCommand(cfg)) // from uninstall_tools.go

	// Add new subcommands
	toolsCmd.AddCommand(newOutdatedCommand()) // from outdated.go
	toolsCmd.AddCommand(newStatusCommand())   // from status.go

	// Set help text
	helptext.SetCommandHelp(toolsCmd)
	helptext.SetCommandHelp(installToolsCmd)
	helptext.SetCommandHelp(listToolsCmd)
}

// extractToolName extracts the binary name from a package path
// e.g., "golang.org/x/tools/cmd/goimports@latest" -> "goimports"
// This function is kept here for backwards compatibility with other files
func extractToolName(packagePath string) string {
	// Remove @version suffix
	if idx := strings.Index(packagePath, "@"); idx != -1 {
		packagePath = packagePath[:idx]
	}

	// Get last component
	parts := strings.Split(packagePath, "/")
	if len(parts) > 0 {
		return parts[len(parts)-1]
	}

	return packagePath
}
