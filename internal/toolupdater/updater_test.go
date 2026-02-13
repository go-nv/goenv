package toolupdater

import (
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/config"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewUpdater(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	updater := NewUpdater(cfg)

	require.NotNil(t, updater, "NewUpdater should not return nil")
	assert.Equal(t, cfg, updater.cfg, "Updater config should be set correctly")
	assert.NotNil(t, updater.cache, "Updater cache should be initialized")
}

func TestUpdateStrategyConstants(t *testing.T) {
	strategies := []UpdateStrategy{
		StrategyLatest,
		StrategyMinor,
		StrategyPatch,
		StrategyPin,
		StrategyAuto,
	}

	// Verify all strategies are distinct
	seen := make(map[UpdateStrategy]bool)
	for _, s := range strategies {
		assert.False(t, seen[s], "Duplicate strategy: %s", s)
		seen[s] = true
	}

	// Verify expected values
	assert.Equal(t, "latest", string(StrategyLatest))
	assert.Equal(t, "minor", string(StrategyMinor))
	assert.Equal(t, "patch", string(StrategyPatch))
	assert.Equal(t, "pin", string(StrategyPin))
	assert.Equal(t, "auto", string(StrategyAuto))
}

func TestShouldUpdate(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	tests := []struct {
		name           string
		currentVersion string
		latestVersion  string
		strategy       UpdateStrategy
		want           bool
	}{
		{
			name:           "pin strategy never updates",
			currentVersion: "v1.0.0",
			latestVersion:  "v2.0.0",
			strategy:       StrategyPin,
			want:           false,
		},
		{
			name:           "latest strategy always updates",
			currentVersion: "v1.0.0",
			latestVersion:  "v1.0.1",
			strategy:       StrategyLatest,
			want:           true,
		},
		{
			name:           "auto strategy allows stable",
			currentVersion: "v1.0.0",
			latestVersion:  "v1.1.0",
			strategy:       StrategyAuto,
			want:           true,
		},
		{
			name:           "auto strategy blocks pre-release",
			currentVersion: "v1.0.0",
			latestVersion:  "v2.0.0-rc1",
			strategy:       StrategyAuto,
			want:           false,
		},
		{
			name:           "auto strategy blocks 0.x versions",
			currentVersion: "v1.0.0",
			latestVersion:  "v0.9.0",
			strategy:       StrategyAuto,
			want:           false,
		},
		// StrategyMinor tests
		{
			name:           "minor strategy allows same major version update",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.3.0",
			strategy:       StrategyMinor,
			want:           true,
		},
		{
			name:           "minor strategy blocks major version update",
			currentVersion: "v1.2.3",
			latestVersion:  "v2.0.0",
			strategy:       StrategyMinor,
			want:           false,
		},
		{
			name:           "minor strategy allows patch update",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.2.4",
			strategy:       StrategyMinor,
			want:           true,
		},
		// StrategyPatch tests
		{
			name:           "patch strategy allows patch update",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.2.4",
			strategy:       StrategyPatch,
			want:           true,
		},
		{
			name:           "patch strategy blocks minor update",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.3.0",
			strategy:       StrategyPatch,
			want:           false,
		},
		{
			name:           "patch strategy blocks major update",
			currentVersion: "v1.2.3",
			latestVersion:  "v2.0.0",
			strategy:       StrategyPatch,
			want:           false,
		},
		// Edge cases
		{
			name:           "minor strategy handles pre-release current version",
			currentVersion: "v1.2.3-rc1",
			latestVersion:  "v1.3.0",
			strategy:       StrategyMinor,
			want:           true,
		},
		{
			name:           "patch strategy handles pre-release latest version",
			currentVersion: "v1.2.3",
			latestVersion:  "v1.2.4-beta1",
			strategy:       StrategyPatch,
			want:           true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := updater.shouldUpdate(tt.currentVersion, tt.latestVersion, tt.strategy)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestIsPreRelease(t *testing.T) {
	tests := []struct {
		version string
		want    bool
	}{
		{"v1.2.3", false},
		{"v1.0.0-rc1", true},
		{"v1.0.0-beta2", true},
		{"v1.0.0-alpha1", true},
		{"v1.0.0-pre", true},
		{"v0.9.0", true}, // 0.x versions considered pre-release
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			got := isPreRelease(tt.version)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestContainsHelper(t *testing.T) {
	slice := []string{"apple", "banana", "cherry"}

	tests := []struct {
		item string
		want bool
	}{
		{"apple", true},
		{"banana", true},
		{"cherry", true},
		{"date", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.item, func(t *testing.T) {
			got := utils.SliceContains(slice, tt.item)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestUpdateOptionsDefaults(t *testing.T) {
	opts := UpdateOptions{
		Strategy:  StrategyAuto,
		GoVersion: "1.21.5",
	}

	assert.Equal(t, StrategyAuto, opts.Strategy)
	assert.False(t, opts.DryRun, "Expected DryRun to be false by default")
	assert.False(t, opts.Force, "Expected Force to be false by default")
	assert.False(t, opts.Verbose, "Expected Verbose to be false by default")
	assert.False(t, opts.CheckOnly, "Expected CheckOnly to be false by default")
}

func TestUpdateResultStructure(t *testing.T) {
	result := &UpdateResult{
		Checked: []UpdateCheck{
			{
				ToolName:        "gopls",
				CurrentVersion:  "v0.14.1",
				LatestVersion:   "v0.14.2",
				UpdateAvailable: true,
				Compatible:      true,
				Strategy:        StrategyLatest,
			},
		},
		Updated: []string{"gopls"},
		Failed:  []string{},
		Skipped: []string{},
		Errors:  []error{},
	}

	assert.Len(t, result.Checked, 1)
	assert.Len(t, result.Updated, 1)

	check := result.Checked[0]
	assert.Equal(t, "gopls", check.ToolName)
	assert.True(t, check.UpdateAvailable)
	assert.True(t, check.Compatible)
}

func TestUpdateCheckStructure(t *testing.T) {
	check := UpdateCheck{
		ToolName:        "gopls",
		CurrentVersion:  "v0.14.1",
		LatestVersion:   "v0.14.2",
		UpdateAvailable: true,
		Compatible:      true,
		Strategy:        StrategyLatest,
		Reason:          "",
	}

	assert.NotEmpty(t, check.ToolName, "ToolName should not be empty")
	assert.NotEmpty(t, check.CurrentVersion, "CurrentVersion should not be empty")
	assert.NotEmpty(t, check.LatestVersion, "LatestVersion should not be empty")
	assert.True(t, check.UpdateAvailable, "UpdateAvailable should be true")
	assert.True(t, check.Compatible, "Compatible should be true")
}

func TestUpdaterCacheIntegration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	// Get cache from updater
	cache := updater.GetUpdateCache()
	require.NotNil(t, cache, "GetUpdateCache should not return nil")

	// Set a value via cache
	packagePath := "golang.org/x/tools/gopls"
	version := "v0.14.2"

	err := cache.SetLatestVersion(packagePath, version)
	require.NoError(t, err, "Failed to set cache value")

	// Should be able to retrieve it
	got, found := cache.GetLatestVersion(packagePath)
	require.True(t, found, "Expected to find cached value")
	assert.Equal(t, version, got)
}

func TestUpdaterClearCache(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	// Add some data to cache
	cache := updater.GetUpdateCache()
	cache.SetLatestVersion("pkg1", "v1.0.0")
	cache.SetLatestVersion("pkg2", "v2.0.0")

	// Clear cache via updater
	err := updater.ClearCache()
	require.NoError(t, err, "ClearCache should not fail")

	// Cache should be empty
	_, found1 := cache.GetLatestVersion("pkg1")
	assert.False(t, found1, "Expected pkg1 to be cleared")

	_, found2 := cache.GetLatestVersion("pkg2")
	assert.False(t, found2, "Expected pkg2 to be cleared")
}

func TestUpdaterGetSetCacheTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	// Set a custom TTL
	customTTL := 2 * time.Hour
	updater.SetCacheTTL(customTTL)

	// Verify TTL was set
	cache := updater.GetUpdateCache()
	assert.Equal(t, customTTL, cache.GetTTL())
}

func TestCheckForUpdatesEmptyOptions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	// Call with empty options (no tools will match, but shouldn't error)
	opts := UpdateOptions{
		Strategy:  StrategyLatest,
		GoVersion: "nonexistent",
	}

	result, err := updater.CheckForUpdates(opts)

	// This will error because the version doesn't exist, which is expected
	if err != nil {
		// Expected behavior - non-existent version
		return
	}

	// If no error, verify result structure
	require.NotNil(t, result, "Expected non-nil result even with no tools")
	assert.NotNil(t, result.Checked, "Expected non-nil Checked slice")
	assert.NotNil(t, result.Updated, "Expected non-nil Updated slice")
	assert.NotNil(t, result.Failed, "Expected non-nil Failed slice")
	assert.NotNil(t, result.Skipped, "Expected non-nil Skipped slice")
	assert.NotNil(t, result.Errors, "Expected non-nil Errors slice")
}

func TestCheckToolUpdateWithoutPackagePath(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	tool := toolspkg.ToolMetadata{
		Name:        "unknown-tool",
		BinaryPath:  "/some/path/unknown-tool",
		PackagePath: "", // No package path
		Version:     "v1.0.0",
		GoVersion:   "1.21.5",
	}

	opts := UpdateOptions{
		Strategy: StrategyLatest,
	}

	check := updater.checkToolUpdate(tool, opts)

	assert.False(t, check.UpdateAvailable, "Expected no update available for tool without package path")
	assert.NotEmpty(t, check.Reason, "Expected reason to be set explaining why no update")
}

func TestUpdateAllMethod(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	// Call UpdateAll (will fail since no real Go version, but tests method signature)
	result, err := updater.UpdateAll("nonexistent", StrategyLatest, true, false)

	// Either error or empty result is acceptable for non-existent version
	if err != nil && result == nil {
		require.NotNil(t, result, "Expected non-nil result even on error")
	}

	// If we got a result, verify structure
	if result != nil {
		assert.NotNil(t, result.Checked, "Expected non-nil Checked slice")
		assert.NotNil(t, result.Updated, "Expected non-nil Updated slice")
	}
}

func TestUpdateSpecificMethod(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	// Call UpdateSpecific with tool names
	toolNames := []string{"gopls", "staticcheck"}
	result, err := updater.UpdateSpecific(toolNames, "nonexistent", StrategyLatest, true, false)

	// Either error or empty result is acceptable for non-existent version
	if err != nil && result == nil {
		require.NotNil(t, result, "Expected non-nil result even on error")
	}

	// If we got a result, verify structure
	if result != nil {
		assert.NotNil(t, result.Checked, "Expected non-nil Checked slice")
	}
}

func TestCheckUpdatesOnlyMethod(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	updater := NewUpdater(cfg)

	// Call CheckUpdatesOnly
	result, err := updater.CheckUpdatesOnly("nonexistent")

	// Either error or empty result is acceptable for non-existent version
	if err != nil && result == nil {
		require.NotNil(t, result, "Expected non-nil result even on error")
	}

	// If we got a result, verify structure
	if result != nil {
		assert.NotNil(t, result.Checked, "Expected non-nil Checked slice")
	}
}

func TestUpdateOptionsWithToolFilter(t *testing.T) {
	opts := UpdateOptions{
		Strategy:  StrategyLatest,
		GoVersion: "1.21.5",
		ToolNames: []string{"gopls", "staticcheck"},
		DryRun:    true,
	}

	assert.Len(t, opts.ToolNames, 2)
	assert.Equal(t, "gopls", opts.ToolNames[0])
	assert.Equal(t, "staticcheck", opts.ToolNames[1])
}
