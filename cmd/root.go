package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/vscode"
	"github.com/go-nv/goenv/internal/workflow"
	"github.com/spf13/cobra"
)

// CommandGroup represents command group IDs for organizing help output
type CommandGroup string

const (
	GroupGettingStarted CommandGroup = "getting-started"
	GroupVersions       CommandGroup = "versions"
	GroupTools          CommandGroup = "tools"
	GroupShell          CommandGroup = "shell"
	GroupDiagnostics    CommandGroup = "diagnostics"
	GroupIntegrations   CommandGroup = "integrations"
	GroupAdvanced       CommandGroup = "advanced"
	GroupMeta           CommandGroup = "meta"
	GroupLegacy         CommandGroup = "legacy"
)

// Store the default help function to call for subcommands
var defaultHelpFunc func(*cobra.Command, []string)

var RootCmd = &cobra.Command{
	Use:   "goenv",
	Short: "Simple Go version management",
	Long: `goenv is a simple Go version management tool that allows you to:
- Install and manage multiple versions of Go
- Switch between Go versions globally or per project
- Automatically download the latest Go versions
- Manage Go installations with ease`,
	SuggestionsMinimumDistance: 2,
	PersistentPreRun: func(cmd *cobra.Command, args []string) {
		ctx := cmd.Context()

		// Parse all GOENV_* environment variables once
		env, err := utils.LoadEnvironment(ctx)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Failed to load environment: %v\n", err)
			os.Exit(1)
		}
		ctx = utils.EnvironmentToContext(ctx, env)

		// Create config and manager once
		cfg := config.LoadFromEnvironment(env)
		ctx = config.ToContext(ctx, cfg)

		mgr := manager.NewManager(cfg)
		ctx = manager.ToContext(ctx, mgr)

		// Store updated context back to command
		cmd.SetContext(ctx)

		// Propagate output options
		utils.SetOutputOptions(NoColor, Plain)
	},
	Run: func(cmd *cobra.Command, args []string) {
		// Only run smart detection if no subcommands or flags were provided
		// This prevents blocking on stdin when user runs "goenv --version" etc.
		if cmd.Flags().NFlag() > 0 || len(args) > 0 {
			// User provided flags or arguments, show help or let cobra handle it
			cmd.Help()
			return
		}

		// Smart version detection (tfswitch-like behavior)
		// Check for .go-version or go.mod in current directory
		cwd, err := os.Getwd()
		if err == nil {
			ctx := cmdutil.GetContexts(cmd)
			cfg := ctx.Config
			mgr := ctx.Manager

			// Check if GOENV_AUTO_INSTALL is enabled
			autoInstall := utils.GoenvEnvVarAutoInstall.IsTrue()

			if autoInstall {
				// Use auto-install workflow
				additionalFlags := utils.GoenvEnvVarAutoInstallFlags.UnsafeValue()

				setup := &workflow.AutoInstallSetup{
					Config:          cfg,
					Manager:         mgr,
					Stdout:          cmd.OutOrStdout(),
					Stderr:          cmd.OutOrStderr(),
					WorkingDir:      cwd,
					AdditionalFlags: additionalFlags,
					VSCodeUpdate: func(version string) error {
						return vscode.UpdateSettingsForVersion(cwd, cfg.Root, version)
					},
					InstallCallback: func(version string) error {
						// Find install command
						var installCmd *cobra.Command
						for _, c := range cmd.Root().Commands() {
							if c.Name() == "install" {
								installCmd = c
								break
							}
						}
						if installCmd == nil {
							return fmt.Errorf("install command not found")
						}

						// Set args based on version (empty = latest)
						if version != "" {
							installCmd.SetArgs([]string{version})
						} else {
							// Install latest with additional flags if provided
							args := []string{}
							if additionalFlags != "" {
								args = append(args, splitArgs(additionalFlags)...)
							}
							installCmd.SetArgs(args)
						}

						return installCmd.Execute()
					},
				}

				_, err := setup.Run()
				if err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
					os.Exit(1)
				}

				fmt.Fprintf(cmd.OutOrStdout(), "\n")
				cmd.Help()
				return
			}

			// Use interactive workflow (default)
			setup := &workflow.InteractiveSetup{
				Config:     cfg,
				Manager:    mgr,
				Stdout:     cmd.OutOrStdout(),
				Stderr:     cmd.OutOrStderr(),
				Stdin:      os.Stdin,
				WorkingDir: cwd,
				VSCodeUpdate: func(version string) error {
					return vscode.UpdateSettingsForVersion(cwd, cfg.Root, version)
				},
			}

			result, err := setup.Run()
			if err != nil {
				fmt.Fprintf(cmd.OutOrStderr(), "Error: %v\n", err)
				os.Exit(1)
			}

			// Handle installation if requested
			if result.InstallRequested {
				fmt.Fprintf(cmd.OutOrStdout(), "Installing Go %s...\n", result.Version)
				installCmd := cmd.Root().Commands()[0] // Find install command
				for _, c := range cmd.Root().Commands() {
					if c.Name() == "install" {
						installCmd = c
						break
					}
				}
				installCmd.SetArgs([]string{result.Version})
				if err := installCmd.Execute(); err != nil {
					fmt.Fprintf(cmd.OutOrStderr(), "Installation failed: %v\n", err)
					os.Exit(1)
				}
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
			}

			// If we found a version, show help and exit
			if result.VersionFound {
				fmt.Fprintf(cmd.OutOrStdout(), "\n")
				cmd.Help()
				return
			}
		}

		// If no command is provided and no .go-version, show help using cobra's built-in grouped help
		cmd.Help()
		os.Exit(1)
	},
}

func Execute() {
	// Check for version shorthand syntax before executing
	// If first arg looks like a version number, route to local command
	if len(os.Args) > 1 {
		arg := os.Args[1]
		if isVersionLike(arg) {
			// Rewrite args to call local command
			os.Args = append([]string{os.Args[0], "local"}, os.Args[1:]...)
		}
	}

	if err := RootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// isVersionLike checks if a string looks like a version number
func isVersionLike(s string) bool {
	// Match patterns like: 1.21.0, 1.21, 1, latest, system
	if s == manager.LatestVersion || s == manager.SystemVersion {
		return true
	}

	// Check for version pattern: digits.digits[.digits]
	// Simple regex-like check
	parts := 0
	for i, c := range s {
		if c >= '0' && c <= '9' {
			continue
		} else if c == '.' {
			if i == 0 || i == len(s)-1 {
				return false // can't start or end with dot
			}
			parts++
			if parts > 2 {
				return false // max 3 parts (x.y.z)
			}
		} else {
			return false // non-digit, non-dot character
		}
	}

	// Must have at least one digit
	return len(s) > 0
}

// customHelpFunc provides context-aware help that adapts to first-run scenarios
func customHelpFunc(cmd *cobra.Command, args []string) {
	// Only show custom first-run help for the root command
	// For subcommands, always show their standard help using the default help function
	if cmd.Parent() != nil {
		// This is a subcommand, use the default help
		if defaultHelpFunc != nil {
			defaultHelpFunc(cmd, args)
		}
		return
	}

	// This is the root command - check for first-run scenario
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	hasVersions := utils.HasAnyVersionsInstalled(cfg.Root)

	if !hasVersions {
		// Show focused help for first-time users
		showFirstRunHelp(cmd)
	} else {
		// Show standard help using default function
		if defaultHelpFunc != nil {
			defaultHelpFunc(cmd, args)
		}
	}
}

// showFirstRunHelp displays a focused, beginner-friendly help message
func showFirstRunHelp(cmd *cobra.Command) {
	w := cmd.OutOrStdout()

	// Welcome message
	fmt.Fprintln(w, utils.BoldBlue("goenv")+" - Go version manager")
	fmt.Fprintln(w)
	fmt.Fprintln(w, cmd.Short)
	fmt.Fprintln(w)

	// Getting started section
	fmt.Fprintln(w, utils.BoldYellow("🚀 Getting Started"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  No Go versions installed yet. Here's how to get started:")
	fmt.Fprintln(w)

	// Essential commands
	essential := []struct {
		name, desc string
	}{
		{"get-started", "Interactive guide to set up goenv (recommended for first-time users)"},
		{"install <version>", "Install a Go version (e.g., goenv install 1.21.5)"},
		{"install -l", "List available Go versions to install"},
		{"doctor", "Check your goenv installation and environment"},
		{"init", "Print shell initialization code"},
	}

	for _, item := range essential {
		fmt.Fprintf(w, "  %s %-20s %s %s\n",
			utils.Green("●"),
			utils.Cyan(item.name),
			utils.Gray("→"),
			item.desc)
	}

	fmt.Fprintln(w)

	// Quick start example
	fmt.Fprintln(w, utils.BoldYellow("📝 Quick Start"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  "+utils.Gray("# Interactive setup (easiest)"))
	fmt.Fprintln(w, "  "+utils.BoldGreen("goenv get-started"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  "+utils.Gray("# Or install manually"))
	fmt.Fprintln(w, "  "+utils.BoldGreen("goenv install 1.21.5"))
	fmt.Fprintln(w, "  "+utils.BoldGreen("goenv global 1.21.5"))
	fmt.Fprintln(w)

	// More help
	fmt.Fprintln(w, utils.BoldYellow("💡 More Information"))
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  To see all available commands:")
	fmt.Fprintln(w, "  "+utils.BoldGreen("goenv --help")+" (after installing a version)")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "  For help on a specific command:")
	fmt.Fprintln(w, "  "+utils.BoldGreen("goenv <command> --help"))
	fmt.Fprintln(w)
}

// handleFlagError provides contextual help for unknown flags or commands
func handleFlagError(cmd *cobra.Command, err error) error {
	fmt.Fprintf(cmd.ErrOrStderr(), "%sError: %v\n\n", utils.EmojiOr("❌ ", ""), err)

	// Check if error is about unknown command
	errMsg := err.Error()
	if strings.Contains(errMsg, "unknown command") || strings.Contains(errMsg, "unknown flag") {
		// Suggest exploring commands
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n", utils.BoldYellow("Not sure what command to use?"))
		fmt.Fprintf(cmd.ErrOrStderr(), "  • Run %s to browse commands by category\n", utils.Cyan("goenv explore"))
		fmt.Fprintf(cmd.ErrOrStderr(), "  • Run %s to see all available commands\n", utils.Cyan("goenv --help"))
		fmt.Fprintf(cmd.ErrOrStderr(), "  • Run %s to troubleshoot issues\n\n", utils.Cyan("goenv doctor"))
	}

	return err
}

func init() {
	// Save the default help function before we replace it
	defaultHelpFunc = RootCmd.HelpFunc()

	// Set custom help function that adapts to first-run scenarios
	RootCmd.SetHelpFunc(customHelpFunc)

	// Enable "did you mean" suggestions for mistyped commands
	RootCmd.SuggestionsMinimumDistance = 2 // Suggest on 2+ char difference
	RootCmd.SuggestFor = []string{}        // Allow suggestions for all commands

	// Custom unknown command handler with contextual suggestions
	RootCmd.SetFlagErrorFunc(handleFlagError)

	// Add helpful message after unknown command suggestions
	RootCmd.SilenceErrors = false // Show errors
	RootCmd.SilenceUsage = true   // Don't show full usage on every error

	// Add command groups to organize help output
	RootCmd.AddGroup(
		&cobra.Group{
			ID:    string(GroupGettingStarted),
			Title: "Getting Started:",
		},
		&cobra.Group{
			ID:    string(GroupVersions),
			Title: "Version Management:",
		},
		&cobra.Group{
			ID:    string(GroupTools),
			Title: "Tool Management:",
		},
		&cobra.Group{
			ID:    string(GroupShell),
			Title: "Shell Integration:",
		},
		&cobra.Group{
			ID:    string(GroupDiagnostics),
			Title: "Diagnostics & Maintenance:",
		},
		&cobra.Group{
			ID:    string(GroupIntegrations),
			Title: "IDE & CI Integration:",
		},
		&cobra.Group{
			ID:    string(GroupAdvanced),
			Title: "Advanced Commands:",
		},
		&cobra.Group{
			ID:    string(GroupMeta),
			Title: "Meta Commands:",
		},
		&cobra.Group{
			ID:    string(GroupLegacy),
			Title: "Legacy Commands (for compatibility):",
		},
	)

	// Add global flags here if needed
	RootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "Enable debug mode")
	RootCmd.PersistentFlags().BoolVar(&NoColor, "no-color", false, "Disable colored output")
	RootCmd.PersistentFlags().BoolVar(&Plain, "plain", false, "Plain output (no colors, no emojis)")

	// Interactive mode flags (for commands that support interactive features)
	RootCmd.PersistentFlags().BoolVar(&Interactive, "interactive", false, "Enable guided interactive mode with helpful prompts")
	RootCmd.PersistentFlags().BoolVarP(&Yes, "yes", "y", false, "Auto-confirm all prompts (non-interactive mode)")
	RootCmd.PersistentFlags().BoolVarP(&Quiet, "quiet", "q", false, "Suppress progress output (only show errors)")

	// Add version flag
	var showVersion bool
	RootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	// Override the default version behavior
	RootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		// Propagate output options to utils package
		utils.SetOutputOptions(NoColor, Plain)

		if showVersion {
			// Simple format for --version flag (matches bash version)
			fmt.Printf("goenv %s\n", AppVersion)
			os.Exit(0)
		}
		return nil
	}
}

var Debug bool
var NoColor bool
var Plain bool

// Interactive mode flags
var Interactive bool // --interactive: Enable guided mode with helpful prompts
var Yes bool         // --yes/-y: Auto-confirm all prompts
var Quiet bool       // --quiet/-q: Suppress progress output

// Version information
var (
	AppVersion   = "dev"
	AppCommit    = "unknown"
	AppBuildTime = "unknown"
)

// SetVersionInfo sets version information from main
func SetVersionInfo(v, c, bt string) {
	AppVersion = v
	AppCommit = c
	AppBuildTime = bt
}

// splitArgs splits a string into arguments, respecting quoted strings
// Handles single quotes, double quotes, and escaped characters
func splitArgs(s string) []string {
	var args []string
	var current strings.Builder
	var inQuote rune // ' or " or 0
	var escape bool

	for _, ch := range s {
		if escape {
			// Previous char was backslash, add this char literally
			current.WriteRune(ch)
			escape = false
			continue
		}

		switch ch {
		case '\\':
			// Escape next character
			escape = true
		case '\'', '"':
			if inQuote == 0 {
				// Start quote
				inQuote = ch
			} else if inQuote == ch {
				// End quote
				inQuote = 0
			} else {
				// Different quote type, add literally
				current.WriteRune(ch)
			}
		case ' ', '\t', '\n':
			if inQuote != 0 {
				// Inside quotes, add whitespace literally
				current.WriteRune(ch)
			} else if current.Len() > 0 {
				// End of argument
				args = append(args, current.String())
				current.Reset()
			}
		default:
			current.WriteRune(ch)
		}
	}

	// Add final argument if any
	if current.Len() > 0 {
		args = append(args, current.String())
	}

	return args
}
