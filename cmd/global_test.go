package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

func setupTestEnv(t *testing.T) (string, func()) {
	// Create temporary test directory
	testDir, err := os.MkdirTemp("", "goenv_test_")
	if err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

	// Set test environment variables
	oldGoenvRoot := utils.GoenvEnvVarRoot.UnsafeValue()
	oldHome := os.Getenv("HOME")
	oldPath := os.Getenv("PATH")
	oldGoenvVersion := utils.GoenvEnvVarVersion.UnsafeValue()

	testRoot := filepath.Join(testDir, "root")
	testHome := filepath.Join(testDir, "home")

	utils.GoenvEnvVarRoot.Set(testRoot)
	os.Setenv("HOME", testHome)
	// Clear PATH to ensure no system go is found unless explicitly added by test
	if runtime.GOOS == "windows" {
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

func createTestVersion(t *testing.T, root, version string) {
	versionDir := filepath.Join(root, "versions", version)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create test version directory: %v", err)
	}

	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create test bin directory: %v", err)
	}

	// Create mock go binary
	goBin := filepath.Join(binDir, "go")
	var content string
	if runtime.GOOS == "windows" {
		goBin += ".bat"
		content = "@echo off\necho go version go" + version + " windows/amd64\n"
	} else {
		content = "#!/bin/sh\necho go version go" + version + " linux/amd64\n"
	}

	if err := os.WriteFile(goBin, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create test go binary: %v", err)
	}
}

func createTestAlias(t *testing.T, root, name, target string) {
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

func TestGlobalCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		globalVersion  string
		expectedOutput string
		expectedError  string
	}{
		{
			name:           "show default system version when no global set",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			expectedOutput: "system",
		},
		{
			name:           "show global version when set",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: "1.21.5",
		},
		{
			name:          "set valid global version",
			args:          []string{"1.21.5"},
			setupVersions: []string{"1.21.5", "1.22.2"},
			globalVersion: "system", // initial
		},
		{
			name:          "set system as global version",
			args:          []string{"system"},
			setupVersions: []string{"1.21.5"},
		},
		{
			name:          "error on invalid version",
			args:          []string{"invalid.version"},
			setupVersions: []string{"1.21.5"},
			expectedError: "version 'invalid.version' not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup test versions
			for _, version := range tt.setupVersions {
				createTestVersion(t, testRoot, version)
			}

			// Set initial global version if specified
			if tt.globalVersion != "" && tt.globalVersion != "system" {
				globalFile := filepath.Join(testRoot, "version")
				err := os.WriteFile(globalFile, []byte(tt.globalVersion), 0644)
				if err != nil {
					t.Fatalf("Failed to set initial global version: %v", err)
				}
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "global",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runGlobal(cmd, args)
				},
			}

			// Capture output
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
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
				got := strings.TrimSpace(stdout.String())
				if got != tt.expectedOutput {
					t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, got)
				}
			}

			// For set operations, verify the file was written correctly
			if len(tt.args) > 0 && tt.args[0] != "__complete" && tt.expectedError == "" {
				globalFile := filepath.Join(testRoot, "version")
				content, err := os.ReadFile(globalFile)
				if err != nil {
					t.Errorf("Failed to read global version file: %v", err)
					return
				}

				expected := tt.args[0]
				if strings.TrimSpace(string(content)) != expected {
					t.Errorf("Expected global version file to contain '%s', got '%s'",
						expected, strings.TrimSpace(string(content)))
				}
			}
		})
	}
}

func TestGlobalUsage(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	cmd := &cobra.Command{
		Use: "global",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGlobal(cmd, args)
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

func TestGlobalWithLocalOverride(t *testing.T) {
	testRoot, cleanup := setupTestEnv(t)
	defer cleanup()

	_ = testRoot // Use the variable

	// Setup versions
	createTestVersion(t, testRoot, "1.21.5")
	createTestVersion(t, testRoot, "1.22.2")

	// Set global version
	globalFile := filepath.Join(testRoot, "version")
	err := os.WriteFile(globalFile, []byte("1.21.5"), 0644)
	if err != nil {
		t.Fatalf("Failed to set global version: %v", err)
	}

	// Create local version file in current directory
	tempDir, err := os.MkdirTemp("", "goenv_local_test_")
	if err != nil {
		t.Fatalf("Failed to create temp directory: %v", err)
	}
	defer os.RemoveAll(tempDir)

	localFile := filepath.Join(tempDir, ".go-version")
	err = os.WriteFile(localFile, []byte("1.22.2"), 0644)
	if err != nil {
		t.Fatalf("Failed to create local version file: %v", err)
	}

	// Change to the directory with local version
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// Global command should still show global version
	cmd := &cobra.Command{
		Use: "global",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGlobal(cmd, args)
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Global command failed: %v", err)
	}

	got := strings.TrimSpace(output.String())
	if got != "1.21.5" {
		t.Errorf("Global command should show global version '1.21.5', got '%s'", got)
	}
}

func TestGlobalCommandRejectsExtraArguments(t *testing.T) {
	testRoot, cleanup := setupTestEnv(t)
	defer cleanup()

	// Setup test versions
	createTestVersion(t, testRoot, "1.21.5")
	createTestVersion(t, testRoot, "1.22.2")

	// Try to set global with extra arguments
	cmd := &cobra.Command{
		Use: "global",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runGlobal(cmd, args)
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"1.21.5", "extra"})

	err := cmd.Execute()

	// Should error with usage message
	if err == nil {
		t.Error("Expected error when extra arguments provided, got nil")
		return
	}

	if !strings.Contains(err.Error(), "Usage: goenv global [version]") {
		t.Errorf("Expected usage error, got: %v", err)
	}
}
