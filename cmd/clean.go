package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var cleanCmd = &cobra.Command{
	Use:     "clean [target]",
	Short:   "Clean Go build and module caches",
	GroupID: "common",
	Long: `Clean Go build and module caches to fix version mismatch issues.

When switching between Go versions, cached build artifacts can cause
"version mismatch" errors. This command helps fix those issues by
clearing the problematic caches.

Available targets:
  build      - Clear Go build cache only (safe, recommended)
  modcache   - Clear module download cache (downloads will be re-fetched)
  all        - Clear both build and module caches

If no target is specified, defaults to 'build'.

Examples:
  goenv clean              # Clear build cache only
  goenv clean build        # Clear build cache only
  goenv clean modcache     # Clear module cache only
  goenv clean all          # Clear both caches`,
	RunE:              runClean,
	ValidArgsFunction: completeCleanTargets,
}

var cleanFlags struct {
	force   bool
	verbose bool
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVarP(&cleanFlags.force, "force", "f", false, "Skip confirmation prompts")
	cleanCmd.Flags().BoolVarP(&cleanFlags.verbose, "verbose", "v", false, "Show detailed output")
	helptext.SetCommandHelp(cleanCmd)
}

func completeCleanTargets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{"build", "modcache", "all"}, cobra.ShellCompDirectiveNoFileComp
}

func runClean(cmd *cobra.Command, args []string) error {
	// Determine target
	target := "build"
	if len(args) > 0 {
		target = args[0]
	}

	// Validate target
	validTargets := map[string]bool{
		"build":    true,
		"modcache": true,
		"all":      true,
	}

	if !validTargets[target] {
		return fmt.Errorf("invalid target '%s'. Valid targets: build, modcache, all", target)
	}

	if len(args) > 1 {
		return fmt.Errorf("too many arguments. Usage: goenv clean [build|modcache|all]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Get current Go version
	version, _, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("cannot determine current Go version: %w", err)
	}

	if version == "" {
		return fmt.Errorf("no Go version is currently set. Use 'goenv global' or 'goenv local' to set a version")
	}

	// Show what will be cleaned
	fmt.Fprintln(cmd.OutOrStdout(), "🧹 Go Cache Cleaner")
	fmt.Fprintln(cmd.OutOrStdout())

	shouldCleanBuild := target == "build" || target == "all"
	shouldCleanModCache := target == "modcache" || target == "all"

	if shouldCleanBuild && shouldCleanModCache {
		fmt.Fprintf(cmd.OutOrStdout(), "This will clean:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  • Go build cache\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  • Go module download cache\n")
	} else if shouldCleanBuild {
		fmt.Fprintf(cmd.OutOrStdout(), "This will clean the Go build cache.\n")
	} else if shouldCleanModCache {
		fmt.Fprintf(cmd.OutOrStdout(), "This will clean the Go module download cache.\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Note: Modules will need to be re-downloaded when needed.\n")
	}

	fmt.Fprintln(cmd.OutOrStdout())

	// Get confirmation unless --force
	if !cleanFlags.force && shouldCleanModCache {
		fmt.Fprint(cmd.OutOrStdout(), "⚠️  Module cache cleanup will require re-downloading dependencies. Continue? [y/N]: ")
		var response string
		fmt.Scanln(&response)
		if response != "y" && response != "Y" && response != "yes" {
			fmt.Fprintln(cmd.OutOrStdout(), "Cancelled.")
			return nil
		}
	}

	// Build the go binary path
	goBinary, err := getGoBinary(cfg, version)
	if err != nil {
		return err
	}

	cleaned := []string{}
	failed := []string{}

	// Clean build cache
	if shouldCleanBuild {
		if cleanFlags.verbose {
			fmt.Fprintln(cmd.OutOrStdout(), "→ Cleaning build cache...")
		}

		cleanCmd := exec.Command(goBinary, "clean", "-cache")
		if cleanFlags.verbose {
			cleanCmd.Stdout = cmd.OutOrStdout()
			cleanCmd.Stderr = cmd.ErrOrStderr()
		}

		if err := cleanCmd.Run(); err != nil {
			failed = append(failed, "build cache")
			if cleanFlags.verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ Failed to clean build cache: %v\n", err)
			}
		} else {
			cleaned = append(cleaned, "build cache")
			if cleanFlags.verbose {
				fmt.Fprintln(cmd.OutOrStdout(), "  ✓ Build cache cleaned")
			}
		}
	}

	// Clean module cache
	if shouldCleanModCache {
		if cleanFlags.verbose {
			fmt.Fprintln(cmd.OutOrStdout(), "→ Cleaning module cache...")
		}

		cleanCmd := exec.Command(goBinary, "clean", "-modcache")
		if cleanFlags.verbose {
			cleanCmd.Stdout = cmd.OutOrStdout()
			cleanCmd.Stderr = cmd.ErrOrStderr()
		}

		if err := cleanCmd.Run(); err != nil {
			failed = append(failed, "module cache")
			if cleanFlags.verbose {
				fmt.Fprintf(cmd.ErrOrStderr(), "  ✗ Failed to clean module cache: %v\n", err)
			}
		} else {
			cleaned = append(cleaned, "module cache")
			if cleanFlags.verbose {
				fmt.Fprintln(cmd.OutOrStdout(), "  ✓ Module cache cleaned")
			}
		}
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout())
	if len(cleaned) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "✅ Successfully cleaned: %s\n", joinWithCommas(cleaned))
	}

	if len(failed) > 0 {
		fmt.Fprintf(cmd.ErrOrStderr(), "❌ Failed to clean: %s\n", joinWithCommas(failed))
		return fmt.Errorf("failed to clean %d target(s)", len(failed))
	}

	fmt.Fprintln(cmd.OutOrStdout(), "\n💡 Tip: If you're still seeing version mismatch errors, try rebuilding your project.")

	return nil
}

func getGoBinary(cfg *config.Config, version string) (string, error) {
	if version == "system" {
		// Use system go
		goBinary, err := exec.LookPath("go")
		if err != nil {
			return "", fmt.Errorf("system Go not found: %w", err)
		}
		return goBinary, nil
	}

	// Build path to versioned Go binary
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	goBinary := filepath.Join(versionPath, "bin", "go")

	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}

	// Verify it exists
	if _, err := os.Stat(goBinary); err != nil {
		return "", fmt.Errorf("Go binary not found for version %s: %w", version, err)
	}

	return goBinary, nil
}

func joinWithCommas(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	result := ""
	for i, item := range items {
		if i == len(items)-1 {
			result += "and " + item
		} else {
			result += item + ", "
		}
	}
	return result
}
