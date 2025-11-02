package core

import (
	"fmt"
	"os"
	"path/filepath"

	cmdhooks "github.com/go-nv/goenv/cmd/hooks"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var uninstallCmd = &cobra.Command{
	Use:     "uninstall <version>",
	Short:   "Uninstall a Go version",
	GroupID: string(cmdpkg.GroupVersions),
	Long:    "Remove an installed Go version from the system",
	Args:    cobra.MaximumNArgs(1),
	RunE:    runUninstall,
}

var uninstallFlags struct {
	complete bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVar(&uninstallFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = uninstallCmd.Flags().MarkHidden("complete")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	cfg, mgr := cmdutil.SetupContext()

	// Handle completion mode
	if uninstallFlags.complete {
		versions, err := mgr.ListInstalledVersions()
		if err == nil {
			for _, v := range versions {
				fmt.Fprintln(cmd.OutOrStdout(), v)
			}
		}
		return nil
	}

	// Validate: uninstall requires a version argument
	if err := cmdutil.ValidateExactArgs(args, 1, "version"); err != nil {
		return fmt.Errorf("usage: goenv uninstall <version>")
	}

	installer := install.NewInstaller(cfg)

	goVersion := args[0]

	if cfg.Debug {
		fmt.Printf("Debug: Uninstalling Go version %s\n", goVersion)
	}

	// Interactive: Check if version is active and offer to switch
	shouldProceed := checkActiveVersionAndOffer(cmd, cfg, mgr, goVersion)
	if !shouldProceed {
		return fmt.Errorf("uninstall cancelled")
	}

	// Interactive: Final safety confirmation
	if !confirmUninstall(cmd, goVersion) {
		fmt.Fprintf(cmd.OutOrStdout(), "Uninstall cancelled\n")
		return nil
	}

	// Execute pre-uninstall hooks
	cmdhooks.ExecuteHooks(hooks.PreUninstall, map[string]string{
		"version": goVersion,
	})

	// Perform the actual uninstallation
	err := installer.Uninstall(goVersion)

	// Execute post-uninstall hooks (even if uninstall failed, for logging)
	cmdhooks.ExecuteHooks(hooks.PostUninstall, map[string]string{
		"version": goVersion,
	})

	if err != nil {
		return errors.FailedTo("uninstall Go", err)
	}
	return nil
}

// checkActiveVersionAndOffer checks if the version is currently active and offers to switch
func checkActiveVersionAndOffer(cmd *cobra.Command, cfg *config.Config, mgr *manager.Manager, version string) bool {
	// Create interactive context
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Skip if non-interactive
	if !ctx.IsInteractive() {
		return true // Proceed without checks
	}

	// Check if version is active in various contexts
	isActive, context := isVersionActive(cfg, version)

	if !isActive {
		return true // Not active, safe to proceed
	}

	// Build problem description
	problem := fmt.Sprintf("Go %s is currently active %s", version, context)
	repairDesc := "Switch to a different Go version before uninstalling"

	// Offer to switch before uninstalling
	if ctx.OfferRepair(problem, repairDesc) {
		// Get list of other installed versions
		allVersions, err := mgr.ListInstalledVersions()
		if err != nil || len(allVersions) <= 1 {
			ctx.ErrorPrintf("No other versions available to switch to\n")
			ctx.ErrorPrintf("Install another version first: goenv install <version>\n")
			return false
		}

		// Filter out the version being uninstalled
		otherVersions := []string{}
		for _, v := range allVersions {
			if v != version {
				otherVersions = append(otherVersions, v)
			}
		}

		if len(otherVersions) == 0 {
			ctx.ErrorPrintf("No other versions available to switch to\n")
			return false
		}

		// Offer version selection
		question := "Which version would you like to switch to?"
		selection := ctx.Select(question, otherVersions)

		if selection > 0 && selection <= len(otherVersions) {
			targetVersion := otherVersions[selection-1]

			// Determine if we should switch globally or locally
			switchGlobally := isGloballyActive(cfg, version)

			// Perform the switch
			if switchGlobally {
				if err := mgr.SetGlobalVersion(targetVersion); err != nil {
					ctx.ErrorPrintf("%sFailed to switch global version: %v\n", utils.Emoji("⚠️  "), err)
					return false
				}
				ctx.Printf("%sSwitched global version to %s\n", utils.Emoji("✓"), targetVersion)
			} else {
				if err := mgr.SetLocalVersion(targetVersion); err != nil {
					ctx.ErrorPrintf("%sFailed to switch local version: %v\n", utils.Emoji("⚠️  "), err)
					return false
				}
				ctx.Printf("%sSwitched local version to %s\n", utils.Emoji("✓"), targetVersion)
			}

			return true // Proceed with uninstall
		}

		return false // User cancelled selection
	}

	return false // User declined to switch
}

// confirmUninstall asks for final confirmation before uninstalling
func confirmUninstall(cmd *cobra.Command, version string) bool {
	// Create interactive context
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Skip confirmation if non-interactive or assume-yes
	if !ctx.IsInteractive() || ctx.AssumeYes {
		return true
	}

	// Ask for confirmation (default: no, for safety)
	question := fmt.Sprintf("Really uninstall Go %s? This cannot be undone", version)
	return ctx.Confirm(question, false)
}

// isVersionActive checks if a version is currently active
func isVersionActive(cfg *config.Config, version string) (bool, string) {
	// Check GOENV_VERSION environment variable
	if envVersion := utils.GoenvEnvVarVersion.UnsafeValue(); envVersion != "" {
		if envVersion == version {
			return true, "(via GOENV_VERSION environment variable)"
		}
	}

	// Check global version
	if isGloballyActive(cfg, version) {
		return true, "(as global default)"
	}

	// Check local version in current directory
	cwd, err := os.Getwd()
	if err == nil {
		if localVersion := readLocalVersion(cwd); localVersion == version {
			return true, "(in current directory)"
		}
	}

	return false, ""
}

// isGloballyActive checks if a version is the global default
func isGloballyActive(cfg *config.Config, version string) bool {
	globalFile := filepath.Join(cfg.Root, "version")
	if data, err := os.ReadFile(globalFile); err == nil {
		globalVersion := filepath.Base(string(data))
		return globalVersion == version
	}
	return false
}

// readLocalVersion reads the local .go-version file
func readLocalVersion(dir string) string {
	versionFile := filepath.Join(dir, config.VersionFileName)
	if data, err := os.ReadFile(versionFile); err == nil {
		return filepath.Base(string(data))
	}
	return ""
}
