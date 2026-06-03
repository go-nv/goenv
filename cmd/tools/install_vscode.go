package tools

import (
	"fmt"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var installVSCodeCmd = &cobra.Command{
	Use:   "install-vscode [version]",
	Short: "Install VSCode Go extension tools for a Go version",
	Long: `Installs all tools required by the VSCode Go extension for a specific Go version.

This command installs the following tools as documented at:
https://github.com/golang/vscode-go/wiki/tools

Tools installed:
  - gopls         (Go language server)
  - dlv           (Delve debugger)
  - vscgo         (VSCode Go utilities)
  - goplay        (Go Playground support)
  - gomodifytags  (Struct tag editor)
  - impl          (Interface stub generator)
  - gotests       (Test generator)
  - staticcheck   (Static analysis tool)

These tools provide the full IntelliSense, debugging, and code editing
capabilities of the VSCode Go extension.

Examples:
  # Install VSCode tools for current version
  goenv tools install-vscode

  # Install VSCode tools for specific version
  goenv tools install-vscode 1.25.2

  # Show what would be installed
  goenv tools install-vscode --dry-run`,
	Args: cobra.RangeArgs(0, 1),
	RunE: runInstallVSCode,
}

var installVSCodeFlags struct {
	dryRun  bool
	verbose bool
}

func init() {
	installVSCodeCmd.Flags().BoolVar(&installVSCodeFlags.dryRun, "dry-run", false, "Show what would be installed without installing")
	installVSCodeCmd.Flags().BoolVarP(&installVSCodeFlags.verbose, "verbose", "v", false, "Show detailed installation output")
}

func runInstallVSCode(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Get version from args or use current version
	var goVersion string
	if len(args) > 0 {
		goVersion = args[0]
	} else {
		// Use GetCurrentVersionResolved to handle partial versions (e.g., "1.25" → "1.25.10")
		var err error
		goVersion, _, _, err = mgr.GetCurrentVersionResolved()
		if err != nil {
			return fmt.Errorf("no Go version specified and no current version set. Use 'goenv global <version>' or specify a version")
		}
	}

	// Validate version is installed
	if !mgr.IsVersionInstalled(goVersion) {
		return fmt.Errorf("Go %s is not installed. Run 'goenv install %s' first", goVersion, goVersion)
	}

	versionPath := cfg.SafeResolvePath(goVersion)

	fmt.Fprintf(cmd.OutOrStdout(), "Installing VSCode Go extension tools for Go %s...\n", goVersion)
	fmt.Fprintln(cmd.OutOrStdout())

	if installVSCodeFlags.dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%sDry run mode - no tools will be installed\n", utils.Emoji("ℹ️  "))
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Build list of tools to install
	toolPackages := make([]string, 0, len(tools.VSCodeTools))
	for _, toolName := range tools.VSCodeTools {
		pkg := tools.NormalizePackagePath(toolName)
		toolPackages = append(toolPackages, pkg)
	}

	// Show what will be installed
	fmt.Fprintln(cmd.OutOrStdout(), "Tools to install:")
	for i, pkg := range toolPackages {
		toolName := tools.ExtractToolName(pkg)
		fmt.Fprintf(cmd.OutOrStdout(), "  %d. %-15s %s\n", i+1, toolName, pkg)
	}
	fmt.Fprintln(cmd.OutOrStdout())

	if installVSCodeFlags.dryRun {
		return nil
	}

	// Install VSCode tools
	if err := tools.InstallVSCodeToolsForVersion(goVersion, cfg.Root, versionPath, installVSCodeFlags.verbose); err != nil {
		return errors.FailedTo("install VSCode tools", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sVSCode tools installed successfully\n", utils.Emoji("✅ "))
	fmt.Fprintf(cmd.OutOrStdout(), "%sRun 'goenv rehash' to make tools available as shims\n", utils.Emoji("💡 "))

	return nil
}
