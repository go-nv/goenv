package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var localCmd = &cobra.Command{
	Use:    "local [version]",
	Short:  "Set or show the local Go version for this directory",
	Long:   "Set a local Go version for the current directory by creating a .go-version file",
	RunE:   runLocal,
	Hidden: true, // Legacy command - use 'goenv use <version>' instead
}

var localFlags struct {
	unset         bool
	complete      bool
	vscode        bool
	vscodeEnvVars bool
	sync          bool
	fromGoMod     bool
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.SilenceUsage = true
	localCmd.Flags().BoolVarP(&localFlags.unset, "unset", "u", false, "Unset the local Go version")
	localCmd.Flags().BoolVar(&localFlags.sync, "sync", false, "Ensure the version from .go-version is installed")
	localCmd.Flags().BoolVar(&localFlags.fromGoMod, "from-gomod", false, "Set version from go.mod file (version must be installed)")
	localCmd.Flags().BoolVar(&localFlags.vscode, "vscode", false, "Also initialize VS Code workspace settings (uses absolute paths by default)")
	localCmd.Flags().BoolVar(&localFlags.vscodeEnvVars, "vscode-env-vars", false, "Use environment variables in VS Code settings (requires terminal launch)")
	localCmd.Flags().BoolVar(&localFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = localCmd.Flags().MarkHidden("complete")
	helptext.SetCommandHelp(localCmd)
}

func runLocal(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if localFlags.complete {
		cfg := config.Load()
		mgr := manager.NewManager(cfg)
		versions, err := mgr.ListInstalledVersions()
		if err == nil {
			for _, v := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "system")
		return nil
	}

	// Deprecation warning
	fmt.Fprintf(cmd.OutOrStderr(), "%sDeprecation warning: 'goenv local' is a legacy command. Use 'goenv use <version>' instead.\n", utils.Emoji("⚠️  "))
	fmt.Fprintf(cmd.OutOrStderr(), "  Modern command: goenv use <version>\n")
	fmt.Fprintf(cmd.OutOrStderr(), "  See: goenv help use\n\n")

	// Validate: local command takes 0 or 1 argument (not including flags)
	// Exception: with --from-gomod, takes 0 arguments
	if !localFlags.fromGoMod && len(args) > 1 {
		return fmt.Errorf("Usage: goenv local [<version>]")
	}
	if localFlags.fromGoMod && len(args) > 0 {
		return fmt.Errorf("--from-gomod flag cannot be used with a version argument")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	if localFlags.unset {
		if len(args) > 0 {
			return fmt.Errorf("Usage: goenv local [<version>]")
		}

		if err := mgr.UnsetLocalVersion(); err != nil {
			return err
		}

		return nil
	}

	if len(args) == 0 && !localFlags.fromGoMod {
		version, err := mgr.GetLocalVersion()
		if err != nil {
			return fmt.Errorf("goenv: no local version configured for this directory")
		}

		// If --sync flag is set, ensure version is installed
		if localFlags.sync {
			fmt.Fprintf(cmd.OutOrStdout(), "Found .go-version: %s\n", version)

			// Check if installed
			if !mgr.IsVersionInstalled(version) {
				fmt.Fprintf(cmd.OutOrStdout(), "%sGo %s is not installed\n", utils.Emoji("⚠️  "), version)
				fmt.Fprintf(cmd.OutOrStdout(), "Installing Go %s...\n", version)

				// Find and execute install command
				installCmd := cmd.Root().Commands()[0]
				for _, c := range cmd.Root().Commands() {
					if c.Name() == "install" {
						installCmd = c
						break
					}
				}

				// Execute install
				installCmd.SetArgs([]string{version})
				if err := installCmd.Execute(); err != nil {
					return fmt.Errorf("installation failed: %w", err)
				}
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%sGo %s is installed\n", utils.Emoji("✓ "), version)
			}

			return nil
		}

		fmt.Fprintln(cmd.OutOrStdout(), version)
		return nil
	}

	var spec string
	var resolvedVersion string

	// If --from-gomod flag is set, try to read version from go.mod
	if localFlags.fromGoMod {
		cwd, _ := os.Getwd()
		gomodPath := filepath.Join(cwd, "go.mod")

		if _, err := os.Stat(gomodPath); os.IsNotExist(err) {
			return fmt.Errorf("--from-gomod specified but no go.mod file found in current directory")
		}

		versionFromGoMod, err := manager.ParseGoModVersion(gomodPath)
		if err != nil {
			return fmt.Errorf("failed to read version from go.mod: %w", err)
		}

		spec = versionFromGoMod
		resolvedVersion = spec

		// Check installation status
		status, err := mgr.CheckVersionStatus(resolvedVersion)
		if err != nil {
			return fmt.Errorf("failed to check version status: %w", err)
		}

		if !status.Installed {
			fmt.Fprintf(cmd.OutOrStdout(), "Go %s (from go.mod) is not installed\n\n", resolvedVersion)

			// Prompt for installation
			if manager.PromptForInstall(resolvedVersion, "Setting local version from go.mod") {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sInstalling Go %s...\n", utils.Emoji("📦 "), resolvedVersion)

				installer := install.NewInstaller(cfg)
				if err := installer.Install(resolvedVersion, false); err != nil {
					return fmt.Errorf("installation failed: %w", err)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%sInstallation complete\n\n", utils.Emoji("✅ "))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\nTo install later:\n")
				fmt.Fprintf(cmd.OutOrStdout(), "  goenv install %s && goenv local %s\n", resolvedVersion, resolvedVersion)
				return fmt.Errorf("installation cancelled")
			}
		} else if status.Corrupted {
			if manager.PromptForReinstall(resolvedVersion) {
				installer := install.NewInstaller(cfg)
				if err := installer.Install(resolvedVersion, true); err != nil {
					return fmt.Errorf("reinstallation failed: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%sReinstallation complete\n\n", utils.Emoji("✅ "))
			} else {
				return fmt.Errorf("corrupted installation - reinstall cancelled")
			}
		}
	} else {
		if len(args) == 0 {
			return fmt.Errorf("no version specified (use VERSION argument or --from-gomod flag)")
		}
		spec = args[0]

		// For manual version spec, resolve and check status
		var err error
		resolvedVersion, err = mgr.ResolveVersionSpec(spec)
		if err != nil {
			return err
		}

		// Check installation status
		status, err := mgr.CheckVersionStatus(resolvedVersion)
		if err != nil {
			return fmt.Errorf("failed to check version status: %w", err)
		}

		if !status.Installed {
			fmt.Fprintf(cmd.OutOrStdout(), "Go %s is not installed\n\n", resolvedVersion)

			// Prompt for installation
			if manager.PromptForInstall(resolvedVersion, fmt.Sprintf("Setting local version to %s", resolvedVersion)) {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sInstalling Go %s...\n", utils.Emoji("📦 "), resolvedVersion)

				installer := install.NewInstaller(cfg)
				if err := installer.Install(resolvedVersion, false); err != nil {
					return fmt.Errorf("installation failed: %w", err)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "%sInstallation complete\n\n", utils.Emoji("✅ "))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), manager.GetInstallHint(resolvedVersion, "local"))
				return fmt.Errorf("installation cancelled")
			}
		} else if status.Corrupted {
			if manager.PromptForReinstall(resolvedVersion) {
				installer := install.NewInstaller(cfg)
				if err := installer.Install(resolvedVersion, true); err != nil {
					return fmt.Errorf("reinstallation failed: %w", err)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "%sReinstallation complete\n\n", utils.Emoji("✅ "))
			} else {
				return fmt.Errorf("corrupted installation - reinstall cancelled")
			}
		}
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Setting local version to %s (input: %s)\n", resolvedVersion, spec)
	}

	if err := mgr.SetLocalVersion(resolvedVersion); err != nil {
		return err
	}

	// Check if go.mod exists and warn about version mismatch
	cwd, _ := os.Getwd()
	gomodPath := filepath.Join(cwd, "go.mod")
	if _, err := os.Stat(gomodPath); err == nil {
		// go.mod exists, check version compatibility
		requiredVersion, err := manager.ParseGoModVersion(gomodPath)
		if err == nil {
			if !manager.VersionSatisfies(resolvedVersion, requiredVersion) {
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintf(cmd.OutOrStdout(), "%sWarning: go.mod requires Go %s but you set version to %s\n", utils.Emoji("⚠️  "), requiredVersion, resolvedVersion)
				fmt.Fprintln(cmd.OutOrStdout(), "   This may cause build errors.")
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintln(cmd.OutOrStdout(), "   To use the version from go.mod:")
				fmt.Fprintf(cmd.OutOrStdout(), "   goenv local %s\n", requiredVersion)
			}
		}
	}

	// Automatically initialize VS Code if --vscode flag is set
	if localFlags.vscode {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Initializing VS Code workspace...")

		// Use absolute paths by default (better UX), unless user wants env vars
		if localFlags.vscodeEnvVars {
			vscodeInitFlags.envVars = true
		} else {
			vscodeInitFlags.envVars = false
		}

		// Call vscode init functionality with the resolved version
		if err := initializeVSCodeWorkspaceWithVersion(cmd, resolvedVersion); err != nil {
			// Don't fail the whole command if VS Code init fails
			fmt.Fprintf(cmd.OutOrStdout(), "%sWarning: VS Code initialization failed: %v\n", utils.Emoji("⚠️  "), err)
			fmt.Fprintln(cmd.OutOrStdout(), "   You can manually run: goenv vscode init")
		}

		// Reset the flag
		vscodeInitFlags.envVars = false

		// Note about updating when version changes (only for absolute paths mode)
		if !localFlags.vscodeEnvVars {
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "%sTip: When you change Go versions, re-run 'goenv local VERSION --vscode' to update VS Code\n", utils.Emoji("💡 "))
		} else {
			// Important note about environment refresh for env vars mode
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintf(cmd.OutOrStdout(), "%sImportant: To use Go %s in VS Code:\n", utils.Emoji("⚠️  "), resolvedVersion)
			fmt.Fprintln(cmd.OutOrStdout(), "   1. Close VS Code completely")
			fmt.Fprintln(cmd.OutOrStdout(), "   2. Reopen from terminal:  code .")
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "   This ensures VS Code inherits the updated GOROOT environment variable.")
		}
	}

	return nil
}
