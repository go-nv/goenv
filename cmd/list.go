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
	cfg := config.Load()
	if cfg.Debug {
		fmt.Println("Debug: Fetching available Go versions...")
	}

	fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: cfg.Debug})
	releases, err := fetcher.FetchWithFallback(cfg.Root)
	if err != nil {
		return fmt.Errorf("failed to fetch versions: %w", err)
	}

	// Filter stable versions if requested
	if listFlags.stable {
		var stableReleases []version.GoRelease
		for _, release := range releases {
			if release.Stable {
				stableReleases = append(stableReleases, release)
			}
		}
		releases = stableReleases
	}

	// Sort versions (newest first)
	version.SortVersions(releases)

	// Display versions
	for _, release := range releases {
		status := ""
		if !release.Stable {
			status = " (unstable)"
		}
		fmt.Printf("%s%s\n", release.Version, status)
	}

	return nil
}
