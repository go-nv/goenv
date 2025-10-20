package cmd

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var vscodeCmd = &cobra.Command{
	Use:   "vscode",
	Short: "Manage VS Code integration",
	Long:  "Commands to configure and manage Visual Studio Code integration with goenv",
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
} // VSCodeSettings represents the VS Code settings.json structure
type VSCodeSettings map[string]interface{}

// VSCodeExtensions represents the VS Code extensions.json structure
type VSCodeExtensions struct {
	Recommendations []string `json:"recommendations"`
}

func runVSCodeInit(cmd *cobra.Command, args []string) error {
	return initializeVSCodeWorkspace(cmd)
}

// initializeVSCodeWorkspace performs the actual VS Code initialization
// This is exported so it can be called from other commands (e.g., goenv local --vscode)
func initializeVSCodeWorkspace(cmd *cobra.Command) error {
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

	// Convert to absolute paths if requested
	if useAbsolutePaths {
		cfg := config.Load()
		mgr := manager.NewManager(cfg)

		// Get current Go version
		version, _, err := mgr.GetCurrentVersion()
		if err != nil {
			return fmt.Errorf("failed to get current Go version: %w", err)
		}

		// Get home directory (cross-platform)
		homeDir, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("failed to get home directory: %w", err)
		}

		// Convert environment variable references to absolute paths
		goroot := filepath.Join(cfg.Root, "versions", version)

		// Build GOPATH respecting GOENV_GOPATH_PREFIX (same as exec.go and sh-rehash.go)
		gopathPrefix := os.Getenv("GOENV_GOPATH_PREFIX")
		var gopath string
		if gopathPrefix == "" {
			gopath = filepath.Join(homeDir, "go", version)
		} else {
			gopath = filepath.Join(gopathPrefix, version)
		}

		settings["go.goroot"] = goroot
		settings["go.gopath"] = gopath

		// Update toolsGopath if it exists
		if _, ok := settings["go.toolsGopath"]; ok {
			settings["go.toolsGopath"] = filepath.Join(homeDir, "go", "tools")
		}
	}

	// Handle existing settings
	existingSettings, err := readExistingSettings(settingsFile)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to read existing settings: %w", err)
	}

	// Merge or overwrite settings
	finalSettings := settings
	if existingSettings != nil && !force {
		// Always override Go-related keys when switching modes or updating paths
		// This ensures the command actually does what the user expects
		finalSettings = mergeSettingsWithOverride(existingSettings, settings, []string{"go.goroot", "go.gopath", "go.toolsGopath"})
		if useAbsolutePaths {
			fmt.Fprintln(cmd.OutOrStdout(), "ℹ️  Updating Go paths with absolute values")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "ℹ️  Updating Go paths with environment variables")
		}
	}

	// Write settings.json
	if err := writeJSON(settingsFile, finalSettings); err != nil {
		return fmt.Errorf("failed to write settings.json: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Created/updated %s\n", settingsFile)

	// Generate and write extensions.json
	extensions := VSCodeExtensions{
		Recommendations: []string{
			"golang.go",
		},
	}

	if err := writeJSON(extensionsFile, extensions); err != nil {
		return fmt.Errorf("failed to write extensions.json: %w", err)
	}
	fmt.Fprintf(cmd.OutOrStdout(), "✓ Created/updated %s\n", extensionsFile)

	// Print configuration summary
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "✨ VS Code workspace configured for goenv!")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Configuration:")

	if useAbsolutePaths {
		fmt.Fprintf(cmd.OutOrStdout(), "  • go.goroot: %v (absolute path)\n", finalSettings["go.goroot"])
		fmt.Fprintf(cmd.OutOrStdout(), "  • go.gopath: %v (absolute path)\n", finalSettings["go.gopath"])
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
	switch template {
	case "basic":
		return VSCodeSettings{
			"go.goroot":            "${env:GOROOT}",
			"go.gopath":            "${env:GOPATH}",
			"go.toolsGopath":       "~/go/tools",
			"go.useLanguageServer": true,
		}, nil

	case "advanced":
		return VSCodeSettings{
			"go.goroot":                     "${env:GOROOT}",
			"go.gopath":                     "${env:GOPATH}",
			"go.toolsGopath":                "~/go/tools",
			"go.useLanguageServer":          true,
			"go.toolsManagement.autoUpdate": true,
			"[go]": map[string]interface{}{
				"editor.formatOnSave": true,
				"editor.codeActionsOnSave": map[string]interface{}{
					"source.organizeImports": "explicit",
				},
			},
			"gopls": map[string]interface{}{
				"ui.completion.usePlaceholders": true,
				"ui.diagnostic.analyses": map[string]interface{}{
					"unusedparams": true,
					"shadow":       true,
				},
			},
		}, nil

	case "monorepo":
		return VSCodeSettings{
			"go.goroot":            "${env:GOROOT}",
			"go.gopath":            "${env:GOPATH}",
			"go.toolsGopath":       "~/go/tools",
			"go.useLanguageServer": true,
			"go.inferGopath":       false,
			"gopls": map[string]interface{}{
				"build.directoryFilters": []string{
					"-node_modules",
					"-vendor",
				},
			},
		}, nil

	default:
		return nil, fmt.Errorf("unknown template: %s (choose: basic, advanced, monorepo)", template)
	}
}

// readExistingSettings reads and parses existing settings.json
func readExistingSettings(path string) (VSCodeSettings, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var settings VSCodeSettings
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("invalid JSON in existing settings: %w", err)
	}

	return settings, nil
}

// mergeSettings merges new settings into existing, preserving existing keys
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

// writeJSON writes a JSON file with proper formatting
func writeJSON(path string, data interface{}) error {
	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return err
	}

	if err := os.WriteFile(path, jsonData, 0644); err != nil {
		return err
	}

	return nil
}
