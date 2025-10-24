package defaulttools

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

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
}

// DefaultConfig returns the default configuration with common Go tools
func DefaultConfig() *Config {
	return &Config{
		Enabled: true,
		Tools: []Tool{
			{
				Name:    "gopls",
				Package: "golang.org/x/tools/gopls",
				Version: "@latest",
				Binary:  "gopls",
			},
			{
				Name:    "golangci-lint",
				Package: "github.com/golangci/golangci-lint/cmd/golangci-lint",
				Version: "@latest",
				Binary:  "golangci-lint",
			},
			{
				Name:    "staticcheck",
				Package: "honnef.co/go/tools/cmd/staticcheck",
				Version: "@latest",
				Binary:  "staticcheck",
			},
			{
				Name:    "delve",
				Package: "github.com/go-delve/delve/cmd/dlv",
				Version: "@latest",
				Binary:  "dlv",
			},
		},
	}
}

// LoadConfig loads the default tools configuration from file
func LoadConfig(configPath string) (*Config, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		// Return default config if file doesn't exist
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	return &config, nil
}

// SaveConfig saves the configuration to file
func SaveConfig(configPath string, config *Config) error {
	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
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
		return fmt.Errorf("Go binary not found for version %s: %w", goVersion, err)
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
			"GOROOT="+goRoot,
			"GOPATH="+hostGopath, // Use host-specific GOPATH
		)
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
