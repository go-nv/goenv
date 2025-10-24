package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var refreshCmd = &cobra.Command{
	Use:   "refresh",
	Short: "Clear caches and fetch fresh version data",
	Long: `Clear all cached version data and force a fresh fetch from the official Go API.

This removes:
  - versions-cache.json (version list cache)
  - releases-cache.json (full release metadata cache)

The next time you run a command that needs version data, it will fetch fresh data from go.dev.`,
	RunE: runRefresh,
}

var refreshFlags struct {
	verbose bool
}

func init() {
	rootCmd.AddCommand(refreshCmd)
	refreshCmd.Flags().BoolVar(&refreshFlags.verbose, "verbose", false, "Show detailed output")
}

func runRefresh(cmd *cobra.Command, args []string) error {
	// Validate: refresh command takes no positional arguments (only --verbose flag)
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv refresh [--verbose]")
	}

	cfg := config.Load()

	cacheFiles := []string{
		filepath.Join(cfg.Root, "versions-cache.json"),
		filepath.Join(cfg.Root, "releases-cache.json"),
	}

	removed := 0
	notFound := 0
	permissionFixed := 0

	for _, cacheFile := range cacheFiles {
		if _, err := os.Stat(cacheFile); err == nil {
			// File exists, remove it
			if err := os.Remove(cacheFile); err != nil {
				return fmt.Errorf("failed to remove %s: %w", filepath.Base(cacheFile), err)
			}
			removed++
			if refreshFlags.verbose {
				fmt.Fprintf(cmd.OutOrStdout(), "%sRemoved %s\n", utils.Emoji("✓ "), filepath.Base(cacheFile))
			}
		} else if os.IsNotExist(err) {
			// File doesn't exist
			notFound++
			if refreshFlags.verbose {
				fmt.Fprintf(cmd.OutOrStdout(), "• %s not found (already clean)\n", filepath.Base(cacheFile))
			}
		} else {
			// Other error (permissions, etc.)
			return fmt.Errorf("failed to check %s: %w", filepath.Base(cacheFile), err)
		}
	}

	// Ensure cache directory has secure permissions
	cacheDir := filepath.Dir(cacheFiles[0])
	if err := ensureCacheDirPermissions(cacheDir, cmd); err != nil && refreshFlags.verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Warning: %v\n", err)
	} else if err == nil && refreshFlags.verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "%sEnsured cache directory has secure permissions\n", utils.Emoji("✓ "))
		permissionFixed++
	}

	// Summary
	if removed > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sCache cleared! Removed %d cache file(s).\n", utils.Emoji("✓ "), removed)
		fmt.Fprintln(cmd.OutOrStdout(), "Next version fetch will retrieve fresh data from go.dev")
	} else if notFound == len(cacheFiles) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cache is already clean (no cache files found)")
	}

	return nil
}

// ensureCacheDirPermissions ensures the cache directory has secure permissions (0700)
func ensureCacheDirPermissions(cacheDir string, cmd *cobra.Command) error {
	// Skip permission checks on Windows (uses ACLs instead of POSIX permissions)
	if runtime.GOOS == "windows" {
		return nil
	}

	// Check if directory exists
	info, err := os.Stat(cacheDir)
	if err != nil {
		if os.IsNotExist(err) {
			// Directory doesn't exist, create it with secure permissions
			return os.MkdirAll(cacheDir, 0700)
		}
		return fmt.Errorf("failed to check cache directory: %w", err)
	}

	// Check permissions
	mode := info.Mode()
	if mode.Perm() != 0700 {
		// Fix permissions
		if err := os.Chmod(cacheDir, 0700); err != nil {
			return fmt.Errorf("failed to fix cache directory permissions: %w", err)
		}
	}

	return nil
}
