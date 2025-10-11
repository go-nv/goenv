package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var versionNameCmd = &cobra.Command{
	Use:          "version-name",
	Short:        "Show the current Go version",
	Long:         "Display the currently active Go version name without source information",
	RunE:         runVersionName,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(versionNameCmd)
}

func runVersionName(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	version, _, err := mgr.GetCurrentVersion()
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
				// Get source info for error message
				_, source, _ := mgr.GetCurrentVersion()
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
				cmd.Println(v)
			}
		}

		if hasErrors {
			return fmt.Errorf("some versions are not installed")
		}
	} else {
		// Single version - validate it
		if version != "system" && !mgr.IsVersionInstalled(version) {
			_, source, _ := mgr.GetCurrentVersion()
			return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", version, source)
		}
		cmd.Println(version)
	}

	return nil
}
