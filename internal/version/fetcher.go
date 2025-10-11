package version

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"runtime"
	"sort"
	"strings"
	"time"
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

// Fetcher handles fetching Go version information
type Fetcher struct {
	client  *http.Client
	baseURL string
	debug   bool
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

// FetchAvailableVersions fetches all available Go versions from the official API
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
