package legacy

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestLocalCommand(t *testing.T) {
	testCases := []struct {
		name                string
		args                []string
		setup               func(t *testing.T, root string) (workDir string, cleanup func())
		expectOutput        string
		expectError         string
		expectedFileContent *string
		expectFileMissing   bool
	}{
		{
			name: "show local version when file exists",
			args: []string{},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(workDir, ".go-version"), []byte("1.2.3\n"), 0644); err != nil {
					t.Fatalf("failed to seed local version: %v", err)
				}
				return workDir, nil
			},
			expectOutput: "1.2.3",
		},
		{
			name: "error when no local version configured",
			args: []string{},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "empty")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectError: "goenv: no local version configured for this directory",
		},
		{
			name: "set exact local version",
			args: []string{"1.2.3"},
			setup: func(t *testing.T, root string) (string, func()) {
				cmdtest.CreateTestVersion(t, root, "1.2.3")
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectedFileContent: ptr("1.2.3"),
		},
		{
			name: "set latest version",
			args: []string{"latest"},
			setup: func(t *testing.T, root string) (string, func()) {
				cmdtest.CreateTestVersion(t, root, "1.10.10")
				cmdtest.CreateTestVersion(t, root, "1.10.9")
				cmdtest.CreateTestVersion(t, root, "1.9.10")
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectedFileContent: ptr("1.10.10"),
		},
		{
			name: "set latest version for major",
			args: []string{"1"},
			setup: func(t *testing.T, root string) (string, func()) {
				cmdtest.CreateTestVersion(t, root, "1.2.10")
				cmdtest.CreateTestVersion(t, root, "1.2.9")
				cmdtest.CreateTestVersion(t, root, "4.5.6")
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectedFileContent: ptr("1.2.10"),
		},
		{
			name: "set latest version for minor",
			args: []string{"2"},
			setup: func(t *testing.T, root string) (string, func()) {
				cmdtest.CreateTestVersion(t, root, "1.2.10")
				cmdtest.CreateTestVersion(t, root, "1.2.9")
				cmdtest.CreateTestVersion(t, root, "1.3.11")
				cmdtest.CreateTestVersion(t, root, "4.5.2")
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectedFileContent: ptr("1.2.10"),
		},
		{
			name: "set latest version for major minor",
			args: []string{"1.2"},
			setup: func(t *testing.T, root string) (string, func()) {
				cmdtest.CreateTestVersion(t, root, "1.1.2")
				cmdtest.CreateTestVersion(t, root, "1.2.9")
				cmdtest.CreateTestVersion(t, root, "1.2.10")
				cmdtest.CreateTestVersion(t, root, "1.3.11")
				cmdtest.CreateTestVersion(t, root, "2.1.2")
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectedFileContent: ptr("1.2.10"),
		},
		{
			name: "fail when version not installed",
			args: []string{"9"},
			setup: func(t *testing.T, root string) (string, func()) {
				cmdtest.CreateTestVersion(t, root, "1.2.9")
				cmdtest.CreateTestVersion(t, root, "4.5.10")
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectError: "goenv: version '9' not installed",
		},
		{
			name: "unset removes local version file",
			args: []string{"--unset"},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(workDir, ".go-version"), []byte("1.2.3\n"), 0644); err != nil {
					t.Fatalf("failed to seed local version: %v", err)
				}
				return workDir, nil
			},
			expectFileMissing: true,
		},
		{
			name: "unset succeeds when file missing",
			args: []string{"--unset"},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				return workDir, nil
			},
			expectFileMissing: true,
		},
		{
			name: "reads parent directory version file",
			args: []string{},
			setup: func(t *testing.T, root string) (string, func()) {
				parentDir := filepath.Join(root, "project")
				subDir := filepath.Join(parentDir, "sub")
				if err := os.MkdirAll(subDir, 0755); err != nil {
					t.Fatalf("failed to create dirs: %v", err)
				}
				if err := os.WriteFile(filepath.Join(parentDir, ".go-version"), []byte("1.2.3\n"), 0644); err != nil {
					t.Fatalf("failed to seed parent local version: %v", err)
				}
				return subDir, nil
			},
			expectOutput: "1.2.3",
		},
		{
			name: "prefers current directory over GOENV_DIR",
			args: []string{},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "project")
				if err := os.MkdirAll(workDir, 0755); err != nil {
					t.Fatalf("failed to create workdir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(workDir, ".go-version"), []byte("1.2.3\n"), 0644); err != nil {
					t.Fatalf("failed to seed local version: %v", err)
				}

				home := os.Getenv("HOME")
				if err := os.MkdirAll(home, 0755); err != nil {
					t.Fatalf("failed to create home dir: %v", err)
				}
				if err := os.WriteFile(filepath.Join(home, ".go-version"), []byte("1.4-home\n"), 0644); err != nil {
					t.Fatalf("failed to seed home version: %v", err)
				}

				if err := os.Setenv("GOENV_DIR", home); err != nil {
					t.Fatalf("failed to set GOENV_DIR: %v", err)
				}
				return workDir, func() {
					os.Unsetenv("GOENV_DIR")
				}
			},
			expectOutput: "1.2.3",
		},
	}

	for _, tt := range testCases {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			oldGoenvVersion := os.Getenv("GOENV_VERSION")
			os.Unsetenv("GOENV_VERSION")
			defer func() {
				if oldGoenvVersion != "" {
					os.Setenv("GOENV_VERSION", oldGoenvVersion)
				}
			}()

			workDir, extraCleanup := tt.setup(t, testRoot)
			if extraCleanup != nil {
				defer extraCleanup()
			}

			oldDir, err := os.Getwd()
			if err != nil {
				t.Fatalf("failed to get current directory: %v", err)
			}
			defer os.Chdir(oldDir)

			if err := os.Chdir(workDir); err != nil {
				t.Fatalf("failed to change directory: %v", err)
			}

			cmd := &cobra.Command{
				Use: "local",
				RunE: func(cmd *cobra.Command, args []string) error {
					return RunLocal(cmd, args)
				},
			}
			cmd.SilenceUsage = true
			cmd.Flags().BoolVar(&localFlags.unset, "unset", false, "Unset the local Go version")
			localFlags.unset = false
			defer func() {
				localFlags.unset = false
			}()

			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs(tt.args)

			err = cmd.Execute()

			if tt.expectError != "" {
				if err == nil {
					t.Fatalf("expected error containing %q, got nil", tt.expectError)
				}
				if !strings.Contains(err.Error(), tt.expectError) {
					t.Fatalf("expected error containing %q, got %q", tt.expectError, err.Error())
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			output := cmdtest.StripDeprecationWarning(stdout.String())
			if tt.expectOutput != "" {
				if output != tt.expectOutput {
					t.Fatalf("expected output %q, got %q", tt.expectOutput, output)
				}
			} else if output != "" {
				t.Fatalf("expected no output, got %q", output)
			}

			if tt.expectedFileContent != nil {
				content, err := os.ReadFile(filepath.Join(workDir, ".go-version"))
				if err != nil {
					t.Fatalf("expected .go-version to exist: %v", err)
				}
				got := strings.TrimSpace(string(content))
				if got != *tt.expectedFileContent {
					t.Fatalf("expected .go-version to contain %q, got %q", *tt.expectedFileContent, got)
				}
			}

			if tt.expectFileMissing {
				if _, err := os.Stat(filepath.Join(workDir, ".go-version")); err == nil {
					t.Fatalf("expected .go-version to be removed")
				} else if !os.IsNotExist(err) {
					t.Fatalf("unexpected error checking .go-version: %v", err)
				}
			}
		})
	}
}

func ptr[T any](value T) *T {
	return &value
}

func TestLocalCommandRejectsExtraArguments(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test versions
	cmdtest.CreateTestVersion(t, testRoot, "1.21.5")
	cmdtest.CreateTestVersion(t, testRoot, "1.22.2")

	// Create a temp directory for the test
	workDir, err := os.MkdirTemp("", "goenv_local_test_")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(workDir)

	// Change to work directory
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(workDir)

	// Try to set local with extra arguments
	cmd := &cobra.Command{
		Use: "local",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunLocal(cmd, args)
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"1.21.5", "extra"})

	err = cmd.Execute()

	// Should error with usage message
	if err == nil {
		t.Error("Expected error when extra arguments provided, got nil")
		return
	}

	if !strings.Contains(err.Error(), "Usage: goenv local [<version>]") {
		t.Errorf("Expected usage error, got: %v", err)
	}
}
