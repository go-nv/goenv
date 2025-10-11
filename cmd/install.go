package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/version"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:   "install [version]",
	Short: "Install a Go version",
	Long:  "Install a specific Go version. If no version is specified, installs the latest stable version.",
	RunE:  runInstall,
}

var installFlags struct {
	force   bool
	list    bool
	keep    bool
	verbose bool
	quiet   bool
	ipv4    bool
	ipv6    bool
	debug   bool
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&installFlags.force, "force", "f", false, "Force installation even if already installed")
	installCmd.Flags().BoolVarP(&installFlags.list, "list", "l", false, "List all available versions")
	installCmd.Flags().BoolVarP(&installFlags.keep, "keep", "k", false, "Keep downloaded files after installation")
	installCmd.Flags().BoolVarP(&installFlags.verbose, "verbose", "v", false, "Verbose mode: print detailed installation info")
	installCmd.Flags().BoolVarP(&installFlags.quiet, "quiet", "q", false, "Quiet mode: disable progress bar")
	installCmd.Flags().BoolVarP(&installFlags.ipv4, "ipv4", "4", false, "Resolve names to IPv4 addresses only")
	installCmd.Flags().BoolVarP(&installFlags.ipv6, "ipv6", "6", false, "Resolve names to IPv6 addresses only")
	installCmd.Flags().BoolVarP(&installFlags.debug, "debug", "g", false, "Enable debug output")
}

func runInstall(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Validate flags
	if installFlags.ipv4 && installFlags.ipv6 {
		return fmt.Errorf("cannot specify both --ipv4 and --ipv6")
	}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create directories: %w", err)
	}

	// Handle --list flag
	if installFlags.list {
		return runList(cmd, args)
	}

	installer := install.NewInstaller(cfg)

	// Configure installer options
	installer.Verbose = installFlags.verbose || installFlags.debug
	installer.Quiet = installFlags.quiet
	installer.KeepBuildPath = installFlags.keep

	// Determine version to install
	var goVersion string
	if len(args) > 0 {
		goVersion = args[0]
	} else {
		// Install latest stable version
		fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: cfg.Debug})
		releases, err := fetcher.FetchWithFallback(cfg.Root)
		if err != nil {
			return fmt.Errorf("failed to get versions: %w", err)
		}

		// Find latest stable version
		for _, release := range releases {
			if release.Stable {
				goVersion = release.Version
				break
			}
		}

		if goVersion == "" {
			return fmt.Errorf("no stable version found")
		}
	}

	if cfg.Debug {
		fmt.Printf("Debug: Installing Go version %s\n", goVersion)
	}

	return installer.Install(goVersion, installFlags.force)
}
