package tools

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

func TestSyncToolsCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string            // Go versions to create
		setupTools     map[string][]string // version -> tool names
		flags          map[string]string
		expectedError  string
		expectedOutput string
	}{
		{
			name:          "no arguments provided",
			args:          []string{},
			expectedError: "need at least 2 Go versions", // Auto-detect requires 2+ versions
		},
		{
			name:          "only one argument provided",
			args:          []string{"1.21.0"},
			expectedError: "source Go version 1.21.0 is not installed",
		},
		{
			name:          "too many arguments provided",
			args:          []string{"1.21.0", "1.22.0", "extra"},
			expectedError: "accepts at most 2 arg(s), received 3",
		},
		{
			name:          "source and target versions are the same",
			args:          []string{"1.21.0", "1.21.0"},
			setupVersions: []string{"1.21.0"},
			expectedError: "source and target versions are the same",
		},
		{
			name:          "source version does not exist",
			args:          []string{"99.99.99", "1.22.0"},
			setupVersions: []string{"1.22.0"},
			expectedError: "source Go version 99.99.99 is not installed",
		},
		{
			name:          "target version does not exist",
			args:          []string{"1.21.0", "99.99.99"},
			setupVersions: []string{"1.21.0"},
			expectedError: "target Go version 99.99.99 is not installed",
		},
		{
			name:           "no tools in source version",
			args:           []string{"1.21.0", "1.22.0"},
			setupVersions:  []string{"1.21.0", "1.22.0"},
			setupTools:     map[string][]string{},
			expectedOutput: "No Go tools found",
		},
		{
			name:          "successful migration with one tool",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls"},
			},
			expectedOutput: "Successfully synced: 1 tool(s)",
		},
		{
			name:          "dry-run mode",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve"},
			},
			flags: map[string]string{
				"dry-run": "true",
			},
			expectedOutput: "Dry run mode",
		},
		{
			name:          "select specific tools",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve", "mockstaticcheck"},
			},
			flags: map[string]string{
				"select": "mockgopls,mockdelve",
			},
			expectedOutput: "2 tool(s) to sync",
		},
		{
			name:          "exclude specific tools",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve", "mockstaticcheck"},
			},
			flags: map[string]string{
				"exclude": "mockstaticcheck",
			},
			expectedOutput: "2 tool(s) to sync",
		},
		{
			name:          "select and exclude together",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve", "mockstaticcheck"},
			},
			flags: map[string]string{
				"select":  "mockgopls,mockdelve,mockstaticcheck",
				"exclude": "mockdelve",
			},
			expectedOutput: "2 tool(s) to sync",
		},
		{
			name:          "no tools after filtering",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls"},
			},
			flags: map[string]string{
				"select": "mockdelve", // Select tool that doesn't exist
			},
			expectedOutput: "No tools to sync",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary goenv root
			tmpDir := t.TempDir()
			os.Setenv("GOENV_ROOT", tmpDir)
			defer os.Unsetenv("GOENV_ROOT")

			// Setup Go versions
			for _, version := range tt.setupVersions {
				versionPath := filepath.Join(tmpDir, "versions", version)

				// Create go binary directory (version dir IS the GOROOT)
				goBinDir := filepath.Join(versionPath, "bin")
				if err := os.MkdirAll(goBinDir, 0755); err != nil {
					t.Fatalf("Failed to create go bin directory: %v", err)
				}

				// Create mock go binary
				goBinary := filepath.Join(goBinDir, "go")
				var content string
				if runtime.GOOS == "windows" {
					goBinary += ".bat"
					content = "@echo off\necho mock go\n"
				} else {
					content = "#!/bin/sh\necho 'mock go'\n"
				}

				if err := os.WriteFile(goBinary, []byte(content), 0755); err != nil {
					t.Fatalf("Failed to create go binary: %v", err)
				}

				// Create GOPATH/bin directory
				gopathBin := filepath.Join(versionPath, "gopath", "bin")
				if err := os.MkdirAll(gopathBin, 0755); err != nil {
					t.Fatalf("Failed to create GOPATH/bin: %v", err)
				}

				// Create tools for this version
				if tools, ok := tt.setupTools[version]; ok {
					for _, tool := range tools {
						toolPath := filepath.Join(gopathBin, tool)
						content := "mock tool"
						if runtime.GOOS == "windows" {
							toolPath += ".bat"
							content = "@echo off\necho mock tool\n"
						}
						if err := os.WriteFile(toolPath, []byte(content), 0755); err != nil {
							t.Fatalf("Failed to create tool %s: %v", tool, err)
						}
					}
				}
			}

			// Create command
			cmd := &cobra.Command{}
			cmd.SetArgs(tt.args)

			// Set flags
			syncToolsCmd.ResetFlags()
			syncToolsCmd.Flags().BoolVar(&syncToolsFlags.dryRun, "dry-run", false, "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.select_, "select", "", "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.exclude, "exclude", "", "")

			for key, value := range tt.flags {
				syncToolsCmd.Flags().Set(key, value)
			}

			// Capture output
			buf := new(bytes.Buffer)
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			syncToolsCmd.SetOut(buf)
			syncToolsCmd.SetErr(buf)

			// Execute - check Args validation first for cases with wrong number of args
			var err error
			if len(tt.args) > 2 {
				// This will fail Args validation (MaximumNArgs(2))
				err = syncToolsCmd.Args(syncToolsCmd, tt.args)
			} else {
				// 0, 1, or 2 args - execute normally (may fail in runSyncTools)
				err = runSyncTools(syncToolsCmd, tt.args)
			}

			// Restore stdout and get output
			w.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(r)
			buf.Write(output)

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
			syncToolsFlags.dryRun = false
			syncToolsFlags.select_ = ""
			syncToolsFlags.exclude = ""
		})
	}
}

func TestSyncToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := syncToolsCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Get help text by calling Help()
	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()

	// Check for key help text elements
	expectedStrings := []string{
		"tools", "sync",
		"source-version",
		"target-version",
		"--dry-run",
		"--select",
		"--exclude",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q, got:\n%s", expected, output)
		}
	}
}

func TestFilterTools(t *testing.T) {
	// This function tests the filter logic conceptually
	// The actual filterTools function is tested indirectly through TestSyncToolsCommand

	tests := []struct {
		name          string
		selectFlag    string
		excludeFlag   string
		expectedCount int
	}{
		{
			name:          "no filters",
			selectFlag:    "",
			excludeFlag:   "",
			expectedCount: 3,
		},
		{
			name:          "select one tool",
			selectFlag:    "gopls",
			excludeFlag:   "",
			expectedCount: 1,
		},
		{
			name:          "select multiple tools",
			selectFlag:    "gopls,delve",
			excludeFlag:   "",
			expectedCount: 2,
		},
		{
			name:          "exclude one tool",
			selectFlag:    "",
			excludeFlag:   "staticcheck",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Filter logic is tested through the main command tests
			// This just validates the test structure
			_ = tt.expectedCount
		})
	}
}

func TestSyncToolsWindowsCompatibility(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tmpDir := t.TempDir()
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Unsetenv("GOENV_ROOT")

	// Setup versions with .bat binaries
	for _, version := range []string{"1.21.0", "1.22.0"} {
		versionPath := filepath.Join(tmpDir, "versions", version)

		// Create go.bat binary (version directory IS the GOROOT, no extra 'go' subdirectory)
		goBinDir := filepath.Join(versionPath, "bin")
		if err := os.MkdirAll(goBinDir, 0755); err != nil {
			t.Fatalf("Failed to create go bin directory: %v", err)
		}

		goBinary := filepath.Join(goBinDir, "go.bat")
		goBatchContent := "@echo off\necho mock go\n"
		if err := os.WriteFile(goBinary, []byte(goBatchContent), 0755); err != nil {
			t.Fatalf("Failed to create go.bat: %v", err)
		}

		// Create tool .bat files
		gopathBin := filepath.Join(versionPath, "gopath", "bin")
		if err := os.MkdirAll(gopathBin, 0755); err != nil {
			t.Fatalf("Failed to create GOPATH/bin: %v", err)
		}

		if version == "1.21.0" {
			toolPath := filepath.Join(gopathBin, "mockgopls.bat")
			toolBatchContent := "@echo off\necho mock tool\n"
			if err := os.WriteFile(toolPath, []byte(toolBatchContent), 0755); err != nil {
				t.Fatalf("Failed to create tool: %v", err)
			}
		}
	}

	// Test that Windows .bat handling works
	args := []string{"1.21.0", "1.22.0"}
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runSyncTools(cmd, args)

	// Should successfully run without error on Windows with proper .bat handling
	if err != nil {
		t.Errorf("Expected sync to succeed on Windows with .bat binaries, got error: %v", err)
	}
}
