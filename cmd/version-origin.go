package cmd

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var versionOriginCmd = &cobra.Command{
	Use:          "version-origin",
	Short:        "Explain how the current Go version is set",
	Long:         "Display the file path or environment variable that sets the current Go version",
	RunE:         runVersionOrigin,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(versionOriginCmd)
	helptext.SetCommandHelp(versionOriginCmd)
}

func runVersionOrigin(cmd *cobra.Command, args []string) error {
	// Validate: version-origin command takes no arguments
	if len(args) > 0 {
		return fmt.Errorf("Usage: goenv version-origin")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Check if GOENV_VERSION_ORIGIN is set (from hooks) - highest precedence
	if origin := os.Getenv("GOENV_VERSION_ORIGIN"); origin != "" {
		fmt.Fprintln(cmd.OutOrStdout(), origin)
		return nil
	}

	// Get the current version and source
	_, source, err := mgr.GetCurrentVersion()
	if err != nil || source == "" {
		// No version set or default fallback, return default global version file path
		fmt.Fprintln(cmd.OutOrStdout(), cfg.GlobalVersionFile())
		return nil
	}

	// Convert source to full path if needed
	switch source {
	case "GOENV_VERSION environment variable":
		fmt.Fprintln(cmd.OutOrStdout(), "GOENV_VERSION environment variable")
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
