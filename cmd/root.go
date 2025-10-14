package cmd

import (
	"fmt"
	"os"

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
		// Check if GOENV_AUTO_INSTALL is enabled
		if os.Getenv("GOENV_AUTO_INSTALL") == "1" {
			// Run install command with GOENV_AUTO_INSTALL_FLAGS if set
			installArgs := []string{"install"}
			if flags := os.Getenv("GOENV_AUTO_INSTALL_FLAGS"); flags != "" {
				// Split flags by space (simple implementation)
				// TODO: Improve this to handle quoted arguments properly
				installArgs = append(installArgs, flags)
			}

			// Set args and execute the install command
			cmd.Root().SetArgs(installArgs)
			if err := cmd.Root().Execute(); err != nil {
				fmt.Fprintf(os.Stderr, "goenv: install failed: %v\n", err)
				os.Exit(1)
			}
			return
		}

		// If no command is provided, show simple help message matching bash version
		fmt.Printf("goenv %s\n", appVersion)
		fmt.Println(`Usage: goenv <command> [<args>]

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
