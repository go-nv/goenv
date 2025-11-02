package legacy

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/lifecycle"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var globalCmd = &cobra.Command{
	Use:   "global [version]",
	Short: "Set or show the global Go version (deprecated: use 'goenv use --global')",
	Long:  "Set the global Go version that is used in all directories unless overridden by a local .go-version file",
	Example: `  # Show current global version
  goenv global

  # Set global version
  goenv global 1.21.5

  # Use system Go
  goenv global system`,
	RunE:    RunGlobal,
	GroupID: string(cmdpkg.GroupLegacy),
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
	fmt.Fprintf(cmd.OutOrStderr(), "%sDeprecation warning: 'goenv global' is a legacy command. Use 'goenv use <version> --global' instead.\n", utils.Emoji("⚠️  "))
	fmt.Fprintf(cmd.OutOrStderr(), "  Modern command: goenv use <version> --global\n")
	fmt.Fprintf(cmd.OutOrStderr(), "  See: goenv help use\n\n")

	// Validate: global command takes 0 or 1 argument
	if err := cmdutil.ValidateMaxArgs(args, 1, "at most one version"); err != nil {
		return fmt.Errorf("usage: goenv global [version]")
	}

	cfg, mgr := cmdutil.SetupContext()

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
		versions := utils.SplitVersions(version)
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

	// Handle installation/reinstallation using consolidated helper
	// Note: Quiet is true to maintain legacy command's silent behavior
	_, err = manager.HandleVersionInstallation(manager.InstallOptions{
		Config:      cfg,
		Manager:     mgr,
		Version:     resolvedVersion,
		AutoInstall: false,
		Force:       false,
		Quiet:       true,
		Reason:      fmt.Sprintf("Setting global version to %s", resolvedVersion),
		Writer:      cmd.OutOrStdout(),
	})
	if err != nil {
		return err
	}

	if cfg.Debug {
		fmt.Printf("Debug: Setting global version to %s\n", resolvedVersion)
	}

	if err := mgr.SetGlobalVersion(resolvedVersion); err != nil {
		return err
	}

	// Check if version is outdated and show warning
	displayVersionWarning(cmd, resolvedVersion)

	// Silent success - no output when setting version (matches BATS behavior)
	return nil
}

// displayVersionWarning shows a warning if the version is outdated or EOL
func displayVersionWarning(cmd *cobra.Command, version string) {
	// Skip warning for system version
	if version == manager.SystemVersion {
		return
	}

	warning := lifecycle.FormatWarning(version)
	if warning != "" {
		fmt.Fprintf(cmd.OutOrStderr(), "\n%s%s\n", utils.EmojiOr("⚠️  ", "Warning: "), utils.Yellow(warning))
		fmt.Fprintln(cmd.OutOrStderr())
	}
}
