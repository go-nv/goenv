package shims

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"

	"github.com/spf13/cobra"
)

func TestShimsCommand(t *testing.T) {
	tests := []struct {
		name           string
		setupShims     []string
		useShortFlag   bool
		expectedOutput []string
	}{
		{
			name:           "prints empty output when no shims present",
			expectedOutput: []string{},
		},
		{
			name:       "prints found shims paths in alphabetic order",
			setupShims: []string{"godoc", "go", "gofmt"},
			expectedOutput: []string{
				"/shims/go",
				"/shims/godoc",
				"/shims/gofmt",
			},
		},
		{
			name:         "prints shim names only with --short flag",
			setupShims:   []string{"godoc", "go", "gofmt"},
			useShortFlag: true,
			expectedOutput: []string{
				"go",
				"godoc",
				"gofmt",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Setup shims
			shimsDir := filepath.Join(testRoot, "shims")
			if len(tt.setupShims) > 0 {
				os.MkdirAll(shimsDir, 0755)
				for _, shim := range tt.setupShims {
					shimPath := filepath.Join(shimsDir, shim)
					var shimContent string
					if runtime.GOOS == "windows" {
						shimPath += ".bat"
						shimContent = "@echo off\necho shim\n"
					} else {
						shimContent = "#!/bin/bash\necho shim\n"
					}
					os.WriteFile(shimPath, []byte(shimContent), 0755)
				}
			}

			// Set the flag
			shimsFlags.short = tt.useShortFlag
			defer func() { shimsFlags.short = false }()

			// Create and execute command
			cmd := &cobra.Command{
				Use: "shims",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runShims(cmd, []string{})
				},
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs([]string{})

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Parse output
			got := strings.TrimSpace(output.String())
			var gotLines []string
			if got != "" {
				gotLines = strings.Split(got, "\n")
			}

			// Verify output
			if len(tt.expectedOutput) == 0 {
				if got != "" {
					t.Errorf("Expected empty output, got %q", got)
				}
			} else {
				// Sort both for comparison (they should already be sorted)
				sort.Strings(gotLines)
				expectedSorted := make([]string, len(tt.expectedOutput))
				copy(expectedSorted, tt.expectedOutput)
				sort.Strings(expectedSorted)

				if len(gotLines) != len(expectedSorted) {
					t.Errorf("Expected %d lines, got %d lines", len(expectedSorted), len(gotLines))
				}

				for i, expected := range expectedSorted {
					if i >= len(gotLines) {
						break
					}
					// For full paths, just check if it ends with the expected suffix
					if !tt.useShortFlag {
						// Normalize path separators for cross-platform comparison
						normalizedGot := filepath.ToSlash(gotLines[i])
						normalizedExpected := filepath.ToSlash(expected)
						if !strings.HasSuffix(normalizedGot, normalizedExpected) {
							t.Errorf("Line %d: expected to end with %q, got %q", i, normalizedExpected, normalizedGot)
						}
					} else {
						if gotLines[i] != expected {
							t.Errorf("Line %d: expected %q, got %q", i, expected, gotLines[i])
						}
					}
				}
			}
		})
	}
}

func TestShimsCompletion(t *testing.T) {
	_, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Set completion flag
	shimsFlags.complete = true
	defer func() { shimsFlags.complete = false }()

	// Create and execute command
	cmd := &cobra.Command{
		Use: "shims",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runShims(cmd, []string{})
		},
		SilenceUsage: true,
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	got := strings.TrimSpace(output.String())
	if got != "--short" {
		t.Errorf("Expected completion output to be '--short', got %q", got)
	}
}

func TestFindInSystemPath(t *testing.T) {
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create GOENV_ROOT/shims directory
	goenvRoot := filepath.Join(tmpDir, "goenv")
	shimsDir := filepath.Join(goenvRoot, "shims")
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create other directories with similar names (edge cases)
	similarDir := filepath.Join(tmpDir, "goenv_shims_backup") // Contains "shims" substring
	otherDir := filepath.Join(tmpDir, "bin")                  // Normal bin directory
	nestedShimsDir := filepath.Join(shimsDir, "subdir")       // Subdirectory of shims

	for _, dir := range []string{similarDir, otherDir, nestedShimsDir} {
		if err := os.MkdirAll(dir, 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create test executables in different directories
	createTestExecutable := func(dir, name string) {
		path := filepath.Join(dir, name)
		var content string
		if runtime.GOOS == "windows" {
			path += ".exe"
			content = "@echo off\necho test\n"
		} else {
			content = "#!/bin/bash\necho test\n"
		}
		if err := os.WriteFile(path, []byte(content), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create "go" executable in each directory
	createTestExecutable(shimsDir, "go")       // Should be excluded
	createTestExecutable(similarDir, "go")     // Should NOT be excluded (different path)
	createTestExecutable(otherDir, "go")       // Should NOT be excluded
	createTestExecutable(nestedShimsDir, "go") // Should be excluded (subdirectory of shims)

	tests := []struct {
		name        string
		pathEnv     string
		commandName string
		expectFound bool
		expectDir   string // Which directory should it be found in
	}{
		{
			name:        "excludes shims directory",
			pathEnv:     shimsDir + string(os.PathListSeparator) + otherDir,
			commandName: "go",
			expectFound: true,
			expectDir:   otherDir,
		},
		{
			name:        "does not exclude similar named directory",
			pathEnv:     shimsDir + string(os.PathListSeparator) + similarDir,
			commandName: "go",
			expectFound: true,
			expectDir:   similarDir,
		},
		{
			name:        "excludes nested subdirectory of shims",
			pathEnv:     nestedShimsDir + string(os.PathListSeparator) + otherDir,
			commandName: "go",
			expectFound: true,
			expectDir:   otherDir,
		},
		{
			name:        "finds command in first non-shims directory",
			pathEnv:     shimsDir + string(os.PathListSeparator) + similarDir + string(os.PathListSeparator) + otherDir,
			commandName: "go",
			expectFound: true,
			expectDir:   similarDir, // First non-shims directory
		},
		{
			name:        "returns not found when only shims directory",
			pathEnv:     shimsDir,
			commandName: "go",
			expectFound: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set PATH environment variable
			originalPath := os.Getenv("PATH")
			defer os.Setenv("PATH", originalPath)
			os.Setenv("PATH", tt.pathEnv)

			// Call findInSystemPath
			foundPath, err := findInSystemPath(tt.commandName, goenvRoot)

			if tt.expectFound {
				if err != nil {
					t.Errorf("Expected to find command, got error: %v", err)
				} else {
					// Check if found in expected directory
					expectedPrefix := tt.expectDir
					foundDir := filepath.Dir(foundPath)
					// Normalize both paths for comparison
					expectedAbs, _ := filepath.Abs(expectedPrefix)
					foundAbs, _ := filepath.Abs(foundDir)
					if expectedAbs != foundAbs {
						t.Errorf("Expected to find in %s, but found in %s (full path: %s)", expectedAbs, foundAbs, foundPath)
					}
				}
			} else {
				if err == nil {
					t.Errorf("Expected error (not found), but found: %s", foundPath)
				}
			}
		})
	}
}
