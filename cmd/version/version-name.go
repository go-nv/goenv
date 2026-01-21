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

	ctx := cmdutil.GetContexts(cmd)
	mgr := ctx.Manager

	// Get current version spec (may contain multiple versions separated by ':')
	versionSpec, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return errors.FailedTo("determine version", err)
	}

	// Handle multiple versions separated by ':'
	versions := utils.SplitVersions(versionSpec)

	if len(versions) > 1 {
		// Multiple versions - resolve and check each one
		hasErrors := false
		var errorMessages []string
		var resolvedVersions []string

		for _, v := range versions {
			// System version doesn't need resolution
			if v == manager.SystemVersion {
				resolvedVersions = append(resolvedVersions, v)
				continue
			}

			// Try to resolve partial versions
			resolved, err := mgr.ResolveVersionSpec(v)
			if err != nil {
				hasErrors = true
				errorMessages = append(errorMessages, fmt.Sprintf("goenv: version '%s' is not installed (set by %s)", v, source))
			} else {
				resolvedVersions = append(resolvedVersions, resolved)
			}
		}

		// Print errors first
		for _, errMsg := range errorMessages {
			cmd.PrintErrln(errMsg)
		}

		// Then print successfully resolved versions
		for _, v := range resolvedVersions {
			fmt.Fprintln(cmd.OutOrStdout(), v)
		}

		if hasErrors {
			return errors.SomeVersionsNotInstalled()
		}
	} else {
		// Single version - use the existing resolution logic
		resolvedVersion, versionSpec, source, err := mgr.GetCurrentVersionResolved()
		if err != nil {
			// Handle resolution errors
			if versionSpec != "" && source != "" {
				return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", versionSpec, source)
			}
			return errors.FailedTo("determine version", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), resolvedVersion)
	}

	return nil
}
