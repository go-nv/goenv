package tools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/utils"
	"gopkg.in/yaml.v3"
)

// Config represents the default tools configuration
type Config struct {
	// Enabled controls whether default tools are installed automatically
	Enabled bool `yaml:"enabled"`

	// Tools is the list of tools to install with each new Go version
	Tools []Tool `yaml:"tools"`

	// UpdateStrategy is the default update strategy for all tools
	// Options: "latest", "minor", "patch", "pin", "auto", "latest_compatible"
	// Can be overridden per-tool
	UpdateStrategy string `yaml:"update_strategy,omitempty"`

	// Auto-update configuration (flat fields for simplicity)
	AutoUpdateEnabled     bool   `yaml:"auto_update_enabled,omitempty"`
	AutoUpdateStrategy    string `yaml:"auto_update_strategy,omitempty"`    // "on_use", "on_install", "manual"
	AutoUpdateInterval    string `yaml:"auto_update_interval,omitempty"`    // "24h", "7d", etc.
	AutoUpdateInteractive bool   `yaml:"auto_update_interactive,omitempty"` // Prompt to install vs just hint

	// LastChecked tracks last update check time per Go version
	LastChecked map[string]string `yaml:"last_checked,omitempty"`
}

// Tool represents a single tool to be installed
type Tool struct {
	// Name is a friendly name for the tool (for display)
	Name string `yaml:"name"`

	// Package is the Go package path (e.g., "golang.org/x/tools/gopls")
	Package string `yaml:"package"`

	// Version is an optional version constraint (e.g., "@latest", "@v0.14.2")
	// If empty, defaults to "@latest"
	Version string `yaml:"version,omitempty"`

	// Binary is the binary name that will be installed (if different from package name)
	// If empty, assumes last part of package path
	Binary string `yaml:"binary,omitempty"`

	// UpdateStrategy defines how this tool should be updated
	// Options: "latest", "minor", "patch", "pin", "auto", "latest_compatible"
	// If empty, uses config's default UpdateStrategy
	UpdateStrategy string `yaml:"update_strategy,omitempty"`

	// VersionStrategy determines version selection method
	// Options: "latest", "pinned", "latest_compatible"
	VersionStrategy string `yaml:"version_strategy,omitempty"`

	// MinGoVersion specifies the minimum Go version required for this tool
	MinGoVersion string `yaml:"min_go_version,omitempty"`

	// VersionOverrides maps Go version patterns to specific tool versions
	// Example: {"1.18": "v0.11.0", "1.19": "v0.13.0", "1.20+": "@latest"}
	VersionOverrides map[string]string `yaml:"version_overrides,omitempty"`
}

// DefaultConfig returns the default configuration with common Go tools
func DefaultConfig() *Config {
	return &Config{
		Enabled:               true,
		UpdateStrategy:        "auto", // Use auto strategy by default
		AutoUpdateEnabled:     false,  // Conservative default: don't auto-check
		AutoUpdateStrategy:    "on_use",
		AutoUpdateInterval:    "24h",
		AutoUpdateInteractive: false, // Just show hints by default
		Tools: []Tool{
			{
				Name:           "gopls",
				Package:        "golang.org/x/tools/gopls",
				Version:        "@latest",
				Binary:         "gopls",
				UpdateStrategy: "latest", // gopls updates frequently, always use latest
			},
			{
				Name:           "golangci-lint",
				Package:        "github.com/golangci/golangci-lint/cmd/golangci-lint",
				Version:        "@latest",
				Binary:         "golangci-lint",
				UpdateStrategy: "auto", // Use stable versions
			},
			{
				Name:           "staticcheck",
				Package:        "honnef.co/go/tools/cmd/staticcheck",
				Version:        "@latest",
				Binary:         "staticcheck",
				UpdateStrategy: "auto", // Use stable versions
			},
			{
				Name:           "delve",
				Package:        "github.com/go-delve/delve/cmd/dlv",
				Version:        "@latest",
				Binary:         "dlv",
				UpdateStrategy: "auto", // Use stable versions
			},
		},
	}
}

// LoadConfig loads the default tools configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Check if config file exists
	if utils.FileNotExists(configPath) {
		// Return default config if file doesn't exist
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, errors.FailedTo("read config file", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, errors.FailedTo("parse config file", err)
	}

	// Validate configuration
	if err := ValidateConfig(&config); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(configPath string, config *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := utils.EnsureDirWithContext(dir, "create config directory"); err != nil {
		return err
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return errors.FailedTo("marshal config", err)
	}

	if err := utils.WriteFileWithContext(configPath, data, utils.PermFileDefault, "write config file"); err != nil {
		return errors.FailedTo("write config file", err)
	}

	return nil
}

// ConfigPath returns the default config file path
func ConfigPath(goenvRoot string) string {
	return filepath.Join(goenvRoot, "default-tools.yaml")
}

// InstallTools installs all configured tools for a specific Go version
// Tools are installed to the host-specific GOPATH to enable cross-architecture dotfile syncing
func InstallTools(config *Config, goVersion string, goenvRoot string, hostGopath string, verbose bool) error {
	if !config.Enabled {
		if verbose {
			fmt.Println("Default tools installation is disabled")
		}
		return nil
	}

	if len(config.Tools) == 0 {
		if verbose {
			fmt.Println("No default tools configured")
		}
		return nil
	}

	// Set up environment for the specific Go version
	versionPath := filepath.Join(goenvRoot, "versions", goVersion)
	goRoot := versionPath // The version directory IS the GOROOT (no extra 'go' subdirectory)
	goBinBase := filepath.Join(goRoot, "bin", "go")

	// Find the executable (handles .exe and .bat on Windows)
	goBin, err := pathutil.FindExecutable(goBinBase)
	if err != nil {
		return fmt.Errorf("go binary not found for version %s: %w", goVersion, err)
	}

	if verbose {
		fmt.Printf("Installing %d default tool(s) for Go %s...\n", len(config.Tools), goVersion)
		fmt.Printf("  Tools will be installed to: %s/bin\n", hostGopath)
	}

	// Track results
	installed := []string{}
	failed := []string{}

	for _, tool := range config.Tools {
		if verbose {
			fmt.Printf("  Installing %s...", tool.Name)
		}

		// Build package reference with version
		pkg := tool.Package
		if tool.Version != "" {
			pkg = pkg + tool.Version
		} else {
			pkg = pkg + "@latest"
		}

		// Run go install
		cmd := exec.Command(goBin, "install", pkg)
		cmd.Env = append(os.Environ(),
			utils.EnvVarGoroot+"="+goRoot,
			utils.EnvVarGopath+"="+hostGopath, // Use host-specific GOPATH
		)

		// Set shared GOMODCACHE if not already set (matches exec.go behavior)
		if os.Getenv(utils.EnvVarGomodcache) == "" {
			sharedGomodcache := filepath.Join(goenvRoot, "shared", "go-mod")
			cmd.Env = append(cmd.Env, utils.EnvVarGomodcache+"="+sharedGomodcache)
		}

		cmd.Stdout = nil // Suppress output unless there's an error
		cmd.Stderr = nil

		if err := cmd.Run(); err != nil {
			if verbose {
				fmt.Printf(" %sFAILED\n", utils.Emoji("❌ "))
			}
			failed = append(failed, tool.Name)
		} else {
			if verbose {
				fmt.Printf(" %s\n", utils.Emoji("✅"))
			}
			installed = append(installed, tool.Name)
		}
	}

	if verbose && len(installed) > 0 {
		fmt.Printf("\n%sInstalled %d tool(s): %s\n", utils.Emoji("✅ "), len(installed), strings.Join(installed, ", "))
	}

	if len(failed) > 0 {
		return fmt.Errorf("failed to install %d tool(s): %s", len(failed), strings.Join(failed, ", "))
	}

	return nil
}

// VerifyTools checks which tools are installed for a specific Go version
func VerifyTools(config *Config, goVersion string, goenvRoot string) (map[string]bool, error) {
	results := make(map[string]bool)

	if len(config.Tools) == 0 {
		return results, nil
	}

	versionPath := filepath.Join(goenvRoot, "versions", goVersion)
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	for _, tool := range config.Tools {
		binaryName := tool.Binary
		if binaryName == "" {
			// Extract binary name from package path
			parts := strings.Split(tool.Package, "/")
			binaryName = parts[len(parts)-1]
		}

		binaryBasePath := filepath.Join(gopathBin, binaryName)

		// Check if executable exists (handles .exe and .bat on Windows)
		_, err := pathutil.FindExecutable(binaryBasePath)
		results[tool.Name] = (err == nil)
	}

	return results, nil
}

// GetToolUpdateStrategy returns the effective update strategy for a tool
// Takes into account per-tool strategy and config default
func (t *Tool) GetUpdateStrategy(config *Config) string {
	if t.UpdateStrategy != "" {
		return t.UpdateStrategy
	}
	if config.UpdateStrategy != "" {
		return config.UpdateStrategy
	}
	return "auto" // Default to auto strategy
}

// GetEffectiveVersion returns the version to install for a tool
// Combines the specified version with @ prefix if needed
func (t *Tool) GetEffectiveVersion() string {
	if t.Version == "" {
		return "@latest"
	}
	if !strings.HasPrefix(t.Version, "@") {
		return "@" + t.Version
	}
	return t.Version
}

// ValidateConfig validates the configuration
func ValidateConfig(config *Config) error {
	if len(config.Tools) == 0 {
		return nil // Empty config is valid
	}

	// Validate update strategies
	validStrategies := map[string]bool{
		"latest":            true,
		"minor":             true,
		"patch":             true,
		"pin":               true,
		"auto":              true,
		"latest_compatible": true,
		"":                  true, // Empty is valid (uses default)
	}

	if !validStrategies[config.UpdateStrategy] {
		return fmt.Errorf("invalid config update strategy: %s (must be: latest, minor, patch, pin, auto, or latest_compatible)", config.UpdateStrategy)
	}

	// Validate trigger strategies
	validTriggerStrategies := map[string]bool{
		"on_use":     true,
		"on_install": true,
		"manual":     true,
		"":           true, // Empty is valid
	}

	if !validTriggerStrategies[config.AutoUpdateStrategy] {
		return fmt.Errorf("invalid auto_update_strategy: %s (must be: on_use, on_install, or manual)", config.AutoUpdateStrategy)
	}

	// Validate each tool
	for i, tool := range config.Tools {
		if tool.Name == "" {
			return fmt.Errorf("tool %d: name is required", i)
		}
		if tool.Package == "" {
			return fmt.Errorf("tool %s: package is required", tool.Name)
		}
		if !validStrategies[tool.UpdateStrategy] {
			return fmt.Errorf("tool %s: invalid update strategy: %s", tool.Name, tool.UpdateStrategy)
		}
	}

	return nil
}

// ShouldAutoUpdate determines if tools should be auto-updated for this config
func (c *Config) ShouldAutoUpdate() bool {
	return c.AutoUpdateEnabled
}

// ShouldCheckOn determines if update checks should run for the given trigger
// trigger can be "use", "install", etc.
func (c *Config) ShouldCheckOn(trigger string) bool {
	if !c.AutoUpdateEnabled {
		return false
	}

	strategy := c.AutoUpdateStrategy
	if strategy == "" {
		strategy = "on_use" // Default
	}

	switch trigger {
	case "use":
		return strategy == "on_use"
	case "install":
		return strategy == "on_install"
	default:
		return false
	}
}

// GetToolByName finds a tool by name in the configuration
func (c *Config) GetToolByName(name string) *Tool {
	for i := range c.Tools {
		if c.Tools[i].Name == name {
			return &c.Tools[i]
		}
	}
	return nil
}

// ShouldCheckNow determines if an update check should run based on check interval
func (c *Config) ShouldCheckNow(goVersion string) bool {
	if !c.AutoUpdateEnabled {
		return false
	}

	// If no last check time, should check
	if c.LastChecked == nil {
		return true
	}

	lastCheckStr, ok := c.LastChecked[goVersion]
	if !ok {
		return true // Never checked for this version
	}

	// Parse last check time
	lastCheck, err := parseTimeString(lastCheckStr)
	if err != nil {
		return true // Invalid time, should check
	}

	// Parse check interval
	interval := c.AutoUpdateInterval
	if interval == "" {
		interval = "24h" // Default
	}

	duration, err := parseDuration(interval)
	if err != nil {
		return true // Invalid interval, check anyway
	}

	// Check if enough time has passed
	return lastCheck.Add(duration).Before(time.Now())
}

// MarkChecked records that an update check was performed
func (c *Config) MarkChecked(goVersion string) {
	if c.LastChecked == nil {
		c.LastChecked = make(map[string]string)
	}
	c.LastChecked[goVersion] = time.Now().Format(time.RFC3339)
}

// Helper functions for time parsing
func parseTimeString(s string) (time.Time, error) {
	return time.Parse(time.RFC3339, s)
}

func parseDuration(s string) (time.Duration, error) {
	// Support common formats like "7d", "24h", "30m"
	if len(s) < 2 {
		return 0, fmt.Errorf("invalid duration format")
	}

	// Extract number and unit
	unitIdx := len(s) - 1
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] >= '0' && s[i] <= '9' {
			unitIdx = i + 1
			break
		}
	}

	if unitIdx >= len(s) {
		return 0, fmt.Errorf("missing duration unit")
	}

	numStr := s[:unitIdx]
	unit := s[unitIdx:]

	var num int
	if _, err := fmt.Sscanf(numStr, "%d", &num); err != nil {
		return 0, fmt.Errorf("invalid duration number: %w", err)
	}

	switch unit {
	case "d":
		return time.Duration(num) * 24 * time.Hour, nil
	case "h":
		return time.Duration(num) * time.Hour, nil
	case "m":
		return time.Duration(num) * time.Minute, nil
	case "s":
		return time.Duration(num) * time.Second, nil
	default:
		return 0, fmt.Errorf("unsupported duration unit: %s", unit)
	}
}
