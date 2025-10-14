package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var globalCmd = &cobra.Command{
	Use:   "global [version]",
	Short: "Set or show the global Go version",
	Long:  "Set the global Go version that is used in all directories unless overridden by a local .go-version file",
	RunE:  runGlobal,
}

var globalFlags struct {
	complete bool
}

func init() {
	rootCmd.AddCommand(globalCmd)
	globalCmd.Flags().BoolVar(&globalFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = globalCmd.Flags().MarkHidden("complete")
	helptext.SetCommandHelp(globalCmd)
}

func runGlobal(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if globalFlags.complete {
		cfg := config.Load()
		mgr := manager.NewManager(cfg)
		versions, err := mgr.ListInstalledVersions()
		if err == nil {
			for _, v := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "system")
		return nil
	}

	// Validate: global command takes 0 or 1 argument
	if len(args) > 1 {
		return fmt.Errorf("Usage: goenv global [version]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	if len(args) == 0 {
		// Show current global version(s) - read raw file to preserve multi-line format
		globalFile := cfg.GlobalVersionFile()
		version, err := mgr.ReadVersionFile(globalFile)
		if err != nil {
			// If file doesn't exist or is empty, default to system
			version = "system"
		}
		// Convert colon-separated to newline-separated for display
		// (ReadVersionFile joins multiple lines with colons)
		versions := splitVersions(version)
		for _, v := range versions {
			fmt.Fprintln(cmd.OutOrStdout(), v)
		}
		return nil
	}

	// Set global version
	version := args[0]

	if cfg.Debug {
		fmt.Printf("Debug: Setting global version to %s\n", version)
	}

	if err := mgr.SetGlobalVersion(version); err != nil {
		return err
	}

	// Silent success - no output when setting version (matches BATS behavior)
	return nil
}
