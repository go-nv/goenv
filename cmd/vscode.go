package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/vscode"
	"github.com/spf13/cobra"
)

var vscodeCmd = &cobra.Command{
	Use:     "vscode",
	Short:   "Manage VS Code integration",
	GroupID: "config",
	Long:    "Commands to configure and manage Visual Studio Code integration with goenv",
}

var vscodeInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize VS Code workspace for goenv",
	Long: `Initialize VS Code workspace with goenv configuration.

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
  goenv vscode init --template advanced`,
	RunE: runVSCodeInit,
}

var vscodeInitFlags struct {
	force    bool
	template string
	envVars  bool
}

func init() {
	rootCmd.AddCommand(vscodeCmd)
	vscodeCmd.AddCommand(vscodeInitCmd)

	vscodeInitCmd.Flags().BoolVarP(&vscodeInitFlags.force, "force", "f", false, "Overwrite existing settings")
	vscodeInitCmd.Flags().StringVarP(&vscodeInitFlags.template, "template", "t", "basic", "Configuration template (basic, advanced, monorepo)")
	vscodeInitCmd.Flags().BoolVar(&vscodeInitFlags.envVars, "env-vars", false, "Use environment variables instead of absolute paths (requires launching VS Code from terminal)")

	vscodeInitCmd.SilenceUsage = true
	helptext.SetCommandHelp(vscodeInitCmd)
}

// VSCodeSettings represents the VS Code settings.json structure
type VSCodeSettings map[string]interface{}

func runVSCodeInit(cmd *cobra.Command, args []string) error {
	return initializeVSCodeWorkspace(cmd)
}

// initializeVSCodeWorkspace performs the actual VS Code initialization
// This is exported so it can be called from other commands (e.g., goenv local --vscode)
func initializeVSCodeWorkspace(cmd *cobra.Command) error {
	return initializeVSCodeWorkspaceWithVersion(cmd, "")
}

// initializeVSCodeWorkspaceWithVersion initializes VS Code settings with a specific version
// If version is empty, it uses the current active version
func initializeVSCodeWorkspaceWithVersion(cmd *cobra.Command, version string) error {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")
	extensionsFile := filepath.Join(vscodeDir, "extensions.json")

	// Create .vscode directory if it doesn't exist
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return fmt.Errorf("failed to create .vscode directory: %w", err)
	}

	// Use basic template for automatic initialization
	template := "basic"
	force := vscodeInitFlags.force

	// Default to absolute paths for better UX (works when opened from GUI)
	// unless user explicitly requests env vars
	useAbsolutePaths := !vscodeInitFlags.envVars // If called from vscode init command, use flags
	if vscodeInitFlags.template != "" && vscodeInitFlags.template != "basic" {
		template = vscodeInitFlags.template
	}

	// Generate settings based on template
	settings, err := generateSettings(template)
	if err != nil {
		return err
	}

	// Convert to explicit paths if requested (using platform-specific env vars for portability)
	if useAbsolutePaths {
		cfg := config.Load()
		mgr := manager.NewManager(cfg)

		// Get Go version - use provided version or current active version
		if version == "" {
			version, _, err = mgr.GetCurrentVersion()
			if err != nil {
				return fmt.Errorf("failed to get current Go version: %w", err)
			}
		}

		// Get home directory (cross-platform)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		// Use platform-specific environment variables for portability
		// Windows: ${env:USERPROFILE}  Unix/macOS: ${env:HOME}
		var homeEnvVar string
		if runtime.GOOS == "windows" {
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

	// Handle existing settings
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing settings: %w", err)
	}

	// Write settings.json
	if existingSettings != nil && !force {
		// File exists - update only the specific Go keys we care about
		if useAbsolutePaths {
			fmt.Fprintln(cmd.OutOrStdout(), "ℹ️  Updating Go paths with absolute values")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "ℹ️  Updating Go paths with environment variables")
		}

		// Update only specific keys, not all settings
		keysToUpdate := make(map[string]interface{})

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

		if err := vscode.UpdateJSONKeys(settingsFile, keysToUpdate); err != nil {
			return fmt.Errorf("failed to update settings.json: %w", err)
		}
	} else {
		// No existing file or force mode - create new file
		if err := vscode.WriteJSONFile(settingsFile, settings); err != nil {
			return fmt.Errorf("failed to write settings.json: %w", err)
		}
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Created/updated %s\n", settingsFile)

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
		return fmt.Errorf("failed to write extensions.json: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Created/updated %s\n", extensionsFile)

	// Print configuration summary
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "✨ VS Code workspace configured for goenv!")
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
		fmt.Fprintln(cmd.OutOrStdout(), "  2. ✨ Ready to use - works even when opened from GUI!")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "💡 Note: If you change Go versions, re-run 'goenv vscode init' to update paths")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "  1. Close VS Code completely")
		fmt.Fprintln(cmd.OutOrStdout(), "  2. Ensure 'eval \"$(goenv init -)\"' is in your shell config")
		fmt.Fprintln(cmd.OutOrStdout(), "  3. Launch from terminal: code .")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "⚠️  Warning: Environment variable mode requires terminal launch")
		fmt.Fprintln(cmd.OutOrStdout(), "   Consider using default --absolute mode for better UX")
	}
	fmt.Fprintln(cmd.OutOrStdout(), "")

	return nil
}

// generateSettings creates settings based on template
func generateSettings(template string) (VSCodeSettings, error) {
	// Determine platform-specific home environment variable
	homeEnvVar := "${env:HOME}"
	if runtime.GOOS == "windows" {
		homeEnvVar = "${env:USERPROFILE}"
	}

	switch template {
	case "basic":
		return VSCodeSettings{
			"go.goroot":            "${env:GOROOT}",
			"go.gopath":            "${env:GOPATH}",
			"go.toolsGopath":       homeEnvVar + "/go/tools",
			"go.useLanguageServer": true,
		}, nil

	case "advanced":
		return VSCodeSettings{
			"go.goroot":                         "${env:GOROOT}",
			"go.gopath":                         "${env:GOPATH}",
			"go.toolsGopath":                    homeEnvVar + "/go/tools",
			"go.useLanguageServer":              true,
			"go.toolsManagement.autoUpdate":     true,
			"go.formatting.formatOnSave":        true,
			"go.lintOnSave":                     true,
			"go.testOnSave":                     false,
			"go.coverOnSave":                    false,
			"go.autocompleteUnimportedPackages": true,
			"gopls": map[string]interface{}{
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
			"go.goroot":            "${env:GOROOT}",
			"go.gopath":            "${env:GOPATH}",
			"go.toolsGopath":       homeEnvVar + "/go/tools",
			"go.useLanguageServer": true,
			"go.inferGopath":       false,
			"gopls": map[string]interface{}{
				"build.directoryFilters": []string{
					"-vendor",
					"-node_modules",
				},
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
