package legacy

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:                "version",
	Short:              "Show the current Go version (deprecated: use 'goenv current')",
	Long:               "Display the currently active Go version and how it was set",
	RunE:               runVersion,
	DisableFlagParsing: true, // Prevent --version from being treated as a flag
	GroupID:            string(cmdpkg.GroupLegacy),
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionCmd)
	helptext.SetCommandHelp(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	// Deprecation warning
	fmt.Fprintf(cmd.OutOrStderr(), "%sDeprecation warning: 'goenv version' is a legacy command. Use 'goenv current' instead.\n", utils.Emoji("⚠️  "))
	fmt.Fprintf(cmd.OutOrStderr(), "  Modern command: goenv current\n")
	fmt.Fprintf(cmd.OutOrStderr(), "  See: goenv help current\n\n")

	// Validate: version command takes no arguments
	if err := cmdutil.ValidateExactArgs(args, 0, ""); err != nil {
		return fmt.Errorf("usage: goenv version")
	}

	ctx := cmdutil.GetContexts(cmd)
	mgr := ctx.Manager

	version, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return errors.FailedTo("determine version", err)
	}

	// Handle multiple versions separated by ':'
	versions := utils.SplitVersions(version)

	if len(versions) > 1 {
		// Multiple versions - check each one and report errors for missing ones
		hasErrors := false
		var errorMessages []string

		for _, v := range versions {
			if !mgr.IsVersionInstalled(v) && v != manager.SystemVersion {
				hasErrors = true
				errorMessages = append(errorMessages, fmt.Sprintf("goenv: version '%s' is not installed (set by %s)", v, source))
			}
		}

		// Print errors first
		for _, errMsg := range errorMessages {
			cmd.PrintErrln(errMsg)
		}

		// Then print successfully installed versions
		for _, v := range versions {
			if mgr.IsVersionInstalled(v) || v == manager.SystemVersion {
				if source != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s (set by %s)\n", v, source)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), v)
				}
			}
		}

		if hasErrors {
			return errors.SomeVersionsNotInstalled()
		}
	} else {
		// Single version - check if it's installed
		if version != manager.SystemVersion && !mgr.IsVersionInstalled(version) {
			// Version not installed - show warning but still display info
			cmd.PrintErrf("Warning: version '%s' is not installed (set by %s)\n", version, source)
			cmd.PrintErrln("")
			cmd.PrintErrln("To install this version:")
			cmd.PrintErrf("  goenv install %s\n", version)
			cmd.PrintErrln("")
			cmd.PrintErrln("Or use a different version:")

			// Get latest installed version if available
			installed, err := mgr.ListInstalledVersions()
			if err == nil && len(installed) > 0 {
				// Find the latest installed version
				latestInstalled := ""
				for _, v := range installed {
					if latestInstalled == "" {
						latestInstalled = v
					}
				}
				if latestInstalled != "" {
					cmd.PrintErrf("  goenv local %s\n", latestInstalled)
				}
			}

			cmd.PrintErrln("  goenv local system")
			cmd.PrintErrln("  goenv local --unset")
			cmd.PrintErrln("")
		}

		// Always display the version info
		if source != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "%s (set by %s)\n", version, source)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		}
	}

	return nil
}
