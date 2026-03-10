package tools

import (
	"encoding/json"
	"path/filepath"
	"slices"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestStatusCommand_NoVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err)

	mgr := manager.NewManager(cfg, nil)
	versions, err := mgr.ListInstalledVersions()
	require.NoError(t, err)

	assert.Len(t, versions, 0, "Expected no versions")
}

func TestStatusCommand_JSONStructure(t *testing.T) {
	var err error
	// Test JSON output structure
	status := toolStatus{
		Name:          "gopls",
		TotalVersions: 3,
		InstalledIn:   3,
		VersionPresence: map[string]bool{
			"1.21.0": true,
			"1.22.0": true,
			"1.23.0": true,
		},
		ConsistencyScore: 100.0,
	}

	type jsonOutput struct {
		SchemaVersion string       `json:"schema_version"`
		GoVersions    []string     `json:"go_versions"`
		Tools         []toolStatus `json:"tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		GoVersions:    []string{"1.21.0", "1.22.0", "1.23.0"},
		Tools:         []toolStatus{status},
	}

	// Verify we can marshal to JSON
	data, err := json.Marshal(output)
	require.NoError(t, err, "Failed to marshal JSON")

	// Verify we can unmarshal
	var parsed jsonOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Failed to unmarshal JSON")

	assert.Equal(t, "1", parsed.SchemaVersion, "Expected schema_version '1'")

	assert.Len(t, parsed.GoVersions, 3, "Expected 3 Go versions")

	assert.Len(t, parsed.Tools, 1, "Expected 1 tool")

	tool := parsed.Tools[0]
	assert.Equal(t, "gopls", tool.Name, "Expected name 'gopls'")
	assert.Equal(t, 3, tool.TotalVersions, "Expected total_versions 3")
	assert.Equal(t, 3, tool.InstalledIn, "Expected installed_in 3")
	assert.Equal(t, 100.0, tool.ConsistencyScore, "Expected consistency_score 100.0")
}

func TestStatusCommand_ConsistencyCalculation(t *testing.T) {
	tests := []struct {
		name          string
		totalVersions int
		installedIn   int
		expectedScore float64
	}{
		{
			name:          "fully consistent",
			totalVersions: 3,
			installedIn:   3,
			expectedScore: 100.0,
		},
		{
			name:          "two thirds consistent",
			totalVersions: 3,
			installedIn:   2,
			expectedScore: 66.66666666666666,
		},
		{
			name:          "one third consistent",
			totalVersions: 3,
			installedIn:   1,
			expectedScore: 33.33333333333333,
		},
		{
			name:          "not installed",
			totalVersions: 3,
			installedIn:   0,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := float64(tt.installedIn) / float64(tt.totalVersions) * 100

			// Use a small epsilon for floating point comparison
			epsilon := 0.0001
			assert.False(t, score < tt.expectedScore-epsilon || score > tt.expectedScore+epsilon, "Expected score")
		})
	}
}

func TestStatusCommand_Categorization(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple versions with different tool installations
	versionsTools := map[string][]string{
		"1.21.0": {"gopls", "staticcheck", "gofmt"},
		"1.22.0": {"gopls", "staticcheck"},
		"1.23.0": {"gopls", "golangci-lint"},
	}

	for version, tools := range versionsTools {
		// Create version with tool directory
		cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)

		// Create individual tool binaries using helper (handles .bat on Windows)
		binPath := cfg.VersionGopathBin(version)
		for _, tool := range tools {
			cmdtest.CreateToolExecutable(t, binPath, tool)
		}
	}

	// Get versions and collect tools
	mgr := manager.NewManager(cfg, nil)
	foundVersions, err := mgr.ListInstalledVersions()
	require.NoError(t, err, "ListInstalledVersions failed")

	toolsByVersion := make(map[string][]string)
	allToolNames := make(map[string]bool)

	for _, version := range foundVersions {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		require.NoError(t, err, "ListForVersion failed for")

		// Extract tool names
		var toolNames []string
		for _, tool := range toolList {
			toolNames = append(toolNames, tool.Name)
		}

		toolsByVersion[version] = toolNames
		for _, toolName := range toolNames {
			allToolNames[toolName] = true
		}
	}

	// Verify gopls is consistent (in all versions)
	goplsCount := 0
	for _, tools := range toolsByVersion {
		if slices.Contains(tools, "gopls") {
			goplsCount++
		}
	}
	assert.Equal(t, 3, goplsCount, "Expected gopls in all 3 versions, found in")

	// Verify staticcheck is partial (in 2 versions)
	staticcheckCount := 0
	for _, tools := range toolsByVersion {
		if slices.Contains(tools, "staticcheck") {
			staticcheckCount++
		}
	}
	assert.Equal(t, 2, staticcheckCount, "Expected staticcheck in 2 versions, found in")

	// Verify gofmt is version-specific (in 1 version)
	gofmtCount := 0
	for _, tools := range toolsByVersion {
		if slices.Contains(tools, "gofmt") {
			gofmtCount++
		}
	}
	assert.Equal(t, 1, gofmtCount, "Expected gofmt in 1 version, found in")

	// Verify golangci-lint is version-specific (in 1 version)
	golangciCount := 0
	for _, tools := range toolsByVersion {
		if slices.Contains(tools, "golangci-lint") {
			golangciCount++
		}
	}
	assert.Equal(t, 1, golangciCount, "Expected golangci-lint in 1 version, found in")
}

func TestStatusCommand_EmptyTools(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create versions but no tools
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		// Create Go binary using helper (handles .bat on Windows)
		cmdtest.CreateTestBinary(t, tmpDir, v, "go")
	}

	mgr := manager.NewManager(cfg, nil)
	foundVersions, err := mgr.ListInstalledVersions()
	require.NoError(t, err, "ListInstalledVersions failed")

	assert.Len(t, foundVersions, 3, "Expected 3 versions")

	// All versions should have no tools
	allToolNames := make(map[string]bool)
	for _, version := range foundVersions {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		require.NoError(t, err, "ListForVersion failed for")

		assert.Len(t, toolList, 0, "Version : expected no tools %v", version)

		for _, tool := range toolList {
			allToolNames[tool.Name] = true
		}
	}

	assert.Len(t, allToolNames, 0, "Expected no tools across all versions")
}

func TestStatusCommand_Command(t *testing.T) {
	// Test that the command can be created
	cmd := newStatusCommand()

	require.NotNil(t, cmd, "Expected command to be created")

	assert.Equal(t, "status", cmd.Use, "Expected Use 'status'")

	assert.Equal(t, "Show tool installation consistency across versions", cmd.Short, "Unexpected Short description")

	// Check flags
	jsonFlag := cmd.Flags().Lookup("json")
	assert.NotNil(t, jsonFlag, "Expected --json flag to exist")
}

func TestConsistencyScoreCalculation(t *testing.T) {
	tests := []struct {
		name          string
		totalVersions int
		installedIn   int
		expectedScore float64
	}{
		{
			name:          "fully consistent",
			totalVersions: 3,
			installedIn:   3,
			expectedScore: 100.0,
		},
		{
			name:          "partially consistent",
			totalVersions: 3,
			installedIn:   2,
			expectedScore: 66.66666666666666,
		},
		{
			name:          "single version",
			totalVersions: 3,
			installedIn:   1,
			expectedScore: 33.33333333333333,
		},
		{
			name:          "not installed",
			totalVersions: 3,
			installedIn:   0,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := float64(tt.installedIn) / float64(tt.totalVersions) * 100
			epsilon := 0.0001
			assert.False(t, score < tt.expectedScore-epsilon || score > tt.expectedScore+epsilon, "Expected score")
		})
	}
}

func TestStatusCommand_MultipleTools(t *testing.T) {
	var err error
	// Test creating multiple tool statuses
	statuses := []toolStatus{
		{
			Name:          "gopls",
			TotalVersions: 3,
			InstalledIn:   3,
			VersionPresence: map[string]bool{
				"1.21.0": true,
				"1.22.0": true,
				"1.23.0": true,
			},
			ConsistencyScore: 100.0,
		},
		{
			Name:          "staticcheck",
			TotalVersions: 3,
			InstalledIn:   2,
			VersionPresence: map[string]bool{
				"1.21.0": true,
				"1.22.0": false,
				"1.23.0": true,
			},
			ConsistencyScore: 66.66666666666666,
		},
		{
			Name:          "gofmt",
			TotalVersions: 3,
			InstalledIn:   1,
			VersionPresence: map[string]bool{
				"1.21.0": true,
				"1.22.0": false,
				"1.23.0": false,
			},
			ConsistencyScore: 33.33333333333333,
		},
	}

	type jsonOutput struct {
		SchemaVersion string       `json:"schema_version"`
		GoVersions    []string     `json:"go_versions"`
		Tools         []toolStatus `json:"tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		GoVersions:    []string{"1.21.0", "1.22.0", "1.23.0"},
		Tools:         statuses,
	}

	data, err := json.Marshal(output)
	require.NoError(t, err, "Failed to marshal JSON")

	var parsed jsonOutput
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err, "Failed to unmarshal JSON")

	assert.Len(t, parsed.Tools, 3, "Expected 3 tools")

	// Categorize tools
	var fullyInstalled, partiallyInstalled, singleVersion []toolStatus
	for _, status := range parsed.Tools {
		if status.InstalledIn == status.TotalVersions {
			fullyInstalled = append(fullyInstalled, status)
		} else if status.InstalledIn == 1 {
			singleVersion = append(singleVersion, status)
		} else {
			partiallyInstalled = append(partiallyInstalled, status)
		}
	}

	assert.Len(t, fullyInstalled, 1, "Expected 1 fully installed tool")

	assert.Len(t, partiallyInstalled, 1, "Expected 1 partially installed tool")

	assert.Len(t, singleVersion, 1, "Expected 1 single-version tool")
}
