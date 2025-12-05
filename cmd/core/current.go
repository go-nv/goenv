package core

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var currentCmd = &cobra.Command{
	Use:     "current",
	Short:   "Show the current Go version",
	GroupID: string(cmdpkg.GroupVersions),
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
	cmdpkg.RootCmd.AddCommand(currentCmd)
	currentCmd.Flags().BoolVarP(&currentFlags.verbose, "verbose", "v", false, "Show detailed information about how version is set")
	currentCmd.Flags().BoolVar(&currentFlags.file, "file", false, "Show only the file that sets the version")
}

func runCurrent(cmd *cobra.Command, args []string) error {
	// Validate: current command takes no positional arguments
	if err := cmdutil.ValidateMaxArgs(args, 0, "no arguments"); err != nil {
		return fmt.Errorf("usage: goenv current [--verbose] [--file]")
	}

	_, mgr := cmdutil.SetupContext()

	// Get resolved version (e.g., "1.25" → "1.25.4")
	resolvedVersion, versionSpec, source, err := mgr.GetCurrentVersionResolved()
	if err != nil {
		if versionSpec != "" && source != "" {
			return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", versionSpec, source)
		}
		return errors.FailedTo("determine active version", err)
	}

	// Use the resolved version for display
	version := resolvedVersion

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
	versions := utils.SplitVersions(version)

	if len(versions) > 1 {
		// Multiple versions - check each one and report errors for missing ones
		hasErrors := false
		var errorMessages []string

		for _, v := range versions {
			if !mgr.IsVersionInstalled(v) && v != manager.SystemVersion {
				hasErrors = true
				errorMessages = append(errorMessages, errors.VersionNotInstalled(v, source).Error())
			}
		}

		// Print errors first
		for _, errMsg := range errorMessages {
			cmd.PrintErrln(errMsg)
		}

		// Then print successfully installed versions
		for _, v := range versions {
			if mgr.IsVersionInstalled(v) || v == manager.SystemVersion {
				if source != "" {
					fmt.Fprintf(cmd.OutOrStdout(), "%s (set by %s)\n", v, source)
				} else {
					fmt.Fprintln(cmd.OutOrStdout(), v)
				}
			}
		}

		if hasErrors {
			return errors.SomeVersionsNotInstalled()
		}
	} else {
		// Single version - check if it's installed
		if version != manager.SystemVersion && !mgr.IsVersionInstalled(version) {
			// Version not installed - return detailed error with suggestions
			installed, _ := mgr.ListInstalledVersions()
			return errors.VersionNotInstalledDetailed(version, source, installed)
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
