package legacy

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var installedCmd = &cobra.Command{
	Use:          "installed [version]",
	Short:        "Display an installed Go version",
	Long:         `Display the installed Go version, searching for shortcuts if necessary.`,
	RunE:         runInstalled,
	SilenceUsage: true,
}

var installedFlags struct {
	complete bool
}

func init() {
	installedCmd.Flags().BoolVar(&installedFlags.complete, "complete", false, "Show completion options")
	installedCmd.Flags().MarkHidden("complete")
	cmdpkg.RootCmd.AddCommand(installedCmd)
}

func runInstalled(cmd *cobra.Command, args []string) error {
	// Validate: installed command takes 0 or 1 argument
	if err := cmdutil.ValidateMaxArgs(args, 1, "at most one version"); err != nil {
		return fmt.Errorf("usage: goenv installed [version]")
	}

	ctx := cmdutil.GetContexts(cmd)
	mgr := ctx.Manager

	// Handle completion
	if installedFlags.complete {
		fmt.Fprintln(cmd.OutOrStdout(), "latest")
		fmt.Fprintln(cmd.OutOrStdout(), "system")
		versions, err := mgr.ListInstalledVersions()
		if err == nil {
			for _, v := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}
		return nil
	}

	// Check if any versions are installed first (unless system is specified)
	installedVersions, err := mgr.ListInstalledVersions()
	hasInstalledVersions := err == nil && len(installedVersions) > 0

	// Determine which version to check
	var versionSpec string
	if len(args) > 0 {
		versionSpec = args[0]
	} else {
		// No argument given - default to "latest" installed version (matching bash behavior)
		versionSpec = "latest"
	}

	// Handle "system" version
	if versionSpec == manager.SystemVersion {
		// Check if system go exists
		if mgr.HasSystemGo() {
			fmt.Fprintln(cmd.OutOrStdout(), "system")
			return nil
		}
		return errors.SystemVersionNotFound()
	}

	// Ensure we have installed versions for resolution
	if !hasInstalledVersions {
		return fmt.Errorf("goenv: no versions installed")
	}

	// Resolve version spec (latest, major, minor, or exact)
	resolvedVersion, err := mgr.ResolveVersionSpec(versionSpec)
	if err != nil {
		return fmt.Errorf("goenv: version '%s' not installed", versionSpec)
	}

	// Verify the resolved version is actually installed
	if !mgr.IsVersionInstalled(resolvedVersion) {
		return fmt.Errorf("goenv: version '%s' not installed", versionSpec)
	}

	fmt.Fprintln(cmd.OutOrStdout(), resolvedVersion)
	return nil
}
