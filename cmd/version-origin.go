package cmd

import (
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
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
}

func runVersionOrigin(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Check if GOENV_VERSION_ORIGIN is set (from hooks) - highest precedence
	if origin := os.Getenv("GOENV_VERSION_ORIGIN"); origin != "" {
		cmd.Println(origin)
		return nil
	}

	// Get the current version and source
	_, source, err := mgr.GetCurrentVersion()
	if err != nil {
		// No version set, return default global version file path
		cmd.Println(cfg.GlobalVersionFile())
		return nil
	}

	// Convert source to full path if needed
	switch source {
	case "GOENV_VERSION environment variable":
		cmd.Println("GOENV_VERSION environment variable")
	case "global":
		// Return the actual global version file path
		cmd.Println(cfg.GlobalVersionFile())
	default:
		// It's a file path (local .go-version or go.mod)
		// Make it absolute if not already
		if !filepath.IsAbs(source) {
			absPath, err := filepath.Abs(source)
			if err != nil {
				cmd.Println(source)
			} else {
				cmd.Println(absPath)
			}
		} else {
			cmd.Println(source)
		}
	}

	return nil
}
