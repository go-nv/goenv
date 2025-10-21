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

var rootCmd = &cobra.Command{
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
						return initializeVSCodeWorkspaceWithVersion(cmd, version)
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
								// Simple space split - TODO: handle quoted args properly
								args = append(args, strings.Fields(additionalFlags)...)
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
				showHelpMessage(cmd)
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
					return initializeVSCodeWorkspaceWithVersion(cmd, version)
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
				showHelpMessage(cmd)
				return
			}
		}

		// If no command is provided and no .go-version, show simple help message matching bash version
		showHelpMessage(cmd)
		os.Exit(1)
	},
}

// showHelpMessage displays the goenv help information
func showHelpMessage(cmd *cobra.Command) {
	fmt.Fprintf(cmd.OutOrStdout(), "goenv %s\n", appVersion)
	fmt.Fprintln(cmd.OutOrStdout(), `Usage: goenv <command> [<args>]

Some useful goenv commands are:
   commands    List all available commands of goenv
   local       Set or show the local application-specific Go version
   global      Set or show the global Go version
   shell       Set or show the shell-specific Go version
   install     Install a Go version using go-build
   uninstall   Uninstall a specific Go version
   refresh     Clear caches and fetch fresh version data
   rehash      Rehash goenv shims (run this after installing executables)
   version     Show the current Go version and its origin
   versions    List all Go versions available to goenv
   which       Display the full path to an executable
   whence      List all Go versions that contain the given executable

See 'goenv help <command>' for information on a specific command.
For full documentation, see: https://github.com/go-nv/goenv#readme`)
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

	if err := rootCmd.Execute(); err != nil {
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
	// Add global flags here if needed
	rootCmd.PersistentFlags().BoolVarP(&Debug, "debug", "d", false, "Enable debug mode")

	// Add version flag
	var showVersion bool
	rootCmd.Flags().BoolVarP(&showVersion, "version", "v", false, "Show version information")

	// Override the default version behavior
	rootCmd.PreRunE = func(cmd *cobra.Command, args []string) error {
		if showVersion {
			// Simple format for --version flag (matches bash version)
			fmt.Printf("goenv %s\n", appVersion)
			os.Exit(0)
		}
		return nil
	}
}

var Debug bool

// Version information
var (
	appVersion   = "dev"
	appCommit    = "unknown"
	appBuildTime = "unknown"
)

// SetVersionInfo sets version information from main
func SetVersionInfo(v, c, bt string) {
	appVersion = v
	appCommit = c
	appBuildTime = bt
}
