package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// Helper to create an executable in a version's bin directory
func createExecutable(t *testing.T, testRoot, version, execName string) {
	var binDir string
	if strings.Contains(version, "/") {
		// Absolute path like "${GOENV_TEST_DIR}/bin"
		binDir = version
	} else {
		// Version name like "1.10.3"
		binDir = filepath.Join(testRoot, "versions", version, "bin")
	}

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	execPath := filepath.Join(binDir, execName)
	content := "#!/bin/sh\necho 'mock executable'\n"
	if err := os.WriteFile(execPath, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create executable: %v", err)
	}
}

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
			expectedError: "Usage: goenv which <command>",
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
			testRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Reset global flags
			whichFlags.complete = false

			// Setup test versions
			for _, version := range tt.setupVersions {
				createTestVersion(t, testRoot, version)
			}

			// Setup executables
			for version, execs := range tt.setupExecs {
				for _, exec := range execs {
					createExecutable(t, testRoot, version, exec)
				}
			}

			// Set environment version if specified
			if tt.envVersion != "" {
				oldEnv := os.Getenv("GOENV_VERSION")
				os.Setenv("GOENV_VERSION", tt.envVersion)
				defer func() {
					if oldEnv != "" {
						os.Setenv("GOENV_VERSION", oldEnv)
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
				if !strings.Contains(got, tt.expectedOutput) {
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
