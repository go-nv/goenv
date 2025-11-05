package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

func TestVersionFileCommand(t *testing.T) {
	var err error
	tests := []struct {
		name           string
		setup          func(t *testing.T, goenvRoot string)
		args           []string
		expectedOutput string
		expectedError  string
		envVars        map[string]string
	}{
		{
			name: "in project with .go-version file",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with .go-version file
				projectDir := filepath.Join(goenvRoot, "test-project")
				err = utils.EnsureDirWithContext(projectDir, "create test directory")
				require.NoError(t, err, "Failed to create project directory")
				versionFile := filepath.Join(projectDir, ".go-version")
				testutil.WriteTestFile(t, versionFile, []byte("1.11.1\n"), utils.PermFileDefault)
				// Change to project directory
				err = os.Chdir(projectDir)
				require.NoError(t, err, "Failed to change directory")
			},
			expectedOutput: "", // Will be computed as full path to .go-version
		},
		{
			name: "in project without .go-version file",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory without .go-version file
				projectDir := filepath.Join(goenvRoot, "test-project")
				err = utils.EnsureDirWithContext(projectDir, "create test directory")
				require.NoError(t, err, "Failed to create project directory")
				err = os.Chdir(projectDir)
				require.NoError(t, err, "Failed to change directory")
			},
			expectedOutput: "", // Will output global version file path
		},
		{
			name: "detects .go-version file in parent directory",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with .go-version file
				projectDir := filepath.Join(goenvRoot, "test-project")
				err = utils.EnsureDirWithContext(projectDir, "create test directory")
				require.NoError(t, err, "Failed to create project directory")
				versionFile := filepath.Join(projectDir, ".go-version")
				testutil.WriteTestFile(t, versionFile, []byte("1.11.1\n"), utils.PermFileDefault)

				// Create subdirectory and change to it
				subDir := filepath.Join(projectDir, "subdir")
				err = utils.EnsureDirWithContext(subDir, "create test directory")
				require.NoError(t, err, "Failed to create subdirectory")
				err = os.Chdir(subDir)
				require.NoError(t, err, "Failed to change directory")
			},
			expectedOutput: "", // Will output parent's .go-version path
		},
		{
			name: "with GOENV_DIR environment variable",
			setup: func(t *testing.T, goenvRoot string) {
				// Create a directory for GOENV_DIR
				goenvDir := filepath.Join(goenvRoot, "goenv-dir")
				err = utils.EnsureDirWithContext(goenvDir, "create test directory")
				require.NoError(t, err, "Failed to create GOENV_DIR")
				versionFile := filepath.Join(goenvDir, ".go-version")
				testutil.WriteTestFile(t, versionFile, []byte("1.10.3\n"), utils.PermFileDefault)

				// Change to a different directory
				otherDir := filepath.Join(goenvRoot, "other-dir")
				err = utils.EnsureDirWithContext(otherDir, "create test directory")
				require.NoError(t, err, "Failed to create other directory")
				err = os.Chdir(otherDir)
				require.NoError(t, err, "Failed to change directory")
			},
			envVars: map[string]string{
				"GOENV_DIR": "", // Will be set to goenv-dir in the test
			},
			expectedOutput: "", // Will output GOENV_DIR's .go-version path
		},
		{
			name: "GOENV_DIR precedence over PWD",
			setup: func(t *testing.T, goenvRoot string) {
				// Create GOENV_DIR with .go-version
				goenvDir := filepath.Join(goenvRoot, "goenv-dir")
				err = utils.EnsureDirWithContext(goenvDir, "create test directory")
				require.NoError(t, err, "Failed to create GOENV_DIR")
				goenvVersionFile := filepath.Join(goenvDir, ".go-version")
				testutil.WriteTestFile(t, goenvVersionFile, []byte("1.10.3\n"), utils.PermFileDefault)

				// Create PWD with different .go-version
				pwdDir := filepath.Join(goenvRoot, "pwd-dir")
				err = utils.EnsureDirWithContext(pwdDir, "create test directory")
				require.NoError(t, err, "Failed to create PWD directory")
				pwdVersionFile := filepath.Join(pwdDir, ".go-version")
				testutil.WriteTestFile(t, pwdVersionFile, []byte("1.11.1\n"), utils.PermFileDefault)

				err = os.Chdir(pwdDir)
				require.NoError(t, err, "Failed to change directory")
			},
			envVars: map[string]string{
				"GOENV_DIR": "", // Will be set to goenv-dir in the test
			},
			expectedOutput: "", // Will output GOENV_DIR's .go-version path
		},
		{
			name: "falls back to PWD when GOENV_DIR has no .go-version",
			setup: func(t *testing.T, goenvRoot string) {
				// Create GOENV_DIR without .go-version
				goenvDir := filepath.Join(goenvRoot, "goenv-dir")
				err = utils.EnsureDirWithContext(goenvDir, "create test directory")
				require.NoError(t, err, "Failed to create GOENV_DIR")

				// Create PWD with .go-version
				pwdDir := filepath.Join(goenvRoot, "pwd-dir")
				err = utils.EnsureDirWithContext(pwdDir, "create test directory")
				require.NoError(t, err, "Failed to create PWD directory")
				pwdVersionFile := filepath.Join(pwdDir, ".go-version")
				testutil.WriteTestFile(t, pwdVersionFile, []byte("1.11.1\n"), utils.PermFileDefault)

				err = os.Chdir(pwdDir)
				require.NoError(t, err, "Failed to change directory")
			},
			envVars: map[string]string{
				"GOENV_DIR": "", // Will be set to goenv-dir in the test
			},
			expectedOutput: "", // Will output PWD's .go-version path
		},
		{
			name: "with target directory argument",
			setup: func(t *testing.T, goenvRoot string) {
				// Create target directory with .go-version
				targetDir := filepath.Join(goenvRoot, "target-dir")
				err = utils.EnsureDirWithContext(targetDir, "create test directory")
				require.NoError(t, err, "Failed to create target directory")
				versionFile := filepath.Join(targetDir, ".go-version")
				testutil.WriteTestFile(t, versionFile, []byte("1.10.3\n"), utils.PermFileDefault)

				// Change to a different directory
				otherDir := filepath.Join(goenvRoot, "other-dir")
				err = utils.EnsureDirWithContext(otherDir, "create test directory")
				require.NoError(t, err, "Failed to create other directory")
				err = os.Chdir(otherDir)
				require.NoError(t, err, "Failed to change directory")
			},
			args:           []string{"target-dir"}, // Will be updated to full path in test
			expectedOutput: "",                     // Will output target-dir's .go-version path
		},
		{
			name: "with target directory argument that has no .go-version",
			setup: func(t *testing.T, goenvRoot string) {
				// Create target directory without .go-version
				targetDir := filepath.Join(goenvRoot, "target-dir")
				err = utils.EnsureDirWithContext(targetDir, "create test directory")
				require.NoError(t, err, "Failed to create target directory")
			},
			args:          []string{"target-dir"}, // Will be updated to full path in test
			expectedError: "no version file found",
		},
		{
			name: "detects go.mod (always enabled)",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with go.mod
				projectDir := filepath.Join(goenvRoot, "test-project")
				err = utils.EnsureDirWithContext(projectDir, "create test directory")
				require.NoError(t, err, "Failed to create project directory")
				gomodFile := filepath.Join(projectDir, "go.mod")
				testutil.WriteTestFile(t, gomodFile, []byte("module test\n\ngo 1.11\n"), utils.PermFileDefault)
				err = os.Chdir(projectDir)
				require.NoError(t, err, "Failed to change directory")
			},
			expectedOutput: "", // Will be computed as full path to go.mod
		},
		{
			name: ".go-version takes precedence over go.mod",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with both .go-version and go.mod
				projectDir := filepath.Join(goenvRoot, "test-project")
				err = utils.EnsureDirWithContext(projectDir, "create test directory")
				require.NoError(t, err, "Failed to create project directory")
				versionFile := filepath.Join(projectDir, ".go-version")
				testutil.WriteTestFile(t, versionFile, []byte("1.11.1\n"), utils.PermFileDefault)
				gomodFile := filepath.Join(projectDir, "go.mod")
				testutil.WriteTestFile(t, gomodFile, []byte("module test\n\ngo 1.11\n"), utils.PermFileDefault)
				err = os.Chdir(projectDir)
				require.NoError(t, err, "Failed to change directory")
			},
			expectedOutput: "", // Will be computed as full path to .go-version
		},
		{
			name: "returns global version file when no local file found",
			setup: func(t *testing.T, goenvRoot string) {
				// Create empty project directory
				projectDir := filepath.Join(goenvRoot, "test-project")
				err = utils.EnsureDirWithContext(projectDir, "create test directory")
				require.NoError(t, err, "Failed to create project directory")
				err = os.Chdir(projectDir)
				require.NoError(t, err, "Failed to change directory")
			},
			expectedOutput: "", // Will output global version file path
		},
		{
			name: "stops at filesystem root",
			setup: func(t *testing.T, goenvRoot string) {
				// Create deep directory structure without .go-version
				deepDir := filepath.Join(goenvRoot, "a", "b", "c", "d")
				err = utils.EnsureDirWithContext(deepDir, "create test directory")
				require.NoError(t, err, "Failed to create deep directory")
				err = os.Chdir(deepDir)
				require.NoError(t, err, "Failed to change directory")
			},
			expectedOutput: "", // Will output global version file path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goenvRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Set up environment variables
			for key, value := range tt.envVars {
				if key == "GOENV_DIR" {
					// Set to full path
					value = filepath.Join(goenvRoot, "goenv-dir")
				}
				err = os.Setenv(key, value)
				require.NoError(t, err, "Failed to set environment variable")
				defer os.Unsetenv(key)
			}

			// Run setup
			if tt.setup != nil {
				tt.setup(t, goenvRoot)
			}

			// Update args with full paths if needed
			args := make([]string, len(tt.args))
			for i, arg := range tt.args {
				if arg == "target-dir" {
					args[i] = filepath.Join(goenvRoot, "target-dir")
				} else {
					args[i] = arg
				}
			}

			// Execute command
			cmd := &cobra.Command{
				Use: "version-file",
				RunE: func(cmd *cobra.Command, _ []string) error {
					return runVersionFile(cmd, args)
				},
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs([]string{})

			err = cmd.Execute()

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			require.NoError(t, err)

			// Check output
			expectedOutput := tt.expectedOutput
			if expectedOutput == "" && tt.expectedError == "" {
				// Expected output is a path - check based on test case
				switch tt.name {
				case "in project without .go-version file",
					"returns global version file when no local file found", "stops at filesystem root":
					// Should output global version file path
					expectedOutput = filepath.Join(goenvRoot, "version") + "\n"
				case "in project with .go-version file", ".go-version takes precedence over go.mod":
					// Should output project's .go-version
					expectedOutput = filepath.Join(goenvRoot, "test-project", ".go-version") + "\n"
				case "detects go.mod (always enabled)":
					// Should output project's go.mod
					expectedOutput = filepath.Join(goenvRoot, "test-project", "go.mod") + "\n"
				case "detects .go-version file in parent directory":
					// Should output parent's .go-version
					expectedOutput = filepath.Join(goenvRoot, "test-project", ".go-version") + "\n"
				case "with GOENV_DIR environment variable", "GOENV_DIR precedence over PWD":
					// Should output GOENV_DIR's .go-version
					expectedOutput = filepath.Join(goenvRoot, "goenv-dir", ".go-version") + "\n"
				case "falls back to PWD when GOENV_DIR has no .go-version":
					// Should output PWD's .go-version
					expectedOutput = filepath.Join(goenvRoot, "pwd-dir", ".go-version") + "\n"
				case "with target directory argument":
					// Should output target-dir's .go-version
					expectedOutput = filepath.Join(goenvRoot, "target-dir", ".go-version") + "\n"
				}
			}

			got := strings.TrimSpace(output.String())
			expected := strings.TrimSpace(expectedOutput)

			// Resolve symlinks for comparison (macOS /var -> /private/var)
			gotResolved, _ := filepath.EvalSymlinks(got)
			if gotResolved == "" {
				gotResolved = got
			}
			expectedResolved, _ := filepath.EvalSymlinks(expected)
			if expectedResolved == "" {
				expectedResolved = expected
			}

			assert.Equal(t, expectedResolved, gotResolved, "Expected output %v %v", expected, got)
		})
	}
}
