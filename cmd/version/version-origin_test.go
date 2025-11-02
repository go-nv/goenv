package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"

	"github.com/spf13/cobra"
)

func TestVersionOriginCommand(t *testing.T) {
	tests := []struct {
		name           string
		setupVersions  []string
		globalVersion  string
		localVersion   string
		envVersion     string
		envOrigin      string
		expectedOutput string
	}{
		{
			name:           "default GOENV_ROOT/version when no version set",
			expectedOutput: "version", // Will be joined with GOENV_ROOT in test
		},
		{
			name:           "GOENV_ROOT/version file when global version exists",
			setupVersions:  []string{"1.11.1"},
			globalVersion:  "1.11.1",
			expectedOutput: "version", // Will be joined with GOENV_ROOT in test
		},
		{
			name:           "GOENV_VERSION environment variable",
			setupVersions:  []string{"1.11.1"},
			localVersion:   "1.10.3", // This should be overridden
			envVersion:     "1.11.1",
			expectedOutput: "GOENV_VERSION environment variable",
		},
		{
			name:           "local .go-version file path",
			setupVersions:  []string{"1.10.3"},
			localVersion:   "1.10.3",
			expectedOutput: ".go-version", // Will be made absolute in test
		},
		{
			name:           "GOENV_VERSION_ORIGIN not inherited if not explicitly set",
			expectedOutput: "version", // Will be joined with GOENV_ROOT in test
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Change to test directory so local version files are found
			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			os.Chdir(tmpDir)

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateMockGoVersion(t, tmpDir, version)
			}

			// Set global version if specified
			if tt.globalVersion != "" {
				globalFile := filepath.Join(tmpDir, "version")
				testutil.WriteTestFile(t, globalFile, []byte(tt.globalVersion), utils.PermFileDefault, "Failed to set global version")
			}

			// Set local version if specified
			if tt.localVersion != "" {
				localFile := filepath.Join(tmpDir, ".go-version")
				testutil.WriteTestFile(t, localFile, []byte(tt.localVersion), utils.PermFileDefault, "Failed to set local version")
			}

			// Set environment version if specified
			if tt.envVersion != "" {
				t.Setenv(utils.GoenvEnvVarVersion.String(), tt.envVersion)
			}

			// Set version origin from hook if specified
			if tt.envOrigin != "" {
				t.Setenv(utils.GoenvEnvVarVersionOrigin.String(), tt.envOrigin)
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "version-origin",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runVersionOrigin(cmd, args)
				},
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs([]string{})

			err := cmd.Execute()
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			got := strings.TrimSpace(output.String())

			// Build expected output
			expected := tt.expectedOutput
			if expected == "version" {
				expected = filepath.Join(tmpDir, "version")
			} else if expected == ".go-version" {
				expected = filepath.Join(tmpDir, ".go-version")
			}

			// Resolve symlinks for comparison (macOS /var -> /private/var)
			gotResolved, _ := filepath.EvalSymlinks(got)
			expectedResolved, _ := filepath.EvalSymlinks(expected)

			// Use resolved paths if available, otherwise use originals
			if gotResolved != "" {
				got = gotResolved
			}
			if expectedResolved != "" {
				expected = expectedResolved
			}

			if got != expected {
				t.Errorf("Expected %q, got %q", expected, got)
			}
		})
	}
}
