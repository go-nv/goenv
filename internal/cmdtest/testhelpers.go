package cmdtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
)

// SetupTestEnv creates an isolated test environment with temporary directories
// and clean environment variables. Returns the test root directory and a cleanup function.
// Exported for use by subpackage tests.
func SetupTestEnv(t *testing.T) (string, func()) {
	t.Helper()
	// Create temporary test directory
	testDir, err := os.MkdirTemp("", "goenv_test_")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Set test environment variables
	oldGoenvRoot := utils.GoenvEnvVarRoot.UnsafeValue()
	oldHome := os.Getenv(utils.EnvVarHome)
	oldPath := os.Getenv(utils.EnvVarPath)
	oldGoenvVersion := utils.GoenvEnvVarVersion.UnsafeValue()

	testRoot := filepath.Join(testDir, "root")
	testHome := filepath.Join(testDir, "home")

	utils.GoenvEnvVarRoot.Set(testRoot)
	os.Setenv(utils.EnvVarHome, testHome)
	// Clear PATH to ensure no system go is found unless explicitly added by test
	if utils.IsWindows() {
		os.Setenv(utils.EnvVarPath, "C:\\Windows\\System32")
	} else {
		os.Setenv(utils.EnvVarPath, "/usr/bin:/bin")
	}
	// Clear GOENV_VERSION to ensure clean test environment
	os.Unsetenv(utils.GoenvEnvVarVersion.String())

	// Create necessary directories
	_ = utils.EnsureDirWithContext(testRoot, "create test directory")
	_ = utils.EnsureDirWithContext(testHome, "create test directory")
	utils.EnsureDir(filepath.Join(testRoot, "versions"))

	// Change to testHome to avoid picking up any .go-version files from the repository
	oldDir, _ := os.Getwd()
	os.Chdir(testHome)

	// Also set GOENV_DIR to testHome to prevent any directory traversal finding repo .go-version
	oldGoenvDir := utils.GoenvEnvVarDir.UnsafeValue()
	utils.GoenvEnvVarDir.Set(testHome)

	// Cleanup function
	cleanup := func() {
		os.Chdir(oldDir)
		utils.GoenvEnvVarRoot.Set(oldGoenvRoot)
		os.Setenv(utils.EnvVarHome, oldHome)
		os.Setenv(utils.EnvVarPath, oldPath)
		utils.GoenvEnvVarDir.Set(oldGoenvDir)
		if oldGoenvVersion != "" {
			utils.GoenvEnvVarVersion.Set(oldGoenvVersion)
		}
		os.RemoveAll(testDir)
	}

	return testRoot, cleanup
}

// CreateTestVersion creates a mock Go version installation with a fake go binary
// that echoes version information when executed.
// Exported for use by subpackage tests.
func CreateTestVersion(t *testing.T, root, version string) {
	t.Helper()
	CreateTestBinary(t, root, version, "go")
}

// CreateTestBinary creates a mock binary in a version's bin directory.
// The binary will echo mock output when executed. This is useful for testing
// commands that need to interact with version-specific binaries.
// Exported for use by subpackage tests.
func CreateTestBinary(t *testing.T, root, version, binaryName string) {
	t.Helper()
	versionDir := filepath.Join(root, "versions", version)
	if err := utils.EnsureDirWithContext(versionDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create test version directory: %v", err)
	}

	binDir := filepath.Join(versionDir, "bin")
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create test bin directory: %v", err)
	}

	// Create mock binary
	binaryPath := filepath.Join(binDir, binaryName)
	var content string
	if utils.IsWindows() {
		binaryPath += ".bat"
		if binaryName == "go" {
			content = "@echo off\necho go version go" + version + " windows/amd64\n"
		} else {
			content = "@echo off\necho Mock " + binaryName + " from version " + version + "\n"
		}
	} else {
		if binaryName == "go" {
			content = "#!/bin/sh\necho go version go" + version + " linux/amd64\n"
		} else {
			content = "#!/bin/bash\necho 'Mock " + binaryName + " from version " + version + "'\n"
		}
	}

	if err := utils.WriteFileWithContext(binaryPath, []byte(content), utils.PermFileExecutable, "create test binary %s"); err != nil {
		t.Fatalf("Failed to create test binary %s: %v", binaryName, err)
	}
}

// CreateExecutable creates a mock executable at the specified location.
// Unlike CreateTestBinary, this supports both version paths and absolute paths.
// If version contains "/", it's treated as an absolute path to the bin directory.
// Otherwise, it's treated as a version name under $GOENV_ROOT/versions.
// Exported for use by subpackage tests.
func CreateExecutable(t *testing.T, testRoot, version, execName string) {
	t.Helper()
	var binDir string
	if strings.Contains(version, "/") {
		// Absolute path like "${GOENV_TEST_DIR}/bin"
		binDir = version
	} else {
		// Version name like "1.10.3"
		binDir = filepath.Join(testRoot, "versions", version, "bin")
	}

	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	execPath := filepath.Join(binDir, execName)
	content := "#!/bin/sh\necho 'mock executable'\n"
	if utils.IsWindows() {
		execPath += ".bat"
		content = "@echo off\necho mock executable\n"
	}
	if err := utils.WriteFileWithContext(execPath, []byte(content), utils.PermFileExecutable, "create executable"); err != nil {
		t.Fatalf("Failed to create executable: %v", err)
	}
}

// CreateTestAlias creates a version alias in the test environment.
// Exported for use by subpackage tests.
func CreateTestAlias(t *testing.T, root, name, target string) {
	t.Helper()
	aliasesFile := filepath.Join(root, "aliases")

	// Read existing aliases if file exists
	var content string
	if data, err := os.ReadFile(aliasesFile); err == nil {
		content = string(data)
	}

	// Append new alias
	content += name + "=" + target + "\n"

	if err := utils.WriteFileWithContext(aliasesFile, []byte(content), utils.PermFileDefault, "create alias"); err != nil {
		t.Fatalf("Failed to create alias: %v", err)
	}
}

// CreateGoExecutable creates a mock go binary at the specified location.
// This is a convenience wrapper for creating Go executables in test scenarios
// where you need a go binary but don't need a full version installation.
// Exported for use by subpackage tests.
func CreateGoExecutable(t *testing.T, binDir string) string {
	t.Helper()
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	binaryPath := filepath.Join(binDir, "go")
	content := "#!/bin/sh\necho go version go1.21.0 linux/amd64\n"
	if utils.IsWindows() {
		binaryPath += ".bat"
		content = "@echo off\necho go version go1.21.0 windows/amd64\n"
	}

	if err := utils.WriteFileWithContext(binaryPath, []byte(content), utils.PermFileExecutable, "create go binary"); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	return binaryPath
}

// CreateToolExecutable creates a mock tool binary at the specified location.
// This is useful for testing tool commands without needing actual tool installations.
// Exported for use by subpackage tests.
func CreateToolExecutable(t *testing.T, binDir, toolName string) string {
	t.Helper()
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	toolPath := filepath.Join(binDir, toolName)
	content := "#!/bin/sh\necho mock " + toolName + "\n"
	if utils.IsWindows() {
		toolPath += ".bat"
		content = "@echo off\necho mock " + toolName + "\n"
	}

	if err := utils.WriteFileWithContext(toolPath, []byte(content), utils.PermFileExecutable, "create tool binary"); err != nil {
		t.Fatalf("Failed to create tool binary: %v", err)
	}

	return toolPath
}

// WriteTestFile writes a file for testing purposes with standardized error handling.
// If the write fails, it immediately fails the test with t.Fatalf.
// The optional msg parameter allows specifying a custom error message context.
// If not provided, defaults to "Failed to write test file <path>".
//
// DEPRECATED: Use testing/testutil.WriteTestFile instead for better import cycle avoidance.
// This wrapper is kept for backward compatibility but will be removed in a future version.
//
// Example:
//
//	WriteTestFile(t, path, data, utils.PermFileDefault)  // Default error message
//	WriteTestFile(t, path, data, utils.PermFileDefault, "Failed to create config file")  // Custom message
func WriteTestFile(tb testing.TB, path string, content []byte, perm os.FileMode, msg ...string) {
	tb.Helper()
	if err := os.WriteFile(path, content, perm); err != nil {
		if len(msg) > 0 && msg[0] != "" {
			tb.Fatalf("%s: %v", msg[0], err)
		} else {
			tb.Fatalf("Failed to write test file %s: %v", path, err)
		}
	}
}

// StripDeprecationWarning removes deprecation warnings from command output.
// This is useful for testing legacy commands that now show deprecation warnings.
//
// DEPRECATED: Use testing/testutil.StripDeprecationWarning instead for better import cycle avoidance.
// This wrapper is kept for backward compatibility but will be removed in a future version.
//
// The deprecation warning format is:
//
//	Deprecation warning: ...
//	  Modern command: ...
//	  See: ...
//	[blank line]
//	[actual output]
//
// This function removes the warning block and returns only the actual output.
func StripDeprecationWarning(output string) string {
	// Check if output contains warnings (deprecation or lifecycle)
	hasDeprecation := strings.Contains(output, "Deprecation warning:")
	hasLifecycleWarning := strings.Contains(output, "Warning: Go")

	if !hasDeprecation && !hasLifecycleWarning {
		return strings.TrimSpace(output)
	}

	lines := strings.Split(output, "\n")
	result := []string{}
	skipUntilBlank := false

	for i, line := range lines {
		// Check if this line starts a warning block
		if strings.HasPrefix(line, "Deprecation warning:") || strings.HasPrefix(line, "Warning: Go") {
			skipUntilBlank = true
			continue
		}

		// If we're skipping, wait for blank line or end of warning
		if skipUntilBlank {
			// Lifecycle warnings are 2-3 lines, look for blank line or next non-warning content
			if strings.TrimSpace(line) == "" {
				skipUntilBlank = false
				continue
			}
			// Skip lines that are part of the warning (indented or continuation)
			if strings.HasPrefix(line, " ") || strings.Contains(line, "Consider upgrading") || strings.Contains(line, "support ends soon") {
				continue
			}
			// If we hit a non-warning line, stop skipping
			skipUntilBlank = false
		}

		// Add non-warning lines
		if !skipUntilBlank && i > 0 && strings.TrimSpace(line) == "" && len(result) == 0 {
			// Skip leading blank lines after warnings
			continue
		}
		if !skipUntilBlank {
			result = append(result, line)
		}
	}

	output = strings.Join(result, "\n")
	return strings.TrimRight(output, " \t\n\r")
}

// CreateMockGoVersion creates a complete mock Go version installation.
// This includes:
//   - Version directory (versions/{version})
//   - Bin directory (versions/{version}/bin)
//   - Go binary (versions/{version}/bin/go[.exe])
//
// The go binary is executable and will echo version information when run.
// Returns the path to the version directory.
//
// This is the recommended way to set up test versions as it uses Config helpers
// for consistent path construction across all tests.
func CreateMockGoVersion(t *testing.T, tmpDir, version string) string {
	t.Helper()

	cfg := &config.Config{Root: tmpDir}

	// Create bin directory
	binDir := cfg.VersionBinDir(version)
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create version bin dir: %v", err)
	}

	// Create go binary
	goBinary := cfg.VersionGoBinary(version)
	if utils.IsWindows() {
		goBinary += ".exe"
	}

	// Create executable mock binary
	var content string
	if utils.IsWindows() {
		content = "@echo off\necho go version go" + version + " windows/amd64\n"
	} else {
		content = "#!/bin/sh\necho go version go" + version + " linux/amd64\n"
	}

	if err := utils.WriteFileWithContext(goBinary, []byte(content), utils.PermFileExecutable, "create go binary"); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	return cfg.VersionDir(version)
}

// CreateMockGoVersions creates multiple mock Go version installations.
// This is a convenience wrapper for creating several versions at once.
//
// Example:
//
//	CreateMockGoVersions(t, tmpDir, "1.21.0", "1.22.0", "1.23.0")
func CreateMockGoVersions(t *testing.T, tmpDir string, versions ...string) {
	t.Helper()
	for _, v := range versions {
		CreateMockGoVersion(t, tmpDir, v)
	}
}

// CreateMockGoVersionWithTools creates a mock Go version with gopath/bin directory.
// This is useful for testing tool installations, as tools are installed to
// versions/{version}/gopath/bin.
//
// Returns the path to the version directory.
func CreateMockGoVersionWithTools(t *testing.T, tmpDir, version string) string {
	t.Helper()

	// Create base version
	versionPath := CreateMockGoVersion(t, tmpDir, version)

	// Create gopath/bin directory
	cfg := &config.Config{Root: tmpDir}
	gopathBin := cfg.VersionGopathBin(version)
	if err := utils.EnsureDirWithContext(gopathBin, "create test directory"); err != nil {
		t.Fatalf("Failed to create gopath bin: %v", err)
	}

	return versionPath
}
