package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var rootCmd2 = &cobra.Command{
	Use:   "root",
	Short: "Display the root directory where goenv stores its data",
	Long:  "Show the GOENV_ROOT directory path",
	RunE:  runRoot,
}

var prefixCmd = &cobra.Command{
	Use:   "prefix [version]",
	Short: "Display the installation prefix for a Go version",
	Long:  "Show the directory where the specified Go version is installed",
	RunE:  runPrefix,
}

var prefixFlags struct {
	complete bool
}

func init() {
	rootCmd.AddCommand(rootCmd2)
	rootCmd.AddCommand(prefixCmd)
	prefixCmd.Flags().BoolVar(&prefixFlags.complete, "complete", false, "Show completion options")
	_ = prefixCmd.Flags().MarkHidden("complete")
}

func runRoot(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	fmt.Fprintln(cmd.OutOrStdout(), cfg.Root)
	return nil
}

func runPrefix(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Handle completion mode
	if prefixFlags.complete {
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
		if source != "" {
			return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", version, source)
		}
		return fmt.Errorf("goenv: version '%s' not installed", version)
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
		if source != "" {
			return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", version, source)
		}
		return fmt.Errorf("goenv: version '%s' not installed", version)
	}

	fmt.Fprintln(cmd.OutOrStdout(), versionPath)
	return nil
}
