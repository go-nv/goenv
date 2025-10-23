package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

func TestVersionsCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		setupAliases   map[string]string // alias name -> target version
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
			expectedOutput: []string{"* system", "  1.21.5", "  1.22.2"},
		},
		{
			name:           "list with system explicitly set in global file",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			globalVersion:  "system",
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
			name:           "list with aliases displayed",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2", "1.23.0"},
			setupAliases:   map[string]string{"stable": "1.22.2", "dev": "1.23.0"},
			globalVersion:  "1.22.2",
			expectedOutput: []string{"  1.21.5", "* 1.22.2 (set by global)", "  1.23.0", "", "Aliases:", "  dev -> 1.23.0", "* stable -> 1.22.2"},
		},
		{
			name:           "skip aliases flag hides aliases",
			args:           []string{"--skip-aliases"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			setupAliases:   map[string]string{"stable": "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: []string{"* 1.21.5 (set by global)", "  1.22.2"},
		},
		{
			name:           "bare mode hides aliases",
			args:           []string{"--bare"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			setupAliases:   map[string]string{"stable": "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: []string{"1.21.5", "1.22.2"},
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

			// Setup test aliases if specified
			for name, target := range tt.setupAliases {
				createTestAlias(t, testRoot, name, target)
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

				// Add to PATH temporarily
				oldPath := os.Getenv("PATH")
				pathSep := string(os.PathListSeparator)
				os.Setenv("PATH", systemBinDir+pathSep+oldPath)
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

				// Adjust expected output if system Go is present (but not expected in test setup)
				expectedLines := tt.expectedOutput
				if !tt.expectSystemGo && hasSystemGoInTest() && !versionsFlags.bare && !versionsFlags.complete {
					// System Go exists on this machine but test didn't explicitly expect it
					// Insert "  system" at the beginning of expected output
					// (but not in completion mode or bare mode)
					expectedLines = append([]string{"  system"}, expectedLines...)
				}

				if len(gotLines) != len(expectedLines) {
					t.Errorf("Expected %d lines, got %d:\nExpected:\n%s\nGot:\n%s",
						len(expectedLines), len(gotLines),
						strings.Join(expectedLines, "\n"), got)
					return
				}

				for i, expectedLine := range expectedLines {
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
	expectedLines := 2
	if hasSystemGoInTest() {
		// System Go adds an extra line at the beginning
		expectedLines = 3
	}

	if len(gotLines) != expectedLines {
		t.Errorf("Expected %d lines, got %d:\nGot:\n%s", expectedLines, len(gotLines), got)
		return
	}

	// Adjust line indices if system Go is present
	offset := 0
	if hasSystemGoInTest() {
		// System Go appears first, so our versions are offset by 1
		offset = 1
		// Verify system line is present
		if !strings.Contains(gotLines[0], "system") {
			t.Errorf("Line 0: expected system line, got '%s'", gotLines[0])
		}
	}

	// Check line: non-current version
	if gotLines[0+offset] != "  1.21.5" {
		t.Errorf("Line %d: expected '  1.21.5', got '%s'", 0+offset, gotLines[0+offset])
	}

	// Check line: current version with suffix
	expectedPrefix := "* 1.22.2 (set by "
	expectedSuffix := "/.go-version)"
	if !strings.HasPrefix(gotLines[1+offset], expectedPrefix) || !strings.HasSuffix(gotLines[1+offset], expectedSuffix) {
		t.Errorf("Line %d: expected to match '* 1.22.2 (set by .../.go-version)', got '%s'", 1+offset, gotLines[1+offset])
	}
}

func TestVersionsNoVersionsInstalled(t *testing.T) {
	// Skip this test if system Go is available
	// This test specifically checks the error case when NO Go is available at all
	if hasSystemGoInTest() {
		t.Skip("System Go is available, cannot test 'no Go at all' scenario")
	}

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

	// Add to PATH temporarily
	oldPath := os.Getenv("PATH")
	pathSep := string(os.PathListSeparator)
	os.Setenv("PATH", systemBinDir+pathSep+oldPath)
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
	expected := "* system"

	if got != expected {
		t.Errorf("Expected output '%s', got '%s'", expected, got)
	}
}

// hasSystemGoInTest checks if system Go is available during test execution
// This is needed because CI/macOS systems may have Go installed in PATH
func hasSystemGoInTest() bool {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)
	return mgr.HasSystemGo()
}
