package core

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestUninstallCommand(t *testing.T) {
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
			expectedError: "Usage: goenv uninstall <version>",
		},
		{
			name:          "too many arguments provided",
			args:          []string{"1.21.0", "1.22.0"},
			expectedError: "Usage: goenv uninstall <version>",
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
			os.Setenv("GOENV_ROOT", tmpDir)
			defer os.Unsetenv("GOENV_ROOT")

			// Set GOENV_DIR to tmpDir to prevent FindVersionFile from looking in parent directories
			os.Setenv("GOENV_DIR", tmpDir)
			defer os.Unsetenv("GOENV_DIR")

			// Change to tmpDir
			oldDir, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(oldDir)

			// Set current version if specified
			if tt.currentVersion != "" {
				os.Setenv("GOENV_VERSION", tt.currentVersion)
				defer os.Unsetenv("GOENV_VERSION")
			}

			// Setup versions
			for _, version := range tt.setupVersions {
				versionPath := filepath.Join(tmpDir, "versions", version)
				goPath := filepath.Join(versionPath, "go")

				if err := os.MkdirAll(goPath, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}

				// Create a marker file to verify existence
				markerFile := filepath.Join(versionPath, ".installed")
				if err := os.WriteFile(markerFile, []byte("installed"), 0644); err != nil {
					t.Fatalf("Failed to create marker file: %v", err)
				}
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
			err := runUninstall(uninstallCmd, tt.args)

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
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got:\n%s", tt.expectedOutput, output)
				}
			}

			// Check if version still exists or not
			if tt.versionToCheck != "" {
				versionPath := filepath.Join(tmpDir, "versions", tt.versionToCheck)
				_, err := os.Stat(versionPath)
				exists := err == nil

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
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()

	// Check for key help text elements
	expectedStrings := []string{
		"uninstall",
		"Remove an installed Go version",
		"<version>",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q, got:\n%s", expected, output)
		}
	}
}

func TestUninstallCompletion(t *testing.T) {
	tmpDir := t.TempDir()
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Unsetenv("GOENV_ROOT")

	// Set GOENV_DIR to prevent looking in parent directories
	os.Setenv("GOENV_DIR", tmpDir)
	defer os.Unsetenv("GOENV_DIR")

	// Change to tmpDir
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(oldDir)

	// Setup some versions
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, version := range versions {
		versionPath := filepath.Join(tmpDir, "versions", version)
		binPath := filepath.Join(versionPath, "bin")

		if err := os.MkdirAll(binPath, 0755); err != nil {
			t.Fatalf("Failed to create bin directory: %v", err)
		}

		// Create go binary for version detection
		goExe := filepath.Join(binPath, "go")
		content := []byte("#!/bin/sh\necho mock go")
		if runtime.GOOS == "windows" {
			goExe += ".bat"
			content = []byte("@echo off\necho mock go")
		}
		if err := os.WriteFile(goExe, content, 0755); err != nil {
			t.Fatalf("Failed to create go binary: %v", err)
		}
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
	err := runUninstall(uninstallCmd, []string{})
	if err != nil {
		t.Fatalf("Completion mode failed: %v", err)
	}

	output := buf.String()

	// Check that all versions are listed
	for _, version := range versions {
		if !strings.Contains(output, version) {
			t.Errorf("Expected completion output to contain %q, got:\n%s", version, output)
		}
	}

	// Reset flags
	uninstallFlags.complete = false
}
