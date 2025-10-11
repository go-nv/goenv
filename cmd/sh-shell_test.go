package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestShShellCommand(t *testing.T) {
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
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
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
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
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
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
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
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
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
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "--unset\nsystem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := setupTestEnv(t)
			defer cleanup()

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

			err := runShShell(cmd, tt.args)

			// Check error expectation
			if tt.shouldFail {
				if err == nil {
					t.Fatalf("Expected command to fail, but it succeeded")
				}
				if tt.expectedError != "" {
					errOutput := err.Error() + errorBuf.String()
					if !strings.Contains(errOutput, tt.expectedError) {
						t.Errorf("Expected error containing %q, got %q", tt.expectedError, errOutput)
					}
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			// Check output
			if tt.expectedOutput != "" {
				output := strings.TrimSpace(outputBuf.String())
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got %q", tt.expectedOutput, output)
				}
			}
		})
	}
}
