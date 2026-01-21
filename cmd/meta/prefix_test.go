package meta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/cobra"
)

func TestPrefixCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		localVersion   string
		envVersion     string
		setupSystemGo  bool
		expectedOutput string
		expectedError  string
		checkPrefix    bool // if true, check output starts with expectedOutput
	}{
		{
			name:           "returns version path from local .go-version file",
			setupVersions:  []string{"1.2.3"},
			localVersion:   "1.2.3",
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.3",
		},
		{
			name:           "returns version path when major.minor given",
			args:           []string{"1.2"},
			setupVersions:  []string{"1.2.3"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.3",
		},
		{
			name:          "fails when no version exists and version is system without go in PATH",
			args:          []string{"system"},
			expectedError: "system version not found in PATH",
		},
		{
			name:          "returns system go directory when system version and go in PATH",
			args:          []string{"system"},
			setupSystemGo: true,
			checkPrefix:   true,
		},
		{
			name:          "fails when local version not installed",
			localVersion:  "1.2.3",
			expectedError: "is not installed",
		},
		{
			name:          "version from argument takes priority over GOENV_VERSION",
			args:          []string{"1.2.4"},
			envVersion:    "1.2.3",
			expectedError: "not installed",
		},
		{
			name:          "uses GOENV_VERSION when no arguments",
			envVersion:    "1.2.3",
			expectedError: "not installed",
		},
		{
			name:           "returns latest version when 'latest' specified",
			args:           []string{"latest"},
			setupVersions:  []string{"1.9.9", "1.9.10", "1.10.9", "1.10.10"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.10.10",
		},
		{
			name:           "returns latest major version when major number given",
			args:           []string{"1"},
			setupVersions:  []string{"1.2.9", "1.2.10", "4.5.6"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.10",
		},
		{
			name:          "fails when major version doesn't match any installed",
			args:          []string{"9"},
			setupVersions: []string{"1.2.9", "4.5.10"},
			expectedError: "not installed",
		},
		{
			name:           "returns latest minor version when single minor number given",
			args:           []string{"2"},
			setupVersions:  []string{"1.2.9", "1.2.10", "1.3.11", "4.5.2"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.10",
		},
		{
			name:           "returns latest patch when major.minor given",
			args:           []string{"1.2"},
			setupVersions:  []string{"1.1.2", "1.2.9", "1.2.10", "1.3.11", "2.1.2"},
			checkPrefix:    true,
			expectedOutput: "/versions/1.2.10",
		},
		{
			name:          "fails when major.minor doesn't match any installed",
			args:          []string{"1.9"},
			setupVersions: []string{"1.1.9"},
			expectedError: "not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Reset global flags
			PrefixFlags.Complete = false

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, testRoot, version)
			}

			// Setup system go if needed
			var systemBinDir string
			oldPath := os.Getenv(utils.EnvVarPath)
			defer os.Setenv(utils.EnvVarPath, oldPath)

			if tt.setupSystemGo {
				systemBinDir = filepath.Join(testRoot, "system_bin")
				_ = utils.EnsureDirWithContext(systemBinDir, "create test directory")
				systemGo := filepath.Join(systemBinDir, "go")
				var content string
				if utils.IsWindows() {
					systemGo += ".bat"
					content = "@echo off\necho go version go1.20.1 windows/amd64\n"
				} else {
					content = "#!/bin/sh\necho go version go1.20.1 linux/amd64\n"
				}

				testutil.WriteTestFile(t, systemGo, []byte(content), utils.PermFileExecutable, "Failed to create system go")

				// Add to PATH
				pathSep := string(os.PathListSeparator)
				os.Setenv(utils.EnvVarPath, systemBinDir+pathSep+oldPath)
			} else {
				// Explicitly set PATH to empty directory to ensure no system Go found
				emptyDir := filepath.Join(testRoot, "empty-bin")
				_ = utils.EnsureDirWithContext(emptyDir, "create test directory")
				os.Setenv(utils.EnvVarPath, emptyDir)
			} // Set local version if specified
			if tt.localVersion != "" {
				localFile := filepath.Join(testRoot, ".go-version")
				testutil.WriteTestFile(t, localFile, []byte(tt.localVersion), utils.PermFileDefault, "Failed to set local version")
				// Change to test root so local version is found
				oldDir, _ := os.Getwd()
				defer os.Chdir(oldDir)
				os.Chdir(testRoot)
			}

			// Set environment version if specified
			if tt.envVersion != "" {
				t.Setenv(utils.GoenvEnvVarVersion.String(), tt.envVersion)
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "prefix",
				RunE: func(cmd *cobra.Command, args []string) error {
					return RunPrefix(cmd, args)
				},
			}

			// Add flags
			cmd.Flags().BoolVar(&PrefixFlags.Complete, "complete", false, "Show completion options")

			output := &strings.Builder{}
			cmd.SetOut(output)
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

			assert.NoError(t, err)

			got := strings.TrimSpace(output.String())

			if tt.checkPrefix {
				if tt.expectedOutput != "" {
					// Normalize paths for cross-platform comparison
					normalizedGot := filepath.ToSlash(got)
					normalizedExpected := filepath.ToSlash(tt.expectedOutput)
					assert.Contains(t, normalizedGot, normalizedExpected)
				}
				if tt.setupSystemGo && systemBinDir != "" {
					// For system go, should return parent of bin dir
					expectedDir := filepath.Dir(systemBinDir)
					assert.Contains(t, got, expectedDir)
				}
			} else {
				assert.Equal(t, tt.expectedOutput, got)
			}
		})
	}
}

func TestPrefixCompletion(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Reset global flags
	PrefixFlags.Complete = false

	// Setup test versions
	cmdtest.CreateTestVersion(t, testRoot, "1.9.10")
	cmdtest.CreateTestVersion(t, testRoot, "1.10.9")

	// Create and execute command
	cmd := &cobra.Command{
		Use: "prefix",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunPrefix(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&PrefixFlags.Complete, "complete", false, "Show completion options")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{"--complete"})

	err := cmd.Execute()
	assert.NoError(t, err)

	got := strings.TrimSpace(output.String())
	gotLines := strings.Split(got, "\n")

	// Should include: latest, system, and all installed versions
	expectedLines := []string{"latest", "system", "1.10.9", "1.9.10"}

	assert.Len(t, gotLines, len(expectedLines), "Expected lines %v", got)

	for i, expected := range expectedLines {
		assert.Equal(t, expected, gotLines[i])
	}
}
