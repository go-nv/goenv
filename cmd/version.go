package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var versionCmd = &cobra.Command{
	Use:                "version",
	Short:              "Show the current Go version",
	Long:               "Display the currently active Go version and how it was set",
	RunE:               runVersion,
	DisableFlagParsing: true, // Prevent --version from being treated as a flag
}

func init() {
	rootCmd.AddCommand(versionCmd)
	helptext.SetCommandHelp(versionCmd)
}

func runVersion(cmd *cobra.Command, args []string) error {
	// Validate: version command takes no arguments
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv version")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	version, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("no version set: %w", err)
	}

	// Handle multiple versions separated by ':'
	versions := splitVersions(version)

	if len(versions) > 1 {
		// Multiple versions - check each one and report errors for missing ones
		hasErrors := false
		var errorMessages []string

		for _, v := range versions {
			if !mgr.IsVersionInstalled(v) && v != "system" {
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
			if mgr.IsVersionInstalled(v) || v == "system" {
				if source != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s (set by %s)\n", v, source)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), v)
				}
			}
		}

		if hasErrors {
			return fmt.Errorf("some versions are not installed")
		}
	} else {
		// Single version - check if it's installed
		if version != "system" && !mgr.IsVersionInstalled(version) {
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

// splitVersions splits a version string by ':' delimiter
func splitVersions(version string) []string {
	if version == "" {
		return []string{}
	}

	result := []string{}
	current := ""

	for _, ch := range version {
		if ch == ':' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}
