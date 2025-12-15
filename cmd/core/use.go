package core

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/cmd/integrations"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/toolupdater"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var useCmd = &cobra.Command{
	Use:     "use [version]",
	Short:   "Install (if needed) and set a Go version",
	GroupID: string(cmdpkg.GroupVersions),
	Long: `Install (if needed) and set a Go version for the current directory or globally.

If no version is specified, uses the latest stable version (consistent with 'goenv install').

This is a convenience command that combines install, local/global, and optionally
VS Code setup into a single step.

Examples:
  goenv use                     # Use latest stable version
  goenv use 1.23.2              # Set local version (installs if needed)
  goenv use 1.23.2 --global     # Set global version (installs if needed)
  goenv use --global            # Use latest stable globally
  goenv use 1.23.2 --vscode     # Set local + configure VS Code
  goenv use latest              # Explicitly use latest stable version
  goenv use 1.23.2 --force      # Reinstall even if already installed

This command will:
  1. Check if the version is installed (or corrupted)
  2. Prompt to install/reinstall if needed (unless --yes is set)
  3. Set the version locally (or globally with --global)
  4. Optionally configure VS Code (with --vscode)
  5. Run rehash to update shims`,
	RunE: runUse,
}

var useFlags struct {
	global    bool
	vscode    bool
	vscodeEnv bool
	yes       bool
	force     bool
	quiet     bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(useCmd)
	useCmd.Flags().BoolVarP(&useFlags.global, "global", "g", false, "Set as global version instead of local")
	useCmd.Flags().BoolVar(&useFlags.vscode, "vscode", false, "Also configure VS Code workspace")
	useCmd.Flags().BoolVar(&useFlags.vscodeEnv, "vscode-env-vars", false, "Use environment variables in VS Code settings")
	useCmd.Flags().BoolVarP(&useFlags.yes, "yes", "y", false, "Auto-confirm installation prompts")
	useCmd.Flags().BoolVarP(&useFlags.force, "force", "f", false, "Force reinstall even if already installed")
	useCmd.Flags().BoolVarP(&useFlags.quiet, "quiet", "q", false, "Suppress progress output")
	helptext.SetCommandHelp(useCmd)
}

func runUse(cmd *cobra.Command, args []string) error {
	// Allow 0 or 1 arguments
	if len(args) > 1 {
		return fmt.Errorf("usage: goenv use [version]")
	}

	var versionSpec string
	if len(args) == 1 {
		versionSpec = args[0]
	} else {
		// No version specified - use latest stable (consistent with 'goenv install')
		versionSpec = "latest"
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "%sNo version specified, using latest stable\n",
				utils.Emoji("ℹ️  "))
		}
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager

	// Resolve version spec (handles "latest", "stable", etc.)
	version, err := mgr.ResolveVersionSpec(versionSpec)
	if err != nil {
		// If resolution fails, try to use as-is
		version = versionSpec
	}

	if !useFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "%sTarget version: %s\n", utils.Emoji("🎯 "), version)
	}

	// Handle installation/reinstallation using consolidated helper
	reason := fmt.Sprintf("Setting %s version to %s",
		map[bool]string{true: "global", false: "local"}[useFlags.global],
		version)

	_, err = manager.HandleVersionInstallation(manager.InstallOptions{
		Config:      cfg,
		Manager:     mgr,
		Version:     version,
		AutoInstall: useFlags.yes,
		Force:       useFlags.force,
		Quiet:       useFlags.quiet,
		Reason:      reason,
		Writer:      cmd.OutOrStdout(),
	})
	if err != nil {
		return err
	}

	// Interactive: Check for version file conflicts before setting
	if !useFlags.global {
		handleVersionConflicts(cmd, cfg, version)
	}

	// Set the version (local or global)
	if useFlags.global {
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "%sSetting global version to %s\n", utils.Emoji("🌍 "), version)
		}
		if err := mgr.SetGlobalVersion(version); err != nil {
			return errors.FailedTo("set global version", err)
		}
	} else {
		if !useFlags.quiet {
			fmt.Fprintf(cmd.OutOrStdout(), "%sSetting local version to %s\n", utils.Emoji("📁 "), version)
		}
		if err := mgr.SetLocalVersion(version); err != nil {
			return errors.FailedTo("set local version", err)
		}
	}

	// Configure VS Code if requested OR auto-detect workspace
	shouldConfigureVSCode := useFlags.vscode

	// Auto-detect VS Code workspace if flag not explicitly set
	if !useFlags.vscode && !useFlags.quiet {
		cwd, _ := os.Getwd()
		vscodeSettingsPath := filepath.Join(cwd, ".vscode", "settings.json")

		// Check if VS Code workspace exists
		if _, err := os.Stat(vscodeSettingsPath); err == nil {
			// Check VS Code setting first, then env var
			autoSync := false

			// Read the settings file to check for goenv.autoSync
			if data, err := os.ReadFile(vscodeSettingsPath); err == nil {
				// Simple JSON check - look for "goenv.autoSync": true
				settingsStr := string(data)
				if strings.Contains(settingsStr, `"goenv.autoSync"`) &&
					(strings.Contains(settingsStr, `"goenv.autoSync": true`) ||
						strings.Contains(settingsStr, `"goenv.autoSync":true`)) {
					autoSync = true
				}
			}

			// Fall back to environment variable
			if !autoSync {
				envAutoSync := os.Getenv("GOENV_VSCODE_AUTO_SYNC")
				if envAutoSync == "1" || envAutoSync == "true" {
					autoSync = true
				}
			}

			if autoSync {
				// Auto-sync enabled
				shouldConfigureVSCode = true
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sAuto-updating VS Code workspace (goenv.autoSync: true)...\n", utils.Emoji("🔧 "))
			} else {
				// Prompt user
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sDetected VS Code workspace. Update settings for Go %s? [Y/n]: ", utils.Emoji("💡 "), version)
				var response string
				fmt.Fscanln(cmd.InOrStdin(), &response)

				// Default to Yes if user just presses Enter
				if response == "" || response == "y" || response == "Y" || response == "yes" {
					shouldConfigureVSCode = true
				}
			}
		}
	}

	if shouldConfigureVSCode {
		if !useFlags.quiet && !useFlags.vscode {
			fmt.Fprintf(cmd.OutOrStdout(), "%sConfiguring VS Code...\n", utils.Emoji("🔧 "))
		}

		integrations.VSCodeInitFlags.EnvVars = useFlags.vscodeEnv
		if err := integrations.InitializeVSCodeWorkspaceWithVersion(cmd, version); err != nil {
			fmt.Fprintf(cmd.OutOrStderr(), "%sWarning: VS Code configuration failed: %v\n", utils.Emoji("⚠️  "), err)
			fmt.Fprintf(cmd.OutOrStderr(), "   You can manually run: goenv vscode init\n")
		} else {
			if !useFlags.quiet {
				fmt.Fprintf(cmd.OutOrStdout(), "%sVS Code configured\n", utils.Emoji("✅ "))
			}
		}
		integrations.VSCodeInitFlags.EnvVars = false
	}

	// Check for tool updates if auto-update is enabled
	checkToolUpdatesForUse(cmd, version)

	// Run rehash to update shims
	if !useFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%sUpdating shims...\n", utils.Emoji("🔄 "))
	}
	if err := runRehashForUse(cfg); err != nil {
		fmt.Fprintf(cmd.OutOrStderr(), "%sWarning: Failed to update shims: %v\n", utils.Emoji("⚠️  "), err)
	}

	// Success message
	if !useFlags.quiet {
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		fmt.Fprintf(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(cmd.OutOrStdout(), "%sSuccess! Now using Go %s\n", utils.Emoji("✨ "), version)
		fmt.Fprintf(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
		fmt.Fprintf(cmd.OutOrStdout(), "\n")
		fmt.Fprintf(cmd.OutOrStdout(), "Verify: go version\n")

		if useFlags.global {
			fmt.Fprintf(cmd.OutOrStdout(), "Scope:  Global (all directories)\n")
		} else {
			cwd, _ := os.Getwd()
			fmt.Fprintf(cmd.OutOrStdout(), "Scope:  Local (%s)\n", cwd)
		}

		if useFlags.vscode {
			if useFlags.vscodeEnv {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sRemember: Reopen VS Code from terminal (code .) to use env vars\n", utils.Emoji("⚠️  "))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sTip: Reload VS Code window (Cmd+Shift+P → Reload Window)\n", utils.Emoji("💡 "))
			}
		}
	}

	return nil
}

// runRehashForUse is a helper to run rehash without output
func runRehashForUse(cfg *config.Config) error {
	shimMgr := shims.NewShimManager(cfg)
	return shimMgr.Rehash()
}

// handleVersionConflicts detects and resolves conflicts between version files
func handleVersionConflicts(cmd *cobra.Command, cfg *config.Config, targetVersion string) {
	// Create interactive context
	ctx := cmdutil.NewInteractiveContext(cmd)

	// Skip if non-interactive, quiet, or not in guided mode
	if !ctx.IsInteractive() || useFlags.quiet {
		return
	}

	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	// Detect version files in current directory
	conflicts := detectVersionFileConflicts(cwd, targetVersion)

	if len(conflicts) == 0 {
		return // No conflicts
	}

	// Build problem description
	problem := buildConflictProblem(conflicts)

	// Build resolution options
	options := buildConflictOptions(conflicts, targetVersion)

	// Only offer resolution in guided mode
	if !ctx.IsGuided() {
		return
	}

	// Show the conflict and offer resolution
	ctx.Println()
	selection := ctx.Select(problem, options)

	if selection > 0 && selection <= len(options) {
		resolveConflict(ctx, cwd, conflicts, targetVersion, selection)
	}
}

// VersionFileConflict represents a detected version file conflict
type VersionFileConflict struct {
	FilePath string
	FileName string
	Version  string
}

// detectVersionFileConflicts checks for conflicting version files
func detectVersionFileConflicts(dir string, targetVersion string) []VersionFileConflict {
	var conflicts []VersionFileConflict

	// Check for .go-version
	goVersionFile := filepath.Join(dir, config.VersionFileName)
	if version := readVersionFileSimple(goVersionFile); version != "" && version != targetVersion {
		conflicts = append(conflicts, VersionFileConflict{
			FilePath: goVersionFile,
			FileName: config.VersionFileName,
			Version:  version,
		})
	}

	// Check for .tool-versions
	toolVersionsFile := filepath.Join(dir, config.ToolVersionsFileName)
	if version := readVersionFileSimple(toolVersionsFile); version != "" && version != targetVersion {
		conflicts = append(conflicts, VersionFileConflict{
			FilePath: toolVersionsFile,
			FileName: config.ToolVersionsFileName,
			Version:  version,
		})
	}

	return conflicts
}

// readVersionFileSimple reads a version from any supported version file
// using the manager API for consistent parsing
func readVersionFileSimple(path string) string {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)
	_ = cfg // unused but required by SetupContext

	version, err := mgr.ReadVersionFile(path)
	if err != nil {
		return ""
	}
	return version
}

// buildConflictProblem creates the problem description
func buildConflictProblem(conflicts []VersionFileConflict) string {
	if len(conflicts) == 1 {
		return fmt.Sprintf("Version conflict detected: %s specifies Go %s",
			conflicts[0].FileName, conflicts[0].Version)
	}

	// Multiple conflicts
	problem := "Version conflicts detected:\n"
	for _, c := range conflicts {
		problem += fmt.Sprintf("  • %s specifies Go %s\n", c.FileName, c.Version)
	}
	problem += "\nHow would you like to resolve this?"
	return problem
}

// buildConflictOptions creates resolution options
func buildConflictOptions(conflicts []VersionFileConflict, targetVersion string) []string {
	options := []string{}

	// Option 1: Update all files to match target version
	if len(conflicts) > 1 {
		options = append(options, fmt.Sprintf("Update all files to %s", targetVersion))
	} else {
		options = append(options, fmt.Sprintf("Update %s to %s", conflicts[0].FileName, targetVersion))
	}

	// Options 2-N: Update individual files
	if len(conflicts) > 1 {
		for _, c := range conflicts {
			options = append(options, fmt.Sprintf("Update only %s to %s", c.FileName, targetVersion))
		}
	}

	// Option: Remove conflicting files
	if len(conflicts) == 1 {
		options = append(options, fmt.Sprintf("Remove %s (use goenv-managed version only)", conflicts[0].FileName))
	} else {
		options = append(options, "Remove all conflicting files")
	}

	// Option: Cancel
	options = append(options, "Cancel (keep conflicts)")

	return options
}

// resolveConflict applies the selected resolution
func resolveConflict(ctx *cmdutil.InteractiveContext, dir string, conflicts []VersionFileConflict, targetVersion string, selection int) {
	if selection == len(conflicts)+2 || (len(conflicts) == 1 && selection == 3) {
		// User selected "Cancel"
		ctx.Println("Keeping version files as-is")
		return
	}

	ctx.Printf("\n%sResolving conflict...\n", utils.Emoji("🔧"))

	if selection == 1 {
		// Update all files
		for _, c := range conflicts {
			if err := updateVersionFile(c, targetVersion); err != nil {
				ctx.ErrorPrintf("%sFailed to update %s: %v\n", utils.Emoji("⚠️  "), c.FileName, err)
			} else {
				ctx.Printf("%sUpdated %s to %s\n", utils.Emoji("✓"), c.FileName, targetVersion)
			}
		}
	} else if len(conflicts) > 1 && selection > 1 && selection <= len(conflicts)+1 {
		// Update specific file
		conflictIdx := selection - 2
		c := conflicts[conflictIdx]
		if err := updateVersionFile(c, targetVersion); err != nil {
			ctx.ErrorPrintf("%sFailed to update %s: %v\n", utils.Emoji("⚠️  "), c.FileName, err)
		} else {
			ctx.Printf("%sUpdated %s to %s\n", utils.Emoji("✓"), c.FileName, targetVersion)
		}
	} else if (len(conflicts) == 1 && selection == 2) || (len(conflicts) > 1 && selection == len(conflicts)+2) {
		// Remove files
		for _, c := range conflicts {
			if err := os.Remove(c.FilePath); err != nil {
				ctx.ErrorPrintf("%sFailed to remove %s: %v\n", utils.Emoji("⚠️  "), c.FileName, err)
			} else {
				ctx.Printf("%sRemoved %s\n", utils.Emoji("✓"), c.FileName)
			}
		}
	}

	ctx.Println()
}

// updateVersionFile updates a version file with the new version
func updateVersionFile(conflict VersionFileConflict, newVersion string) error {
	if conflict.FileName == config.VersionFileName {
		// Simple version file
		return utils.WriteFileWithContext(conflict.FilePath, []byte(newVersion+"\n"), utils.PermFileDefault, "write file")
	}

	if conflict.FileName == config.ToolVersionsFileName {
		// Update the Go line in .tool-versions
		data, err := utils.ReadFileWithContext(conflict.FilePath, "read file")
		if err != nil {
			return err
		}

		lines := strings.Split(string(data), "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)
			if strings.HasPrefix(line, "golang ") || strings.HasPrefix(line, "go ") {
				parts := strings.Fields(line)
				if len(parts) >= 1 {
					lines[i] = parts[0] + " " + newVersion
				}
			}
		}

		return utils.WriteFileWithContext(conflict.FilePath, []byte(strings.Join(lines, "\n")), utils.PermFileDefault, "write file")
	}

	return fmt.Errorf("unknown file type: %s", conflict.FileName)
}

// checkToolUpdatesForUse checks for tool updates when switching Go versions
func checkToolUpdatesForUse(cmd *cobra.Command, goVersion string) {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	configPath := tools.ConfigPath(cfg.Root)

	// Load config
	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		return // Silently skip if config can't be loaded
	}

	// Skip if auto-update is not enabled for this trigger
	if !toolConfig.ShouldCheckOn("use") {
		return
	}

	// Check if enough time has passed since last check (throttling)
	if !toolConfig.ShouldCheckNow(goVersion) {
		return // Too soon since last check
	}

	// Skip if quiet mode (don't clutter output)
	if useFlags.quiet {
		return
	}

	// Create updater
	updater := toolupdater.NewUpdater(cfg)

	// Determine strategy from config
	strategy := toolupdater.StrategyAuto
	if toolConfig.UpdateStrategy != "" {
		strategy = toolupdater.UpdateStrategy(toolConfig.UpdateStrategy)
	}

	// Check for updates
	opts := toolupdater.UpdateOptions{
		Strategy:  strategy,
		GoVersion: goVersion,
		CheckOnly: !toolConfig.AutoUpdateInteractive, // Check only if not interactive
		Verbose:   false,
	}

	result, err := updater.CheckForUpdates(opts)
	if err != nil {
		return // Silently skip if check fails
	}

	// Mark that we checked (for throttling)
	toolConfig.MarkChecked(goVersion)
	_ = tools.SaveConfig(configPath, toolConfig) // Best effort save

	// Count updates available
	updatesAvailable := 0
	for _, check := range result.Checked {
		if check.UpdateAvailable {
			updatesAvailable++
		}
	}

	if updatesAvailable == 0 {
		return // Nothing to report
	}

	// Show updates
	fmt.Fprintf(cmd.OutOrStdout(), "\n%s %d tool update(s) available for Go %s\n",
		utils.Emoji("💡 "), updatesAvailable, goVersion)

	// If interactive mode, prompt to install
	if toolConfig.AutoUpdateInteractive {
		ic := cmdutil.NewInteractiveContext(cmd)
		if ic.IsInteractive() {
			// Show what would be updated
			for _, check := range result.Checked {
				if check.UpdateAvailable {
					fmt.Fprintf(cmd.OutOrStdout(), "  %s %s: %s → %s\n",
						utils.Yellow("⬆"),
						utils.BoldWhite(check.ToolName),
						utils.Gray(check.CurrentVersion),
						utils.Green(check.LatestVersion))
				}
			}

			// Prompt to update
			if ic.Confirm("\nUpdate tools now?", true) {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sUpdating tools...\n", utils.Emoji("🔄 "))

				// Run the actual updates (already computed in result if CheckOnly was false)
				if len(result.Updated) > 0 {
					for _, toolName := range result.Updated {
						fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", utils.Green("✓"), toolName)
					}
					fmt.Fprintf(cmd.OutOrStdout(), "\n%sDone! Run 'goenv rehash' to update shims\n", utils.Emoji("✅ "))
				}
			}
		}
	} else {
		// Just show hint
		fmt.Fprintf(cmd.OutOrStdout(), "   Run 'goenv tools update' to update\n")
	}
}
