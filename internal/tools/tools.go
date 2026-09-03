package tools

import (
	"bytes"
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
				Package:        "github.com/golangci/golangci-lint/v2/cmd/golangci-lint",
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
// verifyInstalledBinary confirms that `go install` actually produced the tool's
// binary in binDir. `go install` names the executable after the final element
// of the package import path; some tools also record an explicit Binary. A
// zero exit with no binary (non-main package, refused toolchain switch, ...)
// must not be reported as success.
//
// If no expected binary name can be determined (e.g. a discovered tool with no
// resolvable package path), there is nothing to verify and it returns nil.
func verifyInstalledBinary(binDir string, tool Tool) error {
	var candidates []string
	if n := installedBinaryName(tool.Package); n != "" {
		candidates = append(candidates, n)
	}
	if tool.Binary != "" && (len(candidates) == 0 || tool.Binary != candidates[0]) {
		candidates = append(candidates, tool.Binary)
	}
	if len(candidates) == 0 {
		return nil
	}
	for _, name := range candidates {
		if _, err := pathutil.FindExecutable(filepath.Join(binDir, name)); err == nil {
			return nil
		}
	}
	return fmt.Errorf("go install reported success but produced no binary (%s) in %s",
		strings.Join(candidates, " or "), binDir)
}

// installedBinaryName returns the executable name `go install <pkg>` produces.
// It is the final element of the import path, except that Go strips a trailing
// major-version element (".../v2" installs a binary named after the preceding
// element), which ExtractToolName does not account for.
func installedBinaryName(pkg string) string {
	name := ExtractToolName(pkg)
	if !isMajorVersionElement(name) {
		return name
	}
	base := pkg
	if at := strings.Index(base, "@"); at != -1 {
		base = base[:at]
	}
	base = strings.TrimSuffix(base, "/"+name)
	return ExtractToolName(base)
}

// isMajorVersionElement reports whether s is a Go module major-version path
// element such as "v2" or "v10".
func isMajorVersionElement(s string) bool {
	if len(s) < 2 || s[0] != 'v' {
		return false
	}
	for _, r := range s[1:] {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

// filterEnv returns a copy of env with any KEY=value entries whose key matches
// one of the given keys removed. It is used to strip inherited Go path
// variables before setting authoritative values, so the child `go` process
// never sees a duplicate (a duplicated GOPATH silently breaks `go install`).
func filterEnv(env []string, keys ...string) []string {
	drop := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		drop[k] = struct{}{}
	}
	out := make([]string, 0, len(env))
	for _, kv := range env {
		name := kv
		if i := strings.IndexByte(kv, '='); i >= 0 {
			name = kv[:i]
		}
		if _, skip := drop[name]; skip {
			continue
		}
		out = append(out, kv)
	}
	return out
}

func InstallTools(config *Config, goVersion string, goenvRoot string, hostGopath string, verbose bool) error {
	if !config.Enabled {
		if verbose {
			fmt.Println("Tools installation is disabled")
		}
		return nil
	}

	if len(config.Tools) == 0 {
		if verbose {
			fmt.Println("No tools configured")
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

	// Build the install environment once. We must REPLACE (not append) the Go
	// path variables: appending onto os.Environ() leaves a caller's ambient
	// GOPATH/GOBIN in place as a duplicate, and a duplicate GOPATH makes
	// `go install` silently install nothing while still exiting 0. Pinning GOBIN
	// to <hostGopath>/bin also guarantees the binary lands exactly where the
	// resolver and rehash look for it, regardless of the ambient environment.
	goBinDir := filepath.Join(hostGopath, "bin")
	gomodcache := pathutil.ExpandPath(os.Getenv(utils.EnvVarGomodcache))
	if gomodcache == "" {
		gomodcache = filepath.Join(goenvRoot, "shared", "go-mod") // matches exec.go behavior
	}
	// Strip GOCACHE too so an inherited literal '~' (or unexpanded $VAR) can't
	// make the `go install` below reject it, just like GOPATH; re-add it expanded.
	installEnv := filterEnv(os.Environ(),
		utils.EnvVarGoroot, utils.EnvVarGopath, utils.EnvVarGobin, utils.EnvVarGomodcache, utils.EnvVarGocache)
	installEnv = append(installEnv,
		utils.EnvVarGoroot+"="+goRoot,
		utils.EnvVarGopath+"="+hostGopath,
		utils.EnvVarGobin+"="+goBinDir,
		utils.EnvVarGomodcache+"="+gomodcache,
	)
	if gocache := pathutil.ExpandPath(os.Getenv(utils.EnvVarGocache)); gocache != "" {
		installEnv = append(installEnv, utils.EnvVarGocache+"="+gocache)
	}

	// Track results
	installed := []string{}
	failed := []string{}
	var firstError error

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
		cmd.Env = installEnv

		// Capture stderr for error reporting
		var stderr bytes.Buffer
		if verbose {
			// In verbose mode, show all output
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		} else {
			// In non-verbose mode, capture stderr for error reporting
			cmd.Stdout = nil
			cmd.Stderr = &stderr
		}

		if err := cmd.Run(); err != nil {
			if verbose {
				fmt.Printf(" %sFAILED\n", utils.Emoji("❌ "))
			} else {
				// Show error details in non-verbose mode
				fmt.Printf("  %s %s: %v\n", utils.Emoji("❌"), tool.Name, err)
				if stderr.Len() > 0 {
					fmt.Printf("    %s\n", strings.TrimSpace(stderr.String()))
				}
			}
			failed = append(failed, tool.Name)
			// Capture first error for context
			if firstError == nil {
				if stderr.Len() > 0 {
					firstError = fmt.Errorf("%s failed: %w\n%s", tool.Name, err, strings.TrimSpace(stderr.String()))
				} else {
					firstError = fmt.Errorf("%s failed: %w", tool.Name, err)
				}
			}
		} else if verifyErr := verifyInstalledBinary(goBinDir, tool); verifyErr != nil {
			// `go install` can exit 0 without producing a binary (for example a
			// non-main package, or a toolchain that declined to switch). Never
			// report that as a successful install.
			if verbose {
				fmt.Printf(" %sFAILED (no binary produced)\n", utils.Emoji("❌ "))
			} else {
				fmt.Printf("  %s %s: %v\n", utils.Emoji("❌"), tool.Name, verifyErr)
			}
			failed = append(failed, tool.Name)
			if firstError == nil {
				firstError = verifyErr
			}
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
		if firstError != nil {
			return fmt.Errorf("failed to install %d tool(s): %s\n\nFirst error: %w", len(failed), strings.Join(failed, ", "), firstError)
		}
		return fmt.Errorf("failed to install %d tool(s): %s", len(failed), strings.Join(failed, ", "))
	}

	return nil
}

// ToolBinDirs returns every directory a default tool binary may have been
// installed into, in the order they should be searched.
//
// These must stay in sync with InstallTools, which sets GOPATH to the
// hostGopath passed by its callers. Both call sites (the automatic
// post-install hook in cmd/core/install.go and the manual
// "goenv tools default-tools install" command) pass
// Config.SafeResolvePath(version), which resolves to the version directory
// itself once the version is installed — so "go install" writes to
// versions/<version>/bin, not versions/<version>/gopath/bin.
//
// Every entry is scoped to goVersion. The host-wide GOPATH
// (hosts/<os>-<arch>/gopath/bin) is deliberately excluded: it is shared by all
// versions, so including it would report a tool built against Go 1.24 as
// installed for 1.21. A false negative is a visible annoyance; a false positive
// is silent and sends the user to debug the wrong thing.
func ToolBinDirs(goenvRoot, goVersion string) []string {
	versionPath := filepath.Join(goenvRoot, "versions", goVersion)

	dirs := []string{
		// Where InstallTools writes today (GOPATH=versions/<version>).
		filepath.Join(versionPath, "bin"),
		// v3 version-scoped GOPATH, used by older goenv versions.
		filepath.Join(versionPath, "gopath", "bin"),
	}

	// Legacy "$GOPATH_PREFIX/<version>/bin" (default $HOME/go/<version>/bin)
	// used by the shims at runtime. Respects GOENV_GOPATH_PREFIX.
	if prefix := os.Getenv("GOENV_GOPATH_PREFIX"); prefix != "" {
		expanded := pathutil.ExpandPath(prefix)
		if expanded != "" {
			dirs = append(dirs, filepath.Join(filepath.Clean(expanded), goVersion, "bin"))
		} else if home, err := os.UserHomeDir(); err == nil && home != "" {
			dirs = append(dirs, filepath.Join(home, "go", goVersion, "bin"))
		}
	} else if home, err := os.UserHomeDir(); err == nil && home != "" {
		dirs = append(dirs, filepath.Join(home, "go", goVersion, "bin"))
	}

	return dirs
}

// VerifyTools checks which tools are installed for a specific Go version
func VerifyTools(config *Config, goVersion string, goenvRoot string) (map[string]bool, error) {
	results := make(map[string]bool)

	if len(config.Tools) == 0 {
		return results, nil
	}

	searchDirs := ToolBinDirs(goenvRoot, goVersion)

	for _, tool := range config.Tools {
		binaryName := tool.Binary
		if binaryName == "" {
			// Extract binary name from package path
			parts := strings.Split(tool.Package, "/")
			binaryName = parts[len(parts)-1]
		}

		found := false
		for _, dir := range searchDirs {
			// Check if executable exists (handles .exe and .bat on Windows)
			if _, err := pathutil.FindExecutable(filepath.Join(dir, binaryName)); err == nil {
				found = true
				break
			}
		}
		results[tool.Name] = found
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
