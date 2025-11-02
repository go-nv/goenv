package toolupdater

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewCache(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	cache := NewCache(cfg)

	require.NotNil(t, cache, "NewCache returned nil")

	assert.Equal(t, cfg, cache.cfg, "Cache config not set correctly")

	assert.Equal(t, DefaultCacheTTL, cache.ttl, "Expected TTL")

	expectedPath := filepath.Join(tmpDir, "cache", "tool-updates")
	assert.Equal(t, expectedPath, cache.cachePath, "Expected cache path")
}

func TestNewCacheWithTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	customTTL := 1 * time.Hour

	cache := NewCacheWithTTL(cfg, customTTL)

	assert.Equal(t, customTTL, cache.ttl, "Expected TTL")
}

func TestCacheSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	packagePath := "golang.org/x/tools/gopls"
	version := "v0.14.2"

	// Set a value
	err := cache.SetLatestVersion(packagePath, version)
	require.NoError(t, err, "SetLatestVersion failed")

	// Get the value immediately (should be valid)
	got, found := cache.GetLatestVersion(packagePath)
	require.True(t, found, "Expected to find cached version")

	assert.Equal(t, version, got, "Expected version")
}

func TestCacheGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	_, found := cache.GetLatestVersion("nonexistent/package")
	assert.False(t, found, "Expected not to find non-existent package")
}

func TestCacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCacheWithTTL(cfg, 100*time.Millisecond)

	packagePath := "golang.org/x/tools/gopls"
	version := "v0.14.2"

	// Set a value
	err := cache.SetLatestVersion(packagePath, version)
	require.NoError(t, err, "SetLatestVersion failed")

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not find expired entry
	_, found := cache.GetLatestVersion(packagePath)
	assert.False(t, found, "Expected not to find expired cached version")
}

func TestCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	packagePath := "golang.org/x/tools/gopls"
	version := "v0.14.2"

	// Create cache and set value
	cache1 := NewCache(cfg)
	err := cache1.SetLatestVersion(packagePath, version)
	require.NoError(t, err, "SetLatestVersion failed")

	// Create new cache instance (simulates process restart)
	cache2 := NewCache(cfg)

	// Should find value from disk
	got, found := cache2.GetLatestVersion(packagePath)
	require.True(t, found, "Expected to find cached version from disk")

	assert.Equal(t, version, got, "Expected version")
}

func TestCacheClear(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	// Set multiple values
	packages := map[string]string{
		"golang.org/x/tools/gopls":                            "v0.14.2",
		"github.com/golangci/golangci-lint/cmd/golangci-lint": "v1.55.0",
		"honnef.co/go/tools/cmd/staticcheck":                  "v0.4.6",
	}

	for pkg, ver := range packages {
		err = cache.SetLatestVersion(pkg, ver)
		require.NoError(t, err, "SetLatestVersion failed")
	}

	// Clear cache
	err = cache.Clear()
	require.NoError(t, err, "Clear failed")

	// All entries should be gone
	for pkg := range packages {
		if _, found := cache.GetLatestVersion(pkg); found {
			t.Errorf("Expected package %s to be cleared", pkg)
		}
	}

	// Cache directory should be empty or not exist
	entries, err := os.ReadDir(cache.cachePath)
	if err != nil && !os.IsNotExist(err) {
		t.Fatalf("Failed to read cache directory: %v", err)
	}
	assert.Empty(t, entries, "Expected empty cache directory, found entries")
}

func TestCacheClearPackage(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	pkg1 := "golang.org/x/tools/gopls"
	pkg2 := "github.com/golangci/golangci-lint/cmd/golangci-lint"

	cache.SetLatestVersion(pkg1, "v0.14.2")
	cache.SetLatestVersion(pkg2, "v1.55.0")

	// Clear only pkg1
	err = cache.ClearPackage(pkg1)
	require.NoError(t, err, "ClearPackage failed")

	// pkg1 should be gone
	if _, found := cache.GetLatestVersion(pkg1); found {
		t.Error("Expected pkg1 to be cleared")
	}

	// pkg2 should still exist
	if _, found := cache.GetLatestVersion(pkg2); !found {
		t.Error("Expected pkg2 to still exist")
	}
}

func TestCacheGetAllEntries(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	packages := map[string]string{
		"golang.org/x/tools/gopls":                            "v0.14.2",
		"github.com/golangci/golangci-lint/cmd/golangci-lint": "v1.55.0",
	}

	for pkg, ver := range packages {
		cache.SetLatestVersion(pkg, ver)
	}

	entries := cache.GetAllEntries()

	assert.Len(t, entries, len(packages), "Expected entries")

	for pkg, expectedVer := range packages {
		entry, ok := entries[pkg]
		if !ok {
			t.Errorf("Missing entry for package %s", pkg)
			continue
		}

		assert.Equal(t, expectedVer, entry.LatestVersion, "Expected version for %v", pkg)

		assert.Equal(t, pkg, entry.PackagePath, "Expected package path")
	}
}

func TestCacheGetStats(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCacheWithTTL(cfg, 100*time.Millisecond)

	// Add some entries
	cache.SetLatestVersion("pkg1", "v1.0.0")
	time.Sleep(50 * time.Millisecond)
	cache.SetLatestVersion("pkg2", "v2.0.0")
	time.Sleep(60 * time.Millisecond) // pkg1 is now expired

	stats := cache.GetStats()

	assert.Equal(t, 2, stats.TotalEntries, "Expected 2 total entries")

	assert.Equal(t, 1, stats.ValidEntries, "Expected 1 valid entry")

	assert.Equal(t, 1, stats.ExpiredEntries, "Expected 1 expired entry")

	assert.Equal(t, 100*time.Millisecond, stats.TTL, "Expected TTL")
}

func TestCachePrune(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCacheWithTTL(cfg, 50*time.Millisecond)

	// Add entries
	cache.SetLatestVersion("pkg1", "v1.0.0")
	time.Sleep(60 * time.Millisecond)
	cache.SetLatestVersion("pkg2", "v2.0.0") // Fresh entry

	// Prune expired entries
	err = cache.Prune()
	require.NoError(t, err, "Prune failed")

	stats := cache.GetStats()
	assert.Equal(t, 1, stats.TotalEntries, "Expected 1 entry after prune")

	assert.Equal(t, 0, stats.ExpiredEntries, "Expected 0 expired entries after prune")
}

func TestCacheLoadAll(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Create cache and add entries
	cache1 := NewCache(cfg)
	cache1.SetLatestVersion("pkg1", "v1.0.0")
	cache1.SetLatestVersion("pkg2", "v2.0.0")

	// Create new cache instance and load all
	cache2 := NewCache(cfg)
	err = cache2.LoadAll()
	require.NoError(t, err, "LoadAll failed")

	// Check entries loaded
	if _, found := cache2.GetLatestVersion("pkg1"); !found {
		t.Error("Expected to find pkg1 after LoadAll")
	}

	if _, found := cache2.GetLatestVersion("pkg2"); !found {
		t.Error("Expected to find pkg2 after LoadAll")
	}
}

func TestCacheSetAndGetTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	initialTTL := cache.GetTTL()
	assert.Equal(t, DefaultCacheTTL, initialTTL, "Expected initial TTL")

	newTTL := 2 * time.Hour
	cache.SetTTL(newTTL)

	assert.Equal(t, newTTL, cache.GetTTL(), "Expected TTL")
}

func TestCacheGetLastCheckTime(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	// Initially should be zero
	lastCheck := cache.GetLastCheckTime()
	assert.True(t, lastCheck.IsZero(), "Expected zero last check time for empty cache")

	// Add an entry
	before := time.Now()
	cache.SetLatestVersion("pkg1", "v1.0.0")
	after := time.Now()

	lastCheck = cache.GetLastCheckTime()
	assert.False(t, lastCheck.Before(before) || lastCheck.After(after), "Last check time not in expected range [, ]")
}

func TestCacheConcurrency(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	// Test concurrent reads and writes
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			pkg := filepath.Join("test/pkg", string(rune('a'+id)))
			version := "v1.0.0"

			// Write
			cache.SetLatestVersion(pkg, version)

			// Read
			_, _ = cache.GetLatestVersion(pkg)

			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// No panic = success
}
