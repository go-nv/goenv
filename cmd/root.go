package cmd

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/workflow"
	"github.com/spf13/cobra"
)

var RootCmd = &cobra.Command{
	Use:   "goenv",
	Short: "Simple Go version management",
	Long: `goenv is a simple Go version management tool that allows you to:
- Install and manage multiple versions of Go
- Switch between Go versions globally or per project
- Automatically download the latest Go versions
- Manage Go installations with ease`,
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
			cfg := config.Load()
			mgr := manager.NewManager(cfg)

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
						return nil // TODO: Wire up VSCode integration without import cycle
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
					return nil // TODO: Wire up VSCode integration without import cycle
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
	if s == "latest" || s == "system" {
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

func init() {
	// Add command groups to organize help output
	RootCmd.AddGroup(
		&cobra.Group{
			ID:    "common",
			Title: "Common Commands:",
		},
		&cobra.Group{
			ID:    "tools",
			Title: "Tool Management:",
		},
		&cobra.Group{
			ID:    "config",
			Title: "Configuration:",
		},
		&cobra.Group{
			ID:    "system",
			Title: "System Commands:",
		},
	)

	// Add global flags here if needed
	RootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "Enable debug mode")
	RootCmd.PersistentFlags().BoolVar(&NoColor, "no-color", false, "Disable colored output")
	RootCmd.PersistentFlags().BoolVar(&Plain, "plain", false, "Plain output (no colors, no emojis)")

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
