package meta

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var getStartedCmd = &cobra.Command{
	Use:     "get-started",
	Aliases: []string{"getting-started", "quickstart", "first-run"},
	Short:   "Show getting started guide for new users",
	GroupID: string(cmdpkg.GroupGettingStarted),
	Long: `Display a quick start guide for new goenv users.

This command shows you how to:
  - Initialize goenv in your shell
  - Install your first Go version
  - Set a default version

Perfect for first-time users or when you need a refresher.`,
	RunE: runGetStarted,
}

func init() {
	cmdpkg.RootCmd.AddCommand(getStartedCmd)
}

func runGetStarted(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Show welcome message
	fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n\n", utils.Emoji("👋 "), utils.BoldBlue("Welcome to goenv!"))

	// Check which steps are needed
	hasVersions := utils.HasAnyVersionsInstalled(cfg.Root)
	shellInit := utils.IsShellInitialized()

	if !shellInit {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.BoldYellow("Step 1: Initialize goenv in your shell"))
		fmt.Fprintf(cmd.OutOrStdout(), "To get started, add goenv to your shell by running:\n\n")

		// Provide shell-specific instructions
		shell := shellutil.DetectShell()
		switch shell {
		case shellutil.ShellTypeBash:
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", utils.Cyan("echo 'eval \"$(goenv init -)\"' >> ~/.bashrc"))
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", utils.Cyan("source ~/.bashrc"))
		case shellutil.ShellTypeZsh:
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", utils.Cyan("echo 'eval \"$(goenv init -)\"' >> ~/.zshrc"))
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", utils.Cyan("source ~/.zshrc"))
		case shellutil.ShellTypeFish:
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", utils.Cyan("echo 'status --is-interactive; and goenv init - | source' >> ~/.config/fish/config.fish"))
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", utils.Cyan("source ~/.config/fish/config.fish"))
		default:
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", utils.Cyan("eval \"$(goenv init -)\""))
		}

		fmt.Fprintf(cmd.OutOrStdout(), "Or for just this session:\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", utils.Cyan("eval \"$(goenv init -)\""))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n", utils.Green("✓"), utils.Green("goenv is initialized in your shell"))
	}

	if !hasVersions {
		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.BoldYellow("Step 2: Install a Go version"))
		fmt.Fprintf(cmd.OutOrStdout(), "Install your first Go version:\n\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s        %s Install latest stable Go\n", utils.Cyan("goenv install"), utils.Gray("→"))
		fmt.Fprintf(cmd.OutOrStdout(), "  %s %s Install specific version\n", utils.Cyan("goenv install 1.21.5"), utils.Gray("→"))
		fmt.Fprintf(cmd.OutOrStdout(), "  %s      %s List all available versions\n\n", utils.Cyan("goenv install -l"), utils.Gray("→"))

		fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.BoldYellow("Step 3: Set your default version"))
		fmt.Fprintf(cmd.OutOrStdout(), "After installing:\n\n")
		fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", utils.Cyan("goenv global <version>"))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n\n", utils.Green("✓"), utils.Green("You have Go versions installed"))
	}

	// Add additional tips
	fmt.Fprintf(cmd.OutOrStdout(), "%s\n", utils.BoldBlue("Helpful Commands:"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s    Check your goenv installation\n", utils.Cyan("goenv doctor"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s    Show your current configuration\n", utils.Cyan("goenv status"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s  List all commands\n", utils.Cyan("goenv --help"))
	fmt.Fprintf(cmd.OutOrStdout(), "  %s List installable versions\n", utils.Cyan("goenv install -l"))
	fmt.Fprintln(cmd.OutOrStdout())

	return nil
}
