package version

import (
	"fmt"
	"path/filepath"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var versionOriginCmd = &cobra.Command{
	Use:          "version-origin",
	Short:        "Explain how the current Go version is set",
	Long:         "Display the file path or environment variable that sets the current Go version",
	RunE:         runVersionOrigin,
	SilenceUsage: true,
	Hidden:       true, // Legacy command - use 'goenv current --verbose' instead
}

func init() {
	cmdpkg.RootCmd.AddCommand(versionOriginCmd)
	helptext.SetCommandHelp(versionOriginCmd)
}

func runVersionOrigin(cmd *cobra.Command, args []string) error {
	// Validate: version-origin command takes no arguments
	if err := cmdutil.ValidateExactArgs(args, 0, ""); err != nil {
		return fmt.Errorf("usage: goenv version-origin")
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Get the current version
	_, source, err := mgr.GetCurrentVersion()

	// Use the source we got earlier
	if err != nil || source == "" {
		// no version set or default fallback, return default global version file path
		fmt.Fprintln(cmd.OutOrStdout(), cfg.GlobalVersionFile())
		return nil
	}

	// Convert source to full path if needed
	envVarSource := fmt.Sprintf("%s environment variable", utils.GoenvEnvVarVersion.String())
	switch source {
	case envVarSource:
		fmt.Fprintln(cmd.OutOrStdout(), envVarSource)
	case "global":
		// Return the actual global version file path
		fmt.Fprintln(cmd.OutOrStdout(), cfg.GlobalVersionFile())
	default:
		// It's a file path (local .go-version or go.mod)
		// Make it absolute if not already
		if !filepath.IsAbs(source) {
			absPath, err := filepath.Abs(source)
			if err != nil {
				fmt.Fprintln(cmd.OutOrStdout(), source)
			} else {
				fmt.Fprintln(cmd.OutOrStdout(), absPath)
			}
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), source)
		}
	}

	return nil
}
