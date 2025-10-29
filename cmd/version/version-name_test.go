package version

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionNameCommand(t *testing.T) {
	tests := []struct {
		name            string
		setupVersions   []string
		globalVersion   string
		localVersion    string
		envVersion      string
		expectedOutput  string
		expectedError   string
		expectErrorCode bool
	}{
		{
			name:           "system default version",
			expectedOutput: "system",
		},
		{
			name:           "GOENV_VERSION variable",
			setupVersions:  []string{"1.11.1"},
			envVersion:     "1.11.1",
			expectedOutput: "1.11.1",
		},
		{
			name:           "local file version",
			setupVersions:  []string{"1.10.3"},
			localVersion:   "1.10.3",
			expectedOutput: "1.10.3",
		},
		{
			name:           "global file version",
			setupVersions:  []string{"1.11.1"},
			globalVersion:  "1.11.1",
			expectedOutput: "1.11.1",
		},
		{
			name:           "GOENV_VERSION overrides local",
			setupVersions:  []string{"1.11.1", "1.10.3"},
			localVersion:   "1.10.3",
			envVersion:     "1.11.1",
			expectedOutput: "1.11.1",
		},
		{
			name:            "missing version error",
			envVersion:      "1.11.1",
			expectedError:   "goenv: version '1.11.1' is not installed (set by GOENV_VERSION environment variable)",
			expectErrorCode: true,
		},
		{
			name:           "multiple versions all installed",
			setupVersions:  []string{"1.11.1", "1.10.3"},
			envVersion:     "1.11.1:1.10.3",
			expectedOutput: "1.11.1\n1.10.3",
		},
		{
			name:            "multiple versions with missing version",
			setupVersions:   []string{"1.11.1", "1.9.7"},
			envVersion:      "1.11.1:1.10.3:1.9.7",
			expectedOutput:  "1.11.1\n1.9.7",
			expectedError:   "goenv: version '1.10.3' is not installed (set by GOENV_VERSION environment variable)",
			expectErrorCode: true,
		},
		{
			name:            "multiple versions all missing",
			envVersion:      "1.11.1:1.10.3",
			expectedError:   "goenv: version '1.11.1' is not installed (set by GOENV_VERSION environment variable)",
			expectErrorCode: true,
		},
		{
			name:           "version-name with system",
			envVersion:     "system",
			expectedOutput: "system",
		},
		{
			name:           "multiple versions with system",
			setupVersions:  []string{"1.11.1"},
			envVersion:     "1.11.1:system",
			expectedOutput: "1.11.1\nsystem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, testRoot, version)
			}

			// Set global version if specified
			if tt.globalVersion != "" {
				globalFile := filepath.Join(testRoot, "version")
				err := os.WriteFile(globalFile, []byte(tt.globalVersion), 0644)
				if err != nil {
					t.Fatalf("Failed to set global version: %v", err)
				}
			}

			// Set local version if specified
			if tt.localVersion != "" {
				localFile := filepath.Join(testRoot, ".go-version")
				err := os.WriteFile(localFile, []byte(tt.localVersion), 0644)
				if err != nil {
					t.Fatalf("Failed to set local version: %v", err)
				}
				// Change to test root so local version is found
				oldDir, _ := os.Getwd()
				defer os.Chdir(oldDir)
				os.Chdir(testRoot)
			}

			// Set environment version if specified
			if tt.envVersion != "" {
				os.Setenv("GOENV_VERSION", tt.envVersion)
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "version-name",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runVersionName(cmd, args)
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
				if tt.expectedError != "" && !strings.Contains(errOutput.String(), tt.expectedError) {
					t.Errorf("Expected error to contain %q, got %q", tt.expectedError, errOutput.String())
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v\nStderr: %s", err, errOutput.String())
				}
			}

			// Check output
			got := strings.TrimSpace(output.String())
			if tt.expectedOutput != "" {
				if got != tt.expectedOutput {
					t.Errorf("Expected output %q, got %q", tt.expectedOutput, got)
				}
			}
		})
	}
}
