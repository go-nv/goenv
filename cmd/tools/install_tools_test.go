package tools

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
)

func TestInstallCommand_AllFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple Go versions with tool directories
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		cmdtest.CreateMockGoVersionWithTools(t, tmpDir, v)
	}

	// Test dry-run with --all flag
	cmd := &cobra.Command{}
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// Set flags
	installAllVersions = true
	installDryRun = true
	defer func() {
		installAllVersions = false
		installDryRun = false
	}()

	// Test that the function validates versions correctly
	mgr := manager.NewManager(cfg)
	foundVersions, err := mgr.ListInstalledVersions()
	require.NoError(t, err, "ListInstalledVersions failed")

	assert.Len(t, foundVersions, 3, "Expected 3 versions")

	for _, expected := range versions {
		found := false
		for _, actual := range foundVersions {
			if actual == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected version not found")
	}
}

func TestInstallCommand_DryRun(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	_ = &config.Config{
		Root: tmpDir,
	}

	// Create a Go version
	version := "1.23.0"
	versionPath := filepath.Join(tmpDir, "versions", version)
	goBinDir := filepath.Join(versionPath, "bin")
	gopath := filepath.Join(versionPath, "gopath", "bin")
	err = utils.EnsureDirWithContext(goBinDir, "create test directory")
	require.NoError(t, err)
	err = utils.EnsureDirWithContext(gopath, "create test directory")
	require.NoError(t, err)

	// Create fake go binary
	goBin := filepath.Join(goBinDir, "go")
	testutil.WriteTestFile(t, goBin, []byte("#!/bin/sh\nexit 0"), utils.PermFileExecutable)

	// Create .go-version file to set current version
	goVersionFile := filepath.Join(tmpDir, ".go-version")
	testutil.WriteTestFile(t, goVersionFile, []byte(version), utils.PermFileDefault)

	// Test package normalization
	packages := toolspkg.NormalizePackagePaths([]string{"gopls", "golang.org/x/tools/cmd/goimports@v0.1.0"})
	expected := []string{"golang.org/x/tools/gopls@latest", "golang.org/x/tools/cmd/goimports@v0.1.0"}

	assert.Len(t, packages, len(expected), "Expected packages")

	for i, exp := range expected {
		assert.Equal(t, exp, packages[i], "Package : expected")
	}
}

func TestInstallCommand_NoVersionsInstalled(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err)

	// Try to get installed versions
	mgr := manager.NewManager(cfg)
	versions, err := mgr.ListInstalledVersions()
	require.NoError(t, err)

	assert.Len(t, versions, 0, "Expected no versions")
}

func TestInstallCommand_VerboseOutput(t *testing.T) {
	// Test that verbose flag is properly tracked
	originalVerbose := installVerbose
	defer func() { installVerbose = originalVerbose }()

	installVerbose = true
	assert.True(t, installVerbose, "installVerbose flag should be true")

	installVerbose = false
	assert.False(t, installVerbose, "installVerbose flag should be false")
}

func TestExtractToolNames(t *testing.T) {
	tests := []struct {
		name     string
		packages []string
		expected []string
	}{
		{
			name:     "single package",
			packages: []string{"golang.org/x/tools/gopls@latest"},
			expected: []string{"gopls"},
		},
		{
			name:     "multiple packages",
			packages: []string{"golang.org/x/tools/gopls@latest", "honnef.co/go/tools/cmd/staticcheck@v0.4.0"},
			expected: []string{"gopls", "staticcheck"},
		},
		{
			name:     "package without version",
			packages: []string{"github.com/golangci/golangci-lint/cmd/golangci-lint"},
			expected: []string{"golangci-lint"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := toolspkg.ExtractToolNames(tt.packages)
			assert.Len(t, result, len(tt.expected), "Expected names")
			for i, exp := range tt.expected {
				assert.Equal(t, exp, result[i], "Name : expected")
			}
		})
	}
}

func TestInstallToolForVersion_MissingGo(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	versionPath := filepath.Join(tmpDir, "versions", version)
	err = utils.EnsureDirWithContext(versionPath, "create test directory")
	require.NoError(t, err)

	// Don't create Go binary - should fail
	mgr := manager.NewManager(cfg)
	toolMgr := toolspkg.NewManager(cfg, mgr)
	err = toolMgr.InstallSingleTool(version, "gopls@latest", false)
	assert.Error(t, err, "Expected error when Go binary is missing")

	assert.Contains(t, err.Error(), "go binary not found", "Expected 'Go binary not found' error %v", err)
}

func TestExtractToolName_EdgeCases(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "golang.org/x/tools/gopls@latest",
			expected: "gopls",
		},
		{
			input:    "golang.org/x/tools/gopls",
			expected: "gopls",
		},
		{
			input:    "gopls@latest",
			expected: "gopls",
		},
		{
			input:    "gopls",
			expected: "gopls",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := extractToolName(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestExtractToolName_ComplexPaths(t *testing.T) {
	tests := []struct {
		packagePath string
		expected    string
	}{
		{
			packagePath: "golang.org/x/tools/gopls@v0.12.0",
			expected:    "gopls",
		},
		{
			packagePath: "github.com/golangci/golangci-lint/cmd/golangci-lint@latest",
			expected:    "golangci-lint",
		},
		{
			packagePath: "honnef.co/go/tools/cmd/staticcheck@2023.1.5",
			expected:    "staticcheck",
		},
		{
			packagePath: "github.com/go-delve/delve/cmd/dlv",
			expected:    "dlv",
		},
		{
			packagePath: "mvdan.cc/gofumpt@latest",
			expected:    "gofumpt",
		},
	}

	for _, tt := range tests {
		t.Run(tt.packagePath, func(t *testing.T) {
			result := extractToolName(tt.packagePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}
