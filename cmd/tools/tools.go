package tools

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var toolsCmd = &cobra.Command{
	Use:     "tools",
	Short:   "Manage Go tools per version",
	GroupID: "tools",
	Long: `Install and manage Go tools on a per-version basis.

This ensures tools are properly isolated per Go version and prevents
accidental global installations.

Subcommands:
  install      Install a tool for the current Go version
  list         List installed tools for the current version
  update       Update installed tools to latest versions
  sync         Copy tools from one version to another
  default      Manage automatic tool installation

Examples:
  goenv tools install golang.org/x/tools/cmd/goimports@latest
  goenv tools list
  goenv tools update
  goenv tools sync 1.23.2 1.24.4
  goenv tools default list`,
}

var toolsInstallCmd = &cobra.Command{
	Use:   "install <package>[@version]",
	Short: "Install a Go tool for the current version",
	Long: `Install a Go tool using 'go install' with proper version isolation.

The tool will be installed to the current Go version's GOPATH/bin,
ensuring it's isolated from other Go versions.

Examples:
  goenv tools install golang.org/x/tools/cmd/goimports@latest
  goenv tools install github.com/golangci/golangci-lint/cmd/golangci-lint@v1.55.2
  goenv tools install honnef.co/go/tools/cmd/staticcheck@latest

Common tools:
  - golang.org/x/tools/cmd/goimports     (import formatting)
  - github.com/golangci/golangci-lint/cmd/golangci-lint  (linting)
  - honnef.co/go/tools/cmd/staticcheck   (static analysis)
  - github.com/go-delve/delve/cmd/dlv    (debugger)
  - mvdan.cc/gofumpt                      (stricter gofmt)`,
	Args: cobra.ExactArgs(1),
	RunE: runToolsInstall,
}

var toolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List installed tools for the current Go version",
	Long: `List all Go tools installed in the current version's GOPATH/bin.

This shows tools that were installed via 'go install' or 'goenv tools install'.

Flags:
  --json    Output results in JSON format`,
	RunE: runToolsList,
}

var toolsListJSON bool

func init() {
	cmdpkg.RootCmd.AddCommand(toolsCmd)
	toolsCmd.AddCommand(toolsInstallCmd)
	toolsCmd.AddCommand(toolsListCmd)
	toolsListCmd.Flags().BoolVar(&toolsListJSON, "json", false, "Output in JSON format")

	// Add update-tools as subcommand
	updateToolsCmd.Use = "update"
	updateToolsCmd.Short = "Update installed Go tools to their latest versions"
	updateToolsCmd.GroupID = "" // Clear GroupID for subcommand
	toolsCmd.AddCommand(updateToolsCmd)

	// Add sync-tools as subcommand
	syncToolsCmd.Use = "sync [source-version] [target-version]"
	syncToolsCmd.Short = "Sync/replicate installed Go tools between versions"
	syncToolsCmd.GroupID = "" // Clear GroupID for subcommand
	toolsCmd.AddCommand(syncToolsCmd)

	// Add default-tools as subcommand group
	defaultToolsCmd.Use = "default"
	defaultToolsCmd.Short = "Manage default tools installed with new Go versions"
	defaultToolsCmd.GroupID = "" // Clear GroupID for subcommand
	toolsCmd.AddCommand(defaultToolsCmd)

	helptext.SetCommandHelp(toolsCmd)
	helptext.SetCommandHelp(toolsInstallCmd)
	helptext.SetCommandHelp(toolsListCmd)
}

func runToolsInstall(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Get current Go version
	currentVersion, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("no Go version set: %w", err)
	}

	if currentVersion == "system" {
		return fmt.Errorf("cannot install tools for system Go - please use a goenv-managed version")
	}

	// Validate version is installed
	if err := mgr.ValidateVersion(currentVersion); err != nil {
		return fmt.Errorf("Go version %s not installed (set by %s)", currentVersion, source)
	}

	packagePath := args[0]

	fmt.Fprintf(cmd.OutOrStdout(), "%sInstalling %s for Go %s...\n", utils.Emoji("📦 "), packagePath, currentVersion)
	fmt.Fprintln(cmd.OutOrStdout())

	// Validate package path format
	if !strings.Contains(packagePath, "/") {
		return fmt.Errorf("invalid package path: %s\nExample: golang.org/x/tools/cmd/goimports@latest", packagePath)
	}

	// Check if @version is specified
	if !strings.Contains(packagePath, "@") {
		fmt.Fprintf(cmd.OutOrStdout(), "%sNo version specified, using @latest\n", utils.Emoji("⚠️  "))
		fmt.Fprintln(cmd.OutOrStdout(), "   Tip: Use @latest or @v1.2.3 for reproducible builds")
		fmt.Fprintln(cmd.OutOrStdout())
		packagePath += "@latest"
	}

	// Run 'go install' through goenv exec to ensure proper environment
	installCmd := exec.Command("goenv", "exec", "go", "install", packagePath)
	installCmd.Stdout = cmd.OutOrStdout()
	installCmd.Stderr = cmd.ErrOrStderr()
	installCmd.Stdin = os.Stdin

	if err := installCmd.Run(); err != nil {
		return fmt.Errorf("failed to install %s: %w", packagePath, err)
	}

	// Extract tool name from package path
	toolName := extractToolName(packagePath)

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sSuccessfully installed %s\n", utils.Emoji("✅ "), toolName)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sUsage:\n", utils.Emoji("💡 "))
	fmt.Fprintf(cmd.OutOrStdout(), "   %s [args...]  # Automatically uses the right version\n", toolName)
	fmt.Fprintf(cmd.OutOrStdout(), "   goenv exec %s [args...]  # Explicit\n", toolName)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   To use in another Go version:")
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv local <other-version>")
	fmt.Fprintf(cmd.OutOrStdout(), "   goenv tools install %s\n", packagePath)

	return nil
}

func runToolsList(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Get current Go version
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("no Go version set: %w", err)
	}

	if currentVersion == "system" {
		fmt.Fprintln(cmd.OutOrStdout(), "Cannot list tools for system Go")
		fmt.Fprintln(cmd.OutOrStdout(), "Use 'which <tool>' or check your system's $GOPATH/bin")
		return nil
	}

	// Validate version is installed
	if err := mgr.ValidateVersion(currentVersion); err != nil {
		return fmt.Errorf("Go version %s not installed", currentVersion)
	}

	// List tools using the existing tooldetect functionality
	// We'll exec 'goenv exec go env GOPATH' to get GOPATH/bin
	listCmd := exec.Command("goenv", "exec", "go", "env", "GOPATH")
	output, err := listCmd.Output()
	if err != nil {
		return fmt.Errorf("failed to get GOPATH: %w", err)
	}

	gopath := strings.TrimSpace(string(output))
	if gopath == "" {
		if toolsListJSON {
			// Empty JSON array for no tools
			fmt.Fprintln(cmd.OutOrStdout(), "[]")
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout(), "No GOPATH configured")
		return nil
	}

	// Get first GOPATH entry (in case of multiple)
	gopaths := strings.Split(gopath, string(os.PathListSeparator))
	binDir := gopaths[0] + "/bin"

	// List executables in bin directory
	entries, err := os.ReadDir(binDir)
	if err != nil {
		if os.IsNotExist(err) {
			if toolsListJSON {
				// Empty JSON array for no tools
				fmt.Fprintln(cmd.OutOrStdout(), "[]")
				return nil
			}
			fmt.Fprintln(cmd.OutOrStdout(), "No tools installed yet")
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Install a tool with:")
			fmt.Fprintln(cmd.OutOrStdout(), "  goenv tools install <package>@<version>")
			return nil
		}
		return fmt.Errorf("failed to read bin directory: %w", err)
	}

	// Collect tools
	type toolInfo struct {
		Name    string `json:"name"`
		Path    string `json:"path"`
		Version string `json:"version"`
	}

	type toolsListOutput struct {
		SchemaVersion string     `json:"schema_version"`
		GoVersion     string     `json:"go_version"`
		Tools         []toolInfo `json:"tools"`
	}

	var tools []toolInfo

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()

		// Skip hidden files
		if strings.HasPrefix(name, ".") {
			continue
		}

		tools = append(tools, toolInfo{
			Name:    name,
			Path:    binDir + "/" + name,
			Version: currentVersion,
		})
	}

	// Handle JSON output
	if toolsListJSON {
		output := toolsListOutput{
			SchemaVersion: "1",
			GoVersion:     currentVersion,
			Tools:         tools,
		}
		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
	}

	// Human-readable output
	fmt.Fprintf(cmd.OutOrStdout(), "%sTools installed for Go %s:\n\n", utils.Emoji("🔧 "), currentVersion)

	if len(tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tools installed yet")
	} else {
		for _, tool := range tools {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", tool.Name)
		}
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "Total: %d tool(s)\n", len(tools))
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sSync tools to another version:\n", utils.Emoji("💡 "))
		fmt.Fprintln(cmd.OutOrStdout(), "   goenv tools sync [source-version]")
	}

	return nil
}

// extractToolName extracts the binary name from a package path
// e.g., "golang.org/x/tools/cmd/goimports@latest" -> "goimports"
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
