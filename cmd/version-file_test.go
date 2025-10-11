package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionFileCommand(t *testing.T) {
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
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("Failed to create project directory: %v", err)
				}
				versionFile := filepath.Join(projectDir, ".go-version")
				if err := os.WriteFile(versionFile, []byte("1.11.1\n"), 0644); err != nil {
					t.Fatalf("Failed to create .go-version: %v", err)
				}
				// Change to project directory
				if err := os.Chdir(projectDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			expectedOutput: "", // Will be computed as full path to .go-version
		},
		{
			name: "in project without .go-version file",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory without .go-version file
				projectDir := filepath.Join(goenvRoot, "test-project")
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("Failed to create project directory: %v", err)
				}
				if err := os.Chdir(projectDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			expectedOutput: "", // Will output global version file path
		},
		{
			name: "detects .go-version file in parent directory",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with .go-version file
				projectDir := filepath.Join(goenvRoot, "test-project")
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("Failed to create project directory: %v", err)
				}
				versionFile := filepath.Join(projectDir, ".go-version")
				if err := os.WriteFile(versionFile, []byte("1.11.1\n"), 0644); err != nil {
					t.Fatalf("Failed to create .go-version: %v", err)
				}

				// Create subdirectory and change to it
				subDir := filepath.Join(projectDir, "subdir")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("Failed to create subdirectory: %v", err)
				}
				if err := os.Chdir(subDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			expectedOutput: "", // Will output parent's .go-version path
		},
		{
			name: "with GOENV_DIR environment variable",
			setup: func(t *testing.T, goenvRoot string) {
				// Create a directory for GOENV_DIR
				goenvDir := filepath.Join(goenvRoot, "goenv-dir")
				if err := os.MkdirAll(goenvDir, 0755); err != nil {
					t.Fatalf("Failed to create GOENV_DIR: %v", err)
				}
				versionFile := filepath.Join(goenvDir, ".go-version")
				if err := os.WriteFile(versionFile, []byte("1.10.3\n"), 0644); err != nil {
					t.Fatalf("Failed to create .go-version: %v", err)
				}

				// Change to a different directory
				otherDir := filepath.Join(goenvRoot, "other-dir")
				if err := os.MkdirAll(otherDir, 0755); err != nil {
					t.Fatalf("Failed to create other directory: %v", err)
				}
				if err := os.Chdir(otherDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
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
				if err := os.MkdirAll(goenvDir, 0755); err != nil {
					t.Fatalf("Failed to create GOENV_DIR: %v", err)
				}
				goenvVersionFile := filepath.Join(goenvDir, ".go-version")
				if err := os.WriteFile(goenvVersionFile, []byte("1.10.3\n"), 0644); err != nil {
					t.Fatalf("Failed to create GOENV_DIR .go-version: %v", err)
				}

				// Create PWD with different .go-version
				pwdDir := filepath.Join(goenvRoot, "pwd-dir")
				if err := os.MkdirAll(pwdDir, 0755); err != nil {
					t.Fatalf("Failed to create PWD directory: %v", err)
				}
				pwdVersionFile := filepath.Join(pwdDir, ".go-version")
				if err := os.WriteFile(pwdVersionFile, []byte("1.11.1\n"), 0644); err != nil {
					t.Fatalf("Failed to create PWD .go-version: %v", err)
				}

				if err := os.Chdir(pwdDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
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
				if err := os.MkdirAll(goenvDir, 0755); err != nil {
					t.Fatalf("Failed to create GOENV_DIR: %v", err)
				}

				// Create PWD with .go-version
				pwdDir := filepath.Join(goenvRoot, "pwd-dir")
				if err := os.MkdirAll(pwdDir, 0755); err != nil {
					t.Fatalf("Failed to create PWD directory: %v", err)
				}
				pwdVersionFile := filepath.Join(pwdDir, ".go-version")
				if err := os.WriteFile(pwdVersionFile, []byte("1.11.1\n"), 0644); err != nil {
					t.Fatalf("Failed to create PWD .go-version: %v", err)
				}

				if err := os.Chdir(pwdDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
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
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatalf("Failed to create target directory: %v", err)
				}
				versionFile := filepath.Join(targetDir, ".go-version")
				if err := os.WriteFile(versionFile, []byte("1.10.3\n"), 0644); err != nil {
					t.Fatalf("Failed to create .go-version: %v", err)
				}

				// Change to a different directory
				otherDir := filepath.Join(goenvRoot, "other-dir")
				if err := os.MkdirAll(otherDir, 0755); err != nil {
					t.Fatalf("Failed to create other directory: %v", err)
				}
				if err := os.Chdir(otherDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			args:           []string{"target-dir"}, // Will be updated to full path in test
			expectedOutput: "",                     // Will output target-dir's .go-version path
		},
		{
			name: "with target directory argument that has no .go-version",
			setup: func(t *testing.T, goenvRoot string) {
				// Create target directory without .go-version
				targetDir := filepath.Join(goenvRoot, "target-dir")
				if err := os.MkdirAll(targetDir, 0755); err != nil {
					t.Fatalf("Failed to create target directory: %v", err)
				}
			},
			args:          []string{"target-dir"}, // Will be updated to full path in test
			expectedError: "no version file found",
		},
		{
			name: "detects go.mod when GOENV_GOMOD_VERSION_ENABLE is set",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with go.mod
				projectDir := filepath.Join(goenvRoot, "test-project")
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("Failed to create project directory: %v", err)
				}
				gomodFile := filepath.Join(projectDir, "go.mod")
				if err := os.WriteFile(gomodFile, []byte("module test\n\ngo 1.11\n"), 0644); err != nil {
					t.Fatalf("Failed to create go.mod: %v", err)
				}
				if err := os.Chdir(projectDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			envVars: map[string]string{
				"GOENV_GOMOD_VERSION_ENABLE": "1",
			},
			expectedOutput: "", // Will be computed as full path to go.mod
		},
		{
			name: ".go-version takes precedence over go.mod",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with both .go-version and go.mod
				projectDir := filepath.Join(goenvRoot, "test-project")
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("Failed to create project directory: %v", err)
				}
				versionFile := filepath.Join(projectDir, ".go-version")
				if err := os.WriteFile(versionFile, []byte("1.11.1\n"), 0644); err != nil {
					t.Fatalf("Failed to create .go-version: %v", err)
				}
				gomodFile := filepath.Join(projectDir, "go.mod")
				if err := os.WriteFile(gomodFile, []byte("module test\n\ngo 1.11\n"), 0644); err != nil {
					t.Fatalf("Failed to create go.mod: %v", err)
				}
				if err := os.Chdir(projectDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			envVars: map[string]string{
				"GOENV_GOMOD_VERSION_ENABLE": "1",
			},
			expectedOutput: "", // Will be computed as full path to .go-version
		},
		{
			name: "ignores go.mod when GOENV_GOMOD_VERSION_ENABLE is not set",
			setup: func(t *testing.T, goenvRoot string) {
				// Create project directory with go.mod only
				projectDir := filepath.Join(goenvRoot, "test-project")
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("Failed to create project directory: %v", err)
				}
				gomodFile := filepath.Join(projectDir, "go.mod")
				if err := os.WriteFile(gomodFile, []byte("module test\n\ngo 1.11\n"), 0644); err != nil {
					t.Fatalf("Failed to create go.mod: %v", err)
				}
				if err := os.Chdir(projectDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			expectedOutput: "", // Will output global version file path
		},
		{
			name: "returns global version file when no local file found",
			setup: func(t *testing.T, goenvRoot string) {
				// Create empty project directory
				projectDir := filepath.Join(goenvRoot, "test-project")
				if err := os.MkdirAll(projectDir, 0755); err != nil {
					t.Fatalf("Failed to create project directory: %v", err)
				}
				if err := os.Chdir(projectDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			expectedOutput: "", // Will output global version file path
		},
		{
			name: "stops at filesystem root",
			setup: func(t *testing.T, goenvRoot string) {
				// Create deep directory structure without .go-version
				deepDir := filepath.Join(goenvRoot, "a", "b", "c", "d")
				if err := os.MkdirAll(deepDir, 0755); err != nil {
					t.Fatalf("Failed to create deep directory: %v", err)
				}
				if err := os.Chdir(deepDir); err != nil {
					t.Fatalf("Failed to change directory: %v", err)
				}
			},
			expectedOutput: "", // Will output global version file path
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goenvRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Set up environment variables
			for key, value := range tt.envVars {
				if key == "GOENV_DIR" {
					// Set to full path
					value = filepath.Join(goenvRoot, "goenv-dir")
				}
				if err := os.Setenv(key, value); err != nil {
					t.Fatalf("Failed to set environment variable %s: %v", key, err)
				}
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

			err := cmd.Execute()

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check output
			expectedOutput := tt.expectedOutput
			if expectedOutput == "" && tt.expectedError == "" {
				// Expected output is a path - check based on test case
				switch tt.name {
				case "in project without .go-version file", "ignores go.mod when GOENV_GOMOD_VERSION_ENABLE is not set",
					"returns global version file when no local file found", "stops at filesystem root":
					// Should output global version file path
					expectedOutput = filepath.Join(goenvRoot, "version") + "\n"
				case "in project with .go-version file", ".go-version takes precedence over go.mod":
					// Should output project's .go-version
					expectedOutput = filepath.Join(goenvRoot, "test-project", ".go-version") + "\n"
				case "detects go.mod when GOENV_GOMOD_VERSION_ENABLE is set":
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

			if gotResolved != expectedResolved {
				t.Errorf("Expected output %q, got %q", expected, got)
			}
		})
	}
}
