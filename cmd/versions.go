package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var versionsCmd = &cobra.Command{
	Use:   "versions",
	Short: "List all installed Go versions",
	Long:  "Show all locally installed Go versions with the current version highlighted",
	RunE:  runVersions,
}

var versionsFlags struct {
	bare        bool
	skipAliases bool
	complete    bool
}

func init() {
	rootCmd.AddCommand(versionsCmd)
	versionsCmd.Flags().BoolVar(&versionsFlags.bare, "bare", false, "Display bare version numbers only")
	versionsCmd.Flags().BoolVar(&versionsFlags.skipAliases, "skip-aliases", false, "Skip aliases")
	versionsCmd.Flags().BoolVar(&versionsFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = versionsCmd.Flags().MarkHidden("complete")
	helptext.SetCommandHelp(versionsCmd)
}

func runVersions(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if versionsFlags.complete {
		fmt.Fprintln(cmd.OutOrStdout(), "--bare")
		fmt.Fprintln(cmd.OutOrStdout(), "--skip-aliases")
		return nil
	}

	// Handle invalid arguments (BATS test expects usage error)
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv versions [--bare] [--skip-aliases]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return fmt.Errorf("failed to list versions: %w", err)
	}

	hasSystemGo := mgr.HasSystemGo()

	// Handle case when no versions installed
	if len(versions) == 0 {
		if !hasSystemGo {
			if versionsFlags.bare {
				// BATS test: bare mode prints empty output when no versions and no system go
				return nil
			}
			return fmt.Errorf("Warning: no Go detected on the system")
		}

		// Only system version available
		if versionsFlags.bare {
			// BATS test: bare mode prints empty when only system available
			return nil
		}

		// Show system version with source info
		globalVersionFile := cfg.GlobalVersionFile()
		fmt.Fprintf(cmd.OutOrStdout(), "* system (set by %s)\n", globalVersionFile)
		return nil
	}

	// Get current version to highlight it
	currentVersion, source, err := mgr.GetCurrentVersion()
	if err != nil {
		// If no version is set, default to empty (no highlighting)
		currentVersion = ""
	}

	// Show system version first (if available and not bare mode)
	// Only show system if: (1) system go exists, OR (2) it's the current version
	showSystem := (hasSystemGo || currentVersion == "system") && !versionsFlags.bare
	if showSystem {
		prefix := "  "
		suffix := ""

		if currentVersion == "system" {
			prefix = "* "
			if source != "" {
				suffix = fmt.Sprintf(" (set by %s)", source)
			} else {
				globalVersionFile := cfg.GlobalVersionFile()
				suffix = fmt.Sprintf(" (set by %s)", globalVersionFile)
			}
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%ssystem%s\n", prefix, suffix)
	}

	// Display installed versions
	for _, version := range versions {
		if versionsFlags.bare {
			fmt.Fprintln(cmd.OutOrStdout(), version)
		} else {
			prefix := "  "
			suffix := ""

			if version == currentVersion {
				prefix = "* "
				if source != "" {
					suffix = fmt.Sprintf(" (set by %s)", source)
				}
			}

			fmt.Fprintf(cmd.OutOrStdout(), "%s%s%s\n", prefix, version, suffix)
		}
	}

	return nil
}
