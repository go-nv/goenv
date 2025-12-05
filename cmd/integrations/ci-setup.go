package integrations

import (
	"fmt"
	"os"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var ciSetupCmd = &cobra.Command{
	Use:     "ci-setup",
	Short:   "Configure goenv for CI/CD environments",
	GroupID: string(cmdpkg.GroupIntegrations),
	Long: `Configures goenv for optimal CI/CD usage with two-phase caching optimization.

This command supports two modes:

1. Environment Setup (default):
   Outputs environment variables and recommendations for CI/CD pipelines.

2. Two-Phase Installation (--install flag):
   Phase 1: Install Go versions (for caching)
   Phase 2: Use the cached versions

This two-phase approach optimizes CI caching by separating Go installation
(which can be cached) from Go usage (which uses the cache).

Example usage in CI:

  # Basic environment setup
  eval "$(goenv ci-setup)"
  goenv install 1.23.2
  goenv global 1.23.2

  # Two-phase optimization (recommended for CI)
  # GitHub Actions:
  - name: Install Go versions (cached)
    run: goenv ci-setup --install 1.23.2 1.22.5
  - name: Use Go version
    run: goenv global 1.23.2

  # Install from .go-version file
  - name: Install Go versions (cached)
    run: goenv ci-setup --install --from-file
  - name: Use Go version from file
    run: goenv use --auto

Options:
  --shell         Output format (bash, zsh, fish, powershell, github, gitlab)
  --verbose       Show detailed recommendations
  --install       Two-phase mode: install Go versions for caching
  --from-file     Read versions from .go-version, .tool-versions, or go.mod
  --skip-rehash   Skip rehashing during installation (faster)`,
	RunE: runCISetup,
}

var (
	ciShell      string
	ciVerbose    bool
	ciInstall    bool
	ciFromFile   bool
	ciSkipRehash bool
)

func init() {
	cmdpkg.RootCmd.AddCommand(ciSetupCmd)
	ciSetupCmd.Flags().StringVar(&ciShell, "shell", "bash", "Shell format (bash, zsh, fish, powershell, github, gitlab)")
	ciSetupCmd.Flags().BoolVar(&ciVerbose, "verbose", false, "Show detailed recommendations")
	ciSetupCmd.Flags().BoolVar(&ciInstall, "install", false, "Two-phase mode: install Go versions")
	ciSetupCmd.Flags().BoolVar(&ciFromFile, "from-file", false, "Read versions from .go-version or go.mod")
	ciSetupCmd.Flags().BoolVar(&ciSkipRehash, "skip-rehash", false, "Skip rehashing during installation")
	helptext.SetCommandHelp(ciSetupCmd)
}

func runCISetup(cmd *cobra.Command, args []string) error {
	cfg, _ := cmdutil.SetupContext()

	// Two-phase installation mode
	if ciInstall {
		return runCIInstallPhase(cmd, args, cfg)
	}

	// Environment setup mode (default)
	if ciVerbose {
		fmt.Fprintln(cmd.OutOrStdout(), "# goenv CI/CD Setup")
		fmt.Fprintln(cmd.OutOrStdout(), "# ==================")
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Determine shell format
	shellFormat := ciShell
	if shellFormat == "" {
		shellFormat = "bash"
	}

	// Detect CI environment
	isGitHubActions := os.Getenv(utils.EnvVarGitHubActions) == "true"
	isGitLabCI := os.Getenv(utils.EnvVarGitLabCI) == "true"
	isCircleCI := os.Getenv(utils.EnvVarCircleCI) == "true"
	isTravisCI := os.Getenv(utils.EnvVarTravisCI) == "true"

	// Auto-detect shell format based on platform and CI environment
	if utils.IsWindows() && shellFormat == "bash" {
		shellFormat = "powershell"
	} else if isGitHubActions && shellFormat == "bash" {
		shellFormat = "github"
	} else if isGitLabCI && shellFormat == "bash" {
		shellFormat = "gitlab"
	}

	// Output CI-optimized environment variables
	switch shellFormat {
	case "github":
		outputGitHubActions(cmd, cfg)
	case "gitlab":
		outputGitLabCI(cmd, cfg)
	case "fish":
		outputFish(cmd, cfg)
	case "powershell":
		outputPowerShell(cmd, cfg)
	default: // bash, zsh
		outputBash(cmd, cfg)
	}

	// Show recommendations if verbose
	if ciVerbose {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "# Recommendations:")
		fmt.Fprintln(cmd.OutOrStdout(), "#")
		fmt.Fprintln(cmd.OutOrStdout(), "# 1. Use two-phase installation for optimal caching:")
		fmt.Fprintln(cmd.OutOrStdout(), "#    Phase 1: goenv ci-setup --install <versions>")
		fmt.Fprintln(cmd.OutOrStdout(), "#    Phase 2: goenv global <version> or goenv use --auto")
		fmt.Fprintln(cmd.OutOrStdout(), "# 2. Cache GOENV_ROOT/versions to speed up subsequent runs")
		fmt.Fprintln(cmd.OutOrStdout(), "# 3. Set GOTOOLCHAIN=local to prevent automatic Go downloads")
		fmt.Fprintln(cmd.OutOrStdout(), "# 4. Use specific Go versions (not 'latest') for reproducibility")
		fmt.Fprintln(cmd.OutOrStdout(), "#")

		if isGitHubActions {
			fmt.Fprintln(cmd.OutOrStdout(), "# GitHub Actions two-phase example:")
			fmt.Fprintln(cmd.OutOrStdout(), "#   - uses: actions/cache@v3")
			fmt.Fprintln(cmd.OutOrStdout(), "#     with:")
			fmt.Fprintf(cmd.OutOrStdout(), "#       path: %s/versions\n", cfg.Root)
			fmt.Fprintln(cmd.OutOrStdout(), "#       key: goenv-${{ runner.os }}-${{ hashFiles('.go-version') }}")
			fmt.Fprintln(cmd.OutOrStdout(), "#   - name: Install Go versions")
			fmt.Fprintln(cmd.OutOrStdout(), "#     run: goenv ci-setup --install --from-file")
			fmt.Fprintln(cmd.OutOrStdout(), "#   - name: Use Go version")
			fmt.Fprintln(cmd.OutOrStdout(), "#     run: goenv use --auto")
		} else if isGitLabCI {
			fmt.Fprintln(cmd.OutOrStdout(), "# GitLab CI two-phase example:")
			fmt.Fprintln(cmd.OutOrStdout(), "#   cache:")
			fmt.Fprintln(cmd.OutOrStdout(), "#     key: goenv-${CI_COMMIT_REF_SLUG}")
			fmt.Fprintln(cmd.OutOrStdout(), "#     paths:")
			fmt.Fprintf(cmd.OutOrStdout(), "#       - %s/versions\n", cfg.Root)
			fmt.Fprintln(cmd.OutOrStdout(), "#   script:")
			fmt.Fprintln(cmd.OutOrStdout(), "#     - goenv ci-setup --install --from-file")
			fmt.Fprintln(cmd.OutOrStdout(), "#     - goenv use --auto")
		} else if isCircleCI {
			fmt.Fprintln(cmd.OutOrStdout(), "# CircleCI two-phase example:")
			fmt.Fprintln(cmd.OutOrStdout(), "#   - save_cache:")
			fmt.Fprintln(cmd.OutOrStdout(), "#       key: goenv-{{ checksum \".go-version\" }}")
			fmt.Fprintln(cmd.OutOrStdout(), "#       paths:")
			fmt.Fprintf(cmd.OutOrStdout(), "#         - %s/versions\n", cfg.Root)
			fmt.Fprintln(cmd.OutOrStdout(), "#   - run: goenv ci-setup --install --from-file")
			fmt.Fprintln(cmd.OutOrStdout(), "#   - run: goenv use --auto")
		} else if isTravisCI {
			fmt.Fprintln(cmd.OutOrStdout(), "# Travis CI two-phase example:")
			fmt.Fprintln(cmd.OutOrStdout(), "#   cache:")
			fmt.Fprintln(cmd.OutOrStdout(), "#     directories:")
			fmt.Fprintf(cmd.OutOrStdout(), "#       - %s/versions\n", cfg.Root)
			fmt.Fprintln(cmd.OutOrStdout(), "#   script:")
			fmt.Fprintln(cmd.OutOrStdout(), "#     - goenv ci-setup --install --from-file")
			fmt.Fprintln(cmd.OutOrStdout(), "#     - goenv use --auto")
		}
	}

	return nil
}

func outputBash(cmd *cobra.Command, cfg *config.Config) {
	out := cmd.OutOrStdout()

	// Set GOENV_ROOT
	fmt.Fprintf(out, "export GOENV_ROOT=%s\n", cfg.Root)

	// Add to PATH - use correct separator for platform
	pathSep := string(os.PathListSeparator)
	fmt.Fprintf(out, "export PATH=\"%s/bin%s%s/shims%s$PATH\"\n", cfg.Root, pathSep, cfg.Root, pathSep)

	// CI-specific optimizations
	fmt.Fprintln(out, "export GOTOOLCHAIN=local")
	fmt.Fprintln(out, "export GOENV_DISABLE_GOPATH=1")

	// Quiet mode for cleaner CI logs
	fmt.Fprintln(out, "# export GOENV_DEBUG=1  # Uncomment for debugging")
}

func outputFish(cmd *cobra.Command, cfg *config.Config) {
	out := cmd.OutOrStdout()

	fmt.Fprintf(out, "set -gx GOENV_ROOT %s\n", cfg.Root)
	fmt.Fprintf(out, "set -gx PATH %s/bin %s/shims $PATH\n", cfg.Root, cfg.Root)
	fmt.Fprintln(out, "set -gx GOTOOLCHAIN local")
	fmt.Fprintln(out, "set -gx GOENV_DISABLE_GOPATH 1")
}

func outputPowerShell(cmd *cobra.Command, cfg *config.Config) {
	out := cmd.OutOrStdout()

	// PowerShell syntax for environment variables
	// Escape single quotes in paths for PowerShell (double them)
	escapedRoot := strings.ReplaceAll(cfg.Root, "'", "''")
	fmt.Fprintf(out, "$env:GOENV_ROOT = '%s'\n", escapedRoot)

	// Use Windows path separator (;) for PATH
	// Wrap entire RHS in double quotes and use single-quoted string literals for paths
	// This handles spaces and special characters correctly
	fmt.Fprintf(out, "$env:PATH = \"%s\\bin;%s\\shims;$env:PATH\"\n",
		strings.ReplaceAll(escapedRoot, "\"", "`\""),
		strings.ReplaceAll(escapedRoot, "\"", "`\""))

	// CI-specific optimizations
	fmt.Fprintln(out, "$env:GOTOOLCHAIN = 'local'")
	fmt.Fprintln(out, "$env:GOENV_DISABLE_GOPATH = '1'")

	// Quiet mode for cleaner CI logs
	fmt.Fprintln(out, "# $env:GOENV_DEBUG = '1'  # Uncomment for debugging")
}

func outputGitHubActions(cmd *cobra.Command, cfg *config.Config) {
	out := cmd.OutOrStdout()

	// GitHub Actions uses special syntax
	fmt.Fprintf(out, "echo \"%s=%s\" >> $GITHUB_ENV\n", utils.GoenvEnvVarRoot.String(), cfg.Root)
	fmt.Fprintf(out, "echo \"%s/bin\" >> $GITHUB_PATH\n", cfg.Root)
	fmt.Fprintf(out, "echo \"%s/shims\" >> $GITHUB_PATH\n", cfg.Root)
	fmt.Fprintln(out, "echo \"GOTOOLCHAIN=local\" >> $GITHUB_ENV")
	fmt.Fprintf(out, "echo \"%s=1\" >> $GITHUB_ENV\n", utils.GoenvEnvVarDisableGopath.String())

	// Also export for current step - use platform-appropriate separator
	pathSep := string(os.PathListSeparator)
	fmt.Fprintf(out, "export %s=%s\n", utils.GoenvEnvVarRoot.String(), cfg.Root)
	fmt.Fprintf(out, "export PATH=\"%s/bin%s%s/shims%s$PATH\"\n", cfg.Root, pathSep, cfg.Root, pathSep)
	fmt.Fprintln(out, "export GOTOOLCHAIN=local")
	fmt.Fprintf(out, "export %s=1\n", utils.GoenvEnvVarDisableGopath.String())
}

func outputGitLabCI(cmd *cobra.Command, cfg *config.Config) {
	out := cmd.OutOrStdout()

	// GitLab CI uses standard bash syntax - use platform-appropriate separator
	pathSep := string(os.PathListSeparator)
	fmt.Fprintf(out, "export GOENV_ROOT=%s\n", cfg.Root)
	fmt.Fprintf(out, "export PATH=\"%s/bin%s%s/shims%s$PATH\"\n", cfg.Root, pathSep, cfg.Root, pathSep)
	fmt.Fprintln(out, "export GOTOOLCHAIN=local")
	fmt.Fprintln(out, "export GOENV_DISABLE_GOPATH=1")

	// GitLab-specific recommendations
	if ciVerbose {
		fmt.Fprintln(out, "# Add to .gitlab-ci.yml:")
		fmt.Fprintln(out, "# variables:")
		fmt.Fprintf(out, "#   GOENV_ROOT: %s\n", cfg.Root)
		fmt.Fprintln(out, "#   GOTOOLCHAIN: local")
	}
}

// runCIInstallPhase handles the two-phase installation mode for CI optimization
func runCIInstallPhase(cmd *cobra.Command, args []string, cfg *config.Config) error {
	var versions []string

	// Determine which versions to install
	if ciFromFile {
		// Read versions from .go-version, .tool-versions, or go.mod
		discoveredVersions, err := discoverVersionsFromFiles()
		if err != nil {
			return errors.FailedTo("discover versions from files", err)
		}
		if len(discoveredVersions) == 0 {
			return fmt.Errorf("no versions found in .go-version, .tool-versions, or go.mod")
		}
		versions = discoveredVersions
		fmt.Fprintf(cmd.OutOrStdout(), "%sDiscovered versions: %v\n", utils.Emoji("📋 "), versions)
	} else if len(args) > 0 {
		// Use versions from command line arguments
		versions = args
	} else {
		return fmt.Errorf("no versions specified. Use --from-file or provide version arguments")
	}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		return errors.FailedTo("create directories", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sCI Setup: Installing %d Go version(s) for caching\n", utils.Emoji("🚀 "), len(versions))
	if ciSkipRehash {
		fmt.Fprintf(cmd.OutOrStdout(), "%sFast mode: rehashing will be skipped during installation\n", utils.Emoji("⚡ "))
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Install each version
	successCount := 0
	skippedCount := 0
	failedCount := 0

	for i, version := range versions {
		fmt.Fprintf(cmd.OutOrStdout(), "[%d/%d] Installing Go %s...\n", i+1, len(versions), version)

		// Check if already installed
		if cfg.IsVersionInstalled(version) {
			fmt.Fprintf(cmd.OutOrStdout(), "  %sAlready installed (cached)\n", utils.Emoji("✅ "))
			skippedCount++
			continue
		}

		// Install using the existing installer
		if err := installVersion(cfg, version, ciSkipRehash); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "  %sFailed: %v\n", utils.Emoji("❌ "), err)
			failedCount++
			continue
		}

		fmt.Fprintf(cmd.OutOrStdout(), "  %sInstalled successfully\n", utils.Emoji("✅ "))
		successCount++
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sInstallation Summary:\n", utils.Emoji("📊 "))
	fmt.Fprintf(cmd.OutOrStdout(), "  %sInstalled: %d\n", utils.Emoji("✅ "), successCount)
	fmt.Fprintf(cmd.OutOrStdout(), "  %sSkipped (cached): %d\n", utils.Emoji("⏭️  "), skippedCount)
	if failedCount > 0 {
		fmt.Fprintf(cmd.OutOrStderr(), "  %sFailed: %d\n", utils.Emoji("❌ "), failedCount)
	}

	// Rehash once at the end if not skipped
	if !ciSkipRehash && successCount > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sRehashing shims...\n", utils.Emoji("🔄 "))
		if err := rehashShims(cfg); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "%sWarning: rehash failed: %v\n", utils.Emoji("⚠️  "), err)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%sRehash complete\n", utils.Emoji("✅ "))
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sNext step: Use the installed version with:\n", utils.Emoji("💡 "))
	if ciFromFile {
		fmt.Fprintln(cmd.OutOrStdout(), "   goenv use --auto")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "   goenv global %s\n", versions[0])
	}

	if failedCount > 0 {
		return fmt.Errorf("%d version(s) failed to install", failedCount)
	}

	return nil
}

// discoverVersionsFromFiles reads version requirements from common files
// Uses manager API for consistent version file parsing across all formats
func discoverVersionsFromFiles() ([]string, error) {
	versions := make(map[string]bool) // Use map to deduplicate
	cfg, mgr := cmdutil.SetupContext()
	_ = cfg // unused but required by SetupContext

	// Check for version files in priority order
	versionFiles := []string{config.VersionFileName, config.ToolVersionsFileName, config.GoModFileName}

	for _, filename := range versionFiles {
		if utils.FileExists(filename) {
			// File exists, try to read version using manager API
			if version, err := mgr.ReadVersionFile(filename); err == nil && version != "" {
				// Manager returns colon-separated versions for multi-line files
				// Split them and add each one
				for _, v := range strings.Split(version, ":") {
					v = strings.TrimSpace(v)
					if v != "" {
						versions[v] = true
					}
				}
				break // Only first go directive matters
			}
		}
	}

	// Convert map to slice
	result := make([]string, 0, len(versions))
	for v := range versions {
		result = append(result, v)
	}

	return result, nil
}

// installVersion installs a single Go version (helper for batch installation)
func installVersion(cfg *config.Config, version string, noRehash bool) error {
	// Use the install package's installer directly
	// This avoids needing to shell out or manipulate global flags
	installer := install.NewInstaller(cfg)
	installer.Quiet = true // Quiet mode for cleaner CI output

	// Install the version
	return installer.Install(version, false)
}

// rehashShims regenerates shims for all installed versions
func rehashShims(cfg *config.Config) error {
	shimMgr := shims.NewShimManager(cfg)
	return shimMgr.Rehash()
}
