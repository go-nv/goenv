package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestCleanCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		flags          []string
		expectError    bool
		expectContains []string
	}{
		{
			name:           "no args defaults to build",
			args:           []string{},
			flags:          []string{"--force"},
			expectError:    false,
			expectContains: []string{"build cache"},
		},
		{
			name:           "build target",
			args:           []string{"build"},
			flags:          []string{"--force"},
			expectError:    false,
			expectContains: []string{"build cache"},
		},
		{
			name:           "invalid target",
			args:           []string{"invalid"},
			flags:          []string{"--force"},
			expectError:    true,
			expectContains: []string{"invalid target"},
		},
		{
			name:           "too many arguments",
			args:           []string{"build", "extra"},
			flags:          []string{"--force"},
			expectError:    true,
			expectContains: []string{"too many arguments"},
		},
		{
			name:           "verbose flag",
			args:           []string{"build"},
			flags:          []string{"--force", "--verbose"},
			expectError:    false,
			expectContains: []string{"Cleaning build cache"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup test environment
			tmpDir := t.TempDir()
			setupTestGoVersion(t, tmpDir, "1.21.0")

			// Set GOENV_ROOT for this test
			oldRoot := os.Getenv("GOENV_ROOT")
			os.Setenv("GOENV_ROOT", tmpDir)
			defer os.Setenv("GOENV_ROOT", oldRoot)

			// Create .go-version file
			versionFile := filepath.Join(tmpDir, ".go-version")
			if err := os.WriteFile(versionFile, []byte("1.21.0\n"), 0644); err != nil {
				t.Fatalf("Failed to write .go-version: %v", err)
			}

			// Change to tmpDir so .go-version is found
			oldWd, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(oldWd)

			// Run command
			cmd := cleanCmd
			cmd.SetArgs(append(tt.args, tt.flags...))

			var stdout, stderr bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stderr)

			err := cmd.Execute()

			// Check error expectation
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output contains expected strings
			output := stdout.String() + stderr.String()
			for _, expected := range tt.expectContains {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, got:\n%s", expected, output)
				}
			}

			// Reset command for next test
			cmd.SetArgs(nil)
			cleanFlags = struct {
				force   bool
				verbose bool
			}{}
		})
	}
}

func TestCleanCommandCompletion(t *testing.T) {
	completions, directive := completeCleanTargets(nil, []string{}, "")

	expectedCompletions := []string{"build", "modcache", "all"}
	if len(completions) != len(expectedCompletions) {
		t.Errorf("Expected %d completions, got %d", len(expectedCompletions), len(completions))
	}

	for _, expected := range expectedCompletions {
		found := false
		for _, completion := range completions {
			if completion == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected completion %q not found", expected)
		}
	}

	if directive != 4 { // cobra.ShellCompDirectiveNoFileComp
		t.Errorf("Expected NoFileComp directive, got %d", directive)
	}
}

func TestCleanCommandNoCompletionAfterTarget(t *testing.T) {
	completions, _ := completeCleanTargets(nil, []string{"build"}, "")

	if len(completions) != 0 {
		t.Errorf("Expected no completions after target is provided, got %d", len(completions))
	}
}

func TestGetGoBinary(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		setupFunc   func(t *testing.T, tmpDir string) string
		expectError bool
	}{
		{
			name:    "valid version",
			version: "1.21.0",
			setupFunc: func(t *testing.T, tmpDir string) string {
				setupTestGoVersion(t, tmpDir, "1.21.0")
				return tmpDir
			},
			expectError: false,
		},
		{
			name:    "missing version",
			version: "9.99.99",
			setupFunc: func(t *testing.T, tmpDir string) string {
				return tmpDir
			},
			expectError: true,
		},
		{
			name:    "system version",
			version: "system",
			setupFunc: func(t *testing.T, tmpDir string) string {
				return tmpDir
			},
			expectError: false, // Assumes system go exists
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			root := tt.setupFunc(t, tmpDir)

			cfg := &config.Config{
				Root: root,
			}

			binary, err := getGoBinary(cfg, tt.version)

			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if !tt.expectError && binary == "" {
				t.Error("Expected non-empty binary path")
			}

			if tt.version != "system" && !tt.expectError {
				expectedPath := filepath.Join(root, "versions", tt.version, "bin", "go")
				if runtime.GOOS == "windows" {
					expectedPath += ".exe"
				}
				if binary != expectedPath {
					t.Errorf("Expected binary path %q, got %q", expectedPath, binary)
				}
			}
		})
	}
}

func TestJoinWithCommas(t *testing.T) {
	tests := []struct {
		name     string
		items    []string
		expected string
	}{
		{
			name:     "empty slice",
			items:    []string{},
			expected: "",
		},
		{
			name:     "single item",
			items:    []string{"build cache"},
			expected: "build cache",
		},
		{
			name:     "two items",
			items:    []string{"build cache", "module cache"},
			expected: "build cache and module cache",
		},
		{
			name:     "three items",
			items:    []string{"build cache", "module cache", "test cache"},
			expected: "build cache, module cache, and test cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := joinWithCommas(tt.items)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

// Helper function to setup a fake Go version for testing
func setupTestGoVersion(t *testing.T, root, version string) {
	versionPath := filepath.Join(root, "versions", version)
	binDir := filepath.Join(versionPath, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create a fake go binary
	goBinary := filepath.Join(binDir, "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}

	// Create a simple script that exits successfully
	script := "#!/bin/sh\nexit 0"
	if runtime.GOOS == "windows" {
		script = "@echo off\nexit 0"
	}

	if err := os.WriteFile(goBinary, []byte(script), 0755); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}
}
