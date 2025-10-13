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
		// Single version
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
