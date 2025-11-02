package tools

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/testing/testutil"
)

// mockVersionManager implements VersionManager for testing
type mockVersionManager struct {
	versions []string
	err      error
}

func (m *mockVersionManager) ListInstalledVersions() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.versions, nil
}

func TestListForVersion(t *testing.T) {
	var err error
	// Create temporary directory structure
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create test tool binaries
	tools := []string{"goimports", "gopls", "dlv"}
	for _, tool := range tools {
		toolPath := filepath.Join(binPath, tool)
		testutil.WriteTestFile(t, toolPath, []byte("#!/bin/bash\necho test"), utils.PermFileExecutable)
	}

	// Test listing tools
	result, err := ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	assert.Len(t, result, 3, "Expected 3 tools")

	// Verify tool metadata
	for _, tool := range result {
		assert.NotEmpty(t, tool.Name, "Tool name should not be empty")
		assert.NotEmpty(t, tool.BinaryPath, "Tool binary path should not be empty")
		assert.Equal(t, version, tool.GoVersion, "Expected GoVersion")
		if tool.ModTime.IsZero() {
			t.Error("Tool ModTime should not be zero")
		}
	}
}

func TestListForVersion_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Test with non-existent version
	result, err := ListForVersion(cfg, "1.99.0")
	require.NoError(t, err, "ListForVersion should not error on missing directory")

	assert.Nil(t, result, "Expected nil result for non-existent version")
}

func TestListForVersion_EmptyDirectory(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Test with empty directory
	result, err := ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	assert.Len(t, result, 0, "Expected 0 tools in empty directory")
}

func TestListForVersion_PlatformVariants(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create tool with multiple platform variants
	variants := []string{"goimports", "goimports.exe", "goimports.darwin"}
	for _, variant := range variants {
		toolPath := filepath.Join(binPath, variant)
		testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)
	}

	// Test deduplication
	result, err := ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	// Should only return one entry despite multiple platform variants
	assert.Len(t, result, 1, "Expected 1 tool (deduplicated)")

	assert.Equal(t, "goimports", result[0].Name, "Expected tool name 'goimports'")
}

func TestListForVersion_HiddenFiles(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create hidden file
	hiddenPath := filepath.Join(binPath, ".hidden")
	testutil.WriteTestFile(t, hiddenPath, []byte("test"), utils.PermFileDefault)

	// Create normal tool
	toolPath := filepath.Join(binPath, "goimports")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	// Test that hidden files are excluded
	result, err := ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	assert.Len(t, result, 1, "Expected 1 tool (hidden file excluded)")

	assert.Equal(t, "goimports", result[0].Name, "Expected tool name 'goimports'")
}

func TestListAll(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Create tools for multiple versions
	versions := []string{"1.20.0", "1.21.0"}
	for _, version := range versions {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		err = utils.EnsureDirWithContext(binPath, "create test directory")
		require.NoError(t, err, "Failed to create bin directory")

		toolPath := filepath.Join(binPath, "goimports")
		testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)
	}

	mgr := &mockVersionManager{versions: versions}

	// Test listing all tools
	result, err := ListAll(cfg, mgr)
	require.NoError(t, err, "ListAll failed")

	assert.Len(t, result, 2, "Expected tools for 2 versions")

	for _, version := range versions {
		tools, ok := result[version]
		if !ok {
			t.Errorf("Expected tools for version %s", version)
			continue
		}
		assert.Len(t, tools, 1, "Expected 1 tool for version %v", version)
	}
}

func TestListAll_EmptyVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	mgr := &mockVersionManager{versions: []string{}}

	result, err := ListAll(cfg, mgr)
	require.NoError(t, err, "ListAll failed")

	assert.Len(t, result, 0, "Expected empty result")
}

func TestIsInstalled(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create test tool
	toolPath := filepath.Join(binPath, "goimports")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	tests := []struct {
		name     string
		version  string
		toolName string
		expected bool
	}{
		{
			name:     "installed tool",
			version:  version,
			toolName: "goimports",
			expected: true,
		},
		{
			name:     "not installed tool",
			version:  version,
			toolName: "gopls",
			expected: false,
		},
		{
			name:     "non-existent version",
			version:  "1.99.0",
			toolName: "goimports",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInstalled(cfg, tt.version, tt.toolName)
			assert.Equal(t, tt.expected, result, "IsInstalled(, ) = %v %v", tt.version, tt.toolName)
		})
	}
}

func TestIsInstalled_PlatformVariants(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create tool with .exe extension (Windows)
	toolPath := filepath.Join(binPath, "goimports.exe")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	// Should find tool even without extension
	assert.True(t, IsInstalled(cfg, version, "goimports"), "IsInstalled should find tool with .exe extension")
}

func TestGetToolInfo(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create test tool
	toolPath := filepath.Join(binPath, "goimports")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	// Test getting tool info
	tool, err := GetToolInfo(cfg, version, "goimports")
	require.NoError(t, err, "GetToolInfo failed")

	require.NotNil(t, tool, "Expected tool info, got nil")

	assert.Equal(t, "goimports", tool.Name, "Expected tool name 'goimports'")

	assert.Equal(t, version, tool.GoVersion, "Expected GoVersion")

	// Test non-existent tool
	tool, err = GetToolInfo(cfg, version, "nonexistent")
	require.NoError(t, err, "GetToolInfo should not error for missing tool")

	assert.Nil(t, tool, "Expected nil for non-existent tool")
}

func TestCollectUniqueTools(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Create same tools across multiple versions
	versions := []string{"1.20.0", "1.21.0"}
	for _, version := range versions {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		err = utils.EnsureDirWithContext(binPath, "create test directory")
		require.NoError(t, err, "Failed to create bin directory")

		// Both versions have goimports
		toolPath := filepath.Join(binPath, "goimports")
		testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)
	}

	// Add version-specific tool
	binPath := filepath.Join(tmpDir, "versions", "1.21.0", "gopath", "bin")
	toolPath := filepath.Join(binPath, "gopls")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	mgr := &mockVersionManager{versions: versions}

	// Test collecting unique tools
	result, err := CollectUniqueTools(cfg, mgr)
	require.NoError(t, err, "CollectUniqueTools failed")

	// Should have 2 unique tools: goimports and gopls
	assert.Len(t, result, 2, "Expected 2 unique tools %v", result)

	// Check that both tools are present
	hasGoimports := false
	hasGopls := false
	for _, name := range result {
		if name == "goimports" {
			hasGoimports = true
		}
		if name == "gopls" {
			hasGopls = true
		}
	}

	assert.True(t, hasGoimports, "Expected 'goimports' in unique tools")
	assert.True(t, hasGopls, "Expected 'gopls' in unique tools")
}
