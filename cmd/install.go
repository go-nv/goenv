package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/defaulttools"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/version"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [version]",
	Short:   "Install a Go version",
	GroupID: "common",
	Long:    "Install a specific Go version. If no version is specified, installs the latest stable version.",
	RunE:    runInstall,
}

var installFlags struct {
	force        bool
	skipExisting bool
	list         bool
	keep         bool
	verbose      bool
	quiet        bool
	ipv4         bool
	ipv6         bool
	debug        bool
	complete     bool
	noRehash     bool
}

func init() {
	rootCmd.AddCommand(installCmd)
	installCmd.Flags().BoolVarP(&installFlags.force, "force", "f", false, "Force installation even if already installed")
	installCmd.Flags().BoolVarP(&installFlags.skipExisting, "skip-existing", "s", false, "Skip if the version appears to be installed already")
	installCmd.Flags().BoolVarP(&installFlags.list, "list", "l", false, "List all available versions")
	installCmd.Flags().BoolVarP(&installFlags.keep, "keep", "k", false, "Keep downloaded files after installation")
	installCmd.Flags().BoolVar(&installFlags.verbose, "verbose", false, "Verbose mode: print detailed installation info")
	installCmd.Flags().BoolVarP(&installFlags.quiet, "quiet", "q", false, "Quiet mode: disable progress bar")
	installCmd.Flags().BoolVarP(&installFlags.ipv4, "ipv4", "4", false, "Resolve names to IPv4 addresses only")
	installCmd.Flags().BoolVarP(&installFlags.ipv6, "ipv6", "6", false, "Resolve names to IPv6 addresses only")
	installCmd.Flags().BoolVarP(&installFlags.debug, "debug", "g", false, "Enable debug output")
	installCmd.Flags().BoolVar(&installFlags.noRehash, "no-rehash", false, "Skip automatic rehash after installation")
	installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = installCmd.Flags().MarkHidden("complete")

	// Apply custom help text to match bash version
	helptext.SetCommandHelp(installCmd)
}

func runInstall(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if installFlags.complete {
		cfg := config.Load()
		fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: false})
		releases, err := fetcher.FetchWithFallback(cfg.Root)
		if err == nil {
			for _, r := range releases {
				fmt.Fprintln(cmd.OutOrStdout(), r.Version)
			}
		}
		return nil
	}

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

	// Handle --skip-existing flag
	if installFlags.skipExisting {
		// Check if version is already installed
		versionPath := filepath.Join(cfg.VersionsDir(), goVersion)
		if _, err := os.Stat(versionPath); err == nil {
			// Already installed, skip silently
			return nil
		}
	}

	// Execute pre-install hooks
	executeHooks(hooks.PreInstall, map[string]string{
		"version": goVersion,
	})

	// Perform the actual installation
	err := installer.Install(goVersion, installFlags.force)

	// Execute post-install hooks (even if installation failed, for logging)
	executeHooks(hooks.PostInstall, map[string]string{
		"version": goVersion,
	})

	// Install default tools if installation succeeded
	if err == nil {
		installDefaultTools(cmd, goVersion)

		// Auto-rehash to update shims for new Go version and installed tools
		// Skip if --no-rehash flag or GOENV_NO_AUTO_REHASH environment variable is set
		shouldRehash := !installFlags.noRehash && !utils.GoenvEnvVarNoAutoRehash.IsTrue()

		if shouldRehash {
			if cfg.Debug {
				fmt.Fprintln(cmd.OutOrStdout(), "Debug: Auto-rehashing after installation")
			}
			shimMgr := shims.NewShimManager(cfg)
			_ = shimMgr.Rehash() // Don't fail the install if rehash fails
		} else if cfg.Debug {
			fmt.Fprintln(cmd.OutOrStdout(), "Debug: Skipping auto-rehash (disabled via flag or environment)")
		}
	}

	return err
}

// installDefaultTools installs configured default tools after a successful Go installation
func installDefaultTools(cmd *cobra.Command, goVersion string) {
	cfg := config.Load()
	configPath := defaulttools.ConfigPath(cfg.Root)

	// Load config (skip if file doesn't exist or has errors)
	toolConfig, err := defaulttools.LoadConfig(configPath)
	if err != nil || !toolConfig.Enabled || len(toolConfig.Tools) == 0 {
		return // Silently skip if not configured or disabled
	}

	// Show message if verbose or not quiet
	if !installFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "\n📦 Installing default tools...\n")
	}

	// Install tools (non-verbose to avoid clutter)
	if err := defaulttools.InstallTools(toolConfig, goVersion, cfg.Root, !installFlags.quiet); err != nil {
		// Don't fail the whole install if default tools fail
		if !installFlags.quiet {
			fmt.Fprintf(cmd.OutOrStderr(), "⚠️  Some default tools failed to install: %v\n", err)
		}
	}
}
