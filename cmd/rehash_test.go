package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestRehashCommand(t *testing.T) {
	tests := []struct {
		name            string
		setupVersions   []string
		setupBinaries   map[string][]string // version -> binaries
		existingShims   []string
		expectedShims   []string
		expectedOutput  string
		expectedError   string
		expectErrorCode bool
	}{
		{
			name:           "creates shims directory when it does not exist",
			setupVersions:  []string{"1.11.1"},
			setupBinaries:  map[string][]string{"1.11.1": {"go"}},
			expectedShims:  []string{"go"},
			expectedOutput: "Rehashed 1 shim",
		},
		{
			name:          "succeeds with no versions installed",
			expectedShims: []string{},
		},
		{
			name:          "creates executable shims for binaries",
			setupVersions: []string{"1.11.1", "1.9.0"},
			setupBinaries: map[string][]string{
				"1.11.1": {"go"},
				"1.9.0":  {"godoc"},
			},
			expectedShims:  []string{"go", "godoc"},
			expectedOutput: "Rehashed 2 shim",
		},
		{
			name:          "removes stale shims that are not present anymore",
			setupVersions: []string{"1.11.1"},
			setupBinaries: map[string][]string{
				"1.11.1": {"go"},
			},
			existingShims:  []string{"oldshim1", "oldshim2"},
			expectedShims:  []string{"go"},
			expectedOutput: "Rehashed 1 shim",
		},
		{
			name:          "handles version names with spaces",
			setupVersions: []string{"dirname1 p247"},
			setupBinaries: map[string][]string{
				"dirname1 p247": {"go"},
			},
			expectedShims:  []string{"go"},
			expectedOutput: "Rehashed 1 shim",
		},
		{
			name:          "creates shims for multiple binaries in one version",
			setupVersions: []string{"1.11.1"},
			setupBinaries: map[string][]string{
				"1.11.1": {"go", "gofmt", "godoc"},
			},
			expectedShims:  []string{"go", "gofmt", "godoc"},
			expectedOutput: "Rehashed 3 shim",
		},
		{
			name:          "deduplicates binaries across versions",
			setupVersions: []string{"1.11.1", "1.10.0"},
			setupBinaries: map[string][]string{
				"1.11.1": {"go", "gofmt"},
				"1.10.0": {"go", "godoc"},
			},
			expectedShims:  []string{"go", "gofmt", "godoc"},
			expectedOutput: "Rehashed 3 shim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup test versions with binaries
			for _, version := range tt.setupVersions {
				createTestVersion(t, testRoot, version)

				if binaries, ok := tt.setupBinaries[version]; ok {
					for _, binary := range binaries {
						createTestBinary(t, testRoot, version, binary)
					}
				}
			}

			// Setup existing shims
			shimsDir := filepath.Join(testRoot, "shims")
			if len(tt.existingShims) > 0 {
				os.MkdirAll(shimsDir, 0755)
				for _, shim := range tt.existingShims {
					shimPath := filepath.Join(shimsDir, shim)
					os.WriteFile(shimPath, []byte("#!/bin/bash\necho old shim"), 0755)
				}
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "rehash",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runRehash(cmd, []string{})
				},
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			errOutput := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetErr(errOutput)
			cmd.SetArgs([]string{})

			err := cmd.Execute()

			// Check error expectations
			if tt.expectErrorCode {
				if err == nil {
					t.Errorf("Expected error but got none")
				}
				if tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error to contain %q, got %q", tt.expectedError, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}

			// Check output
			got := output.String()
			if tt.expectedOutput != "" {
				if !strings.Contains(got, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got %q", tt.expectedOutput, got)
				}
			}

			// Verify expected shims exist
			for _, expectedShim := range tt.expectedShims {
				shimPath := filepath.Join(shimsDir, expectedShim)
				if _, err := os.Stat(shimPath); os.IsNotExist(err) {
					t.Errorf("Expected shim %q to exist but it doesn't", expectedShim)
				}

				// Check that shim is executable
				info, _ := os.Stat(shimPath)
				if info != nil && info.Mode()&0111 == 0 {
					t.Errorf("Expected shim %q to be executable but it isn't", expectedShim)
				}

				// Check shim content has the binary name
				content, _ := os.ReadFile(shimPath)
				if !strings.Contains(string(content), expectedShim) {
					t.Errorf("Expected shim content to contain binary name %q", expectedShim)
				}
			}

			// Verify old shims are removed
			for _, oldShim := range tt.existingShims {
				shimPath := filepath.Join(shimsDir, oldShim)
				found := false
				for _, expected := range tt.expectedShims {
					if oldShim == expected {
						found = true
						break
					}
				}
				if !found {
					if _, err := os.Stat(shimPath); !os.IsNotExist(err) {
						t.Errorf("Expected old shim %q to be removed but it still exists", oldShim)
					}
				}
			}
		})
	}
}

// Helper to create a test binary in a version's bin directory
func createTestBinary(t *testing.T, root, version, binaryName string) {
	binDir := filepath.Join(root, "versions", version, "bin")
	binaryPath := filepath.Join(binDir, binaryName)

	content := fmt.Sprintf("#!/bin/bash\necho 'Mock %s from version %s'\n", binaryName, version)
	if err := os.WriteFile(binaryPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create test binary %s: %v", binaryName, err)
	}
}
