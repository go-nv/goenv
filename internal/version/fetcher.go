package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/utils"
)

// GoRelease represents a Go release from the official API
type GoRelease struct {
	Version string   `json:"version"`
	Stable  bool     `json:"stable"`
	Files   []GoFile `json:"files"`
}

// GoFile represents a downloadable file for a Go version
type GoFile struct {
	Filename string `json:"filename"`
	OS       string `json:"os"`
	Arch     string `json:"arch"`
	Version  string `json:"version"`
	SHA256   string `json:"sha256"`
	Size     int64  `json:"size"`
	Kind     string `json:"kind"`
}

// VersionCache represents cached version information
type VersionCache struct {
	Versions     []string  `json:"versions"`
	UpdatedAt    time.Time `json:"updated_at"`
	FullFetchAt  time.Time `json:"full_fetch_at"`  // Last time we fetched with include=all
	QuickCheckAt time.Time `json:"quick_check_at"` // Last time we checked latest versions only
}

// Fetcher handles fetching Go version information
type Fetcher struct {
	client   *http.Client
	baseURL  string
	debug    bool
	cacheDir string
}

// NewFetcher creates a new version fetcher
func NewFetcher() *Fetcher {
	return &Fetcher{
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		baseURL: "https://go.dev/dl/",
		debug:   false,
	}
}

// NewFetcherWithCache creates a new version fetcher with a cache directory
func NewFetcherWithCache(cacheDir string) *Fetcher {
	f := NewFetcher()
	f.cacheDir = cacheDir
	return f
}

// FetcherOptions contains options for creating a Fetcher
type FetcherOptions struct {
	Debug bool
}

// NewFetcherWithOptions creates a new version fetcher with options
func NewFetcherWithOptions(opts FetcherOptions) *Fetcher {
	f := NewFetcher()
	f.debug = opts.Debug
	return f
}

// SetDebug sets the debug flag
func (f *Fetcher) SetDebug(debug bool) {
	f.debug = debug
}

// IsPrerelease checks if a version string is a pre-release (beta, rc, etc.)
func IsPrerelease(version string) bool {
	lower := strings.ToLower(version)
	return strings.Contains(lower, "beta") ||
		strings.Contains(lower, "rc") ||
		strings.Contains(lower, "alpha") ||
		strings.Contains(lower, "preview")
}

// FetchAvailableVersions fetches latest Go versions from the official API (typically 2 latest)
func (f *Fetcher) FetchAvailableVersions() ([]GoRelease, error) {
	url := f.baseURL + "?mode=json"

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Go versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var releases []GoRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return releases, nil
}

// FetchAllReleases fetches all Go releases (including historical) from the official API
func (f *Fetcher) FetchAllReleases() ([]GoRelease, error) {
	url := f.baseURL + "?mode=json&include=all"

	if f.debug {
		fmt.Println("Debug: Fetching all releases from go.dev API...")
	}

	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Go versions: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	var releases []GoRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse JSON response: %w", err)
	}

	return releases, nil
}

// GetLatestVersion returns the latest stable Go version
func (f *Fetcher) GetLatestVersion() (*GoRelease, error) {
	releases, err := f.FetchAvailableVersions()
	if err != nil {
		return nil, err
	}

	for _, release := range releases {
		if release.Stable {
			return &release, nil
		}
	}

	return nil, fmt.Errorf("no stable release found")
}

// GetVersionsForPlatform filters versions for the current or specified platform
func (f *Fetcher) GetVersionsForPlatform(goos, goarch string) ([]GoRelease, error) {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	releases, err := f.FetchAvailableVersions()
	if err != nil {
		return nil, err
	}

	var filtered []GoRelease
	for _, release := range releases {
		var platformFiles []GoFile
		for _, file := range release.Files {
			if file.OS == goos && file.Arch == goarch && file.Kind == "archive" {
				platformFiles = append(platformFiles, file)
			}
		}
		if len(platformFiles) > 0 {
			release.Files = platformFiles
			filtered = append(filtered, release)
		}
	}

	return filtered, nil
}

// SortVersions sorts Go versions in descending order (newest first)
func SortVersions(versions []GoRelease) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersions(versions[i].Version, versions[j].Version) > 0
	})
}

// compareVersions compares two version strings
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
func compareVersions(v1, v2 string) int {
	// Remove "go" prefix if present
	v1 = strings.TrimPrefix(v1, "go")
	v2 = strings.TrimPrefix(v2, "go")

	// Split versions into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 string
		if i < len(parts1) {
			p1 = parts1[i]
		} else {
			p1 = "0"
		}
		if i < len(parts2) {
			p2 = parts2[i]
		} else {
			p2 = "0"
		}

		// Handle special suffixes like "beta", "rc"
		p1HasPre := strings.Contains(p1, "beta") || strings.Contains(p1, "rc")
		p2HasPre := strings.Contains(p2, "beta") || strings.Contains(p2, "rc")

		if p1HasPre && !p2HasPre {
			return -1 // stable version is greater than beta/rc
		} else if !p1HasPre && p2HasPre {
			return 1 // stable version is greater than beta/rc
		} else if p1HasPre && p2HasPre {
			// Both have pre-release, rc > beta
			p1IsRC := strings.Contains(p1, "rc")
			p2IsRC := strings.Contains(p2, "rc")
			if p1IsRC && !p2IsRC {
				return 1 // rc > beta
			} else if !p1IsRC && p2IsRC {
				return -1 // rc > beta
			}
		}

		// Convert to integers for numeric comparison
		var n1, n2 int
		fmt.Sscanf(p1, "%d", &n1)
		fmt.Sscanf(p2, "%d", &n2)

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}

// GetFileForPlatform returns the download file for a specific platform
func (r *GoRelease) GetFileForPlatform(goos, goarch string) (*GoFile, error) {
	if goos == "" {
		goos = runtime.GOOS
	}
	if goarch == "" {
		goarch = runtime.GOARCH
	}

	for _, file := range r.Files {
		if file.OS == goos && file.Arch == goarch && file.Kind == "archive" {
			return &file, nil
		}
	}

	return nil, fmt.Errorf("no file found for platform %s/%s", goos, goarch)
}

// FetchAllVersions fetches all available Go versions from official API with caching
func (f *Fetcher) FetchAllVersions() ([]string, error) {
	// Check if offline mode is enabled
	if utils.GoenvEnvVarOffline.IsTrue() {
		if f.debug {
			fmt.Println("Debug: GOENV_OFFLINE=1, using embedded versions only")
		}
		// Extract version names from embedded data
		versions := make([]string, 0, len(EmbeddedVersions))
		for _, release := range EmbeddedVersions {
			versions = append(versions, release.Version)
		}
		return versions, nil
	}

	// Try to load from cache first
	if f.cacheDir != "" {
		cachedVersions, err := f.loadCachedVersions()
		if err == nil {
			if f.debug {
				fmt.Println("Debug: Using cached versions")
			}
			return cachedVersions, nil
		}
		if f.debug {
			fmt.Printf("Debug: Cache miss or expired: %v\n", err)
		}
	}

	// Fetch from official go.dev API with include=all parameter
	if f.debug {
		fmt.Println("Debug: Fetching all versions from go.dev API...")
	}

	url := f.baseURL + "?mode=json&include=all"
	resp, err := f.client.Get(url)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch versions from go.dev API: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code from go.dev API: %d", resp.StatusCode)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	// Parse API response (array of GoRelease objects)
	var releases []GoRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return nil, fmt.Errorf("failed to parse go.dev API response: %w", err)
	}

	// Extract version names
	versions := make([]string, 0, len(releases))
	for _, release := range releases {
		versions = append(versions, release.Version)
	}

	// Sort versions (newest first)
	sortVersionStrings(versions)

	// Cache the results
	if f.cacheDir != "" {
		if err := f.cacheVersions(versions); err != nil {
			if f.debug {
				fmt.Printf("Debug: Failed to cache versions: %v\n", err)
			}
			// Don't fail if caching fails
		}
	}

	return versions, nil
}

// loadCachedVersions loads versions from cache with smart freshness checking
func (f *Fetcher) loadCachedVersions() ([]string, error) {
	if f.cacheDir == "" {
		return nil, fmt.Errorf("no cache directory configured")
	}

	cachePath := filepath.Join(f.cacheDir, "versions-cache.json")
	data, err := os.ReadFile(cachePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read cache file: %w", err)
	}

	var cache VersionCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, fmt.Errorf("failed to parse cache file: %w", err)
	}

	// Strategy 1: If cache is very fresh (< 6 hours), use it without checking
	if time.Since(cache.UpdatedAt) < 6*time.Hour {
		if f.debug {
			fmt.Printf("Debug: Cache is fresh (< 6 hours old)\n")
		}
		return cache.Versions, nil
	}

	// Strategy 2: If cache is old but not ancient (< 7 days), do a quick check
	if time.Since(cache.UpdatedAt) < 7*24*time.Hour {
		// Quick check: fetch just latest 2 versions (fast, no include=all)
		if f.debug {
			fmt.Printf("Debug: Cache is %v old, doing quick freshness check...\n", time.Since(cache.UpdatedAt).Round(time.Hour))
		}

		latestReleases, err := f.FetchAvailableVersions() // Just 2 versions, fast!
		if err != nil {
			// Network error, use cache anyway
			if f.debug {
				fmt.Printf("Debug: Quick check failed (%v), using cache anyway\n", err)
			}
			return cache.Versions, nil
		}

		// Check if our cached latest version matches API latest
		if len(latestReleases) > 0 && len(cache.Versions) > 0 {
			if latestReleases[0].Version == cache.Versions[0] {
				// Cache is still current!
				if f.debug {
					fmt.Printf("Debug: Cache is current (latest: %s)\n", cache.Versions[0])
				}
				return cache.Versions, nil
			}
			// New version detected, force full refresh
			if f.debug {
				fmt.Printf("Debug: New version detected (cached: %s, latest: %s), forcing full refresh\n",
					cache.Versions[0], latestReleases[0].Version)
			}
			return nil, fmt.Errorf("new version detected, need full refresh")
		}
	}

	// Strategy 3: Cache is ancient (> 7 days), force refresh
	if f.debug {
		fmt.Printf("Debug: Cache is stale (> 7 days old), forcing full refresh\n")
	}
	return nil, fmt.Errorf("cache expired")
}

// cacheVersions saves versions to cache
func (f *Fetcher) cacheVersions(versions []string) error {
	if f.cacheDir == "" {
		return fmt.Errorf("no cache directory configured")
	}

	// Ensure cache directory exists
	if err := os.MkdirAll(f.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	cache := VersionCache{
		Versions:  versions,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	cachePath := filepath.Join(f.cacheDir, "versions-cache.json")
	if err := os.WriteFile(cachePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	return nil
}

// sortVersionStrings sorts version strings in descending order (newest first)
func sortVersionStrings(versions []string) {
	sort.Slice(versions, func(i, j int) bool {
		return compareVersionStrings(versions[i], versions[j]) > 0
	})
}

// compareVersionStrings compares two version strings
func compareVersionStrings(v1, v2 string) int {
	// Remove "go" prefix if present
	v1 = strings.TrimPrefix(v1, "go")
	v2 = strings.TrimPrefix(v2, "go")

	// Split versions into parts
	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	// Compare each part
	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var p1, p2 string
		if i < len(parts1) {
			p1 = parts1[i]
		} else {
			p1 = "0"
		}
		if i < len(parts2) {
			p2 = parts2[i]
		} else {
			p2 = "0"
		}

		// Handle special suffixes like "beta", "rc"
		p1HasPre := strings.Contains(p1, "beta") || strings.Contains(p1, "rc")
		p2HasPre := strings.Contains(p2, "beta") || strings.Contains(p2, "rc")

		if p1HasPre && !p2HasPre {
			return -1 // stable version is greater than beta/rc
		} else if !p1HasPre && p2HasPre {
			return 1 // stable version is greater than beta/rc
		} else if p1HasPre && p2HasPre {
			// Both have pre-release, rc > beta
			p1IsRC := strings.Contains(p1, "rc")
			p2IsRC := strings.Contains(p2, "rc")
			if p1IsRC && !p2IsRC {
				return 1 // rc > beta
			} else if !p1IsRC && p2IsRC {
				return -1 // rc > beta
			}
		}

		// Extract numeric part from string
		re := regexp.MustCompile(`\d+`)
		matches1 := re.FindString(p1)
		matches2 := re.FindString(p2)

		var n1, n2 int
		if matches1 != "" {
			fmt.Sscanf(matches1, "%d", &n1)
		}
		if matches2 != "" {
			fmt.Sscanf(matches2, "%d", &n2)
		}

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}
