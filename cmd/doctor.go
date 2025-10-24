package cmd

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/binarycheck"
	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/envdetect"
	"github.com/go-nv/goenv/internal/goenv"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/vscode"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Diagnose goenv installation and configuration issues",
	GroupID: "common",
	Long: `Checks your goenv installation and configuration for common issues.

This command verifies:
  - Runtime environment (containers, WSL, native)
  - Filesystem type (NFS, SMB, FUSE, local)
  - goenv binary and paths
  - Shell configuration
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

Exit codes (for CI/automation):
  0 = No issues found (or only warnings when --fail-on=error)
  1 = Errors found
  2 = Warnings found (when --fail-on=warning)

Flags:
  --json       Output results in JSON format for CI/automation
  --fail-on    Exit with non-zero status on 'error' (default) or 'warning'`,
	RunE: runDoctor,
}

type checkResult struct {
	id      string // Machine-readable identifier for CI/automation
	name    string
	status  string // "ok", "warning", "error"
	message string
	advice  string
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
		Status:  c.status,
		Message: c.message,
		Advice:  c.advice,
	})
}

var (
	doctorJSON   bool
	doctorFailOn string
	// doctorExit is a function variable that can be overridden in tests
	doctorExit = os.Exit
)

func init() {
	rootCmd.AddCommand(doctorCmd)
	doctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Output results in JSON format")
	doctorCmd.Flags().StringVar(&doctorFailOn, "fail-on", "error", "Exit with non-zero status on 'error' or 'warning' (for CI strictness)")
	helptext.SetCommandHelp(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	results := []checkResult{}

	// Validate --fail-on flag
	if doctorFailOn != "error" && doctorFailOn != "warning" {
		return fmt.Errorf("invalid --fail-on value: %s (must be 'error' or 'warning')", doctorFailOn)
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
	if runtime.GOOS == "windows" {
		results = append(results, checkWindowsCompiler(cfg))
	}

	// Check 24: Windows ARM64/ARM64EC (Windows only)
	if runtime.GOOS == "windows" {
		results = append(results, checkWindowsARM64(cfg))
	}

	// Check 25: Linux kernel version (Linux only)
	if runtime.GOOS == "linux" {
		results = append(results, checkLinuxKernelVersion(cfg))
	}

	// Count results
	okCount := 0
	warningCount := 0
	errorCount := 0
	for _, result := range results {
		switch result.status {
		case "ok":
			okCount++
		case "warning":
			warningCount++
		case "error":
			errorCount++
		}
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
		} else if doctorFailOn == "warning" && warningCount > 0 {
			doctorExit(2) // Warnings exit with code 2 when --fail-on=warning
		}
		return nil
	}

	// Human-readable output
	fmt.Fprintf(cmd.OutOrStdout(), "%sDiagnostic Results:\n", utils.Emoji("📋 "))
	fmt.Fprintln(cmd.OutOrStdout())

	for _, result := range results {
		var icon string
		switch result.status {
		case "ok":
			icon = utils.Emoji("✅ ")
		case "warning":
			icon = utils.Emoji("⚠️  ")
		case "error":
			icon = utils.Emoji("❌ ")
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s%s\n", icon, result.name)
		if result.message != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", result.message)
		}
		if result.advice != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   %s%s\n", utils.Emoji("💡 "), result.advice)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d OK, %d warnings, %d errors\n", okCount, warningCount, errorCount)

	// Exit codes for CI clarity:
	//   0 = success (no issues or only warnings when --fail-on=error)
	//   1 = errors found
	//   2 = warnings found (when --fail-on=warning)
	if errorCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%sIssues found. Please review the errors above.\n", utils.Emoji("❌ "))
		doctorExit(1) // Errors always exit with code 1
	} else if warningCount > 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%sEverything works, but some warnings should be reviewed.\n", utils.Emoji("⚠️  "))
		// Check if we should fail on warnings based on --fail-on flag
		if doctorFailOn == "warning" {
			doctorExit(2) // Warnings exit with code 2 when --fail-on=warning
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "\n%sEverything looks good!\n", utils.Emoji("✅ "))
	}

	return nil
}

func checkEnvironment(cfg *config.Config) checkResult {
	// Detect runtime environment
	envInfo := envdetect.Detect()

	// Build status message
	message := fmt.Sprintf("Running on %s", envInfo.String())

	// Determine status based on warnings
	status := "ok"
	advice := ""

	if envInfo.IsProblematicEnvironment() {
		warnings := envInfo.GetWarnings()
		if len(warnings) > 0 {
			status = "warning"
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
	status := "ok"
	advice := ""

	switch envInfo.FilesystemType {
	case envdetect.FSTypeNFS:
		status = "warning"
		advice = "NFS filesystems can cause file locking issues and slow I/O. Consider using a local filesystem for GOENV_ROOT."
	case envdetect.FSTypeSMB:
		status = "warning"
		advice = "SMB/CIFS filesystems may have issues with symbolic links and permissions. Consider using a local filesystem for GOENV_ROOT."
	case envdetect.FSTypeBind:
		status = "warning"
		advice = "Bind mounts in containers should be persistent and have correct permissions."
	case envdetect.FSTypeFUSE:
		status = "warning"
		advice = "FUSE filesystems may have performance issues. Consider using a local filesystem for better performance."
	case envdetect.FSTypeUnknown:
		status = "warning"
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
			status:  "error",
			message: fmt.Sprintf("Cannot determine goenv binary location: %v", err),
			advice:  "Ensure goenv is properly installed",
		}
	}

	return checkResult{
		id:      "goenv-binary",
		name:    "goenv binary",
		status:  "ok",
		message: fmt.Sprintf("Found at: %s", goenvPath),
	}
}

func checkGoenvRoot(cfg *config.Config) checkResult {
	root := cfg.Root
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return checkResult{
			id:      "goenvroot-directory",
			name:    "GOENV_ROOT directory",
			status:  "error",
			message: fmt.Sprintf("Directory does not exist: %s", root),
			advice:  "Run 'goenv init' to create the directory structure",
		}
	}

	return checkResult{
		id:      "goenvroot-directory",
		name:    "GOENV_ROOT directory",
		status:  "ok",
		message: fmt.Sprintf("Set to: %s", root),
	}
}

func checkShellConfig(_ *config.Config) checkResult {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return checkResult{
			id:      "shell-configuration",
			name:    "Shell configuration",
			status:  "warning",
			message: "SHELL environment variable not set",
			advice:  "This is unusual. Check your shell configuration.",
		}
	}

	// Determine config file
	homeDir, _ := os.UserHomeDir()
	var configFiles []string

	shellName := filepath.Base(shell)
	switch shellName {
	case "bash":
		configFiles = []string{
			filepath.Join(homeDir, ".bashrc"),
			filepath.Join(homeDir, ".bash_profile"),
			filepath.Join(homeDir, ".profile"),
		}
	case "zsh":
		configFiles = []string{
			filepath.Join(homeDir, ".zshrc"),
			filepath.Join(homeDir, ".zprofile"),
		}
	case "fish":
		configFiles = []string{
			filepath.Join(homeDir, ".config", "fish", "config.fish"),
		}
	default:
		return checkResult{
			id:      "shell-configuration",
			name:    "Shell configuration",
			status:  "warning",
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
			status:  "ok",
			message: fmt.Sprintf("goenv detected in %s", foundIn),
		}
	}

	return checkResult{
		id:      "shell-configuration",
		name:    "Shell configuration",
		status:  "warning",
		message: "goenv init not found in shell config",
		advice:  fmt.Sprintf("Add 'eval \"$(goenv init -)\"' to your %s", configFiles[0]),
	}
}

func checkPath(cfg *config.Config) checkResult {
	pathEnv := os.Getenv("PATH")
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
			status:  "error",
			message: fmt.Sprintf("%s not in PATH", goenvBin),
			advice:  fmt.Sprintf("Add 'export PATH=\"%s:$PATH\"' to your shell config", goenvBin),
		}
	}

	if !hasShims {
		return checkResult{
			id:      "path-configuration",
			name:    "PATH configuration",
			status:  "warning",
			message: fmt.Sprintf("%s not in PATH", shimsDir),
			advice:  "Run 'eval \"$(goenv init -)\"' in your shell config",
		}
	}

	// Check if shims are early in PATH (should be near the front)
	if shimsPosition > 5 {
		return checkResult{
			id:      "path-configuration",
			name:    "PATH configuration",
			status:  "warning",
			message: fmt.Sprintf("Shims directory is at position %d in PATH", shimsPosition),
			advice:  "Shims should be near the beginning of PATH for proper version switching",
		}
	}

	return checkResult{
		id:      "path-configuration",
		name:    "PATH configuration",
		status:  "ok",
		message: "goenv bin and shims directories are in PATH",
	}
}

func checkShimsDir(cfg *config.Config) checkResult {
	shimsDir := cfg.ShimsDir()

	stat, err := os.Stat(shimsDir)
	if os.IsNotExist(err) {
		return checkResult{
			id:      "shims-directory",
			name:    "Shims directory",
			status:  "warning",
			message: fmt.Sprintf("Shims directory does not exist: %s", shimsDir),
			advice:  "Run 'goenv rehash' to create shims",
		}
	}
	if err != nil {
		return checkResult{
			id:      "shims-directory",
			name:    "Shims directory",
			status:  "error",
			message: fmt.Sprintf("Cannot access shims directory: %v", err),
			advice:  "Check file permissions",
		}
	}

	if !stat.IsDir() {
		return checkResult{
			id:      "shims-directory",
			name:    "Shims directory",
			status:  "error",
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
			status:  "warning",
			message: fmt.Sprintf("Cannot read shims directory: %v", err),
		}
	}

	shimCount := len(entries)
	if shimCount == 0 {
		return checkResult{
			id:      "shims-directory",
			name:    "Shims directory",
			status:  "warning",
			message: "No shims found",
			advice:  "Run 'goenv rehash' to create shims",
		}
	}

	return checkResult{
		id:      "shims-directory",
		name:    "Shims directory",
		status:  "ok",
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
			status:  "error",
			message: fmt.Sprintf("Cannot list versions: %v", err),
			advice:  "Check GOENV_ROOT and versions directory",
		}
	}

	if len(versions) == 0 {
		return checkResult{
			id:      "installed-go-versions",
			name:    "Installed Go versions",
			status:  "warning",
			message: "No Go versions installed",
			advice:  "Install a Go version with 'goenv install <version>'",
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
			id:      "installed-go-versions",
			name:    "Installed Go versions",
			status:  "error",
			message: fmt.Sprintf("Found %d version(s), but %d are CORRUPTED: %s", len(versions), len(corruptedVersions), strings.Join(corruptedVersions, ", ")),
			advice:  fmt.Sprintf("Reinstall corrupted versions: goenv uninstall %s && goenv install %s", corruptedVersions[0], corruptedVersions[0]),
		}
	}

	return checkResult{
		id:      "installed-go-versions",
		name:    "Installed Go versions",
		status:  "ok",
		message: fmt.Sprintf("Found %d valid version(s): %s", len(validVersions), strings.Join(validVersions, ", ")),
	}
}

func checkCurrentVersion(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	version, source, err := mgr.GetCurrentVersion()

	if err != nil {
		return checkResult{
			id:      "current-go-version",
			name:    "Current Go version",
			status:  "warning",
			message: fmt.Sprintf("No version set: %v", err),
			advice:  "Set a version with 'goenv global <version>' or create a .go-version file",
		}
	}

	if version == "system" {
		return checkResult{
			id:      "current-go-version",
			name:    "Current Go version",
			status:  "ok",
			message: fmt.Sprintf("Using system Go (set by %s)", source),
		}
	}

	// Validate version is installed
	if err := mgr.ValidateVersion(version); err != nil {
		return checkResult{
			id:      "current-go-version",
			name:    "Current Go version",
			status:  "error",
			message: fmt.Sprintf("Version '%s' is set but not installed (set by %s)", version, source),
			advice:  fmt.Sprintf("Install the version with 'goenv install %s'", version),
		}
	}

	// Check if the installation is corrupted (missing go binary)
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	goBinaryBase := filepath.Join(versionPath, "bin", "go")

	// Check if go binary exists (handles .exe and .bat on Windows)
	if _, err := pathutil.FindExecutable(goBinaryBase); err != nil {
		return checkResult{
			id:      "current-go-version",
			name:    "Current Go version",
			status:  "error",
			message: fmt.Sprintf("Version '%s' is CORRUPTED - go binary missing (set by %s)", version, source),
			advice:  fmt.Sprintf("Reinstall: goenv uninstall %s && goenv install %s", version, version),
		}
	}

	return checkResult{
		id:      "current-go-version",
		name:    "Current Go version",
		status:  "ok",
		message: fmt.Sprintf("%s (set by %s)", version, source),
	}
}

func checkConflictingGo(cfg *config.Config) checkResult {
	// Check for system Go installations that might conflict
	pathEnv := os.Getenv("PATH")
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
		if runtime.GOOS == "windows" {
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
			status:  "ok",
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
			status:  "ok",
			message: fmt.Sprintf("Found system Go at %s, but goenv shims have priority", strings.Join(systemGoLocations, ", ")),
		}
	}

	return checkResult{
		id:      "conflicting-go-installations",
		name:    "Conflicting Go installations",
		status:  "warning",
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
			status:  "ok",
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
				status:  "warning",
				message: fmt.Sprintf("Cannot read %s: %v", cacheName, err),
				advice:  "Run 'goenv refresh cache' to regenerate cache files",
			}
		}
	}

	return checkResult{
		id:      "cache-files",
		name:    "Cache files",
		status:  "ok",
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
			status:  "warning",
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
			status:  "warning",
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
			status:  "ok",
			message: "Can reach go.dev",
		}
	}

	// Got a response but unexpected status code
	return checkResult{
		id:      "network-connectivity",
		name:    "Network connectivity",
		status:  "warning",
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
			status:  "ok",
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
			status:  "ok",
			message: "No .vscode directory found",
			advice:  "Run 'goenv vscode init' to set up VS Code integration with Go settings",
		}
	}

	// Check if settings.json exists
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  "warning",
			message: "Found .vscode directory but no settings.json",
			advice:  "Run 'goenv vscode init' to configure Go extension, or 'goenv vscode doctor' for detailed diagnostics",
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
			status:  "warning",
			message: "Cannot determine current Go version for validation",
			advice:  "Set a Go version with 'goenv global' or 'goenv local' first",
		}
	}

	// Use sophisticated VS Code settings check
	result := vscode.CheckSettings(settingsFile, currentVersion)

	if !result.HasSettings {
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  "warning",
			message: "settings.json exists but missing Go configuration",
			advice:  "Run 'goenv vscode init' to add goenv configuration, or 'goenv vscode doctor' for detailed diagnostics",
		}
	}

	if result.UsesEnvVars {
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  "ok",
			message: "VS Code configured to use goenv environment variables (${env:GOROOT})",
		}
	}

	if result.Mismatch {
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  "warning",
			message: fmt.Sprintf("VS Code settings use Go %s but current version is %s", result.ConfiguredVersion, currentVersion),
			advice:  "Run 'goenv vscode sync' to fix, or 'goenv vscode doctor' for detailed diagnostics",
		}
	}

	if result.ConfiguredVersion != "" {
		return checkResult{
			id:      "vs-code-integration",
			name:    "VS Code integration",
			status:  "ok",
			message: fmt.Sprintf("VS Code configured with absolute path for Go %s", result.ConfiguredVersion),
		}
	}

	// Has go.goroot but couldn't parse version
	return checkResult{
		id:      "vs-code-integration",
		name:    "VS Code integration",
		status:  "warning",
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
			status:  "ok",
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
			status:  "error",
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
			status:  "warning",
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
			id:      "gomod-version",
			name:    "go.mod version",
			status:  "error",
			message: fmt.Sprintf("go.mod requires Go %s but current version is %s", requiredVersion, currentVersion),
			advice:  advice,
		}
	}

	return checkResult{
		id:      "gomod-version",
		name:    "go.mod version",
		status:  "ok",
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
			status:  "warning",
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
			status:  "error",
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
			status:  "error",
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
			status:  "warning",
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
				status:  "ok",
				message: fmt.Sprintf("Using system Go %s via goenv shim at %s", actualVersion, goPath),
			}
		}
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  "ok",
			message: fmt.Sprintf("Using system Go %s at %s (set by %s)", actualVersion, goPath, source),
		}
	}

	// Compare versions
	if actualVersion != expectedVersion {
		if isUsingShim {
			return checkResult{
				id:      "actual-go-binary",
				name:    "Actual 'go' binary",
				status:  "error",
				message: fmt.Sprintf("Version mismatch: expected %s (set by %s) but 'go version' reports %s", expectedVersion, source, actualVersion),
				advice:  "This may indicate a corrupted installation. Try: goenv rehash",
			}
		}

		// Not using shim - PATH issue
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  "error",
			message: fmt.Sprintf("Version mismatch: expected %s (set by %s) but using %s at %s", expectedVersion, source, actualVersion, goPath),
			advice:  "The 'go' binary at " + goPath + " is taking precedence. Ensure goenv shims directory (" + shimsDir + ") is first in your PATH. Run: eval \"$(goenv init -)\". If you see build cache errors, run: goenv cache clean build",
		}
	}

	// Versions match!
	if isUsingShim {
		return checkResult{
			id:      "actual-go-binary",
			name:    "Actual 'go' binary",
			status:  "ok",
			message: fmt.Sprintf("Correctly using Go %s via goenv shim", actualVersion),
		}
	}

	// Version is correct but not using shim - a bit unusual but not wrong
	return checkResult{
		id:      "actual-go-binary",
		name:    "Actual 'go' binary",
		status:  "ok",
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
			status:  "ok",
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
			status:  "ok",
			message: "Only one Go version installed",
		}
	}

	// Check for tools in current version
	currentTools, err := listToolsForVersion(cfg, currentVersion)
	if err != nil {
		return checkResult{
			id:      "tool-migration",
			name:    "Tool migration",
			status:  "ok",
			message: "Cannot detect installed tools",
		}
	}

	// If current version has tools, nothing to suggest
	if len(currentTools) > 0 {
		return checkResult{
			id:      "tool-migration",
			name:    "Tool migration",
			status:  "ok",
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
			status:  "ok",
			message: "No tools installed in any version",
		}
	}

	// Found tools in other versions but not current - suggest sync
	if len(versionsWithTools) == 1 {
		return checkResult{
			id:      "tool-sync",
			name:    "Tool sync",
			status:  "warning",
			message: fmt.Sprintf("Current Go %s has no tools, but Go %s has %d tool(s)", currentVersion, bestSourceVersion, maxToolCount),
			advice:  fmt.Sprintf("Sync tools with: goenv tools sync (or: goenv tools sync %s)", bestSourceVersion),
		}
	}

	// Multiple versions have tools
	return checkResult{
		id:      "tool-sync",
		name:    "Tool sync",
		status:  "warning",
		message: fmt.Sprintf("Current Go %s has no tools, but %d other version(s) have tools (e.g., Go %s has %d tool(s))", currentVersion, len(versionsWithTools), bestSourceVersion, maxToolCount),
		advice:  fmt.Sprintf("Sync tools from best source: goenv tools sync (will auto-select Go %s)", bestSourceVersion),
	}
}

func checkGocacheIsolation(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	version, _, err := mgr.GetCurrentVersion()
	if err != nil || version == "" {
		return checkResult{
			id:      "build-cache-isolation",
			name:    "Build cache isolation",
			status:  "ok",
			message: "Not applicable (no version set)",
		}
	}

	if version == "system" {
		return checkResult{
			id:      "build-cache-isolation",
			name:    "Build cache isolation",
			status:  "ok",
			message: "Not applicable (using system Go)",
		}
	}

	// Check if GOCACHE isolation is disabled
	if os.Getenv("GOENV_DISABLE_GOCACHE") == "1" {
		return checkResult{
			id:      "build-cache-isolation",
			name:    "Build cache isolation",
			status:  "ok",
			message: "Cache isolation disabled by GOENV_DISABLE_GOCACHE",
		}
	}

	// Get expected GOCACHE path
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	customGocacheDir := os.Getenv("GOENV_GOCACHE_DIR")
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
		status:  "ok",
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
			status:  "ok",
			message: "Cannot determine GOCACHE location",
		}
	}

	// Check if cache directory exists
	stat, err := os.Stat(gocache)
	if err != nil || !stat.IsDir() {
		return checkResult{
			id:      "cache-architecture",
			name:    "Cache architecture",
			status:  "ok",
			message: "Build cache is empty or doesn't exist yet",
		}
	}

	// Check if it's a version-specific cache (contains GOENV_ROOT path)
	isVersionSpecific := strings.Contains(gocache, cfg.Root)

	if isVersionSpecific {
		return checkResult{
			id:      "cache-architecture",
			name:    "Cache architecture",
			status:  "ok",
			message: fmt.Sprintf("Using version-specific cache for %s/%s", currentOS, currentArch),
		}
	}

	return checkResult{
		id:      "cache-architecture",
		name:    "Cache architecture",
		status:  "warning",
		message: fmt.Sprintf("Using shared system cache at %s for %s/%s", gocache, currentOS, currentArch),
		advice:  "If you see 'exec format error', run: goenv cache clean build",
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
		// Remove .exe extension on Windows
		if runtime.GOOS == "windows" && strings.HasSuffix(name, ".exe") {
			name = strings.TrimSuffix(name, ".exe")
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
			status:  "ok",
			message: "Not applicable (no version set)",
		}
	}

	if version == "system" {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  "ok",
			message: "Not applicable (using system Go)",
		}
	}

	// Check if GOCACHE isolation is disabled
	if os.Getenv("GOENV_DISABLE_GOCACHE") == "1" {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  "ok",
			message: "Cache isolation disabled by GOENV_DISABLE_GOCACHE",
		}
	}

	// Get expected GOCACHE path
	versionPath := filepath.Join(cfg.VersionsDir(), version)
	customGocacheDir := os.Getenv("GOENV_GOCACHE_DIR")
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
			status:  "warning",
			message: "Cache directory is on a potentially problematic filesystem",
			advice:  warning,
		}
	}

	// Also check if we're in a container
	if envdetect.IsInContainer() {
		return checkResult{
			id:      "cache-mount-type",
			name:    "Cache mount type",
			status:  "ok",
			message: "Running in container (ensure cache directory is properly mounted)",
			advice:  "For best performance in containers, use Docker volumes instead of bind mounts",
		}
	}

	return checkResult{
		id:      "cache-mount-type",
		name:    "Cache mount type",
		status:  "ok",
		message: "Cache directory is on a suitable filesystem",
	}
}

func checkGoToolchain() checkResult {
	gotoolchain := os.Getenv("GOTOOLCHAIN")

	if gotoolchain == "" {
		return checkResult{
			id:      "gotoolchain-setting",
			name:    "GOTOOLCHAIN setting",
			status:  "ok",
			message: "GOTOOLCHAIN not set (using default behavior)",
		}
	}

	if gotoolchain == "auto" {
		return checkResult{
			id:      "gotoolchain-setting",
			name:    "GOTOOLCHAIN setting",
			status:  "warning",
			message: "GOTOOLCHAIN=auto can cause issues with goenv version management",
			advice:  "Consider setting GOTOOLCHAIN=local to prevent automatic toolchain switching. Add 'export GOTOOLCHAIN=local' to your shell config.",
		}
	}

	if gotoolchain == "local" {
		return checkResult{
			id:      "gotoolchain-setting",
			name:    "GOTOOLCHAIN setting",
			status:  "ok",
			message: "GOTOOLCHAIN=local (recommended for goenv users)",
		}
	}

	// Other values like "go1.23.2" or "local+auto"
	return checkResult{
		id:      "gotoolchain-setting",
		name:    "GOTOOLCHAIN setting",
		status:  "warning",
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
			status:  "ok",
			message: "Not applicable (no managed version active)",
		}
	}

	// Check if cache isolation is disabled
	if os.Getenv("GOENV_DISABLE_GOCACHE") == "1" {
		return checkResult{
			id:      "architecture-aware-cache-isolation",
			name:    "Architecture-aware cache isolation",
			status:  "ok",
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
	if runtime.GOOS == "windows" {
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

	customGocacheDir := os.Getenv("GOENV_GOCACHE_DIR")

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
			status:  "ok",
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
			status:  "ok",
			message: message,
			advice:  "This prevents tool binary conflicts between native builds and cross-compilation",
		}
	}

	// Only old cache exists
	return checkResult{
		id:      "architecture-aware-cache-isolation",
		name:    "Architecture-aware cache isolation",
		status:  "warning",
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
			status:  "ok",
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
			status:  "ok",
			message: "Not applicable (not Apple Silicon)",
		}
	}

	hasArm64 := strings.TrimSpace(string(output)) == "1"
	if !hasArm64 {
		// Intel Mac
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  "ok",
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
			status:  "ok",
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
			status:  "ok",
			message: "Cannot determine binary architecture",
		}
	}

	fileStr := string(fileOutput)

	// Check if goenv binary is x86_64
	if strings.Contains(fileStr, "x86_64") {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  "warning",
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
			status:  "ok",
			message: "goenv is native arm64",
		}
	}

	// Check if the current Go version is x86_64
	goPath, err := exec.LookPath("go")
	if err != nil {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  "ok",
			message: "goenv is native arm64",
		}
	}

	goFileCmd := exec.Command("file", goPath)
	goFileOutput, err := goFileCmd.Output()
	if err != nil {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  "ok",
			message: "goenv is native arm64",
		}
	}

	goFileStr := string(goFileOutput)
	if strings.Contains(goFileStr, "x86_64") {
		return checkResult{
			id:      "rosetta-detection",
			name:    "Rosetta detection",
			status:  "warning",
			message: fmt.Sprintf("Go %s is x86_64 (will run under Rosetta)", currentVersion),
			advice:  "Consider using native arm64 Go version for better performance. Install with: goenv install <version>",
		}
	}

	// Everything is native arm64
	return checkResult{
		id:      "rosetta-detection",
		name:    "Rosetta detection",
		status:  "ok",
		message: "Running natively on Apple Silicon (arm64)",
	}
}

func checkPathOrder(cfg *config.Config) checkResult {
	// Check that goenv shims directory appears before system Go in PATH
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return checkResult{
			id:      "path-order",
			name:    "PATH order",
			status:  "error",
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
			if runtime.GOOS == "windows" {
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
			status:  "warning",
			message: fmt.Sprintf("goenv shims directory not in PATH: %s", shimsDir),
			advice:  "Add goenv shims to PATH. Run: eval \"$(goenv init -)\"",
		}
	}

	if systemGoIndex == -1 {
		// No system Go found - this is fine
		return checkResult{
			id:      "path-order",
			name:    "PATH order",
			status:  "ok",
			message: "goenv shims are in PATH (no system Go detected)",
		}
	}

	// Both found - check order
	if shimsIndex < systemGoIndex {
		return checkResult{
			id:      "path-order",
			name:    "PATH order",
			status:  "ok",
			message: "goenv shims appear before system Go in PATH",
		}
	}

	// System Go appears before goenv shims
	return checkResult{
		id:      "path-order",
		name:    "PATH order",
		status:  "warning",
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
			status:  "warning",
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
		status:  "ok",
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
			status:  "ok",
			message: "Not applicable (no managed version active)",
		}
	}

	// Find go binary
	goBinary := filepath.Join(cfg.VersionsDir(), version, "bin", "go")
	if _, err := os.Stat(goBinary); err != nil {
		return checkResult{
			id:      "macos-deployment-target",
			name:    "macOS deployment target",
			status:  "ok",
			message: "Could not find Go binary to check",
		}
	}

	// Check deployment target
	macInfo, issues := binarycheck.CheckMacOSDeploymentTarget(goBinary)
	if macInfo == nil {
		return checkResult{
			id:      "macos-deployment-target",
			name:    "macOS deployment target",
			status:  "ok",
			message: "Binary is not a Mach-O file or could not be checked",
		}
	}

	// Build message
	message := fmt.Sprintf("Go binary deployment target: %s", macInfo.DeploymentTarget)
	if !macInfo.HasVersionMin {
		message = "No minimum version requirement detected"
	}

	// Determine status from issues
	status := "ok"
	advice := ""
	if len(issues) > 0 {
		for _, issue := range issues {
			if issue.Severity == "warning" || issue.Severity == "error" {
				status = "warning"
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
			status:  "ok",
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
	status := "ok"
	advice := ""
	if winInfo.Compiler == "unknown" {
		status = "warning"
	}

	// Collect advice from issues
	if len(issues) > 0 {
		for _, issue := range issues {
			if issue.Severity == "warning" || issue.Severity == "error" {
				status = "warning"
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
			status:  "ok",
			message: "Not applicable (not on Windows)",
		}
	}

	// Build message
	message := fmt.Sprintf("Process mode: %s", winInfo.ProcessMode)
	if winInfo.IsARM64EC {
		message += " (ARM64EC available)"
	}

	// Determine status and advice
	status := "ok"
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
			status:  "ok",
			message: "Not applicable (not on Linux)",
		}
	}

	// Build message
	message := fmt.Sprintf("Kernel: %s (v%d.%d.%d)", linuxInfo.KernelVersion, linuxInfo.KernelMajor, linuxInfo.KernelMinor, linuxInfo.KernelPatch)

	// Determine status
	status := "ok"
	advice := ""
	if len(issues) > 0 {
		for _, issue := range issues {
			if issue.Severity == "error" {
				status = "error"
			} else if issue.Severity == "warning" && status != "error" {
				status = "warning"
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
