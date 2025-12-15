package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:     "explain",
	Short:   "Explain why a particular Go version is active",
	GroupID: string(cmdpkg.GroupGettingStarted),
	Long: `Explains, in plain English, why a particular Go version is active,
where it was set, and how to change it.

This command is helpful for:
  - Understanding goenv's version resolution
  - Troubleshooting version conflicts
  - Learning how to change the active version
  - Onboarding new users to goenv

The explanation includes:
  - Current active version
  - Source of the version setting (environment variable, file, etc.)
  - Priority level in goenv's resolution order
  - Step-by-step instructions for changing the version
  - Related commands and documentation

Examples:
  goenv explain              # Explain current version
  goenv explain --verbose    # Include additional context`,
	Example: `  # Basic explanation
  goenv explain

  # Detailed explanation with resolution order
  goenv explain --verbose`,
	RunE: runExplain,
}

var explainFlags struct {
	verbose bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(explainCmd)
	explainCmd.Flags().BoolVarP(&explainFlags.verbose, "verbose", "v", false, "Show additional context and resolution details")
}

func runExplain(cmd *cobra.Command, args []string) error {
	// Validate: explain command takes no positional arguments
	if err := cmdutil.ValidateMaxArgs(args, 0, "no arguments"); err != nil {
		return fmt.Errorf("usage: goenv explain [--verbose]")
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Get current version and source
	version, source, err := mgr.GetCurrentVersion()
	if err != nil {
		// No version set - explain this situation
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n", utils.Emoji("❓"), utils.BoldRed("No Go Version Set"))
		fmt.Fprintln(cmd.OutOrStdout(), "goenv hasn't been configured yet with a default Go version.")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%s What this means:\n", utils.Emoji("🔍"))
		fmt.Fprintln(cmd.OutOrStdout(), "   - No GOENV_VERSION environment variable is set")
		fmt.Fprintln(cmd.OutOrStdout(), "   - No .go-version file found in current directory or parents")
		fmt.Fprintf(cmd.OutOrStdout(), "   - No global version file exists at %s\n", cfg.GlobalVersionFile())
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%s How to fix this:\n", utils.Emoji("💡"))
		fmt.Fprintln(cmd.OutOrStdout(), "   1. Install a Go version:")
		fmt.Fprintln(cmd.OutOrStdout(), "      goenv install 1.22.0")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "   2. Set it as your default:")
		fmt.Fprintln(cmd.OutOrStdout(), "      goenv global 1.22.0")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "   Or use 'goenv use' to do both in one command:\n")
		fmt.Fprintln(cmd.OutOrStdout(), "      goenv use 1.22.0")
		return nil
	}

	// Show current version prominently
	fmt.Fprintf(cmd.OutOrStdout(), "%s Current Go Version: %s\n\n", utils.Emoji("📍"), utils.BoldGreen(version))

	// Check if version is installed
	isInstalled := version == manager.SystemVersion || mgr.IsVersionInstalled(version)
	if !isInstalled {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n", utils.Emoji("⚠️ "), utils.Yellow("Note: This version is not currently installed"))
	}

	// Explain the source based on type
	explainSource(cmd, version, source, cfg)

	// Show version resolution order
	if explainFlags.verbose {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		showResolutionOrder(cmd, cfg)
	}

	// Additional helpful commands
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "%s Related Commands\n", utils.Emoji("📚"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv current        # Show current version")
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv status         # Show goenv status")
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv list           # List installed versions")
	fmt.Fprintln(cmd.OutOrStdout(), "  goenv doctor         # Diagnose configuration issues")

	return nil
}

func explainSource(cmd *cobra.Command, version, source string, cfg *config.Config) {
	// Determine the type of source
	switch {
	case strings.Contains(source, utils.GoenvEnvVarVersion.String()):
		explainEnvironmentVariable(cmd, version)

	case strings.Contains(source, config.VersionFileName) && !strings.Contains(source, cfg.Root):
		// Local .go-version file (not in GOENV_ROOT)
		explainLocalVersionFile(cmd, version, source)

	case strings.Contains(source, config.GoModFileName):
		explainGoMod(cmd, version, source)

	case strings.Contains(source, cfg.Root):
		// Global version file
		explainGlobalVersionFile(cmd, version, source, cfg)

	case source == "":
		explainDefaultSystem(cmd, version)

	default:
		// Unknown source - provide generic explanation
		explainUnknown(cmd, version, source)
	}
}

func explainEnvironmentVariable(cmd *cobra.Command, version string) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Why this version?\n", utils.Emoji("🔍"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "The %s environment variable is set to %s.\n",
		utils.Cyan(utils.GoenvEnvVarVersion.String()), utils.Green(version))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "This takes the HIGHEST PRIORITY over all other version sources.")
	fmt.Fprintln(cmd.OutOrStdout(), "It overrides both local .go-version files and the global default.")
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintf(cmd.OutOrStdout(), "%s How to change it\n", utils.Emoji("💡"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 1: Unset the environment variable (recommended)")
	fmt.Fprintln(cmd.OutOrStdout(), "   This will allow local/global versions to take effect:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   unset GOENV_VERSION")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 2: Change it to a different version")
	fmt.Fprintln(cmd.OutOrStdout(), "   Temporarily override for this shell session:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   export GOENV_VERSION=1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s Note: The GOENV_VERSION variable is typically used for:\n", utils.Emoji("💭"))
	fmt.Fprintln(cmd.OutOrStdout(), "   - Temporary version overrides in scripts")
	fmt.Fprintln(cmd.OutOrStdout(), "   - CI/CD environments")
	fmt.Fprintln(cmd.OutOrStdout(), "   - Advanced use cases")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   For normal use, set project versions with 'goenv local'")
	fmt.Fprintln(cmd.OutOrStdout(), "   and your default version with 'goenv global'.")
}

func explainLocalVersionFile(cmd *cobra.Command, version, source string) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Why this version?\n", utils.Emoji("🔍"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Found a %s file:\n", utils.Cyan(config.VersionFileName))
	fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray(source))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "This is a PROJECT-SPECIFIC version file.")
	fmt.Fprintln(cmd.OutOrStdout(), "It sets the Go version for this directory and all subdirectories.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Priority: Second highest (after GOENV_VERSION environment variable)")
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintf(cmd.OutOrStdout(), "%s How to change it\n", utils.Emoji("💡"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 1: Change this project's version")
	fmt.Fprintf(cmd.OutOrStdout(), "   Update %s to use a different version:\n", filepath.Base(source))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv local 1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 2: Remove project version (use global default)")
	fmt.Fprintln(cmd.OutOrStdout(), "   Delete the .go-version file to fall back to global:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "   rm %s\n", source)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 3: Temporarily override (just for this shell)")
	fmt.Fprintln(cmd.OutOrStdout(), "   Use GOENV_VERSION environment variable:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   export GOENV_VERSION=1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s Tip: Commit .go-version to version control\n", utils.Emoji("💭"))
	fmt.Fprintln(cmd.OutOrStdout(), "   This ensures all team members use the same Go version!")
}

func explainGoMod(cmd *cobra.Command, version, source string) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Why this version?\n", utils.Emoji("🔍"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Found version requirement in %s:\n", utils.Cyan(config.GoModFileName))
	fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray(source))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "goenv read the Go version from your go.mod file's 'go' directive.")
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintf(cmd.OutOrStdout(), "%s How to change it\n", utils.Emoji("💡"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 1: Update go.mod")
	fmt.Fprintln(cmd.OutOrStdout(), "   Edit go.mod to change the Go version:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   go mod edit -go=1.23")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 2: Override with .go-version")
	fmt.Fprintln(cmd.OutOrStdout(), "   Create a local .go-version file (takes priority over go.mod):")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv local 1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s Note: go.mod integration is optional\n", utils.Emoji("💭"))
	fmt.Fprintln(cmd.OutOrStdout(), "   Set GOENV_DISABLE_GOMOD=1 to disable reading from go.mod")
}

func explainGlobalVersionFile(cmd *cobra.Command, version, source string, cfg *config.Config) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Why this version?\n", utils.Emoji("🔍"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "This is your %s version.\n", utils.Cyan("GLOBAL DEFAULT"))
	fmt.Fprintf(cmd.OutOrStdout(), "Set in: %s\n", utils.Gray(source))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Used when:")
	fmt.Fprintln(cmd.OutOrStdout(), "  - No GOENV_VERSION environment variable is set")
	fmt.Fprintln(cmd.OutOrStdout(), "  - No .go-version file found in current directory or parents")
	fmt.Fprintln(cmd.OutOrStdout(), "  - No go.mod file found (or go.mod integration is disabled)")
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintf(cmd.OutOrStdout(), "%s How to change it\n", utils.Emoji("💡"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 1: Change your global default")
	fmt.Fprintln(cmd.OutOrStdout(), "   Set a new default version for all directories:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv global 1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Option 2: Set a project-specific version")
	fmt.Fprintln(cmd.OutOrStdout(), "   Override for just this directory:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv local 1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s Best Practice\n", utils.Emoji("💭"))
	fmt.Fprintln(cmd.OutOrStdout(), "   - Use 'global' for your personal default Go version")
	fmt.Fprintln(cmd.OutOrStdout(), "   - Use 'local' for project-specific versions")
	fmt.Fprintln(cmd.OutOrStdout(), "   - Commit .go-version files to ensure team consistency")
}

func explainDefaultSystem(cmd *cobra.Command, version string) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Why this version?\n", utils.Emoji("🔍"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Using %s Go installation.\n", utils.Cyan("system"))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "This is goenv's fallback when no version is explicitly set.")
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintf(cmd.OutOrStdout(), "%s How to change it\n", utils.Emoji("💡"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Install and set a managed Go version:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv install 1.23.0")
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv global 1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Or use the shortcut:")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv use 1.23.0")
}

func explainUnknown(cmd *cobra.Command, version, source string) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Why this version?\n", utils.Emoji("🔍"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Version %s is set by: %s\n", utils.Green(version), utils.Cyan(source))
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintf(cmd.OutOrStdout(), "%s How to change it\n", utils.Emoji("💡"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Change global version:")
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv global 1.23.0")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Change local version:")
	fmt.Fprintln(cmd.OutOrStdout(), "   goenv local 1.23.0")
}

func showResolutionOrder(cmd *cobra.Command, cfg *config.Config) {
	fmt.Fprintf(cmd.OutOrStdout(), "%s Version Resolution Order\n", utils.Emoji("📚"))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "goenv searches for version settings in this order:")
	fmt.Fprintln(cmd.OutOrStdout())

	// 1. GOENV_VERSION
	envVersion := utils.GoenvEnvVarVersion.UnsafeValue()
	if envVersion != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "1. %s %s\n", utils.Green("✓"), utils.BoldGreen(fmt.Sprintf("%s environment variable", utils.GoenvEnvVarVersion.String())))
		fmt.Fprintf(cmd.OutOrStdout(), "   Currently: %s\n", utils.Cyan(envVersion))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "1. %s %s environment variable\n", utils.Gray("○"), utils.GoenvEnvVarVersion.String())
		fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray("Not set"))
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// 2. Local .go-version
	cwd, _ := os.Getwd()
	localFile := filepath.Join(cwd, config.VersionFileName)
	if utils.PathExists(localFile) {
		fmt.Fprintf(cmd.OutOrStdout(), "2. %s %s\n", utils.Green("✓"), utils.BoldGreen(".go-version in current directory"))
		fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray(localFile))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "2. %s .go-version in current directory\n", utils.Gray("○"))
		fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray("Not found"))
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// 3. Parent directories
	fmt.Fprintf(cmd.OutOrStdout(), "3. %s .go-version in parent directories\n", utils.Gray("○"))
	fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray("Searches up to HOME or root"))
	fmt.Fprintln(cmd.OutOrStdout())

	// 4. go.mod
	gomodFile := filepath.Join(cwd, config.GoModFileName)
	if utils.PathExists(gomodFile) {
		disableGoMod := utils.GoenvEnvVarDisableGomod.UnsafeValue()
		if disableGoMod == "1" || disableGoMod == "true" {
			fmt.Fprintf(cmd.OutOrStdout(), "4. %s go.mod file\n", utils.Gray("○"))
			fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Yellow("Found but disabled by GOENV_DISABLE_GOMOD"))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "4. %s %s\n", utils.Green("✓"), utils.BoldGreen("go.mod file"))
			fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray(gomodFile))
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "4. %s go.mod file\n", utils.Gray("○"))
		fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray("Not found"))
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// 5. Global version
	globalFile := cfg.GlobalVersionFile()
	if utils.PathExists(globalFile) {
		fmt.Fprintf(cmd.OutOrStdout(), "5. %s %s\n", utils.Green("✓"), utils.BoldGreen("Global version file"))
		fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray(globalFile))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "5. %s Global version file\n", utils.Gray("○"))
		fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray(globalFile))
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// 6. System Go
	fmt.Fprintf(cmd.OutOrStdout(), "6. %s System Go (fallback)\n", utils.Gray("○"))
	fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray("Uses Go installed outside goenv"))
	fmt.Fprintln(cmd.OutOrStdout())

	fmt.Fprintf(cmd.OutOrStdout(), "%s The first matching version found wins!\n", utils.Emoji("🎯"))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "To see which source is active, run: goenv current --verbose")
}
