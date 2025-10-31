package diagnostics

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/binarycheck"
	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/envdetect"
	"github.com/go-nv/goenv/internal/goenv"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/shims"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/vscode"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Diagnose goenv installation and configuration issues",
	GroupID: string(cmdpkg.GroupGettingStarted),
	Long: `Checks your goenv installation and configuration for common issues.

This command verifies:
  - Runtime environment (containers, WSL, native)
  - Filesystem type (NFS, SMB, FUSE, local)
  - goenv binary and paths
  - Shell configuration files
  - Shell environment variables (GOENV_SHELL, GOENV_ROOT)
  - PATH setup and order
  - Shims directory
  - Installed Go versions
  - Cache isolation (version and architecture)
  - GOTOOLCHAIN environment variable
  - Rosetta detection (macOS)
  - System libc compatibility (Linux)
  - macOS deployment target (macOS)
  - Windows compiler availability (Windows)
  - Windows ARM64/ARM64EC support (Windows)
  - Linux kernel version (Linux)
  - Common configuration problems

Use this command to troubleshoot issues with goenv.

Interactive Fix Mode:
  Use --fix to interactively fix detected issues:
    - Missing shell configuration (goenv init not sourced)
    - Duplicate goenv installations
    - Duplicate shell profile entries
    - Stale cache files

  Example: goenv doctor --fix

Exit codes (for CI/automation):
  0 = No issues found (or only warnings when --fail-on=error)
  1 = Errors found
  2 = Warnings found (when --fail-on=warning)

Flags:
  --json              Output results in JSON format for CI/automation
  --fail-on           Exit with non-zero status on 'error' (default) or 'warning'
  --fix               Interactively fix detected issues (shell config, duplicates, stale cache)
  --non-interactive   Disable all interactive prompts (for CI/automation)`,
	RunE: runDoctor,
}

// Status type for check results
type Status string

const (
	StatusOK      Status = "ok"
	StatusWarning Status = "warning"
	StatusError   Status = "error"
)

// FailOn represents the severity level at which doctor should exit with non-zero status
type FailOn string

const (
	FailOnError   FailOn = "error"
	FailOnWarning FailOn = "warning"
)

// Issue type constants for structured fix detection
type IssueType string

const (
	IssueTypeNone                IssueType = ""
	IssueTypeShimsMissing        IssueType = "shims-missing"
	IssueTypeShimsEmpty          IssueType = "shims-empty"
	IssueTypeCacheStale          IssueType = "cache-stale"
	IssueTypeCacheArchMismatch   IssueType = "cache-arch-mismatch"
	IssueTypeVersionNotSet       IssueType = "version-not-set"
	IssueTypeVersionNotInstalled IssueType = "version-not-installed"
	IssueTypeVersionCorrupted    IssueType = "version-corrupted"
	IssueTypeVersionMismatch     IssueType = "version-mismatch"
	IssueTypeNoVersionsInstalled IssueType = "no-versions-installed"
	IssueTypeGoModMismatch       IssueType = "gomod-mismatch"
	IssueTypeVSCodeMissing       IssueType = "vscode-missing"
	IssueTypeVSCodeMismatch      IssueType = "vscode-mismatch"
	IssueTypeToolsMissing        IssueType = "tools-missing"
	IssueTypeMultipleInstalls    IssueType = "multiple-installs"
	IssueTypeShellNotConfigured  IssueType = "shell-not-configured"
	IssueTypeProfileDuplicates   IssueType = "profile-duplicates"
)

type checkResult struct {
	id        string // Machine-readable identifier for CI/automation
	name      string
	status    Status // OK, Warning, or Error
	message   string
	advice    string
	issueType IssueType   // Structured issue type for fix detection
	fixData   interface{} // Additional data needed for fixing (version, path, etc)
}

// JSON-serializable version of checkResult
func (c checkResult) MarshalJSON() ([]byte, error) {
	return json.Marshal(struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Status  string `json:"status"`
		Message string `json:"message"`
		Advice  string `json:"advice,omitempty"`
	}{
		ID:      c.id,
		Name:    c.name,
		Status:  string(c.status),
		Message: c.message,
		Advice:  c.advice,
	})
}

var (
	doctorJSON           bool
	doctorFailOnStr      string // Raw string from flag
	doctorFailOn         FailOn // Parsed enum value
	doctorFix            bool
	doctorNonInteractive bool
	// doctorExit is a function variable that can be overridden in tests
	doctorExit = os.Exit
	// doctorStdin can be overridden in tests
	doctorStdin io.Reader = os.Stdin
)

func init() {
	cmdpkg.RootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results in JSON format")
	doctorCmd.Flags().StringVar(&doctorFailOnStr, "fail-on", "error", "Exit with non-zero status on 'error' or 'warning' (for CI strictness)")
	doctorCmd.Flags().BoolVar(&doctorFix, "fix", false, "Interactively fix detected issues (shell config, duplicates, stale cache)")
	doctorCmd.Flags().BoolVar(&doctorNonInteractive, "non-interactive", false, "Disable all interactive prompts")
	helptext.SetCommandHelp(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	results := []checkResult{}

	// Validate and parse --fail-on flag
	switch doctorFailOnStr {
	case string(FailOnError):
		doctorFailOn = FailOnError
	case string(FailOnWarning):
		doctorFailOn = FailOnWarning
	default:
		return fmt.Errorf("invalid --fail-on value: %s (must be 'error' or 'warning')", doctorFailOnStr)
	}

	// Only show progress message in human-readable mode
	if !doctorJSON {
		fmt.Fprintf(cmd.OutOrStdout(), "%sChecking goenv installation...\n", utils.Emoji("🔍 "))
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Check 0: Environment detection (containers, WSL, filesystems)
	results = append(results, checkEnvironment(cfg))

	// Check 1: goenv binary location
	results = append(results, checkGoenvBinary(cfg))

	// Check 2: GOENV_ROOT
	results = append(results, checkGoenvRoot(cfg))

	// Check 2a: GOENV_ROOT filesystem
	results = append(results, checkGoenvRootFilesystem(cfg))

	// Check 3: Shell configuration
	results = append(results, checkShellConfig(cfg))

	// Check 3a: Shell environment (runtime)
	results = append(results, checkShellEnvironment(cfg))

	// Check 3b: Profile sourcing issues (unsourced profiles, conflicting sources)
	results = append(results, checkProfileSourcingIssues(cfg))

	// Check 4: PATH configuration
	results = append(results, checkPath(cfg))

	// Check 5: Shims directory
	results = append(results, checkShimsDir(cfg))

	// Check 6: Installed versions
	results = append(results, checkInstalledVersions(cfg))

	// Check 7: Current version
	results = append(results, checkCurrentVersion(cfg))

	// Check 8: Conflicting installations
	results = append(results, checkConflictingGo(cfg))

	// Check 9: Cache files
	results = append(results, checkCacheFiles(cfg))

	// Check 10: Network connectivity (optional)
	results = append(results, checkNetwork())

	// Check 11: VS Code integration
	results = append(results, checkVSCodeIntegration(cfg))

	// Check 12: go.mod version compatibility
	results = append(results, checkGoModVersion(cfg))

	// Check 13: Verify 'which go' matches expected version
	results = append(results, checkWhichGo(cfg))

	// Check 14: Check for unmigrated tools when using a new version
	results = append(results, checkToolMigration(cfg))

	// Check 15: GOCACHE isolation
	results = append(results, checkGocacheIsolation(cfg))

	// Check 16: Architecture mismatches in cache
	results = append(results, checkCacheArchitecture(cfg))

	// Check 16a: Cache on problem mounts (NFS, Docker bind mounts)
	results = append(results, checkCacheMountType(cfg))

	// Check 17: GOTOOLCHAIN setting
	results = append(results, checkGoToolchain())

	// Check 18: Cache isolation effectiveness (architecture-aware)
	results = append(results, checkCacheIsolationEffectiveness(cfg))

	// Check 19: Rosetta detection (macOS only)
	results = append(results, checkRosetta(cfg))

	// Check 20: PATH order (goenv shims before system Go)
	results = append(results, checkPathOrder(cfg))

	// Check 21: System libc compatibility (Linux only)
	if runtime.GOOS == "linux" {
		results = append(results, checkLibcCompatibility(cfg))
	}

	// Check 22: macOS deployment target (macOS only)
	if runtime.GOOS == "darwin" {
		results = append(results, checkMacOSDeploymentTarget(cfg))
	}

	// Check 23: Windows compiler availability (Windows only)
	if utils.IsWindows() {
		results = append(results, checkWindowsCompiler(cfg))
	}

	// Check 24: Windows ARM64/ARM64EC (Windows only)
	if utils.IsWindows() {
		results = append(results, checkWindowsARM64(cfg))
	}

	// Check 25: Linux kernel version (Linux only)
	if runtime.GOOS == "linux" {
		results = append(results, checkLinuxKernelVersion(cfg))
	}

	// Check 26: Multiple goenv installations
	results = append(results, checkMultipleInstallations())

	// Count results
	okCount := 0
	warningCount := 0
	errorCount := 0
	for _, result := range results {
		switch result.status {
		case StatusOK:
			okCount++
		case StatusWarning:
			warningCount++
		case StatusError:
			errorCount++
		}
	}

	// Handle --fix flag: unified interactive fix mode
	if doctorFix && !doctorJSON && !doctorNonInteractive {
		return runFixMode(cmd, results, cfg)
	}

	// Output results (JSON or human-readable)
	if doctorJSON {
		type jsonOutput struct {
			SchemaVersion string        `json:"schema_version"`
			Checks        []checkResult `json:"checks"`
			Summary       struct {
				Total    int `json:"total"`
				OK       int `json:"ok"`
				Warnings int `json:"warnings"`
				Errors   int `json:"errors"`
			} `json:"summary"`
		}

		output := jsonOutput{
			SchemaVersion: "1",
			Checks:        results,
		}
		output.Summary.Total = len(results)
		output.Summary.OK = okCount
		output.Summary.Warnings = warningCount
		output.Summary.Errors = errorCount

		encoder := json.NewEncoder(cmd.OutOrStdout())
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(output); err != nil {
			return fmt.Errorf("failed to encode JSON: %w", err)
		}

		// Check if we should fail based on --fail-on flag
		// Exit codes for CI clarity:
		//   0 = success (no issues or only warnings when --fail-on=error)
		//   1 = errors found
		//   2 = warnings found (when --fail-on=warning)
		if errorCount > 0 {
			doctorExit(1) // Errors always exit with code 1
		} else if doctorFailOn == FailOnWarning && warningCount > 0 {
			doctorExit(2) // Warnings exit with code 2 when --fail-on=warning
		}
		// On success (no issues or warnings with --fail-on=error), return normally (exit code 0)
		return nil
	}

	// Human-readable output
	fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", utils.Emoji("📋 "), utils.BoldBlue("Diagnostic Results:"))
	fmt.Fprintln(cmd.OutOrStdout())

	for _, result := range results {
		var icon, colorName string
		switch result.status {
		case StatusOK:
			icon = utils.Emoji("✅ ")
			colorName = utils.Green(result.name)
		case StatusWarning:
			icon = utils.Emoji("⚠️  ")
			colorName = utils.Yellow(result.name)
		case StatusError:
			icon = utils.Emoji("❌ ")
			colorName = utils.Red(result.name)
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", icon, colorName)
		if result.message != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", utils.Gray(result.message))
		}
		if result.advice != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   %s%s\n", utils.Emoji("💡 "), utils.Cyan(result.advice))
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout(), utils.Gray("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"))
	fmt.Fprintf(cmd.OutOrStdout(), "Summary: %s OK, %s warnings, %s errors\n",
		utils.Green(fmt.Sprintf("%d", okCount)),
		utils.Yellow(fmt.Sprintf("%d", warningCount)),
		utils.Red(fmt.Sprintf("%d", errorCount)))

	// Offer interactive shell environment fix (only in interactive mode, not CI)
	if !doctorNonInteractive && isInteractive() {
		offerShellEnvironmentFix(cmd, results, cfg)
	}

	// Exit codes for CI clarity:
	//   0 = success (no issues or only warnings when --fail-on=error)
	//   1 = errors found
	//   2 = warnings found (when --fail-on=warning)
	if errorCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s%s\n", utils.Emoji("❌ "), utils.Red("Issues found. Please review the errors above."))
		doctorExit(1) // Errors always exit with code 1
	} else if warningCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s%s\n", utils.Emoji("⚠️  "), utils.Yellow("Everything works, but some warnings should be reviewed."))
		// Check if we should fail on warnings based on --fail-on flag
		if doctorFailOn == FailOnWarning {
			doctorExit(2) // Warnings exit with code 2 when --fail-on=warning
		}
		// On success with warnings but --fail-on=error, return normally (exit code 0)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%s%s\n", utils.Emoji("✅ "), utils.Green("Everything looks good!"))
		// On success with no issues, return normally (exit code 0)
	}

	return nil
}

func checkEnvironment(cfg *config.Config) checkResult {
	// Detect runtime environment
	envInfo := envdetect.Detect()

	// Build status message
	message := fmt.Sprintf("Running on %s", envInfo.String())

	// Determine status based on warnings
	status := StatusOK
	advice := ""

	if envInfo.IsProblematicEnvironment() {
		warnings := envInfo.GetWarnings()
		if len(warnings) > 0 {
			status = StatusWarning
			// Show first warning in message, rest in advice
			message = fmt.Sprintf("%s - %s", message, warnings[0])
			if len(warnings) > 1 {
				advice = strings.Join(warnings[1:], "\n   ")
			}
		}
	}

	return checkResult{
		id:      "runtime-environment",
		name:    "Runtime environment",
		status:  status,
		message: message,
		advice:  advice,
	}
}

func checkGoenvRootFilesystem(cfg *config.Config) checkResult {
	// Detect filesystem type for GOENV_ROOT
	envInfo := envdetect.DetectFilesystem(cfg.Root)

	message := fmt.Sprintf("Filesystem type: %s", envInfo.FilesystemType)

	// Determine status
	status := StatusOK
	advice := ""

	switch envInfo.FilesystemType {
	case envdetect.FSTypeNFS:
		status = StatusWarning
		advice = "NFS filesystems can cause file locking issues and slow I/O. Consider using a local filesystem for GOENV_ROOT."
	case envdetect.FSTypeSMB:
		status = StatusWarning
		advice = "SMB/CIFS filesystems may have issues with symbolic links and permissions. Consider using a local filesystem for GOENV_ROOT."
	case envdetect.FSTypeBind:
		status = StatusWarning
		advice = "Bind mounts in containers should be persistent and have correct permissions."
	case envdetect.FSTypeFUSE:
		status = StatusWarning
		advice = "FUSE filesystems may have performance issues. Consider using a local filesystem for better performance."
	case envdetect.FSTypeUnknown:
		status = StatusWarning
		message = "Filesystem type: unknown"
		advice = "Could not determine filesystem type. This may indicate an unusual configuration."
	}

	return checkResult{
		id:      "goenvroot-filesystem",
		name:    "GOENV_ROOT filesystem",
		status:  status,
		message: message,
		advice:  advice,
	}
}

func checkGoenvBinary(_ *config.Config) checkResult {
	// Find goenv binary
	goenvPath, err := os.Executable()
	if err != nil {
		return checkResult{
			id:      "goenv-binary",
			name:    "goenv binary",
			status:  StatusError,
			message: fmt.Sprintf("Cannot determine goenv binary location: %v", err),
			advice:  "Ensure goenv is properly installed",
		}
	}

	return checkResult{
		id:      "goenv-binary",
		name:    "goenv binary",
		status:  StatusOK,
		message: fmt.Sprintf("Found at: %s", goenvPath),
	}
}

func checkGoenvRoot(cfg *config.Config) checkResult {
	root := cfg.Root
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return checkResult{
			id:      "goenvroot-directory",
			name:    "GOENV_ROOT directory",
			status:  StatusError,
			message: fmt.Sprintf("Directory does not exist: %s", root),
			advice:  "Run 'goenv init' to create the directory structure",
		}
	}

	return checkResult{
		id:      "goenvroot-directory",
		name:    "GOENV_ROOT directory",
		status:  StatusOK,
		message: fmt.Sprintf("Set to: %s", root),
	}
}

func checkShellConfig(_ *config.Config) checkResult {
	shell := os.Getenv(utils.EnvVarShell)
	if shell == "" {
		return checkResult{
			id:      "shell-configuration",
			name:    "Shell configuration",
			status:  StatusWarning,
			message: "SHELL environment variable not set",
			advice:  "This is unusual. Check your shell configuration.",
		}
	}

	// Determine config file
	homeDir, _ := os.UserHomeDir()
	var configFiles []string

	shellName := filepath.Base(shell)
	switch shellName {
	case string(shellutil.ShellTypeBash):
		configFiles = []string{
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".profile"),
		}
	case string(shellutil.ShellTypeZsh):
		configFiles = []string{
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".zprofile"),
		}
	case string(shellutil.ShellTypeFish):
		configFiles = []string{
			filepath.Join(homeDir, ".config", "fish", "config.fish"),
		}
	default:
		return checkResult{
			id:      "shell-configuration",
			name:    "Shell configuration",
			status:  StatusWarning,
			message: fmt.Sprintf("Unknown shell: %s", shellName),
			advice:  "Manual configuration may be required",
		}
	}

	// Check if goenv init is in any config file
	found := false
	foundIn := ""
	for _, configFile := range configFiles {
		if data, err := os.ReadFile(configFile); err == nil {
			content := string(data)
			if strings.Contains(content, "goenv init") || strings.Contains(content, "GOENV_ROOT") {
				found = true
				foundIn = configFile
				break
			}
		}
	}

	if found {
		return checkResult{
			id:      "shell-configuration",
			name:    "Shell configuration",
			status:  StatusOK,
			message: fmt.Sprintf("goenv detected in %s", foundIn),
		}
	}

	return checkResult{
		id:      "shell-configuration",
		name:    "Shell configuration",
		status:  StatusWarning,
		message: "goenv init not found in shell config",
		advice:  fmt.Sprintf("Add 'eval \"$(goenv init -)\"' to your %s", configFiles[0]),
	}
}

func checkShellEnvironment(cfg *config.Config) checkResult {
	// Check if GOENV_SHELL is set (indicates goenv init has been evaluated)
	goenvShell := utils.GoenvEnvVarShell.UnsafeValue()

	// Check GOENV_ROOT
	goenvRoot := utils.GoenvEnvVarRoot.UnsafeValue()

	// Detect current shell
	currentShell := shellutil.DetectShell()

	// Check if goenv shell function exists (for bash/zsh/ksh)
	hasShellFunction := checkGoenvShellFunction(currentShell)

	// Check for common "undo sourcing" scenarios
	undoScenario := detectUndoSourcing(cfg, currentShell, goenvShell, goenvRoot, hasShellFunction)
	if undoScenario != "" {
		return checkResult{
			id:        "shell-environment",
			name:      "Shell environment",
			status:    StatusError,
			message:   undoScenario,
			advice:    generateUndoSourcingFix(currentShell),
			issueType: IssueTypeShellNotConfigured,
		}
	}

	// Both missing - goenv init not evaluated
	if goenvShell == "" && goenvRoot == "" {
		// Check if function exists but env vars don't - might be un-sourced
		if hasShellFunction {
			return checkResult{
				id:        "shell-environment",
				name:      "Shell environment",
				status:    StatusError,
				message:   "goenv shell function exists but GOENV_SHELL not set - environment may have been reset or unsourced",
				advice:    "Run 'eval \"$(goenv init -)\"' to re-activate goenv in your current shell, or check if another profile is overriding your PATH/environment",
				issueType: IssueTypeShellNotConfigured,
			}
		}

		return checkResult{
			id:        "shell-environment",
			name:      "Shell environment",
			status:    StatusError,
			message:   "GOENV_SHELL and GOENV_ROOT not set - goenv init has not been evaluated in current shell",
			advice:    "Run 'eval \"$(goenv init -)\"' to activate goenv in your current shell, or restart your shell after adding it to your profile",
			issueType: IssueTypeShellNotConfigured,
		}
	}

	// GOENV_SHELL missing but GOENV_ROOT set - partial setup
	if goenvShell == "" {
		return checkResult{
			id:        "shell-environment",
			name:      "Shell environment",
			status:    StatusWarning,
			message:   "GOENV_SHELL not set but GOENV_ROOT is - incomplete shell integration",
			advice:    "Run 'eval \"$(goenv init -)\"' to complete goenv shell integration",
			issueType: IssueTypeShellNotConfigured,
		}
	}

	// Check if GOENV_ROOT matches expected (in case of stale shell or config mismatch)
	if goenvRoot != cfg.Root {
		return checkResult{
			id:        "shell-environment",
			name:      "Shell environment",
			status:    StatusWarning,
			message:   fmt.Sprintf("GOENV_ROOT mismatch: shell has '%s' but config expects '%s'", goenvRoot, cfg.Root),
			advice:    "Your shell environment may be outdated. Run 'eval \"$(goenv init -)\"' or restart your shell",
			issueType: IssueTypeShellNotConfigured,
		}
	}

	// Check if GOENV_SHELL matches current shell
	if goenvShell != string(currentShell) && currentShell != "" {
		return checkResult{
			id:        "shell-environment",
			name:      "Shell environment",
			status:    StatusWarning,
			message:   fmt.Sprintf("GOENV_SHELL mismatch: set to '%s' but running in '%s' shell", goenvShell, currentShell),
			advice:    fmt.Sprintf("You may have switched shells. Run 'eval \"$(goenv init -)\"' to reinitialize for %s", currentShell),
			issueType: IssueTypeShellNotConfigured,
		}
	}

	// Check if shell function exists when it should (bash/zsh/ksh only)
	// Only check if we're reasonably sure we're in a user's interactive shell
	// Skip in test/subprocess environments where function detection is unreliable
	if currentShell == shellutil.ShellTypeBash || currentShell == shellutil.ShellTypeZsh || currentShell == shellutil.ShellTypeKsh {
		// Only check if SHLVL > 1 (indicates real shell, not subprocess)
		// and if the shell binary exists
		shlvl := os.Getenv(utils.EnvVarShlvl)
		if shlvl != "" && shlvl != "0" && shlvl != "1" {
			if _, err := exec.LookPath(string(currentShell)); err == nil {
				// Shell binary exists, we can check for the function
				if !hasShellFunction {
					return checkResult{
						id:        "shell-environment",
						name:      "Shell environment",
						status:    StatusWarning,
						message:   "GOENV_SHELL is set but goenv shell function is missing - may have been unset or profile re-sourced incorrectly",
						advice:    "Run 'eval \"$(goenv init -)\"' to restore the goenv shell function",
						issueType: IssueTypeShellNotConfigured,
					}
				}
			}
		}
	}

	// Check if PATH still has shims (could be reset/modified)
	// Only check this if shims directory actually exists - in test environments
	// the temporary directory may not have shims set up
	pathEnv := os.Getenv(utils.EnvVarPath)
	shimsDir := cfg.ShimsDir()
	if pathEnv != "" {
		// Only perform this check if shims directory exists
		if _, err := os.Stat(shimsDir); err == nil {
			// Shims dir exists, check if it's in PATH (case-insensitive on Windows)
			if !utils.IsPathInPATH(shimsDir, pathEnv) {
				return checkResult{
					id:        "shell-environment",
					name:      "Shell environment",
					status:    StatusError,
					message:   "GOENV_SHELL is set but shims directory not in PATH - environment may have been reset",
					advice:    "Your PATH was modified or reset. Run 'eval \"$(goenv init -)\"' to restore goenv's PATH configuration",
					issueType: IssueTypeShellNotConfigured,
				}
			}
		}
	}

	// All good
	return checkResult{
		id:      "shell-environment",
		name:    "Shell environment",
		status:  StatusOK,
		message: fmt.Sprintf("Shell integration active (shell: %s)", goenvShell),
	}
}

// detectUndoSourcing detects if the user has "undone" their goenv sourcing
// by running 'source ~/.profile' or similar commands that reset the environment.
// Returns a descriptive error message if detected, empty string otherwise.
func detectUndoSourcing(cfg *config.Config, currentShell shellutil.ShellType, goenvShell, goenvRoot string, hasFunction bool) string {
	pathEnv := os.Getenv(utils.EnvVarPath)
	shimsDir := cfg.ShimsDir()

	// NEW CHECK 1: GOENV_SHELL set but shims not in PATH
	// This is THE key "undo sourcing" scenario - re-source profile that resets PATH
	// This check is NOT covered by existing logic which only looks at env var presence
	if goenvShell != "" && pathEnv != "" {
		if _, err := os.Stat(shimsDir); err == nil {
			if !utils.IsPathInPATH(shimsDir, pathEnv) {
				return "GOENV_SHELL is set but shims directory not in PATH - likely caused by re-sourcing a profile that resets PATH without goenv init (e.g., 'source ~/.bashrc' or 'source ~/.zshrc')"
			}
		}
	}

	// NEW CHECK 2: Profile file has goenv init but shell is not initialized
	// Catches cases where profile configuration is correct but environment was manually reset
	homeDir, _ := os.UserHomeDir()
	var profileFile string
	switch currentShell {
	case shellutil.ShellTypeBash:
		// Check both .bashrc and .bash_profile
		bashrc := filepath.Join(homeDir, ".bashrc")
		bashProfile := filepath.Join(homeDir, ".bash_profile")
		if _, err := os.Stat(bashProfile); err == nil {
			profileFile = bashProfile
		} else {
			profileFile = bashrc
		}
	case shellutil.ShellTypeZsh:
		profileFile = filepath.Join(homeDir, ".zshrc")
	case shellutil.ShellTypeFish:
		profileFile = filepath.Join(homeDir, ".config", "fish", "config.fish")
	}

	if profileFile != "" {
		if data, err := os.ReadFile(profileFile); err == nil {
			content := string(data)
			hasGoenvInit := strings.Contains(content, "goenv init")

			// Profile has goenv init but shell not initialized - environment was reset
			if hasGoenvInit && goenvShell == "" {
				return fmt.Sprintf("Profile file %s contains 'goenv init' but shell is not initialized - possible manual unsourcing or environment reset", filepath.Base(profileFile))
			}

			// Manually initialized but function missing - partial reset
			if !hasGoenvInit && goenvShell != "" && !hasFunction {
				return fmt.Sprintf("goenv initialized manually (not in %s) but shell function is missing - environment may have been partially reset", filepath.Base(profileFile))
			}
		}
	}

	// NEW CHECK 3: Environment variables set but goenv command doesn't work
	// Final validation that the environment is truly functional, not just "looks good"
	// Skip this check in test environments (when GOENV_ROOT points to a temp directory)
	// or when versions directory doesn't exist (indicates incomplete setup)
	if goenvShell != "" && goenvRoot != "" {
		versionsDir := filepath.Join(goenvRoot, "versions")
		// Only run this check if versions directory exists (indicates real goenv setup)
		// This prevents false positives in test environments with fake goenv executables
		if _, err := os.Stat(versionsDir); err == nil {
			cmd := exec.Command("goenv", "version-name")
			cmd.Env = os.Environ()
			if err := cmd.Run(); err != nil {
				return "Shell environment variables are set but 'goenv' command fails - possible PATH override or broken shell function"
			}
		}
	}

	return ""
}

// generateUndoSourcingFix generates shell-specific instructions to fix undo sourcing issues
func generateUndoSourcingFix(shell shellutil.ShellType) string {
	var fix strings.Builder

	fix.WriteString("To fix this issue:\n\n")

	switch shell {
	case shellutil.ShellTypeBash:
		fix.WriteString("1. Re-initialize goenv in your current shell:\n")
		fix.WriteString("   eval \"$(goenv init -)\"\n\n")
		fix.WriteString("2. To prevent this in the future, ensure ~/.bashrc or ~/.bash_profile contains:\n")
		fix.WriteString("   eval \"$(goenv init -)\"\n\n")
		fix.WriteString("3. Avoid running 'source ~/.bashrc' if it resets PATH. Use 'exec bash' to restart the shell instead\n\n")
		fix.WriteString("4. If you need to reload your profile, run:\n")
		fix.WriteString("   source ~/.bashrc && eval \"$(goenv init -)\"")

	case shellutil.ShellTypeZsh:
		fix.WriteString("1. Re-initialize goenv in your current shell:\n")
		fix.WriteString("   eval \"$(goenv init -)\"\n\n")
		fix.WriteString("2. To prevent this in the future, ensure ~/.zshrc contains:\n")
		fix.WriteString("   eval \"$(goenv init -)\"\n\n")
		fix.WriteString("3. Avoid running 'source ~/.zshrc' if it resets PATH. Use 'exec zsh' to restart the shell instead\n\n")
		fix.WriteString("4. If you need to reload your profile, run:\n")
		fix.WriteString("   source ~/.zshrc && eval \"$(goenv init -)\"")

	case shellutil.ShellTypeFish:
		fix.WriteString("1. Re-initialize goenv in your current shell:\n")
		fix.WriteString("   source (goenv init -|psub)\n\n")
		fix.WriteString("2. To prevent this in the future, ensure ~/.config/fish/config.fish contains:\n")
		fix.WriteString("   status --is-interactive; and source (goenv init -|psub)\n\n")
		fix.WriteString("3. If you need to reload your profile, run:\n")
		fix.WriteString("   source ~/.config/fish/config.fish")

	case shellutil.ShellTypePowerShell:
		fix.WriteString("1. Re-initialize goenv in your current shell:\n")
		fix.WriteString("   Invoke-Expression (goenv init - | Out-String)\n\n")
		fix.WriteString("2. To prevent this in the future, ensure your PowerShell profile contains:\n")
		fix.WriteString("   Invoke-Expression (goenv init - | Out-String)\n\n")
		fix.WriteString("3. If you need to reload your profile, run:\n")
		fix.WriteString("   . $PROFILE; Invoke-Expression (goenv init - | Out-String)")

	default:
		fix.WriteString("1. Re-initialize goenv in your current shell:\n")
		fix.WriteString("   eval \"$(goenv init -)\"\n\n")
		fix.WriteString("2. Ensure your shell profile contains:\n")
		fix.WriteString("   eval \"$(goenv init -)\"\n\n")
		fix.WriteString("3. Restart your shell or re-source your profile")
	}

	return fix.String()
}

// checkProfileSourcingIssues detects when profiles have been unsourced or incorrectly sourced
// This catches scenarios like:
// - Running `source ~/.bash_profile` which re-exports PATH without goenv
// - Profile files that reset PATH completely
// - Conflicting profile configurations
// - Profiles sourced in wrong order
func checkProfileSourcingIssues(cfg *config.Config) checkResult {
	// Only run this check if we have basic shell integration
	goenvShell := utils.GoenvEnvVarShell.UnsafeValue()
	if goenvShell == "" {
		// Shell not initialized at all - already caught by checkShellEnvironment
		return checkResult{
			id:      "profile-sourcing",
			name:    "Profile sourcing",
			status:  StatusOK,
			message: "Skipped (shell not initialized)",
		}
	}

	currentShell := shellutil.DetectShell()
	homeDir, _ := os.UserHomeDir()

	var issues []string
	var advice []string
	status := StatusOK

	// Check 1: Look for profile files that might reset PATH without including goenv
	var profileFiles []string
	switch currentShell {
	case shellutil.ShellTypeBash:
		profileFiles = []string{
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".profile"),
		}
	case shellutil.ShellTypeZsh:
		profileFiles = []string{
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".zprofile"),
			filepath.Join(homeDir, ".zshenv"),
		}
	case shellutil.ShellTypeFish:
		profileFiles = []string{
			filepath.Join(homeDir, ".config", "fish", "config.fish"),
		}
	}

	hasGoenvInit := false
	hasPathReset := false
	resetFile := ""
	goenvFile := ""

	for _, profileFile := range profileFiles {
		data, err := os.ReadFile(profileFile)
		if err != nil {
			continue
		}
		content := string(data)

		// Check if this file has goenv init
		if strings.Contains(content, "goenv init") {
			hasGoenvInit = true
			goenvFile = filepath.Base(profileFile)
		}

		// Check for patterns that might reset PATH
		// Common patterns that reset PATH:
		// - export PATH="/some/path"  (without $PATH)
		// - PATH="/some/path"
		// - export PATH=...  (without $PATH in the value)
		lines := strings.Split(content, "\n")
		for _, line := range lines {
			line = strings.TrimSpace(line)

			// Skip comments
			if strings.HasPrefix(line, "#") {
				continue
			}

			// Check for PATH reset patterns
			// Match: PATH="/something" or export PATH="/something" where something doesn't contain $PATH
			if (strings.Contains(line, "PATH=") || strings.Contains(line, "PATH =")) &&
				!strings.Contains(line, "goenv") {
				// Extract the value part
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					value := strings.TrimSpace(parts[1])
					// If PATH is set but doesn't reference previous PATH, it's a reset
					if !strings.Contains(value, "$PATH") &&
						!strings.Contains(value, "${PATH}") &&
						!strings.Contains(value, "goenv") {
						// This looks like a PATH reset
						hasPathReset = true
						resetFile = filepath.Base(profileFile)
						break
					}
				}
			}
		}
	}

	// Check 2: Detect if goenv init appears BEFORE path reset
	// This is a common issue where .bashrc has goenv but .bash_profile resets PATH
	if hasGoenvInit && hasPathReset && goenvFile != resetFile {
		issues = append(issues, fmt.Sprintf("PATH reset detected in %s after goenv init in %s", resetFile, goenvFile))
		advice = append(advice, fmt.Sprintf("Move goenv init in %s to appear AFTER the PATH reset in %s, or remove the PATH reset", goenvFile, resetFile))
		status = StatusWarning
	}

	// Check 3: Detect profiles that source other profiles after goenv
	// e.g., .bash_profile sources .bashrc, but .bashrc then resets PATH
	if hasGoenvInit {
		for _, profileFile := range profileFiles {
			data, err := os.ReadFile(profileFile)
			if err != nil {
				continue
			}
			content := string(data)

			// Check if file sources another file AFTER goenv init
			goenvIdx := strings.Index(content, "goenv init")
			if goenvIdx == -1 {
				continue
			}

			// Look for source/. commands after goenv init
			afterGoenv := content[goenvIdx:]
			if strings.Contains(afterGoenv, "source ") || strings.Contains(afterGoenv, ". ") {
				// Extract what's being sourced
				lines := strings.Split(afterGoenv, "\n")
				for _, line := range lines[1:] { // Skip the goenv init line
					line = strings.TrimSpace(line)
					if strings.HasPrefix(line, "source ") || strings.HasPrefix(line, ". ") {
						issues = append(issues, fmt.Sprintf("Profile %s sources another file after goenv init", filepath.Base(profileFile)))
						advice = append(advice, fmt.Sprintf("Ensure files sourced after goenv init don't reset PATH. Consider moving goenv init to the end of %s", filepath.Base(profileFile)))
						status = StatusWarning
						break
					}
				}
			}
		}
	}

	// Check 4: Detect if user is in a subshell without goenv
	// Compare parent process environment
	ppid := os.Getppid()
	if ppid > 1 {
		// Try to read parent's environment
		// This is Linux-specific but won't hurt on other systems
		parentEnvFile := fmt.Sprintf("/proc/%d/environ", ppid)
		if data, err := os.ReadFile(parentEnvFile); err == nil {
			parentEnv := string(data)
			// Check if parent had GOENV_SHELL but we're in a subshell that doesn't
			if !strings.Contains(parentEnv, "GOENV_SHELL=") && goenvShell != "" {
				// This is fine - we initialized in this shell
			} else if strings.Contains(parentEnv, "GOENV_SHELL=") && goenvShell == "" {
				issues = append(issues, "Parent shell has goenv but current shell doesn't - possible subshell without re-init")
				advice = append(advice, "Run 'eval \"$(goenv init -)\"' in this shell, or ensure subshells inherit goenv configuration")
				status = StatusWarning
			}
		}
	}

	// Check 5: Verify the shell function is actually functional
	// Try to get goenv's help output via the function
	if currentShell == shellutil.ShellTypeBash || currentShell == shellutil.ShellTypeZsh {
		shimsDir := cfg.ShimsDir()
		pathEnv := os.Getenv(utils.EnvVarPath)

		// If shims are in PATH but function doesn't work, it's been reset
		if utils.IsPathInPATH(shimsDir, pathEnv) {
			// Try to execute goenv via shell function
			cmd := exec.Command(string(currentShell), "-c", "goenv --version 2>&1")
			output, err := cmd.CombinedOutput()
			if err != nil {
				issues = append(issues, "goenv command fails despite shims in PATH - shell function may be broken")
				advice = append(advice, "Run 'eval \"$(goenv init -)\"' to restore the shell function")
				status = StatusError
			} else if !strings.Contains(string(output), "goenv") {
				issues = append(issues, "goenv command returns unexpected output - shell function may be misconfigured")
				advice = append(advice, "Run 'eval \"$(goenv init -)\"' to fix the shell function")
				status = StatusWarning
			}
		}
	}

	// Summarize results
	if len(issues) == 0 {
		return checkResult{
			id:      "profile-sourcing",
			name:    "Profile sourcing",
			status:  StatusOK,
			message: "No profile sourcing issues detected",
		}
	}

	message := strings.Join(issues, "; ")
	adviceStr := strings.Join(advice, "\n")

	return checkResult{
		id:        "profile-sourcing",
		name:      "Profile sourcing",
		status:    status,
		message:   message,
		advice:    adviceStr,
		issueType: IssueTypeProfileDuplicates,
	}
}

// InstallationType enum for goenv installation types
type InstallationType string

const (
	InstallTypeHomebrewArm   InstallationType = "homebrew-arm"
	InstallTypeHomebrewIntel InstallationType = "homebrew-intel"
	InstallTypeHomebrewLinux InstallationType = "homebrew-linux"
	InstallTypeManual        InstallationType = "manual"
	InstallTypeSystem        InstallationType = "system"
	InstallTypeScoop         InstallationType = "scoop"
	InstallTypeChocolatey    InstallationType = "chocolatey"
	InstallTypeUnknown       InstallationType = "unknown"
)

// Architecture represents CPU architecture types
type Architecture string

const (
	ArchARM64   Architecture = "arm64"
	ArchAMD64   Architecture = "amd64"
	ArchUnknown Architecture = "unknown"
)

// installationType represents the type of goenv installation
type installationType struct {
	path         string
	installType  InstallationType
	architecture Architecture
	recommended  bool // whether this installation is recommended to keep
}

// checkMultipleInstallations detects if multiple goenv installations exist
// Multiple installations can cause confusion and conflicts
func checkMultipleInstallations() checkResult {
	installations := detectAllGoenvInstallations()

	if len(installations) == 0 {
		// This shouldn't happen if doctor is running, but handle it
		return checkResult{
			id:      "multiple-installations",
			name:    "Multiple installations",
			status:  StatusOK,
			message: "No goenv installations found (running from source?)",
		}
	}

	if len(installations) == 1 {
		return checkResult{
			id:      "multiple-installations",
			name:    "Multiple installations",
			status:  StatusOK,
			message: fmt.Sprintf("Single installation: %s", installations[0]),
		}
	}

	// Multiple installations found - classify them
	classified := classifyInstallations(installations)

	// Generate recommendation
	recommendation := generateCleanupRecommendation(classified)

	// Build display list
	installList := ""
	for i, inst := range classified {
		if i > 0 {
			installList += "\n     "
		}
		marker := ""
		if inst.recommended {
			marker = " [KEEP]"
		} else {
			marker = " [can remove]"
		}
		installList += fmt.Sprintf("%d. %s (%s)%s", i+1, inst.path, inst.installType, marker)
	}

	advice := fmt.Sprintf("Multiple installations can cause conflicts. %s\n   Run 'goenv doctor --fix' to interactively remove duplicates.", recommendation)

	return checkResult{
		id:        "multiple-installations",
		name:      "Multiple installations",
		status:    StatusWarning,
		message:   fmt.Sprintf("Found %d goenv installations:\n     %s", len(installations), installList),
		advice:    advice,
		issueType: IssueTypeMultipleInstalls,
	}
}

// detectAllGoenvInstallations finds all goenv binaries on the system
func detectAllGoenvInstallations() []string {
	var found []string
	seen := make(map[string]bool)

	// Method 1: Check PATH
	if path, err := exec.LookPath("goenv"); err == nil {
		resolved := path
		if r, err := filepath.EvalSymlinks(path); err == nil {
			resolved = r
		}
		if !seen[resolved] {
			found = append(found, resolved)
			seen[resolved] = true
		}
	}

	// Build list of common locations
	homeDir, _ := os.UserHomeDir()
	locations := []string{}

	// Method 2: Homebrew locations
	if runtime.GOOS == "darwin" || runtime.GOOS == "linux" {
		locations = append(locations,
			"/opt/homebrew/bin/goenv",              // ARM Mac
			"/usr/local/bin/goenv",                 // Intel Mac / Linux Homebrew
			"/home/linuxbrew/.linuxbrew/bin/goenv", // Linux Homebrew
		)
	}

	// Method 3: Manual installation
	locations = append(locations, filepath.Join(homeDir, ".goenv", "bin", "goenv"))

	// Method 4: System locations (Unix)
	if !utils.IsWindows() {
		locations = append(locations,
			"/usr/bin/goenv",
			"/usr/local/bin/goenv",
			"/opt/goenv/bin/goenv",
		)
	}

	// Method 5: Windows locations
	if utils.IsWindows() {
		locations = append(locations,
			filepath.Join(homeDir, "bin", "goenv.exe"),
			filepath.Join(homeDir, ".goenv", "bin", "goenv.exe"),
			"C:\\Program Files\\goenv\\goenv.exe",
			"C:\\goenv\\bin\\goenv.exe",
		)

		// Check scoop
		if scoopPath := os.Getenv("SCOOP"); scoopPath != "" {
			locations = append(locations, filepath.Join(scoopPath, "shims", "goenv.exe"))
		}

		// Check chocolatey
		if programData := os.Getenv(utils.EnvVarProgramData); programData != "" {
			locations = append(locations, filepath.Join(programData, "chocolatey", "bin", "goenv.exe"))
		}
	}

	// Check all locations
	for _, loc := range locations {
		if stat, err := os.Stat(loc); err == nil && !stat.IsDir() {
			// Resolve symlinks
			resolved := loc
			if r, err := filepath.EvalSymlinks(loc); err == nil {
				resolved = r
			}

			if !seen[resolved] {
				found = append(found, resolved)
				seen[resolved] = true
			}
		}
	}

	return found
}

// classifyInstallations determines the type of each installation
func classifyInstallations(paths []string) []installationType {
	classified := make([]installationType, 0, len(paths))

	for _, path := range paths {
		inst := installationType{
			path:         path,
			installType:  "unknown",
			architecture: ArchUnknown,
			recommended:  false,
		}

		// Determine installation type
		if strings.Contains(path, "/opt/homebrew/") {
			inst.installType = InstallTypeHomebrewArm
			inst.architecture = ArchARM64
		} else if strings.Contains(path, "/usr/local/") && strings.Contains(path, "Cellar/goenv") {
			inst.installType = InstallTypeHomebrewIntel
			inst.architecture = ArchAMD64
		} else if strings.Contains(path, "/home/linuxbrew/") || strings.Contains(path, "linuxbrew") {
			inst.installType = InstallTypeHomebrewLinux
		} else if strings.Contains(path, "/.goenv/bin/") {
			inst.installType = InstallTypeManual
		} else if strings.HasPrefix(path, "/usr/bin/") || strings.HasPrefix(path, "/usr/local/bin/") {
			inst.installType = InstallTypeSystem
		} else if strings.Contains(path, "scoop") {
			inst.installType = InstallTypeScoop
		} else if strings.Contains(path, "chocolatey") {
			inst.installType = InstallTypeChocolatey
		}

		classified = append(classified, inst)
	}

	// Determine recommendations based on platform and installation types
	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		// On M1/M2 Mac, recommend ARM homebrew or manual, remove Intel homebrew
		for i := range classified {
			if classified[i].installType == InstallTypeHomebrewArm || classified[i].installType == InstallTypeManual {
				classified[i].recommended = true
				break // Only keep one
			}
		}
	} else if runtime.GOOS == "darwin" && runtime.GOARCH == "amd64" {
		// On Intel Mac, recommend Intel homebrew or manual
		for i := range classified {
			if classified[i].installType == InstallTypeHomebrewIntel || classified[i].installType == InstallTypeManual {
				classified[i].recommended = true
				break
			}
		}
	} else {
		// On other platforms, recommend manual over system, or homebrew over manual
		for i := range classified {
			if classified[i].installType == InstallTypeHomebrewLinux || classified[i].installType == InstallTypeManual {
				classified[i].recommended = true
				break
			}
		}
	}

	// If nothing was marked as recommended (edge case), recommend the first one in PATH
	hasRecommended := false
	for _, inst := range classified {
		if inst.recommended {
			hasRecommended = true
			break
		}
	}
	if !hasRecommended && len(classified) > 0 {
		classified[0].recommended = true
	}

	return classified
}

// generateCleanupRecommendation generates human-readable advice
func generateCleanupRecommendation(installations []installationType) string {
	if len(installations) <= 1 {
		return ""
	}

	if runtime.GOOS == "darwin" && runtime.GOARCH == "arm64" {
		// Check if there's an Intel homebrew installation
		for _, inst := range installations {
			if inst.installType == InstallTypeHomebrewIntel {
				return "On Apple Silicon, remove the Intel Homebrew installation."
			}
		}
	}

	// Count types
	homebrewCount := 0
	manualCount := 0
	systemCount := 0

	for _, inst := range installations {
		if strings.Contains(string(inst.installType), "homebrew") {
			homebrewCount++
		} else if inst.installType == InstallTypeManual {
			manualCount++
		} else if inst.installType == InstallTypeSystem {
			systemCount++
		}
	}

	if homebrewCount > 0 && manualCount > 0 {
		return "Consider keeping only Homebrew installation for easier updates."
	}

	if homebrewCount > 1 {
		return "Multiple Homebrew installations found. Keep only one."
	}

	return "Keep the installation that's first in your PATH."
}

// runFixMode provides unified interactive fixing for all detected issues
func runFixMode(cmd *cobra.Command, results []checkResult, cfg *config.Config) error {
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "%sInteractive Fix Mode\n", utils.Emoji("🔧 "))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())

	// Detect all fixable issues from results
	issues := detectFixableIssues(results, cfg)

	if len(issues) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sNo fixable issues detected.\n", utils.Emoji("✅ "))
		return nil
	}

	// Organize by tier
	autoFixes := []fixableIssue{}
	promptFixes := []fixableIssue{}
	manualFixes := []fixableIssue{}

	for _, issue := range issues {
		switch issue.tier {
		case FixTierAuto:
			autoFixes = append(autoFixes, issue)
		case FixTierPrompt:
			promptFixes = append(promptFixes, issue)
		case FixTierManual:
			manualFixes = append(manualFixes, issue)
		}
	}

	// Show summary of detected issues
	fmt.Fprintf(cmd.OutOrStdout(), "Detected %d fixable issue(s):\n\n", len(issues))
	issueNum := 1
	for _, issue := range issues {
		icon := "🔧"
		if issue.tier == FixTierAuto {
			icon = "⚡"
		} else if issue.tier == FixTierManual {
			icon = "📝"
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  %d. %s %s\n", issueNum, icon, issue.name)
		fmt.Fprintf(cmd.OutOrStdout(), "     %s\n\n", issue.description)
		issueNum++
	}

	fixedCount := 0

	// Tier 1: Auto-run fixes (no prompt)
	if len(autoFixes) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sAuto-fixing safe issues...\n\n", utils.Emoji("⚡ "))

		for _, issue := range autoFixes {
			fmt.Fprintf(cmd.OutOrStdout(), "  %s %s...", utils.Emoji("⚡"), issue.name)
			err := issue.fixFunc(cmd, cfg)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), " %s\n", utils.Emoji("❌"))
				fmt.Fprintf(cmd.OutOrStdout(), "    Error: %v\n", err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), " %s\n", utils.Emoji("✅"))
				fixedCount++
			}
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Tier 2: Prompted fixes
	reader := bufio.NewReader(doctorStdin)
	if len(promptFixes) > 0 {
		for _, issue := range promptFixes {
			fmt.Fprintln(cmd.OutOrStdout(), "──────────────────────────────────────────────────")
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", utils.Emoji("🔧 "), issue.name)
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", issue.description)
			fmt.Fprintf(cmd.OutOrStdout(), "Would you like to fix this? [Y/n]: ")

			response, err := reader.ReadString('\n')
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "%sSkipping (input error)\n\n", utils.Emoji("⏭️  "))
				continue
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "" && response != "y" && response != "yes" {
				fmt.Fprintf(cmd.OutOrStdout(), "%sSkipped\n\n", utils.Emoji("⏭️  "))
				continue
			}

			fmt.Fprintln(cmd.OutOrStdout())
			err = issue.fixFunc(cmd, cfg)
			if err != nil {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sError: %v\n\n", utils.Emoji("❌ "), err)
			} else {
				fmt.Fprintf(cmd.OutOrStdout(), "\n%sFixed successfully\n\n", utils.Emoji("✅ "))
				fixedCount++
			}
		}
	}

	// Tier 3: Manual fixes (show instructions)
	if len(manualFixes) > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "──────────────────────────────────────────────────")
		fmt.Fprintf(cmd.OutOrStdout(), "%sManual fixes required:\n\n", utils.Emoji("📝 "))

		for _, issue := range manualFixes {
			fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", utils.Emoji("📝 "), issue.name)
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", issue.description)

			// Execute the "fix" function which just shows instructions
			issue.fixFunc(cmd, cfg)
			fmt.Fprintln(cmd.OutOrStdout())
		}

		// Pause so user can read and copy the manual fix commands
		utils.PauseForUser(cmd.OutOrStdout(), reader)
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	if fixedCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sFixed %d issue(s)! Run 'goenv doctor' to verify.\n", utils.Emoji("✨ "), fixedCount)
	} else if len(manualFixes) > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sPlease apply the manual fixes above.\n", utils.Emoji("📝 "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%sNo issues were fixed.\n", utils.Emoji("ℹ️  "))
	}

	return nil
}

// detectFixableIssues analyzes check results and creates fixable issues
func detectFixableIssues(results []checkResult, cfg *config.Config) []fixableIssue {
	var issues []fixableIssue

	for _, result := range results {
		if result.status == StatusOK {
			continue
		}

		// Use structured issueType instead of string matching
		switch result.issueType {
		// Auto-fix tier: Safe operations that don't need confirmation
		case IssueTypeShimsMissing, IssueTypeShimsEmpty:
			issues = append(issues, fixableIssue{
				id:          "rehash",
				name:        "Missing Shims",
				description: "Shims directory missing or empty",
				tier:        FixTierAuto,
				fixFunc:     fixRehash,
			})

		case IssueTypeCacheStale, IssueTypeCacheArchMismatch:
			issues = append(issues, fixableIssue{
				id:          "cache-clean",
				name:        "Stale Build Cache",
				description: "Old or incompatible build cache detected",
				tier:        FixTierAuto,
				fixFunc:     fixCacheClean,
			})

		case IssueTypeVersionMismatch:
			// Version mismatch that needs rehash
			issues = append(issues, fixableIssue{
				id:          "rehash-mismatch",
				name:        "Rehash After Mismatch",
				description: "Go binary version mismatch detected",
				tier:        FixTierAuto,
				fixFunc:     fixRehash,
			})

		// Prompt tier: Operations that need user confirmation
		case IssueTypeVersionNotInstalled:
			version, ok := result.fixData.(string)
			if ok && version != "" {
				issues = append(issues, fixableIssue{
					id:          "install-missing-version",
					name:        fmt.Sprintf("Install Missing Go %s", version),
					description: fmt.Sprintf("Current version %s is set but not installed", version),
					tier:        FixTierPrompt,
					fixFunc: func(cmd *cobra.Command, cfg *config.Config) error {
						return fixInstallMissingVersion(cmd, cfg, version)
					},
				})
			}

		case IssueTypeVersionCorrupted:
			version, ok := result.fixData.(string)
			if ok && version != "" {
				issues = append(issues, fixableIssue{
					id:          "reinstall-corrupted",
					name:        fmt.Sprintf("Reinstall Corrupted Go %s", version),
					description: fmt.Sprintf("Go %s installation is corrupted", version),
					tier:        FixTierPrompt,
					fixFunc: func(cmd *cobra.Command, cfg *config.Config) error {
						return fixReinstallCorrupted(cmd, cfg, version)
					},
				})
			}

		case IssueTypeVersionNotSet:
			issues = append(issues, fixableIssue{
				id:          "set-version",
				name:        "Set Go Version",
				description: "No Go version is currently set",
				tier:        FixTierPrompt,
				fixFunc:     fixSetVersion,
			})

		case IssueTypeNoVersionsInstalled:
			issues = append(issues, fixableIssue{
				id:          "install-latest",
				name:        "Install Latest Go",
				description: "No Go versions are installed",
				tier:        FixTierPrompt,
				fixFunc:     fixInstallLatest,
			})

		case IssueTypeGoModMismatch:
			version, ok := result.fixData.(string)
			if ok && version != "" {
				issues = append(issues, fixableIssue{
					id:          "fix-gomod-version",
					name:        fmt.Sprintf("Install Go %s for go.mod", version),
					description: fmt.Sprintf("go.mod requires Go %s", version),
					tier:        FixTierPrompt,
					fixFunc: func(cmd *cobra.Command, cfg *config.Config) error {
						return fixGoModVersion(cmd, cfg, version)
					},
				})
			}

		case IssueTypeVSCodeMissing:
			issues = append(issues, fixableIssue{
				id:          "vscode-init",
				name:        "Initialize VS Code",
				description: "VS Code is missing goenv configuration",
				tier:        FixTierPrompt,
				fixFunc:     fixVSCodeInit,
			})

		case IssueTypeVSCodeMismatch:
			issues = append(issues, fixableIssue{
				id:          "vscode-sync",
				name:        "Sync VS Code Settings",
				description: "VS Code settings don't match current Go version",
				tier:        FixTierPrompt,
				fixFunc:     fixVSCodeSync,
			})

		case IssueTypeToolsMissing:
			issues = append(issues, fixableIssue{
				id:          "tool-sync",
				name:        "Sync Go Tools",
				description: "Current Go version has no tools installed",
				tier:        FixTierPrompt,
				fixFunc:     fixToolSync,
			})

		case IssueTypeMultipleInstalls:
			issues = append(issues, fixableIssue{
				id:          "cleanup-duplicates",
				name:        "Remove Duplicate Installations",
				description: "Multiple goenv installations detected",
				tier:        FixTierPrompt,
				fixFunc:     fixDuplicateInstallations,
			})

		// Manual tier: Show instructions only
		case IssueTypeShellNotConfigured:
			issues = append(issues, fixableIssue{
				id:          "shell-init",
				name:        "Shell Configuration",
				description: "Shell environment needs configuration",
				tier:        FixTierManual,
				fixFunc:     fixShellEnvironment,
			})

		case IssueTypeProfileDuplicates:
			issues = append(issues, fixableIssue{
				id:          "cleanup-shell-profiles",
				name:        "Clean Up Shell Profiles",
				description: "Duplicate goenv entries in shell profiles",
				tier:        FixTierManual,
				fixFunc:     fixShellProfiles,
			})
		}
	}

	// Deduplicate issues by ID
	seen := make(map[string]bool)
	unique := []fixableIssue{}
	for _, issue := range issues {
		if !seen[issue.id] {
			unique = append(unique, issue)
			seen[issue.id] = true
		}
	}

	return unique
}

// Fix helper functions

func fixRehash(cmd *cobra.Command, cfg *config.Config) error {
	shimMgr := shims.NewShimManager(cfg)
	return shimMgr.Rehash()
}

func fixCacheClean(cmd *cobra.Command, cfg *config.Config) error {
	buildCache := os.Getenv("GOCACHE")
	if buildCache == "" {
		buildCache = filepath.Join(cfg.Root, "go-build")
	}
	return os.RemoveAll(buildCache)
}

func fixInstallMissingVersion(cmd *cobra.Command, cfg *config.Config, version string) error {
	// For now, show instructions - actual install would need to import install package
	fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv install %s\n", version)
	return fmt.Errorf("please run the command manually")
}

func fixReinstallCorrupted(cmd *cobra.Command, cfg *config.Config, version string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv uninstall %s && goenv install %s\n", version, version)
	return fmt.Errorf("please run the commands manually")
}

func fixSetVersion(cmd *cobra.Command, cfg *config.Config) error {
	mgr := manager.NewManager(cfg)
	versions, err := mgr.ListInstalledVersions()
	if err != nil || len(versions) == 0 {
		return fmt.Errorf("no versions installed")
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv global %s\n", versions[0])
	return fmt.Errorf("please run the command manually")
}

func fixInstallLatest(cmd *cobra.Command, cfg *config.Config) error {
	fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv install\n")
	return fmt.Errorf("please run the command manually")
}

func fixGoModVersion(cmd *cobra.Command, cfg *config.Config, version string) error {
	mgr := manager.NewManager(cfg)
	if mgr.IsVersionInstalled(version) {
		fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv local %s\n", version)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv install %s && goenv local %s\n", version, version)
	}
	return fmt.Errorf("please run the command manually")
}

func fixVSCodeInit(cmd *cobra.Command, cfg *config.Config) error {
	fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv vscode init\n")
	return fmt.Errorf("please run the command manually")
}

func fixVSCodeSync(cmd *cobra.Command, cfg *config.Config) error {
	fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv vscode sync\n")
	return fmt.Errorf("please run the command manually")
}

func fixToolSync(cmd *cobra.Command, cfg *config.Config) error {
	fmt.Fprintf(cmd.OutOrStdout(), "  Run: goenv tools sync\n")
	return fmt.Errorf("please run the command manually")
}

func fixDuplicateInstallations(cmd *cobra.Command, cfg *config.Config) error {
	return cleanupDuplicateInstallations(cmd)
}

func fixShellEnvironment(cmd *cobra.Command, cfg *config.Config) error {
	shell := shellutil.DetectShell()
	initCommand := shellutil.GetInitLine(shell)
	profilePath := shellutil.GetProfilePathDisplay(shell)

	fmt.Fprintf(cmd.OutOrStdout(), "  Run this command in your current shell:\n")
	fmt.Fprintf(cmd.OutOrStdout(), "    %s\n\n", initCommand)
	fmt.Fprintf(cmd.OutOrStdout(), "  To make it permanent, add to %s:\n", profilePath)
	fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", initCommand)
	return nil
}

func fixShellProfiles(cmd *cobra.Command, cfg *config.Config) error {
	return cleanupShellProfiles(cmd, cfg)
}

// FixTier type for categorizing fix operations
type FixTier string

const (
	FixTierAuto   FixTier = "auto"   // Auto-run without prompt
	FixTierPrompt FixTier = "prompt" // Prompt user before running
	FixTierManual FixTier = "manual" // Show instructions, don't auto-run
)

// fixableIssue represents an issue that can be automatically or interactively fixed
type fixableIssue struct {
	id          string
	name        string
	description string
	tier        FixTier
	fixFunc     func(*cobra.Command, *config.Config) error
}

// cleanupDuplicateInstallations removes duplicate goenv installations
func cleanupDuplicateInstallations(cmd *cobra.Command) error {
	// Detect all installations
	installations := detectAllGoenvInstallations()

	if len(installations) <= 1 {
		fmt.Printf("%sNo duplicates found.\n", utils.Emoji("✅ "))
		return nil
	}

	// Classify installations
	classified := classifyInstallations(installations)

	// Display installations
	fmt.Printf("%sFound %d goenv installations:\n\n", utils.Emoji("🔍 "), len(classified))

	for i, inst := range classified {
		marker := ""
		color := ""
		if inst.recommended {
			marker = " [RECOMMENDED TO KEEP]"
			color = "\033[0;32m" // green
		} else {
			marker = " [can safely remove]"
			color = "\033[0;33m" // yellow
		}
		fmt.Printf("  %d. %s%s%s\n", i+1, color, inst.path, "\033[0m")
		fmt.Printf("     Type: %s\n", inst.installType)
		if inst.architecture != ArchUnknown {
			fmt.Printf("     Architecture: %s\n", inst.architecture)
		}
		fmt.Printf("     %s\n", marker)
		fmt.Println()
	}

	// Show recommendation
	recommendation := generateCleanupRecommendation(classified)
	if recommendation != "" {
		fmt.Printf("%sRecommendation: %s\n\n", utils.Emoji("💡 "), recommendation)
	}

	// Prompt for which to remove
	fmt.Println("Which installations would you like to remove?")
	fmt.Println("Enter numbers separated by spaces (e.g., '2 3'), or 'none' to cancel:")
	fmt.Print("> ")

	scanner := bufio.NewScanner(doctorStdin)
	if !scanner.Scan() {
		return fmt.Errorf("failed to read input")
	}

	input := strings.TrimSpace(scanner.Text())
	if input == "none" || input == "" {
		fmt.Println("\n" + utils.Emoji("❌ ") + "Cleanup cancelled.")
		return nil
	}

	// Parse selection
	toRemove := []int{}
	parts := strings.Fields(input)
	for _, part := range parts {
		var num int
		if _, err := fmt.Sscanf(part, "%d", &num); err == nil {
			if num >= 1 && num <= len(classified) {
				// Don't allow removing the recommended one
				if classified[num-1].recommended {
					fmt.Printf("\n%sWarning: Installation %d is recommended to keep. Skipping.\n", utils.Emoji("⚠️  "), num)
					continue
				}
				toRemove = append(toRemove, num-1)
			}
		}
	}

	if len(toRemove) == 0 {
		fmt.Println("\n" + utils.Emoji("❌ ") + "No valid selections. Cleanup cancelled.")
		return nil
	}

	// Confirm removal
	fmt.Printf("\n%sYou are about to remove %d installation(s):\n", utils.Emoji("⚠️  "), len(toRemove))
	for _, idx := range toRemove {
		fmt.Printf("  - %s\n", classified[idx].path)
	}
	fmt.Print("\nProceed? (yes/no): ")

	if !scanner.Scan() {
		return fmt.Errorf("failed to read confirmation")
	}

	confirmation := strings.ToLower(strings.TrimSpace(scanner.Text()))
	if confirmation != "yes" && confirmation != "y" {
		fmt.Println("\n" + utils.Emoji("❌ ") + "Cleanup cancelled.")
		return nil
	}

	// Remove installations
	fmt.Println()
	for _, idx := range toRemove {
		path := classified[idx].path
		fmt.Printf("%sRemoving: %s\n", utils.Emoji("🗑️  "), path)

		if err := os.Remove(path); err != nil {
			fmt.Printf("%sFailed to remove: %v\n", utils.Emoji("❌ "), err)
			fmt.Printf("   You may need to run: sudo rm %s\n", path)
			continue
		}

		fmt.Printf("%sSuccessfully removed\n", utils.Emoji("✅ "))
	}

	fmt.Printf("%sRemoved %d duplicate installation(s)\n", utils.Emoji("✅ "), len(toRemove))

	return nil
}

// cleanupShellProfiles removes duplicate or stale goenv entries from shell profiles
func cleanupShellProfiles(cmd *cobra.Command, cfg *config.Config) error {
	homeDir, _ := os.UserHomeDir()
	profiles := map[string]string{
		".bashrc":       filepath.Join(homeDir, ".bashrc"),
		".bash_profile": filepath.Join(homeDir, ".bash_profile"),
		".zshrc":        filepath.Join(homeDir, ".zshrc"),
		".profile":      filepath.Join(homeDir, ".profile"),
	}

	var foundIssues []string

	for name, path := range profiles {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}

		content := string(data)
		lines := strings.Split(content, "\n")

		// Find goenv-related lines
		var goenvLines []int
		for i, line := range lines {
			if strings.Contains(line, "goenv init") || strings.Contains(line, "GOENV_ROOT") {
				goenvLines = append(goenvLines, i)
			}
		}

		if len(goenvLines) > 1 {
			foundIssues = append(foundIssues, fmt.Sprintf("%s has %d goenv entries", name, len(goenvLines)))
		}
	}

	if len(foundIssues) == 0 {
		fmt.Printf("%sNo duplicate profile entries found.\n", utils.Emoji("✅ "))
		return nil
	}

	fmt.Println("Found potential issues:")
	for _, issue := range foundIssues {
		fmt.Printf("  - %s\n", issue)
	}
	fmt.Println()
	fmt.Println(utils.Emoji("💡 ") + "Tip: Manually review your shell profile files to remove duplicates.")
	fmt.Println("Run 'goenv setup' to reconfigure your shell properly.")

	return nil
}

// checkGoenvShellFunction checks if the goenv shell function exists in the current shell
// This helps detect if the environment was un-sourced or reset
// Returns true if function exists, false if it doesn't or can't be checked
func checkGoenvShellFunction(shell shellutil.ShellType) bool {
	// Only check for shells that use the function
	if shell != shellutil.ShellTypeBash && shell != shellutil.ShellTypeZsh && shell != shellutil.ShellTypeKsh {
		return true // Not applicable for fish/pwsh/cmd - assume OK
	}

	// If GOENV_SHELL is not set, we shouldn't detect a function
	// This prevents false positives in test environments
	if utils.GoenvEnvVarShell.UnsafeValue() == "" {
		return false
	}

	// Check if BASH_FUNC_goenv%% or similar exists (more direct check)
	// This works when running IN the shell that has the function
	for _, key := range os.Environ() {
		// Bash stores functions as BASH_FUNC_name%%=...
		// zsh/ksh don't export functions this way, but we can try
		if strings.HasPrefix(key, "BASH_FUNC_goenv") {
			// Extract the value to check if it's actually set (not empty)
			parts := strings.SplitN(key, "=", 2)
			if len(parts) == 2 && parts[1] != "" {
				return true
			}
		}
	}

	// Fallback: Try to check if the function exists using shell-specific commands
	// This creates a subprocess so may not work in all cases
	var cmd *exec.Cmd
	switch shell {
	case shellutil.ShellTypeBash:
		// Check if goenv function exists in bash
		cmd = exec.Command(string(shell), "-c", "declare -F goenv >/dev/null 2>&1")
	case shellutil.ShellTypeZsh:
		// Check if goenv function exists in zsh
		cmd = exec.Command(string(shell), "-c", "whence -w goenv | grep -q function")
	case shellutil.ShellTypeKsh:
		// Check if goenv function exists in ksh
		cmd = exec.Command(string(shell), "-c", "typeset -f goenv >/dev/null 2>&1")
	default:
		return true // Unknown shell, assume OK
	}

	// If the command fails to run (shell not found, etc), assume OK
	// We don't want to false-positive in test or restricted environments
	err := cmd.Run()
	if err != nil {
		// Could be that shell isn't available, function doesn't exist, or other error
		// In a real doctor run, we'd already know the shell works (they're running goenv)
		// So only return false if we're confident function is missing
		if _, lookErr := exec.LookPath(string(shell)); lookErr != nil {
			// Shell binary doesn't exist, can't check - assume OK
			return true
		}
		// Shell exists but function check failed - function likely missing
		return false
	}
	return true
}

func checkPath(cfg *config.Config) checkResult {
	pathEnv := os.Getenv(utils.EnvVarPath)
	pathDirs := filepath.SplitList(pathEnv)

	goenvBin := filepath.Join(cfg.Root, "bin")
	shimsDir := cfg.ShimsDir()

	hasBin := false
	hasShims := false
	shimsPosition := -1

	for i, dir := range pathDirs {
		if dir == goenvBin {
			hasBin = true
		}
		if dir == shimsDir {
			hasShims = true
			shimsPosition = i
		}
	}

	if !hasBin {
		return checkResult{
			id:      "path-configuration",
			name:    "PATH configuration",
			status:  StatusError,
			message: fmt.Sprintf("%s not in PATH", goenvBin),
			advice:  fmt.Sprintf("Add 'export PATH=\"%s:$PATH\"' to your shell config", goenvBin),
		}
	}

	if !hasShims {
		return checkResult{
			id:      "path-configuration",
			name:    "PATH configuration",
			status:  StatusWarning,
			message: fmt.Sprintf("%s not in PATH", shimsDir),
			advice:  "Run 'eval \"$(goenv init -)\"' in your shell config",
		}
	}

	// Check if shims are early in PATH (should be near the front)
	if shimsPosition > 5 {
		return checkResult{
			id:      "path-configuration",
			name:    "PATH configuration",
			status:  StatusWarning,
			message: fmt.Sprintf("Shims directory is at position %d in PATH", shimsPosition),
			advice:  "Shims should be near the beginning of PATH for proper version switching",
		}
	}

	return checkResult{
		id:      "path-configuration",
		name:    "PATH configuration",
		status:  StatusOK,
		message: "goenv bin and shims directories are in PATH",
	}
}

func checkShimsDir(cfg *config.Config) checkResult {
	shimsDir := cfg.ShimsDir()

	stat, err := os.Stat(shimsDir)
	if os.IsNotExist(err) {
		return checkResult{
			id:        "shims-directory",
			name:      "Shims directory",
			status:    StatusWarning,
			message:   fmt.Sprintf("Shims directory does not exist: %s", shimsDir),
			advice:    "Run 'goenv rehash' to create shims",
			issueType: IssueTypeShimsMissing,
		}
	}
	if err != nil {
		return checkResult{
			id:      "shims-directory",
			name:    "Shims directory",
			status:  StatusError,
			message: fmt.Sprintf("Cannot access shims directory: %v", err),
			advice:  "Check file permissions",
		}
	}

	if !stat.IsDir() {
		return checkResult{
			id:      "shims-directory",
			name:    "Shims directory",
			status:  StatusError,
			message: fmt.Sprintf("Shims path exists but is not a directory: %s", shimsDir),
			advice:  "Remove the file and run 'goenv rehash'",
		}
	}

	// Count shims
	entries, err := os.ReadDir(shimsDir)
	if err != nil {
		return checkResult{
			id:      "shims-directory",
			name:    "Shims directory",
			status:  StatusWarning,
			message: fmt.Sprintf("Cannot read shims directory: %v", err),
		}
	}

	shimCount := len(entries)
	if shimCount == 0 {
		return checkResult{
			id:        "shims-directory",
			name:      "Shims directory",
			status:    StatusWarning,
			message:   "No shims found",
			advice:    "Run 'goenv rehash' to create shims",
			issueType: IssueTypeShimsEmpty,
		}
	}

	return checkResult{
		id:      "shims-directory",
		name:    "Shims directory",
		status:  StatusOK,
		message: fmt.Sprintf("Found %d shim(s)", shimCount),
	}
}

func checkInstalledVersions(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	versions, err := mgr.ListInstalledVersions()

	if err != nil {
		return checkResult{
			id:      "installed-go-versions",
			name:    "Installed Go versions",
			status:  StatusError,
			message: fmt.Sprintf("Cannot list versions: %v", err),
			advice:  "Check GOENV_ROOT and versions directory",
		}
	}

	if len(versions) == 0 {
		return checkResult{
			id:        "installed-go-versions",
			name:      "Installed Go versions",
			status:    StatusWarning,
			message:   "No Go versions installed",
			advice:    "Install a Go version with 'goenv install <version>'",
			issueType: IssueTypeNoVersionsInstalled,
		}
	}

	// Validate each installation for corruption
	var corruptedVersions []string
	var validVersions []string
	versionsDir := cfg.VersionsDir()

	for _, ver := range versions {
		goBinaryBase := filepath.Join(versionsDir, ver, "bin", "go")

		// Check if go binary exists (handles .exe and .bat on Windows)
		if _, err := pathutil.FindExecutable(goBinaryBase); err != nil {
			corruptedVersions = append(corruptedVersions, ver)
		} else {
			validVersions = append(validVersions, ver)
		}
	}

	if len(corruptedVersions) > 0 {
		return checkResult{
			id:        "installed-go-versions",
			name:      "Installed Go versions",
			status:    StatusError,
			message:   fmt.Sprintf("Found %d version(s), but %d are CORRUPTED: %s", len(versions), len(corruptedVersions), strings.Join(corruptedVersions, ", ")),
			advice:    fmt.Sprintf("Reinstall corrupted versions: goenv uninstall %s && goenv install %s", corruptedVersions[0], corruptedVersions[0]),
			issueType: IssueTypeVersionCorrupted,
			fixData:   corruptedVersions[0],
		}
	}

	return checkResult{
		id:      "installed-go-versions",
		name:    "Installed Go versions",
		status:  StatusOK,
		message: fmt.Sprintf("Found %d valid version(s): %s", len(validVersions), strings.Join(validVersions, ", ")),
	}
}

func checkCurrentVersion(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	version, source, err := mgr.GetCurrentVersion()

	if err != nil {
		return checkResult{
			id:        "current-go-version",
			name:      "Current Go version",
			status:    StatusWarning,
			message:   fmt.Sprintf("No version set: %v", err),
			advice:    "Set a version with 'goenv global <version>' or create a .go-version file",
			issueType: IssueTypeVersionNotSet,
		}
	}

	if version == "system" {
		return checkResult{
			id:      "current-go-version",
			name:    "Current Go version",
			status:  StatusOK,
			message: fmt.Sprintf("Using system Go (set by %s)", source),
		}
	}

	// Validate version is installed
	if err := mgr.ValidateVersion(version); err != nil {
		return checkResult{
			id:        "current-go-version",
			name:      "Current Go version",
			status:    StatusError,
			message:   fmt.Sprintf("Version '%s' is set but not installed (set by %s)", version, source),
			advice:    fmt.Sprintf("Install the version with 'goenv install %s'", version),
			issueType: IssueTypeVersionNotInstalled,
			fixData:   version,
		}
	}

	// Check if the installation is corrupted (missing go binary)
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	goBinaryBase := filepath.Join(versionPath, "bin", "go")

	// Check if go binary exists (handles .exe and .bat on Windows)
	if _, err := pathutil.FindExecutable(goBinaryBase); err != nil {
		return checkResult{
			id:        "current-go-version",
			name:      "Current Go version",
			status:    StatusError,
			message:   fmt.Sprintf("Version '%s' is CORRUPTED - go binary missing (set by %s)", version, source),
			advice:    fmt.Sprintf("Reinstall: goenv uninstall %s && goenv install %s", version, version),
			issueType: IssueTypeVersionCorrupted,
			fixData:   version,
		}
	}

	return checkResult{
		id:      "current-go-version",
		name:    "Current Go version",
		status:  StatusOK,
		message: fmt.Sprintf("%s (set by %s)", version, source),
	}
}

func checkConflictingGo(cfg *config.Config) checkResult {
	// Check for system Go installations that might conflict
	pathEnv := os.Getenv(utils.EnvVarPath)
	pathDirs := filepath.SplitList(pathEnv)
	shimsDir := cfg.ShimsDir()

	var systemGoLocations []string

	for _, dir := range pathDirs {
		// Skip goenv directories
		if strings.Contains(dir, cfg.Root) {
			continue
		}

		// Check for 'go' binary
		goBinary := filepath.Join(dir, "go")
		if utils.IsWindows() {
			goBinary += ".exe"
		}

		if _, err := os.Stat(goBinary); err == nil {
			systemGoLocations = append(systemGoLocations, dir)
		}
	}

	if len(systemGoLocations) == 0 {
		return checkResult{
			id:      "conflicting-go-installations",
			name:    "Conflicting Go installations",
			status:  StatusOK,
			message: "No system Go installations found that could conflict",
		}
	}

	// Check if shims come before system Go
	shimsFirst := false
	for _, dir := range pathDirs {
		if dir == shimsDir {
			shimsFirst = true
			break
		}
		if slices.Contains(systemGoLocations, dir) {
			shimsFirst = false
			break
		}
		if !shimsFirst {
			break
		}
	}

	if shimsFirst {
		return checkResult{
			id:      "conflicting-go-installations",
			name:    "Conflicting Go installations",
			status:  StatusOK,
			message: fmt.Sprintf("Found system Go at %s, but goenv shims have priority", strings.Join(systemGoLocations, ", ")),
		}
	}

	return checkResult{
		id:      "conflicting-go-installations",
		name:    "Conflicting Go installations",
		status:  StatusWarning,
		message: fmt.Sprintf("System Go at %s may take priority over goenv", strings.Join(systemGoLocations, ", ")),
		advice:  "Ensure goenv shims directory comes before system Go in PATH",
	}
}

func checkCacheFiles(cfg *config.Config) checkResult {
	// Cache files are stored in GOENV_ROOT, not a separate directory
	// Check for releases-cache.json and versions-cache.json
	releasesCache := filepath.Join(cfg.Root, "releases-cache.json")
	versionsCache := filepath.Join(cfg.Root, "versions-cache.json")

	var foundCaches []string
	if _, err := os.Stat(releasesCache); err == nil {
		foundCaches = append(foundCaches, "releases-cache.json")
	}
	if _, err := os.Stat(versionsCache); err == nil {
		foundCaches = append(foundCaches, "versions-cache.json")
	}

	if len(foundCaches) == 0 {
		return checkResult{
			id:      "cache-files",
			name:    "Cache files",
			status:  StatusOK,
			message: "No cache files (will be created when needed)",
		}
	}

	// Check if cache files are readable
	for _, cacheName := range foundCaches {
		cachePath := filepath.Join(cfg.Root, cacheName)
		if _, err := os.ReadFile(cachePath); err != nil {
			return checkResult{
				id:      "cache-files",
				name:    "Cache files",
				status:  StatusWarning,
				message: fmt.Sprintf("Cannot read %s: %v", cacheName, err),
				advice:  "Run 'goenv refresh cache' to regenerate cache files",
			}
		}
	}

	return checkResult{
		id:      "cache-files",
		name:    "Cache files",
		status:  StatusOK,
		message: fmt.Sprintf("Found %d cache file(s): %v", len(foundCaches), foundCaches),
	}
}

func checkNetwork() checkResult {
	// Use HTTPS HEAD request instead of ping (works in CI/containers where ICMP is blocked)
	// Use a small, fast endpoint with short timeout
	client := &http.Client{
		Timeout: 3 * time.Second,
		// Don't follow redirects - just need to confirm connectivity
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}

	// Try a HEAD request to golang.org (lightweight, no body download)
	req, err := http.NewRequest("HEAD", "https://go.dev", nil)
	if err != nil {
		return checkResult{
			id:      "network-connectivity",
			name:    "Network connectivity",
			status:  StatusWarning,
			message: "Failed to create network request",
			advice:  "This is unusual and may indicate a system configuration issue.",
		}
	}

	// Set a reasonable user agent
	req.Header.Set("User-Agent", "goenv-doctor/1.0")

	resp, err := client.Do(req)
	if err != nil {
		return checkResult{
			id:      "network-connectivity",
			name:    "Network connectivity",
			status:  StatusWarning,
			message: "Cannot reach go.dev",
			advice:  "You may not be able to fetch new Go versions. Check your internet connection and firewall settings.",
		}
	}
	defer resp.Body.Close()

	// Any 2xx or 3xx response indicates connectivity
	if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		return checkResult{
			id:      "network-connectivity",
			name:    "Network connectivity",
			status:  StatusOK,
			message: "Can reach go.dev",
		}
	}

	// Got a response but unexpected status code
	return checkResult{
		id:      "network-connectivity",
		name:    "Network connectivity",
		status:  StatusWarning,
		message: fmt.Sprintf("Unexpected response from go.dev (HTTP %d)", resp.StatusCode),
		advice:  "Network connectivity exists but may have issues. You should still be able to fetch Go versions.",
	}
}

func checkVSCodeIntegration(cfg *config.Config) checkResult {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// Can't check, but not critical
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  StatusOK,
			message: "Unable to check (not in a project directory)",
		}
	}

	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")

	// Check if .vscode directory exists
	if _, err := os.Stat(vscodeDir); os.IsNotExist(err) {
		// No .vscode directory - this is fine, just informational
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  StatusOK,
			message: "No .vscode directory found",
			advice:  "Run 'goenv vscode init' to set up VS Code integration with Go settings",
		}
	}

	// Check if settings.json exists
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		return checkResult{
			id:        "vs-code-integration",
			name:      "VS Code integration",
			status:    StatusWarning,
			message:   "Found .vscode directory but no settings.json",
			advice:    "Run 'goenv vscode init' to configure Go extension, or 'goenv vscode doctor' for detailed diagnostics",
			issueType: IssueTypeVSCodeMissing,
		}
	}

	// Get current Go version to validate against
	mgr := manager.NewManager(cfg)
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil || currentVersion == "" {
		// Can't determine current version - do basic check
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  StatusWarning,
			message: "Cannot determine current Go version for validation",
			advice:  "Set a Go version with 'goenv global' or 'goenv local' first",
		}
	}

	// Use sophisticated VS Code settings check
	result := vscode.CheckSettings(settingsFile, currentVersion)

	if !result.HasSettings {
		return checkResult{
			id:        "vs-code-integration",
			name:      "VS Code integration",
			status:    StatusWarning,
			message:   "settings.json exists but missing Go configuration",
			advice:    "Run 'goenv vscode init' to add goenv configuration, or 'goenv vscode doctor' for detailed diagnostics",
			issueType: IssueTypeVSCodeMissing,
		}
	}

	if result.UsesEnvVars {
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  StatusOK,
			message: "VS Code configured to use goenv environment variables (${env:GOROOT})",
		}
	}

	if result.Mismatch {
		return checkResult{
			id:        "vs-code-integration",
			name:      "VS Code integration",
			status:    StatusWarning,
			message:   fmt.Sprintf("VS Code settings use Go %s but current version is %s", result.ConfiguredVersion, currentVersion),
			advice:    "Run 'goenv vscode sync' to fix, or 'goenv vscode doctor' for detailed diagnostics",
			issueType: IssueTypeVSCodeMismatch,
		}
	}

	if result.ConfiguredVersion != "" {
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  StatusOK,
			message: fmt.Sprintf("VS Code configured with absolute path for Go %s", result.ConfiguredVersion),
		}
	}

	// Has go.goroot but couldn't parse version
	return checkResult{
		id:      "vs-code-integration",
		name:    "VS Code integration",
		status:  StatusWarning,
		message: "VS Code has Go configuration but cannot determine version",
		advice:  "Run 'goenv vscode init --force' to update settings, or 'goenv vscode doctor' for detailed diagnostics",
	}
}

func checkGoModVersion(cfg *config.Config) checkResult {
	cwd, _ := os.Getwd()
	gomodPath := filepath.Join(cwd, "go.mod")

	// Only check if go.mod exists
	if _, err := os.Stat(gomodPath); os.IsNotExist(err) {
		return checkResult{
			id:      "gomod-version",
			name:    "go.mod version",
			status:  StatusOK,
			message: "No go.mod file in current directory",
		}
	}

	// Get current Go version
	mgr := manager.NewManager(cfg)
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil {
		return checkResult{
			id:      "gomod-version",
			name:    "go.mod version",
			status:  StatusError,
			message: "Cannot determine current Go version",
			advice:  "Ensure a Go version is set with 'goenv global' or 'goenv local'",
		}
	}

	// Parse go.mod for required version
	requiredVersion, err := manager.ParseGoModVersion(gomodPath)
	if err != nil {
		return checkResult{
			id:      "gomod-version",
			name:    "go.mod version",
			status:  StatusWarning,
			message: fmt.Sprintf("Cannot parse go.mod: %v", err),
			advice:  "Ensure go.mod has a valid 'go' directive",
		}
	}

	// Compare versions
	if !manager.VersionSatisfies(currentVersion, requiredVersion) {
		// Check if required version is installed
		installedVersions, err := mgr.ListInstalledVersions()
		isInstalled := false
		if err == nil {
			for _, v := range installedVersions {
				if v == requiredVersion || "v"+v == requiredVersion || v == "v"+requiredVersion {
					isInstalled = true
					break
				}
			}
		}

		advice := fmt.Sprintf("Run: goenv local %s", requiredVersion)
		if !isInstalled {
			advice = fmt.Sprintf("Run: goenv install %s && goenv local %s", requiredVersion, requiredVersion)
		}

		return checkResult{
			id:        "gomod-version",
			name:      "go.mod version",
			status:    StatusError,
			message:   fmt.Sprintf("go.mod requires Go %s but current version is %s", requiredVersion, currentVersion),
			advice:    advice,
			issueType: IssueTypeGoModMismatch,
			fixData:   requiredVersion,
		}
	}

	return checkResult{
		id:      "gomod-version",
		name:    "go.mod version",
		status:  StatusOK,
		message: fmt.Sprintf("Current Go %s satisfies go.mod requirement (>= %s)", currentVersion, requiredVersion),
	}
}

func checkWhichGo(cfg *config.Config) checkResult {
	// Get what goenv thinks the version should be
	mgr := manager.NewManager(cfg)
	expectedVersion, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  StatusWarning,
			message: "Cannot determine expected Go version",
			advice:  "Set a Go version with 'goenv global' or 'goenv local'",
		}
	}

	// Find which 'go' binary is actually being used
	goPath, err := exec.LookPath("go")
	if err != nil {
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  StatusError,
			message: "No 'go' binary found in PATH",
			advice:  "Ensure goenv is properly initialized and a version is installed",
		}
	}

	// Get the actual version by running 'go version'
	cmd := exec.Command("go", "version")
	output, err := cmd.Output()
	if err != nil {
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  StatusError,
			message: fmt.Sprintf("Cannot determine actual Go version at %s", goPath),
			advice:  "Verify the Go installation is not corrupted",
		}
	}

	// Parse version from output (format: "go version go1.23.2 darwin/arm64")
	versionStr := string(output)
	parts := strings.Fields(versionStr)
	var actualVersion string
	if len(parts) >= 3 {
		actualVersion = strings.TrimPrefix(parts[2], "go")
	} else {
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  StatusWarning,
			message: fmt.Sprintf("Cannot parse 'go version' output: %s", versionStr),
		}
	}

	// Check if it's in the goenv shims directory
	shimsDir := cfg.ShimsDir()
	isUsingShim := strings.HasPrefix(goPath, shimsDir)

	// If expected version is "system", we just need to verify go works
	if expectedVersion == "system" {
		if isUsingShim {
			return checkResult{
				id:      "actual-go-binary",
				name:    "Actual 'go' binary",
				status:  StatusOK,
				message: fmt.Sprintf("Using system Go %s via goenv shim at %s", actualVersion, goPath),
			}
		}
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  StatusOK,
			message: fmt.Sprintf("Using system Go %s at %s (set by %s)", actualVersion, goPath, source),
		}
	}

	// Compare versions
	if actualVersion != expectedVersion {
		if isUsingShim {
			return checkResult{
				id:      "actual-go-binary",
				name:    "Actual 'go' binary",
				status:  StatusError,
				message: fmt.Sprintf("Version mismatch: expected %s (set by %s) but 'go version' reports %s", expectedVersion, source, actualVersion),
				advice:  "This may indicate a corrupted installation. Try: goenv rehash",
			}
		}

		// Not using shim - PATH issue
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  StatusError,
			message: fmt.Sprintf("Version mismatch: expected %s (set by %s) but using %s at %s", expectedVersion, source, actualVersion, goPath),
			advice:  "The 'go' binary at " + goPath + " is taking precedence. Ensure goenv shims directory (" + shimsDir + ") is first in your PATH. Run: eval \"$(goenv init -)\". If you see build cache errors, run: goenv cache clean build",
		}
	}

	// Versions match!
	if isUsingShim {
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  StatusOK,
			message: fmt.Sprintf("Correctly using Go %s via goenv shim", actualVersion),
		}
	}

	// Version is correct but not using shim - a bit unusual but not wrong
	return checkResult{
		id:      "actual-go-binary",
		name:    "Actual 'go' binary",
		status:  StatusOK,
		message: fmt.Sprintf("Using Go %s at %s (not via shim)", actualVersion, goPath),
	}
}

func checkToolMigration(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)

	// Get current version
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil || currentVersion == "" || currentVersion == "system" {
		// Can't check if no version is set or using system
		return checkResult{
			id:      "tool-migration",
			name:    "Tool migration",
			status:  StatusOK,
			message: "Not applicable (no managed version active)",
		}
	}

	// Get all installed versions
	installedVersions, err := mgr.ListInstalledVersions()
	if err != nil || len(installedVersions) <= 1 {
		// Can't check if we can't list versions or there's only one version
		return checkResult{
			id:      "tool-migration",
			name:    "Tool migration",
			status:  StatusOK,
			message: "Only one Go version installed",
		}
	}

	// Check for tools in current version
	currentTools, err := listToolsForVersion(cfg, currentVersion)
	if err != nil {
		return checkResult{
			id:      "tool-migration",
			name:    "Tool migration",
			status:  StatusOK,
			message: "Cannot detect installed tools",
		}
	}

	// If current version has tools, nothing to suggest
	if len(currentTools) > 0 {
		return checkResult{
			id:      "tool-migration",
			name:    "Tool migration",
			status:  StatusOK,
			message: fmt.Sprintf("Found %d tool(s) in current version", len(currentTools)),
		}
	}

	// Current version has no tools - check if other versions have tools
	versionsWithTools := []string{}
	maxToolCount := 0
	bestSourceVersion := ""

	for _, version := range installedVersions {
		if version == currentVersion {
			continue
		}

		tools, err := listToolsForVersion(cfg, version)
		if err != nil {
			continue
		}

		if len(tools) > 0 {
			versionsWithTools = append(versionsWithTools, version)
			if len(tools) > maxToolCount {
				maxToolCount = len(tools)
				bestSourceVersion = version
			}
		}
	}

	// If no other version has tools, all good
	if len(versionsWithTools) == 0 {
		return checkResult{
			id:      "tool-migration",
			name:    "Tool migration",
			status:  StatusOK,
			message: "No tools installed in any version",
		}
	}

	// Found tools in other versions but not current - suggest sync
	if len(versionsWithTools) == 1 {
		return checkResult{
			id:        "tool-sync",
			name:      "Tool sync",
			status:    StatusWarning,
			message:   fmt.Sprintf("Current Go %s has no tools, but Go %s has %d tool(s)", currentVersion, bestSourceVersion, maxToolCount),
			advice:    fmt.Sprintf("Sync tools with: goenv tools sync (or: goenv tools sync %s)", bestSourceVersion),
			issueType: IssueTypeToolsMissing,
		}
	}

	// Multiple versions have tools
	return checkResult{
		id:        "tool-sync",
		name:      "Tool sync",
		status:    StatusWarning,
		message:   fmt.Sprintf("Current Go %s has no tools, but %d other version(s) have tools (e.g., Go %s has %d tool(s))", currentVersion, len(versionsWithTools), bestSourceVersion, maxToolCount),
		advice:    fmt.Sprintf("Sync tools from best source: goenv tools sync (will auto-select Go %s)", bestSourceVersion),
		issueType: IssueTypeToolsMissing,
	}
}

func checkGocacheIsolation(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	version, _, err := mgr.GetCurrentVersion()
	if err != nil || version == "" {
		return checkResult{
			id:      "build-cache-isolation",
			name:    "Build cache isolation",
			status:  StatusOK,
			message: "Not applicable (no version set)",
		}
	}

	if version == "system" {
		return checkResult{
			id:      "build-cache-isolation",
			name:    "Build cache isolation",
			status:  StatusOK,
			message: "Not applicable (using system Go)",
		}
	}

	// Check if GOCACHE isolation is disabled
	if utils.GoenvEnvVarDisableGocache.IsTrue() {
		return checkResult{
			id:      "build-cache-isolation",
			name:    "Build cache isolation",
			status:  StatusOK,
			message: "Cache isolation disabled by GOENV_DISABLE_GOCACHE",
		}
	}

	// Get expected GOCACHE path
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	customGocacheDir := utils.GoenvEnvVarGocacheDir.UnsafeValue()
	var expectedGocache string
	if customGocacheDir != "" {
		expectedGocache = filepath.Join(customGocacheDir, version)
	} else {
		expectedGocache = filepath.Join(versionPath, "go-build")
	}

	// Check what GOCACHE would be set to when running commands
	// Note: We can't rely on current env var since exec.go sets it
	return checkResult{
		id:      "build-cache-isolation",
		name:    "Build cache isolation",
		status:  StatusOK,
		message: fmt.Sprintf("Version-specific cache: %s", expectedGocache),
		advice:  "Cache isolation prevents 'exec format error' when switching versions",
	}
}

func checkCacheArchitecture(cfg *config.Config) checkResult {
	// Detect current architecture
	currentArch := runtime.GOARCH
	currentOS := runtime.GOOS

	// Try to get GOCACHE location
	cmd := exec.Command("go", "env", "GOCACHE")
	output, err := cmd.Output()
	var gocache string
	if err == nil {
		gocache = strings.TrimSpace(string(output))
	} else {
		// Fallback to environment variable
		gocache = os.Getenv("GOCACHE")
	}

	if gocache == "" {
		return checkResult{
			id:      "cache-architecture",
			name:    "Cache architecture",
			status:  StatusOK,
			message: "Cannot determine GOCACHE location",
		}
	}

	// Check if cache directory exists
	stat, err := os.Stat(gocache)
	if err != nil || !stat.IsDir() {
		return checkResult{
			id:      "cache-architecture",
			name:    "Cache architecture",
			status:  StatusOK,
			message: "Build cache is empty or doesn't exist yet",
		}
	}

	// Check if it's a version-specific cache (contains GOENV_ROOT path)
	isVersionSpecific := strings.Contains(gocache, cfg.Root)

	if isVersionSpecific {
		return checkResult{
			id:      "cache-architecture",
			name:    "Cache architecture",
			status:  StatusOK,
			message: fmt.Sprintf("Using version-specific cache for %s/%s", currentOS, currentArch),
		}
	}

	return checkResult{
		id:        "cache-architecture",
		name:      "Cache architecture",
		status:    StatusWarning,
		message:   fmt.Sprintf("Using shared system cache at %s for %s/%s", gocache, currentOS, currentArch),
		advice:    "If you see 'exec format error', run: goenv cache clean build",
		issueType: IssueTypeCacheArchMismatch,
	}
}

// Helper to list tools for a version without importing tooldetect (to avoid circular deps)
func listToolsForVersion(cfg *config.Config, version string) ([]string, error) {
	gopathBin := filepath.Join(cfg.VersionsDir(), version, "gopath", "bin")

	// Check if directory exists
	if _, err := os.Stat(gopathBin); os.IsNotExist(err) {
		return []string{}, nil
	}

	// Read directory
	entries, err := os.ReadDir(gopathBin)
	if err != nil {
		return nil, err
	}

	// Filter for executables
	var tools []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		// Remove Windows executable extensions for deduplication
		if utils.IsWindows() {
			for _, ext := range utils.WindowsExecutableExtensions() {
				name = strings.TrimSuffix(name, ext)
			}
		}

		// Skip common non-tool files
		if name == ".DS_Store" {
			continue
		}

		tools = append(tools, name)
	}

	return tools, nil
}

func checkCacheMountType(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	version, _, err := mgr.GetCurrentVersion()
	if err != nil || version == "" {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  StatusOK,
			message: "Not applicable (no version set)",
		}
	}

	if version == "system" {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  StatusOK,
			message: "Not applicable (using system Go)",
		}
	}

	// Check if GOCACHE isolation is disabled
	if utils.GoenvEnvVarDisableGocache.IsTrue() {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  StatusOK,
			message: "Cache isolation disabled by GOENV_DISABLE_GOCACHE",
		}
	}

	// Get expected GOCACHE path
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	customGocacheDir := utils.GoenvEnvVarGocacheDir.UnsafeValue()
	var cachePath string
	if customGocacheDir != "" {
		cachePath = filepath.Join(customGocacheDir, version)
	} else {
		cachePath = filepath.Join(versionPath, "go-build")
	}

	// Check if cache is on a problem mount
	warning := envdetect.CheckCacheOnProblemMount(cachePath)
	if warning != "" {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  StatusWarning,
			message: "Cache directory is on a potentially problematic filesystem",
			advice:  warning,
		}
	}

	// Also check if we're in a container
	if envdetect.IsInContainer() {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  StatusOK,
			message: "Running in container (ensure cache directory is properly mounted)",
			advice:  "For best performance in containers, use Docker volumes instead of bind mounts",
		}
	}

	return checkResult{
		id:      "cache-mount-type",
		name:    "Cache mount type",
		status:  StatusOK,
		message: "Cache directory is on a suitable filesystem",
	}
}

func checkGoToolchain() checkResult {
	gotoolchain := os.Getenv("GOTOOLCHAIN")

	if gotoolchain == "" {
		return checkResult{
			id:      "gotoolchain-setting",
			name:    "GOTOOLCHAIN setting",
			status:  StatusOK,
			message: "GOTOOLCHAIN not set (using default behavior)",
		}
	}

	if gotoolchain == "auto" {
		return checkResult{
			id:      "gotoolchain-setting",
			name:    "GOTOOLCHAIN setting",
			status:  StatusWarning,
			message: "GOTOOLCHAIN=auto can cause issues with goenv version management",
			advice:  "Consider setting GOTOOLCHAIN=local to prevent automatic toolchain switching. Add 'export GOTOOLCHAIN=local' to your shell config.",
		}
	}

	if gotoolchain == "local" {
		return checkResult{
			id:      "gotoolchain-setting",
			name:    "GOTOOLCHAIN setting",
			status:  StatusOK,
			message: "GOTOOLCHAIN=local (recommended for goenv users)",
		}
	}

	// Other values like "go1.23.2" or "local+auto"
	return checkResult{
		id:      "gotoolchain-setting",
		name:    "GOTOOLCHAIN setting",
		status:  StatusWarning,
		message: fmt.Sprintf("GOTOOLCHAIN=%s may interfere with goenv", gotoolchain),
		advice:  "Consider setting GOTOOLCHAIN=local for consistent goenv behavior",
	}
}

func checkCacheIsolationEffectiveness(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	version, _, err := mgr.GetCurrentVersion()
	if err != nil || version == "" || version == "system" {
		return checkResult{
			id:      "architecture-aware-cache-isolation",
			name:    "Architecture-aware cache isolation",
			status:  StatusOK,
			message: "Not applicable (no managed version active)",
		}
	}

	// Check if cache isolation is disabled
	if utils.GoenvEnvVarDisableGocache.IsTrue() {
		return checkResult{
			id:      "architecture-aware-cache-isolation",
			name:    "Architecture-aware cache isolation",
			status:  StatusOK,
			message: "Cache isolation disabled by GOENV_DISABLE_GOCACHE",
		}
	}

	// Get GOOS and GOARCH (if set for cross-compile)
	goos := os.Getenv("GOOS")
	goarch := os.Getenv("GOARCH")
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	// Get Go binary path for ABI detection
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	goBinaryPath := filepath.Join(versionPath, "bin", "go")
	if utils.IsWindows() {
		goBinaryPath += ".exe"
	}

	// Build expected cache path using same logic as exec.go
	// Start with OS-arch
	cacheSuffix := fmt.Sprintf("go-build-%s-%s", goos, goarch)

	// Add ABI variants (GOAMD64, GOARM, etc.)
	abiSuffix := goenv.BuildABISuffix(goBinaryPath, goarch, os.Environ())
	cacheSuffix += abiSuffix

	// Add GOEXPERIMENT if set
	if goexp := os.Getenv("GOEXPERIMENT"); goexp != "" {
		goexp = strings.ReplaceAll(goexp, ",", "-")
		cacheSuffix += "-exp-" + goexp
	}

	// Add CGO toolchain hash if CGO is enabled
	if cgo.IsCGOEnabled(os.Environ()) {
		cgoHash := cgo.ComputeToolchainHash(os.Environ())
		if cgoHash != "" {
			cacheSuffix += "-cgo-" + cgoHash[:8]
		}
	}

	customGocacheDir := utils.GoenvEnvVarGocacheDir.UnsafeValue()

	var expectedGocache string
	if customGocacheDir != "" {
		expectedGocache = filepath.Join(customGocacheDir, version, cacheSuffix)
	} else {
		expectedGocache = filepath.Join(versionPath, cacheSuffix)
	}

	// Check if the cache directory exists
	cacheExists := false
	if stat, err := os.Stat(expectedGocache); err == nil && stat.IsDir() {
		cacheExists = true
	}

	// Check for old-style cache (without architecture suffix)
	oldCachePath := filepath.Join(versionPath, "go-build")
	oldCacheExists := false
	if stat, err := os.Stat(oldCachePath); err == nil && stat.IsDir() {
		oldCacheExists = true
	}

	if !cacheExists && !oldCacheExists {
		return checkResult{
			id:      "architecture-aware-cache-isolation",
			name:    "Architecture-aware cache isolation",
			status:  StatusOK,
			message: fmt.Sprintf("Cache will be created at: %s", expectedGocache),
			advice:  "Architecture-aware isolation prevents 'exec format error' during cross-compilation",
		}
	}

	if cacheExists {
		message := fmt.Sprintf("Using architecture-aware cache: %s", cacheSuffix)
		if oldCacheExists {
			message += " (old cache also exists but will be ignored)"
		}

		return checkResult{
			id:      "architecture-aware-cache-isolation",
			name:    "Architecture-aware cache isolation",
			status:  StatusOK,
			message: message,
			advice:  "This prevents tool binary conflicts between native builds and cross-compilation",
		}
	}

	// Only old cache exists
	return checkResult{
		id:      "architecture-aware-cache-isolation",
		name:    "Architecture-aware cache isolation",
		status:  StatusWarning,
		message: fmt.Sprintf("Found old-style cache at %s", oldCachePath),
		advice:  fmt.Sprintf("New architecture-aware cache will be created at: %s. Old cache can be removed with: goenv cache clean build", expectedGocache),
	}
}

func checkRosetta(cfg *config.Config) checkResult {
	// Only relevant on macOS
	if runtime.GOOS != "darwin" {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "Not applicable (not macOS)",
		}
	}

	// Check if we're running under Rosetta
	// On Apple Silicon, running x86_64 binaries under Rosetta can be detected
	// by checking if the process architecture differs from the machine architecture

	// Get the native architecture
	cmd := exec.Command("sysctl", "-n", "hw.optional.arm64")
	output, err := cmd.Output()
	if err != nil {
		// Probably not Apple Silicon, or sysctl failed
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "Not applicable (not Apple Silicon)",
		}
	}

	hasArm64 := strings.TrimSpace(string(output)) == "1"
	if !hasArm64 {
		// Intel Mac
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "Not applicable (Intel Mac)",
		}
	}

	// We're on Apple Silicon - check if running under Rosetta
	// The Go runtime reports arm64 even when running x86_64 binary under Rosetta
	// We need to check the actual binary architecture
	executable, err := os.Executable()
	if err != nil {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "Cannot determine executable path",
		}
	}

	// Use 'file' command to check actual binary architecture
	fileCmd := exec.Command("file", executable)
	fileOutput, err := fileCmd.Output()
	if err != nil {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "Cannot determine binary architecture",
		}
	}

	fileStr := string(fileOutput)

	// Check if goenv binary is x86_64
	if strings.Contains(fileStr, "x86_64") {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusWarning,
			message: "Running under Rosetta (x86_64 binary on Apple Silicon)",
			advice:  "For better performance, use native arm64 version of goenv. Reinstall via: brew reinstall goenv",
		}
	}

	// Check current Go version architecture
	mgr := manager.NewManager(cfg)
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil || currentVersion == "" || currentVersion == "system" {
		// Can't check Go version
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "goenv is native arm64",
		}
	}

	// Check if the current Go version is x86_64
	goPath, err := exec.LookPath("go")
	if err != nil {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "goenv is native arm64",
		}
	}

	goFileCmd := exec.Command("file", goPath)
	goFileOutput, err := goFileCmd.Output()
	if err != nil {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusOK,
			message: "goenv is native arm64",
		}
	}

	goFileStr := string(goFileOutput)
	if strings.Contains(goFileStr, "x86_64") {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  StatusWarning,
			message: fmt.Sprintf("Go %s is x86_64 (will run under Rosetta)", currentVersion),
			advice:  "Consider using native arm64 Go version for better performance. Install with: goenv install <version>",
		}
	}

	// Everything is native arm64
	return checkResult{
		id:      "rosetta-detection",
		name:    "Rosetta detection",
		status:  StatusOK,
		message: "Running natively on Apple Silicon (arm64)",
	}
}

func checkPathOrder(cfg *config.Config) checkResult {
	// Check that goenv shims directory appears before system Go in PATH
	pathEnv := os.Getenv(utils.EnvVarPath)
	if pathEnv == "" {
		return checkResult{
			id:      "path-order",
			name:    "PATH order",
			status:  StatusError,
			message: "PATH environment variable is empty",
			advice:  "Ensure your shell is properly configured",
		}
	}

	pathDirs := filepath.SplitList(pathEnv)
	shimsDir := filepath.Join(cfg.Root, "shims")

	var shimsIndex int = -1
	var systemGoIndex int = -1

	// Find positions of shims and system Go
	for i, dir := range pathDirs {
		// Check if this is the goenv shims directory
		if dir == shimsDir {
			shimsIndex = i
		}

		// Check if this directory contains a system 'go' binary
		if systemGoIndex == -1 { // Only find first occurrence
			goPath := filepath.Join(dir, "go")
			if utils.IsWindows() {
				goPath += ".exe"
			}

			// Check if file exists and is not in goenv directories
			if stat, err := os.Stat(goPath); err == nil && !stat.IsDir() {
				// Skip if this is in goenv root (versions or shims)
				if !strings.HasPrefix(dir, cfg.Root) {
					systemGoIndex = i
				}
			}
		}
	}

	// Analyze the findings
	if shimsIndex == -1 {
		return checkResult{
			id:      "path-order",
			name:    "PATH order",
			status:  StatusWarning,
			message: fmt.Sprintf("goenv shims directory not in PATH: %s", shimsDir),
			advice:  "Add goenv shims to PATH. Run: eval \"$(goenv init -)\"",
		}
	}

	if systemGoIndex == -1 {
		// No system Go found - this is fine
		return checkResult{
			id:      "path-order",
			name:    "PATH order",
			status:  StatusOK,
			message: "goenv shims are in PATH (no system Go detected)",
		}
	}

	// Both found - check order
	if shimsIndex < systemGoIndex {
		return checkResult{
			id:      "path-order",
			name:    "PATH order",
			status:  StatusOK,
			message: "goenv shims appear before system Go in PATH",
		}
	}

	// System Go appears before goenv shims
	return checkResult{
		id:      "path-order",
		name:    "PATH order",
		status:  StatusWarning,
		message: "System Go appears before goenv shims in PATH",
		advice:  fmt.Sprintf("System Go at position %d, goenv shims at position %d. Commands like 'go' will bypass goenv. Fix: eval \"$(goenv init -)\" in your shell config", systemGoIndex+1, shimsIndex+1),
	}
}

func checkLibcCompatibility(_ *config.Config) checkResult {
	// Detect system libc
	libcInfo := binarycheck.DetectLibc()

	if libcInfo.Type == "unknown" {
		return checkResult{
			id:      "system-c-library",
			name:    "System C library",
			status:  StatusWarning,
			message: "Could not detect system C library (glibc/musl)",
			advice:  "This may indicate an unusual system configuration. CGO-based builds may fail.",
		}
	}

	// Build informative message
	var message string
	switch libcInfo.Type {
	case "musl":
		message = fmt.Sprintf("System uses musl libc (%s)", libcInfo.Path)
	case "glibc":
		if libcInfo.Version != "" {
			message = fmt.Sprintf("System uses glibc (%s, version: %s)", libcInfo.Path, libcInfo.Version)
		} else {
			message = fmt.Sprintf("System uses glibc (%s)", libcInfo.Path)
		}
	}

	// Provide compatibility advice
	var advice string
	switch libcInfo.Type {
	case "musl":
		advice = "💡 You're on a musl-based system (like Alpine Linux).\n" +
			"   • CGO builds: Will work but binaries are musl-specific\n" +
			"   • Static builds: Recommended - use CGO_ENABLED=0 for portability\n" +
			"   • Pre-built tools: May fail if built for glibc. Rebuild locally with: go install <package>\n" +
			"   • Cross-compilation: Binaries built here won't run on glibc systems unless statically linked"
	case "glibc":
		advice = "💡 You're on a glibc-based system (standard Linux).\n" +
			"   • CGO builds: Will work and are portable across glibc systems\n" +
			"   • Static builds: Use CGO_ENABLED=0 for maximum portability\n" +
			"   • Pre-built tools: Generally work across glibc distros\n" +
			"   • Cross-compilation: Binaries built here won't run on musl systems unless statically linked"
	}

	return checkResult{
		id:      "system-c-library",
		name:    "System C library",
		status:  StatusOK,
		message: message,
		advice:  advice,
	}
}

func checkMacOSDeploymentTarget(cfg *config.Config) checkResult {
	// Get current Go binary
	mgr := manager.NewManager(cfg)
	version, _, err := mgr.GetCurrentVersion()
	if err != nil || version == "" || version == "system" {
		return checkResult{
			id:      "macos-deployment-target",
			name:    "macOS deployment target",
			status:  StatusOK,
			message: "Not applicable (no managed version active)",
		}
	}

	// Find go binary
	goBinary := filepath.Join(cfg.VersionsDir(), version, "bin", "go")
	if _, err := os.Stat(goBinary); err != nil {
		return checkResult{
			id:      "macos-deployment-target",
			name:    "macOS deployment target",
			status:  StatusOK,
			message: "Could not find Go binary to check",
		}
	}

	// Check deployment target
	macInfo, issues := binarycheck.CheckMacOSDeploymentTarget(goBinary)
	if macInfo == nil {
		return checkResult{
			id:      "macos-deployment-target",
			name:    "macOS deployment target",
			status:  StatusOK,
			message: "Binary is not a Mach-O file or could not be checked",
		}
	}

	// Build message
	message := fmt.Sprintf("Go binary deployment target: %s", macInfo.DeploymentTarget)
	if !macInfo.HasVersionMin {
		message = "No minimum version requirement detected"
	}

	// Determine status from issues
	status := StatusOK
	advice := ""
	if len(issues) > 0 {
		for _, issue := range issues {
			if issue.Severity == "warning" || issue.Severity == "error" {
				status = StatusWarning
			}
		}
		// Collect advice
		adviceList := []string{}
		for _, issue := range issues {
			if issue.Hint != "" {
				adviceList = append(adviceList, issue.Hint)
			}
		}
		if len(adviceList) > 0 {
			advice = strings.Join(adviceList, "\n   ")
		}
	}

	return checkResult{
		id:      "macos-deployment-target",
		name:    "macOS deployment target",
		status:  status,
		message: message,
		advice:  advice,
	}
}

func checkWindowsCompiler(_ *config.Config) checkResult {
	winInfo, issues := binarycheck.CheckWindowsCompiler()
	if winInfo == nil {
		return checkResult{
			id:      "windows-compiler",
			name:    "Windows compiler",
			status:  StatusOK,
			message: "Not applicable (not on Windows)",
		}
	}

	// Build message
	message := fmt.Sprintf("Compiler: %s", winInfo.Compiler)
	if winInfo.HasCLExe {
		message += " (cl.exe available)"
	}
	if winInfo.HasVCRuntime {
		message += ", VC++ runtime: available"
	} else {
		message += ", VC++ runtime: not detected"
	}

	// Determine status
	status := StatusOK
	advice := ""
	if winInfo.Compiler == "unknown" {
		status = StatusWarning
	}

	// Collect advice from issues
	if len(issues) > 0 {
		for _, issue := range issues {
			if issue.Severity == "warning" || issue.Severity == "error" {
				status = StatusWarning
			}
		}
		adviceList := []string{}
		for _, issue := range issues {
			if issue.Hint != "" {
				adviceList = append(adviceList, issue.Hint)
			}
		}
		if len(adviceList) > 0 {
			advice = strings.Join(adviceList, "\n   ")
		}
	}

	return checkResult{
		id:      "windows-compiler",
		name:    "Windows compiler",
		status:  status,
		message: message,
		advice:  advice,
	}
}

func checkWindowsARM64(_ *config.Config) checkResult {
	winInfo, issues := binarycheck.CheckWindowsARM64()
	if winInfo == nil {
		return checkResult{
			id:      "windows-arm64arm64ec",
			name:    "Windows ARM64/ARM64EC",
			status:  StatusOK,
			message: "Not applicable (not on Windows)",
		}
	}

	// Build message
	message := fmt.Sprintf("Process mode: %s", winInfo.ProcessMode)
	if winInfo.IsARM64EC {
		message += " (ARM64EC available)"
	}

	// Determine status and advice
	status := StatusOK
	advice := ""
	if len(issues) > 0 {
		adviceList := []string{}
		for _, issue := range issues {
			if issue.Hint != "" {
				adviceList = append(adviceList, issue.Hint)
			}
		}
		if len(adviceList) > 0 {
			advice = strings.Join(adviceList, "\n   ")
		}
	}

	return checkResult{
		id:      "windows-arm64arm64ec",
		name:    "Windows ARM64/ARM64EC",
		status:  status,
		message: message,
		advice:  advice,
	}
}

func checkLinuxKernelVersion(_ *config.Config) checkResult {
	linuxInfo, issues := binarycheck.CheckLinuxKernelVersion()
	if linuxInfo == nil {
		return checkResult{
			id:      "linux-kernel-version",
			name:    "Linux kernel version",
			status:  StatusOK,
			message: "Not applicable (not on Linux)",
		}
	}

	// Build message
	message := fmt.Sprintf("Kernel: %s (v%d.%d.%d)", linuxInfo.KernelVersion, linuxInfo.KernelMajor, linuxInfo.KernelMinor, linuxInfo.KernelPatch)

	// Determine status
	status := StatusOK
	advice := ""
	if len(issues) > 0 {
		for _, issue := range issues {
			if issue.Severity == "error" {
				status = StatusError
			} else if issue.Severity == "warning" && status != "error" {
				status = StatusWarning
			}
		}
		// Collect advice
		adviceList := []string{}
		for _, issue := range issues {
			if issue.Hint != "" {
				adviceList = append(adviceList, issue.Hint)
			}
		}
		if len(adviceList) > 0 {
			advice = strings.Join(adviceList, "\n   ")
		}
	}

	return checkResult{
		id:      "linux-kernel-version",
		name:    "Linux kernel version",
		status:  status,
		message: message,
		advice:  advice,
	}
}

// isInteractive checks if the terminal is interactive
func isInteractive() bool {
	// Check if stdin is a terminal
	fileInfo, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}

// offerShellEnvironmentFix prompts the user to fix shell environment issues
func offerShellEnvironmentFix(cmd *cobra.Command, results []checkResult, cfg *config.Config) {
	// Find the shell-environment check result
	var shellEnvResult *checkResult
	for _, result := range results {
		if result.id == "shell-environment" {
			shellEnvResult = &result
			break
		}
	}

	// Only offer fix if there's an issue
	if shellEnvResult == nil || shellEnvResult.status == StatusOK {
		return
	}

	// Determine the appropriate init command for the shell
	shell := shellutil.DetectShell()
	initCommand := shellutil.GetInitLine(shell)
	profilePath := shellutil.GetProfilePathDisplay(shell)

	// Print a clear separator
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "%sShell Environment Issue Detected\n", utils.Emoji("⚠️  "))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", utils.Emoji("❌"), shellEnvResult.message)
	fmt.Fprintln(cmd.OutOrStdout())

	// Ask if they want to see the fix
	fmt.Fprintf(cmd.OutOrStdout(), "Would you like to see the command to fix this? [Y/n]: ")

	reader := bufio.NewReader(doctorStdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		// If we can't read input, just continue
		return
	}

	response = strings.TrimSpace(strings.ToLower(response))
	if response == "" || response == "y" || response == "yes" {
		// Show the fix command prominently with extra spacing
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintf(cmd.OutOrStdout(), "%s Quick Fix\n", utils.Emoji("🔧 "))
		fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sRun this command to activate goenv in your current shell:\n\n", utils.Emoji("💡 "))
		fmt.Fprintf(cmd.OutOrStdout(), "    %s\n\n", utils.BoldGreen(initCommand))
		fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintln(cmd.OutOrStdout())

		// Also provide instructions for making it permanent
		fmt.Fprintf(cmd.OutOrStdout(), "%sTo make this permanent, add the following to %s:\n\n", utils.Emoji("📝 "), profilePath)
		fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", initCommand)
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

		// Pause so user can read and copy the command before more output
		utils.PauseForUser(cmd.OutOrStdout(), reader)
	}
}
