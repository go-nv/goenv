package cmd

import (
	"fmt"
	"os"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <version>",
	Short: "Uninstall a Go version",
	Long:  "Remove an installed Go version from the system",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runUninstall,
}

var uninstallFlags struct {
	complete bool
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
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

	// Execute uninstall hooks before uninstallation
	hookEnv := []string{
		"GOENV_VERSION=" + goVersion,
	}
	if err := executeHooks("uninstall", hookEnv); err != nil && cfg.Debug {
		fmt.Fprintf(os.Stderr, "goenv: uninstall hooks failed: %v\n", err)
	}

	return installer.Uninstall(goVersion)
}
