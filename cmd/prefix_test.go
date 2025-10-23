package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestPrefixCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		localVersion   string
		envVersion     string
		setupSystemGo  bool
		expectedOutput string
		expectedError  string
		checkPrefix    bool // if true, check output starts with expectedOutput
	}{
		{
			name:           "returns version path from local .go-version file",
			setupVersions:  []string{"1.2.3"},
			localVersion:   "1.2.3",
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.3",
		},
		{
			name:           "returns version path when major.minor given",
			args:           []string{"1.2"},
			setupVersions:  []string{"1.2.3"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.3",
		},
		{
			name:          "fails when no version exists and version is system without go in PATH",
			args:          []string{"system"},
			expectedError: "system version not found in PATH",
		},
		{
			name:          "returns system go directory when system version and go in PATH",
			args:          []string{"system"},
			setupSystemGo: true,
			checkPrefix:   true,
		},
		{
			name:          "fails when local version not installed",
			localVersion:  "1.2.3",
			expectedError: "is not installed",
		},
		{
			name:          "version from argument takes priority over GOENV_VERSION",
			args:          []string{"1.2.4"},
			envVersion:    "1.2.3",
			expectedError: "not installed",
		},
		{
			name:          "uses GOENV_VERSION when no arguments",
			envVersion:    "1.2.3",
			expectedError: "not installed",
		},
		{
			name:           "returns latest version when 'latest' specified",
			args:           []string{"latest"},
			setupVersions:  []string{"1.9.9", "1.9.10", "1.10.9", "1.10.10"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.10.10",
		},
		{
			name:           "returns latest major version when major number given",
			args:           []string{"1"},
			setupVersions:  []string{"1.2.9", "1.2.10", "4.5.6"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.10",
		},
		{
			name:          "fails when major version doesn't match any installed",
			args:          []string{"9"},
			setupVersions: []string{"1.2.9", "4.5.10"},
			expectedError: "not installed",
		},
		{
			name:           "returns latest minor version when single minor number given",
			args:           []string{"2"},
			setupVersions:  []string{"1.2.9", "1.2.10", "1.3.11", "4.5.2"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.10",
		},
		{
			name:           "returns latest patch when major.minor given",
			args:           []string{"1.2"},
			setupVersions:  []string{"1.1.2", "1.2.9", "1.2.10", "1.3.11", "2.1.2"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.10",
		},
		{
			name:          "fails when major.minor doesn't match any installed",
			args:          []string{"1.9"},
			setupVersions: []string{"1.1.9"},
			expectedError: "not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Reset global flags
			prefixFlags.complete = false

			// Setup test versions
			for _, version := range tt.setupVersions {
				createTestVersion(t, testRoot, version)
			}

			// Setup system go if needed
			var systemBinDir string
			oldPath := os.Getenv("PATH")
			defer os.Setenv("PATH", oldPath)

			if tt.setupSystemGo {
				systemBinDir = filepath.Join(testRoot, "system_bin")
				os.MkdirAll(systemBinDir, 0755)
				systemGo := filepath.Join(systemBinDir, "go")
				var content string
				if runtime.GOOS == "windows" {
					systemGo += ".bat"
					content = "@echo off\necho go version go1.20.1 windows/amd64\n"
				} else {
					content = "#!/bin/sh\necho go version go1.20.1 linux/amd64\n"
				}

				err := os.WriteFile(systemGo, []byte(content), 0755)
				if err != nil {
					t.Fatalf("Failed to create system go: %v", err)
				}

				// Add to PATH
				pathSep := string(os.PathListSeparator)
				os.Setenv("PATH", systemBinDir+pathSep+oldPath)
			} else {
				// Explicitly set PATH to empty directory to ensure no system Go found
				emptyDir := filepath.Join(testRoot, "empty-bin")
				os.MkdirAll(emptyDir, 0755)
				os.Setenv("PATH", emptyDir)
			} // Set local version if specified
			if tt.localVersion != "" {
				localFile := filepath.Join(testRoot, ".go-version")
				err := os.WriteFile(localFile, []byte(tt.localVersion), 0644)
				if err != nil {
					t.Fatalf("Failed to set local version: %v", err)
				}
				// Change to test root so local version is found
				oldDir, _ := os.Getwd()
				defer os.Chdir(oldDir)
				os.Chdir(testRoot)
			}

			// Set environment version if specified
			if tt.envVersion != "" {
				oldEnv := os.Getenv("GOENV_VERSION")
				os.Setenv("GOENV_VERSION", tt.envVersion)
				defer func() {
					if oldEnv != "" {
						os.Setenv("GOENV_VERSION", oldEnv)
					} else {
						os.Unsetenv("GOENV_VERSION")
					}
				}()
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "prefix",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runPrefix(cmd, args)
				},
			}

			// Add flags
			cmd.Flags().BoolVar(&prefixFlags.complete, "complete", false, "Show completion options")

			output := &strings.Builder{}
			cmd.SetOut(output)
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

			got := strings.TrimSpace(output.String())

			if tt.checkPrefix {
				if tt.expectedOutput != "" && !strings.Contains(got, tt.expectedOutput) {
					t.Errorf("Expected output to contain '%s', got '%s'", tt.expectedOutput, got)
				}
				if tt.setupSystemGo && systemBinDir != "" {
					// For system go, should return parent of bin dir
					expectedDir := filepath.Dir(systemBinDir)
					if !strings.Contains(got, expectedDir) {
						t.Errorf("Expected output to contain system dir '%s', got '%s'", expectedDir, got)
					}
				}
			} else {
				if got != tt.expectedOutput {
					t.Errorf("Expected '%s', got '%s'", tt.expectedOutput, got)
				}
			}
		})
	}
}

func TestPrefixCompletion(t *testing.T) {
	testRoot, cleanup := setupTestEnv(t)
	defer cleanup()

	// Reset global flags
	prefixFlags.complete = false

	// Setup test versions
	createTestVersion(t, testRoot, "1.9.10")
	createTestVersion(t, testRoot, "1.10.9")

	// Create and execute command
	cmd := &cobra.Command{
		Use: "prefix",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPrefix(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&prefixFlags.complete, "complete", false, "Show completion options")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{"--complete"})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	got := strings.TrimSpace(output.String())
	gotLines := strings.Split(got, "\n")

	// Should include: latest, system, and all installed versions
	expectedLines := []string{"latest", "system", "1.10.9", "1.9.10"}

	if len(gotLines) != len(expectedLines) {
		t.Errorf("Expected %d lines, got %d:\n%s", len(expectedLines), len(gotLines), got)
		return
	}

	for i, expected := range expectedLines {
		if gotLines[i] != expected {
			t.Errorf("Line %d: expected '%s', got '%s'", i, expected, gotLines[i])
		}
	}
}
