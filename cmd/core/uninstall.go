package core

import (
	cmdhooks "github.com/go-nv/goenv/cmd/hooks"
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <version>",
	Short:   "Uninstall a Go version",
	GroupID: "common",
	Long:    "Remove an installed Go version from the system",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runUninstall,
}

var uninstallFlags struct {
	complete bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVar(&uninstallFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = uninstallCmd.Flags().MarkHidden("complete")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if uninstallFlags.complete {
		cfg := config.Load()
		mgr := manager.NewManager(cfg)
		versions, err := mgr.ListInstalledVersions()
		if err == nil {
			for _, v := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}
		return nil
	}

	// Validate: uninstall requires a version argument
	if len(args) != 1 {
		return fmt.Errorf("Usage: goenv uninstall <version>")
	}

	cfg := config.Load()
	installer := install.NewInstaller(cfg)

	goVersion := args[0]

	if cfg.Debug {
		fmt.Printf("Debug: Uninstalling Go version %s\n", goVersion)
	}

	// Execute pre-uninstall hooks
	cmdhooks.ExecuteHooks(hooks.PreUninstall, map[string]string{
		"version": goVersion,
	})

	// Perform the actual uninstallation
	err := installer.Uninstall(goVersion)

	// Execute post-uninstall hooks (even if uninstall failed, for logging)
	cmdhooks.ExecuteHooks(hooks.PostUninstall, map[string]string{
		"version": goVersion,
	})

	return err
}
