package version

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"time"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// Cache manages local caching of version information
type Cache struct {
	cachePath string
}

// CachedData represents cached version information with integrity and ETag support
type CachedData struct {
	LastUpdated time.Time   `json:"last_updated"`
	Releases    []GoRelease `json:"releases"`
	ETag        string      `json:"etag,omitempty"`     // HTTP ETag for conditional requests
	SHA256      string      `json:"sha256,omitempty"`   // SHA256 hash of the releases data
	Checksum    string      `json:"checksum,omitempty"` // Legacy field, kept for compatibility
}

// NewCache creates a new cache instance
func NewCache(goenvRoot string) *Cache {
	// Use consistent cache location with fetcher.go
	cachePath := filepath.Join(goenvRoot, "releases-cache.json")
	return &Cache{cachePath: cachePath}
}

// Get retrieves cached version data with integrity verification
func (c *Cache) Get() (*CachedData, error) {
	// Check file permissions first
	if err := c.checkPermissions(); err != nil {
		// Log warning but don't fail - try to fix permissions
		if fixErr := c.ensureSecurePermissions(); fixErr != nil {
			return nil, fmt.Errorf("cache file has insecure permissions and cannot be fixed: %w", err)
		}
	}

	var cached CachedData
	if err := utils.UnmarshalJSONFile(c.cachePath, &cached); err != nil {
		return nil, err
	}

	// Verify SHA256 integrity if present
	if cached.SHA256 != "" {
		if err := c.verifySHA256(&cached); err != nil {
			return nil, fmt.Errorf("cache integrity check failed: %w", err)
		}
	}

	return &cached, nil
}

// verifySHA256 verifies the SHA256 hash of the releases data
func (c *Cache) verifySHA256(cached *CachedData) error {
	// Compute SHA256 of releases data
	releasesJSON, err := json.Marshal(cached.Releases)
	if err != nil {
		return errors.FailedTo("marshal releases for verification", err)
	}

	computed := utils.SHA256Bytes(releasesJSON)

	if computed != cached.SHA256 {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", cached.SHA256, computed)
	}

	return nil
}

// checkPermissions and ensureSecurePermissions are implemented in:
// - cache_unix.go for Unix/Linux/macOS (strict permission checks)
// - cache_windows.go for Windows (no-op due to ACL-based security)

// Set stores version data in cache with SHA256 integrity and secure permissions
func (c *Cache) Set(releases []GoRelease) error {
	return c.SetWithETag(releases, "")
}

// SetWithETag stores version data in cache with ETag support
func (c *Cache) SetWithETag(releases []GoRelease, etag string) error {
	// Ensure cache directory exists with secure permissions (utils.PermDirSecure)
	cacheDir := filepath.Dir(c.cachePath)
	if err := utils.EnsureDirWithContext(cacheDir, "create cache directory"); err != nil {
		return err
	}

	// Compute SHA256 hash of releases data for integrity
	releasesJSON, err := json.Marshal(releases)
	if err != nil {
		return errors.FailedTo("marshal releases for hashing", err)
	}

	sha256Hash := utils.SHA256Bytes(releasesJSON)

	cached := CachedData{
		LastUpdated: time.Now(),
		Releases:    releases,
		ETag:        etag,
		SHA256:      sha256Hash,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return errors.FailedTo("marshal cache data", err)
	}

	// Write with secure permissions (utils.PermFileSecure)
	if err := utils.WriteFileWithContext(c.cachePath, data, utils.PermFileSecure, "write cache file"); err != nil {
		return err
	}

	// Verify permissions were set correctly
	if err := c.checkPermissions(); err != nil {
		// Try to fix if they weren't set correctly
		return c.ensureSecurePermissions()
	}

	return nil
}

// IsStale checks if cached data is older than the specified duration
func (c *CachedData) IsStale(maxAge time.Duration) bool {
	return time.Since(c.LastUpdated) > maxAge
}

// FetchWithFallback tries to fetch all versions online with ETag support, falls back to cache, then embedded data
func (f *Fetcher) FetchWithFallback(goenvRoot string) ([]GoRelease, error) {
	cache := NewCache(goenvRoot)

	// Check if offline mode is enabled
	if utils.GoenvEnvVarOffline.IsTrue() {
		if f.debug {
			fmt.Println("Debug: GOENV_OFFLINE=1, skipping online fetch and using embedded versions")
		}
		// Skip cache entirely in offline mode and go straight to embedded
		return EmbeddedVersions, nil
	}

	// Try to get cached data first for ETag support
	cached, cacheErr := cache.Get()
	var etag string
	if cacheErr == nil {
		etag = cached.ETag
	}

	// Try to fetch ALL versions online with ETag for conditional request
	releases, newETag, err := f.FetchAllReleasesWithETag(etag)
	if err == nil {
		// Check if content was modified (newETag will be empty if 304 Not Modified)
		if newETag == "" && etag != "" {
			// Server returned 304 Not Modified - use cached data
			if f.debug {
				fmt.Println("Debug: Server returned 304 Not Modified, using cached data")
			}
			if cacheErr == nil {
				// Optionally trigger background refresh if enabled
				if utils.GoenvEnvVarCacheBgRefresh.IsTrue() {
					go f.backgroundRefresh(cache, goenvRoot)
				}
				return cached.Releases, nil
			}
		} else {
			// Success! Cache the result with ETag for future use
			if cacheErr := cache.SetWithETag(releases, newETag); cacheErr != nil && f.debug {
				// Non-fatal error, just log if debug is enabled
				fmt.Printf("Debug: Failed to cache versions: %v\n", cacheErr)
			}
			return releases, nil
		}
	}

	// Online fetch failed, try cache
	if f.debug {
		fmt.Printf("Debug: Online fetch failed (%v), trying cache...\n", err)
	}

	if cacheErr == nil && !cached.IsStale(24*time.Hour) {
		if f.debug {
			fmt.Printf("Debug: Using cached versions (last updated: %s)\n", cached.LastUpdated.Format(time.RFC3339))
		}
		return cached.Releases, nil
	}

	// Cache is also stale or missing, but try it anyway if available (better than nothing)
	if cacheErr == nil {
		if f.debug {
			fmt.Printf("Debug: Using stale cache (last updated: %s)\n", cached.LastUpdated.Format(time.RFC3339))
		}
		return cached.Releases, nil
	}

	// No cache available, use embedded data as last resort
	if f.debug {
		fmt.Printf("Debug: Using embedded fallback versions\n")
	}

	// EmbeddedVersions is defined in embedded_versions.go (generated at build time)
	// To regenerate: go run scripts/generate_embedded_versions.go
	return EmbeddedVersions, nil
}

// backgroundRefresh performs a background refresh of the cache
func (f *Fetcher) backgroundRefresh(cache *Cache, goenvRoot string) {
	if f.debug {
		fmt.Println("Debug: Starting background cache refresh...")
	}

	// Get current cached data for ETag
	cached, err := cache.Get()
	if err != nil {
		return // Can't refresh without current cache
	}

	// Try to fetch with ETag
	releases, newETag, err := f.FetchAllReleasesWithETag(cached.ETag)
	if err != nil {
		if f.debug {
			fmt.Printf("Debug: Background refresh failed: %v\n", err)
		}
		return
	}

	// Only update if content changed
	if newETag != "" && newETag != cached.ETag {
		if err := cache.SetWithETag(releases, newETag); err != nil && f.debug {
			fmt.Printf("Debug: Background cache update failed: %v\n", err)
		} else if f.debug {
			fmt.Println("Debug: Background cache refresh completed")
		}
	}
}
