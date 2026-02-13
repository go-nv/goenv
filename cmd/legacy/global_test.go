package legacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			if tt.globalVersion != "" && tt.globalVersion != manager.SystemVersion {
				globalFile := filepath.Join(testRoot, "version")
				testutil.WriteTestFile(t, globalFile, []byte(tt.globalVersion), utils.PermFileDefault, "Failed to set initial global version")
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

			assert.NoError(t, err)

			if tt.expectedOutput != "" {
				got := cmdtest.StripDeprecationWarning(stdout.String())
				assert.Equal(t, tt.expectedOutput, got)
			}

			// For set operations, verify the file was written correctly
			if len(tt.args) > 0 && tt.args[0] != "__complete" && tt.expectedError == "" {
				globalFile := filepath.Join(testRoot, "version")
				content, err := os.ReadFile(globalFile)
				assert.NoError(t, err, "Failed to read global version file")

				expected := tt.args[0]
				assert.Equal(t, expected, strings.TrimSpace(string(content)))
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
	assert.NoError(t, err, "Help command failed")

	helpOutput := output.String()
	assert.Contains(t, helpOutput, "Usage:", "Help output should contain usage information")
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
	testutil.WriteTestFile(t, globalFile, []byte("1.21.5"), utils.PermFileDefault, "Failed to set global version")

	// Create local version file in current directory
	tempDir, err := os.MkdirTemp("", "goenv_local_test_")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	localFile := filepath.Join(tempDir, ".go-version")
	testutil.WriteTestFile(t, localFile, []byte("1.22.2"), utils.PermFileDefault)

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
	assert.NoError(t, err, "Global command failed")

	got := cmdtest.StripDeprecationWarning(output.String())
	assert.Equal(t, "1.21.5", got, "Global command should show global version '1.21.5'")
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
	assert.Error(t, err, "Expected error when extra arguments provided, got nil")

	assert.Contains(t, err.Error(), "usage: goenv global [version]", "Expected usage error %v", err)
}
