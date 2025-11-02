package tools

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// Helper struct for test setup
type testToolInfo struct {
	name    string
	pkgPath string
	version string
}

// setupUpdateTestEnv creates a test environment for update tests
func setupUpdateTestEnv(t *testing.T, version string, tools []testToolInfo, shouldCreateVersion bool) string {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldDir) })

	if version != "" {
		t.Setenv(utils.GoenvEnvVarVersion.String(), version)
	}

	if shouldCreateVersion && version != "" && version != manager.SystemVersion {
		versionPath := filepath.Join(tmpDir, "versions", version)

		goBinDir := filepath.Join(versionPath, "go", "bin")
		err = utils.EnsureDirWithContext(goBinDir, "create test directory")
		require.NoError(t, err, "Failed to create go bin directory")

		goBinary := filepath.Join(goBinDir, "go")
		mockScript := "#!/bin/sh\nexit 0"
		if utils.IsWindows() {
			goBinary += ".bat"
			mockScript = "@echo off\nexit 0"
		}
		testutil.WriteTestFile(t, goBinary, []byte(mockScript), utils.PermFileExecutable)

		gopathBin := filepath.Join(versionPath, "gopath", "bin")
		err = utils.EnsureDirWithContext(gopathBin, "create test directory")
		require.NoError(t, err, "Failed to create GOPATH/bin")

		for _, tool := range tools {
			cmdtest.CreateToolExecutable(t, gopathBin, tool.name)
		}
	}

	return tmpDir
}

func TestUpdateTools_VersionValidation(t *testing.T) {
	tests := []struct {
		name          string
		setupVersion  string
		expectedError string
	}{
		{
			name:          "system go version",
			setupVersion:  "system",
			expectedError: "cannot update tools for system Go version",
		},
		{
			name:          "go version not installed",
			setupVersion:  "1.21.0",
			expectedError: "version '1.21.0' is not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, tt.setupVersion, nil, false)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")

			buf := new(bytes.Buffer)
			updateToolsCmd.SetOut(buf)
			updateToolsCmd.SetErr(buf)

			err := runUpdateTools(updateToolsCmd, []string{})

			if err == nil {
				t.Errorf("Expected error containing %q, got nil", tt.expectedError)
			} else if !strings.Contains(err.Error(), tt.expectedError) {
				t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
			}

			updateToolsFlags.check = false
			updateToolsFlags.dryRun = false
		})
	}
}

func TestUpdateTools_BasicOperation(t *testing.T) {
	tests := []struct {
		name           string
		setupTools     []testToolInfo
		expectedOutput string
	}{
		{
			name:           "no tools installed",
			setupTools:     nil,
			expectedOutput: "No tools found",
		},
		{
			name: "tools up to date",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v1.0.0"},
			},
			expectedOutput: "All tools are up to date",
		},
		{
			name: "tool needs update - skipped without package path",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			expectedOutput: "All tools are up to date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, "1.21.0", tt.setupTools, true)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")

			buf := new(bytes.Buffer)
			updateToolsCmd.SetOut(buf)
			updateToolsCmd.SetErr(buf)

			err := runUpdateTools(updateToolsCmd, []string{})

			assert.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.expectedOutput, "Expected output to contain , got:\\n %v %v", tt.expectedOutput, output)

			updateToolsFlags.check = false
			updateToolsFlags.dryRun = false
		})
	}
}

func TestUpdateTools_Modes(t *testing.T) {
	tests := []struct {
		name           string
		setupTools     []testToolInfo
		checkMode      bool
		dryRunMode     bool
		expectedOutput string
	}{
		{
			name: "check mode",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			checkMode:      true,
			expectedOutput: "All tools are up to date",
		},
		{
			name: "dry-run mode",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			dryRunMode:     true,
			expectedOutput: "Dry run",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, "1.21.0", tt.setupTools, true)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")

			if tt.checkMode {
				updateToolsCmd.Flags().Set("check", "true")
			}
			if tt.dryRunMode {
				updateToolsCmd.Flags().Set("dry-run", "true")
			}

			buf := new(bytes.Buffer)
			updateToolsCmd.SetOut(buf)
			updateToolsCmd.SetErr(buf)

			err := runUpdateTools(updateToolsCmd, []string{})

			assert.NoError(t, err)

			output := buf.String()
			assert.Contains(t, output, tt.expectedOutput, "Expected output to contain , got:\\n %v %v", tt.expectedOutput, output)

			updateToolsFlags.check = false
			updateToolsFlags.dryRun = false
		})
	}
}

func TestUpdateTools_ToolSelection(t *testing.T) {
	tests := []struct {
		name           string
		setupTools     []testToolInfo
		toolFlag       string
		expectedError  string
		expectedOutput string
	}{
		{
			name: "update specific tool",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
				{name: "mockdelve", pkgPath: "github.com/go-delve/delve/cmd/dlv", version: "v1.0.0"},
			},
			toolFlag:       "mockgopls",
			expectedOutput: "Updating tools",
		},
		{
			name: "tool not found",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			toolFlag:       "nonexistent",
			expectedOutput: "No tools found",
		},
		{
			name: "tool without package path",
			setupTools: []testToolInfo{
				{name: "mocktool", pkgPath: "", version: "unknown"},
			},
			expectedOutput: "All tools are up to date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, "1.21.0", tt.setupTools, true)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")

			buf := new(bytes.Buffer)
			updateToolsCmd.SetOut(buf)
			updateToolsCmd.SetErr(buf)

			// Pass tool as argument instead of flag
			args := []string{}
			if tt.toolFlag != "" {
				args = []string{tt.toolFlag}
			}

			err := runUpdateTools(updateToolsCmd, args)

			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if tt.expectedOutput != "" {
				output := buf.String()
				assert.Contains(t, output, tt.expectedOutput, "Expected output to contain , got:\\n %v %v", tt.expectedOutput, output)
			}

			updateToolsFlags.check = false
			updateToolsFlags.dryRun = false
		})
	}
}

func TestUpdateToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := updateToolsCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()

	expectedStrings := []string{
		"tools", "update",
		"Updates Go tools to their latest compatible versions",
		"--check",
		"--strategy",
		"--dry-run",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing , got:\\n %v %v", expected, output)
	}
}

// TestUpdateToolsVersionFlag tests that the --version flag is properly used
func TestUpdateToolsVersionFlag(t *testing.T) {
	tmpDir := t.TempDir()

	goVersion := "1.21.0"
	cmdtest.CreateMockGoVersion(t, tmpDir, goVersion)

	// Set environment for the test
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarVersion.String(), goVersion)

	// Change to tmpDir for version file detection
	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldDir) })

	globalFile := filepath.Join(tmpDir, "version")
	testutil.WriteTestFile(t, globalFile, []byte(goVersion), utils.PermFileDefault)

	updateToolsFlags.check = false
	updateToolsFlags.dryRun = true

	cmd := updateToolsCmd
	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := runUpdateTools(cmd, []string{})
	require.NoError(t, err)

	t.Log("✓ Update tools command executed successfully")

	updateToolsFlags.check = false
	updateToolsFlags.dryRun = false
}
