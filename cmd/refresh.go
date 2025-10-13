package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
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
	refreshCmd.Flags().BoolVarP(&refreshFlags.verbose, "verbose", "v", false, "Show detailed output")
}

func runRefresh(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	cacheFiles := []string{
		filepath.Join(cfg.Root, "versions-cache.json"),
		filepath.Join(cfg.Root, "releases-cache.json"),
	}

	removed := 0
	notFound := 0

	for _, cacheFile := range cacheFiles {
		if _, err := os.Stat(cacheFile); err == nil {
			// File exists, remove it
			if err := os.Remove(cacheFile); err != nil {
				return fmt.Errorf("failed to remove %s: %w", filepath.Base(cacheFile), err)
			}
			removed++
			if refreshFlags.verbose {
				fmt.Fprintf(cmd.OutOrStdout(), "✓ Removed %s\n", filepath.Base(cacheFile))
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

	// Summary
	if removed > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Cache cleared! Removed %d cache file(s).\n", removed)
		fmt.Fprintln(cmd.OutOrStdout(), "Next version fetch will retrieve fresh data from go.dev")
	} else if notFound == len(cacheFiles) {
		fmt.Fprintln(cmd.OutOrStdout(), "Cache is already clean (no cache files found)")
	}

	return nil
}
