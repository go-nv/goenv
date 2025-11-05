package legacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

func TestVersionCommand(t *testing.T) {
	tests := []struct {
		name             string
		setupVersions    []string
		globalVersion    string
		localVersion     string
		envVersion       string
		expectedOutput   string
		expectedError    string
		checkOutputMatch bool // if true, check prefix/suffix match instead of exact
	}{
		{
			name:             "system version when no versions installed",
			expectedOutput:   "system",
			checkOutputMatch: false,
		},
		{
			name:           "version from GOENV_VERSION environment variable",
			setupVersions:  []string{"1.11.1"},
			envVersion:     "1.11.1",
			expectedOutput: "1.11.1 (set by GOENV_VERSION environment variable)",
		},
		{
			name:             "version from local .go-version file",
			setupVersions:    []string{"1.11.1"},
			localVersion:     "1.11.1",
			expectedOutput:   "1.11.1 (set by ",
			checkOutputMatch: true,
		},
		{
			name:             "version from GOENV_ROOT/version file",
			setupVersions:    []string{"1.11.1"},
			globalVersion:    "1.11.1",
			expectedOutput:   "1.11.1 (set by ",
			checkOutputMatch: true,
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
				testutil.WriteTestFile(t, globalFile, []byte(tt.globalVersion), utils.PermFileDefault, "Failed to set global version")
			}

			// Set local version if specified
			if tt.localVersion != "" {
				localFile := filepath.Join(testRoot, ".go-version")
				testutil.WriteTestFile(t, localFile, []byte(tt.localVersion), utils.PermFileDefault, "Failed to set local version")
				// Change to test root so local version is found
				oldDir, _ := os.Getwd()
				defer os.Chdir(oldDir)
				os.Chdir(testRoot)
			}

			// Set environment version if specified
			if tt.envVersion != "" {
				oldEnv := utils.GoenvEnvVarVersion.UnsafeValue()
				utils.GoenvEnvVarVersion.Set(tt.envVersion)
				defer func() {
					if oldEnv != "" {
						utils.GoenvEnvVarVersion.Set(oldEnv)
					} else {
						os.Unsetenv("GOENV_VERSION")
					}
				}()
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "version",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runVersion(cmd, args)
				},
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs([]string{})

			err := cmd.Execute()

			// Verify results
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			assert.NoError(t, err)

			got := cmdtest.StripDeprecationWarning(output.String())

			if tt.checkOutputMatch {
				// Check if output starts with expected prefix
				assert.True(t, strings.HasPrefix(got, tt.expectedOutput))
			} else {
				assert.Equal(t, tt.expectedOutput, got)
			}
		})
	}
}

func TestVersionWithMultipleVersions(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test versions
	cmdtest.CreateTestVersion(t, testRoot, "1.11.1")
	cmdtest.CreateTestVersion(t, testRoot, "1.10.3")

	// Set global version with multiple versions separated by ':'
	globalFile := filepath.Join(testRoot, "version")
	testutil.WriteTestFile(t, globalFile, []byte("1.11.1:1.10.3"), utils.PermFileDefault, "Failed to set global version")

	// Create and execute command
	cmd := &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd, args)
		},
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.NoError(t, err)

	got := cmdtest.StripDeprecationWarning(output.String())
	gotLines := strings.Split(got, "\n")

	// Should show both versions
	assert.Len(t, gotLines, 2, "Expected 2 lines %v", got)

	expectedPrefix1 := "1.11.1 (set by "
	expectedPrefix2 := "1.10.3 (set by "

	assert.True(t, strings.HasPrefix(gotLines[0], expectedPrefix1))

	assert.True(t, strings.HasPrefix(gotLines[1], expectedPrefix2))
}

func TestVersionWithMissingVersions(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup only one version
	cmdtest.CreateTestVersion(t, testRoot, "1.11.1")

	// Set GOENV_VERSION with multiple versions, some missing
	oldEnv := utils.GoenvEnvVarVersion.UnsafeValue()
	utils.GoenvEnvVarVersion.Set("1.1:1.11.1:1.2")
	defer func() {
		if oldEnv != "" {
			utils.GoenvEnvVarVersion.Set(oldEnv)
		} else {
			os.Unsetenv("GOENV_VERSION")
		}
	}()

	// Create and execute command
	cmd := &cobra.Command{
		Use: "version",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runVersion(cmd, args)
		},
	}

	output := &strings.Builder{}
	errOutput := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(errOutput)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	// Should fail but show error for missing versions and success for installed one
	assert.Error(t, err, "Expected error for missing versions, got nil")

	// Check that error output contains information about missing versions
	combined := output.String() + errOutput.String() + err.Error()

	assert.Contains(t, combined, "1.1", "Expected error output to mention missing version '1.1'")
	assert.Contains(t, combined, "1.2", "Expected error output to mention missing version '1.2'")
	assert.Contains(t, combined, "1.11.1", "Expected output to mention installed version '1.11.1'")
}

func TestVersionCommandRejectsExtraArguments(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup a test version
	cmdtest.CreateTestVersion(t, testRoot, "1.21.5")

	// Set global version
	globalFile := filepath.Join(testRoot, "version")
	testutil.WriteTestFile(t, globalFile, []byte("1.21.5"), utils.PermFileDefault, "Failed to set global version")

	// Create command with extra arguments
	cmd := &cobra.Command{
		Use:                "version",
		RunE:               runVersion,
		DisableFlagParsing: true,
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{"extra"})

	err := cmd.Execute()

	// Should error with usage message
	assert.Error(t, err, "Expected error when extra arguments provided, got nil")

	assert.Contains(t, err.Error(), "usage: goenv version", "Expected usage error %v", err)
}
