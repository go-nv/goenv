package cmd

import (
	"fmt"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
	"github.com/go-nv/goenv/internal/helptext"
)

var versionFileCmd = &cobra.Command{
	Use:          "version-file [dir]",
	Short:        "Detect the file that sets the current goenv version",
	Long:         "Find and display the path to the file that determines the Go version",
	Args:         cobra.MaximumNArgs(1),
	RunE:         runVersionFile,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(versionFileCmd)
	helptext.SetCommandHelp(versionFileCmd)
}

func runVersionFile(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	var targetDir string
	if len(args) > 0 {
		targetDir = args[0]
	}

	versionFile, err := mgr.FindVersionFile(targetDir)
	if err != nil {
		return err
	}

	if versionFile == "" {
		// No version file found, return global version file path
		fmt.Fprintln(cmd.OutOrStdout(), cfg.GlobalVersionFile())
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), versionFile)
	}

	return nil
}
