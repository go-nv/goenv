package core

import (
	"bytes"
	"github.com/go-nv/goenv/internal/utils"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/testing/testutil"
)

func TestExplainCommand(t *testing.T) {
	tests := []struct {
		name           string
		envVersion     string // GOENV_VERSION to set (empty means unset)
		setup          func(testDir string)
		cleanup        func(testDir string)
		verbose        bool
		expectedOutput []string
		expectError    bool
	}{
		{
			name: "no version set",
			setup: func(testDir string) {
				// Remove all version files to truly have no version
				os.Remove(filepath.Join(testDir, "version"))
				os.Remove(filepath.Join(testDir, ".go-version"))
			},
			cleanup: func(testDir string) {},
			expectedOutput: []string{
				"system",
				"goenv install",
				"goenv global",
			},
			expectError: false,
		},
		{
			name:       "GOENV_VERSION environment variable",
			envVersion: "1.22.0",
			setup:      func(testDir string) {},
			cleanup:    func(testDir string) {},
			expectedOutput: []string{
				"Current Go Version: 1.22.0",
				"GOENV_VERSION environment variable",
				"HIGHEST PRIORITY",
				"unset GOENV_VERSION",
			},
			expectError: false,
		},
		{
			name: "global version file",
			setup: func(testDir string) {
				globalFile := filepath.Join(testDir, "version")
				testutil.WriteTestFile(t, globalFile, []byte("1.21.0\n"), utils.PermFileDefault)
			},
			cleanup: func(testDir string) {
				os.Remove(filepath.Join(testDir, "version"))
			},
			expectedOutput: []string{
				"Current Go Version: 1.21.0",
				"GLOBAL DEFAULT",
				"goenv global",
			},
			expectError: false,
		},
		{
			name: "local .go-version file",
			setup: func(testDir string) {
				// Create a subdirectory with .go-version to be truly "local"
				subDir := filepath.Join(testDir, "project")
				_ = utils.EnsureDirWithContext(subDir, "create test directory")
				localFile := filepath.Join(subDir, ".go-version")
				testutil.WriteTestFile(t, localFile, []byte("1.23.0\n"), utils.PermFileDefault)
				os.Chdir(subDir)
			},
			cleanup: func(testDir string) {
				os.RemoveAll(filepath.Join(testDir, "project"))
			},
			expectedOutput: []string{
				"Current Go Version: 1.23.0",
				".go-version",
				"goenv local",
			},
			expectError: false,
		},
		{
			name: "verbose mode shows resolution order",
			setup: func(testDir string) {
				globalFile := filepath.Join(testDir, "version")
				testutil.WriteTestFile(t, globalFile, []byte("1.21.0\n"), utils.PermFileDefault)
			},
			cleanup: func(testDir string) {
				os.Remove(filepath.Join(testDir, "version"))
			},
			verbose: true,
			expectedOutput: []string{
				"Version Resolution Order",
				"GOENV_VERSION",
				".go-version in current directory",
				"parent directories",
				"Global version file",
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create isolated test environment
			testDir := t.TempDir()
			oldCwd, _ := os.Getwd()
			defer os.Chdir(oldCwd)

			// Setup test environment
			t.Setenv(utils.GoenvEnvVarRoot.String(), testDir)
			if tt.envVersion != "" {
				t.Setenv(utils.GoenvEnvVarVersion.String(), tt.envVersion)
			}
			os.Chdir(testDir)
			utils.EnsureDir(filepath.Join(testDir, "versions"))

			// Run test setup
			tt.setup(testDir)
			defer tt.cleanup(testDir)

			// Execute command
			buf := new(bytes.Buffer)
			explainCmd.SetOut(buf)
			explainCmd.SetErr(buf)
			explainFlags.verbose = tt.verbose

			err := runExplain(explainCmd, []string{})

			// Verify results
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output := buf.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it didn't.\nOutput:\n%s", expected, output)
				}
			}
		})
	}
}

func TestExplainArgs(t *testing.T) {
	// Create isolated test environment
	testDir := t.TempDir()
	oldCwd, _ := os.Getwd()
	defer os.Chdir(oldCwd)

	// Setup test environment
	t.Setenv(utils.GoenvEnvVarRoot.String(), testDir)
	os.Chdir(testDir)
	utils.EnsureDir(filepath.Join(testDir, "versions"))

	// Test with invalid argument
	err := runExplain(explainCmd, []string{"1.22.0"})

	if err == nil {
		t.Error("Expected error with positional argument, but got none")
	}

	if !strings.Contains(err.Error(), "usage") {
		t.Errorf("Expected usage error, got: %v", err)
	}
}
