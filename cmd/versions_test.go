package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionsCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		globalVersion  string
		expectSystemGo bool
		expectedOutput []string
		expectedError  string
	}{
		{
			name:           "list installed versions with current marked",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2", "1.23.0"},
			globalVersion:  "1.22.2",
			expectedOutput: []string{"  1.21.5", "* 1.22.2 (set by global)", "  1.23.0"},
		},
		{
			name:           "list with system version as current",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			expectSystemGo: true,
			expectedOutput: []string{"* system (set by global)", "  1.21.5", "  1.22.2"},
		},
		{
			name:           "bare output without indicators",
			args:           []string{"--bare"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: []string{"1.21.5", "1.22.2"},
		},
		{
			name:           "skip aliases flag (no change in Go version)",
			args:           []string{"--skip-aliases"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: []string{"* 1.21.5 (set by global)", "  1.22.2"},
		},
		{
			name:           "bare and skip aliases combined",
			args:           []string{"--bare", "--skip-aliases"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			expectedOutput: []string{"1.21.5", "1.22.2"},
		},
		{
			name:          "error with invalid arguments",
			args:          []string{"invalid", "args"},
			expectedError: "Usage:",
		},
		{
			name:           "completion support",
			args:           []string{"--complete"},
			expectedOutput: []string{"--bare", "--skip-aliases"},
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

			// Set global version if specified
			if tt.globalVersion != "" {
				globalFile := filepath.Join(testRoot, "version")
				err := os.WriteFile(globalFile, []byte(tt.globalVersion), 0644)
				if err != nil {
					t.Fatalf("Failed to set global version: %v", err)
				}
			}

			// Setup system go if needed
			if tt.expectSystemGo {
				// Create a mock system go in PATH
				systemBinDir := filepath.Join(testRoot, "system_bin")
				os.MkdirAll(systemBinDir, 0755)
				systemGo := filepath.Join(systemBinDir, "go")
				err := os.WriteFile(systemGo, []byte("#!/bin/sh\necho go version go1.20.1 linux/amd64\n"), 0755)
				if err != nil {
					t.Fatalf("Failed to create system go: %v", err)
				}

				// Add to PATH temporarily
				oldPath := os.Getenv("PATH")
				os.Setenv("PATH", systemBinDir+":"+oldPath)
				defer os.Setenv("PATH", oldPath)
			}

			// Reset global flags before each test
			versionsFlags.bare = false
			versionsFlags.skipAliases = false
			versionsFlags.complete = false

			// Create and execute command
			cmd := &cobra.Command{
				Use: "versions",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runVersions(cmd, args)
				},
			}

			// Add flags bound to global struct (same as real command)
			cmd.Flags().BoolVar(&versionsFlags.bare, "bare", false, "Display bare version numbers only")
			cmd.Flags().BoolVar(&versionsFlags.skipAliases, "skip-aliases", false, "Skip aliases")
			cmd.Flags().BoolVar(&versionsFlags.complete, "complete", false, "Internal flag for shell completions")
			_ = cmd.Flags().MarkHidden("complete")

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

			if len(tt.expectedOutput) > 0 {
				got := strings.TrimRight(output.String(), "\n")
				gotLines := strings.Split(got, "\n")

				if len(gotLines) != len(tt.expectedOutput) {
					t.Errorf("Expected %d lines, got %d:\nExpected:\n%s\nGot:\n%s",
						len(tt.expectedOutput), len(gotLines),
						strings.Join(tt.expectedOutput, "\n"), got)
					return
				}

				for i, expectedLine := range tt.expectedOutput {
					if i >= len(gotLines) {
						t.Errorf("Missing expected line %d: '%s'", i, expectedLine)
						continue
					}
					if gotLines[i] != expectedLine {
						t.Errorf("Line %d: expected '%s', got '%s' (bare=%v)",
							i, expectedLine, gotLines[i], versionsFlags.bare)
					}
				}
			}
		})
	}
}

func TestVersionsWithLocalVersion(t *testing.T) {
	testRoot, cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset global flags
	versionsFlags.bare = false
	versionsFlags.skipAliases = false
	versionsFlags.complete = false

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

	// Create and execute command
	cmd := &cobra.Command{
		Use: "versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersions(cmd, args)
		},
	}

	// Add flags bound to global struct
	cmd.Flags().BoolVar(&versionsFlags.bare, "bare", false, "Display bare version numbers only")
	cmd.Flags().BoolVar(&versionsFlags.skipAliases, "skip-aliases", false, "Skip aliases")
	cmd.Flags().BoolVar(&versionsFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = cmd.Flags().MarkHidden("complete")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Versions command failed: %v", err)
	}

	got := strings.TrimRight(output.String(), "\n")
	gotLines := strings.Split(got, "\n")

	// Should show local version (1.22.2) as current with source, not global (1.21.5)
	// The source path will be the tempDir/.go-version
	if len(gotLines) != 2 {
		t.Errorf("Expected 2 lines, got %d:\nGot:\n%s", len(gotLines), got)
		return
	}

	// Check line 0: non-current version
	if gotLines[0] != "  1.21.5" {
		t.Errorf("Line 0: expected '  1.21.5', got '%s'", gotLines[0])
	}

	// Check line 1: current version with suffix
	expectedPrefix := "* 1.22.2 (set by "
	expectedSuffix := "/.go-version)"
	if !strings.HasPrefix(gotLines[1], expectedPrefix) || !strings.HasSuffix(gotLines[1], expectedSuffix) {
		t.Errorf("Line 1: expected to match '* 1.22.2 (set by .../.go-version)', got '%s'", gotLines[1])
	}
}

func TestVersionsNoVersionsInstalled(t *testing.T) {
	_, cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset global flags
	versionsFlags.bare = false
	versionsFlags.skipAliases = false
	versionsFlags.complete = false

	// Don't create any versions, no system go either

	// Create and execute command
	cmd := &cobra.Command{
		Use: "versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersions(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&versionsFlags.bare, "bare", false, "Display bare version numbers only")
	cmd.Flags().BoolVar(&versionsFlags.skipAliases, "skip-aliases", false, "Skip aliases")
	cmd.Flags().BoolVar(&versionsFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = cmd.Flags().MarkHidden("complete")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	// Should fail with warning when no versions installed and no system go
	if err == nil {
		t.Error("Expected error when no versions installed and no system go")
		return
	}

	if !strings.Contains(err.Error(), "Warning: no Go detected") {
		t.Errorf("Expected 'Warning: no Go detected' error, got: %v", err)
	}
}

func TestVersionsSystemGoOnly(t *testing.T) {
	testRoot, cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset global flags
	versionsFlags.bare = false
	versionsFlags.skipAliases = false
	versionsFlags.complete = false

	// Create system go but no installed versions
	systemBinDir := filepath.Join(testRoot, "system_bin")
	os.MkdirAll(systemBinDir, 0755)
	systemGo := filepath.Join(systemBinDir, "go")
	err := os.WriteFile(systemGo, []byte("#!/bin/sh\necho go version go1.20.1 linux/amd64\n"), 0755)
	if err != nil {
		t.Fatalf("Failed to create system go: %v", err)
	}

	// Add to PATH temporarily
	oldPath := os.Getenv("PATH")
	os.Setenv("PATH", systemBinDir+":"+oldPath)
	defer os.Setenv("PATH", oldPath)

	// Create and execute command
	cmd := &cobra.Command{
		Use: "versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersions(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&versionsFlags.bare, "bare", false, "Display bare version numbers only")
	cmd.Flags().BoolVar(&versionsFlags.skipAliases, "skip-aliases", false, "Skip aliases")
	cmd.Flags().BoolVar(&versionsFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = cmd.Flags().MarkHidden("complete")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got := strings.TrimSpace(output.String())
	expectedPattern := "* system (set by"

	if !strings.Contains(got, expectedPattern) {
		t.Errorf("Expected output to contain '%s', got '%s'", expectedPattern, got)
	}
}
