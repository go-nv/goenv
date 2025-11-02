package toolupdater

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/config"
)

func TestNewCache(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	cache := NewCache(cfg)

	if cache == nil {
		t.Fatal("NewCache returned nil")
	}

	if cache.cfg != cfg {
		t.Error("Cache config not set correctly")
	}

	if cache.ttl != DefaultCacheTTL {
		t.Errorf("Expected TTL %v, got %v", DefaultCacheTTL, cache.ttl)
	}

	expectedPath := filepath.Join(tmpDir, "cache", "tool-updates")
	if cache.cachePath != expectedPath {
		t.Errorf("Expected cache path %s, got %s", expectedPath, cache.cachePath)
	}
}

func TestNewCacheWithTTL(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	customTTL := 1 * time.Hour

	cache := NewCacheWithTTL(cfg, customTTL)

	if cache.ttl != customTTL {
		t.Errorf("Expected TTL %v, got %v", customTTL, cache.ttl)
	}
}

func TestCacheSetAndGet(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	packagePath := "golang.org/x/tools/gopls"
	version := "v0.14.2"

	// Set a value
	err := cache.SetLatestVersion(packagePath, version)
	if err != nil {
		t.Fatalf("SetLatestVersion failed: %v", err)
	}

	// Get the value immediately (should be valid)
	got, found := cache.GetLatestVersion(packagePath)
	if !found {
		t.Fatal("Expected to find cached version")
	}

	if got != version {
		t.Errorf("Expected version %s, got %s", version, got)
	}
}

func TestCacheGetNonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	_, found := cache.GetLatestVersion("nonexistent/package")
	if found {
		t.Error("Expected not to find non-existent package")
	}
}

func TestCacheExpiration(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCacheWithTTL(cfg, 100*time.Millisecond)

	packagePath := "golang.org/x/tools/gopls"
	version := "v0.14.2"

	// Set a value
	err := cache.SetLatestVersion(packagePath, version)
	if err != nil {
		t.Fatalf("SetLatestVersion failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should not find expired entry
	_, found := cache.GetLatestVersion(packagePath)
	if found {
		t.Error("Expected not to find expired cached version")
	}
}

func TestCachePersistence(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	packagePath := "golang.org/x/tools/gopls"
	version := "v0.14.2"

	// Create cache and set value
	cache1 := NewCache(cfg)
	err := cache1.SetLatestVersion(packagePath, version)
	if err != nil {
		t.Fatalf("SetLatestVersion failed: %v", err)
	}

	// Create new cache instance (simulates process restart)
	cache2 := NewCache(cfg)

	// Should find value from disk
	got, found := cache2.GetLatestVersion(packagePath)
	if !found {
		t.Fatal("Expected to find cached version from disk")
	}

	if got != version {
		t.Errorf("Expected version %s, got %s", version, got)
	}
}

func TestCacheClear(t *testing.T) {
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
		if err := cache.SetLatestVersion(pkg, ver); err != nil {
			t.Fatalf("SetLatestVersion failed: %v", err)
		}
	}

	// Clear cache
	if err := cache.Clear(); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

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
	if len(entries) > 0 {
		t.Errorf("Expected empty cache directory, found %d entries", len(entries))
	}
}

func TestCacheClearPackage(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	pkg1 := "golang.org/x/tools/gopls"
	pkg2 := "github.com/golangci/golangci-lint/cmd/golangci-lint"

	cache.SetLatestVersion(pkg1, "v0.14.2")
	cache.SetLatestVersion(pkg2, "v1.55.0")

	// Clear only pkg1
	if err := cache.ClearPackage(pkg1); err != nil {
		t.Fatalf("ClearPackage failed: %v", err)
	}

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

	if len(entries) != len(packages) {
		t.Errorf("Expected %d entries, got %d", len(packages), len(entries))
	}

	for pkg, expectedVer := range packages {
		entry, ok := entries[pkg]
		if !ok {
			t.Errorf("Missing entry for package %s", pkg)
			continue
		}

		if entry.LatestVersion != expectedVer {
			t.Errorf("Expected version %s for %s, got %s", expectedVer, pkg, entry.LatestVersion)
		}

		if entry.PackagePath != pkg {
			t.Errorf("Expected package path %s, got %s", pkg, entry.PackagePath)
		}
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

	if stats.TotalEntries != 2 {
		t.Errorf("Expected 2 total entries, got %d", stats.TotalEntries)
	}

	if stats.ValidEntries != 1 {
		t.Errorf("Expected 1 valid entry, got %d", stats.ValidEntries)
	}

	if stats.ExpiredEntries != 1 {
		t.Errorf("Expected 1 expired entry, got %d", stats.ExpiredEntries)
	}

	if stats.TTL != 100*time.Millisecond {
		t.Errorf("Expected TTL %v, got %v", 100*time.Millisecond, stats.TTL)
	}
}

func TestCachePrune(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCacheWithTTL(cfg, 50*time.Millisecond)

	// Add entries
	cache.SetLatestVersion("pkg1", "v1.0.0")
	time.Sleep(60 * time.Millisecond)
	cache.SetLatestVersion("pkg2", "v2.0.0") // Fresh entry

	// Prune expired entries
	if err := cache.Prune(); err != nil {
		t.Fatalf("Prune failed: %v", err)
	}

	stats := cache.GetStats()
	if stats.TotalEntries != 1 {
		t.Errorf("Expected 1 entry after prune, got %d", stats.TotalEntries)
	}

	if stats.ExpiredEntries != 0 {
		t.Errorf("Expected 0 expired entries after prune, got %d", stats.ExpiredEntries)
	}
}

func TestCacheLoadAll(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Create cache and add entries
	cache1 := NewCache(cfg)
	cache1.SetLatestVersion("pkg1", "v1.0.0")
	cache1.SetLatestVersion("pkg2", "v2.0.0")

	// Create new cache instance and load all
	cache2 := NewCache(cfg)
	if err := cache2.LoadAll(); err != nil {
		t.Fatalf("LoadAll failed: %v", err)
	}

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
	if initialTTL != DefaultCacheTTL {
		t.Errorf("Expected initial TTL %v, got %v", DefaultCacheTTL, initialTTL)
	}

	newTTL := 2 * time.Hour
	cache.SetTTL(newTTL)

	if cache.GetTTL() != newTTL {
		t.Errorf("Expected TTL %v, got %v", newTTL, cache.GetTTL())
	}
}

func TestCacheGetLastCheckTime(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	cache := NewCache(cfg)

	// Initially should be zero
	lastCheck := cache.GetLastCheckTime()
	if !lastCheck.IsZero() {
		t.Error("Expected zero last check time for empty cache")
	}

	// Add an entry
	before := time.Now()
	cache.SetLatestVersion("pkg1", "v1.0.0")
	after := time.Now()

	lastCheck = cache.GetLastCheckTime()
	if lastCheck.Before(before) || lastCheck.After(after) {
		t.Errorf("Last check time %v not in expected range [%v, %v]", lastCheck, before, after)
	}
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
