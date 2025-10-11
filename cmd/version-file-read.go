package cmd

import (
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var versionFileReadCmd = &cobra.Command{
	Use:          "version-file-read <file>",
	Short:        "Read version from the specified file",
	Long:         "Parse and display the Go version from the specified version file or go.mod",
	Args:         cobra.ExactArgs(1),
	RunE:         runVersionFileRead,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(versionFileReadCmd)
}

func runVersionFileRead(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	filename := args[0]
	version, err := mgr.ReadVersionFile(filename)
	if err != nil {
		return err
	}

	cmd.Println(version)
	return nil
}
