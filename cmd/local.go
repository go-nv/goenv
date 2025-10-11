package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var localCmd = &cobra.Command{
	Use:   "local [version]",
	Short: "Set or show the local Go version for this directory",
	Long:  "Set a local Go version for the current directory by creating a .go-version file",
	RunE:  runLocal,
}

var localFlags struct {
	unset bool
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.SilenceUsage = true
	localCmd.Flags().BoolVar(&localFlags.unset, "unset", false, "Unset the local Go version")
}

func runLocal(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	if localFlags.unset {
		if len(args) > 0 {
			return fmt.Errorf("Usage: goenv local [<version>]")
		}

		if err := mgr.UnsetLocalVersion(); err != nil {
			return err
		}

		return nil
	}

	if len(args) == 0 {
		version, err := mgr.GetLocalVersion()
		if err != nil {
			return fmt.Errorf("goenv: no local version configured for this directory")
		}
		cmd.Println(version)
		return nil
	}

	spec := args[0]

	resolvedVersion, err := mgr.ResolveVersionSpec(spec)
	if err != nil {
		return err
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Setting local version to %s (input: %s)\n", resolvedVersion, spec)
	}

	if err := mgr.SetLocalVersion(resolvedVersion); err != nil {
		return err
	}

	return nil
}
