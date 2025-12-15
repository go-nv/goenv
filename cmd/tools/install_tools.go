package tools

import (
	"fmt"
	"path/filepath"
	"slices"
	"strings"

	"github.com/go-nv/goenv/cmd/shims"
	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	toolspkg "github.com/go-nv/goenv/internal/tools"
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
  - gotest.tools/gotestsum                (test runner with nice output)
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
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager
	toolsMgr := toolspkg.NewManager(cfg, mgr)

	// Normalize package paths (add @latest if missing)
	packages := toolspkg.NormalizePackagePaths(args)

	// Determine target versions
	var targetVersions []string
	if installAllVersions {
		versions, err := mgr.ListInstalledVersions()
		if err != nil {
			return errors.FailedTo("list installed versions", err)
		}
		if len(versions) == 0 {
			return errors.NoVersionsInstalled()
		}
		targetVersions = versions
	} else {
		currentVersion, _, source, err := mgr.GetCurrentVersionResolved()
		if err != nil {
			return errors.FailedTo("determine Go version", err)
		}
		if currentVersion == manager.SystemVersion {
			return fmt.Errorf("cannot install tools for system Go - please use a goenv-managed version")
		}
		if err := mgr.ValidateVersion(currentVersion); err != nil {
			return errors.VersionNotInstalled(currentVersion, source)
		}
		targetVersions = []string{currentVersion}
	}

	// Show installation plan
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s %s\n\n",
		utils.EmojiOr("📦 ", ""),
		utils.BoldBlue("Installation Plan"))

	fmt.Fprintf(cmd.OutOrStdout(), "Tools to install: %s\n", utils.Cyan(strings.Join(toolspkg.ExtractToolNames(packages), ", ")))
	fmt.Fprintf(cmd.OutOrStdout(), "Target versions:  %s\n", utils.Cyan(strings.Join(targetVersions, ", ")))
	fmt.Fprintf(cmd.OutOrStdout(), "Total operations: %s\n\n",
		utils.Yellow(fmt.Sprintf("%d", len(packages)*len(targetVersions))))

	if installDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%s Dry run mode - no changes made\n",
			utils.EmojiOr("ℹ️  ", ""))
		return nil
	}

	// Execute installations using tools.Manager
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n",
		utils.EmojiOr("🔧 ", ""),
		utils.BoldBlue("Installing Tools"))

	opts := toolspkg.InstallOptions{
		Packages: packages,
		Versions: targetVersions,
		DryRun:   false, // Already checked above
		Verbose:  installVerbose,
	}

	result, err := toolsMgr.Install(opts)
	if err != nil {
		return errors.FailedTo("install tools", err)
	}

	// Display results per version
	successCount := 0
	failureCount := 0

	for _, version := range targetVersions {
		if len(targetVersions) > 1 {
			fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n",
				utils.BoldCyan("Go "+version+":"),
				utils.Gray("("+filepath.Join(cfg.Root, "versions", version)+")"))
		}

		for _, pkg := range packages {
			toolName := toolspkg.ExtractToolName(pkg)
			installedKey := fmt.Sprintf("%s@%s", toolName, version)
			failedKey := fmt.Sprintf("%s@%s", toolName, version)

			if slices.Contains(result.Installed, installedKey) {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s Installed %s\n",
					utils.Green("✓"), utils.BoldWhite(toolName))
				successCount++
			} else if slices.Contains(result.Failed, failedKey) {
				// Find the error
				for i, failed := range result.Failed {
					if failed == failedKey && i < len(result.Errors) {
						errMsg := result.Errors[i].Error()
						fmt.Fprintf(cmd.ErrOrStderr(), "  %s Failed to install %s: %v\n",
							utils.Red("✗"), toolName, result.Errors[i])

						// Provide helpful hint for common errors
						if strings.Contains(errMsg, "missing dot in first path element") {
							fmt.Fprintf(cmd.ErrOrStderr(), "    %s Hint: Use the full package path, e.g., gotest.tools/gotestsum@latest\n",
								utils.EmojiOr("💡 ", ""))
						}
						break
					}
				}
				failureCount++
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

	// Trigger rehash if tools were installed for current version
	if successCount > 0 {
		currentVersion, _, err := mgr.GetCurrentVersion()
		if err == nil && slices.Contains(targetVersions, currentVersion) {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%sRehashing shims...\n", utils.Emoji("🔄 "))
			if err := shims.RunRehash(cmd, []string{}); err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "Warning: Failed to rehash: %v\n", err)
			}
		}
	}

	if !installAllVersions && len(packages) == 1 {
		// Show usage hint for single tool, single version
		toolName := toolspkg.ExtractToolName(packages[0])
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sUsage:\n", utils.Emoji("💡 "))
		fmt.Fprintf(cmd.OutOrStdout(), "   %s [args...]  # Automatically uses the right version\n", toolName)
	}

	return nil
}
