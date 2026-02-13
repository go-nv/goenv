package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:   "hooks",
	Short: "Manage declarative hooks configuration",
	Long: `Manage declarative hooks that extend goenv functionality.

Hooks are defined in ~/.goenv/hooks.yaml using a declarative YAML format.
They allow you to automate actions like logging, notifications, and webhooks
without executing arbitrary code.

Available subcommands:
  init     - Generate a template hooks.yaml configuration
  list     - Show all configured hooks
  validate - Validate hooks.yaml configuration
  test     - Test hooks without executing them (dry-run)

For more information, see: https://github.com/go-nv/goenv/blob/master/docs/HOOKS.md`,
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Help()
	},
}

var hooksInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Generate a template hooks.yaml configuration",
	Long: `Generate a template hooks.yaml configuration file.

This creates ~/.goenv/hooks.yaml with examples of all available actions
and hook points. The configuration is disabled by default for safety.

To enable hooks after reviewing the configuration:
  1. Set 'enabled: true'
  2. Set 'acknowledged_risks: true'
  3. Customize the hooks for your needs

Example:
  goenv hooks init
  $EDITOR ~/.goenv/hooks.yaml
  goenv hooks validate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHooksInit(cmd, args)
	},
}

var hooksListCmd = &cobra.Command{
	Use:   "list",
	Short: "Show all configured hooks",
	Long: `List all configured hooks and their actions.

This displays:
  - Hook points (pre_install, post_install, etc.)
  - Actions configured for each hook
  - Current configuration status (enabled/disabled)
  - Available actions in the registry

Example:
  goenv hooks list`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHooksList(cmd, args)
	},
}

var hooksValidateCmd = &cobra.Command{
	Use:   "validate",
	Short: "Validate hooks.yaml configuration",
	Long: `Validate the hooks.yaml configuration file.

This checks:
  - YAML syntax is valid
  - All required fields are present
  - Action names are registered
  - Action parameters are valid
  - Security settings are proper

Example:
  goenv hooks validate`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHooksValidate(cmd, args)
	},
}

var hooksTestCmd = &cobra.Command{
	Use:   "test [hook-point]",
	Short: "Test hooks without executing them (dry-run)",
	Long: `Test hooks without actually executing them.

This performs a dry-run that:
  - Loads the configuration
  - Validates all hooks
  - Shows what would be executed
  - Does NOT perform actual actions

You can optionally specify a hook point to test only that hook.

Examples:
  goenv hooks test              # Test all hooks
  goenv hooks test pre_install  # Test only pre_install hooks`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runHooksTest(cmd, args)
	},
}

func init() {
	cmdpkg.RootCmd.AddCommand(hooksCmd)
	hooksCmd.AddCommand(hooksInitCmd)
	hooksCmd.AddCommand(hooksListCmd)
	hooksCmd.AddCommand(hooksValidateCmd)
	hooksCmd.AddCommand(hooksTestCmd)
}

// runHooksInit generates a template hooks.yaml configuration
func runHooksInit(cmd *cobra.Command, args []string) error {
	configPath := hooks.DefaultConfigPath()

	// Check if config already exists
	if utils.PathExists(configPath) {
		return fmt.Errorf("hooks configuration already exists at %s", configPath)
	}

	// Create template configuration
	template := generateTemplateConfig()

	// Ensure directory exists
	dir := filepath.Dir(configPath)
	if err := utils.EnsureDirWithContext(dir, "create directory"); err != nil {
		return err
	}

	// Write template
	if err := utils.WriteFileWithContext(configPath, []byte(template), utils.PermFileDefault, "write configuration"); err != nil {
		return err
	}

	fmt.Printf("%sCreated hooks configuration template at: %s\n\n", utils.Emoji("✅ "), configPath)
	fmt.Printf("%sIMPORTANT: Hooks are DISABLED by default for security.\n", utils.Emoji("⚠️  "))
	fmt.Println("\nTo enable hooks:")
	fmt.Println("  1. Review the configuration carefully")
	fmt.Println("  2. Set 'enabled: true'")
	fmt.Println("  3. Set 'acknowledged_risks: true'")
	fmt.Println("  4. Run: goenv hooks validate")

	return nil
}

// runHooksList shows all configured hooks
func runHooksList(cmd *cobra.Command, args []string) error {
	// Load configuration
	config, err := hooks.LoadConfig(hooks.DefaultConfigPath())
	if err != nil {
		return errors.FailedTo("load configuration", err)
	}

	// Show status
	fmt.Println("Hooks Configuration")
	fmt.Println("===================")
	fmt.Printf("Status: %s\n", formatStatus(config))
	fmt.Printf("Config: %s\n\n", hooks.DefaultConfigPath())

	// Show settings
	fmt.Println("Settings:")
	fmt.Printf("  Timeout:            %s\n", config.Settings.Timeout)
	fmt.Printf("  Max Actions:        %d\n", config.Settings.MaxActions)
	fmt.Printf("  Continue on Error:  %t\n", config.Settings.ContinueOnError)
	fmt.Printf("  Allow HTTP:         %t\n", config.Settings.AllowHTTP)
	fmt.Printf("  Allow Internal IPs: %t\n\n", config.Settings.AllowInternalIPs)

	// Show configured hooks
	if len(config.Hooks) == 0 {
		fmt.Println("No hooks configured.")
	} else {
		fmt.Println("Configured Hooks:")
		for hookPoint, actions := range config.Hooks {
			fmt.Printf("\n  %s: (%d actions)\n", hookPoint, len(actions))
			for i, action := range actions {
				fmt.Printf("    %d. %s\n", i+1, action.Action)
				// Show a few key parameters
				if file, ok := action.Params["file"].(string); ok {
					fmt.Printf("       file: %s\n", file)
				}
				if url, ok := action.Params["url"].(string); ok {
					fmt.Printf("       url: %s\n", url)
				}
				if title, ok := action.Params["title"].(string); ok {
					fmt.Printf("       title: %s\n", title)
				}
			}
		}
	}

	// Show available actions
	fmt.Println("\n\nAvailable Actions:")
	registry := hooks.DefaultRegistry()
	actions := registry.List()

	w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
	fmt.Fprintln(w, "  NAME\tDESCRIPTION")
	fmt.Fprintln(w, "  ----\t-----------")
	for _, name := range actions {
		if executor, ok := registry.Get(name); ok {
			fmt.Fprintf(w, "  %s\t%s\n", name, executor.Description())
		}
	}
	w.Flush()

	return nil
}

// runHooksValidate validates the hooks configuration
func runHooksValidate(cmd *cobra.Command, args []string) error {
	configPath := hooks.DefaultConfigPath()

	// Load configuration
	config, err := hooks.LoadConfig(configPath)
	if err != nil {
		fmt.Printf("%sConfiguration validation FAILED\n\n", utils.Emoji("❌ "))
		return errors.FailedTo("load configuration", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		fmt.Printf("%sConfiguration validation FAILED\n\n", utils.Emoji("❌ "))
		return errors.FailedTo("validate configuration", err)
	}

	// Validate each hook's actions
	registry := hooks.DefaultRegistry()
	errorCount := 0

	for hookPoint, actions := range config.Hooks {
		for i, action := range actions {
			// Check action exists
			executor, ok := registry.Get(action.Action)
			if !ok {
				fmt.Printf("%sHook '%s' action %d: unknown action '%s'\n", utils.Emoji("❌ "), hookPoint, i+1, action.Action)
				errorCount++
				continue
			}

			// Validate action parameters
			if err := executor.Validate(action.Params); err != nil {
				fmt.Printf("%sHook '%s' action %d (%s): %v\n", utils.Emoji("❌ "), hookPoint, i+1, action.Action, err)
				errorCount++
			}
		}
	}

	if errorCount > 0 {
		fmt.Printf("\n%sConfiguration validation FAILED with %d error(s)\n", utils.Emoji("❌ "), errorCount)
		return fmt.Errorf("validation failed")
	}

	// Success
	fmt.Printf("%sConfiguration validation PASSED\n\n", utils.Emoji("✅ "))
	fmt.Printf("Configuration file: %s\n", configPath)
	fmt.Printf("Status: %s\n", formatStatus(config))
	fmt.Printf("Hook points: %d\n", len(config.Hooks))

	totalActions := 0
	for _, actions := range config.Hooks {
		totalActions += len(actions)
	}
	fmt.Printf("Total actions: %d\n", totalActions)

	if !config.IsEnabled() {
		fmt.Printf("\n%sNote: Hooks are currently DISABLED\n", utils.Emoji("⚠️  "))
		fmt.Println("To enable: set 'enabled: true' and 'acknowledged_risks: true'")
	}

	return nil
}

// runHooksTest performs a dry-run of hooks
func runHooksTest(cmd *cobra.Command, args []string) error {
	configPath := hooks.DefaultConfigPath()

	// Load configuration
	config, err := hooks.LoadConfig(configPath)
	if err != nil {
		return errors.FailedTo("load configuration", err)
	}

	// Validate first
	if err := config.Validate(); err != nil {
		return errors.FailedTo("validate configuration", err)
	}

	// For testing, temporarily enable hooks (bypass the IsEnabled check)
	originalEnabled := config.Enabled
	originalAcknowledged := config.AcknowledgedRisks
	config.Enabled = true
	config.AcknowledgedRisks = true
	defer func() {
		config.Enabled = originalEnabled
		config.AcknowledgedRisks = originalAcknowledged
	}()

	// Determine which hook points to test
	hookPoints := []string{}
	if len(args) > 0 {
		hookPoints = args
	} else {
		// Test all configured hooks
		for hookPoint := range config.Hooks {
			hookPoints = append(hookPoints, hookPoint)
		}
	}

	if len(hookPoints) == 0 {
		fmt.Println("No hooks configured to test.")
		return nil
	}

	// Create executor
	executor := hooks.NewExecutor(config)

	// Test each hook point
	fmt.Printf("%sTesting hooks (dry-run mode)\n\n", utils.Emoji("🧪 "))

	for _, hookPoint := range hookPoints {
		// Get actions directly from config (bypass IsEnabled check for testing)
		actions := config.Hooks[hookPoint]
		if len(actions) == 0 {
			fmt.Printf("%sNo actions configured for hook point: %s\n\n", utils.Emoji("⚠️  "), hookPoint)
			continue
		}

		fmt.Printf("Testing hook point: %s (%d actions)\n", hookPoint, len(actions))
		fmt.Println(strings.Repeat("-", 50))

		// Test with sample variables
		testVars := map[string]string{
			"hook":      hookPoint,
			"version":   "1.21.0",
			"timestamp": time.Now().Format(time.RFC3339),
		}

		// Validate hook point
		if !hooks.IsValidHookPoint(hookPoint) {
			fmt.Printf("%sInvalid hook point: %s\n\n", utils.Emoji("⚠️  "), hookPoint)
			continue
		}

		messages, err := executor.TestExecute(hooks.HookPoint(hookPoint), testVars)
		if err != nil {
			fmt.Printf("%sTest FAILED: %v\n\n", utils.Emoji("❌ "), err)
		} else {
			fmt.Printf("%sTest PASSED\n", utils.Emoji("✅ "))
			if len(messages) > 0 {
				fmt.Println("   Actions that would be executed:")
				for _, msg := range messages {
					fmt.Printf("   - %s\n", msg)
				}
			}
			fmt.Println()
		}
	}

	fmt.Printf("%sDry-run testing complete\n", utils.Emoji("🧪 "))
	fmt.Println("\nNote: This was a simulation. No actual actions were performed.")

	return nil
}

// formatStatus returns a formatted status string
func formatStatus(config *hooks.Config) string {
	if config.IsEnabled() {
		return utils.Emoji("✅ ") + "ENABLED"
	}
	if config.Enabled && !config.AcknowledgedRisks {
		return utils.Emoji("⚠️  ") + "DISABLED (risks not acknowledged)"
	}
	return utils.Emoji("❌ ") + "DISABLED"
}

// generateTemplateConfig generates a template hooks.yaml
func generateTemplateConfig() string {
	return `# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
# ⚠️  SECURITY WARNING ⚠️
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━
#
# This file executes automated actions on your system.
# Treat it as executable code with the same caution as shell scripts.
#
# BEFORE ENABLING HOOKS:
#   - Review all hook configurations below carefully
#   - Understand what each action does
#   - Verify URLs, file paths, and commands are trusted
#   - Use restrictive file permissions: chmod 600 ~/.goenv/hooks.yaml
#   - Never commit secrets or credentials to version control
#
# SECURITY BEST PRACTICES:
#   - Keep allow_http: false (HTTPS only, recommended)
#   - Keep allow_internal_ips: false (prevents SSRF attacks)
#   - Only use run_command with trusted commands
#   - Review changes before enabling hooks (set both flags below to true)
#
# Documentation: https://github.com/go-nv/goenv/blob/master/docs/HOOKS.md
#
# ━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━

# goenv Hooks Configuration
# This file defines declarative hooks that extend goenv functionality.
# Hooks are executed at specific points during goenv operations.

version: 1

# IMPORTANT: Hooks are DISABLED by default for security
# To enable:
#   1. Review this configuration carefully
#   2. Set enabled: true
#   3. Set acknowledged_risks: true (confirms you understand security implications)
enabled: false
acknowledged_risks: false

# Global settings
settings:
  # Maximum time for hook execution (default: 5s, max: 30s)
  timeout: 5s
  
  # Maximum number of actions per hook (default: 10)
  max_actions: 10
  
  # Log file for hook execution (optional)
  log_file: ~/.goenv/hooks.log
  
  # Continue executing remaining actions if one fails (default: true)
  continue_on_error: true
  
  # Allow non-HTTPS URLs in http_webhook (default: false, RECOMMENDED)
  allow_http: false
  
  # Allow internal/private IPs in http_webhook (default: false, RECOMMENDED)
  allow_internal_ips: false

# Hook definitions
# Available hook points:
#   - pre_install    - Before installing a Go version
#   - post_install   - After installing a Go version
#   - pre_uninstall  - Before uninstalling a Go version
#   - post_uninstall - After uninstalling a Go version
#   - pre_exec       - Before executing a command
#   - post_exec      - After executing a command
#   - pre_rehash     - Before rehashing shims
#   - post_rehash    - After rehashing shims

hooks:
  # Example: Log installations
  pre_install:
    - action: log_to_file
      file: ~/.goenv/logs/install.log
      format: "[{timestamp}] Starting installation of Go {version}"
    
    - action: check_disk_space
      path: ~/.goenv
      min_free_mb: 1000
      on_insufficient: error
  
  post_install:
    - action: log_to_file
      file: ~/.goenv/logs/install.log
      format: "[{timestamp}] Completed installation of Go {version}"
    
    # Example: Desktop notification (macOS/Linux/Windows)
    - action: notify_desktop
      title: "goenv"
      message: "Successfully installed Go {version}"
      level: info
    
    # Example: Webhook notification
    # - action: http_webhook
    #   url: https://api.example.com/hooks
    #   method: POST
    #   headers:
    #     Content-Type: application/json
    #   body: '{"event":"install","version":"{version}"}'
  
  # Example: Set environment variables before execution
  # pre_exec:
  #   - action: set_env
  #     scope: hook
  #     variables:
  #       CGO_ENABLED: "1"
  #       GO_BUILD_FLAGS: "-v"

# Available Actions:
# 
# 1. log_to_file - Write log entries to files
#    Parameters:
#      file: <path>          (required) - Log file path
#      format: <template>    (optional) - Log entry format
# 
# 2. http_webhook - Send HTTP requests
#    Parameters:
#      url: <url>            (required) - Target URL (HTTPS recommended)
#      method: <method>      (optional) - GET, POST, PUT, PATCH (default: POST)
#      headers: <map>        (optional) - HTTP headers
#      body: <string>        (optional) - Request body
#      timeout: <seconds>    (optional) - Request timeout (1-30, default: 5)
# 
# 3. notify_desktop - Desktop notifications
#    Parameters:
#      title: <string>       (required) - Notification title (max 256 chars)
#      message: <string>     (required) - Notification message (max 1024 chars)
#      level: <level>        (optional) - info, warning, error (default: info)
# 
# 4. check_disk_space - Validate available disk space
#    Parameters:
#      path: <path>          (required) - Path to check
#      min_free_mb: <number> (required) - Minimum free space in MB
#      on_insufficient: <action> (optional) - error or warn (default: error)
# 
# 5. set_env - Set environment variables
#    Parameters:
#      variables: <map>      (required) - Variable name → value map
#      scope: <scope>        (optional) - hook or process (default: hook)
#
# Template Variables (available in all string parameters):
#   {version}    - Current Go version
#   {hook}       - Hook point name (e.g., "pre_install")
#   {timestamp}  - ISO 8601 timestamp
#   <custom>     - Variables set by set_env action
`
}
