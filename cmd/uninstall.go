package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/install"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:   "uninstall <version>",
	Short: "Uninstall a Go version",
	Long:  "Remove an installed Go version from the system",
	Args:  cobra.ExactArgs(1),
	RunE:  runUninstall,
}

func init() {
	rootCmd.AddCommand(uninstallCmd)
}

func runUninstall(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	installer := install.NewInstaller(cfg)

	goVersion := args[0]

	if cfg.Debug {
		fmt.Printf("Debug: Uninstalling Go version %s\n", goVersion)
	}

	return installer.Uninstall(goVersion)
}
