package toolupdater

import (
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// DefaultCacheTTL is the default time-to-live for cached update checks (24 hours)
const DefaultCacheTTL = 24 * time.Hour

// CacheEntry represents a cached version check result
type CacheEntry struct {
	PackagePath   string    `json:"package_path"`
	LatestVersion string    `json:"latest_version"`
	CheckedAt     time.Time `json:"checked_at"`
	ExpiresAt     time.Time `json:"expires_at"`
}

// Cache manages caching of tool version checks
type Cache struct {
	cfg       *config.Config
	ttl       time.Duration
	mu        sync.RWMutex
	entries   map[string]*CacheEntry // In-memory cache
	cachePath string
}

// NewCache creates a new update cache
func NewCache(cfg *config.Config) *Cache {
	cachePath := filepath.Join(cfg.Root, "cache", "tool-updates")
	return &Cache{
		cfg:       cfg,
		ttl:       DefaultCacheTTL,
		entries:   make(map[string]*CacheEntry),
		cachePath: cachePath,
	}
}

// NewCacheWithTTL creates a cache with a custom TTL
func NewCacheWithTTL(cfg *config.Config, ttl time.Duration) *Cache {
	cache := NewCache(cfg)
	cache.ttl = ttl
	return cache
}

// GetLatestVersion retrieves the latest version for a package from cache
// Returns the version and a boolean indicating if it was found and still valid
func (c *Cache) GetLatestVersion(packagePath string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Check in-memory cache first
	if entry, ok := c.entries[packagePath]; ok {
		if time.Now().Before(entry.ExpiresAt) {
			return entry.LatestVersion, true
		}
		// Expired, remove it
		delete(c.entries, packagePath)
	}

	// Check disk cache
	entry, err := c.loadFromDisk(packagePath)
	if err != nil {
		return "", false
	}

	if time.Now().Before(entry.ExpiresAt) {
		// Valid cache entry, store in memory
		c.entries[packagePath] = entry
		return entry.LatestVersion, true
	}

	return "", false
}

// SetLatestVersion stores the latest version for a package in cache
func (c *Cache) SetLatestVersion(packagePath, version string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	entry := &CacheEntry{
		PackagePath:   packagePath,
		LatestVersion: version,
		CheckedAt:     now,
		ExpiresAt:     now.Add(c.ttl),
	}

	// Store in memory
	c.entries[packagePath] = entry

	// Store on disk
	return c.saveToDisk(entry)
}

// Clear removes all cached entries
func (c *Cache) Clear() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Clear in-memory cache
	c.entries = make(map[string]*CacheEntry)

	// Clear disk cache
	if err := os.RemoveAll(c.cachePath); err != nil && !os.IsNotExist(err) {
		return errors.FailedTo("clear cache", err)
	}

	return nil
}

// ClearPackage removes cached entry for a specific package
func (c *Cache) ClearPackage(packagePath string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Remove from memory
	delete(c.entries, packagePath)

	// Remove from disk
	cacheFile := c.getCacheFilePath(packagePath)
	if err := os.Remove(cacheFile); err != nil && !os.IsNotExist(err) {
		return errors.FailedTo("remove cache entry", err)
	}

	return nil
}

// GetAllEntries returns all cached entries (for inspection/debugging)
func (c *Cache) GetAllEntries() map[string]*CacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	// Return a copy to avoid concurrent modification
	result := make(map[string]*CacheEntry, len(c.entries))
	for k, v := range c.entries {
		result[k] = v
	}

	return result
}

// GetStats returns cache statistics
func (c *Cache) GetStats() CacheStats {
	c.mu.RLock()
	defer c.mu.RUnlock()

	stats := CacheStats{
		TotalEntries:   len(c.entries),
		ValidEntries:   0,
		ExpiredEntries: 0,
		TTL:            c.ttl,
	}

	now := time.Now()
	var oldestCheck, newestCheck time.Time

	for _, entry := range c.entries {
		if now.Before(entry.ExpiresAt) {
			stats.ValidEntries++
		} else {
			stats.ExpiredEntries++
		}

		if oldestCheck.IsZero() || entry.CheckedAt.Before(oldestCheck) {
			oldestCheck = entry.CheckedAt
		}
		if newestCheck.IsZero() || entry.CheckedAt.After(newestCheck) {
			newestCheck = entry.CheckedAt
		}
	}

	stats.OldestCheck = oldestCheck
	stats.NewestCheck = newestCheck

	return stats
}

// CacheStats contains statistics about the cache
type CacheStats struct {
	TotalEntries   int
	ValidEntries   int
	ExpiredEntries int
	OldestCheck    time.Time
	NewestCheck    time.Time
	TTL            time.Duration
}

// SetTTL updates the cache TTL
func (c *Cache) SetTTL(ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.ttl = ttl
}

// GetTTL returns the current cache TTL
func (c *Cache) GetTTL() time.Duration {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.ttl
}

// GetLastCheckTime returns when the last update check was performed
func (c *Cache) GetLastCheckTime() time.Time {
	stats := c.GetStats()
	return stats.NewestCheck
}

// Prune removes expired entries from cache
func (c *Cache) Prune() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	removedCount := 0

	// Remove expired entries from memory
	for packagePath, entry := range c.entries {
		if now.After(entry.ExpiresAt) {
			delete(c.entries, packagePath)
			removedCount++

			// Also remove from disk
			cacheFile := c.getCacheFilePath(packagePath)
			_ = os.Remove(cacheFile) // Ignore errors
		}
	}

	return nil
}

// LoadAll loads all cache entries from disk into memory
func (c *Cache) LoadAll() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Ensure cache directory exists
	if err := utils.EnsureDirWithContext(c.cachePath, "create cache directory"); err != nil {
		return err
	}

	// Read all cache files
	entries, err := os.ReadDir(c.cachePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.FailedTo("read cache directory", err)
	}

	now := time.Now()
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		cacheFile := filepath.Join(c.cachePath, entry.Name())
		var cacheEntry CacheEntry
		if err := utils.UnmarshalJSONFile(cacheFile, &cacheEntry); err != nil {
			continue // Skip files we can't read or malformed entries
		}

		// Only load valid entries
		if now.Before(cacheEntry.ExpiresAt) {
			c.entries[cacheEntry.PackagePath] = &cacheEntry
		} else {
			// Remove expired entries from disk
			_ = os.Remove(cacheFile)
		}
	}

	return nil
}

// Private helper methods

func (c *Cache) getCacheFilePath(packagePath string) string {
	// Use hash of package path as filename to avoid filesystem issues
	fullHash := utils.SHA256String(packagePath)
	filename := fullHash[:32] + ".json" // Use first 32 hex chars (16 bytes)
	return filepath.Join(c.cachePath, filename)
}

func (c *Cache) loadFromDisk(packagePath string) (*CacheEntry, error) {
	cacheFile := c.getCacheFilePath(packagePath)

	var entry CacheEntry
	if err := utils.UnmarshalJSONFile(cacheFile, &entry); err != nil {
		return nil, errors.FailedTo("parse cache entry", err)
	}

	return &entry, nil
}

func (c *Cache) saveToDisk(entry *CacheEntry) error {
	// Ensure cache directory exists
	if err := utils.EnsureDirWithContext(c.cachePath, "create cache directory"); err != nil {
		return err
	}

	cacheFile := c.getCacheFilePath(entry.PackagePath)

	if err := utils.MarshalJSONFileCompact(cacheFile, entry); err != nil {
		return errors.FailedTo("write cache file", err)
	}

	return nil
}
