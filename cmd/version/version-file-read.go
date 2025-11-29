package version

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/spf13/cobra"
)

var versionFileReadCmd = &cobra.Command{
	Use:          "version-file-read <file>",
	Short:        "Read version from the specified file",
	Long:         "Parse and display the Go version from the specified version file or go.mod",
	Args:         cobra.ExactArgs(1),
	RunE:         runVersionFileRead,
	SilenceUsage: true,
	Hidden:       true, // Internal command - not for typical user interaction
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionFileReadCmd)
	helptext.SetCommandHelp(versionFileReadCmd)
}

func runVersionFileRead(cmd *cobra.Command, args []string) error {
	_, mgr := cmdutil.SetupContext()

	filename := args[0]
	version, err := mgr.ReadVersionFile(filename)
	if err != nil {
		return errors.FailedTo("read version file", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), version)
	return nil
}
