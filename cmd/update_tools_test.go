package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestUpdateToolsCommand(t *testing.T) {
	tests := []struct {
		name           string
		setupVersion   string
		setupTools     []toolInfo
		flags          map[string]string
		expectedError  string
		expectedOutput string
	}{
		{
			name:          "system go version",
			setupVersion:  "system",
			expectedError: "cannot update tools for system Go version",
		},
		{
			name:          "go version not installed",
			setupVersion:  "1.21.0",
			expectedError: "Go version 1.21.0 is not installed",
		},
		{
			name:           "no tools installed",
			setupVersion:   "1.21.0",
			expectedOutput: "No Go tools installed yet",
		},
		{
			name:         "tools up to date",
			setupVersion: "1.21.0",
			setupTools: []toolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v1.0.0"},
			},
			expectedOutput: "Found 1 tool(s)",
		},
		{
			name:         "tool needs update - skipped without package path",
			setupVersion: "1.21.0",
			setupTools: []toolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			expectedOutput: "unknown package path, skipping",
		},
		{
			name:         "check mode",
			setupVersion: "1.21.0",
			setupTools: []toolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			flags: map[string]string{
				"check": "true",
			},
			expectedOutput: "All tools are up to date",
		},
		{
			name:         "dry-run mode",
			setupVersion: "1.21.0",
			setupTools: []toolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			flags: map[string]string{
				"dry-run": "true",
			},
			expectedOutput: "All tools are up to date",
		},
		{
			name:         "update specific tool",
			setupVersion: "1.21.0",
			setupTools: []toolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
				{name: "mockdelve", pkgPath: "github.com/go-delve/delve/cmd/dlv", version: "v1.0.0"},
			},
			flags: map[string]string{
				"tool": "mockgopls",
			},
			expectedOutput: "Found 1 tool(s)",
		},
		{
			name:         "tool not found",
			setupVersion: "1.21.0",
			setupTools: []toolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			flags: map[string]string{
				"tool": "nonexistent",
			},
			expectedError: "tool 'nonexistent' not found",
		},
		{
			name:         "tool without package path",
			setupVersion: "1.21.0",
			setupTools: []toolInfo{
				{name: "mocktool", pkgPath: "", version: "unknown"},
			},
			expectedOutput: "unknown package path, skipping",
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

			// Change to tmpDir to avoid picking up .go-version from test directory
			oldDir, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(oldDir)

			// Set GOENV_VERSION to control which version is active (only if not empty)
			if tt.setupVersion != "" {
				os.Setenv("GOENV_VERSION", tt.setupVersion)
				defer os.Unsetenv("GOENV_VERSION")
			}

			// Setup version directory if specified (but not for "not installed" test case)
			shouldCreateVersionDir := tt.setupVersion != "" && tt.setupVersion != "system" && tt.expectedError != "Go version 1.21.0 is not installed"
			if shouldCreateVersionDir {
				versionPath := filepath.Join(tmpDir, "versions", tt.setupVersion)

				// Create go binary directory
				goBinDir := filepath.Join(versionPath, "go", "bin")
				if err := os.MkdirAll(goBinDir, 0755); err != nil {
					t.Fatalf("Failed to create go bin directory: %v", err)
				}

				// Create mock go binary
				goBinary := filepath.Join(goBinDir, "go")
				if runtime.GOOS == "windows" {
					goBinary += ".exe"
				}
				// Mock go install that succeeds
				mockScript := "#!/bin/sh\nexit 0"
				if runtime.GOOS == "windows" {
					mockScript = "@echo off\nexit 0"
				}
				if err := os.WriteFile(goBinary, []byte(mockScript), 0755); err != nil {
					t.Fatalf("Failed to create go binary: %v", err)
				}

				// Create GOPATH/bin directory
				gopathBin := filepath.Join(versionPath, "gopath", "bin")
				if err := os.MkdirAll(gopathBin, 0755); err != nil {
					t.Fatalf("Failed to create GOPATH/bin: %v", err)
				}

				// Create tools
				for _, tool := range tt.setupTools {
					toolPath := filepath.Join(gopathBin, tool.name)
					if runtime.GOOS == "windows" {
						toolPath += ".exe"
					}

					// Create mock binary with module info
					mockContent := "mock tool binary"
					if err := os.WriteFile(toolPath, []byte(mockContent), 0755); err != nil {
						t.Fatalf("Failed to create tool %s: %v", tool.name, err)
					}

					// Note: Package path discovery requires `go version -m` which reads build info
					// For testing, tooldetect.ListInstalledTools will return tools without package paths
					// unless the binary was built with proper build info
				}
			} else if tt.setupVersion == "system" {
				// GOENV_VERSION is already set to "system" above
			}

			// Create command
			cmd := &cobra.Command{}
			cmd.SetArgs([]string{})

			// Reset and set flags
			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.tool, "tool", "", "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.version, "version", "latest", "")

			for key, value := range tt.flags {
				updateToolsCmd.Flags().Set(key, value)
			}

			// Capture output
			buf := new(bytes.Buffer)
			updateToolsCmd.SetOut(buf)
			updateToolsCmd.SetErr(buf)

			// Execute
			err := runUpdateTools(updateToolsCmd, []string{})

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
				output := buf.String()
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got:\n%s", tt.expectedOutput, output)
				}
			}

			// Reset flags after each test
			updateToolsFlags.check = false
			updateToolsFlags.tool = ""
			updateToolsFlags.dryRun = false
			updateToolsFlags.version = "latest"
		})
	}
}

func TestUpdateToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := updateToolsCmd
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
		"update-tools",
		"Updates all installed Go tools",
		"--check",
		"--tool",
		"--dry-run",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q, got:\n%s", expected, output)
		}
	}
}

// Helper struct for test setup
type toolInfo struct {
	name    string
	pkgPath string
	version string
}
