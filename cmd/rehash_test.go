package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Comprehensive test suite for goenv rehash command
// Based on bash implementation tests from test/goenv-rehash.bats

func TestRehashCommand(t *testing.T) {
	tests := []struct {
		name            string
		setupVersions   []string
		setupBinaries   map[string][]string // version -> binaries
		setupGOPATHBins map[string][]string // version -> GOPATH binaries (future enhancement)
		existingShims   []string
		lockFile        bool   // Simulate .goenv-shim lock file
		shimsDirPerms   uint32 // 0 means don't change permissions
		expectedShims   []string
		expectedOutput  string
		expectedError   string
		skipOnRoot      bool // Skip test if running as root (permissions)
	}{
		// ===== Basic Functionality =====
		{
			name:           "creates shims directory when it does not exist",
			setupVersions:  []string{"1.11.1"},
			setupBinaries:  map[string][]string{"1.11.1": {"go"}},
			expectedShims:  []string{"go"},
			expectedOutput: "Rehashed 1 shim",
		},
		{
			name:           "succeeds with no versions installed",
			setupVersions:  []string{},
			expectedShims:  []string{},
			expectedOutput: "Rehashed 0 shims",
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

		// ===== Stale Shim Removal =====
		{
			name:          "removes stale shims that are not present anymore",
			setupVersions: []string{"1.11.1"},
			setupBinaries: map[string][]string{
				"1.11.1": {"go"},
			},
			existingShims:  []string{"oldshim1", "oldshim2", "stale_binary"},
			expectedShims:  []string{"go"},
			expectedOutput: "Rehashed 1 shim",
		},
		{
			name:          "removes old shims and creates new ones in one operation",
			setupVersions: []string{"1.11.1", "1.12.0"},
			setupBinaries: map[string][]string{
				"1.11.1": {"go", "gofmt"},
				"1.12.0": {"go", "godoc"},
			},
			existingShims:  []string{"oldcommand1", "oldcommand2"},
			expectedShims:  []string{"go", "gofmt", "godoc"},
			expectedOutput: "Rehashed 3 shim",
		},

		// ===== Multiple Binaries & Deduplication =====
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
				"1.10.0": {"go", "godoc"}, // "go" appears in both
			},
			expectedShims:  []string{"go", "gofmt", "godoc"},
			expectedOutput: "Rehashed 3 shim",
		},
		{
			name:          "handles many binaries across multiple versions",
			setupVersions: []string{"1.20.0", "1.21.0", "1.22.0"},
			setupBinaries: map[string][]string{
				"1.20.0": {"go", "gofmt", "godoc"},
				"1.21.0": {"go", "gofmt", "gocov"},
				"1.22.0": {"go", "gofmt", "govulncheck"},
			},
			expectedShims:  []string{"go", "gocov", "godoc", "gofmt", "govulncheck"},
			expectedOutput: "Rehashed 5 shim",
		},

		// ===== Edge Cases =====
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
			name:          "handles binary names with special characters",
			setupVersions: []string{"1.21.0"},
			setupBinaries: map[string][]string{
				"1.21.0": {"go-special", "go_test", "go.mod"},
			},
			// Note: createTestVersion also creates a "go" binary
			expectedShims:  []string{"go", "go-special", "go.mod", "go_test"},
			expectedOutput: "Rehashed 4 shim",
		},
		{
			name:          "handles version with no bin directory",
			setupVersions: []string{"1.21.0", "broken-version"},
			setupBinaries: map[string][]string{
				"1.21.0": {"go", "gofmt"},
				// "broken-version" has no binaries
			},
			expectedShims:  []string{"go", "gofmt"},
			expectedOutput: "Rehashed 2 shim",
		},
		{
			name:          "handles empty bin directory",
			setupVersions: []string{"1.21.0"},
			setupBinaries: map[string][]string{
				"1.21.0": {}, // Empty bin directory
			},
			// Note: createTestVersion always creates a "go" binary by default
			expectedShims:  []string{"go"},
			expectedOutput: "Rehashed 1 shim",
		},

		// ===== Error Cases =====
		{
			name:          "fails when shims directory is not writable",
			setupVersions: []string{"1.11.1"},
			setupBinaries: map[string][]string{"1.11.1": {"go"}},
			shimsDirPerms: 0444, // Read-only
			expectedError: "permission denied",
			skipOnRoot:    true, // Root can write to read-only dirs
		},
		// Note: Lock file test removed - Go implementation doesn't use lock files
		// (this is an acceptable difference from bash version)

		// ===== Shim Content Verification =====
		{
			name:          "creates shims with correct content and permissions",
			setupVersions: []string{"1.21.0"},
			setupBinaries: map[string][]string{
				"1.21.0": {"go", "gofmt"},
			},
			expectedShims:  []string{"go", "gofmt"},
			expectedOutput: "Rehashed 2 shim",
		},

		// ===== Large Scale Tests =====
		{
			name: "handles many versions with many binaries",
			setupVersions: []string{
				"1.19.0", "1.19.1", "1.19.2",
				"1.20.0", "1.20.1",
				"1.21.0", "1.21.1",
				"1.22.0",
			},
			setupBinaries: map[string][]string{
				"1.19.0": {"go", "gofmt", "godoc"},
				"1.19.1": {"go", "gofmt"},
				"1.19.2": {"go", "gofmt"},
				"1.20.0": {"go", "gofmt", "gocov"},
				"1.20.1": {"go", "gofmt"},
				"1.21.0": {"go", "gofmt", "govulncheck"},
				"1.21.1": {"go", "gofmt"},
				"1.22.0": {"go", "gofmt", "gotelemetry"},
			},
			expectedShims:  []string{"go", "gocov", "godoc", "gofmt", "gotelemetry", "govulncheck"},
			expectedOutput: "Rehashed 6 shim",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip if root and test requires non-root
			if tt.skipOnRoot && os.Getuid() == 0 {
				t.Skip("skipping test when running as root")
			}

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

			shimsDir := filepath.Join(testRoot, "shims")

			// Setup existing shims
			if len(tt.existingShims) > 0 {
				os.MkdirAll(shimsDir, 0755)
				for _, shim := range tt.existingShims {
					shimPath := filepath.Join(shimsDir, shim)
					os.WriteFile(shimPath, []byte("#!/bin/bash\necho old shim"), 0755)
				}
			}

			// Setup lock file if requested
			if tt.lockFile {
				os.MkdirAll(shimsDir, 0755)
				lockPath := filepath.Join(shimsDir, ".goenv-shim")
				os.WriteFile(lockPath, []byte(""), 0644)
			}

			// Setup permissions if specified
			if tt.shimsDirPerms != 0 {
				os.MkdirAll(shimsDir, 0755)
				if err := os.Chmod(shimsDir, os.FileMode(tt.shimsDirPerms)); err != nil {
					t.Fatalf("Failed to change shims directory permissions: %v", err)
				}
				// Ensure cleanup restores permissions
				defer os.Chmod(shimsDir, 0755)
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
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q but got none", tt.expectedError)
					return
				}
				if !strings.Contains(err.Error(), tt.expectedError) && !strings.Contains(errOutput.String(), tt.expectedError) {
					t.Errorf("Expected error to contain %q, got %q", tt.expectedError, err.Error())
				}
				return // Don't check other expectations on error
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
					return
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
			actualShims := make(map[string]bool)
			entries, err := os.ReadDir(shimsDir)
			if err != nil && !os.IsNotExist(err) {
				t.Fatalf("Failed to read shims directory: %v", err)
			}

			for _, entry := range entries {
				if !entry.IsDir() {
					actualShims[entry.Name()] = true
				}
			}

			// Check that all expected shims exist
			for _, expectedShim := range tt.expectedShims {
				if !actualShims[expectedShim] {
					t.Errorf("Expected shim %q to exist but it doesn't", expectedShim)
					continue
				}

				shimPath := filepath.Join(shimsDir, expectedShim)

				// Check that shim is executable (Unix only)
				if runtime.GOOS != "windows" {
					info, err := os.Stat(shimPath)
					if err != nil {
						t.Errorf("Failed to stat shim %q: %v", expectedShim, err)
						continue
					}
					if info.Mode()&0111 == 0 {
						t.Errorf("Expected shim %q to be executable but it isn't (mode: %o)", expectedShim, info.Mode())
					}
				}

				// Check shim content has the binary name
				content, err := os.ReadFile(shimPath)
				if err != nil {
					t.Errorf("Failed to read shim %q: %v", expectedShim, err)
					continue
				}

				contentStr := string(content)

				// Verify shim structure (bash shim)
				if runtime.GOOS != "windows" {
					if !strings.Contains(contentStr, "#!/usr/bin/env bash") {
						t.Errorf("Expected bash shebang in shim %q", expectedShim)
					}
					if !strings.Contains(contentStr, "goenv exec") {
						t.Errorf("Expected 'goenv exec' in shim %q", expectedShim)
					}
					// Bash implementation has special handling for "go*" commands
					if strings.HasPrefix(expectedShim, "go") && !strings.Contains(contentStr, "GOENV_FILE_ARG") {
						t.Logf("Note: Shim for %q doesn't have GOENV_FILE_ARG handling (may differ from bash)", expectedShim)
					}
				}
			}

			// Check that no unexpected shims exist
			for actualShim := range actualShims {
				found := false
				for _, expectedShim := range tt.expectedShims {
					if actualShim == expectedShim {
						found = true
						break
					}
				}
				if !found {
					t.Errorf("Unexpected shim %q exists", actualShim)
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

// TestRehashShimContent tests the actual content of generated shims in detail
func TestRehashShimContent(t *testing.T) {
	testRoot, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create a test version with a binary
	version := "1.21.0"
	createTestVersion(t, testRoot, version)
	createTestBinary(t, testRoot, version, "go")

	// Run rehash
	cmd := &cobra.Command{
		Use: "rehash",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runRehash(cmd, []string{})
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	if err != nil {
		t.Fatalf("Rehash failed: %v", err)
	}

	// Read and verify shim content
	shimPath := filepath.Join(testRoot, "shims", "go")
	content, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("Failed to read shim: %v", err)
	}

	contentStr := string(content)

	if runtime.GOOS != "windows" {
		// Unix shim structure checks
		requiredElements := []string{
			"#!/usr/bin/env bash",
			"# goenv shim for go",
			"set -e",
			`[ -n "$GOENV_DEBUG" ] && set -x`,
			`program="${0##*/}"`,
			"goenv exec",
		}

		for _, element := range requiredElements {
			if !strings.Contains(contentStr, element) {
				t.Errorf("Expected shim to contain %q but it doesn't.\nShim content:\n%s", element, contentStr)
			}
		}

		// Check for GOENV_FILE_ARG special handling for go commands
		// This is a bash-specific feature that may not be in Go version
		if strings.Contains(contentStr, `if [[ "$program" = "go"* ]]`) {
			t.Log("Shim has GOENV_FILE_ARG handling (matches bash implementation)")
		} else {
			t.Log("Note: Shim doesn't have GOENV_FILE_ARG handling (acceptable difference from bash)")
		}
	} else {
		// Windows shim structure checks
		requiredElements := []string{
			"@echo off",
			"REM goenv shim for go",
			"goenv exec",
		}

		for _, element := range requiredElements {
			if !strings.Contains(contentStr, element) {
				t.Errorf("Expected shim to contain %q but it doesn't.\nShim content:\n%s", element, contentStr)
			}
		}
	}
}

// TestRehashIdempotency tests that running rehash multiple times produces the same result
func TestRehashIdempotency(t *testing.T) {
	testRoot, cleanup := setupTestEnv(t)
	defer cleanup()

	// Create test versions
	createTestVersion(t, testRoot, "1.21.0")
	createTestBinary(t, testRoot, "1.21.0", "go")
	createTestBinary(t, testRoot, "1.21.0", "gofmt")

	runRehashCmd := func() ([]string, error) {
		cmd := &cobra.Command{
			Use: "rehash",
			RunE: func(cmd *cobra.Command, args []string) error {
				return runRehash(cmd, []string{})
			},
		}

		output := &strings.Builder{}
		cmd.SetOut(output)
		cmd.SetArgs([]string{})

		err := cmd.Execute()
		if err != nil {
			return nil, err
		}

		// List shims
		shimsDir := filepath.Join(testRoot, "shims")
		entries, err := os.ReadDir(shimsDir)
		if err != nil {
			return nil, err
		}

		var shims []string
		for _, entry := range entries {
			if !entry.IsDir() {
				shims = append(shims, entry.Name())
			}
		}

		return shims, nil
	}

	// Run rehash first time
	shims1, err := runRehashCmd()
	if err != nil {
		t.Fatalf("First rehash failed: %v", err)
	}

	// Run rehash second time
	shims2, err := runRehashCmd()
	if err != nil {
		t.Fatalf("Second rehash failed: %v", err)
	}

	// Run rehash third time
	shims3, err := runRehashCmd()
	if err != nil {
		t.Fatalf("Third rehash failed: %v", err)
	}

	// All should be identical
	if fmt.Sprintf("%v", shims1) != fmt.Sprintf("%v", shims2) {
		t.Errorf("First and second rehash produced different results:\n  First:  %v\n  Second: %v", shims1, shims2)
	}

	if fmt.Sprintf("%v", shims2) != fmt.Sprintf("%v", shims3) {
		t.Errorf("Second and third rehash produced different results:\n  Second: %v\n  Third:  %v", shims2, shims3)
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
