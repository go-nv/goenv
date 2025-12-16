package diagnostics

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var statusCmd = &cobra.Command{
	Use:     "status",
	Short:   "Show goenv status and configuration",
	GroupID: string(cmdpkg.GroupGettingStarted),
	Long: `Show a quick overview of your goenv installation and configuration.

This displays:
  - Initialization status (shell integration)
  - Current Go version and source
  - Installed versions count
  - Shims status
  - Configuration settings

Similar to 'git status' - provides a quick health check at a glance.`,
	Example: `  # Show current status
  goenv status

  # Check if properly initialized
  goenv status | grep initialized`,
	RunE: runStatus,
}

func init() {
	cmdpkg.RootCmd.AddCommand(statusCmd)
}

func runStatus(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager
	env := ctx.Environment
	
	// Fallback: Load environment if not already in context (e.g., in tests)
	if env == nil {
		env, _ = utils.LoadEnvironment(cmd.Context())
	}

	// Header
	fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", utils.Emoji("📊 "), utils.BoldBlue("goenv Status"))
	fmt.Fprintln(cmd.OutOrStdout())

	// Check 1: Initialization status
	goenvShell := env.GetShell()
	goenvRoot := env.GetRoot()

	if goenvShell != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", utils.Green("✓"), utils.Green("goenv is initialized"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Shell: %s\n", utils.Cyan(goenvShell))
		if goenvRoot != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Root: %s\n", utils.Gray(goenvRoot))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "  Root: %s %s\n", utils.Gray(cfg.Root), utils.Gray("(from config)"))
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", utils.Red("✗"), utils.Red("goenv is not initialized in this shell"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Run: %s\n", utils.Yellow("eval \"$(goenv init -)\""))
		fmt.Fprintln(cmd.OutOrStdout())
		return nil
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check 2: Current version
	resolvedVersion, versionSpec, source, err := mgr.GetCurrentVersionResolved()
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s\n", utils.Gray("none (not set)"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Set with: %s\n", utils.Yellow("goenv global <version>"))
	} else {
		// Display resolved version (already validated as installed by GetCurrentVersionResolved)
		installed := ""
		if resolvedVersion != manager.SystemVersion {
			installed = utils.Green(" ✓")
		}
		// Show both spec and resolved if different
		displayVersion := resolvedVersion
		if versionSpec != resolvedVersion && versionSpec != manager.SystemVersion {
			displayVersion = fmt.Sprintf("%s (resolved from %s)", resolvedVersion, versionSpec)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "Current version: %s%s\n", utils.BoldBlue(displayVersion), installed)

		// Show source
		sourceDisplay := source
		if strings.HasPrefix(source, cfg.Root) {
			// Make path relative for cleaner display
			relPath, err := filepath.Rel(cfg.Root, source)
			if err == nil && !strings.HasPrefix(relPath, "..") {
				sourceDisplay = "$GOENV_ROOT/" + relPath
			}
		}
		// Abbreviate home directory
		if homeDir, err := os.UserHomeDir(); err == nil {
			sourceDisplay = strings.Replace(sourceDisplay, homeDir, "~", 1)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  Set by: %s\n", utils.Gray(sourceDisplay))
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check 3: Installed versions
	versions, err := mgr.ListInstalledVersions()
	if err != nil || len(versions) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "Installed versions: %s\n", utils.Yellow("0"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Install with: %s\n", utils.Yellow("goenv install"))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Installed versions: %s\n", utils.Green(fmt.Sprintf("%d", len(versions))))

		// Show up to 5 versions
		displayCount := len(versions)
		if displayCount > 5 {
			displayCount = 5
		}

		for i := 0; i < displayCount; i++ {
			v := versions[i]
			marker := "  "
			if v == versionSpec {
				marker = utils.Cyan("→ ") // Current version
			}
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", marker, utils.Blue(v))
		}

		if len(versions) > 5 {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", utils.Gray(fmt.Sprintf("... and %d more", len(versions)-5)))
		}
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check 4: Shims
	shimsDir := cfg.ShimsDir()
	if utils.DirExists(shimsDir) {
		entries, err := os.ReadDir(shimsDir)
		if err == nil {
			shimCount := 0
			for _, entry := range entries {
				if !entry.IsDir() {
					shimCount++
				}
			}

			if shimCount > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "Shims: %s available\n", utils.Green(fmt.Sprintf("%d", shimCount)))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "Shims: %s\n", utils.Yellow("none (run 'goenv rehash')"))
			}
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Shims: %s\n", utils.Red("directory not found"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Run: %s\n", utils.Yellow("goenv rehash"))
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check 5: Auto-rehash setting
	autoRehash := env.HasAutoRehash()
	if autoRehash {
		fmt.Fprintf(cmd.OutOrStdout(), "Auto-rehash: %s\n", utils.Green("✓ enabled"))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Auto-rehash: %s\n", utils.Gray("disabled"))
		fmt.Fprintf(cmd.OutOrStdout(), "  Enable with: %s\n", utils.Yellow("export GOENV_AUTO_REHASH=1"))
	}

	// Check 6: Auto-install setting
	autoInstall := env.HasAutoInstall()
	if autoInstall {
		fmt.Fprintf(cmd.OutOrStdout(), "Auto-install: %s\n", utils.Green("✓ enabled"))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Auto-install: %s\n", utils.Gray("disabled"))
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.Gray("Run 'goenv doctor' for detailed diagnostics"))

	return nil
}
