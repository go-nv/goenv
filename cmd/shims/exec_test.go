package shims

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

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
			expectedError: "Usage: goenv exec <command> [arg1 arg2...]",
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
			oldGoenvVersion := os.Getenv("GOENV_VERSION")
			os.Unsetenv("GOENV_VERSION")
			defer func() {
				if oldGoenvVersion != "" {
					os.Setenv("GOENV_VERSION", oldGoenvVersion)
				}
			}()

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, testRoot, version)

				// Create custom binaries if specified
				if content, exists := tt.createBinaries[version]; exists {
					binPath := filepath.Join(testRoot, "versions", version, "bin", "go")
					if runtime.GOOS == "windows" {
						binPath += ".bat"
						// Convert Unix shell script to Windows batch file
						content = strings.ReplaceAll(content, "#!/bin/sh\n", "@echo off\n")
						// Convert Unix shell variable expansion to Windows batch
						content = strings.ReplaceAll(content, "$@", "%*")
						// Windows batch echo includes quotes in output, so strip them
						content = strings.ReplaceAll(content, `echo "`, `echo `)
						content = strings.ReplaceAll(content, `"`+"\n", "\n")
					}
					err := os.WriteFile(binPath, []byte(content), 0755)
					if err != nil {
						t.Fatalf("Failed to create custom binary for version %s: %v", version, err)
					}
				}
			}

			// Set global version if specified
			if tt.globalVersion != "" {
				globalFile := filepath.Join(testRoot, "version")
				err := os.WriteFile(globalFile, []byte(tt.globalVersion), 0644)
				if err != nil {
					t.Fatalf("Failed to set global version: %v", err)
				}
			}

			// Setup local version if specified
			var tempDir string
			if tt.localVersion != "" {
				var err error
				tempDir, err = os.MkdirTemp("", "goenv_exec_test_")
				if err != nil {
					t.Fatalf("Failed to create temp directory: %v", err)
				}
				defer os.RemoveAll(tempDir)

				localFile := filepath.Join(tempDir, ".go-version")
				err = os.WriteFile(localFile, []byte(tt.localVersion), 0644)
				if err != nil {
					t.Fatalf("Failed to create local version file: %v", err)
				}

				// Change to the directory with local version
				oldDir, _ := os.Getwd()
				defer os.Chdir(oldDir)
				os.Chdir(tempDir)
			}

			// Setup environment version if specified
			if tt.envVersion != "" {
				oldEnvVersion := os.Getenv("GOENV_VERSION")
				os.Setenv("GOENV_VERSION", tt.envVersion)
				defer os.Setenv("GOENV_VERSION", oldEnvVersion)
			}

			// Setup system go for system version test
			if tt.globalVersion == "system" {
				systemBinDir := filepath.Join(testRoot, "system_bin")
				os.MkdirAll(systemBinDir, 0755)
				systemGo := filepath.Join(systemBinDir, "go")
				var content string
				if runtime.GOOS == "windows" {
					systemGo += ".bat"
					content = "@echo off\necho system go version\n"
				} else {
					content = "#!/bin/sh\necho system go version\n"
				}

				err := os.WriteFile(systemGo, []byte(content), 0755)
				if err != nil {
					t.Fatalf("Failed to create system go: %v", err)
				}

				// Add to PATH temporarily
				oldPath := os.Getenv("PATH")
				pathSep := string(os.PathListSeparator)
				os.Setenv("PATH", systemBinDir+pathSep+oldPath)
				defer os.Setenv("PATH", oldPath)
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

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectedOutput != "" {
				got := strings.TrimSpace(output.String())
				if got != tt.expectedOutput {
					t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, got)
				}
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
	if err != nil {
		t.Errorf("Help command failed: %v", err)
	}

	helpOutput := output.String()
	if !strings.Contains(helpOutput, "Usage:") {
		t.Error("Help output should contain usage information")
	}
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
	if runtime.GOOS == "windows" {
		binPath += ".bat"
		content = "@echo off\necho GOROOT=%GOROOT%\necho GOPATH=%GOPATH%\n"
	} else {
		content = "#!/bin/sh\necho \"GOROOT=$GOROOT\"\necho \"GOPATH=$GOPATH\"\n"
	}

	err := os.WriteFile(binPath, []byte(content), 0755)
	if err != nil {
		t.Fatalf("Failed to create env-test binary: %v", err)
	}

	// Set global version
	globalFile := filepath.Join(testRoot, "version")
	err = os.WriteFile(globalFile, []byte(version), 0644)
	if err != nil {
		t.Fatalf("Failed to set global version: %v", err)
	}

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

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Exec command failed: %v", err)
	}

	got := output.String()

	// Should set GOROOT to the version directory
	expectedGoroot := filepath.Join(testRoot, "versions", version)
	if !strings.Contains(got, "GOROOT="+expectedGoroot) {
		t.Errorf("Expected GOROOT to be set to '%s', output: %s", expectedGoroot, got)
	}
}

func TestExecWithShims(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test version
	version := "1.21.5"
	cmdtest.CreateTestVersion(t, testRoot, version)

	// Set global version
	globalFile := filepath.Join(testRoot, "version")
	err := os.WriteFile(globalFile, []byte(version), 0644)
	if err != nil {
		t.Fatalf("Failed to set global version: %v", err)
	}

	// Create shims directory and shim
	shimsDir := filepath.Join(testRoot, "shims")
	err = os.MkdirAll(shimsDir, 0755)
	if err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}

	shimPath := filepath.Join(shimsDir, "go")
	var shimContent string
	if runtime.GOOS == "windows" {
		shimPath += ".bat"
		shimContent = "@echo off\ngoenv exec go %*\n"
	} else {
		shimContent = "#!/usr/bin/env bash\nexec goenv exec \"$(basename \"$0\")\" \"$@\"\n"
	}
	err = os.WriteFile(shimPath, []byte(shimContent), 0755)
	if err != nil {
		t.Fatalf("Failed to create shim: %v", err)
	}

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
	if err != nil {
		t.Errorf("Exec command failed: %v", err)
	}

	got := strings.TrimSpace(output.String())
	if !strings.Contains(got, "go1.21.5") {
		t.Errorf("Expected output to contain version info, got: %s", got)
	}
}
