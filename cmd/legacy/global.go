package legacy

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var globalCmd = &cobra.Command{
	Use:    "global [version]",
	Short:  "Set or show the global Go version",
	Long:   "Set the global Go version that is used in all directories unless overridden by a local .go-version file",
	RunE:   RunGlobal,
	Hidden: true, // Legacy command - use 'goenv use <version> --global' instead
}

var globalFlags struct {
	complete bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(globalCmd)
	globalCmd.Flags().BoolVar(&globalFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = globalCmd.Flags().MarkHidden("complete")
	helptext.SetCommandHelp(globalCmd)
}

// RunGlobal executes the global command logic. Exported for testing.
func RunGlobal(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if globalFlags.complete {
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
	fmt.Fprintf(cmd.OutOrStderr(), "%sDeprecation warning: 'goenv global' is a legacy command. Use 'goenv use <version> --global' instead.\n", utils.Emoji("⚠️  "))
	fmt.Fprintf(cmd.OutOrStderr(), "  Modern command: goenv use <version> --global\n")
	fmt.Fprintf(cmd.OutOrStderr(), "  See: goenv help use\n\n")

	// Validate: global command takes 0 or 1 argument
	if len(args) > 1 {
		return fmt.Errorf("Usage: goenv global [version]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	if len(args) == 0 {
		// Show current global version(s) - read raw file to preserve multi-line format
		globalFile := cfg.GlobalVersionFile()
		version, err := mgr.ReadVersionFile(globalFile)
		if err != nil {
			// If file doesn't exist or is empty, default to system
			version = "system"
		}
		// Convert colon-separated to newline-separated for display
		// (ReadVersionFile joins multiple lines with colons)
		versions := SplitVersions(version)
		for _, v := range versions {
			fmt.Fprintln(cmd.OutOrStdout(), v)
		}
		return nil
	}

	// Set global version
	version := args[0]

	// Resolve version spec
	resolvedVersion, err := mgr.ResolveVersionSpec(version)
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
		if manager.PromptForInstall(resolvedVersion, fmt.Sprintf("Setting global version to %s", resolvedVersion)) {
			fmt.Fprintf(cmd.OutOrStdout(), "\n%sInstalling Go %s...\n", utils.Emoji("📦 "), resolvedVersion)

			installer := install.NewInstaller(cfg)
			if err := installer.Install(resolvedVersion, false); err != nil {
				return fmt.Errorf("installation failed: %w", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%sInstallation complete\n\n", utils.Emoji("✅ "))
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), manager.GetInstallHint(resolvedVersion, "global"))
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

	if cfg.Debug {
		fmt.Printf("Debug: Setting global version to %s\n", resolvedVersion)
	}

	if err := mgr.SetGlobalVersion(resolvedVersion); err != nil {
		return err
	}

	// Silent success - no output when setting version (matches BATS behavior)
	return nil
}
