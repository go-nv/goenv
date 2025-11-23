package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

func TestShShellCommand(t *testing.T) {
	var err error
	if utils.IsWindows() {
		t.Skip("Skipping Unix shell test on Windows")
	}

	tests := []struct {
		name           string
		args           []string
		envVars        map[string]string
		setupFunc      func(t *testing.T, tmpDir string)
		expectedOutput string
		expectedError  string
		shouldFail     bool
	}{
		{
			name: "fails when there's no go version specified in arguments and in 'GOENV_VERSION' environment variable",
			args: []string{},
			envVars: map[string]string{
				"GOENV_VERSION": "",
			},
			shouldFail:    true,
			expectedError: "no shell-specific version configured",
		},
		{
			name: "prints 'GOENV_VERSION' when there's no go version specified in arguments, but there's one in 'GOENV_VERSION' environment variable",
			args: []string{},
			envVars: map[string]string{
				"GOENV_VERSION": "1.2.3",
			},
			expectedOutput: `echo "$GOENV_VERSION"`,
		},
		{
			name: "prints unset variable when '--unset' is given in arguments and shell is 'bash'",
			args: []string{"--unset"},
			envVars: map[string]string{
				"GOENV_SHELL": "bash",
			},
			expectedOutput: "unset GOENV_VERSION",
		},
		{
			name: "prints unset variable when '--unset' is given in arguments and shell is 'zsh'",
			args: []string{"--unset"},
			envVars: map[string]string{
				"GOENV_SHELL": "zsh",
			},
			expectedOutput: "unset GOENV_VERSION",
		},
		{
			name: "prints unset variable when '--unset' is given in arguments and shell is 'ksh'",
			args: []string{"--unset"},
			envVars: map[string]string{
				"GOENV_SHELL": "ksh",
			},
			expectedOutput: "unset GOENV_VERSION",
		},
		{
			name: "prints unset variable when '--unset' is given in arguments and shell is 'fish'",
			args: []string{"--unset"},
			envVars: map[string]string{
				"GOENV_SHELL": "fish",
			},
			expectedOutput: "set -e GOENV_VERSION",
		},
		{
			name: "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'bash'",
			args: []string{"1.2.3"},
			envVars: map[string]string{
				"GOENV_SHELL": "bash",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.2.3")
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
			},
			expectedOutput: `export GOENV_VERSION="1.2.3"`,
		},
		{
			name: "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'zsh'",
			args: []string{"1.2.3"},
			envVars: map[string]string{
				"GOENV_SHELL": "zsh",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.2.3")
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
			},
			expectedOutput: `export GOENV_VERSION="1.2.3"`,
		},
		{
			name: "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'ksh'",
			args: []string{"1.2.3"},
			envVars: map[string]string{
				"GOENV_SHELL": "ksh",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.2.3")
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
			},
			expectedOutput: `export GOENV_VERSION="1.2.3"`,
		},
		{
			name: "changes 'GOENV_VERSION' environment variable to specified shell version argument if it's installed in GOENV_ROOT/versions/<version> and shell is 'fish'",
			args: []string{"1.2.3"},
			envVars: map[string]string{
				"GOENV_SHELL": "fish",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.2.3")
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
			},
			expectedOutput: `set -gx GOENV_VERSION "1.2.3"`,
		},
		{
			name: "fails changing 'GOENV_VERSION' environment variable to specified shell version argument if version does not exist in GOENV_ROOT/versions/<version>",
			args: []string{"1.2.3"},
			envVars: map[string]string{
				"GOENV_SHELL": "bash",
			},
			shouldFail:     true,
			expectedError:  "version '1.2.3' not installed",
			expectedOutput: "false",
		},
		{
			name: "has completion support",
			args: []string{"--complete"},
			envVars: map[string]string{
				"GOENV_SHELL": "bash",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				// Create a test version
				versionDir := filepath.Join(tmpDir, "versions", "1.10.0")
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
			},
			expectedOutput: "--unset\nsystem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Run setup function if provided
			if tt.setupFunc != nil {
				tt.setupFunc(t, tmpDir)
			}

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Execute command directly via RunE
			outputBuf := &strings.Builder{}
			errorBuf := &strings.Builder{}

			// Create a temporary command with our buffers
			cmd := &cobra.Command{}
			cmd.SetOut(outputBuf)
			cmd.SetErr(errorBuf)

			err = runShShell(cmd, tt.args)

			// Check error expectation
			if tt.shouldFail {
				assert.Error(t, err, "Expected command to fail, but it succeeded")
				if tt.expectedError != "" {
					errOutput := err.Error() + errorBuf.String()
					assert.Contains(t, errOutput, tt.expectedError, "Expected error containing %v %v", tt.expectedError, errOutput)
				}
			} else {
				require.NoError(t, err)
			}

			// Check output
			if tt.expectedOutput != "" {
				output := strings.TrimSpace(outputBuf.String())
				assert.Contains(t, output, tt.expectedOutput, "Expected output to contain %v %v", tt.expectedOutput, output)
			}
		})
	}
}
