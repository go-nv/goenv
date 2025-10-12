package version

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// Cache manages local caching of version information
type Cache struct {
	cachePath string
}

// CachedData represents cached version information
type CachedData struct {
	LastUpdated time.Time   `json:"last_updated"`
	Releases    []GoRelease `json:"releases"`
}

// NewCache creates a new cache instance
func NewCache(goenvRoot string) *Cache {
	// Use consistent cache location with fetcher.go
	cachePath := filepath.Join(goenvRoot, "releases-cache.json")
	return &Cache{cachePath: cachePath}
}

// Get retrieves cached version data
func (c *Cache) Get() (*CachedData, error) {
	data, err := os.ReadFile(c.cachePath)
	if err != nil {
		return nil, err
	}

	var cached CachedData
	if err := json.Unmarshal(data, &cached); err != nil {
		return nil, err
	}

	return &cached, nil
}

// Set stores version data in cache
func (c *Cache) Set(releases []GoRelease) error {
	// Ensure cache directory exists
	if err := os.MkdirAll(filepath.Dir(c.cachePath), 0755); err != nil {
		return err
	}

	cached := CachedData{
		LastUpdated: time.Now(),
		Releases:    releases,
	}

	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(c.cachePath, data, 0644)
}

// IsStale checks if cached data is older than the specified duration
func (c *CachedData) IsStale(maxAge time.Duration) bool {
	return time.Since(c.LastUpdated) > maxAge
}

// FetchWithFallback tries to fetch all versions online, falls back to cache, then embedded data
func (f *Fetcher) FetchWithFallback(goenvRoot string) ([]GoRelease, error) {
	cache := NewCache(goenvRoot)

	// Check if offline mode is enabled
	if os.Getenv("GOENV_OFFLINE") == "1" {
		if f.debug {
			fmt.Println("Debug: GOENV_OFFLINE=1, skipping online fetch and using embedded versions")
		}
		// Skip cache entirely in offline mode and go straight to embedded
		return EmbeddedVersions, nil
	}

	// Try to fetch ALL versions online first (using FetchAllReleases)
	releases, err := f.FetchAllReleases()
	if err == nil {
		// Success! Cache the result for future offline use
		if cacheErr := cache.Set(releases); cacheErr != nil && f.debug {
			// Non-fatal error, just log if debug is enabled
			fmt.Printf("Debug: Failed to cache versions: %v\n", cacheErr)
		}
		return releases, nil
	}

	// Online fetch failed, try cache
	if f.debug {
		fmt.Printf("Debug: Online fetch failed (%v), trying cache...\n", err)
	}

	cached, cacheErr := cache.Get()
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
