package shims

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"

	"github.com/spf13/cobra"
)

func TestShRehashCommand(t *testing.T) {
	if utils.IsWindows() {
		t.Skip("Skipping Unix shell test on Windows")
	}

	tests := []struct {
		name           string
		args           []string
		envVars        map[string]string
		setupFunc      func(t *testing.T, tmpDir string)
		expectedOutput string
		shouldFail     bool
	}{
		{
			name: "has completion support (but pointless)",
			args: []string{"--complete"},
			envVars: map[string]string{
				"GOENV_SHELL": "bash",
			},
			expectedOutput: "",
		},
		{
			name: "when current set 'version' is 'system', it does not export GOPATH and GOROOT env variables",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION": "system",
				"GOENV_SHELL":   "bash",
			},
			expectedOutput: "",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'bash', it only echoes rehash of binaries",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "bash",
				"GOENV_DISABLE_GOROOT": "1",
				"GOENV_DISABLE_GOPATH": "1",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "hash -r 2>/dev/null || true",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'fish', it does not echo anything",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "fish",
				"GOENV_DISABLE_GOROOT": "1",
				"GOENV_DISABLE_GOPATH": "1",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'ksh', it only echoes rehash of binaries",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "ksh",
				"GOENV_DISABLE_GOROOT": "1",
				"GOENV_DISABLE_GOPATH": "1",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "hash -r 2>/dev/null || true",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 1, shell is 'zsh', it only echoes rehash of binaries",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "zsh",
				"GOENV_DISABLE_GOROOT": "1",
				"GOENV_DISABLE_GOPATH": "1",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "hash -r 2>/dev/null || true",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash', it echoes export of 'GOPATH' and rehash of binaries",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "bash",
				"GOENV_DISABLE_GOROOT": "1",
				"GOENV_DISABLE_GOPATH": "0",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "export GOPATH=",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 1, 'GOENV_DISABLE_GOPATH' is 0, shell is 'fish', it echoes only export of 'GOPATH'",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "fish",
				"GOENV_DISABLE_GOROOT": "1",
				"GOENV_DISABLE_GOPATH": "0",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "set -gx GOPATH",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, shell is 'bash' and 'GOENV_GOPATH_PREFIX' is empty, it echoes export of 'GOROOT', 'GOPATH=$HOME/go' and rehash of binaries",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "bash",
				"GOENV_DISABLE_GOROOT": "0",
				"GOENV_DISABLE_GOPATH": "0",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "export GOROOT=",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, 'GOENV_APPEND_GOPATH' is 1, shell is 'bash', it echoes GOPATH with append",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "bash",
				"GOENV_DISABLE_GOROOT": "0",
				"GOENV_DISABLE_GOPATH": "0",
				"GOENV_APPEND_GOPATH":  "1",
				"GOPATH":               "/fake-gopath",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: ":/fake-gopath",
		},
		{
			name: "when current set 'version' is not 'system', 'GOENV_DISABLE_GOROOT' is 0, 'GOENV_DISABLE_GOPATH' is 0, 'GOENV_PREPEND_GOPATH' is 1, shell is 'bash', it echoes GOPATH with prepend",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "bash",
				"GOENV_DISABLE_GOROOT": "0",
				"GOENV_DISABLE_GOPATH": "0",
				"GOENV_PREPEND_GOPATH": "1",
				"GOPATH":               "/fake-gopath",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				if err := os.MkdirAll(versionDir, 0755); err != nil {
					t.Fatalf("Failed to create version directory: %v", err)
				}
			},
			expectedOutput: "/fake-gopath:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := cmdtest.SetupTestEnv(t)
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

			err := runShRehash(cmd, tt.args)

			// Check error expectation
			if tt.shouldFail {
				if err == nil {
					t.Fatalf("Expected command to fail, but it succeeded")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			// Check output
			output := strings.TrimSpace(outputBuf.String())
			if tt.expectedOutput != "" {
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got %q", tt.expectedOutput, output)
				}
			} else {
				// Expect empty output
				if output != "" {
					t.Errorf("Expected empty output, got %q", output)
				}
			}
		})
	}
}
