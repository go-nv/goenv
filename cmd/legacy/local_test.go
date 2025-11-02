package legacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalCommand(t *testing.T) {
	var err error
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
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
				testutil.WriteTestFile(t, filepath.Join(workDir, ".go-version"), []byte("1.2.3\n"), utils.PermFileDefault, "failed to seed local version")
				return workDir, nil
			},
			expectOutput: "1.2.3",
		},
		{
			name: "error when no local version configured",
			args: []string{},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "empty")
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
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
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
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
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
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
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
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
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
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
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
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
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
				return workDir, nil
			},
			expectError: "goenv: version '9' not installed",
		},
		{
			name: "unset removes local version file",
			args: []string{"--unset"},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "project")
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
				testutil.WriteTestFile(t, filepath.Join(workDir, ".go-version"), []byte("1.2.3\n"), utils.PermFileDefault, "failed to seed local version")
				return workDir, nil
			},
			expectFileMissing: true,
		},
		{
			name: "unset succeeds when file missing",
			args: []string{"--unset"},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "project")
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
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
				err = utils.EnsureDirWithContext(subDir, "create test directory")
				require.NoError(t, err, "failed to create dirs")
				testutil.WriteTestFile(t, filepath.Join(parentDir, ".go-version"), []byte("1.2.3\n"), utils.PermFileDefault, "failed to seed parent local version")
				return subDir, nil
			},
			expectOutput: "1.2.3",
		},
		{
			name: "prefers current directory over GOENV_DIR",
			args: []string{},
			setup: func(t *testing.T, root string) (string, func()) {
				workDir := filepath.Join(root, "project")
				err = utils.EnsureDirWithContext(workDir, "create test directory")
				require.NoError(t, err, "failed to create workdir")
				testutil.WriteTestFile(t, filepath.Join(workDir, ".go-version"), []byte("1.2.3\n"), utils.PermFileDefault, "failed to seed local version")

				home := os.Getenv(utils.EnvVarHome)
				err = utils.EnsureDirWithContext(home, "create test directory")
				require.NoError(t, err, "failed to create home dir")
				testutil.WriteTestFile(t, filepath.Join(home, ".go-version"), []byte("1.4-home\n"), utils.PermFileDefault, "failed to seed home version")

				err = os.Setenv(utils.GoenvEnvVarDir.String(), home)
				require.NoError(t, err, "failed to set GOENV_DIR")
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

			oldGoenvVersion := os.Getenv(utils.GoenvEnvVarVersion.String())
			os.Unsetenv("GOENV_VERSION")
			defer func() {
				if oldGoenvVersion != "" {
					os.Setenv(utils.GoenvEnvVarVersion.String(), oldGoenvVersion)
				}
			}()

			workDir, extraCleanup := tt.setup(t, testRoot)
			if extraCleanup != nil {
				defer extraCleanup()
			}

			oldDir, err := os.Getwd()
			require.NoError(t, err, "failed to get current directory")
			defer os.Chdir(oldDir)

			err = os.Chdir(workDir)
			require.NoError(t, err, "failed to change directory")

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
				assert.Error(t, err, "expected error containing")
				assert.Contains(t, err.Error(), tt.expectError, "expected error containing %v %v", tt.expectError, err.Error())
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
				require.NoError(t, err, "expected .go-version to exist")
				got := strings.TrimSpace(string(content))
				if got != *tt.expectedFileContent {
					t.Fatalf("expected .go-version to contain %q, got %q", *tt.expectedFileContent, got)
				}
			}

			if tt.expectFileMissing {
				if utils.PathExists(filepath.Join(workDir, ".go-version")) {
					t.Fatalf("expected .go-version to be removed")
				}
			}
		})
	}
}

func ptr[T any](value T) *T {
	return &value
}

func TestLocalCommandRejectsExtraArguments(t *testing.T) {
	var err error
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test versions
	cmdtest.CreateTestVersion(t, testRoot, "1.21.5")
	cmdtest.CreateTestVersion(t, testRoot, "1.22.2")

	// Create a temp directory for the test
	workDir, err := os.MkdirTemp("", "goenv_local_test_")
	require.NoError(t, err, "Failed to create temp dir")
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
	assert.Error(t, err, "Expected error when extra arguments provided, got nil")

	assert.Contains(t, err.Error(), "usage: goenv local [<version>]", "Expected usage error %v", err)
}
