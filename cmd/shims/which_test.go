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

// Helper to create an executable in a version's bin directory

func TestWhichCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		setupExecs     map[string][]string // version -> executables
		envVersion     string
		expectedOutput string
		expectedError  string
		checkContains  bool
	}{
		{
			name:          "fails when no command argument given",
			args:          []string{},
			expectedError: "usage: goenv which <command>",
		},
		{
			name:          "prints path when GOENV_VERSION set and executable exists",
			args:          []string{"gofmt"},
			setupVersions: []string{"1.10.3"},
			setupExecs: map[string][]string{
				"1.10.3": {"gofmt"},
			},
			envVersion:     "1.10.3",
			checkContains:  true,
			expectedOutput: "/versions/1.10.3/bin/gofmt",
		},
		{
			name:          "prints path from first version when multiple versions specified",
			args:          []string{"gofmt"},
			setupVersions: []string{"1.10.3", "1.11.1"},
			setupExecs: map[string][]string{
				"1.10.3": {"gofmt"},
				"1.11.1": {"gofmt"},
			},
			envVersion:     "1.11.1:1.10.3",
			checkContains:  true,
			expectedOutput: "/versions/1.11.1/bin/gofmt",
		},
		{
			name:          "fails when version not installed",
			args:          []string{"go"},
			envVersion:    "1.10.3",
			expectedError: "is not installed",
		},
		{
			name:          "fails when multiple versions not installed",
			args:          []string{"go"},
			envVersion:    "1.10.3:1.11.1",
			expectedError: "is not installed",
		},
		{
			name:          "fails when executable not found in installed version",
			args:          []string{"gofmt"},
			setupVersions: []string{"1.8.1"},
			setupExecs: map[string][]string{
				"1.8.1": {"go"},
			},
			envVersion:    "1.8.1",
			expectedError: "command not found",
		},
		{
			name:          "shows versions that have executable when not found in current version",
			args:          []string{"gofmt"},
			setupVersions: []string{"1.4.0", "1.10.3", "1.11.1"},
			setupExecs: map[string][]string{
				"1.4.0":  {"go"},
				"1.10.3": {"gofmt"},
				"1.11.1": {"gofmt"},
			},
			envVersion:     "1.4.0",
			expectedError:  "command not found",
			checkContains:  true,
			expectedOutput: "1.10.3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Reset global flags
			whichFlags.complete = false

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, testRoot, version)
			}

			// Setup executables
			for version, execs := range tt.setupExecs {
				for _, exec := range execs {
					cmdtest.CreateExecutable(t, testRoot, version, exec)
				}
			}

			// Set environment version if specified
			if tt.envVersion != "" {
				oldEnv := os.Getenv(utils.GoenvEnvVarVersion.String())
				os.Setenv(utils.GoenvEnvVarVersion.String(), tt.envVersion)
				defer func() {
					if oldEnv != "" {
						os.Setenv(utils.GoenvEnvVarVersion.String(), oldEnv)
					} else {
						os.Unsetenv("GOENV_VERSION")
					}
				}()
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use:  "which",
				Args: cobra.MaximumNArgs(1),
				RunE: func(cmd *cobra.Command, args []string) error {
					return runWhich(cmd, args)
				},
			}

			output := &strings.Builder{}
			errOutput := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetErr(errOutput)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			// Verify results
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}

				// Check error output for additional messages
				if tt.checkContains && tt.expectedOutput != "" {
					combined := err.Error() + errOutput.String()
					if !strings.Contains(combined, tt.expectedOutput) {
						t.Errorf("Expected error output to contain '%s', got:\n%s\nError: %v", tt.expectedOutput, errOutput.String(), err)
					}
				}
				return
			}

			if err != nil {
				t.Errorf("Unexpected error: %v\nStderr: %s", err, errOutput.String())
				return
			}

			got := strings.TrimSpace(output.String())

			if tt.checkContains {
				// Normalize paths for cross-platform comparison
				normalizedGot := filepath.ToSlash(got)
				normalizedExpected := filepath.ToSlash(tt.expectedOutput)
				if !strings.Contains(normalizedGot, normalizedExpected) {
					t.Errorf("Expected output to contain '%s', got '%s'", tt.expectedOutput, got)
				}
			} else {
				if got != tt.expectedOutput {
					t.Errorf("Expected '%s', got '%s'", tt.expectedOutput, got)
				}
			}
		})
	}
}
