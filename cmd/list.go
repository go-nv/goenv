package cmd

import (
	"encoding/json"
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/version"
	"github.com/spf13/cobra"
)

var listCmd = &cobra.Command{
	Use:     "list",
	Short:   "List installed Go versions",
	GroupID: "common",
	Long: `List all locally installed Go versions with the current version highlighted.

By default, shows installed versions (same as 'goenv versions').
Use --remote to list available versions from golang.org.

Examples:
  goenv list              # Show installed versions
  goenv list --remote     # Show available versions to install
  goenv list --bare       # Show version numbers only`,
	RunE: runList,
}

var listFlags struct {
	bare        bool
	skipAliases bool
	remote      bool
	stable      bool
	json        bool
}

func init() {
	rootCmd.AddCommand(listCmd)

	// Flags for installed versions (when --remote is not used)
	listCmd.Flags().BoolVarP(&listFlags.bare, "bare", "b", false, "Display bare version numbers only")
	listCmd.Flags().BoolVar(&listFlags.skipAliases, "skip-aliases", false, "Skip aliases")
	listCmd.Flags().BoolVar(&listFlags.json, "json", false, "Output in JSON format")

	// Flags for remote versions (when --remote is used)
	listCmd.Flags().BoolVarP(&listFlags.remote, "remote", "r", false, "List available versions from golang.org")
	listCmd.Flags().BoolVar(&listFlags.stable, "stable", false, "Show only stable releases (with --remote)")
}

func runList(cmd *cobra.Command, args []string) error {
	// Validate: list command takes no positional arguments
	if len(args) > 0 {
		if listFlags.remote {
			return fmt.Errorf("Usage: goenv list --remote [--stable]")
		}
		return fmt.Errorf("Usage: goenv list [--bare] [--skip-aliases] [--remote]")
	}

	// Route to appropriate handler
	if listFlags.remote {
		return runListRemote(cmd)
	}
	return runListInstalled(cmd)
}

// runListInstalled shows locally installed versions (reuses versions command logic)
func runListInstalled(cmd *cobra.Command) error {
	// Copy flags to versionsFlags so we can reuse runVersions
	versionsFlags.bare = listFlags.bare
	versionsFlags.skipAliases = listFlags.skipAliases
	versionsFlags.json = listFlags.json

	// Reuse the versions command implementation
	return runVersions(cmd, []string{})
}

// runListRemote shows available versions from golang.org
func runListRemote(cmd *cobra.Command) error {
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

	// Handle JSON output
	if listFlags.json {
		type remoteVersionsOutput struct {
			SchemaVersion string   `json:"schema_version"`
			Remote        bool     `json:"remote"`
			StableOnly    bool     `json:"stable_only"`
			Versions      []string `json:"versions"`
		}

		// Strip "go" prefix from all versions for JSON output
		strippedVersions := make([]string, len(versions))
		for i, v := range versions {
			if len(v) > 2 && v[:2] == "go" {
				strippedVersions[i] = v[2:]
			} else {
				strippedVersions[i] = v
			}
		}

		output := remoteVersionsOutput{
			SchemaVersion: "1",
			Remote:        true,
			StableOnly:    listFlags.stable,
			Versions:      strippedVersions,
		}

		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		return encoder.Encode(output)
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
