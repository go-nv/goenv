package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/vscode"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:     "doctor",
	Short:   "Diagnose goenv installation and configuration issues",
	GroupID: "common",
	Long: `Checks your goenv installation and configuration for common issues.

This command verifies:
  - goenv binary and paths
  - Shell configuration
  - PATH setup
  - Shims directory
  - Installed Go versions
  - Common configuration problems

Use this command to troubleshoot issues with goenv.`,
	RunE: runDoctor,
}

type checkResult struct {
	name    string
	status  string // "ok", "warning", "error"
	message string
	advice  string
}

func init() {
	rootCmd.AddCommand(doctorCmd)
	helptext.SetCommandHelp(doctorCmd)
}

func runDoctor(cmd *cobra.Command, args []string) error {
	cfg := config.Load()
	results := []checkResult{}

	fmt.Fprintln(cmd.OutOrStdout(), "🔍 Checking goenv installation...")
	fmt.Fprintln(cmd.OutOrStdout())

	// Check 1: goenv binary location
	results = append(results, checkGoenvBinary(cfg))

	// Check 2: GOENV_ROOT
	results = append(results, checkGoenvRoot(cfg))

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

	// Print results
	fmt.Fprintln(cmd.OutOrStdout(), "📋 Diagnostic Results:")
	fmt.Fprintln(cmd.OutOrStdout())

	okCount := 0
	warningCount := 0
	errorCount := 0

	for _, result := range results {
		var icon string
		switch result.status {
		case "ok":
			icon = "✅"
			okCount++
		case "warning":
			icon = "⚠️ "
			warningCount++
		case "error":
			icon = "❌"
			errorCount++
		}

		fmt.Fprintf(cmd.OutOrStdout(), "%s %s\n", icon, result.name)
		if result.message != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   %s\n", result.message)
		}
		if result.advice != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   💡 %s\n", result.advice)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d OK, %d warnings, %d errors\n", okCount, warningCount, errorCount)

	if errorCount > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\n❌ Issues found. Please review the errors above.")
		return fmt.Errorf("goenv installation has %d error(s)", errorCount)
	} else if warningCount > 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\n⚠️  Everything works, but some warnings should be reviewed.")
	} else {
		fmt.Fprintln(cmd.OutOrStdout(), "\n✅ Everything looks good!")
	}

	return nil
}

func checkGoenvBinary(cfg *config.Config) checkResult {
	// Find goenv binary
	goenvPath, err := os.Executable()
	if err != nil {
		return checkResult{
			name:    "goenv binary",
			status:  "error",
			message: fmt.Sprintf("Cannot determine goenv binary location: %v", err),
			advice:  "Ensure goenv is properly installed",
		}
	}

	return checkResult{
		name:    "goenv binary",
		status:  "ok",
		message: fmt.Sprintf("Found at: %s", goenvPath),
	}
}

func checkGoenvRoot(cfg *config.Config) checkResult {
	root := cfg.Root
	if _, err := os.Stat(root); os.IsNotExist(err) {
		return checkResult{
			name:    "GOENV_ROOT directory",
			status:  "error",
			message: fmt.Sprintf("Directory does not exist: %s", root),
			advice:  "Run 'goenv init' to create the directory structure",
		}
	}

	return checkResult{
		name:    "GOENV_ROOT directory",
		status:  "ok",
		message: fmt.Sprintf("Set to: %s", root),
	}
}

func checkShellConfig(cfg *config.Config) checkResult {
	shell := os.Getenv("SHELL")
	if shell == "" {
		return checkResult{
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
			name:    "Shell configuration",
			status:  "ok",
			message: fmt.Sprintf("goenv detected in %s", foundIn),
		}
	}

	return checkResult{
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
			name:    "PATH configuration",
			status:  "error",
			message: fmt.Sprintf("%s not in PATH", goenvBin),
			advice:  fmt.Sprintf("Add 'export PATH=\"%s:$PATH\"' to your shell config", goenvBin),
		}
	}

	if !hasShims {
		return checkResult{
			name:    "PATH configuration",
			status:  "warning",
			message: fmt.Sprintf("%s not in PATH", shimsDir),
			advice:  "Run 'eval \"$(goenv init -)\"' in your shell config",
		}
	}

	// Check if shims are early in PATH (should be near the front)
	if shimsPosition > 5 {
		return checkResult{
			name:    "PATH configuration",
			status:  "warning",
			message: fmt.Sprintf("Shims directory is at position %d in PATH", shimsPosition),
			advice:  "Shims should be near the beginning of PATH for proper version switching",
		}
	}

	return checkResult{
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
			name:    "Shims directory",
			status:  "warning",
			message: fmt.Sprintf("Shims directory does not exist: %s", shimsDir),
			advice:  "Run 'goenv rehash' to create shims",
		}
	}
	if err != nil {
		return checkResult{
			name:    "Shims directory",
			status:  "error",
			message: fmt.Sprintf("Cannot access shims directory: %v", err),
			advice:  "Check file permissions",
		}
	}

	if !stat.IsDir() {
		return checkResult{
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
			name:    "Shims directory",
			status:  "warning",
			message: fmt.Sprintf("Cannot read shims directory: %v", err),
		}
	}

	shimCount := len(entries)
	if shimCount == 0 {
		return checkResult{
			name:    "Shims directory",
			status:  "warning",
			message: "No shims found",
			advice:  "Run 'goenv rehash' to create shims",
		}
	}

	return checkResult{
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
			name:    "Installed Go versions",
			status:  "error",
			message: fmt.Sprintf("Cannot list versions: %v", err),
			advice:  "Check GOENV_ROOT and versions directory",
		}
	}

	if len(versions) == 0 {
		return checkResult{
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
			name:    "Installed Go versions",
			status:  "error",
			message: fmt.Sprintf("Found %d version(s), but %d are CORRUPTED: %s", len(versions), len(corruptedVersions), strings.Join(corruptedVersions, ", ")),
			advice:  fmt.Sprintf("Reinstall corrupted versions: goenv uninstall %s && goenv install %s", corruptedVersions[0], corruptedVersions[0]),
		}
	}

	return checkResult{
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
			name:    "Current Go version",
			status:  "warning",
			message: fmt.Sprintf("No version set: %v", err),
			advice:  "Set a version with 'goenv global <version>' or create a .go-version file",
		}
	}

	if version == "system" {
		return checkResult{
			name:    "Current Go version",
			status:  "ok",
			message: fmt.Sprintf("Using system Go (set by %s)", source),
		}
	}

	// Validate version is installed
	if err := mgr.ValidateVersion(version); err != nil {
		return checkResult{
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
			name:    "Current Go version",
			status:  "error",
			message: fmt.Sprintf("Version '%s' is CORRUPTED - go binary missing (set by %s)", version, source),
			advice:  fmt.Sprintf("Reinstall: goenv uninstall %s && goenv install %s", version, version),
		}
	}

	return checkResult{
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
		for _, sysGo := range systemGoLocations {
			if dir == sysGo {
				shimsFirst = false
				break
			}
		}
		if !shimsFirst {
			break
		}
	}

	if shimsFirst {
		return checkResult{
			name:    "Conflicting Go installations",
			status:  "ok",
			message: fmt.Sprintf("Found system Go at %s, but goenv shims have priority", strings.Join(systemGoLocations, ", ")),
		}
	}

	return checkResult{
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
				name:    "Cache files",
				status:  "warning",
				message: fmt.Sprintf("Cannot read %s: %v", cacheName, err),
				advice:  "Run 'goenv refresh cache' to regenerate cache files",
			}
		}
	}

	return checkResult{
		name:    "Cache files",
		status:  "ok",
		message: fmt.Sprintf("Found %d cache file(s): %v", len(foundCaches), foundCaches),
	}
}

func checkNetwork() checkResult {
	// Try to reach golang.org
	cmd := exec.Command("ping", "-c", "1", "-W", "2", "golang.org")
	if runtime.GOOS == "windows" {
		cmd = exec.Command("ping", "-n", "1", "-w", "2000", "golang.org")
	}

	if err := cmd.Run(); err != nil {
		return checkResult{
			name:    "Network connectivity",
			status:  "warning",
			message: "Cannot reach golang.org",
			advice:  "You may not be able to fetch new Go versions. Check your internet connection.",
		}
	}

	return checkResult{
		name:    "Network connectivity",
		status:  "ok",
		message: "Can reach golang.org",
	}
}

func checkVSCodeIntegration(cfg *config.Config) checkResult {
	// Get current working directory
	cwd, err := os.Getwd()
	if err != nil {
		// Can't check, but not critical
		return checkResult{
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
			name:    "VS Code integration",
			status:  "ok",
			message: "No .vscode directory found",
			advice:  "Run 'goenv vscode init' to set up VS Code integration",
		}
	}

	// Check if settings.json exists
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		return checkResult{
			name:    "VS Code integration",
			status:  "warning",
			message: "Found .vscode directory but no settings.json",
			advice:  "Run 'goenv vscode init' to configure Go extension",
		}
	}

	// Get current Go version to validate against
	mgr := manager.NewManager(cfg)
	currentVersion, _, err := mgr.GetCurrentVersion()
	if err != nil || currentVersion == "" {
		// Can't determine current version - do basic check
		return checkResult{
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
			name:    "VS Code integration",
			status:  "warning",
			message: "settings.json exists but missing Go configuration",
			advice:  "Run 'goenv vscode init' to add goenv configuration",
		}
	}

	if result.UsesEnvVars {
		return checkResult{
			name:    "VS Code integration",
			status:  "ok",
			message: "VS Code configured to use goenv environment variables (${env:GOROOT})",
		}
	}

	if result.Mismatch {
		return checkResult{
			name:    "VS Code integration",
			status:  "warning",
			message: fmt.Sprintf("VS Code settings use Go %s but current version is %s", result.ConfiguredVersion, currentVersion),
			advice:  "Run 'goenv vscode init --force' to update VS Code settings to match current version",
		}
	}

	if result.ConfiguredVersion != "" {
		return checkResult{
			name:    "VS Code integration",
			status:  "ok",
			message: fmt.Sprintf("VS Code configured with absolute path for Go %s", result.ConfiguredVersion),
		}
	}

	// Has go.goroot but couldn't parse version
	return checkResult{
		name:    "VS Code integration",
		status:  "warning",
		message: "VS Code has Go configuration but cannot determine version",
		advice:  "Run 'goenv vscode init --force' to update settings",
	}
}

func checkGoModVersion(cfg *config.Config) checkResult {
	cwd, _ := os.Getwd()
	gomodPath := filepath.Join(cwd, "go.mod")

	// Only check if go.mod exists
	if _, err := os.Stat(gomodPath); os.IsNotExist(err) {
		return checkResult{
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
			name:    "go.mod version",
			status:  "error",
			message: fmt.Sprintf("go.mod requires Go %s but current version is %s", requiredVersion, currentVersion),
			advice:  advice,
		}
	}

	return checkResult{
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
				name:    "Actual 'go' binary",
				status:  "ok",
				message: fmt.Sprintf("Using system Go %s via goenv shim at %s", actualVersion, goPath),
			}
		}
		return checkResult{
			name:    "Actual 'go' binary",
			status:  "ok",
			message: fmt.Sprintf("Using system Go %s at %s (set by %s)", actualVersion, goPath, source),
		}
	}

	// Compare versions
	if actualVersion != expectedVersion {
		if isUsingShim {
			return checkResult{
				name:    "Actual 'go' binary",
				status:  "error",
				message: fmt.Sprintf("Version mismatch: expected %s (set by %s) but 'go version' reports %s", expectedVersion, source, actualVersion),
				advice:  "This may indicate a corrupted installation. Try: goenv rehash",
			}
		}

		// Not using shim - PATH issue
		return checkResult{
			name:    "Actual 'go' binary",
			status:  "error",
			message: fmt.Sprintf("Version mismatch: expected %s (set by %s) but using %s at %s", expectedVersion, source, actualVersion, goPath),
			advice:  "The 'go' binary at " + goPath + " is taking precedence. Ensure goenv shims directory (" + shimsDir + ") is first in your PATH. Run: eval \"$(goenv init -)\". If you see build cache errors, run: goenv clean",
		}
	}

	// Versions match!
	if isUsingShim {
		return checkResult{
			name:    "Actual 'go' binary",
			status:  "ok",
			message: fmt.Sprintf("Correctly using Go %s via goenv shim", actualVersion),
		}
	}

	// Version is correct but not using shim - a bit unusual but not wrong
	return checkResult{
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
			name:    "Tool migration",
			status:  "ok",
			message: "Only one Go version installed",
		}
	}

	// Check for tools in current version
	currentTools, err := listToolsForVersion(cfg, currentVersion)
	if err != nil {
		return checkResult{
			name:    "Tool migration",
			status:  "ok",
			message: "Cannot detect installed tools",
		}
	}

	// If current version has tools, nothing to suggest
	if len(currentTools) > 0 {
		return checkResult{
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
			name:    "Tool migration",
			status:  "ok",
			message: "No tools installed in any version",
		}
	}

	// Found tools in other versions but not current - suggest sync
	if len(versionsWithTools) == 1 {
		return checkResult{
			name:    "Tool sync",
			status:  "warning",
			message: fmt.Sprintf("Current Go %s has no tools, but Go %s has %d tool(s)", currentVersion, bestSourceVersion, maxToolCount),
			advice:  fmt.Sprintf("Sync tools with: goenv sync-tools (or: goenv sync-tools %s)", bestSourceVersion),
		}
	}

	// Multiple versions have tools
	return checkResult{
		name:    "Tool sync",
		status:  "warning",
		message: fmt.Sprintf("Current Go %s has no tools, but %d other version(s) have tools (e.g., Go %s has %d tool(s))", currentVersion, len(versionsWithTools), bestSourceVersion, maxToolCount),
		advice:  fmt.Sprintf("Sync tools from best source: goenv sync-tools (will auto-select Go %s)", bestSourceVersion),
	}
}

func checkGocacheIsolation(cfg *config.Config) checkResult {
	mgr := manager.NewManager(cfg)
	version, _, err := mgr.GetCurrentVersion()
	if err != nil || version == "" {
		return checkResult{
			name:    "Build cache isolation",
			status:  "ok",
			message: "Not applicable (no version set)",
		}
	}

	if version == "system" {
		return checkResult{
			name:    "Build cache isolation",
			status:  "ok",
			message: "Not applicable (using system Go)",
		}
	}

	// Check if GOCACHE isolation is disabled
	if os.Getenv("GOENV_DISABLE_GOCACHE") == "1" {
		return checkResult{
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
			name:    "Cache architecture",
			status:  "ok",
			message: "Cannot determine GOCACHE location",
		}
	}

	// Check if cache directory exists
	stat, err := os.Stat(gocache)
	if err != nil || !stat.IsDir() {
		return checkResult{
			name:    "Cache architecture",
			status:  "ok",
			message: "Build cache is empty or doesn't exist yet",
		}
	}

	// Check if it's a version-specific cache (contains GOENV_ROOT path)
	isVersionSpecific := strings.Contains(gocache, cfg.Root)

	if isVersionSpecific {
		return checkResult{
			name:    "Cache architecture",
			status:  "ok",
			message: fmt.Sprintf("Using version-specific cache for %s/%s", currentOS, currentArch),
		}
	}

	return checkResult{
		name:    "Cache architecture",
		status:  "warning",
		message: fmt.Sprintf("Using shared system cache at %s for %s/%s", gocache, currentOS, currentArch),
		advice:  "If you see 'exec format error', run: goenv clean build",
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
