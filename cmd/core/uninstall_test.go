package core

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUninstallCommand(t *testing.T) {
	var err error
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		currentVersion string
		expectedError  string
		expectedOutput string
		shouldExist    bool
		versionToCheck string
	}{
		{
			name:          "no arguments provided",
			args:          []string{},
			expectedError: "usage: goenv uninstall <version>",
		},
		{
			name:          "too many arguments provided",
			args:          []string{"1.21.0", "1.22.0"},
			expectedError: "usage: goenv uninstall <version>",
		},
		{
			name:           "uninstall non-existent version",
			args:           []string{"1.99.0"},
			setupVersions:  []string{"1.21.0"},
			expectedError:  "version 1.99.0 is not installed",
			shouldExist:    true,
			versionToCheck: "1.21.0",
		},
		{
			name:           "successful uninstall",
			args:           []string{"1.21.0"},
			setupVersions:  []string{"1.21.0", "1.22.0"},
			currentVersion: "1.22.0",
			expectedOutput: "Successfully uninstalled Go 1.21.0",
			shouldExist:    false,
			versionToCheck: "1.21.0",
		},
		{
			name:           "uninstall current version - allowed",
			args:           []string{"1.21.0"},
			setupVersions:  []string{"1.21.0", "1.22.0"},
			currentVersion: "1.21.0",
			expectedOutput: "Successfully uninstalled Go 1.21.0",
			shouldExist:    false,
			versionToCheck: "1.21.0",
		},
		{
			name:           "uninstall system version",
			args:           []string{"system"},
			setupVersions:  []string{"1.21.0"},
			currentVersion: "1.21.0",
			expectedError:  "version system is not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			// Set GOENV_DIR to tmpDir to prevent FindVersionFile from looking in parent directories
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
			// Set CI to disable interactive prompts in tests
			t.Setenv(utils.EnvVarCI, "true")

			// Change to tmpDir
			oldDir, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(oldDir)

			// Set current version if specified
			if tt.currentVersion != "" {
				t.Setenv(utils.GoenvEnvVarVersion.String(), tt.currentVersion)
			}

			// Setup versions
			for _, version := range tt.setupVersions {
				versionPath := filepath.Join(tmpDir, "versions", version)
				goPath := filepath.Join(versionPath, "go")

				err = utils.EnsureDirWithContext(goPath, "create test directory")
				require.NoError(t, err, "Failed to create version directory")

				// Create a marker file to verify existence
				markerFile := filepath.Join(versionPath, ".installed")
				testutil.WriteTestFile(t, markerFile, []byte("installed"), utils.PermFileDefault)
			}

			// Create command
			cmd := &cobra.Command{}
			cmd.SetArgs(tt.args)

			// Reset flags
			uninstallCmd.ResetFlags()
			uninstallCmd.Flags().BoolVar(&uninstallFlags.complete, "complete", false, "")
			_ = uninstallCmd.Flags().MarkHidden("complete")

			// Capture output
			buf := new(bytes.Buffer)
			uninstallCmd.SetOut(buf)
			uninstallCmd.SetErr(buf)

			// Also capture stdout for installer's fmt.Printf
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Execute
			err = runUninstall(uninstallCmd, tt.args)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			stdoutOutput, _ := io.ReadAll(r)

			// Combine outputs
			combinedOutput := buf.String() + string(stdoutOutput)

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			if tt.expectedOutput != "" {
				output := combinedOutput
				assert.Contains(t, output, tt.expectedOutput, "Expected output to contain , got:\\n %v %v", tt.expectedOutput, output)
			}

			// Check if version still exists or not
			if tt.versionToCheck != "" {
				versionPath := filepath.Join(tmpDir, "versions", tt.versionToCheck)
				exists := utils.PathExists(versionPath)

				if tt.shouldExist && !exists {
					t.Errorf("Expected version %s to still exist, but it doesn't", tt.versionToCheck)
				} else if !tt.shouldExist && exists {
					t.Errorf("Expected version %s to be removed, but it still exists", tt.versionToCheck)
				}
			}

			// Reset flags after each test
			uninstallFlags.complete = false
		})
	}
}

func TestUninstallHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := uninstallCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Get help text
	err := cmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()

	// Check for key help text elements
	expectedStrings := []string{
		"uninstall",
		"Remove an installed Go version",
		"<version>",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing , got:\\n %v %v", expected, output)
	}
}

func TestUninstallCompletion(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	// Set GOENV_DIR to prevent looking in parent directories
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Change to tmpDir
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Setup some versions
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, version := range versions {
		versionPath := filepath.Join(tmpDir, "versions", version)
		binPath := filepath.Join(versionPath, "bin")

		err = utils.EnsureDirWithContext(binPath, "create test directory")
		require.NoError(t, err, "Failed to create bin directory")

		// Create go binary for version detection
		goExe := filepath.Join(binPath, "go")
		content := []byte("#!/bin/sh\necho mock go")
		if utils.IsWindows() {
			goExe += ".bat"
			content = []byte("@echo off\necho mock go")
		}
		testutil.WriteTestFile(t, goExe, content, utils.PermFileExecutable)
	}

	// Create command with --complete flag
	cmd := &cobra.Command{}
	cmd.SetArgs([]string{})

	// Set completion flag
	uninstallCmd.ResetFlags()
	uninstallCmd.Flags().BoolVar(&uninstallFlags.complete, "complete", false, "")
	_ = uninstallCmd.Flags().MarkHidden("complete")
	uninstallCmd.Flags().Set("complete", "true")

	// Capture output
	buf := new(bytes.Buffer)
	uninstallCmd.SetOut(buf)
	uninstallCmd.SetErr(buf)

	// Execute
	err = runUninstall(uninstallCmd, []string{})
	require.NoError(t, err, "Completion mode failed")

	output := buf.String()

	// Check that all versions are listed
	for _, version := range versions {
		assert.Contains(t, output, version, "Expected completion output to contain , got:\\n %v %v", version, output)
	}

	// Reset flags
	uninstallFlags.complete = false
}
