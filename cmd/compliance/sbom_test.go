package compliance

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSBOMProject_FlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		setupFlags  func()
		expectError bool
		errorText   string
	}{
		{
			name: "both image and dir specified",
			setupFlags: func() {
				sbomImage = "myimage:latest"
				sbomDir = "/some/dir"
			},
			expectError: true,
			errorText:   "cannot specify both --image and --dir",
		},
		{
			name: "image with non-syft tool",
			setupFlags: func() {
				sbomImage = "myimage:latest"
				sbomDir = "."
				sbomTool = "cyclonedx-gomod"
			},
			expectError: true,
			errorText:   "--image is only supported with --tool=syft",
		},
		{
			name: "valid cyclonedx-gomod",
			setupFlags: func() {
				sbomImage = ""
				sbomDir = "."
				sbomTool = "cyclonedx-gomod"
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			sbomImage = ""
			sbomDir = "."
			sbomTool = "cyclonedx-gomod"

			// Setup
			tt.setupFlags()

			// Create temp directory for test
			tmpDir := t.TempDir()
			os.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			defer os.Unsetenv(utils.GoenvEnvVarRoot.String())

			// Create a fake Go version so GetCurrentVersion() succeeds
			testVersion := "1.21.0"
			versionDir := filepath.Join(tmpDir, "versions", testVersion, "bin")
			err := os.MkdirAll(versionDir, 0755)
			require.NoError(t, err, "Failed to create version dir")

			// Create a fake go binary so version validation passes
			goBinary := filepath.Join(versionDir, "go")
			err = os.WriteFile(goBinary, []byte("#!/bin/sh\necho go1.21.0\n"), 0755)
			require.NoError(t, err, "Failed to create fake go binary")

			// Create global version file
			versionFile := filepath.Join(tmpDir, "version")
			err = os.WriteFile(versionFile, []byte(testVersion+"\n"), 0644)
			require.NoError(t, err, "Failed to create version file")

			// Create mock SBOM tool in version bin directory for valid test cases
			if !tt.expectError {
				toolPath := filepath.Join(versionDir, sbomTool)
				var content string
				if utils.IsWindows() {
					toolPath += ".exe"
					content = "@echo off\necho {}\n"
				} else {
					content = "#!/bin/sh\necho '{}'\n"
				}
				testutil.WriteTestFile(t, toolPath, []byte(content), utils.PermFileExecutable)
			}

			// Change to tmpDir so go.mod from repo doesn't interfere
			oldWd, err := os.Getwd()
			require.NoError(t, err, "Failed to get working directory")
			err = os.Chdir(tmpDir)
			require.NoError(t, err, "Failed to change directory")
			defer func() {
				_ = os.Chdir(oldWd)
			}()

			// Run command
			cmd := sbomProjectCmd
			cmd.SetArgs([]string{})
			err = cmd.RunE(cmd, []string{})

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.errorText)
				} else if !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error containing %q, got %q", tt.errorText, err.Error())
				}
			} else if err != nil {
				t.Errorf("Expected no error, got %q", err.Error())
			}
		})
	}
}

func TestResolveSBOMTool(t *testing.T) {
	var err error
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create host bin directory
	hostBinDir := cfg.HostBinDir()
	err = utils.EnsureDirWithContext(hostBinDir, "create test directory")
	require.NoError(t, err, "Failed to create host bin dir")

	// Create mock tool
	toolName := "cyclonedx-gomod"
	toolPath := filepath.Join(hostBinDir, toolName)
	var content string
	if utils.IsWindows() {
		toolPath += ".exe"
		content = "@echo off\necho mock"
	} else {
		content = "#!/bin/sh\necho mock"
	}

	testutil.WriteTestFile(t, toolPath, []byte(content), utils.PermFileExecutable)

	// Test resolution with global version context (host bin accessible)
	resolvedPath, err := resolveSBOMTool(cfg, toolName, "1.21.0", "")
	require.NoError(t, err, "Failed to resolve tool")

	assert.Equal(t, toolPath, resolvedPath, "Expected path")
}

func TestResolveSBOMTool_NotFound(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Root: tmpDir,
	}

	// Test resolution for non-existent tool with global context
	_, err := resolveSBOMTool(cfg, "nonexistent-tool", "1.21.0", "")
	assert.Error(t, err, "Expected error for non-existent tool")

	assert.Contains(t, err.Error(), "not found", "Expected 'not found' error %v", err)

	assert.Contains(t, err.Error(), "goenv tools install", "Expected installation instructions in error %v", err)
}

func TestBuildCycloneDXCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	tests := []struct {
		name        string
		format      string
		modulesOnly bool
		output      string
		expectError bool
		expectArgs  []string
	}{
		{
			name:        "json format",
			format:      "cyclonedx-json",
			modulesOnly: false,
			output:      "sbom.json",
			expectArgs:  []string{"mod", "-output", "sbom.json", "-json"},
		},
		{
			name:        "modules only",
			format:      "cyclonedx-json",
			modulesOnly: true,
			output:      "sbom.json",
			expectArgs:  []string{"mod", "-output", "sbom.json", "-json", "-licenses", "-type", "library"},
		},
		{
			name:        "unsupported format",
			format:      "spdx-json",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set globals
			sbomFormat = tt.format
			sbomModulesOnly = tt.modulesOnly
			sbomOutput = tt.output
			sbomToolArgs = ""

			cmd, err := buildCycloneDXCommand("/mock/tool", cfg)

			if tt.expectError {
				assert.Error(t, err, "Expected error, got nil")
				return
			}

			require.NoError(t, err)

			// Check args
			for _, expectedArg := range tt.expectArgs {
				found := false
				for _, arg := range cmd.Args[1:] { // Skip binary name
					if arg == expectedArg {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected arg not found in")
			}
		})
	}
}

func TestBuildSyftCommand(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	tests := []struct {
		name        string
		format      string
		image       string
		dir         string
		output      string
		expectError bool
		expectArgs  []string
	}{
		{
			name:       "directory scan with cyclonedx",
			format:     "cyclonedx-json",
			dir:        ".",
			output:     "sbom.json",
			expectArgs: []string{".", "-o", "cyclonedx-json=sbom.json", "-q"},
		},
		{
			name:       "image scan with spdx",
			format:     "spdx-json",
			image:      "myimage:latest",
			output:     "sbom.json",
			expectArgs: []string{"myimage:latest", "-o", "spdx-json=sbom.json", "-q"},
		},
		{
			name:        "unsupported format",
			format:      "invalid-format",
			dir:         ".",
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set globals
			sbomFormat = tt.format
			sbomImage = tt.image
			sbomDir = tt.dir
			sbomOutput = tt.output
			sbomToolArgs = ""

			cmd, err := buildSyftCommand("/mock/syft", cfg)

			if tt.expectError {
				assert.Error(t, err, "Expected error, got nil")
				return
			}

			require.NoError(t, err)

			// Check args
			for _, expectedArg := range tt.expectArgs {
				found := false
				for _, arg := range cmd.Args[1:] { // Skip binary name
					if arg == expectedArg {
						found = true
						break
					}
				}
				assert.True(t, found, "Expected arg not found in")
			}
		})
	}
}
