package cmd

import (
	"fmt"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var localCmd = &cobra.Command{
	Use:   "local [version]",
	Short: "Set or show the local Go version for this directory",
	Long:  "Set a local Go version for the current directory by creating a .go-version file",
	RunE:  runLocal,
}

var localFlags struct {
	unset         bool
	complete      bool
	vscode        bool
	vscodeEnvVars bool
}

func init() {
	rootCmd.AddCommand(localCmd)
	localCmd.SilenceUsage = true
	localCmd.Flags().BoolVarP(&localFlags.unset, "unset", "u", false, "Unset the local Go version")
	localCmd.Flags().BoolVar(&localFlags.vscode, "vscode", false, "Also initialize VS Code workspace settings (uses absolute paths by default)")
	localCmd.Flags().BoolVar(&localFlags.vscodeEnvVars, "vscode-env-vars", false, "Use environment variables in VS Code settings (requires terminal launch)")
	localCmd.Flags().BoolVar(&localFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = localCmd.Flags().MarkHidden("complete")
	helptext.SetCommandHelp(localCmd)
}

func runLocal(cmd *cobra.Command, args []string) error {
	// Handle completion mode
	if localFlags.complete {
		cfg := config.Load()
		mgr := manager.NewManager(cfg)
		versions, err := mgr.ListInstalledVersions()
		if err == nil {
			for _, v := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}
		fmt.Fprintln(cmd.OutOrStdout(), "system")
		return nil
	}

	// Validate: local command takes 0 or 1 argument (not including --unset flag)
	if len(args) > 1 {
		return fmt.Errorf("Usage: goenv local [<version>]")
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	if localFlags.unset {
		if len(args) > 0 {
			return fmt.Errorf("Usage: goenv local [<version>]")
		}

		if err := mgr.UnsetLocalVersion(); err != nil {
			return err
		}

		return nil
	}

	if len(args) == 0 {
		version, err := mgr.GetLocalVersion()
		if err != nil {
			return fmt.Errorf("goenv: no local version configured for this directory")
		}
		fmt.Fprintln(cmd.OutOrStdout(), version)
		return nil
	}

	spec := args[0]

	resolvedVersion, err := mgr.ResolveVersionSpec(spec)
	if err != nil {
		return err
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Setting local version to %s (input: %s)\n", resolvedVersion, spec)
	}

	if err := mgr.SetLocalVersion(resolvedVersion); err != nil {
		return err
	}

	// Automatically initialize VS Code if --vscode flag is set
	if localFlags.vscode {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Initializing VS Code workspace...")

		// Use absolute paths by default (better UX), unless user wants env vars
		if localFlags.vscodeEnvVars {
			vscodeInitFlags.envVars = true
		} else {
			vscodeInitFlags.envVars = false
		}

		// Call vscode init functionality
		if err := initializeVSCodeWorkspace(cmd); err != nil {
			// Don't fail the whole command if VS Code init fails
			fmt.Fprintf(cmd.OutOrStdout(), "⚠️  Warning: VS Code initialization failed: %v\n", err)
			fmt.Fprintln(cmd.OutOrStdout(), "   You can manually run: goenv vscode init")
		}

		// Reset the flag
		vscodeInitFlags.envVars = false

		// Note about updating when version changes (only for absolute paths mode)
		if !localFlags.vscodeEnvVars {
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "💡 Tip: When you change Go versions, re-run 'goenv local VERSION --vscode' to update VS Code")
		} else {
			// Important note about environment refresh for env vars mode
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "⚠️  Important: To use Go "+resolvedVersion+" in VS Code:")
			fmt.Fprintln(cmd.OutOrStdout(), "   1. Close VS Code completely")
			fmt.Fprintln(cmd.OutOrStdout(), "   2. Reopen from terminal:  code .")
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "   This ensures VS Code inherits the updated GOROOT environment variable.")
		}
	}

	return nil
}
