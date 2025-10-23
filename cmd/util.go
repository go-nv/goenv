package cmd

import (
	"fmt"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

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
	rootCmd.AddCommand(prefixCmd)
	prefixCmd.Flags().BoolVar(&prefixFlags.complete, "complete", false, "Show completion options")
	_ = prefixCmd.Flags().MarkHidden("complete")
}

func runPrefix(cmd *cobra.Command, args []string) error {
	// Validate: prefix command takes 0 or 1 argument
	if len(args) > 1 {
		return fmt.Errorf("Usage: goenv prefix [version]")
	}

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
		return formatVersionNotInstalledError(version, source, mgr)
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
		return formatVersionNotInstalledError(resolvedVersion, source, mgr)
	}

	fmt.Fprintln(cmd.OutOrStdout(), versionPath)
	return nil
}

// formatVersionNotInstalledError creates an enhanced error message with helpful suggestions
// for when a Go version is not installed
func formatVersionNotInstalledError(version, source string, mgr *manager.Manager) error {
	var sb strings.Builder

	// Main error message
	if source != "" {
		sb.WriteString(fmt.Sprintf("goenv: version '%s' is not installed (set by %s)\n", version, source))
	} else {
		sb.WriteString(fmt.Sprintf("goenv: version '%s' is not installed\n", version))
	}

	sb.WriteString("\n")

	// Suggest installation
	sb.WriteString("To install this version:\n")
	sb.WriteString(fmt.Sprintf("  goenv install %s\n", version))

	sb.WriteString("\n")

	// Suggest alternatives
	sb.WriteString("Or use a different version:\n")

	// Get latest installed version if available
	installed, err := mgr.ListInstalledVersions()
	if err == nil && len(installed) > 0 {
		// Find the latest installed version
		latestInstalled := ""
		for _, v := range installed {
			if latestInstalled == "" || compareGoVersionsSimple(v, latestInstalled) > 0 {
				latestInstalled = v
			}
		}
		if latestInstalled != "" {
			sb.WriteString(fmt.Sprintf("  goenv local %s          # Use %s in this directory\n", latestInstalled, latestInstalled))
		}
	}

	sb.WriteString("  goenv local system          # Use system Go\n")
	sb.WriteString("  goenv local --unset         # Remove .go-version file\n")

	// Return error without the trailing newline
	return fmt.Errorf("%s", strings.TrimRight(sb.String(), "\n"))
}

// compareGoVersionsSimple is a simplified version comparison for error messages
// Returns positive if v1 > v2, negative if v1 < v2, zero if equal
func compareGoVersionsSimple(v1, v2 string) int {
	// Simple comparison - good enough for displaying latest version
	if v1 > v2 {
		return 1
	} else if v1 < v2 {
		return -1
	}
	return 0
}
