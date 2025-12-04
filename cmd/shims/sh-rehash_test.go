package shims

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

func TestShRehashCommand(t *testing.T) {
	var err error
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
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
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
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
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
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
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
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
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
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
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
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
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
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
			},
			expectedOutput: "export GOROOT=",
		},
		{
			name: "preserves existing GOPATH when already set",
			args: []string{"--only-manage-paths"},
			envVars: map[string]string{
				"GOENV_VERSION":        "1.12.0",
				"GOENV_SHELL":          "bash",
				"GOENV_DISABLE_GOROOT": "0",
				"GOENV_DISABLE_GOPATH": "0",
				"GOPATH":               "/custom/gopath",
			},
			setupFunc: func(t *testing.T, tmpDir string) {
				versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
				err = utils.EnsureDirWithContext(versionDir, "create test directory")
				require.NoError(t, err, "Failed to create version directory")
			},
			expectedOutput: "export GOPATH=",
		},
	}

	// Additional test for duplicate prevention
	t.Run("prevents duplicate GOPATH entries when re-sourced", func(t *testing.T) {
		if utils.IsWindows() {
			t.Skip("Skipping Unix shell test on Windows")
		}

		tmpDir, cleanup := cmdtest.SetupTestEnv(t)
		defer cleanup()

		// Create version directory
		versionDir := filepath.Join(tmpDir, "versions", "1.12.0")
		err := utils.EnsureDirWithContext(versionDir, "create test directory")
		require.NoError(t, err)

		home, _ := os.UserHomeDir()
		versionGopath := filepath.Join(home, "go", "1.12.0")

		// Simulate GOPATH that already has the version-specific path (from previous init)
		existingGopath := versionGopath + ":/existing/path1:" + versionGopath + ":/existing/path2"

		// Set environment variables
		os.Setenv("GOENV_VERSION", "1.12.0")
		os.Setenv("GOENV_SHELL", "bash")
		os.Setenv("GOPATH", existingGopath)
		defer os.Unsetenv("GOENV_VERSION")
		defer os.Unsetenv("GOENV_SHELL")
		defer os.Unsetenv("GOPATH")

		// Execute command
		outputBuf := &strings.Builder{}
		cmd := &cobra.Command{}
		cmd.SetOut(outputBuf)
		cmd.SetErr(&strings.Builder{})

		err = runShRehash(cmd, []string{"--only-manage-paths"})
		require.NoError(t, err)

		output := outputBuf.String()

		// Count occurrences of the version-specific GOPATH
		count := strings.Count(output, versionGopath)

		// Should only appear once in the output, not multiple times
		assert.Equal(t, 1, count, "Version-specific GOPATH should appear only once in output, got: %s", output)

		// Verify both custom paths are still there
		assert.Contains(t, output, "/existing/path1")
		assert.Contains(t, output, "/existing/path2")
	})
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

			err = runShRehash(cmd, tt.args)

			// Check error expectation
			if tt.shouldFail {
				assert.Error(t, err, "Expected command to fail, but it succeeded")
			} else {
				require.NoError(t, err)
			}

			// Check output
			output := strings.TrimSpace(outputBuf.String())
			if tt.expectedOutput != "" {
				assert.Contains(t, output, tt.expectedOutput, "Expected output to contain %v %v", tt.expectedOutput, output)
			} else {
				// Expect empty output
				assert.Empty(t, output, "Expected empty output")
			}
		})
	}
}

func TestShRehashGOPATHDuplication(t *testing.T) {
	if utils.IsWindows() {
		t.Skip("Skipping Unix shell test on Windows")
	}

	tests := []struct {
		name           string
		existingGOPATH string
		version        string
		shell          string
		expectedCount  int // How many times the version path should appear
		description    string
	}{
		{
			name:           "filters out duplicate goenv-managed paths on re-init",
			existingGOPATH: "", // Will be set to $HOME/go/1.23.2 in setup
			version:        "1.23.2",
			shell:          "bash",
			expectedCount:  1,
			description:    "Should only have version path once, not duplicated",
		},
		{
			name:           "filters triple duplication",
			existingGOPATH: "", // Will be set to $HOME/go/1.23.2:$HOME/go/1.23.2:$HOME/go/1.23.2
			version:        "1.23.2",
			shell:          "bash",
			expectedCount:  1,
			description:    "Should filter all duplicates down to one",
		},
		{
			name:           "preserves custom paths while filtering duplicates",
			existingGOPATH: "", // Will be set to $HOME/go/1.23.2:/custom/path:$HOME/go/1.23.2
			version:        "1.23.2",
			shell:          "bash",
			expectedCount:  1,
			description:    "Should have version once and keep custom path",
		},
		{
			name:           "preserves $HOME/go if present",
			existingGOPATH: "", // Will be set to $HOME/go/1.23.2:$HOME/go
			version:        "1.23.2",
			shell:          "bash",
			expectedCount:  1,
			description:    "Should keep both $HOME/go and $HOME/go/1.23.2",
		},
		{
			name:           "fish shell filters duplicates",
			existingGOPATH: "",
			version:        "1.23.2",
			shell:          "fish",
			expectedCount:  1,
			description:    "Fish shell should also filter duplicates",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Create version directory
			versionDir := filepath.Join(tmpDir, "versions", tt.version)
			err := utils.EnsureDirWithContext(versionDir, "create test directory")
			require.NoError(t, err, "Failed to create version directory")

			home, err := os.UserHomeDir()
			require.NoError(t, err)

			versionPath := filepath.Join(home, "go", tt.version)

			// Set up existing GOPATH based on test case
			var existingPath string
			switch tt.name {
			case "filters out duplicate goenv-managed paths on re-init":
				existingPath = versionPath
			case "filters triple duplication":
				existingPath = strings.Join([]string{versionPath, versionPath, versionPath}, string(os.PathListSeparator))
			case "preserves custom paths while filtering duplicates":
				existingPath = strings.Join([]string{versionPath, "/custom/path", versionPath}, string(os.PathListSeparator))
			case "preserves $HOME/go if present":
				existingPath = strings.Join([]string{versionPath, filepath.Join(home, "go")}, string(os.PathListSeparator))
			case "fish shell filters duplicates":
				existingPath = versionPath
			}

			// Set environment
			os.Setenv("GOENV_VERSION", tt.version)
			os.Setenv("GOENV_SHELL", tt.shell)
			os.Setenv("GOPATH", existingPath)
			defer os.Unsetenv("GOENV_VERSION")
			defer os.Unsetenv("GOENV_SHELL")
			defer os.Unsetenv("GOPATH")

			// Run command
			outputBuf := &strings.Builder{}
			cmd := &cobra.Command{}
			cmd.SetOut(outputBuf)
			cmd.SetErr(&strings.Builder{})

			err = runShRehash(cmd, []string{"--only-manage-paths"})
			require.NoError(t, err)

			output := outputBuf.String()

			// Extract GOPATH value from output
			var gopathLine string
			lines := strings.Split(output, "\n")
			for _, line := range lines {
				if strings.Contains(line, "GOPATH") {
					gopathLine = line
					break
				}
			}

			require.NotEmpty(t, gopathLine, "Expected GOPATH to be set in output")

			// Extract the value
			var gopathValue string
			if tt.shell == "fish" {
				// Fish: set -gx GOPATH "value"
				parts := strings.SplitN(gopathLine, `"`, 3)
				if len(parts) >= 2 {
					gopathValue = parts[1]
				}
			} else {
				// Bash: export GOPATH="value"
				parts := strings.SplitN(gopathLine, `"`, 3)
				if len(parts) >= 2 {
					gopathValue = parts[1]
				}
			}

			require.NotEmpty(t, gopathValue, "Failed to extract GOPATH value from: %s", gopathLine)

			// Count occurrences of version path
			pathSegments := filepath.SplitList(gopathValue)
			count := 0
			for _, segment := range pathSegments {
				if segment == versionPath {
					count++
				}
			}

			assert.Equal(t, tt.expectedCount, count,
				"%s: Expected version path to appear %d time(s), but appeared %d time(s) in GOPATH: %s",
				tt.description, tt.expectedCount, count, gopathValue)

			// Additional checks for specific test cases
			switch tt.name {
			case "preserves custom paths while filtering duplicates":
				assert.Contains(t, gopathValue, "/custom/path",
					"Should preserve custom path")
			case "preserves $HOME/go if present":
				assert.Contains(t, gopathValue, filepath.Join(home, "go"),
					"Should preserve $HOME/go")
			}
		})
	}
}
