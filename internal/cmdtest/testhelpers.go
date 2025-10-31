package cmdtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

// SetupTestEnv creates an isolated test environment with temporary directories
// and clean environment variables. Returns the test root directory and a cleanup function.
// Exported for use by subpackage tests.
func SetupTestEnv(t *testing.T) (string, func()) {
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
	os.Setenv("HOME", testHome)
	// Clear PATH to ensure no system go is found unless explicitly added by test
	if utils.IsWindows() {
		os.Setenv("PATH", "C:\\Windows\\System32")
	} else {
		os.Setenv("PATH", "/usr/bin:/bin")
	}
	// Clear GOENV_VERSION to ensure clean test environment
	os.Unsetenv("GOENV_VERSION")

	// Create necessary directories
	os.MkdirAll(testRoot, 0755)
	os.MkdirAll(testHome, 0755)
	os.MkdirAll(filepath.Join(testRoot, "versions"), 0755)

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
		os.Setenv("HOME", oldHome)
		os.Setenv("PATH", oldPath)
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
	CreateTestBinary(t, root, version, "go")
}

// CreateTestBinary creates a mock binary in a version's bin directory.
// The binary will echo mock output when executed. This is useful for testing
// commands that need to interact with version-specific binaries.
// Exported for use by subpackage tests.
func CreateTestBinary(t *testing.T, root, version, binaryName string) {
	versionDir := filepath.Join(root, "versions", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create test version directory: %v", err)
	}

	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
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

	if err := os.WriteFile(binaryPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create test binary %s: %v", binaryName, err)
	}
}

// CreateExecutable creates a mock executable at the specified location.
// Unlike CreateTestBinary, this supports both version paths and absolute paths.
// If version contains "/", it's treated as an absolute path to the bin directory.
// Otherwise, it's treated as a version name under $GOENV_ROOT/versions.
// Exported for use by subpackage tests.
func CreateExecutable(t *testing.T, testRoot, version, execName string) {
	var binDir string
	if strings.Contains(version, "/") {
		// Absolute path like "${GOENV_TEST_DIR}/bin"
		binDir = version
	} else {
		// Version name like "1.10.3"
		binDir = filepath.Join(testRoot, "versions", version, "bin")
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	execPath := filepath.Join(binDir, execName)
	content := "#!/bin/sh\necho 'mock executable'\n"
	if utils.IsWindows() {
		execPath += ".bat"
		content = "@echo off\necho mock executable\n"
	}
	if err := os.WriteFile(execPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create executable: %v", err)
	}
}

// CreateTestAlias creates a version alias in the test environment.
// Exported for use by subpackage tests.
func CreateTestAlias(t *testing.T, root, name, target string) {
	aliasesFile := filepath.Join(root, "aliases")

	// Read existing aliases if file exists
	var content string
	if data, err := os.ReadFile(aliasesFile); err == nil {
		content = string(data)
	}

	// Append new alias
	content += name + "=" + target + "\n"

	if err := os.WriteFile(aliasesFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create alias: %v", err)
	}
}

// CreateGoExecutable creates a mock go binary at the specified location.
// This is a convenience wrapper for creating Go executables in test scenarios
// where you need a go binary but don't need a full version installation.
// Exported for use by subpackage tests.
func CreateGoExecutable(t *testing.T, binDir string) string {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	binaryPath := filepath.Join(binDir, "go")
	content := "#!/bin/sh\necho go version go1.21.0 linux/amd64\n"
	if utils.IsWindows() {
		binaryPath += ".bat"
		content = "@echo off\necho go version go1.21.0 windows/amd64\n"
	}

	if err := os.WriteFile(binaryPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	return binaryPath
}

// CreateToolExecutable creates a mock tool binary at the specified location.
// This is useful for testing tool commands without needing actual tool installations.
// Exported for use by subpackage tests.
func CreateToolExecutable(t *testing.T, binDir, toolName string) string {
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	toolPath := filepath.Join(binDir, toolName)
	content := "#!/bin/sh\necho mock " + toolName + "\n"
	if utils.IsWindows() {
		toolPath += ".bat"
		content = "@echo off\necho mock " + toolName + "\n"
	}

	if err := os.WriteFile(toolPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create tool binary: %v", err)
	}

	return toolPath
}

// StripDeprecationWarning removes deprecation warnings from command output.
// This is useful for testing legacy commands that now show deprecation warnings.
// The deprecation warning format is:
//
//	Deprecation warning: ...
//	  Modern command: ...
//	  See: ...
//	[blank line]
//	[actual output]
//
// This function removes the warning block and returns only the actual output.
// Exported for use by subpackage tests.
func StripDeprecationWarning(output string) string {
	// Check if output contains a deprecation warning
	if !strings.Contains(output, "Deprecation warning:") {
		return strings.TrimSpace(output)
	}

	lines := strings.Split(output, "\n")

	// Find the blank line that separates warning from actual output
	// The warning block is: "Deprecation warning:" + 2 indent lines + blank line
	blankLineIdx := -1
	for i, line := range lines {
		if i > 0 && strings.HasPrefix(lines[0], "Deprecation warning:") && strings.TrimSpace(line) == "" {
			blankLineIdx = i
			break
		}
	}

	if blankLineIdx == -1 {
		// No blank line found, warning only
		return ""
	}

	// Return everything after the blank line
	if blankLineIdx+1 >= len(lines) {
		return ""
	}

	result := strings.Join(lines[blankLineIdx+1:], "\n")
	// Only trim trailing whitespace to preserve formatting of the output
	return strings.TrimRight(result, " \t\n\r")
}
