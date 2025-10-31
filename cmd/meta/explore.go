package meta

import (
	"fmt"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var exploreCmd = &cobra.Command{
	Use:     "explore [category]",
	Aliases: []string{"discover"},
	Short:   "Discover goenv commands by category",
	GroupID: string(cmdpkg.GroupGettingStarted),
	Long: `Interactive command discovery tool to help you find the right command.

Browse commands organized by what you want to do:
  - Getting Started: Setup and first-time use
  - Version Management: Install, switch, and manage Go versions
  - Tools: Manage Go tools and binaries
  - Shell Integration: Configure your shell
  - Diagnostics: Troubleshoot issues
  - Integrations: IDE and CI/CD setup
  - Advanced: Power user features

Examples:
  goenv explore                  # Interactive category selection
  goenv explore versions         # Show version management commands
  goenv explore getting-started  # Show beginner commands`,
	RunE: runExplore,
}

func init() {
	cmdpkg.RootCmd.AddCommand(exploreCmd)
}

type commandInfo struct {
	Name        string
	Description string
	Example     string
	Category    string
}

func runExplore(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Define command catalog organized by user intent
	catalog := map[string][]commandInfo{
		"getting-started": {
			{
				Name:        "get-started",
				Description: "Interactive setup guide for new users",
				Example:     "goenv get-started",
				Category:    "Getting Started",
			},
			{
				Name:        "setup",
				Description: "Automatic shell and IDE configuration",
				Example:     "goenv setup",
				Category:    "Getting Started",
			},
			{
				Name:        "doctor",
				Description: "Check installation and diagnose issues",
				Example:     "goenv doctor",
				Category:    "Getting Started",
			},
		},
		"versions": {
			{
				Name:        "install",
				Description: "Download and install a Go version",
				Example:     "goenv install 1.21.5",
				Category:    "Version Management",
			},
			{
				Name:        "uninstall",
				Description: "Remove an installed Go version",
				Example:     "goenv uninstall 1.20.0",
				Category:    "Version Management",
			},
			{
				Name:        "list",
				Description: "Show all installed Go versions",
				Example:     "goenv list",
				Category:    "Version Management",
			},
			{
				Name:        "install-list",
				Description: "Show available versions to install",
				Example:     "goenv install-list",
				Category:    "Version Management",
			},
			{
				Name:        "global",
				Description: "Set default Go version for all projects",
				Example:     "goenv global 1.21.5",
				Category:    "Version Management",
			},
			{
				Name:        "local",
				Description: "Set Go version for current directory",
				Example:     "goenv local 1.20.5",
				Category:    "Version Management",
			},
			{
				Name:        "shell",
				Description: "Set Go version for current shell session",
				Example:     "goenv shell 1.22.0",
				Category:    "Version Management",
			},
			{
				Name:        "version",
				Description: "Show currently active Go version",
				Example:     "goenv version",
				Category:    "Version Management",
			},
			{
				Name:        "info",
				Description: "Show detailed information about a version",
				Example:     "goenv info 1.21.5",
				Category:    "Version Management",
			},
			{
				Name:        "compare",
				Description: "Compare two Go versions side-by-side",
				Example:     "goenv compare 1.20.5 1.21.5",
				Category:    "Version Management",
			},
		},
		"tools": {
			{
				Name:        "tools install",
				Description: "Install a Go tool (gopls, staticcheck, etc.)",
				Example:     "goenv tools install golang.org/x/tools/gopls@latest",
				Category:    "Tool Management",
			},
			{
				Name:        "tools list",
				Description: "List installed tools for current version",
				Example:     "goenv tools list",
				Category:    "Tool Management",
			},
			{
				Name:        "tools update",
				Description: "Update all installed tools",
				Example:     "goenv tools update",
				Category:    "Tool Management",
			},
		},
		"shell": {
			{
				Name:        "init",
				Description: "Configure shell environment",
				Example:     "eval \"$(goenv init -)\"",
				Category:    "Shell Integration",
			},
			{
				Name:        "rehash",
				Description: "Rebuild shim files after installing tools",
				Example:     "goenv rehash",
				Category:    "Shell Integration",
			},
		},
		"diagnostics": {
			{
				Name:        "doctor",
				Description: "Diagnose installation issues",
				Example:     "goenv doctor",
				Category:    "Diagnostics",
			},
			{
				Name:        "status",
				Description: "Show current goenv status",
				Example:     "goenv status",
				Category:    "Diagnostics",
			},
		},
		"integrations": {
			{
				Name:        "vscode init",
				Description: "Configure VS Code for goenv",
				Example:     "goenv vscode init",
				Category:    "IDE Integration",
			},
			{
				Name:        "ci-setup",
				Description: "Set up goenv for CI/CD pipelines",
				Example:     "goenv ci-setup --cache",
				Category:    "CI/CD Integration",
			},
		},
		"advanced": {
			{
				Name:        "which",
				Description: "Show full path to an executable",
				Example:     "goenv which go",
				Category:    "Advanced",
			},
			{
				Name:        "whence",
				Description: "List versions with an executable",
				Example:     "goenv whence gofmt",
				Category:    "Advanced",
			},
			{
				Name:        "exec",
				Description: "Execute command with specific Go version",
				Example:     "goenv exec go version",
				Category:    "Advanced",
			},
		},
	}

	// If category specified, show commands for that category
	if len(args) > 0 {
		category := args[0]
		commands, exists := catalog[category]
		if !exists {
			// Show available categories
			fmt.Fprintf(cmd.OutOrStderr(), "%sUnknown category: %s\n\n", utils.EmojiOr("❌ ", "Error: "), category)
			fmt.Fprintf(cmd.OutOrStderr(), "Available categories:\n")
			for cat := range catalog {
				fmt.Fprintf(cmd.OutOrStderr(), "  • %s\n", utils.Cyan(cat))
			}
			return fmt.Errorf("unknown category")
		}

		// Show commands in category
		fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", utils.Emoji("📖 "), utils.BoldBlue(commands[0].Category))
		fmt.Fprintln(cmd.OutOrStdout())

		for _, c := range commands {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s\n", utils.Green("●"), utils.Cyan(c.Name))
			fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", c.Description)
			fmt.Fprintf(cmd.OutOrStdout(), "    %s %s\n", utils.Gray("Example:"), utils.Gray(c.Example))
			fmt.Fprintln(cmd.OutOrStdout())
		}

		// Show tip
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.Gray("Tip: Use 'goenv <command> --help' for detailed information"))
		return nil
	}

	// Interactive mode - show all categories with descriptions
	hasVersions := utils.HasAnyVersionsInstalled(cfg.Root)
	shellInit := utils.IsShellInitialized()

	fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n\n", utils.Emoji("🧭 "), utils.BoldBlue("Explore goenv Commands"))

	// Show status
	if !hasVersions || !shellInit {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.BoldYellow("Quick Status:"))
		if !shellInit {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s Shell not initialized → %s\n", utils.EmojiOr("⚠️  ", "!"), utils.Cyan("goenv get-started"))
		}
		if !hasVersions {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s No versions installed → %s\n", utils.EmojiOr("ℹ️  ", "i"), utils.Cyan("goenv install"))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Show categories
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n\n", utils.BoldBlue("Command Categories:"))

	categories := []struct {
		key         string
		name        string
		description string
		icon        string
	}{
		{"getting-started", "Getting Started", "Setup and first-time use", "🚀"},
		{"versions", "Version Management", "Install, switch, and manage Go versions", "📦"},
		{"tools", "Tool Management", "Manage Go tools and binaries", "🔧"},
		{"shell", "Shell Integration", "Configure your shell environment", "🐚"},
		{"diagnostics", "Diagnostics", "Troubleshoot and verify installation", "🔍"},
		{"integrations", "Integrations", "IDE and CI/CD setup", "🔌"},
		{"advanced", "Advanced", "Power user commands", "⚡"},
	}

	maxKeyLen := 0
	for _, cat := range categories {
		if len(cat.key) > maxKeyLen {
			maxKeyLen = len(cat.key)
		}
	}

	for _, cat := range categories {
		commands := catalog[cat.key]
		padding := strings.Repeat(" ", maxKeyLen-len(cat.key)+2)

		fmt.Fprintf(cmd.OutOrStdout(), "  %s %s%s%s %s (%d commands)\n",
			utils.Emoji(cat.icon+" "),
			utils.Cyan(cat.key),
			padding,
			utils.Gray("→"),
			cat.description,
			len(commands))
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.BoldBlue("Usage:"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s                  %s Browse all commands\n", utils.Cyan("goenv explore"), utils.Gray("→"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s %s  %s Show commands in category\n", utils.Cyan("goenv explore"), utils.Gray("<category>"), utils.Gray("→"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s %s     %s Get detailed help\n", utils.Cyan("goenv"), utils.Gray("<command> --help"), utils.Gray("→"))

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.BoldYellow("Quick Examples:"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s %s List version management commands\n", utils.Cyan("goenv explore versions"), utils.Gray("→"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s     %s Show diagnostic commands\n", utils.Cyan("goenv explore diagnostics"), utils.Gray("→"))

	return nil
}
