package legacy

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

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
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, testRoot, version)
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
					return RunGlobal(cmd, args)
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
				got := cmdtest.StripDeprecationWarning(stdout.String())
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
	_, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	cmd := &cobra.Command{
		Use: "global",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGlobal(cmd, args)
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
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	_ = testRoot // Use the variable

	// Setup versions
	cmdtest.CreateTestVersion(t, testRoot, "1.21.5")
	cmdtest.CreateTestVersion(t, testRoot, "1.22.2")

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
			return RunGlobal(cmd, args)
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Global command failed: %v", err)
	}

	got := cmdtest.StripDeprecationWarning(output.String())
	if got != "1.21.5" {
		t.Errorf("Global command should show global version '1.21.5', got '%s'", got)
	}
}

func TestGlobalCommandRejectsExtraArguments(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test versions
	cmdtest.CreateTestVersion(t, testRoot, "1.21.5")
	cmdtest.CreateTestVersion(t, testRoot, "1.22.2")

	// Try to set global with extra arguments
	cmd := &cobra.Command{
		Use: "global",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunGlobal(cmd, args)
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

	if !strings.Contains(err.Error(), "usage: goenv global [version]") {
		t.Errorf("Expected usage error, got: %v", err)
	}
}
