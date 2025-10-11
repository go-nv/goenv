package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var globalCmd = &cobra.Command{
	Use:   "global [version]",
	Short: "Set or show the global Go version",
	Long:  "Set the global Go version that is used in all directories unless overridden by a local .go-version file",
	RunE:  runGlobal,
}

func init() {
	rootCmd.AddCommand(globalCmd)
}

func runGlobal(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	if len(args) == 0 {
		// Show current global version
		version, err := mgr.GetGlobalVersion()
		if err != nil {
			return fmt.Errorf("no global version set")
		}
		cmd.Println(version)
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
