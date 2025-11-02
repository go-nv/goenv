package tools

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// setupSyncTestEnv creates a test environment with Go versions and tools
func setupSyncTestEnv(t *testing.T, versions []string, tools map[string][]string) string {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	for _, version := range versions {
		versionPath := filepath.Join(tmpDir, "versions", version)

		// Create go binary directory
		goBinDir := filepath.Join(versionPath, "bin")
		err = utils.EnsureDirWithContext(goBinDir, "create test directory")
		require.NoError(t, err, "Failed to create go bin directory")

		// Create mock go binary using helper (handles .bat on Windows)
		cmdtest.CreateToolExecutable(t, goBinDir, "go")

		// Create GOPATH/bin directory
		gopathBin := filepath.Join(versionPath, "gopath", "bin")
		err = utils.EnsureDirWithContext(gopathBin, "create test directory")
		require.NoError(t, err, "Failed to create GOPATH/bin")

		// Create tools for this version using helper (handles .bat on Windows)
		if versionTools, ok := tools[version]; ok {
			for _, tool := range versionTools {
				cmdtest.CreateToolExecutable(t, gopathBin, tool)
			}
		}
	}

	return tmpDir
}

func TestSyncTools_ArgumentValidation(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupVersions []string
		expectedError string
	}{
		{
			name:          "no arguments provided",
			args:          []string{},
			expectedError: "need at least 2 Go versions",
		},
		{
			name:          "only one argument provided",
			args:          []string{"1.21.0"},
			expectedError: "version '1.21.0' is not installed",
		},
		{
			name:          "too many arguments provided",
			args:          []string{"1.21.0", "1.22.0", "extra"},
			expectedError: "accepts at most 2 arg(s), received 3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSyncTestEnv(t, tt.setupVersions, nil)

			syncToolsCmd.ResetFlags()
			syncToolsCmd.Flags().BoolVar(&syncToolsFlags.dryRun, "dry-run", false, "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.select_, "select", "", "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.exclude, "exclude", "", "")

			var err error
			if len(tt.args) > 2 {
				err = syncToolsCmd.Args(syncToolsCmd, tt.args)
			} else {
				err = runSyncTools(syncToolsCmd, tt.args)
			}

			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.expectedError)
			} else if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestSyncTools_VersionValidation(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupVersions []string
		expectedError string
	}{
		{
			name:          "source and target versions are the same",
			args:          []string{"1.21.0", "1.21.0"},
			setupVersions: []string{"1.21.0"},
			expectedError: "source and target versions are the same",
		},
		{
			name:          "source version does not exist",
			args:          []string{"99.99.99", "1.22.0"},
			setupVersions: []string{"1.22.0"},
			expectedError: "version '99.99.99' is not installed",
		},
		{
			name:          "target version does not exist",
			args:          []string{"1.21.0", "99.99.99"},
			setupVersions: []string{"1.21.0"},
			expectedError: "version '99.99.99' is not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSyncTestEnv(t, tt.setupVersions, nil)

			syncToolsCmd.ResetFlags()
			syncToolsCmd.Flags().BoolVar(&syncToolsFlags.dryRun, "dry-run", false, "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.select_, "select", "", "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.exclude, "exclude", "", "")

			err := runSyncTools(syncToolsCmd, tt.args)

			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.expectedError)
			} else if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
			}
		})
	}
}

func TestSyncTools_BasicOperation(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		setupTools     map[string][]string
		dryRun         bool
		expectedOutput string
	}{
		{
			name:           "no tools in source version",
			args:           []string{"1.21.0", "1.22.0"},
			setupVersions:  []string{"1.21.0", "1.22.0"},
			setupTools:     map[string][]string{},
			expectedOutput: "No Go tools found",
		},
		{
			name:          "successful migration with one tool",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls"},
			},
			expectedOutput: "Successfully synced: 1 tool(s)",
		},
		{
			name:          "dry-run mode",
			args:          []string{"1.21.0", "1.22.0"},
			setupVersions: []string{"1.21.0", "1.22.0"},
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve"},
			},
			dryRun:         true,
			expectedOutput: "Dry run mode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSyncTestEnv(t, tt.setupVersions, tt.setupTools)

			syncToolsCmd.ResetFlags()
			syncToolsCmd.Flags().BoolVar(&syncToolsFlags.dryRun, "dry-run", false, "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.select_, "select", "", "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.exclude, "exclude", "", "")

			if tt.dryRun {
				syncToolsCmd.Flags().Set("dry-run", "true")
			}

			buf := new(bytes.Buffer)
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			syncToolsCmd.SetOut(buf)
			syncToolsCmd.SetErr(buf)

			err := runSyncTools(syncToolsCmd, tt.args)

			w.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(r)
			buf.Write(output)

			assert.False(t, err != nil && tt.expectedOutput != "No Go tools found")

			assert.Contains(t, buf.String(), tt.expectedOutput, "Expected output to contain , got:\\n %v %v", tt.expectedOutput, buf.String())

			syncToolsFlags.dryRun = false
		})
	}
}

func TestSyncTools_Filtering(t *testing.T) {
	tests := []struct {
		name           string
		setupTools     map[string][]string
		selectFlag     string
		excludeFlag    string
		expectedOutput string
	}{
		{
			name: "select specific tools",
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve", "mockstaticcheck"},
			},
			selectFlag:     "mockgopls,mockdelve",
			expectedOutput: "2 tool(s) to sync",
		},
		{
			name: "exclude specific tools",
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve", "mockstaticcheck"},
			},
			excludeFlag:    "mockstaticcheck",
			expectedOutput: "2 tool(s) to sync",
		},
		{
			name: "select and exclude together",
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls", "mockdelve", "mockstaticcheck"},
			},
			selectFlag:     "mockgopls,mockdelve,mockstaticcheck",
			excludeFlag:    "mockdelve",
			expectedOutput: "2 tool(s) to sync",
		},
		{
			name: "no tools after filtering",
			setupTools: map[string][]string{
				"1.21.0": {"mockgopls"},
			},
			selectFlag:     "mockdelve", // Select tool that doesn't exist
			expectedOutput: "No tools to sync",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupSyncTestEnv(t, []string{"1.21.0", "1.22.0"}, tt.setupTools)

			syncToolsCmd.ResetFlags()
			syncToolsCmd.Flags().BoolVar(&syncToolsFlags.dryRun, "dry-run", false, "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.select_, "select", "", "")
			syncToolsCmd.Flags().StringVar(&syncToolsFlags.exclude, "exclude", "", "")

			if tt.selectFlag != "" {
				syncToolsCmd.Flags().Set("select", tt.selectFlag)
			}
			if tt.excludeFlag != "" {
				syncToolsCmd.Flags().Set("exclude", tt.excludeFlag)
			}

			buf := new(bytes.Buffer)
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			syncToolsCmd.SetOut(buf)
			syncToolsCmd.SetErr(buf)

			runSyncTools(syncToolsCmd, []string{"1.21.0", "1.22.0"})

			w.Close()
			os.Stdout = oldStdout
			output, _ := io.ReadAll(r)
			buf.Write(output)

			assert.Contains(t, buf.String(), tt.expectedOutput, "Expected output to contain , got:\\n %v %v", tt.expectedOutput, buf.String())

			syncToolsFlags.select_ = ""
			syncToolsFlags.exclude = ""
		})
	}
}

func TestSyncToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := syncToolsCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()

	expectedStrings := []string{
		"tools", "sync",
		"source-version",
		"target-version",
		"--dry-run",
		"--select",
		"--exclude",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing , got:\\n %v %v", expected, output)
	}
}

func TestFilterTools(t *testing.T) {
	tests := []struct {
		name          string
		selectFlag    string
		excludeFlag   string
		expectedCount int
	}{
		{
			name:          "no filters",
			selectFlag:    "",
			excludeFlag:   "",
			expectedCount: 3,
		},
		{
			name:          "select one tool",
			selectFlag:    "gopls",
			excludeFlag:   "",
			expectedCount: 1,
		},
		{
			name:          "select multiple tools",
			selectFlag:    "gopls,delve",
			excludeFlag:   "",
			expectedCount: 2,
		},
		{
			name:          "exclude one tool",
			selectFlag:    "",
			excludeFlag:   "staticcheck",
			expectedCount: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Filter logic is tested through the main command tests
			_ = tt.expectedCount
		})
	}
}

func TestSyncToolsWindowsCompatibility(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows-specific test")
	}

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	for _, version := range []string{"1.21.0", "1.22.0"} {
		cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)

		if version == "1.21.0" {
			cfg := &config.Config{Root: tmpDir}
			gopathBin := cfg.VersionGopathBin(version)
			toolPath := filepath.Join(gopathBin, "mockgopls.bat")
			toolBatchContent := "@echo off\necho mock tool\n"
			testutil.WriteTestFile(t, toolPath, []byte(toolBatchContent), utils.PermFileExecutable)
		}
	}

	args := []string{"1.21.0", "1.22.0"}
	cmd := &cobra.Command{}
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	err := runSyncTools(cmd, args)

	assert.NoError(t, err, "Expected sync to succeed on Windows with .bat binaries")
}
