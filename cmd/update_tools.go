package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/tooldetect"
	"github.com/spf13/cobra"
)

var updateToolsCmd = &cobra.Command{
	Use:     "update-tools",
	Short:   "Update installed Go tools to their latest versions",
	GroupID: "tools",
	Long: `Updates all installed Go tools to their latest versions.

This command checks all tools installed in the current Go version's GOPATH
and updates them to the latest available versions from the Go module proxy.

Examples:
  goenv update-tools                    # Update all tools to latest
  goenv update-tools --check            # Check for updates without installing
  goenv update-tools --tool gopls       # Update only gopls
  goenv update-tools --version v0.12.0  # Update to specific version
  goenv update-tools --dry-run          # Show what would be updated`,
	RunE: runUpdateTools,
}

var updateToolsFlags struct {
	check   bool
	tool    string
	dryRun  bool
	version string
}

func init() {
	updateToolsCmd.Flags().BoolVarP(&updateToolsFlags.check, "check", "c", false, "Check for updates without installing")
	updateToolsCmd.Flags().StringVarP(&updateToolsFlags.tool, "tool", "t", "", "Update only the specified tool")
	updateToolsCmd.Flags().BoolVarP(&updateToolsFlags.dryRun, "dry-run", "n", false, "Show what would be updated without actually updating")
	updateToolsCmd.Flags().StringVar(&updateToolsFlags.version, "version", "latest", "Target version (default: latest)")
	rootCmd.AddCommand(updateToolsCmd)
}

func runUpdateTools(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Get current Go version
	goVersion, _, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("no Go version set: %w", err)
	}

	if goVersion == "system" {
		return fmt.Errorf("cannot update tools for system Go version")
	}

	// Validate version is installed
	if err := mgr.ValidateVersion(goVersion); err != nil {
		return fmt.Errorf("Go version %s is not installed", goVersion)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "🔄 Checking for tool updates in Go %s...\n", goVersion)
	fmt.Fprintln(cmd.OutOrStdout())

	// List installed tools
	tools, err := tooldetect.ListInstalledTools(cfg.Root, goVersion)
	if err != nil {
		return fmt.Errorf("failed to list tools: %w", err)
	}

	if len(tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No Go tools installed yet.")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv default-tools init' to set up automatic tool installation.")
		return nil
	}

	// Filter to specific tool if requested
	if updateToolsFlags.tool != "" {
		filtered := []tooldetect.Tool{}
		for _, t := range tools {
			if t.Name == updateToolsFlags.tool {
				filtered = append(filtered, t)
				break
			}
		}
		if len(filtered) == 0 {
			return fmt.Errorf("tool '%s' not found", updateToolsFlags.tool)
		}
		tools = filtered
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Found %d tool(s):\n", len(tools))
	fmt.Fprintln(cmd.OutOrStdout())

	// Check for updates
	type UpdateInfo struct {
		Tool          tooldetect.Tool
		LatestVersion string
		NeedsUpdate   bool
	}

	var updates []UpdateInfo

	for _, tool := range tools {
		// Skip tools without package path
		if tool.PackagePath == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s (unknown package path, skipping)\n", tool.Name)
			continue
		}

		// Query latest version
		latestVersion, err := tooldetect.GetLatestVersion(tool.PackagePath)
		if err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s (%s) - failed to check: %v\n", tool.Name, tool.Version, err)
			continue
		}

		// Compare versions
		needsUpdate := tooldetect.CompareVersions(tool.Version, latestVersion) < 0

		if needsUpdate {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s (%s) → %s available ⬆️\n", tool.Name, tool.Version, latestVersion)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s (%s) - up to date ✅\n", tool.Name, tool.Version)
		}

		if needsUpdate {
			updates = append(updates, UpdateInfo{
				Tool:          tool,
				LatestVersion: latestVersion,
				NeedsUpdate:   true,
			})
		}
	}

	if len(updates) == 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "✅ All tools are up to date!")
		return nil
	}

	// If check-only mode, stop here
	if updateToolsFlags.check {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "💡 Run 'goenv update-tools' to update %d tool(s)\n", len(updates))
		return nil
	}

	// If dry-run mode, show what would be updated
	if updateToolsFlags.dryRun {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "🔍 Dry run - would update:")
		for _, update := range updates {
			// Determine target version
			targetVersion := updateToolsFlags.version
			if targetVersion == "" || targetVersion == "latest" {
				targetVersion = update.LatestVersion
			}

			fmt.Fprintf(cmd.OutOrStdout(), "  • %s: %s → %s\n",
				update.Tool.Name, update.Tool.Version, targetVersion)
		}
		return nil
	}

	// Perform updates
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "📦 Updating tools...")
	fmt.Fprintln(cmd.OutOrStdout())

	versionPath := filepath.Join(cfg.Root, "versions", goVersion)
	goRoot := filepath.Join(versionPath, "go")
	goBin := filepath.Join(goRoot, "bin", "go")

	successCount := 0
	failCount := 0

	for _, update := range updates {
		fmt.Fprintf(cmd.OutOrStdout(), "  Updating %s...", update.Tool.Name)

		// Determine target version: use flag value or latest
		targetVersion := updateToolsFlags.version
		if targetVersion == "" || targetVersion == "latest" {
			targetVersion = update.LatestVersion
		}

		// Build package reference with target version
		pkg := update.Tool.PackagePath + "@" + targetVersion

		// Run go install
		installCmd := exec.Command(goBin, "install", pkg)
		installCmd.Env = append(os.Environ(),
			"GOROOT="+goRoot,
			"GOPATH="+filepath.Join(versionPath, "gopath"),
		)

		if err := installCmd.Run(); err != nil {
			fmt.Fprintln(cmd.OutOrStdout(), " ❌")
			failCount++
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), " ✅")
			successCount++
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	if successCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "✅ Updated %d tool(s) successfully\n", successCount)
	}
	if failCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "❌ Failed to update %d tool(s)\n", failCount)
		return fmt.Errorf("%d tool(s) failed to update", failCount)
	}

	return nil
}
