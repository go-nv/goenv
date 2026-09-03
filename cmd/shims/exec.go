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

	// Recursion guard: detect if we're being called from a shim that we spawned.
	// We only treat this as recursion if the value equals our parent PID, indicating
	// we were spawned by another goenv exec (shim → exec → shim → exec).
	// This avoids false positives when running tests under goenv-managed Go.
	const recursionEnvVar = "_GOENV_EXEC_ACTIVE"
	parentPIDStr := fmt.Sprintf("%d", os.Getppid())
	if os.Getenv(recursionEnvVar) == parentPIDStr {
		return fmt.Errorf("goenv: infinite recursion detected — shim is calling goenv exec again. Run 'goenv rehash' to regenerate shims")
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	mgr := ctx.Manager
	env := ctx.Environment

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
	execEnv := os.Environ()

	// Normalize any inherited GOPATH up front so goenv never hands `go` a value
	// it rejects (a literal '~' from a quoted GOPATH="~/go", a non-absolute
	// entry, etc.). This runs on every platform, including Windows .bat shims.
	rawGopath := os.Getenv(utils.EnvVarGopath)
	sanitizedGopath := sanitizeInheritedGopath(rawGopath, cfg.GopathPrefix())
	if rawGopath != "" {
		execEnv = setEnvVar(execEnv, utils.EnvVarGopath, strings.Join(sanitizedGopath, string(os.PathListSeparator)))
	}

	// Normalize the other Go path env vars we may hand to `go`. A shell that left
	// a literal '~' or an unexpanded $VAR in one (e.g. a quoted GOCACHE="~/c")
	// would otherwise make `go` reject it just like GOPATH.
	for _, key := range []string{utils.EnvVarGocache, utils.EnvVarGomodcache, utils.EnvVarGobin} {
		if v := utils.GetEnvValue(execEnv, key); v != "" {
			if expanded := pathutil.ExpandPath(v); expanded != v {
				execEnv = setEnvVar(execEnv, key, expanded)
			}
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
		execEnv = setEnvVar(execEnv, utils.EnvVarGoroot, versionPath)

		// Prepend to PATH
		execEnv = prependToPath(execEnv, goBinPath)

		// Set GOPATH if not disabled
		if !env.HasDisableGopath() {
			// Build version-specific GOPATH: $GOPATH_PREFIX/<version> (default
			// $HOME/go/<version>), then keep the user's own (normalized) entries so
			// their source stays where they expect. See issue #147.
			versionGopath := cfg.ManagedGopath(currentVersion)
			if len(sanitizedGopath) > 0 {
				versionGopath = versionGopath + string(os.PathListSeparator) + strings.Join(sanitizedGopath, string(os.PathListSeparator))
			}
			execEnv = setEnvVar(execEnv, utils.EnvVarGopath, versionGopath)
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
			customGocacheDir := pathutil.ExpandPath(utils.GoenvEnvVarGocacheDir.UnsafeValue())
			var versionGocache string

			// Determine target GOOS/GOARCH for cache isolation
			goos := utils.GetEnvValue(execEnv, "GOOS")
			goarch := utils.GetEnvValue(execEnv, "GOARCH")
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
			cacheSuffix := buildCacheSuffix(goBinaryPath, goos, goarch, execEnv)

			if customGocacheDir != "" {
				// Use custom GOCACHE directory if specified
				versionGocache = filepath.Join(customGocacheDir, currentVersion, cacheSuffix)
			} else {
				// Use GOENV_ROOT/versions/{version}/go-build-{GOOS}-{GOARCH} as default GOCACHE
				versionGocache = filepath.Join(versionPath, cacheSuffix)
			}
			execEnv = setEnvVar(execEnv, utils.EnvVarGocache, versionGocache)

			// Write build.info file for CGO builds (diagnostic data)
			// Non-blocking - failures don't affect execution
			if cgo.IsCGOEnabled(execEnv) {
				buildInfo := cgo.GetBuildInfo(execEnv)
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
			execEnv = setEnvVar(execEnv, utils.EnvVarGomodcache, versionGomodcache)
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
		r := resolver.New(cfg, env)
		var err error
		commandPath, err = r.ResolveBinary(command, currentVersion, source)
		if err != nil {
			return fmt.Errorf("goenv: %s: command not found", command)
		}
	} else {
		// For system version, use PATH lookup but exclude shims directory
		// to prevent infinite recursion (shim → goenv exec → shim → ...)
		shimsDir := cfg.ShimsDir()
		originalPath := os.Getenv("PATH")
		filteredPath := pathutil.FilterFromPath(originalPath, shimsDir)
		os.Setenv("PATH", filteredPath)
		var err error
		commandPath, err = exec.LookPath(command)
		os.Setenv("PATH", originalPath)
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
	// Set recursion guard so shims don't re-enter goenv exec.
	// Use our PID as the value so the child can detect if it's being called recursively.
	execEnv = setEnvVar(execEnv, recursionEnvVar, fmt.Sprintf("%d", os.Getpid()))
	execCmd := exec.Command(commandPath, commandArgs...)
	execCmd.Env = execEnv
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
	if err == nil && shouldAutoRehash(command, commandArgs) && !env.HasNoAutoRehash() {
		if cfg.Debug {
			fmt.Fprintln(cmd.OutOrStdout(), "Debug: Auto-rehashing after go install")
		}
		// Run rehash silently - don't fail if it errors
		_ = runRehashSilent(cfg, env)
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
func runRehashSilent(cfg *config.Config, env *utils.GoenvEnvironment) error {
	shimMgr := shims.NewShimManager(cfg, env)
	return shimMgr.Rehash()
}

// findEnvVar finds an environment variable in the env slice using case-insensitive matching on Windows.
// Returns the index and the actual key name (with original casing) if found, or -1 and empty string if not found.
// On Unix systems, matching is case-sensitive.
func findEnvVar(env []string, key string) (int, string) {
	for i, envVar := range env {
		// Find the equals sign to separate key from value
		eqIdx := strings.Index(envVar, "=")
		if eqIdx == -1 {
			continue
		}
		actualKey := envVar[:eqIdx]

		// On Windows, environment variables are case-insensitive
		if utils.IsWindows() {
			if strings.EqualFold(actualKey, key) {
				return i, actualKey
			}
		} else {
			// On Unix, case-sensitive match
			if actualKey == key {
				return i, actualKey
			}
		}
	}
	return -1, ""
}

// setEnvVar sets or updates an environment variable.
// On Windows, this handles case-insensitive environment variable names to prevent
// creating duplicate entries (e.g., both "Path=" and "PATH=").
func setEnvVar(env []string, key, value string) []string {
	if idx, actualKey := findEnvVar(env, key); idx != -1 {
		// Update existing variable, preserving original key casing
		env[idx] = actualKey + "=" + value
		return env
	}
	// Variable not found, add it with the requested key
	return append(env, key+"="+value)
}

// buildCacheSuffix constructs a cache directory suffix that includes ABI variants.
// This is a wrapper around cache.BuildCacheSuffix for backward compatibility.
func buildCacheSuffix(goBinaryPath, goos, goarch string, env []string) string {
	return cache.BuildCacheSuffix(goBinaryPath, goos, goarch, env)
}

// prependToPath prepends a directory to the PATH environment variable.
// On Windows, this handles case-insensitive PATH matching to prevent creating duplicate
// PATH entries (e.g., both "Path=" and "PATH="), which would cause the original system
// PATH to be lost.
func prependToPath(env []string, dir string) []string {
	if idx, actualKey := findEnvVar(env, "PATH"); idx != -1 {
		// Found PATH, extract current value and prepend new directory
		currentPath := env[idx][len(actualKey)+1:] // +1 for the "=" sign

		// Detect separator from existing PATH to handle cross-platform scenarios
		// (e.g., testing Unix-style paths on Windows)
		sep := string(os.PathListSeparator) // default to OS separator
		if strings.Contains(currentPath, ";") {
			// Semicolon found - Windows-style separator
			sep = ";"
		} else {
			// Check for colons, but exclude Windows drive letter colons (e.g., "C:")
			colonCount := strings.Count(currentPath, ":")
			if len(currentPath) >= 2 && currentPath[1] == ':' {
				// Has a drive letter at the start, subtract it from the count
				colonCount--
			}
			if colonCount > 0 {
				// Has colons that are path separators (Unix-style)
				sep = ":"
			}
			// Otherwise, use OS default (already set)
		}

		newPath := dir + sep + currentPath
		// Preserve original key casing (e.g., "Path" on Windows, "PATH" on Unix)
		env[idx] = actualKey + "=" + newPath
		return env
	}
	// PATH not found, add it
	return append(env, "PATH="+dir)
}
