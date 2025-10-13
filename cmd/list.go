package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/version"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:    "list",
	Short:  "List all available Go versions from the official repository",
	Long:   "Fetches and displays all available Go versions from the official Go website",
	RunE:   runList,
	Hidden: true, // Hidden from commands list to match bash version
}

var listFlags struct {
	stable bool
}

func init() {
	rootCmd.AddCommand(listCmd)
	listCmd.Flags().BoolVar(&listFlags.stable, "stable", false, "Show only stable releases")
}

func runList(cmd *cobra.Command, args []string) error {
	// Validate: list command takes no positional arguments (only --stable flag)
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv list [--stable]")
	}

	cfg := config.Load()
	if cfg.Debug {
		fmt.Println("Debug: Fetching available Go versions...")
	}

	// Create fetcher with cache directory
	fetcher := version.NewFetcherWithCache(cfg.Root)
	fetcher.SetDebug(cfg.Debug)

	// Fetch all versions (from cache or Railway API)
	versions, err := fetcher.FetchAllVersions()
	if err != nil {
		return fmt.Errorf("failed to fetch versions: %w", err)
	}

	// Filter stable versions if requested
	if listFlags.stable {
		var stableVersions []string
		for _, v := range versions {
			// Versions without beta, rc, or other pre-release markers are stable
			if !version.IsPrerelease(v) {
				stableVersions = append(stableVersions, v)
			}
		}
		versions = stableVersions
	}

	// Match bash goenv install --list format:
	// - Header "Available versions:"
	// - Two-space indentation
	// - Strip "go" prefix from version numbers
	// - Reverse order (oldest first, like bash does)
	fmt.Fprintln(cmd.OutOrStdout(), "Available versions:")

	// Reverse the slice to show oldest first
	for i := len(versions) - 1; i >= 0; i-- {
		v := versions[i]
		// Strip "go" prefix if present
		displayVersion := v
		if len(v) > 2 && v[:2] == "go" {
			displayVersion = v[2:]
		}

		// Display with two-space indentation (no unstable marker for install --list)
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", displayVersion)
	}

	return nil
}
