package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/tooldetect"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var updateToolsCmd = &cobra.Command{
	Use:   "update-tools",
	Short: "Update installed Go tools to their latest versions",
	Long: `Updates all installed Go tools to their latest versions.

By default, updates tools in the current Go version's GOPATH.
Use --all to update tools across all installed Go versions.

Examples:
  goenv tools update                    # Update all tools to latest
  goenv tools update --all              # Update across all Go versions
  goenv tools update --check            # Check for updates without installing
  goenv tools update --tool gopls       # Update only gopls
  goenv tools update --version v0.12.0  # Update to specific version
  goenv tools update --dry-run          # Show what would be updated`,
	RunE: runUpdateTools,
}

var updateToolsFlags struct {
	check   bool
	tool    string
	dryRun  bool
	version string
	all     bool
}

func init() {
	// Now registered as subcommand in tools.go
	updateToolsCmd.Flags().BoolVarP(&updateToolsFlags.check, "check", "c", false, "Check for updates without installing")
	updateToolsCmd.Flags().StringVarP(&updateToolsFlags.tool, "tool", "t", "", "Update only the specified tool")
	updateToolsCmd.Flags().BoolVarP(&updateToolsFlags.dryRun, "dry-run", "n", false, "Show what would be updated without actually updating")
	updateToolsCmd.Flags().StringVar(&updateToolsFlags.version, "version", "latest", "Target version (default: latest)")
	updateToolsCmd.Flags().BoolVar(&updateToolsFlags.all, "all", false, "Update tools across all installed Go versions")
}

func runUpdateTools(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Determine target versions
	var targetVersions []string
	if updateToolsFlags.all {
		versions, err := getInstalledVersions(cfg)
		if err != nil {
			return err
		}
		if len(versions) == 0 {
			return fmt.Errorf("no Go versions installed")
		}
		targetVersions = versions
	} else {
		mgr := manager.NewManager(cfg)
		goVersion, _, err := mgr.GetCurrentVersion()
		if err != nil {
			return fmt.Errorf("no Go version set: %w", err)
		}
		if goVersion == "system" {
			return fmt.Errorf("cannot update tools for system Go version")
		}
		if err := mgr.ValidateVersion(goVersion); err != nil {
			return fmt.Errorf("go version %s is not installed", goVersion)
		}
		targetVersions = []string{goVersion}
	}

	// Process all target versions
	totalSuccess := 0
	totalFail := 0

	for versionIdx, goVersion := range targetVersions {
		if len(targetVersions) > 1 {
			if versionIdx > 0 {
				fmt.Fprintln(cmd.OutOrStdout())
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s Go %s\n", utils.Emoji("🔧"), goVersion)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%sChecking for tool updates in Go %s...\n", utils.Emoji("🔄 "), goVersion)
		fmt.Fprintln(cmd.OutOrStdout())

		// List installed tools
		tools, err := tooldetect.ListInstalledTools(cfg.Root, goVersion)
		if err != nil {
			return fmt.Errorf("failed to list tools for Go %s: %w", goVersion, err)
		}

		if len(tools) == 0 {
			fmt.Fprintln(cmd.OutOrStdout(), "No Go tools installed yet.")
			if len(targetVersions) == 1 {
				fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv default-tools init' to set up automatic tool installation.")
			}
			continue
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
				if len(targetVersions) == 1 {
					return fmt.Errorf("tool '%s' not found", updateToolsFlags.tool)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "Tool '%s' not found in Go %s\n", updateToolsFlags.tool, goVersion)
				continue
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
				fmt.Fprintf(cmd.OutOrStdout(), "  • %s (%s) → %s available %s\n", tool.Name, tool.Version, latestVersion, utils.Emoji("⬆️"))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %s (%s) - up to date %s\n", tool.Name, tool.Version, utils.Emoji("✅"))
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
			fmt.Fprintf(cmd.OutOrStdout(), "%sAll tools are up to date!\n", utils.Emoji("✅ "))
			continue
		}

		// If check-only mode, show and continue
		if updateToolsFlags.check {
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "%sRun 'goenv tools update' to update %d tool(s)\n", utils.Emoji("💡 "), len(updates))
			continue
		}

		// If dry-run mode, show what would be updated
		if updateToolsFlags.dryRun {
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "%sDry run - would update:\n", utils.Emoji("🔍 "))
			for _, update := range updates {
				// Determine target version
				targetVersion := updateToolsFlags.version
				if targetVersion == "" || targetVersion == "latest" {
					targetVersion = update.LatestVersion
				}

				fmt.Fprintf(cmd.OutOrStdout(), "  • %s: %s → %s\n",
					update.Tool.Name, update.Tool.Version, targetVersion)
			}
			continue
		}

		// Perform updates
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sUpdating tools...\n", utils.Emoji("📦 "))
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
				fmt.Fprintf(cmd.OutOrStdout(), " %s\n", utils.Emoji("❌"))
				failCount++
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), " %s\n", utils.Emoji("✅"))
				successCount++
			}
		}

		fmt.Fprintln(cmd.OutOrStdout())
		if successCount > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "%sUpdated %d tool(s) successfully\n", utils.Emoji("✅ "), successCount)
		}
		if failCount > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "%sFailed to update %d tool(s)\n", utils.Emoji("❌ "), failCount)
		}

		totalSuccess += successCount
		totalFail += failCount
	}

	// Summary for --all mode
	if len(targetVersions) > 1 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sSummary: Updated %d tool(s) across %d Go version(s)\n",
			utils.Emoji("📊 "), totalSuccess, len(targetVersions))
		if totalFail > 0 {
			return fmt.Errorf("%d tool(s) failed to update", totalFail)
		}
	} else if totalFail > 0 {
		return fmt.Errorf("%d tool(s) failed to update", totalFail)
	}

	return nil
}
