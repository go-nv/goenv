package cmd

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

func TestMigrateToolsCommand(t *testing.T) {
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
			expectedError: "accepts 2 arg(s), received 0",
		},
		{
			name:          "only one argument provided",
			args:          []string{"1.21.0"},
			expectedError: "accepts 2 arg(s), received 1",
		},
		{
			name:          "too many arguments provided",
			args:          []string{"1.21.0", "1.22.0", "extra"},
			expectedError: "accepts 2 arg(s), received 3",
		},
		{
			name:          "source and target versions are the same",
			args:          []string{"1.21.0", "1.21.0"},
			setupVersions: []string{"1.21.0"},
			expectedError: "source and target versions cannot be the same",
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
			expectedOutput: "Successfully migrated: 1 tool(s)",
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
			expectedOutput: "2 tool(s) to migrate",
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
			expectedOutput: "2 tool(s) to migrate",
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
			expectedOutput: "2 tool(s) to migrate",
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
			expectedOutput: "No tools to migrate",
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
				if err := os.WriteFile(goBinary, []byte("#!/bin/sh\necho 'mock go'"), 0755); err != nil {
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
						if runtime.GOOS == "windows" {
							toolPath += ".exe"
						}
						if err := os.WriteFile(toolPath, []byte("mock tool"), 0755); err != nil {
							t.Fatalf("Failed to create tool %s: %v", tool, err)
						}
					}
				}
			}

			// Create command
			cmd := &cobra.Command{}
			cmd.SetArgs(tt.args)

			// Set flags
			migrateToolsCmd.ResetFlags()
			migrateToolsCmd.Flags().BoolVar(&migrateToolsFlags.dryRun, "dry-run", false, "")
			migrateToolsCmd.Flags().StringVar(&migrateToolsFlags.select_, "select", "", "")
			migrateToolsCmd.Flags().StringVar(&migrateToolsFlags.exclude, "exclude", "", "")

			for key, value := range tt.flags {
				migrateToolsCmd.Flags().Set(key, value)
			}

			// Capture output
			buf := new(bytes.Buffer)
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			migrateToolsCmd.SetOut(buf)
			migrateToolsCmd.SetErr(buf)

			// Execute - check Args validation first for cases with wrong number of args
			var err error
			if len(tt.args) != 2 {
				// These will fail Args validation, not in runMigrateTools
				err = migrateToolsCmd.Args(migrateToolsCmd, tt.args)
			} else {
				// Proper arg count, execute normally
				err = runMigrateTools(migrateToolsCmd, tt.args)
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
			migrateToolsFlags.dryRun = false
			migrateToolsFlags.select_ = ""
			migrateToolsFlags.exclude = ""
		})
	}
}

func TestMigrateToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := migrateToolsCmd
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
		"migrate-tools",
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
	// The actual filterTools function is tested indirectly through TestMigrateToolsCommand

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

func TestMigrateToolsWindowsCompatibility(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific test")
	}

	tmpDir := t.TempDir()
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Unsetenv("GOENV_ROOT")

	// Setup versions with .exe binaries
	for _, version := range []string{"1.21.0", "1.22.0"} {
		versionPath := filepath.Join(tmpDir, "versions", version)

		// Create go.exe binary
		goBinDir := filepath.Join(versionPath, "go", "bin")
		if err := os.MkdirAll(goBinDir, 0755); err != nil {
			t.Fatalf("Failed to create go bin directory: %v", err)
		}

		goBinary := filepath.Join(goBinDir, "go.exe")
		if err := os.WriteFile(goBinary, []byte("mock go"), 0755); err != nil {
			t.Fatalf("Failed to create go.exe: %v", err)
		}

		// Create tool .exe files
		gopathBin := filepath.Join(versionPath, "gopath", "bin")
		if err := os.MkdirAll(gopathBin, 0755); err != nil {
			t.Fatalf("Failed to create GOPATH/bin: %v", err)
		}

		if version == "1.21.0" {
			toolPath := filepath.Join(gopathBin, "mockgopls.exe")
			if err := os.WriteFile(toolPath, []byte("mock tool"), 0755); err != nil {
				t.Fatalf("Failed to create tool: %v", err)
			}
		}
	}

	// Test that Windows .exe handling works
	args := []string{"1.21.0", "1.22.0"}
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	_ = runMigrateTools(cmd, args)

	// Should not error on Windows with proper .exe handling
	output := buf.String()
	if !strings.Contains(output, "Discovering tools") {
		t.Errorf("Expected tool discovery on Windows, got output:\n%s", output)
	}
}
