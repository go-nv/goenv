package version

import (
	"encoding/json"
	"fmt"
	"net/http"
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
	cachePath := filepath.Join(goenvRoot, "cache", "versions.json")
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

// EmbeddedVersions contains a fallback list of Go versions
// This is updated at build time or manually when needed
var EmbeddedVersions = []GoRelease{
	{
		Version: "go1.25.2",
		Stable:  true,
		Files: []GoFile{
			{Filename: "go1.25.2.darwin-amd64.tar.gz", OS: "darwin", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.25.2.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.25.2.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.25.2.linux-arm64.tar.gz", OS: "linux", Arch: "arm64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.25.2.freebsd-amd64.tar.gz", OS: "freebsd", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
		},
	},
	{
		Version: "go1.24.8",
		Stable:  true,
		Files: []GoFile{
			{Filename: "go1.24.8.darwin-amd64.tar.gz", OS: "darwin", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.24.8.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.24.8.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.24.8.linux-arm64.tar.gz", OS: "linux", Arch: "arm64", Kind: "archive", SHA256: "placeholder"},
			{Filename: "go1.24.8.freebsd-amd64.tar.gz", OS: "freebsd", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
		},
	},
	// Add more versions as needed...
}

// FetchWithFallback tries to fetch versions online, falls back to cache, then embedded data
func (f *Fetcher) FetchWithFallback(goenvRoot string) ([]GoRelease, error) {
	cache := NewCache(goenvRoot)

	// Try to fetch online first
	releases, err := f.FetchAvailableVersions()
	if err == nil {
		// Success! Cache the result for future offline use
		if cacheErr := cache.Set(releases); cacheErr != nil && f.debug {
			// Non-fatal error, just log if debug is enabled
			fmt.Printf("Debug: Failed to cache versions: %v\n", cacheErr)
		}
		return releases, nil
	}

	// Online fetch failed, try cache
	cached, cacheErr := cache.Get()
	if cacheErr == nil && !cached.IsStale(24*time.Hour) {
		if f.debug {
			fmt.Printf("Debug: Using cached versions (last updated: %s)\n", cached.LastUpdated.Format(time.RFC3339))
		}
		return cached.Releases, nil
	}

	// Cache is stale or missing, use embedded data
	if f.debug {
		fmt.Printf("Debug: Using embedded fallback versions\n")
	}

	return EmbeddedVersions, nil
}

// Add debug field to Fetcher
type FetcherOptions struct {
	Debug bool
}

// NewFetcherWithOptions creates a new version fetcher with options
func NewFetcherWithOptions(opts FetcherOptions) *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://go.dev/dl/",
		debug:   opts.Debug,
	}
}

// Update the Fetcher struct to include debug field
// (This would be added to the existing struct in the same file)
