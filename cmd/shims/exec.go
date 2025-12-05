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
	"github.com/go-nv/goenv/internal/resolver"
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

	// Get the current version with resolution (e.g., "1.25" → "1.25.4")
	currentVersion, versionSpec, source, err := mgr.GetCurrentVersionResolved()
	if err != nil {
		// Provide enhanced error message with suggestions
		installed, _ := mgr.ListInstalledVersions()
		return errors.VersionNotInstalledDetailed(versionSpec, source, installed)
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
			env = setEnvVar(env, utils.EnvVarGopath, expanded)
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
		env = setEnvVar(env, utils.EnvVarGoroot, versionPath)

		// Prepend to PATH
		env = prependToPath(env, goBinPath)

		// Set GOPATH if not disabled
		if utils.GoenvEnvVarDisableGopath.UnsafeValue() != "1" {
			// Build version-specific GOPATH: $HOME/go/{version}
			homeDir, _ := os.UserHomeDir()
			versionGopath := filepath.Join(homeDir, "go", currentVersion)

			// Preserve existing GOPATH by prepending version-specific path.
			// This allows users to keep source code in existing locations while
			// giving priority to version-specific installed tools/packages.
			// See: https://github.com/go-nv/goenv/issues/147
			if gopath != "" {
				versionGopath = versionGopath + string(os.PathListSeparator) + gopath
			}

			env = setEnvVar(env, utils.EnvVarGopath, versionGopath)
		}

		// Set per-version and per-architecture GOCACHE
		//
		// Prevents two types of "exec format error":
		// 1. Version conflicts: Go 1.23 binaries incompatible with Go 1.24
		// 2. Architecture conflicts: Cross-compile binaries built for different arch
		//
		// Cache format: go-build-{GOOS}-{GOARCH}[-cgo]
		// Examples:
		//   - go-build-darwin-arm64      (native, no CGO)
		//   - go-build-darwin-arm64-cgo  (native, with CGO)
		//   - go-build-linux-amd64       (cross-compile)
		//
		// Simplified vs v2: Removed over-engineered ABI variants, GOEXPERIMENT,
		// and CGO hash suffixes that caused cache proliferation.
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
			env = setEnvVar(env, utils.EnvVarGocache, versionGocache)

			// Write build.info file for CGO builds (diagnostic data)
			// Non-blocking - failures don't affect execution
			if cgo.IsCGOEnabled(env) {
				buildInfo := cgo.GetBuildInfo(env)
				if writer, err := cache.TryNewAtomicWriter(versionGocache); err == nil && writer != nil {
					defer writer.Close()
					if buildInfoJSON, err := json.MarshalIndent(buildInfo, "", "  "); err == nil {
						buildInfoPath := filepath.Join(versionGocache, "build.info")
						_ = writer.WriteFile(buildInfoPath, buildInfoJSON, utils.PermFileDefault)
					}
				}
			}
		}

		// Set shared GOMODCACHE across all Go versions
		//
		// Module source code is version-agnostic and contains no compiled artifacts.
		// Sharing GOMODCACHE:
		// - Matches Go's native behavior (~/go/pkg/mod by default)
		// - Simpler than per-version GOPATH management
		// - Works safely for all use cases
		//
		// Location: $GOENV_ROOT/shared/go-mod
		//
		// Respects existing GOMODCACHE if already set (via go env -w or environment)
		if os.Getenv(utils.EnvVarGomodcache) == "" {
			versionGomodcache := filepath.Join(cfg.Root, "shared", "go-mod")
			env = setEnvVar(env, utils.EnvVarGomodcache, versionGomodcache)
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
		// Use centralized resolver to find the binary
		// Pass version source to control whether host bin is checked
		r := resolver.New(cfg)
		var err error
		commandPath, err = r.ResolveBinary(command, currentVersion, source)
		if err != nil {
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
