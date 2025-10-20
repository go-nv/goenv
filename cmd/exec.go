package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:   "exec <command> [args...]",
	Short: "Execute a command with the selected Go version",
	Long: `Runs an executable by first preparing PATH so that the selected Go version's bin directory is at the front.

goenv automatically rehashes after successful 'go install' commands, so installed tools are immediately available without running 'goenv rehash' manually.`,
	DisableFlagParsing: true, // Pass all flags through to the executed command
	Args: func(cmd *cobra.Command, args []string) error {
		// Handle -- separator (skip it if present)
		actualArgs := args
		if len(args) > 0 && args[0] == "--" {
			actualArgs = args[1:]
		}
		if len(actualArgs) == 0 {
			return fmt.Errorf("Usage: goenv exec <command> [arg1 arg2...]")
		}
		return nil
	},
	RunE: runExec,
}

func init() {
	rootCmd.AddCommand(execCmd)
	helptext.SetCommandHelp(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	// Handle -- separator (skip it if present)
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	// Get the current version
	currentVersion, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return fmt.Errorf("no Go version set: %w", err)
	}

	// Validate that the version is installed
	if currentVersion != "system" {
		if err := mgr.ValidateVersion(currentVersion); err != nil {
			// Provide specific error message based on where version was set
			if source == "GOENV_VERSION environment variable" {
				return fmt.Errorf("goenv: version '%s' is not installed (set by GOENV_VERSION environment variable)", currentVersion)
			} else if strings.Contains(source, ".go-version") || strings.Contains(source, "local") {
				return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", currentVersion, source)
			} else {
				return fmt.Errorf("goenv: version '%s' is not installed", currentVersion)
			}
		}
	}

	if cfg.Debug {
		fmt.Printf("Debug: Executing with Go version %s\n", currentVersion)
	}

	// Prepare environment
	env := os.Environ()

	// Expand GOPATH early if it needs expansion (handles $HOME, ~/, etc.)
	// This ensures Go doesn't error on shell metacharacters or variables
	gopath := os.Getenv("GOPATH")
	if gopath != "" {
		expanded := pathutil.ExpandPath(gopath)
		if expanded != gopath {
			gopath = expanded
			env = setEnvVar(env, "GOPATH", expanded)
		}
	}

	if currentVersion != "system" {
		versionPath, err := mgr.GetVersionPath(currentVersion)
		if err != nil {
			return fmt.Errorf("version path not found: %w", err)
		}

		// Add Go version's bin directory to PATH
		goBinPath := filepath.Join(versionPath, "bin")

		// Set GOROOT
		env = setEnvVar(env, "GOROOT", versionPath)

		// Prepend to PATH
		env = prependToPath(env, goBinPath)

		// Set GOPATH if not disabled
		if os.Getenv("GOENV_DISABLE_GOPATH") != "1" {
			if gopath == "" {
				// Set default GOPATH
				homeDir, _ := os.UserHomeDir()
				gopath = filepath.Join(homeDir, "go", currentVersion)
			}
			// gopath was already expanded above if it came from environment
			env = setEnvVar(env, "GOPATH", gopath)
		}
	}

	// Execute the command
	if len(args) == 0 {
		return fmt.Errorf("Usage: goenv exec <command> [arg1 arg2...]")
	}
	command := args[0]
	commandArgs := args[1:]

	// Execute pre-exec hooks
	executeHooks(hooks.PreExec, map[string]string{
		"version": currentVersion,
		"command": command,
	})

	var commandPath string

	if currentVersion != "system" {
		// First try to find command in the version's bin directory
		versionPath, err := mgr.GetVersionPath(currentVersion)
		if err != nil {
			return err
		}

		versionBinDir := filepath.Join(versionPath, "bin")
		commandPath = findBinaryInDir(versionBinDir, command)
		if commandPath == "" {
			return fmt.Errorf("goenv: %s: command not found", command)
		}
	} else {
		// For system version, use PATH lookup
		var err error
		commandPath, err = exec.LookPath(command)
		if err != nil {
			return fmt.Errorf("goenv: %s: command not found", command)
		}
	}

	// Execute with the modified environment
	execCmd := exec.Command(commandPath, commandArgs...)
	execCmd.Env = env
	execCmd.Stdin = os.Stdin
	execCmd.Stdout = cmd.OutOrStdout()
	execCmd.Stderr = cmd.ErrOrStderr()

	err = execCmd.Run()

	// Execute post-exec hooks
	executeHooks(hooks.PostExec, map[string]string{
		"version": currentVersion,
		"command": command,
	})

	// Auto-rehash after successful 'go install' command
	// Skip if GOENV_NO_AUTO_REHASH environment variable is set
	if err == nil && shouldAutoRehash(command, commandArgs) && os.Getenv("GOENV_NO_AUTO_REHASH") != "1" {
		if cfg.Debug {
			fmt.Fprintln(cmd.OutOrStdout(), "Debug: Auto-rehashing after go install")
		}
		// Run rehash silently - don't fail if it errors
		_ = runRehashSilent(cfg)
	}

	return err
}

// shouldAutoRehash determines if we should automatically rehash after command execution
func shouldAutoRehash(command string, args []string) bool {
	// Check if command is 'go' and first arg is 'install'
	if command != "go" && !strings.HasSuffix(command, "/go") && !strings.HasSuffix(command, "\\go.exe") {
		return false
	}

	// Check if 'install' is in the arguments
	for _, arg := range args {
		if arg == "install" {
			return true
		}
		// Stop at first non-flag argument
		if !strings.HasPrefix(arg, "-") {
			break
		}
	}

	return false
}

// runRehashSilent runs rehash without printing output
func runRehashSilent(cfg *config.Config) error {
	shimMgr := shims.NewShimManager(cfg)
	return shimMgr.Rehash()
}

// setEnvVar sets or updates an environment variable
func setEnvVar(env []string, key, value string) []string {
	prefix := key + "="
	for i, envVar := range env {
		if strings.HasPrefix(envVar, prefix) {
			env[i] = prefix + value
			return env
		}
	}
	return append(env, prefix+value)
}

// prependToPath prepends a directory to the PATH environment variable
func prependToPath(env []string, dir string) []string {
	const pathPrefix = "PATH="
	for i, envVar := range env {
		if strings.HasPrefix(envVar, pathPrefix) {
			currentPath := envVar[len(pathPrefix):]
			newPath := dir + string(os.PathListSeparator) + currentPath
			env[i] = pathPrefix + newPath
			return env
		}
	}
	// PATH not found, add it
	return append(env, pathPrefix+dir)
}

// findBinaryInDir searches for a binary in a directory, handling .exe on Windows
func findBinaryInDir(binDir, command string) string {
	// Try exact name first
	binaryPath := filepath.Join(binDir, command)
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// On Windows, try adding .exe extension
	if filepath.Ext(command) == "" {
		exePath := filepath.Join(binDir, command+".exe")
		if _, err := os.Stat(exePath); err == nil {
			return exePath
		}
	}

	return ""
}
