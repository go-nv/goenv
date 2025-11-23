package legacy

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/cmd/integrations"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var localCmd = &cobra.Command{
	Use:   "local [version]",
	Short: "Set or show the local Go version for this directory (deprecated: use 'goenv use')",
	Long:  "Set a local Go version for the current directory by creating a .go-version file",
	Example: `  # Show current local version
  goenv local

  # Set local version for this project
  goenv local 1.21.5

  # Remove local version (use global)
  goenv local --unset

  # Set from go.mod file
  goenv local --from-gomod

  # Set and configure VS Code
  goenv local 1.21.5 --vscode`,
	RunE:    RunLocal,
	GroupID: string(cmdpkg.GroupLegacy),
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
	cmdpkg.RootCmd.AddCommand(localCmd)
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

// RunLocal executes the local command logic. Exported for testing.
func RunLocal(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if localFlags.complete {
		_, mgr := cmdutil.SetupContext()
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
		return fmt.Errorf("usage: goenv local [<version>]")
	}
	if localFlags.fromGoMod && len(args) > 0 {
		return fmt.Errorf("--from-gomod flag cannot be used with a version argument")
	}

	cfg, mgr := cmdutil.SetupContext()

	if localFlags.unset {
		if len(args) > 0 {
			return fmt.Errorf("usage: goenv local [<version>]")
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
					return errors.FailedTo("install version", err)
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
		gomodPath := filepath.Join(cwd, config.GoModFileName)

		if utils.FileNotExists(gomodPath) {
			return fmt.Errorf("--from-gomod specified but no go.mod file found in current directory")
		}

		versionFromGoMod, err := manager.ParseGoModVersion(gomodPath)
		if err != nil {
			return errors.FailedTo("read version from go.mod", err)
		}

		spec = versionFromGoMod
		resolvedVersion = spec

		// Handle installation/reinstallation using consolidated helper
		// Note: Quiet is true to maintain legacy command's silent behavior
		_, err = manager.HandleVersionInstallation(manager.InstallOptions{
			Config:      cfg,
			Manager:     mgr,
			Version:     resolvedVersion,
			AutoInstall: false,
			Force:       false,
			Quiet:       true,
			Reason:      "Setting local version from go.mod",
			Writer:      cmd.OutOrStdout(),
		})
		if err != nil {
			return err
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

		// Handle installation/reinstallation using consolidated helper
		// Note: Quiet is true to maintain legacy command's silent behavior
		_, err = manager.HandleVersionInstallation(manager.InstallOptions{
			Config:      cfg,
			Manager:     mgr,
			Version:     resolvedVersion,
			AutoInstall: false,
			Force:       false,
			Quiet:       true,
			Reason:      fmt.Sprintf("Setting local version to %s", resolvedVersion),
			Writer:      cmd.OutOrStdout(),
		})
		if err != nil {
			return err
		}
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Setting local version to %s (input: %s)\n", resolvedVersion, spec)
	}

	if err := mgr.SetLocalVersion(resolvedVersion); err != nil {
		return err
	}

	// Check if version is outdated and show warning
	displayVersionWarning(cmd, resolvedVersion)

	// Check if go.mod exists and warn about version mismatch
	cwd, _ := os.Getwd()
	gomodPath := filepath.Join(cwd, config.GoModFileName)
	if utils.FileExists(gomodPath) {
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

	// Automatically initialize VS Code if --vscode flag is set OR auto-detect workspace
	shouldConfigureVSCode := localFlags.vscode
	
	// Auto-detect VS Code workspace if flag not explicitly set
	if !localFlags.vscode {
		cwd, _ := os.Getwd()
		vscodeSettingsPath := filepath.Join(cwd, ".vscode", "settings.json")
		
		// Check if VS Code workspace exists
		if _, err := os.Stat(vscodeSettingsPath); err == nil {
			// Check VS Code setting first, then env var
			autoSync := false
			
			// Read the settings file to check for goenv.autoSync
			if data, err := os.ReadFile(vscodeSettingsPath); err == nil {
				// Simple JSON check - look for "goenv.autoSync": true
				settingsStr := string(data)
				if strings.Contains(settingsStr, `"goenv.autoSync"`) && 
				   (strings.Contains(settingsStr, `"goenv.autoSync": true`) || 
				    strings.Contains(settingsStr, `"goenv.autoSync":true`)) {
					autoSync = true
				}
			}
			
			// Fall back to environment variable
			if !autoSync {
				envAutoSync := os.Getenv("GOENV_VSCODE_AUTO_SYNC")
				if envAutoSync == "1" || envAutoSync == "true" {
					autoSync = true
				}
			}
			
			if autoSync {
				// Auto-sync enabled
				shouldConfigureVSCode = true
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintf(cmd.OutOrStdout(), "%sAuto-updating VS Code workspace (goenv.autoSync: true)...\n", utils.Emoji("🔧 "))
			} else {
				// Prompt user
				fmt.Fprintln(cmd.OutOrStdout())
				fmt.Fprintf(cmd.OutOrStdout(), "%sDetected VS Code workspace. Update settings for Go %s? [Y/n]: ", utils.Emoji("💡 "), resolvedVersion)
				var response string
				fmt.Fscanln(cmd.InOrStdin(), &response)
				
				// Default to Yes if user just presses Enter
				if response == "" || response == "y" || response == "Y" || response == "yes" {
					shouldConfigureVSCode = true
				}
			}
		}
	}
	
	if shouldConfigureVSCode {
		if !localFlags.vscode {
			fmt.Fprintln(cmd.OutOrStdout())
		}
		fmt.Fprintln(cmd.OutOrStdout(), "Initializing VS Code workspace...")

		// Use absolute paths by default (better UX), unless user wants env vars
		if localFlags.vscodeEnvVars {
			integrations.VSCodeInitFlags.EnvVars = true
		} else {
			integrations.VSCodeInitFlags.EnvVars = false
		}

		// Call vscode init functionality with the resolved version
		if err := integrations.InitializeVSCodeWorkspaceWithVersion(cmd, resolvedVersion); err != nil {
			// Don't fail the whole command if VS Code init fails
			fmt.Fprintf(cmd.OutOrStdout(), "%sWarning: VS Code initialization failed: %v\n", utils.Emoji("⚠️  "), err)
			fmt.Fprintln(cmd.OutOrStdout(), "   You can manually run: goenv vscode init")
		}

		// Reset the flag
		integrations.VSCodeInitFlags.EnvVars = false

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
