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
	all      bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(uninstallCmd)
	uninstallCmd.Flags().BoolVar(&uninstallFlags.complete, "complete", false, "Internal flag for shell completions")
	uninstallCmd.Flags().BoolVar(&uninstallFlags.all, "all", false, "Uninstall all versions matching the given prefix")
	_ = uninstallCmd.Flags().MarkHidden("complete")
}

func runUninstall(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

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
	requestedVersion := args[0]

	// Get all installed versions
	installedVersions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list installed versions", err)
	}

	// Find all matching versions
	matchingVersions, err := findAllMatchingVersions(requestedVersion, installedVersions)
	if err != nil {
		return err
	}

	// Determine which versions to uninstall
	var versionsToUninstall []string

	if uninstallFlags.all {
		// --all flag: uninstall all matching versions
		versionsToUninstall = matchingVersions
	} else if len(matchingVersions) == 1 {
		// Only one match: uninstall it
		versionsToUninstall = matchingVersions
		// Show resolution feedback if version was resolved
		if matchingVersions[0] != requestedVersion {
			fmt.Fprintf(cmd.OutOrStdout(), "%sResolved %s to %s\n",
				utils.Emoji("🔍 "),
				utils.Cyan(requestedVersion),
				utils.Cyan(matchingVersions[0]))
		}
	} else {
		// Multiple matches: prompt user (unless --yes is set)
		ctx := cmdutil.NewInteractiveContext(cmd)
		if ctx.AssumeYes || !ctx.IsInteractive() {
			// Non-interactive or --yes: pick the latest (first in sorted list)
			versionsToUninstall = []string{matchingVersions[0]}
			fmt.Fprintf(cmd.OutOrStdout(), "%sResolved %s to %s (latest installed)\n",
				utils.Emoji("🔍 "),
				utils.Cyan(requestedVersion),
				utils.Cyan(matchingVersions[0]))
		} else {
			// Interactive mode: show selection prompt
			selected, err := promptVersionSelection(cmd, requestedVersion, matchingVersions)
			if err != nil {
				return err
			}
			versionsToUninstall = selected
		}
	}

	// Uninstall each version
	for _, goVersion := range versionsToUninstall {
		if cfg.Debug {
			fmt.Printf("Debug: Uninstalling Go version %s\n", goVersion)
		}

		// Interactive: Check if version is active and offer to switch
		shouldProceed := checkActiveVersionAndOffer(cmd, cfg, mgr, goVersion)
		if !shouldProceed {
			fmt.Fprintf(cmd.OutOrStdout(), "Skipping uninstall of %s\n", goVersion)
			continue
		}

		// Interactive: Final safety confirmation
		if !confirmUninstall(cmd, goVersion) {
			fmt.Fprintf(cmd.OutOrStdout(), "Skipped uninstall of %s\n", goVersion)
			continue
		}

		// Execute pre-uninstall hooks
		cmdhooks.ExecuteHooks(hooks.PreUninstall, map[string]string{
			"version": goVersion,
		})

		// Perform the actual uninstallation
		err = installer.Uninstall(goVersion)

		// Execute post-uninstall hooks (even if uninstall failed, for logging)
		cmdhooks.ExecuteHooks(hooks.PostUninstall, map[string]string{
			"version": goVersion,
		})

		if err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "Error uninstalling %s: %v\n", goVersion, err)
			// Continue with other versions if multiple
			if len(versionsToUninstall) == 1 {
				return errors.FailedTo("uninstall Go", err)
			}
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%sUninstalled Go %s\n",
				utils.Emoji("✓ "),
				utils.Cyan(goVersion))
		}
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

// promptVersionSelection prompts the user to select which versions to uninstall
// when multiple versions match the requested prefix
func promptVersionSelection(cmd *cobra.Command, requestedVersion string, matchingVersions []string) ([]string, error) {
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Build options list with descriptive labels
	options := make([]string, len(matchingVersions)+1)
	for i, version := range matchingVersions {
		if i == 0 {
			options[i] = fmt.Sprintf("%s (latest)", version)
		} else {
			options[i] = version
		}
	}
	options[len(matchingVersions)] = "All of the above"

	// Prompt for selection (ctx.Select displays the list automatically)
	question := fmt.Sprintf("Found %d installed versions matching %s. Which would you like to uninstall?",
		len(matchingVersions),
		utils.Cyan(requestedVersion))

	selection := ctx.Select(question, options)

	if selection == 0 {
		// User cancelled
		return nil, fmt.Errorf("uninstall cancelled")
	}

	if selection == len(options) {
		// User selected "All of the above"
		return matchingVersions, nil
	}

	// User selected a specific version
	return []string{matchingVersions[selection-1]}, nil
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

// resolveInstalledVersion resolves a partial version (e.g., "1.21") to a full installed version (e.g., "1.21.13")
// Similar to resolvePartialVersion in install.go but works with installed versions instead of available versions
// findAllMatchingVersions finds all installed versions matching the requested version prefix
// Returns them sorted in descending order (highest first)
func findAllMatchingVersions(requestedVersion string, installedVersions []string) ([]string, error) {
	normalized := utils.NormalizeGoVersion(requestedVersion)

	// First try exact match
	for _, installed := range installedVersions {
		if utils.MatchesVersion(installed, normalized) {
			return []string{utils.NormalizeGoVersion(installed)}, nil
		}
	}

	// If no exact match, try prefix match to find all matches
	var candidates []string
	for _, installed := range installedVersions {
		installedNormalized := utils.NormalizeGoVersion(installed)
		if installedNormalized == normalized ||
			(len(installedNormalized) > len(normalized) &&
				installedNormalized[:len(normalized)] == normalized &&
				installedNormalized[len(normalized)] == '.') {
			candidates = append(candidates, installedNormalized)
		}
	}

	if len(candidates) == 0 {
		return nil, fmt.Errorf("version %s is not installed", requestedVersion)
	}

	// Sort candidates in descending order (highest first)
	for i := 0; i < len(candidates)-1; i++ {
		for j := i + 1; j < len(candidates); j++ {
			if utils.CompareGoVersions(candidates[j], candidates[i]) > 0 {
				candidates[i], candidates[j] = candidates[j], candidates[i]
			}
		}
	}

	return candidates, nil
}

func resolveInstalledVersion(requestedVersion string, installedVersions []string) (string, error) {
	// Find all matching versions
	candidates, err := findAllMatchingVersions(requestedVersion, installedVersions)
	if err != nil {
		return "", err
	}

	// Return the highest version (first in the sorted list)
	return candidates[0], nil
}
