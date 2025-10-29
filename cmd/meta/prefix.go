package meta

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var prefixCmd = &cobra.Command{
	Use:   "prefix [version]",
	Short: "Display the installation prefix for a Go version",
	Long:  "Show the directory where the specified Go version is installed",
	RunE:  RunPrefix,
}

// PrefixFlags holds flags for the prefix command. Exported for testing.
var PrefixFlags struct {
	Complete bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(prefixCmd)
	prefixCmd.Flags().BoolVar(&PrefixFlags.Complete, "complete", false, "Show completion options")
	_ = prefixCmd.Flags().MarkHidden("complete")
}

// RunPrefix executes the prefix command logic. Exported for testing.
func RunPrefix(cmd *cobra.Command, args []string) error {
	// Validate: prefix command takes 0 or 1 argument
	if len(args) > 1 {
		return fmt.Errorf("Usage: goenv prefix [version]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Handle completion mode
	if PrefixFlags.Complete {
		versions, err := mgr.ListInstalledVersions()
		if err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), "latest")
			fmt.Fprintln(cmd.OutOrStdout(), "system")
			for _, v := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}
		return nil
	}

	var version string
	var source string

	if len(args) > 0 {
		// Version specified as argument
		version = args[0]
	} else {
		// Use current version
		currentVersion, versionSource, err := mgr.GetCurrentVersion()
		if err != nil {
			return fmt.Errorf("no version specified and no current version set")
		}
		version = currentVersion
		source = versionSource
	}

	// Resolve version spec (handles "latest", "1", "1.2", etc.)
	resolvedVersion, err := mgr.ResolveVersionSpec(version)
	if err != nil {
		// If resolution fails, provide helpful error
		if version == "system" {
			// Special handling for system version
			if !mgr.HasSystemGo() {
				return fmt.Errorf("goenv: system version not found in PATH")
			}
			// Return directory containing system go
			systemGoDir, err := mgr.GetSystemGoDir()
			if err != nil {
				return fmt.Errorf("goenv: system version not found in PATH")
			}
			fmt.Fprintln(cmd.OutOrStdout(), systemGoDir)
			return nil
		}

		// Check if it's an exact version that's not installed
		return errors.VersionNotInstalledDetailed(version, source, mgr)
	}

	// Handle system version
	if resolvedVersion == "system" {
		if !mgr.HasSystemGo() {
			return fmt.Errorf("goenv: system version not found in PATH")
		}
		systemGoDir, err := mgr.GetSystemGoDir()
		if err != nil {
			return fmt.Errorf("goenv: system version not found in PATH")
		}
		fmt.Fprintln(cmd.OutOrStdout(), systemGoDir)
		return nil
	}

	// Get version path
	versionPath, err := mgr.GetVersionPath(resolvedVersion)
	if err != nil {
		return errors.VersionNotInstalledDetailed(resolvedVersion, source, mgr)
	}

	fmt.Fprintln(cmd.OutOrStdout(), versionPath)
	return nil
}
