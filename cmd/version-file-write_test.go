package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionFileWriteCommand(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		setupVersions    []string
		setupSystemGo    bool
		existingFile     string
		existingContent  string
		expectedOutput   string
		expectedError    string
		checkFileContent string
		checkFileRemoved bool
	}{
		{
			name:          "fails when fewer than 2 arguments specified (no args)",
			args:          []string{},
			expectedError: "requires at least 2 arg(s)",
		},
		{
			name:          "fails when fewer than 2 arguments specified (one arg)",
			args:          []string{"one"},
			expectedError: "requires at least 2 arg(s)",
		},
		{
			name:          "fails when version is non-existent",
			args:          []string{".go-version", "1.11.1"},
			expectedError: "goenv: version '1.11.1' not installed",
		},
		{
			name:             "writes version to file when version exists",
			args:             []string{"my-version", "1.11.1"},
			setupVersions:    []string{"1.11.1"},
			checkFileContent: "1.11.1\n",
		},
		{
			name:             "writes multiple versions to file",
			args:             []string{"my-version", "1.11.1", "1.10.3"},
			setupVersions:    []string{"1.11.1", "1.10.3"},
			checkFileContent: "1.11.1\n1.10.3\n",
		},
		{
			name:             "overwrites existing file content",
			args:             []string{"my-version", "1.11.1"},
			setupVersions:    []string{"1.11.1"},
			existingFile:     "my-version",
			existingContent:  "old-version\n",
			checkFileContent: "1.11.1\n",
		},
		{
			name:             "removes local version when system version is given and system exists",
			args:             []string{".go-version", "system"},
			setupSystemGo:    true,
			existingFile:     ".go-version",
			existingContent:  "1.2.3\n",
			expectedOutput:   "goenv: using system version instead of 1.2.3 now\n",
			checkFileRemoved: true,
		},
		{
			name:            "fails to set system when system Go not found in PATH",
			args:            []string{".go-version", "system"},
			setupSystemGo:   false,
			existingFile:    ".go-version",
			existingContent: "1.2.3\n",
			expectedError:   "goenv: system version not found in PATH",
		},
		{
			name:             "removes file when system version given and file has multi-line versions",
			args:             []string{".go-version", "system"},
			setupSystemGo:    true,
			existingFile:     ".go-version",
			existingContent:  "1.11.1\n1.10.3\n",
			expectedOutput:   "goenv: using system version instead of 1.11.1:1.10.3 now\n",
			checkFileRemoved: true,
		},
		{
			name:             "removes file silently when system version given and no existing file",
			args:             []string{".go-version", "system"},
			setupSystemGo:    true,
			checkFileRemoved: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goenvRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup test versions
			for _, version := range tt.setupVersions {
				createTestVersion(t, goenvRoot, version)
			}

			// Setup system Go if needed
			if tt.setupSystemGo {
				// Create a bin directory in PATH with go executable
				binDir := filepath.Join(goenvRoot, "system-bin")
				if err := os.MkdirAll(binDir, 0755); err != nil {
					t.Fatalf("Failed to create bin directory: %v", err)
				}
				goExec := filepath.Join(binDir, "go")
				if err := os.WriteFile(goExec, []byte("#!/bin/sh\necho go version go1.21.0 linux/amd64\n"), 0755); err != nil {
					t.Fatalf("Failed to create go executable: %v", err)
				}
				// Add to PATH
				originalPath := os.Getenv("PATH")
				os.Setenv("PATH", binDir+":"+originalPath)
				defer os.Setenv("PATH", originalPath)
			}

			// Create existing file if specified
			var testFilePath string
			if tt.existingFile != "" {
				testFilePath = filepath.Join(goenvRoot, tt.existingFile)
				if err := os.WriteFile(testFilePath, []byte(tt.existingContent), 0644); err != nil {
					t.Fatalf("Failed to create existing file: %v", err)
				}
			}

			// Prepare args with full paths
			args := make([]string, len(tt.args))
			for i, arg := range tt.args {
				if i == 0 {
					// First arg is the filename
					args[i] = filepath.Join(goenvRoot, arg)
					if testFilePath == "" {
						testFilePath = args[i]
					}
				} else {
					// Rest are version arguments
					args[i] = arg
				}
			}

			// Execute command
			cmd := &cobra.Command{
				Use: "version-file-write",
				RunE: func(cmd *cobra.Command, cmdArgs []string) error {
					return runVersionFileWrite(cmd, cmdArgs)
				},
				Args:         cobra.MinimumNArgs(2),
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs(args)

			err := cmd.Execute()

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check output
			got := output.String()
			if tt.expectedOutput != "" && got != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, got)
			}

			// Check file content
			if tt.checkFileContent != "" {
				content, err := os.ReadFile(testFilePath)
				if err != nil {
					t.Errorf("Failed to read file: %v", err)
				} else if string(content) != tt.checkFileContent {
					t.Errorf("Expected file content %q, got %q", tt.checkFileContent, string(content))
				}
			}

			// Check file removal
			if tt.checkFileRemoved {
				if _, err := os.Stat(testFilePath); !os.IsNotExist(err) {
					t.Errorf("Expected file to be removed, but it still exists")
				}
			}
		})
	}
}
