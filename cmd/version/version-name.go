package version

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

var versionNameCmd = &cobra.Command{
	Use:          "version-name",
	Short:        "Show the current Go version",
	Long:         "Display the currently active Go version name without source information",
	RunE:         runVersionName,
	SilenceUsage: true,
	Hidden:       true, // Legacy command - use 'goenv current' instead
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionNameCmd)
	helptext.SetCommandHelp(versionNameCmd)
}

func runVersionName(cmd *cobra.Command, args []string) error {
	// Validate: version-name command takes no arguments
	if err := cmdutil.ValidateExactArgs(args, 0, ""); err != nil {
		return fmt.Errorf("usage: goenv version-name")
	}

	_, mgr := cmdutil.SetupContext()

	// Get resolved version (e.g., "1.25" → "1.25.4")
	resolvedVersion, versionSpec, source, err := mgr.GetCurrentVersionResolved()
	if err != nil {
		// Handle resolution errors
		if versionSpec != "" && source != "" {
			return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", versionSpec, source)
		}
		return errors.FailedTo("determine version", err)
	}

	// Handle multiple versions separated by ':'
	versions := utils.SplitVersions(resolvedVersion)

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
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}

		if hasErrors {
			return errors.SomeVersionsNotInstalled()
		}
	} else {
		// Single version - already resolved and validated by GetCurrentVersionResolved
		fmt.Fprintln(cmd.OutOrStdout(), resolvedVersion)
	}

	return nil
}
