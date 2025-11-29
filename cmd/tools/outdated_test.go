package tools

import (
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
)

func TestOutdatedCommand_NoVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err)

	// Get installed versions - should be empty
	mgr := manager.NewManager(cfg)
	versions, err := mgr.ListInstalledVersions()
	require.NoError(t, err)

	assert.Len(t, versions, 0, "Expected no versions")
}

func TestOutdatedCommand_JSONStructure(t *testing.T) {
	var err error
	// Test JSON output structure
	outdated := outdatedTool{
		Name:           "gopls",
		GoVersion:      "1.23.0",
		CurrentVersion: "v0.12.0",
		LatestVersion:  "v0.13.2",
		PackagePath:    "golang.org/x/tools/gopls",
	}

	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		OutdatedTools []outdatedTool `json:"outdated_tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		OutdatedTools: []outdatedTool{outdated},
	}

	// Verify we can marshal to JSON
	data, err := json.Marshal(output)
	require.NoError(t, err, "Failed to marshal JSON")

	// Verify we can unmarshal
	var parsed jsonOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Failed to unmarshal JSON")

	assert.Equal(t, "1", parsed.SchemaVersion, "Expected schema_version '1'")

	assert.Len(t, parsed.OutdatedTools, 1, "Expected 1 outdated tool")

	tool := parsed.OutdatedTools[0]
	assert.Equal(t, "gopls", tool.Name, "Expected name 'gopls'")
	assert.Equal(t, "1.23.0", tool.GoVersion, "Expected go_version '1.23.0'")
	assert.Equal(t, "v0.12.0", tool.CurrentVersion, "Expected current_version 'v0.12.0'")
	assert.Equal(t, "v0.13.2", tool.LatestVersion, "Expected latest_version 'v0.13.2'")
	assert.Equal(t, "golang.org/x/tools/gopls", tool.PackagePath, "Expected package_path 'golang.org/x/tools/gopls'")
}

func TestOutdatedCommand_EmptyResult(t *testing.T) {
	var err error
	// Test empty outdated tools result
	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		OutdatedTools []outdatedTool `json:"outdated_tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		OutdatedTools: []outdatedTool{},
	}

	data, err := json.Marshal(output)
	require.NoError(t, err, "Failed to marshal JSON")

	var parsed jsonOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Failed to unmarshal JSON")

	assert.Len(t, parsed.OutdatedTools, 0, "Expected 0 outdated tools")
}

func TestOutdatedCommand_MultipleTools(t *testing.T) {
	var err error
	// Test multiple outdated tools
	tools := []outdatedTool{
		{
			Name:           "gopls",
			GoVersion:      "1.21.0",
			CurrentVersion: "v0.12.0",
			LatestVersion:  "v0.13.2",
			PackagePath:    "golang.org/x/tools/gopls",
		},
		{
			Name:           "staticcheck",
			GoVersion:      "1.21.0",
			CurrentVersion: "v0.4.0",
			LatestVersion:  "v0.4.6",
			PackagePath:    "honnef.co/go/tools/cmd/staticcheck",
		},
		{
			Name:           "gopls",
			GoVersion:      "1.23.0",
			CurrentVersion: "v0.12.0",
			LatestVersion:  "v0.13.2",
			PackagePath:    "golang.org/x/tools/gopls",
		},
	}

	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		OutdatedTools []outdatedTool `json:"outdated_tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		OutdatedTools: tools,
	}

	data, err := json.Marshal(output)
	require.NoError(t, err, "Failed to marshal JSON")

	var parsed jsonOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Failed to unmarshal JSON")

	assert.Len(t, parsed.OutdatedTools, 3, "Expected 3 outdated tools")

	// Verify grouping by Go version
	versionGroups := make(map[string][]outdatedTool)
	for _, tool := range parsed.OutdatedTools {
		versionGroups[tool.GoVersion] = append(versionGroups[tool.GoVersion], tool)
	}

	assert.Len(t, versionGroups, 2, "Expected 2 Go versions")

	assert.Len(t, versionGroups["1.21.0"], 2, "Expected 2 outdated tools for Go 1.21.0")

	assert.Len(t, versionGroups["1.23.0"], 1, "Expected 1 outdated tool for Go 1.23.0")
}

func TestOutdatedCommand_Command(t *testing.T) {
	// Test that the command can be created
	cmd := newOutdatedCommand()

	require.NotNil(t, cmd, "Expected command to be created")

	assert.Equal(t, "outdated", cmd.Use, "Expected Use 'outdated'")

	assert.Equal(t, "Show outdated tools across all Go versions", cmd.Short, "Unexpected Short description")

	// Check flags
	jsonFlag := cmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag, "Expected --json flag to exist")
}

func TestOutdatedCommand_WithVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create versions but no tools
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		cmdtest.CreateMockGoVersion(t, tmpDir, v)
	}

	mgr := manager.NewManager(cfg)
	foundVersions, err := mgr.ListInstalledVersions()
	require.NoError(t, err, "ListInstalledVersions failed")

	assert.Len(t, foundVersions, 3, "Expected 3 versions")

	// No tools installed - outdated should be empty
	for _, version := range foundVersions {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		require.NoError(t, err, "ListForVersion failed for")

		assert.Len(t, toolList, 0, "Version : expected no tools %v", version)
	}
}
