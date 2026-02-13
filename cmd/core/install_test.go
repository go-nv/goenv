package core

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/version"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstallCommand_FlagValidation(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "ipv4 and ipv6 together",
			args: []string{"1.21.0"},
			flags: map[string]string{
				"ipv4": "true",
				"ipv6": "true",
			},
			expectedError: "cannot specify both --ipv4 and --ipv6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Reset flags
			installCmd.ResetFlags()
			installCmd.Flags().BoolVarP(&installFlags.force, "force", "f", false, "")
			installCmd.Flags().BoolVarP(&installFlags.skipExisting, "skip-existing", "s", false, "")
			installCmd.Flags().BoolVarP(&installFlags.list, "list", "l", false, "")
			installCmd.Flags().BoolVarP(&installFlags.keep, "keep", "k", false, "")
			installCmd.Flags().BoolVarP(&installFlags.verbose, "verbose", "v", false, "")
			installCmd.Flags().BoolVarP(&installFlags.quiet, "quiet", "q", false, "")
			installCmd.Flags().BoolVarP(&installFlags.ipv4, "ipv4", "4", false, "")
			installCmd.Flags().BoolVarP(&installFlags.ipv6, "ipv6", "6", false, "")
			installCmd.Flags().BoolVarP(&installFlags.debug, "debug", "g", false, "")
			installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "")

			// Set flags
			for key, value := range tt.flags {
				installCmd.Flags().Set(key, value)
			}

			// Capture output
			buf := new(bytes.Buffer)
			installCmd.SetOut(buf)
			installCmd.SetErr(buf)

			// Execute
			err := runInstall(installCmd, tt.args)

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

			// Reset flags
			installFlags.force = false
			installFlags.skipExisting = false
			installFlags.list = false
			installFlags.keep = false
			installFlags.verbose = false
			installFlags.quiet = false
			installFlags.ipv4 = false
			installFlags.ipv6 = false
			installFlags.debug = false
			installFlags.complete = false
		})
	}
}

func TestInstallHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := installCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Get help text
	err := cmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()

	// Check for key help text elements (custom help text via helptext package)
	expectedStrings := []string{
		"install",
		"Usage:",
		"--force",
		"--skip-existing",
		"--list",
		"--keep",
		"--verbose",
		"--quiet",
		"Keep source tree",
		"Verbose mode",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
	}
}

func TestInstallCommand_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create a proper mock installed version with go binary
	cmdtest.CreateMockGoVersion(t, tmpDir, "1.21.0")

	// Reset flags
	installCmd.ResetFlags()
	installCmd.Flags().BoolVarP(&installFlags.skipExisting, "skip-existing", "s", false, "")
	installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "")
	installCmd.Flags().Set("skip-existing", "true")

	// Capture output
	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	// Execute - should skip silently since version already exists
	err := runInstall(installCmd, []string{"1.21.0"})

	// Should not error when skipping
	assert.NoError(t, err, "Unexpected error with skip-existing")

	// Reset flags
	installFlags.skipExisting = false
	installFlags.complete = false
}

func TestInstallCommand_AutoDetection(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test - set INTEGRATION=1 to run (requires network access)")
	}

	tests := []struct {
		name           string
		setupFiles     func(dir string) error
		expectedOutput string
		expectError    bool
	}{
		{
			name: "detects version from .go-version",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.21.0\n"), 0644)
			},
			expectedOutput: "Detected version 1.21.0",
			expectError:    false,
		},
		{
			name: "detects version from go.mod",
			setupFiles: func(dir string) error {
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22.0\n"), 0644)
			},
			expectedOutput: "Detected version 1.22.0",
			expectError:    false,
		},
		{
			name: ".go-version takes precedence over go.mod",
			setupFiles: func(dir string) error {
				if err := os.WriteFile(filepath.Join(dir, ".go-version"), []byte("1.21.0\n"), 0644); err != nil {
					return err
				}
				return os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module test\n\ngo 1.22.0\n"), 0644)
			},
			expectedOutput: "Detected version 1.21.0",
			expectError:    false,
		},
		{
			name:           "falls back to latest stable when no version files",
			setupFiles:     func(dir string) error { return nil },
			expectedOutput: "No version file found, installing latest stable",
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {

			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Setup test files
			if tt.setupFiles != nil {
				err := tt.setupFiles(tmpDir)
				require.NoError(t, err, "Failed to setup test files")
			}

			// Change to test directory
			originalWd, _ := os.Getwd()
			os.Chdir(tmpDir)
			defer os.Chdir(originalWd)

			// Reset flags
			installCmd.ResetFlags()
			installCmd.Flags().BoolVarP(&installFlags.force, "force", "f", false, "")
			installCmd.Flags().BoolVarP(&installFlags.quiet, "quiet", "q", false, "")
			installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "")

			// Capture output
			buf := new(bytes.Buffer)
			installCmd.SetOut(buf)
			installCmd.SetErr(buf)

			// Execute without version argument (triggers auto-detection)
			err := runInstall(installCmd, []string{})

			// Check expectations
			output := buf.String()
			if tt.expectError {
				assert.Error(t, err, "Expected an error")
			} else if err != nil {
				// We expect errors about versions not being found, but we're testing detection message
				t.Logf("Got expected error (version doesn't exist): %v", err)
			}

			// Check output contains expected detection message
			assert.Contains(t, output, tt.expectedOutput, "Expected output to contain detection message")

			// Reset flags
			installFlags.force = false
			installFlags.quiet = false
			installFlags.complete = false
		})
	}
}

func TestInstallCommand_ExplicitVersionOverridesAutoDetection(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test - set INTEGRATION=1 to run (attempts real Go installation)")
	}

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create .go-version with one version
	err := os.WriteFile(filepath.Join(tmpDir, ".go-version"), []byte("1.21.0\n"), 0644)
	require.NoError(t, err)

	// Change to test directory
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	// Create a mock version to "install"
	cmdtest.CreateMockGoVersion(t, tmpDir, "1.22.0")

	// Reset flags
	installCmd.ResetFlags()
	installCmd.Flags().BoolVarP(&installFlags.skipExisting, "skip-existing", "s", false, "")
	installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "")
	installCmd.Flags().Set("skip-existing", "true")

	// Capture output
	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	// Execute WITH explicit version argument (should ignore .go-version)
	err = runInstall(installCmd, []string{"1.22.0"})

	// Should skip silently (version exists)
	assert.NoError(t, err, "Unexpected error")

	// Output should NOT contain auto-detection message
	output := buf.String()
	assert.NotContains(t, output, "Detected version 1.21.0", "Should not auto-detect when explicit version given")

	// Reset flags
	installFlags.skipExisting = false
	installFlags.complete = false
}

func TestInstallCommand_QuietModeNoDetectionMessages(t *testing.T) {
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Skipping integration test - set INTEGRATION=1 to run (attempts real Go installation)")
	}

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create .go-version
	err := os.WriteFile(filepath.Join(tmpDir, ".go-version"), []byte("1.21.0\n"), 0644)
	require.NoError(t, err)

	// Change to test directory
	originalWd, _ := os.Getwd()
	os.Chdir(tmpDir)
	defer os.Chdir(originalWd)

	// Reset flags
	installCmd.ResetFlags()
	installCmd.Flags().BoolVarP(&installFlags.quiet, "quiet", "q", false, "")
	installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "")
	installCmd.Flags().Set("quiet", "true")

	// Capture output
	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	// Execute (will fail because version doesn't exist, but that's ok)
	_ = runInstall(installCmd, []string{})

	// Output should NOT contain detection message (quiet mode)
	output := buf.String()
	assert.NotContains(t, output, "Detected version", "Quiet mode should suppress detection messages")
	assert.NotContains(t, output, "📍", "Quiet mode should suppress emoji")

	// Reset flags
	installFlags.quiet = false
	installFlags.complete = false
}

func TestInstallCommand_HelpTextUpdated(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := installCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Get help text
	err := cmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()

	// Check for updated help text about auto-detection
	expectedStrings := []string{
		"If no version is specified, goenv will auto-detect",
		"Check for .go-version",
		"Check for go.mod",
		"Fall back to installing the latest stable",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output should describe auto-detection behavior")
	}
}

// TestResolvePartialVersion tests the partial version resolution logic
func TestResolvePartialVersion(t *testing.T) {
	// Create mock releases list (should be sorted with latest first)
	mockReleases := []version.GoRelease{
		{Version: "1.23.0", Stable: true},
		{Version: "1.22.9", Stable: true},
		{Version: "1.22.8", Stable: true},
		{Version: "1.22.0", Stable: true},
		{Version: "1.21.13", Stable: true},
		{Version: "1.21.5", Stable: true},
		{Version: "1.21.0", Stable: true},
		{Version: "1.20.0", Stable: true},
	}

	tests := []struct {
		name             string
		requestedVersion string
		expectedVersion  string
		expectError      bool
	}{
		{
			name:             "exact match",
			requestedVersion: "1.22.8",
			expectedVersion:  "1.22.8",
			expectError:      false,
		},
		{
			name:             "partial match returns latest patch",
			requestedVersion: "1.22",
			expectedVersion:  "1.22.9",
			expectError:      false,
		},
		{
			name:             "partial match with older version",
			requestedVersion: "1.21",
			expectedVersion:  "1.21.13",
			expectError:      false,
		},
		{
			name:             "no match returns error",
			requestedVersion: "1.19",
			expectedVersion:  "",
			expectError:      true,
		},
		{
			name:             "version with go prefix",
			requestedVersion: "go1.22",
			expectedVersion:  "1.22.9",
			expectError:      false,
		},
		{
			name:             "single digit partial",
			requestedVersion: "1.23",
			expectedVersion:  "1.23.0",
			expectError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, err := resolvePartialVersion(tt.requestedVersion, mockReleases)

			if tt.expectError {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found", "Error should indicate version not found")
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedVersion, resolved)
			}
		})
	}
}

// TestResolvePartialVersion_PrefixMatching tests edge cases in prefix matching
func TestResolvePartialVersion_PrefixMatching(t *testing.T) {
	mockReleases := []version.GoRelease{
		{Version: "1.23.0", Stable: true},
		{Version: "1.22.0", Stable: true},
		{Version: "1.21.13", Stable: true},
		{Version: "1.2.0", Stable: true}, // Should NOT match "1.21"
	}

	// Test that "1.2" doesn't match "1.21" or "1.23"
	resolved, err := resolvePartialVersion("1.2", mockReleases)
	require.NoError(t, err)
	assert.Equal(t, "1.2.0", resolved, "1.2 should match 1.2.0, not 1.21.x or 1.23.x")

	// Test that "1.21" only matches 1.21.x
	resolved, err = resolvePartialVersion("1.21", mockReleases)
	require.NoError(t, err)
	assert.Equal(t, "1.21.13", resolved, "1.21 should match 1.21.13")
}
