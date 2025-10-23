package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Short:   "Show the current Go version",
	GroupID: "common",
	Long: `Display the currently active Go version and how it was set.

This is the modern command for checking the current version.
The 'version' command still works for backward compatibility.

Examples:
  goenv current           # Show current version
  goenv current -v        # Show with detailed source info
  goenv current --file    # Show just the file that sets the version`,
	RunE: runCurrent,
}

var currentFlags struct {
	verbose bool
	file    bool
}

func init() {
	rootCmd.AddCommand(currentCmd)
	currentCmd.Flags().BoolVarP(&currentFlags.verbose, "verbose", "v", false, "Show detailed information about how version is set")
	currentCmd.Flags().BoolVar(&currentFlags.file, "file", false, "Show only the file that sets the version")
}

func runCurrent(cmd *cobra.Command, args []string) error {
	// Validate: current command takes no positional arguments
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv current [--verbose] [--file]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	version, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("no version set: %w", err)
	}

	// --file flag: just show the source file
	if currentFlags.file {
		if source == "" {
			fmt.Fprintln(cmd.OutOrStdout(), "(none)")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), source)
		}
		return nil
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
			return fmt.Errorf("version not installed")
		}

		// Show version with source info
		if currentFlags.verbose {
			// Verbose mode: show more details
			if source != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "%s (set by %s)\n", version, source)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%s (default)\n", version)
			}
		} else {
			// Normal mode: simple output with source if available
			if source != "" {
				fmt.Fprintf(cmd.OutOrStdout(), "%s (set by %s)\n", version, source)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), version)
			}
		}
	}

	return nil
}
