package lifecycle

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"time"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// EOLData represents lifecycle information from endoflife.date API
type EOLData struct {
	Cycle             string      `json:"cycle"`
	ReleaseDate       string      `json:"releaseDate"`
	EOL               interface{} `json:"eol"` // Can be string date or bool false
	Latest            string      `json:"latest"`
	LatestReleaseDate string      `json:"latestReleaseDate"`
	LTS               bool        `json:"lts"`
}

// LifecycleCache represents cached lifecycle information
type LifecycleCache struct {
	Data      map[string]VersionInfo `json:"data"`
	UpdatedAt time.Time              `json:"updated_at"`
	ETag      string                 `json:"etag,omitempty"`
	SHA256    string                 `json:"sha256,omitempty"`
}

// Fetcher handles fetching Go lifecycle information
type Fetcher struct {
	client   *http.Client
	baseURL  string
	debug    bool
	cacheDir string
}

// NewFetcher creates a new lifecycle fetcher
func NewFetcher() *Fetcher {
	return &Fetcher{
		client:  utils.NewHTTPClientDefault(),
		baseURL: "https://endoflife.date/api/go.json",
		debug:   false,
	}
}

// NewFetcherWithCache creates a new lifecycle fetcher with a cache directory
func NewFetcherWithCache(cacheDir string) *Fetcher {
	f := NewFetcher()
	f.cacheDir = cacheDir
	return f
}

// SetDebug sets the debug flag
func (f *Fetcher) SetDebug(debug bool) {
	f.debug = debug
}

// FetchLifecycleData fetches lifecycle data from endoflife.date API
func (f *Fetcher) FetchLifecycleData() (map[string]VersionInfo, error) {
	data, _, err := f.FetchLifecycleDataWithETag("")
	return data, err
}

// FetchLifecycleDataWithETag fetches lifecycle data with ETag support for conditional requests
func (f *Fetcher) FetchLifecycleDataWithETag(etag string) (map[string]VersionInfo, string, error) {
	if f.debug {
		fmt.Println("Debug: Fetching lifecycle data from endoflife.date API...")
		if etag != "" {
			fmt.Printf("Debug: Using ETag for conditional request: %s\n", etag)
		}
	}

	// Create request with ETag support
	req, err := http.NewRequest("GET", f.baseURL, nil)
	if err != nil {
		return nil, "", errors.FailedTo("create request", err)
	}

	// Add If-None-Match header if we have an ETag
	if etag != "" {
		req.Header.Set("If-None-Match", etag)
	}

	resp, err := f.client.Do(req)
	if err != nil {
		return nil, "", errors.FailedTo("fetch lifecycle data", err)
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		if f.debug {
			fmt.Println("Debug: Server returned 304 Not Modified")
		}
		return nil, "", nil // Signal that cache is still valid
	}

	if resp.StatusCode != http.StatusOK {
		return nil, "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	// Get new ETag from response
	newETag := resp.Header.Get("ETag")

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, "", errors.FailedTo("read response body", err)
	}

	var eolData []EOLData
	if err := json.Unmarshal(body, &eolData); err != nil {
		return nil, "", errors.FailedTo("parse JSON response", err)
	}

	// Convert to VersionInfo map
	versionInfo := make(map[string]VersionInfo)
	for _, item := range eolData {
		info := convertEOLDataToVersionInfo(item)
		if info.Version != "" {
			versionInfo[info.Version] = info
		}
	}

	return versionInfo, newETag, nil
}

// convertEOLDataToVersionInfo converts EOLData from API to VersionInfo
func convertEOLDataToVersionInfo(data EOLData) VersionInfo {
	info := VersionInfo{
		Version: data.Cycle,
	}

	// Parse release date
	if data.ReleaseDate != "" {
		info.ReleaseDate = parseDate(data.ReleaseDate)
	}

	// Parse EOL date
	switch v := data.EOL.(type) {
	case string:
		info.EOLDate = parseDate(v)
	case bool:
		if !v {
			// EOL is false, meaning still supported - use far future date
			info.EOLDate = parseDate("2099-12-31")
		}
	}

	// Status, Recommended, and SecurityOnly are calculated dynamically at runtime
	info.Status = StatusUnknown
	info.Recommended = ""
	info.SecurityOnly = false

	return info
}

// FetchWithFallback tries to fetch lifecycle data online with ETag support, falls back to cache, then embedded data
func (f *Fetcher) FetchWithFallback(goenvRoot string) (map[string]VersionInfo, error) {
	// Check if offline mode is enabled
	if utils.GoenvEnvVarOffline.IsTrue() {
		if f.debug {
			fmt.Println("Debug: GOENV_OFFLINE=1, using embedded lifecycle data")
		}
		return EmbeddedLifecycleData, nil
	}

	// Try to get cached data first for ETag support
	var cachedData *LifecycleCache
	var cacheErr error
	if f.cacheDir != "" {
		cachedData, cacheErr = f.loadCache()
	}

	var etag string
	if cacheErr == nil && cachedData != nil {
		etag = cachedData.ETag
	}

	// Try to fetch online with ETag for conditional request
	data, newETag, err := f.FetchLifecycleDataWithETag(etag)
	if err == nil {
		// Check if content was modified (newETag will be empty if 304 Not Modified)
		if newETag == "" && etag != "" {
			// Server returned 304 Not Modified - use cached data
			if f.debug {
				fmt.Println("Debug: Server returned 304 Not Modified, using cached lifecycle data")
			}
			if cacheErr == nil && cachedData != nil {
				return cachedData.Data, nil
			}
		} else {
			// Success! Cache the result with ETag for future use
			if f.cacheDir != "" {
				if cacheErr := f.cacheData(data, newETag); cacheErr != nil && f.debug {
					fmt.Printf("Debug: Failed to cache lifecycle data: %v\n", cacheErr)
				}
			}
			return data, nil
		}
	}

	// Online fetch failed, try cache
	if f.debug {
		fmt.Printf("Debug: Online fetch failed (%v), trying cache...\n", err)
	}

	if cacheErr == nil && cachedData != nil && !cachedData.IsStale(24*time.Hour) {
		if f.debug {
			fmt.Printf("Debug: Using cached lifecycle data (last updated: %s)\n", cachedData.UpdatedAt.Format(time.RFC3339))
		}
		return cachedData.Data, nil
	}

	// Cache is also stale or missing, but try it anyway if available (better than nothing)
	if cacheErr == nil && cachedData != nil {
		if f.debug {
			fmt.Printf("Debug: Using stale cache (last updated: %s)\n", cachedData.UpdatedAt.Format(time.RFC3339))
		}
		return cachedData.Data, nil
	}

	// No cache available, use embedded data as last resort
	if f.debug {
		fmt.Println("Debug: Using embedded fallback lifecycle data")
	}

	return EmbeddedLifecycleData, nil
}

// loadCache loads lifecycle data from cache
func (f *Fetcher) loadCache() (*LifecycleCache, error) {
	if f.cacheDir == "" {
		return nil, fmt.Errorf("no cache directory configured")
	}

	cachePath := filepath.Join(f.cacheDir, "lifecycle-cache.json")

	var cache LifecycleCache
	if err := utils.UnmarshalJSONFile(cachePath, &cache); err != nil {
		return nil, fmt.Errorf("failed to read/parse cache file: %w", err)
	}

	// Verify SHA256 integrity if present
	if cache.SHA256 != "" {
		if err := f.verifySHA256(&cache); err != nil {
			return nil, fmt.Errorf("cache integrity check failed: %w", err)
		}
	}

	return &cache, nil
}

// verifySHA256 verifies the SHA256 hash of the lifecycle data
func (f *Fetcher) verifySHA256(cache *LifecycleCache) error {
	// Compute SHA256 of data
	dataJSON, err := json.Marshal(cache.Data)
	if err != nil {
		return errors.FailedTo("marshal data for verification", err)
	}

	computed := utils.SHA256Bytes(dataJSON)

	if computed != cache.SHA256 {
		return fmt.Errorf("SHA256 mismatch: expected %s, got %s", cache.SHA256, computed)
	}

	return nil
}

// cacheData saves lifecycle data to cache with ETag and SHA256 integrity
func (f *Fetcher) cacheData(data map[string]VersionInfo, etag string) error {
	if f.cacheDir == "" {
		return fmt.Errorf("no cache directory configured")
	}

	// Ensure cache directory exists with secure permissions
	if err := utils.EnsureDirWithContext(f.cacheDir, "create cache directory"); err != nil {
		return err
	}

	// Compute SHA256 hash of data for integrity
	dataJSON, err := json.Marshal(data)
	if err != nil {
		return errors.FailedTo("marshal data for hashing", err)
	}

	hash := utils.SHA256Bytes(dataJSON)

	cache := LifecycleCache{
		Data:      data,
		UpdatedAt: time.Now(),
		ETag:      etag,
		SHA256:    hash,
	}

	cacheData, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return errors.FailedTo("marshal cache", err)
	}

	cachePath := filepath.Join(f.cacheDir, "lifecycle-cache.json")
	// Write with secure permissions
	if err := utils.WriteFileWithContext(cachePath, cacheData, utils.PermFileSecure, "write cache file"); err != nil {
		return errors.FailedTo("write cache file", err)
	}

	return nil
}

// IsStale checks if cached data is older than the specified duration
func (c *LifecycleCache) IsStale(maxAge time.Duration) bool {
	return time.Since(c.UpdatedAt) > maxAge
}
