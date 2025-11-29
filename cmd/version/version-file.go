package version

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/spf13/cobra"
)

var versionFileCmd = &cobra.Command{
	Use:          "version-file [dir]",
	Short:        "Detect the file that sets the current goenv version",
	Long:         "Find and display the path to the file that determines the Go version",
	Args:         cobra.MaximumNArgs(1),
	RunE:         runVersionFile,
	SilenceUsage: true,
	Hidden:       true, // Legacy command - use 'goenv current --file' instead
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionFileCmd)
	helptext.SetCommandHelp(versionFileCmd)
}

func runVersionFile(cmd *cobra.Command, args []string) error {
	cfg, mgr := cmdutil.SetupContext()

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
