package cmd

import (
	"fmt"
	"os"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:     "use <version>",
	Short:   "Install (if needed) and set a Go version",
	GroupID: "common",
	Long: `Install (if needed) and set a Go version for the current directory or globally.

This is a convenience command that combines install, local/global, and optionally
VS Code setup into a single step.

Examples:
  goenv use 1.23.2              # Set local version (installs if needed)
  goenv use 1.23.2 --global     # Set global version (installs if needed)
  goenv use 1.23.2 --vscode     # Set local + configure VS Code
  goenv use latest              # Use latest stable version
  goenv use 1.23.2 --force      # Reinstall even if already installed

This command will:
  1. Check if the version is installed (or corrupted)
  2. Prompt to install/reinstall if needed (unless --yes is set)
  3. Set the version locally (or globally with --global)
  4. Optionally configure VS Code (with --vscode)
  5. Run rehash to update shims`,
	RunE: runUse,
}

var useFlags struct {
	global    bool
	vscode    bool
	vscodeEnv bool
	yes       bool
	force     bool
	quiet     bool
}

func init() {
	rootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolVarP(&useFlags.global, "global", "g", false, "Set as global version instead of local")
	useCmd.Flags().BoolVar(&useFlags.vscode, "vscode", false, "Also configure VS Code workspace")
	useCmd.Flags().BoolVar(&useFlags.vscodeEnv, "vscode-env-vars", false, "Use environment variables in VS Code settings")
	useCmd.Flags().BoolVarP(&useFlags.yes, "yes", "y", false, "Auto-confirm installation prompts")
	useCmd.Flags().BoolVarP(&useFlags.force, "force", "f", false, "Force reinstall even if already installed")
	useCmd.Flags().BoolVarP(&useFlags.quiet, "quiet", "q", false, "Suppress progress output")
	helptext.SetCommandHelp(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("Usage: goenv use <version>")
	}

	versionSpec := args[0]
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Resolve version spec (handles "latest", "stable", etc.)
	version, err := mgr.ResolveVersionSpec(versionSpec)
	if err != nil {
		// If resolution fails, try to use as-is
		version = versionSpec
	}

	if !useFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "🎯 Target version: %s\n", version)
	}

	// Check installation status
	status, err := mgr.CheckVersionStatus(version)
	if err != nil {
		return fmt.Errorf("failed to check version status: %w", err)
	}

	needsInstall := false
	needsReinstall := false

	if !status.Installed {
		needsInstall = true
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "❌ Go %s is not installed\n", version)
		}
	} else if status.Corrupted {
		needsReinstall = true
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "⚠️  Go %s is CORRUPTED (missing go binary)\n", version)
		}
	} else if useFlags.force {
		needsReinstall = true
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "🔄 Force reinstall requested\n")
		}
	} else {
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✅ Go %s is installed\n", version)
		}
	}

	// Handle installation/reinstallation
	if needsInstall || needsReinstall {
		shouldInstall := useFlags.yes

		if !shouldInstall {
			if needsReinstall {
				shouldInstall = manager.PromptForReinstall(version)
			} else {
				reason := fmt.Sprintf("Setting %s version to %s",
					map[bool]string{true: "global", false: "local"}[useFlags.global],
					version)
				shouldInstall = manager.PromptForInstall(version, reason)
			}
		}

		if !shouldInstall {
			return fmt.Errorf("installation cancelled")
		}

		// Install the version
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "\n📦 Installing Go %s...\n", version)
		}

		installer := install.NewInstaller(cfg)
		installer.Quiet = useFlags.quiet
		installer.Verbose = !useFlags.quiet

		if err := installer.Install(version, needsReinstall || useFlags.force); err != nil {
			return fmt.Errorf("installation failed: %w", err)
		}

		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "✅ Installation complete\n\n")
		}
	}

	// Set the version (local or global)
	if useFlags.global {
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "🌍 Setting global version to %s\n", version)
		}
		if err := mgr.SetGlobalVersion(version); err != nil {
			return fmt.Errorf("failed to set global version: %w", err)
		}
	} else {
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "📁 Setting local version to %s\n", version)
		}
		if err := mgr.SetLocalVersion(version); err != nil {
			return fmt.Errorf("failed to set local version: %w", err)
		}
	}

	// Configure VS Code if requested
	if useFlags.vscode {
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "\n🔧 Configuring VS Code...\n")
		}

		vscodeInitFlags.envVars = useFlags.vscodeEnv
		if err := initializeVSCodeWorkspaceWithVersion(cmd, version); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "⚠️  Warning: VS Code configuration failed: %v\n", err)
			fmt.Fprintf(cmd.OutOrStderr(), "   You can manually run: goenv vscode init\n")
		} else {
			if !useFlags.quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "✅ VS Code configured\n")
			}
		}
		vscodeInitFlags.envVars = false
	}

	// Run rehash to update shims
	if !useFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "\n🔄 Updating shims...\n")
	}
	if err := runRehashForUse(cfg); err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "⚠️  Warning: Failed to update shims: %v\n", err)
	}

	// Success message
	if !useFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		fmt.Fprintf(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(cmd.OutOrStdout(), "✨ Success! Now using Go %s\n", version)
		fmt.Fprintf(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Verify: go version\n")

		if useFlags.global {
			fmt.Fprintf(cmd.OutOrStdout(), "Scope:  Global (all directories)\n")
		} else {
			cwd, _ := os.Getwd()
			fmt.Fprintf(cmd.OutOrStdout(), "Scope:  Local (%s)\n", cwd)
		}

		if useFlags.vscode {
			if useFlags.vscodeEnv {
				fmt.Fprintf(cmd.OutOrStdout(), "\n⚠️  Remember: Reopen VS Code from terminal (code .) to use env vars\n")
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n💡 Tip: Reload VS Code window (Cmd+Shift+P → Reload Window)\n")
			}
		}
	}

	return nil
}

// runRehashForUse is a helper to run rehash without output
func runRehashForUse(cfg *config.Config) error {
	shimMgr := shims.NewShimManager(cfg)
	return shimMgr.Rehash()
}
