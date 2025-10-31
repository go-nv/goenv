package tools

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var (
	installAllVersions bool
	installDryRun      bool
	installVerbose     bool
)

var installToolsCmd = &cobra.Command{
	Use:   "install <package>[@version]...",
	Short: "Install Go tools for current or all versions",
	Long: `Install Go tools using 'go install' with proper version isolation.

By default, installs to the current Go version's GOPATH/bin.
Use --all to install across all installed Go versions.

Examples:
  # Install for current Go version
  goenv tools install golang.org/x/tools/cmd/goimports@latest

  # Install for all Go versions
  goenv tools install gopls@latest --all

  # Install multiple tools at once
  goenv tools install gopls@latest staticcheck@latest golangci-lint@latest

  # Install across all versions with preview
  goenv tools install gopls@latest --all --dry-run

Common tools:
  - golang.org/x/tools/gopls              (Go language server)
  - golang.org/x/tools/cmd/goimports      (import formatting)
  - github.com/golangci/golangci-lint/cmd/golangci-lint  (linting)
  - honnef.co/go/tools/cmd/staticcheck    (static analysis)
  - github.com/go-delve/delve/cmd/dlv     (debugger)
  - mvdan.cc/gofumpt                       (stricter gofmt)`,
	Args: cobra.MinimumNArgs(1),
	RunE: runInstall,
}

func init() {
	installToolsCmd.Flags().BoolVar(&installAllVersions, "all", false, "Install across all installed Go versions")
	installToolsCmd.Flags().BoolVar(&installDryRun, "dry-run", false, "Show what would be installed without installing")
	installToolsCmd.Flags().BoolVarP(&installVerbose, "verbose", "v", false, "Show detailed output")
}

func runInstall(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Normalize package paths (add @latest if missing)
	packages := normalizePackagePaths(args)

	// Determine target versions
	var targetVersions []string
	if installAllVersions {
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
		currentVersion, source, err := mgr.GetCurrentVersion()
		if err != nil {
			return fmt.Errorf("no Go version set: %w", err)
		}
		if currentVersion == "system" {
			return fmt.Errorf("cannot install tools for system Go - please use a goenv-managed version")
		}
		if err := mgr.ValidateVersion(currentVersion); err != nil {
			return fmt.Errorf("go version %s not installed (set by %s)", currentVersion, source)
		}
		targetVersions = []string{currentVersion}
	}

	// Show installation plan
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n\n",
		utils.EmojiOr("📦 ", ""),
		utils.BoldBlue("Installation Plan"))

	fmt.Fprintf(cmd.OutOrStdout(), "Tools to install: %s\n", utils.Cyan(strings.Join(extractToolNames(packages), ", ")))
	fmt.Fprintf(cmd.OutOrStdout(), "Target versions:  %s\n", utils.Cyan(strings.Join(targetVersions, ", ")))
	fmt.Fprintf(cmd.OutOrStdout(), "Total operations: %s\n\n",
		utils.Yellow(fmt.Sprintf("%d", len(packages)*len(targetVersions))))

	if installDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%s Dry run mode - no changes made\n",
			utils.EmojiOr("ℹ️  ", ""))
		return nil
	}

	// Execute installations
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n",
		utils.EmojiOr("🔧 ", ""),
		utils.BoldBlue("Installing Tools"))

	successCount := 0
	failureCount := 0

	for _, version := range targetVersions {
		if len(targetVersions) > 1 {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
				utils.BoldCyan("Go "+version+":"),
				utils.Gray("("+filepath.Join(cfg.Root, "versions", version)+")"))
		}

		for _, pkg := range packages {
			toolName := extractToolName(pkg)

			if installVerbose {
				fmt.Fprintf(cmd.OutOrStdout(), "  Installing %s... ", utils.Cyan(toolName))
			}

			err := installToolForVersion(cfg, version, pkg, installVerbose)
			if err != nil {
				if installVerbose {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.Red("✗"))
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "  %s Failed to install %s for Go %s: %v\n",
					utils.Red("✗"), toolName, version, err)
				failureCount++
			} else {
				if installVerbose {
					fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.Green("✓"))
				} else {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s Installed %s\n",
						utils.Green("✓"), utils.BoldWhite(toolName))
				}
				successCount++
			}
		}

		if len(targetVersions) > 1 {
			fmt.Fprintln(cmd.OutOrStdout())
		}
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout())
	if failureCount == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%s Successfully installed %d tool(s) across %d version(s)\n",
			utils.EmojiOr("✅ ", ""),
			len(packages), len(targetVersions))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s Installed %d tool(s), %d failed\n",
			utils.EmojiOr("⚠️  ", ""),
			successCount, failureCount)
		return fmt.Errorf("%d installation(s) failed", failureCount)
	}

	if !installAllVersions && len(packages) == 1 {
		// Show usage hint for single tool, single version
		toolName := extractToolName(packages[0])
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sUsage:\n", utils.Emoji("💡 "))
		fmt.Fprintf(cmd.OutOrStdout(), "   %s [args...]  # Automatically uses the right version\n", toolName)
	}

	return nil
}
