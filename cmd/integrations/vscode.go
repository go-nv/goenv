package integrations

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/vscode"
	"github.com/spf13/cobra"
)

var vscodeCmd = &cobra.Command{
	Use:     "vscode",
	Short:   "Manage VS Code integration",
	GroupID: string(cmdpkg.GroupIntegrations),
	Long: `Commands to configure and manage Visual Studio Code integration with goenv.

Quick Start:
  goenv vscode setup    Complete setup (recommended for first-time users)
  goenv vscode status   Check current integration status
  goenv vscode doctor   Diagnose and fix issues

The 'setup' command combines initialization, syncing, and validation in one step.`,
}

var vscodeInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize VS Code workspace for goenv",
	Long: `Initialize VS Code workspace with goenv configuration.

Note: For first-time setup, use 'goenv vscode setup' instead (combines init + sync + doctor).

This command creates or updates .vscode/settings.json and .vscode/extensions.json
in the current directory to ensure VS Code uses the correct Go version from goenv.

The settings configure:
  - go.goroot to use GOROOT environment variable
  - go.gopath to use GOPATH environment variable
  - go.toolsGopath for Go tools installation
  - Recommended Go extension

This makes VS Code automatically detect and use goenv-managed Go versions.`,
	Example: `  # Initialize VS Code in current directory
  goenv vscode init

  # Force overwrite existing configuration
  goenv vscode init --force

  # Use advanced template with gopls settings
  goenv vscode init --template advanced

  # Preview changes without writing
  goenv vscode init --dry-run`,
	RunE: runVSCodeInit,
}

var vscodeSyncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Sync VS Code settings with current Go version",
	Long: `Sync VS Code settings with the current Go version.

Note: For complete setup and validation, use 'goenv vscode setup' instead.

This command updates .vscode/settings.json to point to the currently active
Go version from goenv. Use this after switching Go versions to keep VS Code
in sync without re-running full initialization.

The sync command:
  - Preserves your existing VS Code settings
  - Updates only Go-related paths (go.goroot, go.gopath)
  - Creates a backup before making changes
  - Works with both absolute and environment variable modes`,
	Example: `  # Sync after changing versions
  goenv local 1.24.4 && goenv vscode sync

  # Check what would change (dry run)
  goenv vscode sync --dry-run`,
	RunE: runVSCodeSync,
}

var vscodeStatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show VS Code integration status",
	Long: `Show the current VS Code integration status.

This command checks the .vscode/settings.json configuration and reports:
  - Whether VS Code settings exist
  - Current configuration mode (absolute paths or env vars)
  - Configured Go version vs expected version
  - Whether a re-sync is needed

Use --json for machine-readable output in CI pipelines.`,
	Example: `  # Show human-readable status
  goenv vscode status

  # Get JSON output for scripts
  goenv vscode status --json`,
	RunE: runVSCodeStatus,
}

var vscodeRevertCmd = &cobra.Command{
	Use:   "revert",
	Short: "Revert to previous VS Code settings",
	Long: `Revert VS Code settings to the last backup.

This command restores settings.json from the most recent backup created by
goenv. Backups are automatically created before any settings modification.`,
	Example: `  # Restore last backup
  goenv vscode revert`,
	RunE: runVSCodeRevert,
}

var vscodeDoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check VS Code integration health",
	Long: `Run diagnostics on VS Code integration and Go tooling.

This command checks:
  - VS Code settings configuration
  - Go toolchain availability
  - gopls installation and version
  - go.toolsGopath writability
  - Workspace structure (go.work, go.mod)
  - Common configuration issues

Provides actionable advice for fixing any issues found.`,
	Example: `  # Run health checks
  goenv vscode doctor

  # Get JSON output for automation
  goenv vscode doctor --json`,
	RunE: runVSCodeDoctor,
}

var vscodeSetupCmd = &cobra.Command{
	Use:   "setup",
	Short: "One-command VS Code setup (init + sync + doctor)",
	Long: `Unified command for new users to set up VS Code integration.

This command combines three operations:
  1. Initialize .vscode/settings.json (vscode init)
  2. Sync with current Go version (vscode sync)
  3. Validate configuration (vscode doctor)

Perfect for getting started quickly without running multiple commands.`,
	Example: `  # Complete VS Code setup in one command
  goenv vscode setup

  # Setup with advanced template
  goenv vscode setup --template advanced

  # Preview what would be done
  goenv vscode setup --dry-run

  # Setup and exit with error if validation fails
  goenv vscode setup --strict`,
	RunE: runVSCodeSetup,
}

var VSCodeInitFlags struct {
	Force          bool
	Template       string
	EnvVars        bool
	DryRun         bool
	Diff           bool
	WorkspacePaths bool
	VersionedTools bool
	Tasks          bool
	Launch         bool
	TerminalEnv    bool
	Devcontainer   bool
}

var vscodeSyncFlags struct {
	dryRun bool
	diff   bool
}

var vscodeStatusFlags struct {
	json bool
}

var vscodeDoctorFlags struct {
	json bool
}

var vscodeSetupFlags struct {
	template string
	dryRun   bool
	strict   bool
	json     bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(vscodeCmd)
	vscodeCmd.AddCommand(vscodeSetupCmd)
	vscodeCmd.AddCommand(vscodeInitCmd)
	vscodeCmd.AddCommand(vscodeSyncCmd)
	vscodeCmd.AddCommand(vscodeStatusCmd)
	vscodeCmd.AddCommand(vscodeRevertCmd)
	vscodeCmd.AddCommand(vscodeDoctorCmd)

	// Init flags
	vscodeInitCmd.Flags().BoolVarP(&VSCodeInitFlags.Force, "force", "f", false, "Overwrite existing settings")
	vscodeInitCmd.Flags().StringVarP(&VSCodeInitFlags.Template, "template", "t", "basic", "Configuration template (basic, advanced, monorepo)")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.EnvVars, "env-vars", false, "Use environment variables instead of absolute paths (requires launching VS Code from terminal)")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.DryRun, "dry-run", false, "Preview changes without writing files")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.Diff, "diff", false, "Show diff of changes (implies --dry-run)")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.WorkspacePaths, "workspace-paths", false, "Use ${workspaceFolder}-relative paths for portability")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.VersionedTools, "versioned-tools", false, "Use per-version tools directory")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.Tasks, "tasks", false, "Generate tasks.json for build and test")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.Launch, "launch", false, "Generate launch.json for debugging")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.TerminalEnv, "terminal-env", false, "Configure integrated terminal environment")
	vscodeInitCmd.Flags().BoolVar(&VSCodeInitFlags.Devcontainer, "devcontainer", false, "Generate .devcontainer configuration")

	// Sync flags
	vscodeSyncCmd.Flags().BoolVar(&vscodeSyncFlags.dryRun, "dry-run", false, "Preview changes without writing files")
	vscodeSyncCmd.Flags().BoolVar(&vscodeSyncFlags.diff, "diff", false, "Show diff of changes (implies --dry-run)")

	// Status flags
	vscodeStatusCmd.Flags().BoolVar(&vscodeStatusFlags.json, "json", false, "Output status as JSON")

	// Doctor flags
	vscodeDoctorCmd.Flags().BoolVar(&vscodeDoctorFlags.json, "json", false, "Output diagnostics as JSON")

	// Setup flags
	vscodeSetupCmd.Flags().StringVarP(&vscodeSetupFlags.template, "template", "t", "basic", "Template to use (basic, advanced, monorepo)")
	vscodeSetupCmd.Flags().BoolVar(&vscodeSetupFlags.dryRun, "dry-run", false, "Preview what would be done")
	vscodeSetupCmd.Flags().BoolVar(&vscodeSetupFlags.strict, "strict", false, "Exit with error if doctor validation fails")
	vscodeSetupCmd.Flags().BoolVar(&vscodeSetupFlags.json, "json", false, "Output doctor results in JSON")

	vscodeSetupCmd.SilenceUsage = true
	vscodeInitCmd.SilenceUsage = true
	vscodeSyncCmd.SilenceUsage = true
	vscodeStatusCmd.SilenceUsage = true
	vscodeRevertCmd.SilenceUsage = true
	vscodeDoctorCmd.SilenceUsage = true

	helptext.SetCommandHelp(vscodeSetupCmd)
	helptext.SetCommandHelp(vscodeInitCmd)
	helptext.SetCommandHelp(vscodeSyncCmd)
	helptext.SetCommandHelp(vscodeStatusCmd)
	helptext.SetCommandHelp(vscodeRevertCmd)
	helptext.SetCommandHelp(vscodeDoctorCmd)
}

// VSCodeSettings represents the VS Code settings.json structure
type VSCodeSettings map[string]interface{}

func runVSCodeInit(cmd *cobra.Command, args []string) error {
	return initializeVSCodeWorkspace(cmd)
}

// initializeVSCodeWorkspace performs the actual VS Code initialization
// This is exported so it can be called from other commands (e.g., goenv local --vscode)
func initializeVSCodeWorkspace(cmd *cobra.Command) error {
	return InitializeVSCodeWorkspaceWithVersion(cmd, "")
}

// initializeVSCodeWorkspaceWithVersion initializes VS Code settings with a specific version
// If version is empty, it uses the current active version
func InitializeVSCodeWorkspaceWithVersion(cmd *cobra.Command, version string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return errors.FailedTo("get current directory", err)
	}

	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")
	extensionsFile := filepath.Join(vscodeDir, "extensions.json")

	// Create .vscode directory if it doesn't exist
	if err := utils.EnsureDirWithContext(vscodeDir, "create .vscode directory"); err != nil {
		return err
	}

	// Use basic template for automatic initialization
	template := "basic"
	force := VSCodeInitFlags.Force

	// Auto-detect monorepo (go.work file) if template not explicitly set
	if VSCodeInitFlags.Template == "" || VSCodeInitFlags.Template == "basic" {
		goWorkPath := filepath.Join(cwd, "go.work")
		if utils.FileExists(goWorkPath) {
			template = "monorepo"
			fmt.Fprintf(cmd.OutOrStdout(), "%sDetected go.work file - using monorepo template\n", utils.Emoji("ℹ️  "))
		}
	} else {
		template = VSCodeInitFlags.Template
	}

	// Default to absolute paths for better UX (works when opened from GUI)
	// unless user explicitly requests env vars
	useAbsolutePaths := !VSCodeInitFlags.EnvVars // If called from vscode init command, use flags

	// Generate settings based on template
	settings, err := generateSettings(template)
	if err != nil {
		return err
	}

	// Convert to explicit paths if requested (using platform-specific env vars for portability)
	if useAbsolutePaths {
		cfg, mgr := cmdutil.SetupContext()

		// Get Go version - use provided version or current active version
		if version == "" {
			version, _, err = mgr.GetCurrentVersion()
			if err != nil {
				return errors.FailedTo("get current Go version", err)
			}
		}

		// Get home directory (cross-platform)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return errors.FailedTo("get home directory", err)
		}

		// Use platform-specific environment variables for portability
		// Windows: ${env:USERPROFILE}  Unix/macOS: ${env:HOME}
		var homeEnvVar string
		if utils.IsWindows() {
			homeEnvVar = "${env:USERPROFILE}"
		} else {
			homeEnvVar = "${env:HOME}"
		}

		// Build paths using env var prefix for portability across users
		gorootAbs := filepath.Join(cfg.Root, "versions", version)
		goroot := strings.Replace(gorootAbs, homeDir, homeEnvVar, 1)

		// Build GOPATH respecting GOENV_GOPATH_PREFIX (same as exec.go and sh-rehash.go)
		gopathPrefix := utils.GoenvEnvVarGopathPrefix.UnsafeValue()
		var gopathAbs string
		if gopathPrefix == "" {
			gopathAbs = filepath.Join(homeDir, "go", version)
		} else {
			gopathAbs = filepath.Join(gopathPrefix, version)
		}
		gopath := strings.Replace(gopathAbs, homeDir, homeEnvVar, 1)

		settings["go.goroot"] = goroot
		settings["go.gopath"] = gopath

		// toolsGopath: use platform-specific env var
		if _, ok := settings["go.toolsGopath"]; ok {
			settings["go.toolsGopath"] = homeEnvVar + "/go/tools"
		}
	}

	// Handle workspace-relative paths if requested
	if VSCodeInitFlags.WorkspacePaths {
		settings = vscode.ConvertToWorkspacePaths(settings, cwd)
	}

	// Schema validation: ensure we only touch Go-related keys
	warnings, err := vscode.ValidateSettingsKeys(settings)
	if err != nil {
		return err
	}

	// Display any warnings about deprecated keys
	for _, warning := range warnings {
		fmt.Fprintln(cmd.OutOrStderr(), warning)
	}

	// Handle existing settings
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	if err != nil && !os.IsNotExist(err) {
		return errors.FailedTo("read existing settings", err)
	}

	// Prepare keys to update
	keysToUpdate := make(map[string]interface{})
	if existingSettings != nil && !force {
		// Only add keys that actually have values
		if val, ok := settings["go.goroot"]; ok && val != nil {
			keysToUpdate["go.goroot"] = val
		}
		if val, ok := settings["go.gopath"]; ok && val != nil {
			keysToUpdate["go.gopath"] = val
		}
		if val, ok := settings["go.toolsGopath"]; ok && val != nil {
			keysToUpdate["go.toolsGopath"] = val
		}

		// Validate keys to update
		warnings, err := vscode.ValidateSettingsKeys(keysToUpdate)
		if err != nil {
			return err
		}
		for _, warning := range warnings {
			fmt.Fprintln(cmd.OutOrStderr(), warning)
		}
	}

	// Dry-run mode - preview changes
	isDryRun := VSCodeInitFlags.DryRun || VSCodeInitFlags.Diff
	if isDryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%sDry-run mode: Preview of changes\n", utils.Emoji("🔍 "))
		fmt.Fprintln(cmd.OutOrStdout())

		if existingSettings != nil && !force {
			// Show what keys would be updated
			fmt.Fprintln(cmd.OutOrStdout(), "Would update existing settings:")
			hasChanges := false
			for key, newVal := range keysToUpdate {
				oldVal := existingSettings[key]
				if oldVal != newVal {
					hasChanges = true
					fmt.Fprintf(cmd.OutOrStdout(), "  %s:\n", key)
					fmt.Fprintf(cmd.OutOrStdout(), "    - %v\n", oldVal)
					fmt.Fprintf(cmd.OutOrStdout(), "    + %v\n", newVal)
				}
			}
			if !hasChanges {
				fmt.Fprintln(cmd.OutOrStdout(), "  (No changes needed - settings are already correct)")
			}
		} else {
			// Show what would be created
			fmt.Fprintln(cmd.OutOrStdout(), "Would create new settings file:")
			for key, val := range settings {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s: %v\n", key, val)
			}
		}

		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sRun without --dry-run to apply these changes\n", utils.Emoji("💡 "))
		return nil
	}

	// Write settings.json
	if existingSettings != nil && !force {
		// File exists - check if any values actually differ before updating
		hasChanges := false
		for key, newVal := range keysToUpdate {
			oldVal := existingSettings[key]
			if oldVal != newVal {
				hasChanges = true
				break
			}
		}

		if !hasChanges {
			// No changes needed - skip backup and update
			fmt.Fprintf(cmd.OutOrStdout(), "✓ Settings are already up-to-date, no changes needed\n")
		} else {
			// File exists - update only the specific Go keys we care about
			if useAbsolutePaths {
				fmt.Fprintf(cmd.OutOrStdout(), "%sUpdating Go paths with absolute values\n", utils.Emoji("ℹ️  "))
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "%sUpdating Go paths with environment variables\n", utils.Emoji("ℹ️  "))
			}

			// Backup before modifying
			if err := vscode.BackupFile(settingsFile); err != nil {
				return errors.FailedTo("create backup", err)
			}

			if err := vscode.UpdateJSONKeys(settingsFile, keysToUpdate); err != nil {
				return errors.FailedTo("update settings.json", err)
			}

			fmt.Fprintf(cmd.OutOrStdout(), "✓ Updated %s\n", settingsFile)
		}
	} else {
		// No existing file or force mode - create new file
		if err := vscode.WriteJSONFile(settingsFile, settings); err != nil {
			return errors.FailedTo("write settings.json", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Created %s\n", settingsFile)
	}

	// Generate and write extensions.json
	// Read existing extensions if file exists
	var recommendations []string
	existingExtensions, err := vscode.ReadExistingExtensions(extensionsFile)
	if err == nil {
		// File exists - merge with existing recommendations
		recommendations = existingExtensions.Recommendations
	}

	// Add golang.go if not already present
	goExtension := "golang.go"
	hasGoExtension := false
	for _, rec := range recommendations {
		if rec == goExtension {
			hasGoExtension = true
			break
		}
	}
	if !hasGoExtension {
		recommendations = append(recommendations, goExtension)
	}

	extensions := vscode.Extensions{
		Recommendations: recommendations,
	}

	if err := vscode.WriteJSONFile(extensionsFile, extensions); err != nil {
		return errors.FailedTo("write extensions.json", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Created/updated %s\n", extensionsFile)

	// Print configuration summary
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sVS Code workspace configured for goenv!\n", utils.Emoji("✨ "))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Configuration:")

	if useAbsolutePaths {
		fmt.Fprintf(cmd.OutOrStdout(), "  • go.goroot: %v (absolute path)\n", settings["go.goroot"])
		fmt.Fprintf(cmd.OutOrStdout(), "  • go.gopath: %v (absolute path)\n", settings["go.gopath"])
		fmt.Fprintf(cmd.OutOrStdout(), "  • Mode: Absolute paths (works when opened from GUI!)\n")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  • go.goroot: ${env:GOROOT}\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  • go.gopath: ${env:GOPATH}\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  • Mode: Environment variables (requires terminal launch)\n")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  • Template: %s\n", template)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Next steps:")
	if useAbsolutePaths {
		fmt.Fprintln(cmd.OutOrStdout(), "  1. Reload VS Code window (Cmd+Shift+P → 'Developer: Reload Window')")
		fmt.Fprintf(cmd.OutOrStdout(), "  2. %sReady to use - works even when opened from GUI!\n", utils.Emoji("✨ "))
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sNote: If you change Go versions, re-run 'goenv vscode init' to update paths\n", utils.Emoji("💡 "))
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  1. Close VS Code completely")
		fmt.Fprintln(cmd.OutOrStdout(), "  2. Ensure 'eval \"$(goenv init -)\"' is in your shell config")
		fmt.Fprintln(cmd.OutOrStdout(), "  3. Launch from terminal: code .")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sWarning: Environment variable mode requires terminal launch\n", utils.Emoji("⚠️  "))
		fmt.Fprintln(cmd.OutOrStdout(), "   Consider using default --absolute mode for better UX")
	}
	fmt.Fprintln(cmd.OutOrStdout(), "")

	return nil
}

// generateSettings creates settings based on template
func generateSettings(template string) (VSCodeSettings, error) {
	// Determine platform-specific home environment variable
	homeEnvVar := "${env:HOME}"
	if utils.IsWindows() {
		homeEnvVar = "${env:USERPROFILE}"
	}

	switch template {
	case "basic":
		return VSCodeSettings{
			"go.goroot":      "${env:GOROOT}",
			"go.gopath":      "${env:GOPATH}",
			"go.toolsGopath": homeEnvVar + "/go/tools",
		}, nil

	case "advanced":
		return VSCodeSettings{
			"go.goroot":                         "${env:GOROOT}",
			"go.gopath":                         "${env:GOPATH}",
			"go.toolsGopath":                    homeEnvVar + "/go/tools",
			"go.toolsManagement.autoUpdate":     true,
			"go.formatTool":                     "gofumpt",
			"go.lintTool":                       "golangci-lint",
			"go.lintOnSave":                     "package",
			"go.testOnSave":                     false,
			"go.coverOnSave":                    false,
			"go.coverOnSingleTest":              true,
			"go.autocompleteUnimportedPackages": true,
			"go.gotoSymbol.includeImports":      true,
			"go.testExplorer.enable":            true,
			"gopls": map[string]interface{}{
				"formatting.gofumpt":            true,
				"ui.completion.usePlaceholders": true,
				"ui.diagnostic.annotations": map[string]interface{}{
					"bounds": true,
					"escape": true,
					"inline": false,
					"null":   true,
				},
			},
		}, nil

	case "monorepo":
		return VSCodeSettings{
			"go.goroot":                          "${env:GOROOT}",
			"go.gopath":                          "${env:GOPATH}",
			"go.toolsGopath":                     homeEnvVar + "/go/tools",
			"go.inferGopath":                     false,
			"go.formatTool":                      "gofumpt",
			"go.testExplorer.enable":             true,
			"go.testExplorer.packageDisplayMode": "flat",
			"go.coverOnSingleTest":               true,
			"gopls": map[string]interface{}{
				"formatting.gofumpt": true,
				"build.directoryFilters": []string{
					"-vendor",
					"-node_modules",
					"-third_party",
				},
				"ui.completion.usePlaceholders": true,
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown template: %s (choose: %s, %s, %s)", template, "basic", "advanced", "monorepo")
	}
} // mergeSettings merges new settings into existing, preserving existing keys
func mergeSettings(existing, new VSCodeSettings) VSCodeSettings {
	result := make(VSCodeSettings)

	// Copy all existing settings
	for k, v := range existing {
		result[k] = v
	}

	// Add new settings only if key doesn't exist
	for k, v := range new {
		if _, exists := result[k]; !exists {
			result[k] = v
		}
	}

	return result
}

// mergeSettingsWithOverride merges settings but forces override of specified keys
func mergeSettingsWithOverride(existing, new VSCodeSettings, overrideKeys []string) VSCodeSettings {
	result := make(VSCodeSettings)

	// Copy all existing settings
	for k, v := range existing {
		result[k] = v
	}

	// Create map for quick override key lookup
	shouldOverride := make(map[string]bool)
	for _, key := range overrideKeys {
		shouldOverride[key] = true
	}

	// Add or override settings based on override keys
	for k, v := range new {
		if shouldOverride[k] {
			// Force override this key
			result[k] = v
		} else if _, exists := result[k]; !exists {
			// Only add if doesn't exist
			result[k] = v
		}
	}

	return result
}

// runVSCodeSync syncs VS Code settings with current Go version
func runVSCodeSync(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.FailedTo("get current directory", err)
	}

	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")

	// Check if settings file exists
	if !utils.FileExists(settingsFile) {
		return fmt.Errorf("no VS Code settings found. Run 'goenv vscode init' first")
	}

	// Get current Go version
	cfg, mgr := cmdutil.SetupContext()
	version, _, err := mgr.GetCurrentVersion()
	if err != nil {
		return errors.FailedTo("get current Go version", err)
	}

	// Read existing settings
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	if err != nil {
		return errors.FailedTo("read existing settings", err)
	}

	// Determine mode (env vars vs absolute paths)
	useEnvVars := false
	if goroot, ok := existingSettings["go.goroot"].(string); ok {
		if strings.Contains(goroot, "${env:GOROOT}") {
			useEnvVars = true
		}
	}

	if useEnvVars {
		fmt.Fprintf(cmd.OutOrStdout(), "%sSettings use environment variables - no sync needed\n", utils.Emoji("ℹ️  "))
		fmt.Fprintln(cmd.OutOrStdout(), "   (env vars automatically track the current version)")
		return nil
	}

	// Build new paths for current version
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return errors.FailedTo("get home directory", err)
	}

	var homeEnvVar string
	if utils.IsWindows() {
		homeEnvVar = "${env:USERPROFILE}"
	} else {
		homeEnvVar = "${env:HOME}"
	}

	gorootAbs := filepath.Join(cfg.Root, "versions", version)
	goroot := strings.Replace(gorootAbs, homeDir, homeEnvVar, 1)

	gopathPrefix := utils.GoenvEnvVarGopathPrefix.UnsafeValue()
	var gopathAbs string
	if gopathPrefix == "" {
		gopathAbs = filepath.Join(homeDir, "go", version)
	} else {
		gopathAbs = filepath.Join(gopathPrefix, version)
	}
	gopath := strings.Replace(gopathAbs, homeDir, homeEnvVar, 1)

	keysToUpdate := map[string]interface{}{
		"go.goroot": goroot,
		"go.gopath": gopath,
	}

	// Schema validation: ensure we only touch Go-related keys
	warnings, err := vscode.ValidateSettingsKeys(keysToUpdate)
	if err != nil {
		return err
	}

	// Display any warnings about deprecated keys
	for _, warning := range warnings {
		fmt.Fprintln(cmd.OutOrStderr(), warning)
	}

	// Dry run mode
	if vscodeSyncFlags.dryRun || vscodeSyncFlags.diff {
		fmt.Fprintf(cmd.OutOrStdout(), "%sPreview of changes:\n", utils.Emoji("🔍 "))
		fmt.Fprintln(cmd.OutOrStdout())
		hasChanges := false
		for key, newVal := range keysToUpdate {
			oldVal := existingSettings[key]
			if oldVal != newVal {
				hasChanges = true
				fmt.Fprintf(cmd.OutOrStdout(), "  %s:\n", key)
				fmt.Fprintf(cmd.OutOrStdout(), "    - %v\n", oldVal)
				fmt.Fprintf(cmd.OutOrStdout(), "    + %v\n", newVal)
			}
		}
		if !hasChanges {
			fmt.Fprintln(cmd.OutOrStdout(), "  (No changes needed - settings are already correct)")
		}
		if vscodeSyncFlags.dryRun || vscodeSyncFlags.diff {
			return nil
		}
	}

	// Check if any values actually differ before updating
	hasChanges := false
	for key, newVal := range keysToUpdate {
		oldVal := existingSettings[key]
		if oldVal != newVal {
			hasChanges = true
			break
		}
	}

	if !hasChanges {
		// No changes needed - skip backup and update
		fmt.Fprintf(cmd.OutOrStdout(), "✓ Settings are already synced to Go %s\n", version)
		return nil
	}

	// Backup before modifying
	if err := vscode.BackupFile(settingsFile); err != nil {
		return errors.FailedTo("create backup", err)
	}

	// Update settings
	if err := vscode.UpdateJSONKeys(settingsFile, keysToUpdate); err != nil {
		return errors.FailedTo("update settings", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Synced VS Code settings to Go %s\n", version)
	fmt.Fprintln(cmd.OutOrStdout(), "  Backup saved to settings.json.bak")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sReload VS Code window to apply changes\n", utils.Emoji("💡 "))

	return nil
}

// runVSCodeStatus shows VS Code integration status
func runVSCodeStatus(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.FailedTo("get current directory", err)
	}

	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")

	// Get current Go version
	_, mgr := cmdutil.SetupContext()
	version, source, err := mgr.GetCurrentVersion()
	if err != nil {
		version = "none"
		source = "none"
	}

	// Check settings
	result := vscode.CheckSettings(settingsFile, version)

	// JSON output
	if vscodeStatusFlags.json {
		status := map[string]interface{}{
			"hasSettings":       result.HasSettings,
			"usesEnvVars":       result.UsesEnvVars,
			"configuredVersion": result.ConfiguredVersion,
			"expectedVersion":   result.ExpectedVersion,
			"mismatch":          result.Mismatch,
			"settingsPath":      result.SettingsPath,
			"versionSource":     source,
		}
		if err := cmdutil.OutputJSON(cmd.OutOrStdout(), status); err != nil {
			return errors.FailedTo("output JSON", err)
		}
		if result.Mismatch {
			return fmt.Errorf("version mismatch detected")
		}
		return nil
	}

	// Human-readable output
	fmt.Fprintln(cmd.OutOrStdout(), "VS Code Integration Status")
	fmt.Fprintln(cmd.OutOrStdout(), "==========================")
	fmt.Fprintln(cmd.OutOrStdout())

	if !result.HasSettings {
		fmt.Fprintf(cmd.OutOrStdout(), "%sNo VS Code settings found\n", utils.Emoji("❌ "))
		fmt.Fprintln(cmd.OutOrStdout(), "   Run 'goenv vscode init' to configure VS Code")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sVS Code settings found\n", utils.Emoji("✓ "))
	fmt.Fprintln(cmd.OutOrStdout())

	if result.UsesEnvVars {
		fmt.Fprintln(cmd.OutOrStdout(), "Mode: Environment Variables")
		fmt.Fprintln(cmd.OutOrStdout(), "  Settings use ${env:GOROOT} and ${env:GOPATH}")
		fmt.Fprintf(cmd.OutOrStdout(), "  %sAutomatically tracks version changes\n", utils.Emoji("✓ "))
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "Mode: Absolute Paths")
		fmt.Fprintf(cmd.OutOrStdout(), "  Configured version: %s\n", result.ConfiguredVersion)
		fmt.Fprintf(cmd.OutOrStdout(), "  Expected version: %s (from %s)\n", result.ExpectedVersion, source)
		fmt.Fprintln(cmd.OutOrStdout())

		if result.Mismatch {
			fmt.Fprintf(cmd.OutOrStdout(), "%sVersion mismatch detected!\n", utils.Emoji("⚠️  "))
			fmt.Fprintln(cmd.OutOrStdout(), "   Run 'goenv vscode sync' to update settings")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%sSettings are in sync\n", utils.Emoji("✓ "))
		}
	}

	return nil
}

// runVSCodeRevert reverts VS Code settings to last backup
func runVSCodeRevert(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.FailedTo("get current directory", err)
	}

	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")

	// Check if backup exists
	backupFile := settingsFile + ".bak"
	if !utils.FileExists(backupFile) {
		return fmt.Errorf("no backup found at %s", backupFile)
	}

	// Restore from backup
	if err := vscode.RestoreFromBackup(settingsFile); err != nil {
		return err
	}

	fmt.Fprintf(cmd.OutOrStdout(), "✓ Restored settings from backup\n")
	fmt.Fprintf(cmd.OutOrStdout(), "  %s → %s\n", backupFile, settingsFile)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sReload VS Code window to apply changes\n", utils.Emoji("💡 "))

	return nil
}

// Diagnostic check result
type diagnosticCheck struct {
	Name    string `json:"name"`
	Status  string `json:"status"` // "pass", "warn", "fail"
	Message string `json:"message"`
	Advice  string `json:"advice,omitempty"`
	Details string `json:"details,omitempty"`
}

// runVSCodeDoctor runs health checks on VS Code integration
func runVSCodeDoctor(cmd *cobra.Command, args []string) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.FailedTo("get current directory", err)
	}

	_, mgr := cmdutil.SetupContext()

	var checks []diagnosticCheck

	// Check 1: VS Code settings exist
	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")

	if !utils.FileExists(settingsFile) {
		checks = append(checks, diagnosticCheck{
			Name:    "VS Code Settings",
			Status:  "fail",
			Message: "No settings.json found",
			Advice:  "Run 'goenv vscode init' to create VS Code configuration",
		})
	} else {
		checks = append(checks, diagnosticCheck{
			Name:    "VS Code Settings",
			Status:  "pass",
			Message: "Settings file exists",
		})
	}

	// Check 2: Go version configured
	version, source, err := mgr.GetCurrentVersion()
	if err != nil {
		checks = append(checks, diagnosticCheck{
			Name:    "Go Version",
			Status:  "fail",
			Message: "No Go version configured",
			Advice:  "Run 'goenv local <version>' or 'goenv global <version>'",
		})
	} else {
		checks = append(checks, diagnosticCheck{
			Name:    "Go Version",
			Status:  "pass",
			Message: fmt.Sprintf("Go %s (from %s)", version, source),
		})

		// Check if version is installed
		if !mgr.IsVersionInstalled(version) {
			checks = append(checks, diagnosticCheck{
				Name:    "Go Installation",
				Status:  "fail",
				Message: fmt.Sprintf("Go %s is not installed", version),
				Advice:  fmt.Sprintf("Run 'goenv install %s'", version),
			})
		} else {
			checks = append(checks, diagnosticCheck{
				Name:    "Go Installation",
				Status:  "pass",
				Message: fmt.Sprintf("Go %s is installed", version),
			})
		}
	}

	// Check 3: gopls availability
	goplsPath, err := exec.LookPath("gopls")
	if err != nil {
		checks = append(checks, diagnosticCheck{
			Name:    "gopls",
			Status:  "warn",
			Message: "gopls not found in PATH",
			Advice:  "Install with: go install golang.org/x/tools/gopls@latest",
			Details: "gopls is the official Go language server for VS Code",
		})
	} else {
		checks = append(checks, diagnosticCheck{
			Name:    "gopls",
			Status:  "pass",
			Message: "gopls found",
			Details: goplsPath,
		})
	}

	// Check 4: go.toolsGopath writability
	homeDir, _ := os.UserHomeDir()
	toolsPath := filepath.Join(homeDir, "go", "tools")

	// Try to create if doesn't exist
	if err := utils.EnsureDirWithContext(toolsPath, "create tools directory"); err != nil {
		checks = append(checks, diagnosticCheck{
			Name:    "Tools Path",
			Status:  "fail",
			Message: "Cannot create tools directory",
			Advice:  fmt.Sprintf("Check permissions on %s", toolsPath),
			Details: err.Error(),
		})
	} else {
		// Test writability with a temp file
		testFile := filepath.Join(toolsPath, ".goenv-test")
		if err := utils.WriteFileWithContext(testFile, []byte("test"), utils.PermFileDefault, "tools path"); err != nil {
			checks = append(checks, diagnosticCheck{
				Name:    "Tools Path",
				Status:  "warn",
				Message: "Tools directory not writable",
				Advice:  fmt.Sprintf("Check permissions on %s", toolsPath),
				Details: err.Error(),
			})
		} else {
			os.Remove(testFile)
			checks = append(checks, diagnosticCheck{
				Name:    "Tools Path",
				Status:  "pass",
				Message: "Tools directory is writable",
				Details: toolsPath,
			})
		}
	}

	// Check 5: Workspace structure
	goWorkPath := filepath.Join(cwd, "go.work")
	goModPath := filepath.Join(cwd, config.GoModFileName)

	if utils.FileExists(goWorkPath) {
		checks = append(checks, diagnosticCheck{
			Name:    "Workspace",
			Status:  "pass",
			Message: "go.work found (monorepo)",
			Advice:  "Consider using 'goenv vscode init --template monorepo'",
		})
	} else if utils.FileExists(goModPath) {
		checks = append(checks, diagnosticCheck{
			Name:    "Workspace",
			Status:  "pass",
			Message: "go.mod found (single module)",
		})
	} else {
		checks = append(checks, diagnosticCheck{
			Name:    "Workspace",
			Status:  "warn",
			Message: "No go.mod or go.work found",
			Advice:  "Run 'go mod init' to initialize a module",
		})
	}

	// Check 6: Settings sync status (if settings exist)
	if utils.PathExists(settingsFile) && version != "" {
		result := vscode.CheckSettings(settingsFile, version)
		if result.HasSettings {
			if result.UsesEnvVars {
				checks = append(checks, diagnosticCheck{
					Name:    "Settings Sync",
					Status:  "pass",
					Message: "Using environment variables (auto-tracking)",
					Details: "Settings automatically track version changes",
				})
			} else if result.Mismatch {
				checks = append(checks, diagnosticCheck{
					Name:    "Settings Sync",
					Status:  "warn",
					Message: fmt.Sprintf("Version mismatch: configured=%s, current=%s", result.ConfiguredVersion, result.ExpectedVersion),
					Advice:  "Run 'goenv vscode sync' to update settings",
				})
			} else {
				checks = append(checks, diagnosticCheck{
					Name:    "Settings Sync",
					Status:  "pass",
					Message: "Settings match current version",
				})
			}
		}
	}

	// Check 7: Go extension recommendation
	extensionsFile := filepath.Join(vscodeDir, "extensions.json")
	if !utils.FileExists(extensionsFile) {
		checks = append(checks, diagnosticCheck{
			Name:    "Go Extension",
			Status:  "warn",
			Message: "No extension recommendations",
			Advice:  "Run 'goenv vscode init' to add Go extension recommendation",
		})
	} else {
		checks = append(checks, diagnosticCheck{
			Name:    "Go Extension",
			Status:  "pass",
			Message: "Extension recommendations configured",
		})
	}

	// Output results
	if vscodeDoctorFlags.json {
		// JSON output
		output := map[string]interface{}{
			"checks":    checks,
			"timestamp": time.Now().Format(time.RFC3339),
		}

		// Calculate overall status
		hasFailures := false
		hasWarnings := false
		for _, check := range checks {
			if check.Status == "fail" {
				hasFailures = true
			} else if check.Status == "warn" {
				hasWarnings = true
			}
		}

		if hasFailures {
			output["status"] = "fail"
		} else if hasWarnings {
			output["status"] = "warn"
		} else {
			output["status"] = "pass"
		}

		if err := cmdutil.OutputJSON(cmd.OutOrStdout(), output); err != nil {
			return errors.FailedTo("output JSON", err)
		}

		if hasFailures {
			return fmt.Errorf("health checks failed")
		}
		return nil
	}

	// Human-readable output
	fmt.Fprintln(cmd.OutOrStdout(), "🏥 VS Code Integration Health Check")
	fmt.Fprintln(cmd.OutOrStdout(), "===================================")
	fmt.Fprintln(cmd.OutOrStdout())

	passCount := 0
	warnCount := 0
	failCount := 0

	for _, check := range checks {
		var icon string
		switch check.Status {
		case "pass":
			icon = "✓"
			passCount++
		case "warn":
			icon = utils.Emoji("⚠️")
			warnCount++
		case "fail":
			icon = "✗"
			failCount++
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s %s: %s\n", icon, check.Name, check.Message)
		if check.Details != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", check.Details)
		}
		if check.Advice != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s%s\n", utils.Emoji("💡 "), check.Advice)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout(), "───────────────────────────────────")
	fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d passed, %d warnings, %d failed\n", passCount, warnCount, failCount)
	fmt.Fprintln(cmd.OutOrStdout())

	if failCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sSome critical checks failed. Please address the issues above.\n", utils.Emoji("❌ "))
		return fmt.Errorf("health checks failed")
	} else if warnCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sEverything is working, but some improvements are recommended.\n", utils.Emoji("⚠️  "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%sAll checks passed! Your VS Code integration is healthy.\n", utils.Emoji("✅ "))
	}

	return nil
}

// runVSCodeSetup runs unified setup (init + sync + doctor)
func runVSCodeSetup(cmd *cobra.Command, args []string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "%sRunning unified VS Code setup...\n\n",
		utils.Emoji("🚀 "))

	// Step 1: Initialize
	fmt.Fprintf(cmd.OutOrStdout(), "%sStep 1/3: Initializing VS Code workspace\n",
		utils.Emoji("📝 "))

	// Set init flags for this invocation
	originalForce := VSCodeInitFlags.Force
	originalTemplate := VSCodeInitFlags.Template
	originalDryRun := VSCodeInitFlags.DryRun

	VSCodeInitFlags.Force = true // Always overwrite in setup
	VSCodeInitFlags.Template = vscodeSetupFlags.template
	VSCodeInitFlags.DryRun = vscodeSetupFlags.dryRun

	err := runVSCodeInit(cmd, args)

	// Restore original flags
	VSCodeInitFlags.Force = originalForce
	VSCodeInitFlags.Template = originalTemplate
	VSCodeInitFlags.DryRun = originalDryRun

	if err != nil {
		return errors.FailedTo("initialize VS Code settings", err)
	}
	fmt.Fprintln(cmd.OutOrStdout())

	// Step 2: Sync (only if not dry-run)
	if !vscodeSetupFlags.dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "%sStep 2/3: Syncing with current Go version\n",
			utils.Emoji("🔄 "))

		// Set sync flags
		originalSyncDryRun := vscodeSyncFlags.dryRun
		vscodeSyncFlags.dryRun = false

		err = runVSCodeSync(cmd, args)

		// Restore original flag
		vscodeSyncFlags.dryRun = originalSyncDryRun

		if err != nil {
			// Sync might fail if using env vars (which is ok), or if no settings exist yet
			// Don't fail the whole setup, just warn
			fmt.Fprintf(cmd.OutOrStderr(), "%sSync step skipped: %v\n",
				utils.Emoji("ℹ️  "), err)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%sStep 2/3: Skipping sync (dry-run mode)\n\n",
			utils.Emoji("⏭️  "))
	}

	// Step 3: Doctor
	fmt.Fprintf(cmd.OutOrStdout(), "%sStep 3/3: Validating configuration\n",
		utils.Emoji("🔍 "))

	// Set doctor flags
	originalDoctorJSON := vscodeDoctorFlags.json
	vscodeDoctorFlags.json = vscodeSetupFlags.json

	err = runVSCodeDoctor(cmd, args)

	// Restore original flag
	vscodeDoctorFlags.json = originalDoctorJSON

	if err != nil {
		if vscodeSetupFlags.strict {
			return errors.FailedTo("validate VS Code settings", err)
		}
		fmt.Fprintf(cmd.OutOrStderr(), "%sWarning: Some checks failed (use --strict to fail on errors)\n",
			utils.Emoji("⚠️  "))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "\n%sSetup complete! VS Code is ready to use with goenv.\n",
		utils.Emoji("✅ "))

	return nil
}
