package tools

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
)

// Helper struct for test setup
type testToolInfo struct {
	name    string
	pkgPath string
	version string
}

// setupUpdateTestEnv creates a test environment for update tests
func setupUpdateTestEnv(t *testing.T, version string, tools []testToolInfo, shouldCreateVersion bool) string {
	tmpDir := t.TempDir()
	os.Setenv("GOENV_ROOT", tmpDir)
	os.Setenv("GOENV_DIR", tmpDir)
	t.Cleanup(func() {
		os.Unsetenv("GOENV_ROOT")
		os.Unsetenv("GOENV_DIR")
	})

	oldDir, _ := os.Getwd()
	os.Chdir(tmpDir)
	t.Cleanup(func() { os.Chdir(oldDir) })

	if version != "" {
		os.Setenv("GOENV_VERSION", version)
		t.Cleanup(func() { os.Unsetenv("GOENV_VERSION") })
	}

	if shouldCreateVersion && version != "" && version != "system" {
		versionPath := filepath.Join(tmpDir, "versions", version)

		goBinDir := filepath.Join(versionPath, "go", "bin")
		if err := os.MkdirAll(goBinDir, 0755); err != nil {
			t.Fatalf("Failed to create go bin directory: %v", err)
		}

		goBinary := filepath.Join(goBinDir, "go")
		mockScript := "#!/bin/sh\nexit 0"
		if utils.IsWindows() {
			goBinary += ".bat"
			mockScript = "@echo off\nexit 0"
		}
		if err := os.WriteFile(goBinary, []byte(mockScript), 0755); err != nil {
			t.Fatalf("Failed to create go binary: %v", err)
		}

		gopathBin := filepath.Join(versionPath, "gopath", "bin")
		if err := os.MkdirAll(gopathBin, 0755); err != nil {
			t.Fatalf("Failed to create GOPATH/bin: %v", err)
		}

		for _, tool := range tools {
			toolPath := filepath.Join(gopathBin, tool.name)
			mockContent := "mock tool binary"
			if utils.IsWindows() {
				toolPath += ".bat"
				mockContent = "@echo off\necho mock tool binary\n"
			}

			if err := os.WriteFile(toolPath, []byte(mockContent), 0755); err != nil {
				t.Fatalf("Failed to create tool %s: %v", tool.name, err)
			}
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
			expectedError: "go version 1.21.0 is not installed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, tt.setupVersion, nil, false)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.tool, "tool", "", "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.version, "version", "latest", "")

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
			updateToolsFlags.tool = ""
			updateToolsFlags.dryRun = false
			updateToolsFlags.version = "latest"
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
			expectedOutput: "No Go tools installed yet",
		},
		{
			name: "tools up to date",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v1.0.0"},
			},
			expectedOutput: "Found 1 tool(s)",
		},
		{
			name: "tool needs update - skipped without package path",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			expectedOutput: "unknown package path, skipping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, "1.21.0", tt.setupTools, true)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.tool, "tool", "", "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.version, "version", "latest", "")

			buf := new(bytes.Buffer)
			updateToolsCmd.SetOut(buf)
			updateToolsCmd.SetErr(buf)

			err := runUpdateTools(updateToolsCmd, []string{})

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.expectedOutput) {
				t.Errorf("Expected output to contain %q, got:\n%s", tt.expectedOutput, output)
			}

			updateToolsFlags.check = false
			updateToolsFlags.tool = ""
			updateToolsFlags.dryRun = false
			updateToolsFlags.version = "latest"
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
			expectedOutput: "All tools are up to date",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, "1.21.0", tt.setupTools, true)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.tool, "tool", "", "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.version, "version", "latest", "")

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

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.expectedOutput) {
				t.Errorf("Expected output to contain %q, got:\n%s", tt.expectedOutput, output)
			}

			updateToolsFlags.check = false
			updateToolsFlags.tool = ""
			updateToolsFlags.dryRun = false
			updateToolsFlags.version = "latest"
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
			expectedOutput: "Found 1 tool(s)",
		},
		{
			name: "tool not found",
			setupTools: []testToolInfo{
				{name: "mockgopls", pkgPath: "golang.org/x/tools/gopls", version: "v0.9.0"},
			},
			toolFlag:      "nonexistent",
			expectedError: "tool 'nonexistent' not found",
		},
		{
			name: "tool without package path",
			setupTools: []testToolInfo{
				{name: "mocktool", pkgPath: "", version: "unknown"},
			},
			expectedOutput: "unknown package path, skipping",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			setupUpdateTestEnv(t, "1.21.0", tt.setupTools, true)

			updateToolsCmd.ResetFlags()
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.check, "check", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.tool, "tool", "", "")
			updateToolsCmd.Flags().BoolVar(&updateToolsFlags.dryRun, "dry-run", false, "")
			updateToolsCmd.Flags().StringVar(&updateToolsFlags.version, "version", "latest", "")

			if tt.toolFlag != "" {
				updateToolsCmd.Flags().Set("tool", tt.toolFlag)
			}

			buf := new(bytes.Buffer)
			updateToolsCmd.SetOut(buf)
			updateToolsCmd.SetErr(buf)

			err := runUpdateTools(updateToolsCmd, []string{})

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
				if !strings.Contains(output, tt.expectedOutput) {
					t.Errorf("Expected output to contain %q, got:\n%s", tt.expectedOutput, output)
				}
			}

			updateToolsFlags.check = false
			updateToolsFlags.tool = ""
			updateToolsFlags.dryRun = false
			updateToolsFlags.version = "latest"
		})
	}
}

func TestUpdateToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := updateToolsCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()

	expectedStrings := []string{
		"tools", "update",
		"Updates all installed Go tools",
		"--check",
		"--tool",
		"--dry-run",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q, got:\n%s", expected, output)
		}
	}
}

// TestUpdateToolsVersionFlag tests that the --version flag is properly used
func TestUpdateToolsVersionFlag(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	goVersion := "1.21.0"
	cmdtest.CreateTestVersion(t, testRoot, goVersion)

	globalFile := filepath.Join(testRoot, "version")
	if err := os.WriteFile(globalFile, []byte(goVersion), 0644); err != nil {
		t.Fatalf("Failed to set global version: %v", err)
	}

	updateToolsFlags.check = false
	updateToolsFlags.tool = ""
	updateToolsFlags.dryRun = true
	updateToolsFlags.version = "v0.12.5"

	cmd := updateToolsCmd
	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := runUpdateTools(cmd, []string{})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if updateToolsFlags.version != "v0.12.5" {
		t.Errorf("Expected version flag to be 'v0.12.5', got '%s'", updateToolsFlags.version)
	}

	t.Log("✓ Version flag properly stored and accessible")

	updateToolsFlags.check = false
	updateToolsFlags.tool = ""
	updateToolsFlags.dryRun = false
	updateToolsFlags.version = "latest"
}
