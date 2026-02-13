package tools

import (
	"fmt"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var defaultToolsCmd = &cobra.Command{
	Use:   "default-tools",
	Short: "Manage default tools installed with new Go versions",
	Long: `Manages the list of tools automatically installed with each new Go version.

Default tools are specified in ~/.goenv/default-tools.yaml and are automatically
installed after each 'goenv install' command completes successfully.

Common use cases:
  - Auto-install gopls, golangci-lint, staticcheck, dlv
  - Ensure consistent development environment across Go versions
  - Reduce manual setup after installing new Go versions

Examples:
  goenv tools default list              # Show configured tools
  goenv tools default init              # Create default config file
  goenv tools default enable            # Enable auto-installation
  goenv tools default disable           # Disable auto-installation
  goenv tools default install 1.25.2    # Install tools for specific version`,
}

var defaultToolsListCmd = &cobra.Command{
	Use:   "list",
	Short: "List configured default tools",
	Long:  `Displays the list of tools that will be automatically installed with new Go versions.`,
	RunE:  runDefaultToolsList,
}

var defaultToolsInitCmd = &cobra.Command{
	Use:   "init",
	Short: "Initialize default tools configuration",
	Long: `Creates ~/.goenv/default-tools.yaml with sensible defaults.

Default tools included:
  - gopls (Go language server)
  - golangci-lint (Linter aggregator)
  - staticcheck (Static analysis)
  - delve (Debugger)`,
	RunE: runDefaultToolsInit,
}

var enableCmd = &cobra.Command{
	Use:   "enable",
	Short: "Enable automatic tool installation",
	Long:  `Enables automatic installation of default tools when installing new Go versions.`,
	RunE:  runEnable,
}

var disableCmd = &cobra.Command{
	Use:   "disable",
	Short: "Disable automatic tool installation",
	Long:  `Disables automatic installation of default tools. The config file is preserved.`,
	RunE:  runDisable,
}

var installDefaultToolsCmd = &cobra.Command{
	Use:   "install <version>",
	Short: "Install default tools for a specific Go version",
	Long: `Manually installs all configured default tools for a specific Go version.

This is useful for:
  - Installing tools for existing Go versions
  - Reinstalling tools after they were removed
  - Testing the installation process`,
	Args: cobra.ExactArgs(1),
	RunE: runInstallTools,
}

var verifyCmd = &cobra.Command{
	Use:   "verify <version>",
	Short: "Verify which tools are installed for a Go version",
	Long:  `Checks which configured default tools are currently installed for a specific Go version.`,
	Args:  cobra.ExactArgs(1),
	RunE:  runVerify,
}

func init() {
	// Now registered as subcommand in tools.go
	defaultToolsCmd.AddCommand(defaultToolsListCmd)
	defaultToolsCmd.AddCommand(defaultToolsInitCmd)
	defaultToolsCmd.AddCommand(enableCmd)
	defaultToolsCmd.AddCommand(disableCmd)
	defaultToolsCmd.AddCommand(installDefaultToolsCmd)
	defaultToolsCmd.AddCommand(verifyCmd)
}

func runDefaultToolsList(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	configPath := tools.ConfigPath(cfg.Root)

	// Check if config exists
	if utils.FileNotExists(configPath) {
		fmt.Fprintln(cmd.OutOrStdout(), "No default tools configuration found.")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv tools default init' to create one.")
		return nil
	}

	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		return errors.FailedTo("load config", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), "Default Tools Configuration")
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "Status: %s\n", statusString(toolConfig.Enabled))
	fmt.Fprintf(cmd.OutOrStdout(), "Config: %s\n", configPath)
	fmt.Fprintln(cmd.OutOrStdout())

	if len(toolConfig.Tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tools configured.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Configured Tools (%d):\n", len(toolConfig.Tools))
	fmt.Fprintln(cmd.OutOrStdout())

	for i, tool := range toolConfig.Tools {
		version := tool.Version
		if version == "" {
			version = "@latest"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "%d. %s\n", i+1, tool.Name)
		fmt.Fprintf(cmd.OutOrStdout(), "   Package: %s\n", tool.Package)
		fmt.Fprintf(cmd.OutOrStdout(), "   Version: %s\n", version)
		if tool.Binary != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   Binary:  %s\n", tool.Binary)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	return nil
}

func runDefaultToolsInit(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	configPath := tools.ConfigPath(cfg.Root)

	// Check if config already exists
	if utils.PathExists(configPath) {
		fmt.Fprintf(cmd.OutOrStderr(), "Configuration file already exists: %s\n", configPath)
		fmt.Fprintln(cmd.OutOrStderr(), "Delete it first if you want to recreate with defaults.")
		return fmt.Errorf("config file already exists")
	}

	// Create default config
	toolConfig := tools.DefaultConfig()
	if err := tools.SaveConfig(configPath, toolConfig); err != nil {
		return errors.FailedTo("create config", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sCreated default tools configuration\n", utils.Emoji("✅ "))
	fmt.Fprintf(cmd.OutOrStdout(), "   Location: %s\n", configPath)
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "Default tools included:")
	for _, tool := range toolConfig.Tools {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s%s (%s)\n", utils.Emoji("• "), tool.Name, tool.Package)
	}
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "These tools will be automatically installed with new Go versions.")
	fmt.Fprintln(cmd.OutOrStdout(), "Edit the config file to customize the tool list.")

	return nil
}

func runEnable(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	configPath := tools.ConfigPath(cfg.Root)

	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		return errors.FailedTo("load config", err)
	}

	if toolConfig.Enabled {
		fmt.Fprintln(cmd.OutOrStdout(), "Default tools are already enabled.")
		return nil
	}

	toolConfig.Enabled = true
	if err := tools.SaveConfig(configPath, toolConfig); err != nil {
		return errors.FailedTo("save config", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sDefault tools enabled\n", utils.Emoji("✅ "))
	fmt.Fprintln(cmd.OutOrStdout(), "Tools will be automatically installed with new Go versions.")

	return nil
}

func runDisable(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	configPath := tools.ConfigPath(cfg.Root)

	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		return errors.FailedTo("load config", err)
	}

	if !toolConfig.Enabled {
		fmt.Fprintln(cmd.OutOrStdout(), "Default tools are already disabled.")
		return nil
	}

	toolConfig.Enabled = false
	if err := tools.SaveConfig(configPath, toolConfig); err != nil {
		return errors.FailedTo("save config", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sDefault tools disabled\n", utils.Emoji("✅ "))
	fmt.Fprintln(cmd.OutOrStdout(), "Tools will not be automatically installed with new Go versions.")
	fmt.Fprintln(cmd.OutOrStdout(), "Configuration file preserved.")

	return nil
}

func runInstallTools(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	configPath := tools.ConfigPath(cfg.Root)
	goVersion := args[0]

	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		return errors.FailedTo("load config", err)
	}

	if !toolConfig.Enabled {
		fmt.Fprintf(cmd.OutOrStderr(), "%sDefault tools are disabled in config\n", utils.Emoji("⚠️  "))
		fmt.Fprintln(cmd.OutOrStderr(), "Installing anyway...")
		fmt.Fprintln(cmd.OutOrStdout())
	}

	if len(toolConfig.Tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tools configured to install.")
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv tools default init' to create a default configuration.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Installing default tools for Go %s...\n", goVersion)
	fmt.Fprintln(cmd.OutOrStdout())

	if err := tools.InstallTools(toolConfig, goVersion, cfg.Root, cfg.SafeResolvePath(goVersion), true); err != nil {
		return errors.FailedTo("install default tools", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sRun 'goenv rehash' to make tools available as shims\n", utils.Emoji("💡 "))

	return nil
}

func runVerify(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	configPath := tools.ConfigPath(cfg.Root)
	goVersion := args[0]

	toolConfig, err := tools.LoadConfig(configPath)
	if err != nil {
		return errors.FailedTo("load config", err)
	}

	if len(toolConfig.Tools) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "No tools configured.")
		return nil
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Checking default tools for Go %s...\n", goVersion)
	fmt.Fprintln(cmd.OutOrStdout())

	results, err := tools.VerifyTools(toolConfig, goVersion, cfg.Root)
	if err != nil {
		return errors.FailedTo("verify tools", err)
	}

	installed := []string{}
	missing := []string{}

	for _, tool := range toolConfig.Tools {
		if results[tool.Name] {
			installed = append(installed, tool.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", utils.Emoji("✅ "), tool.Name)
		} else {
			missing = append(missing, tool.Name)
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s (not installed)\n", utils.Emoji("❌ "), tool.Name)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d installed, %d missing\n", len(installed), len(missing))

	if len(missing) > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "To install missing tools: goenv tools default install %s\n", goVersion)
	}

	return nil
}

func statusString(enabled bool) string {
	if enabled {
		return utils.Emoji("✅ ") + "Enabled"
	}
	return utils.Emoji("❌ ") + "Disabled"
}
