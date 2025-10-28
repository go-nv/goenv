package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/go-nv/goenv/internal/utils"
	yaml "gopkg.in/yaml.v3"
)

// Config represents the complete hooks configuration
type Config struct {
	Version           int                 `yaml:"version"`
	Enabled           bool                `yaml:"enabled"`
	AcknowledgedRisks bool                `yaml:"acknowledged_risks"`
	Settings          Settings            `yaml:"settings"`
	Aliases           map[string]string   `yaml:"aliases,omitempty"`
	DefaultTools      []string            `yaml:"default_tools,omitempty"`
	Hooks             map[string][]Action `yaml:"hooks"`
}

// Settings contains global hook settings
type Settings struct {
	Timeout          string `yaml:"timeout"`            // Default: "5s"
	MaxActions       int    `yaml:"max_actions"`        // Default: 10
	LogFile          string `yaml:"log_file"`           // Default: ~/.goenv/hooks.log
	ContinueOnError  bool   `yaml:"continue_on_error"`  // Default: true
	AllowHTTP        bool   `yaml:"allow_http"`         // Default: false (HTTPS only)
	AllowInternalIPs bool   `yaml:"allow_internal_ips"` // Default: false (SSRF protection)
	StrictDNS        bool   `yaml:"strict_dns"`         // Default: false (reject on DNS failure when allow_internal_ips=false)
}

// Action represents a single hook action
type Action struct {
	Action    string                 `yaml:"action"`
	Condition string                 `yaml:"condition,omitempty"`
	Params    map[string]interface{} `yaml:",inline"`
}

// HookContext provides runtime context to actions
type HookContext struct {
	Command   string            // The goenv command being run
	Variables map[string]string // Template variables for interpolation
	Settings  Settings          // Global settings
	StartTime time.Time         // When hook execution started
}

// DefaultConfig returns a config with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Version:           1,
		Enabled:           false,
		AcknowledgedRisks: false,
		Settings: Settings{
			Timeout:          "5s",
			MaxActions:       10,
			LogFile:          "",
			ContinueOnError:  true,
			AllowHTTP:        false,
			AllowInternalIPs: false,
			StrictDNS:        false,
		},
		Hooks: make(map[string][]Action),
	}
}

// DefaultConfigPath returns the default hooks configuration path
func DefaultConfigPath() string {
	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join(".goenv", "hooks.yaml")
	}
	return filepath.Join(home, ".goenv", "hooks.yaml")
}

// LoadConfig loads the hooks configuration from the specified path
func LoadConfig(path string) (*Config, error) {
	// If file doesn't exist, return default config (hooks disabled)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	config := DefaultConfig()
	if err := yaml.Unmarshal(data, config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Apply defaults for empty settings
	if config.Settings.Timeout == "" {
		config.Settings.Timeout = "5s"
	}
	if config.Settings.MaxActions == 0 {
		config.Settings.MaxActions = 10
	}
	if config.Settings.LogFile == "" {
		goenvRoot := utils.GoenvEnvVarRoot.UnsafeValue()
		if goenvRoot == "" {
			home, _ := os.UserHomeDir()
			goenvRoot = filepath.Join(home, ".goenv")
		}
		config.Settings.LogFile = filepath.Join(goenvRoot, "hooks.log")
	}

	return config, nil
}

// SaveConfig saves the configuration to the specified path
func SaveConfig(path string, config *Config) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	// Ensure directory exists
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// ConfigPath returns the path to the hooks configuration file
func ConfigPath() string {
	// Check for GOENV_HOOKS_CONFIG env var first
	if path := utils.GoenvEnvVarHooksConfig.UnsafeValue(); path != "" {
		return path
	}

	// Default to $GOENV_ROOT/hooks.yaml
	goenvRoot := utils.GoenvEnvVarRoot.UnsafeValue()
	if goenvRoot == "" {
		home, _ := os.UserHomeDir()
		goenvRoot = filepath.Join(home, ".goenv")
	}

	return filepath.Join(goenvRoot, "hooks.yaml")
}

// Validate performs validation on the configuration
func (c *Config) Validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported config version: %d (expected 1)", c.Version)
	}

	if c.Enabled && !c.AcknowledgedRisks {
		return fmt.Errorf("hooks are enabled but acknowledged_risks is not set to true")
	}

	// Validate timeout format
	if _, err := time.ParseDuration(c.Settings.Timeout); err != nil {
		return fmt.Errorf("invalid timeout format: %s", c.Settings.Timeout)
	}

	// Validate max actions
	if c.Settings.MaxActions < 1 || c.Settings.MaxActions > 100 {
		return fmt.Errorf("max_actions must be between 1 and 100, got %d", c.Settings.MaxActions)
	}

	// Validate all actions
	registry := DefaultRegistry()
	for hookPoint, actions := range c.Hooks {
		if len(actions) > c.Settings.MaxActions {
			return fmt.Errorf("hook point %s has %d actions, max is %d",
				hookPoint, len(actions), c.Settings.MaxActions)
		}

		for i, action := range actions {
			if _, exists := registry.Get(action.Action); !exists {
				return fmt.Errorf("hook %s[%d]: unknown action type: %s",
					hookPoint, i, action.Action)
			}
		}
	}

	return nil
}

// IsEnabled returns true if hooks are enabled
func (c *Config) IsEnabled() bool {
	return c.Enabled && c.AcknowledgedRisks
}

// GetHooks returns the actions for a specific hook point
func (c *Config) GetHooks(hookPoint string) []Action {
	if !c.IsEnabled() {
		return nil
	}
	return c.Hooks[hookPoint]
}

// GetTimeout returns the parsed timeout duration
func (c *Config) GetTimeout() time.Duration {
	duration, err := time.ParseDuration(c.Settings.Timeout)
	if err != nil {
		return 5 * time.Second // Default fallback
	}
	return duration
}
