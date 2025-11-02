package tools

import (
	"bytes"
	"encoding/json"
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

func TestListCommand_AllFlag(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple versions with tools
	versionsTools := map[string][]string{
		"1.21.0": {"gopls", "staticcheck"},
		"1.22.0": {"gopls"},
		"1.23.0": {"gopls", "staticcheck", "gofmt"},
	}

	for version, tools := range versionsTools {
		// Create version with tool directory
		cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)

		// Create individual tool binaries using helper (handles .bat on Windows)
		cfg := &config.Config{Root: tmpDir}
		binPath := cfg.VersionGopathBin(version)

		for _, tool := range tools {
			cmdtest.CreateToolExecutable(t, binPath, tool)
		}
	}

	// Test with --all flag
	listAllVersions = true
	defer func() { listAllVersions = false }()

	// Get all versions
	mgr := manager.NewManager(cfg)
	foundVersions, err := mgr.ListInstalledVersions()
	require.NoError(t, err, "ListInstalledVersions failed")

	assert.Len(t, foundVersions, 3, "Expected 3 versions")

	// Get tools for each version
	for version, expectedTools := range versionsTools {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		require.NoError(t, err, "ListForVersion failed for")

		// Extract tool names from Tool structs
		var toolNames []string
		for _, tool := range toolList {
			toolNames = append(toolNames, tool.Name)
		}

		assert.Len(t, toolNames, len(expectedTools), "Version : expected tools %v", version)

		for _, expected := range expectedTools {
			found := false
			for _, actual := range toolNames {
				if actual == expected {
					found = true
					break
				}
			}
			assert.True(t, found, "Version : expected tool not found")
		}
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create a version with tools
	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err)

	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		cmdtest.CreateToolExecutable(t, binPath, tool)
	}

	// Enable JSON output
	listToolsJSON = true
	defer func() { listToolsJSON = false }()

	// Get tools
	toolList, err := toolspkg.ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	// Extract tool names from Tool structs
	var foundTools []string
	for _, tool := range toolList {
		foundTools = append(foundTools, tool.Name)
	}

	// Verify we can marshal to JSON
	type versionTools struct {
		Version string   `json:"version"`
		Tools   []string `json:"tools"`
	}

	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		Versions      []versionTools `json:"versions"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		Versions: []versionTools{
			{
				Version: version,
				Tools:   foundTools,
			},
		},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err, "Failed to marshal JSON")

	// Verify JSON is valid
	var parsed jsonOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Failed to unmarshal JSON")

	assert.Equal(t, "1", parsed.SchemaVersion, "Expected schema_version '1'")

	assert.Len(t, parsed.Versions, 1, "Expected 1 version")

	assert.Equal(t, version, parsed.Versions[0].Version, "Expected version")
}

func TestListCommand_EmptyVersion(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create version with no tools
	version := "1.23.0"
	versionPath := filepath.Join(tmpDir, "versions", version)
	err = utils.EnsureDirWithContext(versionPath, "create test directory")
	require.NoError(t, err)

	// No bin directory created - should return empty
	toolList, err := toolspkg.ListForVersion(cfg, version)
	require.NoError(t, err)

	assert.Len(t, toolList, 0, "Expected no tools")
}

func TestListCommand_HiddenFiles(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err)

	// Create regular and hidden files (use helper for gopls to handle .bat on Windows)
	cmdtest.CreateToolExecutable(t, binPath, "gopls")
	testutil.WriteTestFile(t, filepath.Join(binPath, ".hidden"), []byte("fake"), utils.PermFileExecutable)

	toolList, err := toolspkg.ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	// Extract tool names from Tool structs
	var toolNames []string
	for _, tool := range toolList {
		toolNames = append(toolNames, tool.Name)
	}

	// Should only get non-hidden files
	assert.Len(t, toolNames, 1, "Expected 1 tool")

	assert.False(t, len(toolNames) > 0 && toolNames[0] != "gopls", "Expected 'gopls'")
}

func TestListCommand_PlatformVariants(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err)

	// Create tool with platform variants
	testutil.WriteTestFile(t, filepath.Join(binPath, "gopls"), []byte("fake"), utils.PermFileExecutable)
	testutil.WriteTestFile(t, filepath.Join(binPath, "gopls.exe"), []byte("fake"), utils.PermFileExecutable)

	toolList, err := toolspkg.ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	// Extract tool names from Tool structs
	var toolNames []string
	for _, tool := range toolList {
		toolNames = append(toolNames, tool.Name)
	}

	// Should deduplicate - only get base name
	assert.Len(t, toolNames, 1, "Expected 1 tool (deduplicated)")

	assert.False(t, len(toolNames) > 0 && toolNames[0] != "gopls", "Expected 'gopls'")
}

func TestListCommand_RunE(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create a version with tools
	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err)

	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		cmdtest.CreateToolExecutable(t, binPath, tool)
	}

	// Create .go-version to set current version
	goVersionFile := filepath.Join(tmpDir, ".go-version")
	testutil.WriteTestFile(t, goVersionFile, []byte(version), utils.PermFileDefault)

	// Test the command can be created
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	// Verify tools are found
	toolList, err := toolspkg.ListForVersion(cfg, version)
	require.NoError(t, err, "ListForVersion failed")

	// Extract tool names from Tool structs
	var foundTools []string
	for _, tool := range toolList {
		foundTools = append(foundTools, tool.Name)
	}

	assert.Len(t, foundTools, 2, "Expected 2 tools")
}

func TestListCommand_MultipleVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple versions
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		// Create Go binary for version using helper (handles .bat on Windows)
		cmdtest.CreateTestBinary(t, tmpDir, v, "go")

		// Create tools using helper (handles .bat on Windows)
		binPath := filepath.Join(tmpDir, "versions", v, "gopath", "bin")
		err = utils.EnsureDirWithContext(binPath, "create test directory")
		require.NoError(t, err)
		cmdtest.CreateToolExecutable(t, binPath, "gopls")
	}

	// Get all versions
	mgr := manager.NewManager(cfg)
	foundVersions, err := mgr.ListInstalledVersions()
	require.NoError(t, err, "ListInstalledVersions failed")

	assert.Len(t, foundVersions, 3, "Expected 3 versions")

	// Verify each version has gopls
	for _, version := range foundVersions {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		require.NoError(t, err, "ListForVersion failed for")

		// Extract tool names from Tool structs
		var toolNames []string
		for _, tool := range toolList {
			toolNames = append(toolNames, tool.Name)
		}

		assert.Len(t, toolNames, 1, "Version : expected 1 tool %v", version)

		assert.False(t, len(toolNames) > 0 && toolNames[0] != "gopls", "Version : expected 'gopls'")
	}
}
