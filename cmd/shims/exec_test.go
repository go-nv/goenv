package shims

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

func TestExecCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		globalVersion  string
		localVersion   string
		envVersion     string
		expectedOutput string
		expectedError  string
		createBinaries map[string]string // version -> binary content
	}{
		{
			name:           "exec with global version",
			args:           []string{"go", "version"},
			setupVersions:  []string{"1.21.5"},
			globalVersion:  "1.21.5",
			createBinaries: map[string]string{"1.21.5": "#!/bin/sh\necho go version go1.21.5 linux/amd64\n"},
			expectedOutput: "go version go1.21.5 linux/amd64",
		},
		{
			name:           "exec with local version override",
			args:           []string{"go", "version"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			globalVersion:  "1.21.5",
			localVersion:   "1.22.2",
			createBinaries: map[string]string{"1.22.2": "#!/bin/sh\necho go version go1.22.2 linux/amd64\n"},
			expectedOutput: "go version go1.22.2 linux/amd64",
		},
		{
			name:           "exec with environment version override",
			args:           []string{"go", "version"},
			setupVersions:  []string{"1.21.5", "1.22.2", "1.23.0"},
			globalVersion:  "1.21.5",
			localVersion:   "1.22.2",
			envVersion:     "1.23.0",
			createBinaries: map[string]string{"1.23.0": "#!/bin/sh\necho go version go1.23.0 linux/amd64\n"},
			expectedOutput: "go version go1.23.0 linux/amd64",
		},
		{
			name:          "error with no command specified",
			args:          []string{},
			expectedError: "usage: goenv exec <command> [arg1 arg2...]",
		},
		{
			name:          "error with version not installed (env)",
			args:          []string{"go", "version"},
			setupVersions: []string{"1.21.5"},
			envVersion:    "1.6.1",
			expectedError: "version '1.6.1' is not installed (set by GOENV_VERSION environment variable)",
		},
		{
			name:          "error with version not installed (local)",
			args:          []string{"go", "build"},
			setupVersions: []string{"1.21.5"},
			localVersion:  "1.6.1",
			expectedError: "version '1.6.1' is not installed (set by",
		},
		{
			name:          "exec binary that doesn't exist in version",
			args:          []string{"nonexistent", "arg"},
			setupVersions: []string{"1.21.5"},
			globalVersion: "1.21.5",
			expectedError: "goenv: nonexistent: command not found",
		},
		{
			name:           "exec with multiple arguments",
			args:           []string{"--", "go", "build", "-o", "output", "main.go"},
			setupVersions:  []string{"1.21.5"},
			globalVersion:  "1.21.5",
			createBinaries: map[string]string{"1.21.5": "#!/bin/sh\necho \"Building: $@\"\n"},
			expectedOutput: "Building: build -o output main.go",
		},
		{
			name:           "exec with system version",
			args:           []string{"go", "version"},
			setupVersions:  []string{"1.21.5"},
			globalVersion:  "system",
			expectedOutput: "system go version",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Clear GOENV_VERSION to avoid interference from outer environment
			t.Setenv(utils.GoenvEnvVarVersion.String(), "")
			os.Unsetenv("GOENV_VERSION")

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, testRoot, version)

				// Create custom binaries if specified
				if content, exists := tt.createBinaries[version]; exists {
					binPath := filepath.Join(testRoot, "versions", version, "bin", "go")
					if utils.IsWindows() {
						binPath += ".bat"
						// Convert Unix shell script to Windows batch file
						content = strings.ReplaceAll(content, "#!/bin/sh\n", "@echo off\n")
						// Convert Unix shell variable expansion to Windows batch
						content = strings.ReplaceAll(content, "$@", "%*")
						// Windows batch echo includes quotes in output, so strip them
						content = strings.ReplaceAll(content, `echo "`, `echo `)
						content = strings.ReplaceAll(content, `"`+"\n", "\n")
					}
					testutil.WriteTestFile(t, binPath, []byte(content), utils.PermFileExecutable, fmt.Sprintf("Failed to create custom binary for version %s", version))
				}
			}

			// Set global version if specified
			if tt.globalVersion != "" {
				globalFile := filepath.Join(testRoot, "version")
				testutil.WriteTestFile(t, globalFile, []byte(tt.globalVersion), utils.PermFileDefault, "Failed to set global version")
			}

			// Setup local version if specified
			var tempDir string
			if tt.localVersion != "" {
				var err error
				tempDir, err = os.MkdirTemp("", "goenv_exec_test_")
				require.NoError(t, err, "Failed to create temp directory")
				defer os.RemoveAll(tempDir)

				localFile := filepath.Join(tempDir, ".go-version")
				testutil.WriteTestFile(t, localFile, []byte(tt.localVersion), utils.PermFileDefault)

				// Change to the directory with local version
				oldDir, _ := os.Getwd()
				defer os.Chdir(oldDir)
				os.Chdir(tempDir)
			}

			// Setup environment version if specified
			if tt.envVersion != "" {
				t.Setenv(utils.GoenvEnvVarVersion.String(), tt.envVersion)
			}

			// Setup system go for system version test
			if tt.globalVersion == manager.SystemVersion {
				systemBinDir := filepath.Join(testRoot, "system_bin")
				_ = utils.EnsureDirWithContext(systemBinDir, "create test directory")
				systemGo := filepath.Join(systemBinDir, "go")
				var content string
				if utils.IsWindows() {
					systemGo += ".bat"
					content = "@echo off\necho system go version\n"
				} else {
					content = "#!/bin/sh\necho system go version\n"
				}

				testutil.WriteTestFile(t, systemGo, []byte(content), utils.PermFileExecutable, "Failed to create system go")

				// Add to PATH temporarily
				oldPath := os.Getenv(utils.EnvVarPath)
				pathSep := string(os.PathListSeparator)
				os.Setenv(utils.EnvVarPath, systemBinDir+pathSep+oldPath)
				defer os.Setenv(utils.EnvVarPath, oldPath)
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "exec",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runExec(cmd, args)
				},
			}

			// Capture output
			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetErr(output)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			// Verify results
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			assert.NoError(t, err)

			if tt.expectedOutput != "" {
				got := strings.TrimSpace(output.String())
				assert.Equal(t, tt.expectedOutput, got)
			}
		})
	}
}

func TestExecUsage(t *testing.T) {
	_, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	cmd := &cobra.Command{
		Use: "exec",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExec(cmd, args)
		},
	}

	// Test help output
	cmd.SetArgs([]string{"--help"})
	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	assert.NoError(t, err, "Help command failed")

	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Usage:", "Help output should contain usage information")
}

func TestExecEnvironmentVariables(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test version with a binary that prints environment
	version := "1.21.5"
	cmdtest.CreateTestVersion(t, testRoot, version)

	// Create a binary that prints specific environment variables
	binPath := filepath.Join(testRoot, "versions", version, "bin", "env-test")
	var content string
	if utils.IsWindows() {
		binPath += ".bat"
		content = "@echo off\necho GOROOT=%GOROOT%\necho GOPATH=%GOPATH%\n"
	} else {
		content = "#!/bin/sh\necho \"GOROOT=$GOROOT\"\necho \"GOPATH=$GOPATH\"\n"
	}

	testutil.WriteTestFile(t, binPath, []byte(content), utils.PermFileExecutable, "Failed to create env-test binary")

	// Set global version
	globalFile := filepath.Join(testRoot, "version")
	testutil.WriteTestFile(t, globalFile, []byte(version), utils.PermFileDefault, "Failed to set global version")

	// Create and execute command
	cmd := &cobra.Command{
		Use: "exec",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExec(cmd, args)
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{"env-test"})

	err := cmd.Execute()
	assert.NoError(t, err, "Exec command failed")

	got := output.String()

	// Should set GOROOT to the version directory
	expectedGoroot := filepath.Join(testRoot, "versions", version)
	assert.Contains(t, got, "GOROOT="+expectedGoroot)
}

func TestExecWithShims(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test version
	version := "1.21.5"
	cmdtest.CreateTestVersion(t, testRoot, version)

	// Set global version
	globalFile := filepath.Join(testRoot, "version")
	testutil.WriteTestFile(t, globalFile, []byte(version), utils.PermFileDefault, "Failed to set global version")

	// Create shims directory and shim
	shimsDir := filepath.Join(testRoot, "shims")
	err := utils.EnsureDirWithContext(shimsDir, "create test directory")
	require.NoError(t, err, "Failed to create shims directory")

	shimPath := filepath.Join(shimsDir, "go")
	var shimContent string
	if utils.IsWindows() {
		shimPath += ".bat"
		shimContent = "@echo off\ngoenv exec go %*\n"
	} else {
		shimContent = "#!/usr/bin/env bash\nexec goenv exec \"$(basename \"$0\")\" \"$@\"\n"
	}
	testutil.WriteTestFile(t, shimPath, []byte(shimContent), utils.PermFileExecutable)

	// Create and execute command - should work even when called via shim path
	cmd := &cobra.Command{
		Use: "exec",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runExec(cmd, args)
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{"go", "version"})

	err = cmd.Execute()
	assert.NoError(t, err, "Exec command failed")

	got := strings.TrimSpace(output.String())
	assert.Contains(t, got, "go1.21.5", "Expected output to contain version info %v", got)
}

// TestFindEnvVar tests the case-insensitive environment variable finder
func TestFindEnvVar(t *testing.T) {
	tests := []struct {
		name         string
		env          []string
		key          string
		expectedIdx  int
		expectedKey  string
		onlyOnWindows bool
	}{
		{
			name:        "exact match - PATH",
			env:         []string{"HOME=/home/user", "PATH=/usr/bin:/bin", "USER=testuser"},
			key:         "PATH",
			expectedIdx: 1,
			expectedKey: "PATH",
		},
		{
			name:        "exact match - HOME",
			env:         []string{"HOME=/home/user", "PATH=/usr/bin", "USER=testuser"},
			key:         "HOME",
			expectedIdx: 0,
			expectedKey: "HOME",
		},
		{
			name:        "not found",
			env:         []string{"HOME=/home/user", "USER=testuser"},
			key:         "PATH",
			expectedIdx: -1,
			expectedKey: "",
		},
		{
			name:          "case-insensitive match - Path (Windows only)",
			env:           []string{"Path=C:\\Windows\\System32", "HOME=/home/user"},
			key:           "PATH",
			expectedIdx:   0,
			expectedKey:   "Path",
			onlyOnWindows: true,
		},
		{
			name:          "case-insensitive match - path (Windows only)",
			env:           []string{"path=C:\\Windows\\System32", "HOME=/home/user"},
			key:           "PATH",
			expectedIdx:   0,
			expectedKey:   "path",
			onlyOnWindows: true,
		},
		{
			name:          "case-insensitive match - PaTh (Windows only)",
			env:           []string{"HOME=/home/user", "PaTh=C:\\Windows\\System32"},
			key:           "PATH",
			expectedIdx:   1,
			expectedKey:   "PaTh",
			onlyOnWindows: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.onlyOnWindows && !utils.IsWindows() {
				t.Skip("Windows-only test")
			}

			idx, key := findEnvVar(tt.env, tt.key)
			assert.Equal(t, tt.expectedIdx, idx, "Index mismatch")
			assert.Equal(t, tt.expectedKey, key, "Key mismatch")
		})
	}
}

// TestSetEnvVar tests setting environment variables with case-insensitive handling on Windows
func TestSetEnvVar(t *testing.T) {
	tests := []struct {
		name          string
		env           []string
		key           string
		value         string
		expected      []string
		onlyOnWindows bool
	}{
		{
			name:     "add new variable",
			env:      []string{"HOME=/home/user", "USER=testuser"},
			key:      "PATH",
			value:    "/usr/bin:/bin",
			expected: []string{"HOME=/home/user", "USER=testuser", "PATH=/usr/bin:/bin"},
		},
		{
			name:     "update existing variable - exact match",
			env:      []string{"HOME=/home/user", "PATH=/usr/bin", "USER=testuser"},
			key:      "PATH",
			value:    "/new/path:/usr/bin",
			expected: []string{"HOME=/home/user", "PATH=/new/path:/usr/bin", "USER=testuser"},
		},
		{
			name:          "update existing variable - case mismatch on Windows",
			env:           []string{"Path=C:\\Windows\\System32", "HOME=/home/user"},
			key:           "PATH",
			value:         "C:\\goenv\\bin;C:\\Windows\\System32",
			expected:      []string{"Path=C:\\goenv\\bin;C:\\Windows\\System32", "HOME=/home/user"},
			onlyOnWindows: true,
		},
		{
			name:          "update existing variable - mixed case (Windows)",
			env:           []string{"HOME=/home/user", "PaTh=C:\\Windows\\System32"},
			key:           "PATH",
			value:         "C:\\goenv\\bin;C:\\Windows\\System32",
			expected:      []string{"HOME=/home/user", "PaTh=C:\\goenv\\bin;C:\\Windows\\System32"},
			onlyOnWindows: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.onlyOnWindows && !utils.IsWindows() {
				t.Skip("Windows-only test")
			}

			result := setEnvVar(tt.env, tt.key, tt.value)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestPrependToPath tests prepending directories to PATH with case-insensitive handling on Windows
func TestPrependToPath(t *testing.T) {
	tests := []struct {
		name          string
		env           []string
		dir           string
		expected      []string
		onlyOnWindows bool
	}{
		{
			name:     "prepend to existing PATH - Unix",
			env:      []string{"HOME=/home/user", "PATH=/usr/bin:/bin", "USER=testuser"},
			dir:      "/opt/goenv/bin",
			expected: []string{"HOME=/home/user", "PATH=/opt/goenv/bin:/usr/bin:/bin", "USER=testuser"},
		},
		{
			name:     "add PATH when missing",
			env:      []string{"HOME=/home/user", "USER=testuser"},
			dir:      "/opt/goenv/bin",
			expected: []string{"HOME=/home/user", "USER=testuser", "PATH=/opt/goenv/bin"},
		},
		{
			name:          "prepend to existing Path - Windows (case mismatch)",
			env:           []string{"Path=C:\\Windows\\System32;C:\\Windows", "HOME=C:\\Users\\test"},
			dir:           "C:\\goenv\\bin",
			expected:      []string{"Path=C:\\goenv\\bin;C:\\Windows\\System32;C:\\Windows", "HOME=C:\\Users\\test"},
			onlyOnWindows: true,
		},
		{
			name:          "prepend to existing path - Windows (lowercase)",
			env:           []string{"HOME=C:\\Users\\test", "path=C:\\Windows\\System32"},
			dir:           "C:\\goenv\\bin",
			expected:      []string{"HOME=C:\\Users\\test", "path=C:\\goenv\\bin;C:\\Windows\\System32"},
			onlyOnWindows: true,
		},
		{
			name:          "prepend to existing PaTh - Windows (mixed case)",
			env:           []string{"PaTh=C:\\Windows\\System32", "USER=testuser"},
			dir:           "C:\\goenv\\bin",
			expected:      []string{"PaTh=C:\\goenv\\bin;C:\\Windows\\System32", "USER=testuser"},
			onlyOnWindows: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.onlyOnWindows && !utils.IsWindows() {
				t.Skip("Windows-only test")
			}

			result := prependToPath(tt.env, tt.dir)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// TestWindowsPathPreservation is a regression test for issue #557
// It verifies that the original system PATH is preserved on Windows when goenv exec
// prepends its own paths, even when PATH has different casing (Path, path, PATH, etc.)
func TestWindowsPathPreservation(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows-only test")
	}

	tests := []struct {
		name     string
		env      []string
		dir      string
		checkFor string // a path component that should still be present after prepending
	}{
		{
			name:     "Path with C:\\TDM-GCC-64\\bin (issue #557 scenario)",
			env:      []string{"Path=C:\\Windows\\System32;C:\\TDM-GCC-64\\bin;C:\\Windows"},
			dir:      "C:\\Users\\test\\.goenv\\versions\\1.24.0\\bin",
			checkFor: "C:\\TDM-GCC-64\\bin",
		},
		{
			name:     "PATH with system paths",
			env:      []string{"PATH=C:\\Windows\\System32;C:\\Program Files\\Git\\cmd"},
			dir:      "C:\\goenv\\bin",
			checkFor: "C:\\Program Files\\Git\\cmd",
		},
		{
			name:     "path (lowercase) with MinGW",
			env:      []string{"path=C:\\msys64\\mingw64\\bin;C:\\Windows\\System32"},
			dir:      "C:\\goenv\\bin",
			checkFor: "C:\\msys64\\mingw64\\bin",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := prependToPath(tt.env, tt.dir)

			// Find the PATH variable in the result (case-insensitive)
			var pathValue string
			for _, envVar := range result {
				if idx := strings.Index(envVar, "="); idx != -1 {
					key := envVar[:idx]
					if strings.EqualFold(key, "PATH") {
						pathValue = envVar[idx+1:]
						break
					}
				}
			}

			require.NotEmpty(t, pathValue, "PATH should be present in result")

			// Verify the new directory was prepended
			assert.True(t, strings.HasPrefix(pathValue, tt.dir+";"), 
				"PATH should start with new directory: got %s", pathValue)

			// Verify the original path component is still present (regression test for #557)
			assert.Contains(t, pathValue, tt.checkFor,
				"Original path component %s should be preserved in PATH: got %s", tt.checkFor, pathValue)
		})
	}
}
