package shims

import (
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
				_ = utils.EnsureDirWithContext(shimsDir, "create test directory")
				for _, shim := range tt.setupShims {
					shimPath := filepath.Join(shimsDir, shim)
					var shimContent string
					if utils.IsWindows() {
						shimPath += ".bat"
						shimContent = "@echo off\necho shim\n"
					} else {
						shimContent = "#!/bin/bash\necho shim\n"
					}
					testutil.WriteTestFile(t, shimPath, []byte(shimContent), utils.PermFileExecutable)
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
			assert.NoError(t, err)

			// Parse output
			got := strings.TrimSpace(output.String())
			var gotLines []string
			if got != "" {
				gotLines = strings.Split(got, "\n")
			}

			// Verify output
			if len(tt.expectedOutput) == 0 {
				assert.Empty(t, got, "Expected empty output")
			} else {
				// Sort both for comparison (they should already be sorted)
				sort.Strings(gotLines)
				expectedSorted := make([]string, len(tt.expectedOutput))
				copy(expectedSorted, tt.expectedOutput)
				sort.Strings(expectedSorted)

				assert.Len(t, gotLines, len(expectedSorted), "Expected lines")

				for i, expected := range expectedSorted {
					if i >= len(gotLines) {
						break
					}
					// For full paths, just check if it ends with the expected suffix
					if !tt.useShortFlag {
						// Normalize path separators for cross-platform comparison
						normalizedGot := filepath.ToSlash(gotLines[i])
						normalizedExpected := filepath.ToSlash(expected)
						assert.True(t, strings.HasSuffix(normalizedGot, normalizedExpected), "Line : expected to end with")
					} else {
						assert.Equal(t, expected, gotLines[i], "Line : expected")
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
	assert.NoError(t, err)

	got := strings.TrimSpace(output.String())
	assert.Equal(t, "--short", got, "Expected completion output to be '--short'")
}

func TestFindInSystemPath(t *testing.T) {
	var err error
	// Create a temporary directory structure for testing
	tmpDir := t.TempDir()

	// Create GOENV_ROOT/shims directory
	goenvRoot := filepath.Join(tmpDir, "goenv")
	shimsDir := filepath.Join(goenvRoot, "shims")
	err = utils.EnsureDirWithContext(shimsDir, "create test directory")
	require.NoError(t, err)

	// Create other directories with similar names (edge cases)
	similarDir := filepath.Join(tmpDir, "goenv_shims_backup") // Contains "shims" substring
	otherDir := filepath.Join(tmpDir, "bin")                  // Normal bin directory
	nestedShimsDir := filepath.Join(shimsDir, "subdir")       // Subdirectory of shims

	for _, dir := range []string{similarDir, otherDir, nestedShimsDir} {
		err = utils.EnsureDirWithContext(dir, "create test directory")
		require.NoError(t, err)
	}

	// Create test executables in different directories
	createTestExecutable := func(dir, name string) {
		path := filepath.Join(dir, name)
		var content string
		if utils.IsWindows() {
			path += ".exe"
			content = "@echo off\necho test\n"
		} else {
			content = "#!/bin/bash\necho test\n"
		}
		testutil.WriteTestFile(t, path, []byte(content), utils.PermFileExecutable)
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
			originalPath := os.Getenv(utils.EnvVarPath)
			defer os.Setenv(utils.EnvVarPath, originalPath)
			os.Setenv(utils.EnvVarPath, tt.pathEnv)

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
					assert.Equal(t, expectedAbs, foundAbs, "Expected to find in , but found in (full path: ) %v", foundPath)
				}
			} else {
				assert.Error(t, err, "Expected error (not found), but found %v", foundPath)
			}
		})
	}
}
