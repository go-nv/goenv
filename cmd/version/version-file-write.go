package version

import (
	"fmt"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var versionFileWriteCmd = &cobra.Command{
	Use:          "version-file-write <file> <version>...",
	Short:        "Write version(s) to the specified file",
	Long:         "Write one or more Go versions to the specified version file",
	Args:         cobra.MinimumNArgs(2),
	RunE:         runVersionFileWrite,
	SilenceUsage: true,
	Hidden:       true, // Internal command - not for typical user interaction
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionFileWriteCmd)
	helptext.SetCommandHelp(versionFileWriteCmd)
}

func runVersionFileWrite(cmd *cobra.Command, args []string) error {
	_, mgr := cmdutil.SetupContext()

	filename := args[0]
	versions := args[1:]

	// Check if we're setting to "system"
	if len(versions) == 1 && versions[0] == manager.SystemVersion {
		// Check if system Go exists
		if !mgr.HasSystemGo() {
			return errors.SystemVersionNotFound()
		}

		// Read old version before removing (for message)
		oldVersion, _ := mgr.ReadVersionFile(filename)

		// Remove the version file
		if err := mgr.UnsetVersionFile(filename); err != nil {
			return err
		}

		// Print success message with context about what was replaced
		if oldVersion != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "goenv: using system version instead of %s now\n", oldVersion)
		}
		return nil
	}

	// Validate that all versions are installed
	for _, version := range versions {
		if !mgr.IsVersionInstalled(version) {
			return fmt.Errorf("goenv: version '%s' not installed", version)
		}
	}

	// Write the version(s) to file
	versionStr := strings.Join(versions, "\n")
	if err := mgr.WriteVersionFile(filename, versionStr); err != nil {
		return err
	}

	return nil
}
