package shims

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	cmdhooks "github.com/go-nv/goenv/cmd/hooks"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/binarycheck"
	"github.com/go-nv/goenv/internal/cache"
	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/envdetect"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/hooks"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/session"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var execCmd = &cobra.Command{
	Use:     "exec <command> [args...]",
	Short:   "Execute a command with the selected Go version",
	GroupID: string(cmdpkg.GroupAdvanced),
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
			return fmt.Errorf("usage: goenv exec <command> [arg1 arg2...]")
		}
		return nil
	},
	RunE: runExec,
}

func init() {
	cmdpkg.RootCmd.AddCommand(execCmd)
	helptext.SetCommandHelp(execCmd)
}

func runExec(cmd *cobra.Command, args []string) error {
	// Handle -- separator (skip it if present)
	if len(args) > 0 && args[0] == "--" {
		args = args[1:]
	}

	cfg, mgr := cmdutil.SetupContext()

	// Get the current version
	currentVersion, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return errors.FailedTo("determine active Go version", err)
	}

	// Validate that the version is installed
	if currentVersion != manager.SystemVersion {
		if err := mgr.ValidateVersion(currentVersion); err != nil {
			// Provide enhanced error message with suggestions
			installed, _ := mgr.ListInstalledVersions()
			return errors.VersionNotInstalledDetailed(currentVersion, source, installed)
		}
	}

	if cfg.Debug {
		fmt.Printf("Debug: Executing with Go version %s\n", currentVersion)
	}

	// Prepare environment
	env := os.Environ()

	// Expand GOPATH early if it needs expansion (handles $HOME, ~/, etc.)
	// This ensures Go doesn't error on shell metacharacters or variables
	gopath := os.Getenv(utils.EnvVarGopath)
	if gopath != "" {
		expanded := pathutil.ExpandPath(gopath)
		if expanded != gopath {
			gopath = expanded
			env = setEnvVar(env, "GOPATH", expanded)
		}
	}

	if currentVersion != manager.SystemVersion {
		versionPath, err := mgr.GetVersionPath(currentVersion)
		if err != nil {
			return errors.FailedTo("get version path", err)
		}

		// Add Go version's bin directory to PATH
		goBinPath := filepath.Join(versionPath, "bin")

		// Set GOROOT
		env = setEnvVar(env, "GOROOT", versionPath)

		// Prepend to PATH
		env = prependToPath(env, goBinPath)

		// Set GOPATH if not disabled
		if utils.GoenvEnvVarDisableGopath.UnsafeValue() != "1" {
			// Check environment variables for GOPATH control
			gopathPrefix := utils.GoenvEnvVarGopathPrefix.UnsafeValue()
			appendGopath := utils.GoenvEnvVarAppendGopath.UnsafeValue() == "1"
			prependGopath := utils.GoenvEnvVarPrependGopath.UnsafeValue() == "1"

			// Build version-specific GOPATH
			var versionGopath string
			if gopathPrefix == "" {
				homeDir, _ := os.UserHomeDir()
				versionGopath = filepath.Join(homeDir, "go", currentVersion)
			} else {
				versionGopath = filepath.Join(gopathPrefix, currentVersion)
			}

			// Handle GOPATH appending/prepending
			if gopath != "" && appendGopath {
				versionGopath = versionGopath + string(os.PathListSeparator) + gopath
			} else if gopath != "" && prependGopath {
				versionGopath = gopath + string(os.PathListSeparator) + versionGopath
			}

			env = setEnvVar(env, "GOPATH", versionGopath)
		}

		// Set version AND architecture-specific GOCACHE to prevent conflicts
		//
		// This prevents two types of "exec format error":
		// 1. Version conflicts: Go 1.23 binaries incompatible with Go 1.24
		// 2. Architecture conflicts: Cross-compile tool binaries (staticcheck, generators)
		//    built for linux/amd64 accidentally executed on darwin/arm64
		//
		// By isolating caches per version+GOOS+GOARCH, we ensure:
		// - Native builds use: go-build-host-host
		// - Cross-compiles use: go-build-{GOOS}-{GOARCH}
		// - Tool binaries stay architecture-appropriate
		if !utils.GoenvEnvVarDisableGocache.IsTrue() {
			customGocacheDir := utils.GoenvEnvVarGocacheDir.UnsafeValue()
			var versionGocache string

			// Determine target GOOS/GOARCH for cache isolation
			goos := utils.GetEnvValue(env, "GOOS")
			goarch := utils.GetEnvValue(env, "GOARCH")
			if goos == "" {
				goos = "host" // Use "host" as marker when targeting host platform
			}
			if goarch == "" {
				goarch = "host"
			}

			// Get path to Go binary for ABI auto-discovery
			goBinaryPath := filepath.Join(versionPath, "bin", "go")

			// Build cache path with architecture AND ABI variant suffix
			// ABI variants affect binary compatibility even when GOOS/GOARCH match
			cacheSuffix := buildCacheSuffix(goBinaryPath, goos, goarch, env)

			if customGocacheDir != "" {
				// Use custom GOCACHE directory if specified
				versionGocache = filepath.Join(customGocacheDir, currentVersion, cacheSuffix)
			} else {
				// Use GOENV_ROOT/versions/{version}/go-build-{GOOS}-{GOARCH} as default GOCACHE
				versionGocache = filepath.Join(versionPath, cacheSuffix)
			}
			env = setEnvVar(env, "GOCACHE", versionGocache)

			// Write build.info file to record CGO toolchain configuration
			// This helps diagnose cache issues and ensures cache transparency
			// Use atomic writer to prevent corruption from concurrent processes
			if cgo.IsCGOEnabled(env) {
				buildInfo := cgo.GetBuildInfo(env)

				// Try to acquire lock for writing (non-blocking)
				// If cache is locked by another process, skip writing (it's just diagnostic data)
				if writer, err := cache.TryNewAtomicWriter(versionGocache); err != nil {
					if cfg.Debug {
						fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Failed to create atomic writer: %v\n", err)
					}
				} else if writer != nil {
					defer writer.Close()

					// Marshal build info and write atomically via the writer
					buildInfoJSON, err := json.MarshalIndent(buildInfo, "", "  ")
					if err != nil && cfg.Debug {
						fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Failed to marshal build.info: %v\n", err)
					} else {
						buildInfoPath := filepath.Join(versionGocache, "build.info")
						if err := writer.WriteFile(buildInfoPath, buildInfoJSON, utils.PermFileDefault); err != nil && cfg.Debug {
							fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Failed to write build.info: %v\n", err)
						}
					}
				}
			}
		}

		// Set version-specific GOMODCACHE to prevent module conflicts
		if !utils.GoenvEnvVarDisableGomodcache.IsTrue() {
			customGomodcacheDir := utils.GoenvEnvVarGomodcacheDir.UnsafeValue()
			var versionGomodcache string
			if customGomodcacheDir != "" {
				// Use custom GOMODCACHE directory if specified
				versionGomodcache = filepath.Join(customGomodcacheDir, currentVersion)
			} else {
				// Use GOENV_ROOT/versions/{version}/go-mod as default GOMODCACHE
				versionGomodcache = filepath.Join(versionPath, "go-mod")
			}
			env = setEnvVar(env, "GOMODCACHE", versionGomodcache)
		}
	}

	// Execute the command
	if len(args) == 0 {
		return fmt.Errorf("usage: goenv exec <command> [arg1 arg2...]")
	}
	command := args[0]
	commandArgs := args[1:]

	// Execute pre-exec hooks
	cmdhooks.ExecuteHooks(hooks.PreExec, map[string]string{
		"version": currentVersion,
		"command": command,
	})

	var commandPath string

	if currentVersion != manager.SystemVersion {
		// First try to find command in the version's bin directory
		versionPath, err := mgr.GetVersionPath(currentVersion)
		if err != nil {
			return err
		}

		versionBinDir := filepath.Join(versionPath, "bin")
		commandPath = findBinaryInDir(versionBinDir, command)

		// If not found in version bin, check host-specific bin directory first
		if commandPath == "" {
			hostBinDir := cfg.HostBinDir()
			commandPath = findBinaryInDir(hostBinDir, command)
		}

		// If still not found, check GOPATH bin (if GOPATH is enabled)
		if commandPath == "" && utils.GoenvEnvVarDisableGopath.UnsafeValue() != "1" {
			// Get the GOPATH from environment (already set above)
			for _, envVar := range env {
				if strings.HasPrefix(envVar, "GOPATH=") {
					gopathValue := strings.TrimPrefix(envVar, "GOPATH=")
					// Handle multiple GOPATH entries (colon-separated on Unix, semicolon on Windows)
					gopaths := filepath.SplitList(gopathValue)
					for _, gp := range gopaths {
						gopathBinDir := filepath.Join(gp, "bin")
						commandPath = findBinaryInDir(gopathBinDir, command)
						if commandPath != "" {
							break
						}
					}
					break
				}
			}
		}

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

	// Verify binary architecture matches host (prevent exec format error)
	// Use session memoization to avoid repeated checks for the same tool
	memo := session.GetRebuildMemo()
	if !memo.HasChecked(commandPath) {
		// Note: Basic architecture verification happens at the OS level via exec
		// The Go runtime will return "exec format error" for architecture mismatches
		// Additional compatibility checks (ELF interpreter, libc) happen below

		// Mark as checked so we don't verify again this session
		memo.MarkChecked(commandPath)
	}

	// Check for WSL cross-execution issues (Windows binaries in WSL)
	if wslWarning := envdetect.CheckWSLCrossExecution(commandPath); wslWarning != "" {
		if cfg.Debug {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: WSL cross-execution warning\n")
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", wslWarning)
		// Continue execution - this is just a warning, not a fatal error
	}

	// Check for Rosetta mixed architecture issues (macOS Apple Silicon)
	if rosettaWarning := envdetect.CheckRosettaMixedArchitecture(commandPath); rosettaWarning != "" {
		if cfg.Debug {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Rosetta architecture warning\n")
		}
		fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n", rosettaWarning)
		// Continue execution - this is just a warning, not a fatal error
	}

	// Perform additional compatibility checks (ELF interpreter, glibc/musl, shebang)
	if binInfo, err := binarycheck.CheckBinary(commandPath); err == nil {
		issues := binarycheck.CheckCompatibility(binInfo)
		// Filter to errors only (ignore warnings for now to not break existing behavior)
		hasErrors := false
		for _, issue := range issues {
			if issue.Severity == "error" {
				hasErrors = true
				break
			}
		}
		if hasErrors {
			if cfg.Debug {
				fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Binary compatibility check failed\n")
			}
			return fmt.Errorf("cannot execute %s:\n\n%s", command, binarycheck.FormatIssues(issues))
		}
		// Log warnings in debug mode
		if cfg.Debug && len(issues) > 0 {
			fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Binary compatibility warnings:\n%s\n", binarycheck.FormatIssues(issues))
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
	cmdhooks.ExecuteHooks(hooks.PostExec, map[string]string{
		"version": currentVersion,
		"command": command,
	})

	// Auto-rehash after successful 'go install' command
	// Skip if GOENV_NO_AUTO_REHASH environment variable is set
	if err == nil && shouldAutoRehash(command, commandArgs) && utils.GoenvEnvVarNoAutoRehash.UnsafeValue() != "1" {
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
	// Check if command is 'go' (with or without path, with or without extension)
	baseName := filepath.Base(command)
	// Remove any Windows executable extensions
	for _, ext := range utils.WindowsExecutableExtensions() {
		baseName = strings.TrimSuffix(baseName, ext)
	}

	if baseName != "go" {
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

// buildCacheSuffix constructs a cache directory suffix that includes ABI variants.
// This is a wrapper around cache.BuildCacheSuffix for backward compatibility.
func buildCacheSuffix(goBinaryPath, goos, goarch string, env []string) string {
	return cache.BuildCacheSuffix(goBinaryPath, goos, goarch, env)
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
	// Use cross-platform binary finder
	if binaryPath, err := utils.FindExecutable(binDir, command); err == nil {
		return binaryPath
	}

	return ""
}
