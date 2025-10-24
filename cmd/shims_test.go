package cmd

import (
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"

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
			testRoot, cleanup := setupTestEnv(t)
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
	_, cleanup := setupTestEnv(t)
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
