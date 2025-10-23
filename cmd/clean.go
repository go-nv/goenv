package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/pathutil"
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
	force    bool
	verbose  bool
	diagnose bool
}

func init() {
	rootCmd.AddCommand(cleanCmd)
	cleanCmd.Flags().BoolVarP(&cleanFlags.force, "force", "f", false, "Skip confirmation prompts")
	cleanCmd.Flags().BoolVarP(&cleanFlags.verbose, "verbose", "v", false, "Show detailed output")
	cleanCmd.Flags().BoolVar(&cleanFlags.diagnose, "diagnose", false, "Show why cache might need cleaning")
	helptext.SetCommandHelp(cleanCmd)
}

func completeCleanTargets(cmd *cobra.Command, args []string, toComplete string) ([]string, cobra.ShellCompDirective) {
	if len(args) > 0 {
		return nil, cobra.ShellCompDirectiveNoFileComp
	}
	return []string{"build", "modcache", "all"}, cobra.ShellCompDirectiveNoFileComp
}

func runClean(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// If diagnose flag is set, run diagnostics and exit
	if cleanFlags.diagnose {
		return diagnoseCacheIssues(cmd, cfg)
	}

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
	goBinaryBase := filepath.Join(versionPath, "bin", "go")

	// Find the executable (handles .exe and .bat on Windows)
	goBinary, err := pathutil.FindExecutable(goBinaryBase)
	if err != nil {
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

func diagnoseCacheIssues(cmd *cobra.Command, cfg *config.Config) error {
	fmt.Fprintln(cmd.OutOrStdout(), "🔍 Diagnosing cache issues...")
	fmt.Fprintln(cmd.OutOrStdout())

	// Check architecture
	fmt.Fprintf(cmd.OutOrStdout(), "Current architecture: %s/%s\n", runtime.GOOS, runtime.GOARCH)

	// Get current Go version
	mgr := manager.NewManager(cfg)
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err == nil && currentVersion != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Current Go version: %s\n", currentVersion)
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check GOCACHE
	gocacheCmd := exec.Command("go", "env", "GOCACHE")
	if output, err := gocacheCmd.Output(); err == nil {
		gocache := strings.TrimSpace(string(output))
		fmt.Fprintf(cmd.OutOrStdout(), "GOCACHE location: %s\n", gocache)

		// Check if it's shared or version-specific
		if strings.Contains(gocache, cfg.Root) {
			fmt.Fprintln(cmd.OutOrStdout(), "✅ Using version-specific cache (goenv managed)")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "⚠️  WARNING: Using shared system cache (not version-specific)")
			fmt.Fprintln(cmd.OutOrStdout(), "   This can cause 'exec format error' when switching versions")
			fmt.Fprintln(cmd.OutOrStdout(), "   or architectures (e.g., Rosetta vs native on Apple Silicon)")
		}

		// Check if cache exists and estimate size
		if stat, err := os.Stat(gocache); err == nil && stat.IsDir() {
			// Try to estimate cache size
			var totalSize int64
			filepath.Walk(gocache, func(path string, info os.FileInfo, err error) error {
				if err == nil && !info.IsDir() {
					totalSize += info.Size()
				}
				return nil
			})
			sizeMB := totalSize / (1024 * 1024)
			if sizeMB > 0 {
				fmt.Fprintf(cmd.OutOrStdout(), "   Cache size: ~%d MB\n", sizeMB)
			}
		}
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "⚠️  Cannot determine GOCACHE location")
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check GOMODCACHE
	gomodcacheCmd := exec.Command("go", "env", "GOMODCACHE")
	if output, err := gomodcacheCmd.Output(); err == nil {
		gomodcache := strings.TrimSpace(string(output))
		fmt.Fprintf(cmd.OutOrStdout(), "GOMODCACHE location: %s\n", gomodcache)

		// Check if it's shared or version-specific
		if strings.Contains(gomodcache, cfg.Root) {
			fmt.Fprintln(cmd.OutOrStdout(), "✅ Using version-specific module cache (goenv managed)")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "⚠️  Using shared system module cache")
		}
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Check for multiple Go versions
	versions, err := mgr.ListInstalledVersions()
	if err == nil && len(versions) > 1 {
		fmt.Fprintf(cmd.OutOrStdout(), "Installed Go versions: %d\n", len(versions))
		for _, v := range versions {
			if v == currentVersion {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %s (current)\n", v)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", v)
			}
		}
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "💡 With multiple versions, version-specific caching is recommended")
		fmt.Fprintln(cmd.OutOrStdout(), "   Run: goenv clean build")
	}

	// Check for cache isolation settings
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Cache isolation settings:")
	if os.Getenv("GOENV_DISABLE_GOCACHE") == "1" {
		fmt.Fprintln(cmd.OutOrStdout(), "  ⚠️  GOENV_DISABLE_GOCACHE=1 (cache isolation disabled)")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  ✅ GOENV_DISABLE_GOCACHE not set (cache isolation enabled)")
	}
	if os.Getenv("GOENV_DISABLE_GOMODCACHE") == "1" {
		fmt.Fprintln(cmd.OutOrStdout(), "  ⚠️  GOENV_DISABLE_GOMODCACHE=1 (module cache isolation disabled)")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  ✅ GOENV_DISABLE_GOMODCACHE not set (module cache isolation enabled)")
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout(), "To clean caches:")
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv clean build      # Clean build cache only")
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv clean modcache   # Clean module cache")
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv clean all        # Clean both caches")

	return nil
}
