package core

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cmdhooks "github.com/go-nv/goenv/cmd/hooks"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/toolupdater"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/version"
	"github.com/spf13/cobra"
)

var installCmd = &cobra.Command{
	Use:     "install [version]",
	Short:   "Install a Go version",
	GroupID: string(cmdpkg.GroupVersions),
	Long: `Install a specific Go version.

If no version is specified, goenv will:
  1. Check for .go-version in current directory or parent directories
  2. Check for go.mod and use the go directive
  3. Fall back to installing the latest stable version`,
	Example: `  # Auto-detect from .go-version or go.mod
  goenv install

  # Install specific version
  goenv install 1.21.5

  # Install latest patch version
  goenv install 1.21

  # Force reinstall
  goenv install -f 1.21.5

  # List available versions
  goenv install -l`,
	RunE: runInstall,
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
	cmdpkg.RootCmd.AddCommand(installCmd)
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
		cfg, _ := cmdutil.SetupContext()
		fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: false})
		releases, err := fetcher.FetchWithFallback(cfg.Root)
		if err == nil {
			for _, r := range releases {
				fmt.Fprintln(cmd.OutOrStdout(), r.Version)
			}
		}
		return nil
	}

	cfg, _ := cmdutil.SetupContext()

	// Validate flags
	if installFlags.ipv4 && installFlags.ipv6 {
		return fmt.Errorf("cannot specify both --ipv4 and --ipv6")
	}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		return errors.FailedTo("create directories", err)
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

		// Resolve partial versions (e.g., "1.21" -> "1.21.13")
		// This handles cases like: goenv install 1.21
		fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: cfg.Debug})
		releases, err := fetcher.FetchWithFallback(cfg.Root)
		if err != nil {
			return errors.FailedTo("get versions", err)
		}

		resolved, err := resolvePartialVersion(goVersion, releases)
		if err != nil {
			return err
		}

		if resolved != goVersion && !installFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "%sResolved %s to %s (latest patch)\n",
				utils.Emoji("🔍 "),
				utils.Cyan(goVersion),
				utils.Cyan(resolved))
		}

		goVersion = resolved // Use resolved version for installation
	} else {
		// Try to detect version from current directory (as documented in help text)
		mgr := manager.NewManager(cfg)
		detectedVersion, source, err := mgr.GetCurrentVersion()

		if err == nil && detectedVersion != "" && detectedVersion != "system" {
			// Found a version from .go-version or go.mod
			if !installFlags.quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "%sDetected version %s from %s\n",
					utils.Emoji("📍 "),
					utils.Cyan(detectedVersion),
					utils.Gray(source))
			}

			// Resolve partial versions from project files too
			fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: cfg.Debug})
			releases, err := fetcher.FetchWithFallback(cfg.Root)
			if err != nil {
				return errors.FailedTo("get versions", err)
			}

			resolved, err := resolvePartialVersion(detectedVersion, releases)
			if err != nil {
				return err
			}

			if resolved != detectedVersion && !installFlags.quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "%sResolved %s to %s (latest patch)\n",
					utils.Emoji("� "),
					utils.Cyan(detectedVersion),
					utils.Cyan(resolved))
			}

			goVersion = resolved
		} else {
			// No version detected, install latest stable (fallback)
			fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: cfg.Debug})
			releases, err := fetcher.FetchWithFallback(cfg.Root)
			if err != nil {
				return errors.FailedTo("get versions", err)
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

			if !installFlags.quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "%sNo version file found, installing latest stable: %s\n",
					utils.Emoji("ℹ️  "),
					utils.Cyan(goVersion))
			}
		}
	}

	if cfg.Debug {
		fmt.Printf("Debug: Installing Go version %s\n", goVersion)
	}

	// Handle --skip-existing flag
	if installFlags.skipExisting {
		// Check if version is already installed
		if cfg.IsVersionInstalled(goVersion) {
			// Already installed, skip silently
			return nil
		}
	}

	// Execute pre-install hooks
	cmdhooks.ExecuteHooks(hooks.PreInstall, map[string]string{
		"version": goVersion,
	})

	// Interactive: Check build dependencies before installation
	checkBuildDependencies(cmd, cfg)

	// Perform the actual installation
	err := installer.Install(goVersion, installFlags.force)

	// Execute post-install hooks (even if installation failed, for logging)
	cmdhooks.ExecuteHooks(hooks.PostInstall, map[string]string{
		"version": goVersion,
	})

	// Install default tools if installation succeeded
	if err == nil {
		installDefaultTools(cmd, goVersion)

		// Check for tool updates if auto-update is enabled
		checkToolUpdates(cmd, goVersion)

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

		// Interactive: Offer to set as global version
		offerSetGlobal(cmd, cfg, goVersion)
	}

	if err != nil {
		return errors.FailedTo(fmt.Sprintf("install Go %s", goVersion), err)
	}
	return nil
}

// installDefaultTools installs configured default tools after a successful Go installation
func installDefaultTools(cmd *cobra.Command, goVersion string) {
	cfg, _ := cmdutil.SetupContext()
	configPath := tools.ConfigPath(cfg.Root)

	// Load config (skip if file doesn't exist or has errors)
	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil || !toolConfig.Enabled || len(toolConfig.Tools) == 0 {
		return // Silently skip if not configured or disabled
	}

	// Show message if verbose or not quiet
	if !installFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%sInstalling default tools...\n", utils.Emoji("📦 "))
	}

	// Install tools (non-verbose to avoid clutter)
	if err := tools.InstallTools(toolConfig, goVersion, cfg.Root, cfg.HostGopath(), !installFlags.quiet); err != nil {
		// Don't fail the whole install if default tools fail
		if !installFlags.quiet {
			fmt.Fprintf(cmd.OutOrStderr(), "%sSome default tools failed to install: %v\n", utils.Emoji("⚠️  "), err)
		}
	}
}

// checkToolUpdates checks for and optionally updates tools if auto-update is enabled
func checkToolUpdates(cmd *cobra.Command, goVersion string) {
	cfg, _ := cmdutil.SetupContext()
	configPath := tools.ConfigPath(cfg.Root)

	// Load config
	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		return // Silently skip if config can't be loaded
	}

	// Skip if auto-update is not enabled for this trigger
	if !toolConfig.ShouldCheckOn("install") {
		return
	}

	// Check if enough time has passed since last check (throttling)
	if !toolConfig.ShouldCheckNow(goVersion) {
		return // Too soon since last check
	}

	// Skip if quiet mode (don't clutter output)
	if installFlags.quiet {
		return
	}

	// Import toolupdater here to avoid circular dependencies
	updater := toolupdater.NewUpdater(cfg)

	// Determine strategy from config
	strategy := toolupdater.StrategyAuto
	if toolConfig.UpdateStrategy != "" {
		strategy = toolupdater.UpdateStrategy(toolConfig.UpdateStrategy)
	}

	// Check for updates
	opts := toolupdater.UpdateOptions{
		Strategy:  strategy,
		GoVersion: goVersion,
		CheckOnly: !toolConfig.AutoUpdateInteractive, // Check only if not interactive
		Verbose:   false,
	}

	result, err := updater.CheckForUpdates(opts)
	if err != nil {
		return // Silently skip if check fails
	}

	// Mark that we checked (for throttling)
	toolConfig.MarkChecked(goVersion)
	_ = tools.SaveConfig(configPath, toolConfig) // Best effort save

	// Count updates available
	updatesAvailable := 0
	for _, check := range result.Checked {
		if check.UpdateAvailable {
			updatesAvailable++
		}
	}

	if updatesAvailable == 0 {
		return // Nothing to report
	}

	// Show updates
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s %d tool update(s) available\n",
		utils.Emoji("💡 "), updatesAvailable)

	// If interactive mode, prompt to install (similar to use.go)
	if toolConfig.AutoUpdateInteractive {
		ic := cmdutil.NewInteractiveContext(cmd)
		if ic.IsInteractive() {
			// Show what would be updated
			for _, check := range result.Checked {
				if check.UpdateAvailable {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s → %s\n",
						utils.Yellow("⬆"),
						utils.BoldWhite(check.ToolName),
						utils.Gray(check.CurrentVersion),
						utils.Green(check.LatestVersion))
				}
			}

			// Prompt to update
			if ic.Confirm("\nUpdate tools now?", true) {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sUpdating tools...\n", utils.Emoji("🔄 "))

				// Run the actual updates (already computed in result if CheckOnly was false)
				if len(result.Updated) > 0 {
					for _, toolName := range result.Updated {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", utils.Green("✓"), toolName)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "\n%sDone! Run 'goenv rehash' to update shims\n", utils.Emoji("✅ "))
				}
			}
		}
	} else {
		// Just show hint
		fmt.Fprintf(cmd.OutOrStdout(), "   Run 'goenv tools update' to update\n")
	}
}

// offerSetGlobal offers to set the newly installed version as the global default
func offerSetGlobal(cmd *cobra.Command, cfg *config.Config, goVersion string) {
	// Create interactive context
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Skip if non-interactive or quiet
	if !ctx.IsInteractive() || installFlags.quiet {
		return
	}

	// Check if there's already a global version set
	globalFile := filepath.Join(cfg.Root, "version")
	hasGlobal := false
	var currentGlobal string

	if data, err := os.ReadFile(globalFile); err == nil {
		currentGlobal = string(data)
		currentGlobal = filepath.Base(currentGlobal) // Remove any path components
		hasGlobal = currentGlobal != ""
	}

	// Construct the question based on whether there's an existing global version
	var question string
	if hasGlobal {
		question = fmt.Sprintf("Set Go %s as your global default? (currently: %s)", goVersion, currentGlobal)
	} else {
		question = fmt.Sprintf("Set Go %s as your global default?", goVersion)
	}

	// Offer to set as global (default: yes for first global, no if replacing)
	defaultYes := !hasGlobal
	if ctx.Confirm(question, defaultYes) {
		// Write the version file
		versionContent := goVersion + "\n"
		if err := utils.WriteFileWithContext(globalFile, []byte(versionContent), utils.PermFileDefault, "set global version"); err != nil {
			ctx.ErrorPrintf("%sFailed to set global version: %v\n", utils.Emoji("⚠️  "), err)
			return
		}

		ctx.Printf("%sGo %s is now your global default\n", utils.Emoji("✓"), goVersion)
		ctx.Printf("\nTo use this version in your current shell, run:\n")
		ctx.Printf("  %s\n", utils.BoldBlue("eval \"$(goenv init -)\""))
	}
}

// checkBuildDependencies checks for required build dependencies
// Note: Go binaries are pre-built, so build deps are only needed for source builds
// This is mainly informational for Linux users who might need these for other tools
func checkBuildDependencies(cmd *cobra.Command, cfg *config.Config) {
	// Create interactive context
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Skip if non-interactive, quiet, or not in guided mode
	if !ctx.IsGuided() || installFlags.quiet {
		return
	}

	// Only check on Linux (Mac has Xcode tools, Windows doesn't need build tools for binaries)
	if utils.IsWindows() {
		return
	}

	// Check for common build tools
	missingTools := checkForBuildTools()

	if len(missingTools) > 0 {
		// Build the problem description
		problem := fmt.Sprintf("Missing build tools: %s", missingTools[0])
		if len(missingTools) > 1 {
			problem = fmt.Sprintf("Missing build tools: %s", formatList(missingTools))
		}

		// Build repair description based on platform
		repairDesc := getInstallInstructions(missingTools)

		// Offer guidance (not automatic repair since we can't safely install system packages)
		if ctx.OfferRepair(problem, repairDesc) {
			ctx.Println(repairDesc)

			// Pause to let user install tools and return
			// WaitForUser automatically skips in CI/non-interactive mode
			ctx.WaitForUser("\nAfter installing the tools, press Enter to continue...")
		}
	}
}

// checkForBuildTools checks if common build tools are available
func checkForBuildTools() []string {
	var missing []string

	tools := []string{"gcc", "make"}

	for _, tool := range tools {
		if _, err := exec.LookPath(tool); err != nil {
			missing = append(missing, tool)
		}
	}

	return missing
}

// getInstallInstructions returns platform-specific instructions for installing build tools
func getInstallInstructions(tools []string) string {
	// Detect package manager and provide appropriate instructions
	if _, err := exec.LookPath("apt-get"); err == nil {
		return "Install with: sudo apt-get install build-essential"
	}
	if _, err := exec.LookPath("yum"); err == nil {
		return "Install with: sudo yum groupinstall 'Development Tools'"
	}
	if _, err := exec.LookPath("dnf"); err == nil {
		return "Install with: sudo dnf groupinstall 'Development Tools'"
	}
	if _, err := exec.LookPath("brew"); err == nil {
		return "Install Xcode Command Line Tools: xcode-select --install"
	}
	if _, err := exec.LookPath("pacman"); err == nil {
		return "Install with: sudo pacman -S base-devel"
	}

	// Generic fallback
	return fmt.Sprintf("Install %s using your system's package manager", formatList(tools))
}

// formatList formats a list of strings as "a, b, and c"
func formatList(items []string) string {
	if len(items) == 0 {
		return ""
	}
	if len(items) == 1 {
		return items[0]
	}
	if len(items) == 2 {
		return items[0] + " and " + items[1]
	}

	// 3 or more items
	result := ""
	for i := 0; i < len(items)-1; i++ {
		result += items[i] + ", "
	}
	result += "and " + items[len(items)-1]
	return result
}

// resolvePartialVersion resolves a partial version (e.g., "1.21") to the latest patch version (e.g., "1.21.13")
// If the version is already a full version, returns it unchanged.
func resolvePartialVersion(requestedVersion string, releases []version.GoRelease) (string, error) {
	// Normalize the input version
	normalized := utils.NormalizeGoVersion(requestedVersion)

	// First try exact match
	for _, release := range releases {
		if utils.MatchesVersion(release.Version, normalized) {
			return utils.NormalizeGoVersion(release.Version), nil
		}
	}

	// If no exact match, try prefix match to find latest patch
	// e.g., "1.21" should match "1.21.13", "1.21.12", etc.
	var candidates []version.GoRelease
	for _, release := range releases {
		releaseNormalized := utils.NormalizeGoVersion(release.Version)
		if strings.HasPrefix(releaseNormalized, normalized+".") || releaseNormalized == normalized {
			candidates = append(candidates, release)
		}
	}

	if len(candidates) == 0 {
		// No matches found, return helpful error
		return "", fmt.Errorf("version %s not found\n\nUse 'goenv install --list' to see all available versions", requestedVersion)
	}

	// Return the first candidate (releases are sorted, so this is the latest)
	return utils.NormalizeGoVersion(candidates[0].Version), nil
}
