package tools

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiVersionToolManagement is a comprehensive integration test
// that verifies tool management across multiple Go versions
func TestMultiVersionToolManagement(t *testing.T) {
	var err error
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create 3 Go versions
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		// Create Go binary using helper (handles .bat on Windows)
		cmdtest.CreateTestBinary(t, tmpDir, v, "go")

		// Create GOPATH/bin directory
		versionPath := filepath.Join(tmpDir, "versions", v)
		gopath := filepath.Join(versionPath, "gopath", "bin")
		err = utils.EnsureDirWithContext(gopath, "create test directory")
		require.NoError(t, err)
	}

	// Install different tools in different versions
	toolInstallations := map[string][]string{
		"1.21.0": {"gopls", "staticcheck"},
		"1.22.0": {"gopls"},
		"1.23.0": {"gopls", "staticcheck", "gofmt"},
	}

	for version, tools := range toolInstallations {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		for _, tool := range tools {
			cmdtest.CreateToolExecutable(t, binPath, tool)
		}
	}

	// Test 1: Verify ListInstalledVersions
	mgr := manager.NewManager(cfg, nil)
	installedVersions, err := mgr.ListInstalledVersions()
	require.NoError(t, err)
	assert.Len(t, installedVersions, 3)
	assert.True(t, utils.SlicesEqual(versions, installedVersions), "slices not equal: expected")

	// Test 2: Verify tools for each version
	for version, expectedTools := range toolInstallations {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		require.NoError(t, err)

		// Extract tool names
		var toolNames []string
		for _, tool := range toolList {
			toolNames = append(toolNames, tool.Name)
		}

		assert.True(t, utils.SlicesEqual(expectedTools, toolNames), "Tools for version don't match: expected")
	}

	// Test 3: Collect all unique tools
	allTools := make(map[string]bool)
	for _, version := range versions {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		require.NoError(t, err)
		for _, tool := range toolList {
			allTools[tool.Name] = true
		}
	}
	assert.Len(t, allTools, 3, "Should have 3 unique tools: expected length")
	assert.True(t, allTools["gopls"], "expected gopls in all tools")
	assert.True(t, allTools["staticcheck"], "expected staticcheck in all tools")
	assert.True(t, allTools["gofmt"], "expected gofmt in all tools")
}
