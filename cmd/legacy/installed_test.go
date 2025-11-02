package legacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/cobra"
)

func TestInstalledCommand(t *testing.T) {
	tests := []struct {
		name            string
		setupVersions   []string
		localVersion    string
		args            []string
		expectedOutput  string
		expectedError   string
		expectErrorCode bool
	}{
		{
			name:            "fails when no versions are installed",
			expectedError:   "goenv: no versions installed",
			expectErrorCode: true,
		},
		{
			name:           "prints installed version from .go-version file when no arguments given",
			setupVersions:  []string{"1.2.3"},
			localVersion:   "1.2.3",
			expectedOutput: "1.2.3",
		},
		{
			name:           "sets installed version when exact version argument matches",
			setupVersions:  []string{"1.2.3"},
			args:           []string{"1.2.3"},
			expectedOutput: "1.2.3",
		},
		{
			name:           "sets latest version when 'latest' is given",
			setupVersions:  []string{"1.10.10", "1.10.9", "1.9.10", "1.9.9"},
			args:           []string{"latest"},
			expectedOutput: "1.10.10",
		},
		{
			name:           "sets latest major version when major number is given",
			setupVersions:  []string{"1.2.10", "1.2.9", "4.5.6"},
			args:           []string{"1"},
			expectedOutput: "1.2.10",
		},
		{
			name:            "fails when major version does not match",
			setupVersions:   []string{"1.2.9", "4.5.10"},
			args:            []string{"9"},
			expectedError:   "goenv: version '9' not installed",
			expectErrorCode: true,
		},
		{
			name:           "sets latest version when minor version is given as single number",
			setupVersions:  []string{"1.2.10", "1.2.9", "1.3.11", "4.5.2"},
			args:           []string{"2"},
			expectedOutput: "1.2.10",
		},
		{
			name:           "sets latest version when minor version is given as major.minor",
			setupVersions:  []string{"1.2.10", "1.2.9", "1.2.2", "1.3.11", "2.1.2"},
			args:           []string{"1.2"},
			expectedOutput: "1.2.10",
		},
		{
			name:            "fails when major.minor does not match",
			setupVersions:   []string{"1.1.9"},
			args:            []string{"1.9"},
			expectedError:   "goenv: version '1.9' not installed",
			expectErrorCode: true,
		},
		{
			name:            "fails when exact version does not match",
			setupVersions:   []string{"1.2.3"},
			args:            []string{"1.2.4"},
			expectedError:   "goenv: version '1.2.4' not installed",
			expectErrorCode: true,
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

			// Set local version if specified
			if tt.localVersion != "" {
				localFile := filepath.Join(testRoot, ".go-version")
				testutil.WriteTestFile(t, localFile, []byte(tt.localVersion), utils.PermFileDefault, "Failed to set local version")
				// Change to test root so local version is found
				oldDir, _ := os.Getwd()
				defer os.Chdir(oldDir)
				os.Chdir(testRoot)
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "installed",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runInstalled(cmd, tt.args)
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
				assert.Error(t, err, "Expected error but got none")
				assert.False(t, tt.expectedError != "" && !strings.Contains(err.Error(), tt.expectedError), "Expected error to contain")
			} else {
				assert.NoError(t, err)
			}

			// Check output
			got := strings.TrimSpace(output.String())
			if tt.expectedOutput != "" {
				assert.Equal(t, tt.expectedOutput, got, "Expected output")
			}
		})
	}
}

func TestInstalledCompletion(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Setup test versions
	cmdtest.CreateTestVersion(t, testRoot, "1.10.9")
	cmdtest.CreateTestVersion(t, testRoot, "1.9.10")

	// Create command directly with flag
	installedFlags.complete = true
	defer func() { installedFlags.complete = false }()

	cmd := &cobra.Command{
		Use: "installed",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runInstalled(cmd, []string{})
		},
		SilenceUsage: true,
	}

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.NoError(t, err)

	got := output.String()
	expectedLines := []string{"latest", "system", "1.10.9", "1.9.10"}

	for _, expected := range expectedLines {
		assert.Contains(t, got, expected, "Expected output to contain %v %v", expected, got)
	}
}
