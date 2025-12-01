// Package lifecycle provides Go version support status information
package lifecycle

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/go-nv/goenv/internal/utils"
)

// SupportStatus represents the support state of a Go version
type SupportStatus int

const (
	// StatusCurrent indicates actively supported version
	StatusCurrent SupportStatus = iota
	// StatusNearEOL indicates version approaching end of life (within 3 months)
	StatusNearEOL
	// StatusEOL indicates version is no longer supported
	StatusEOL
	// StatusUnknown indicates version status cannot be determined
	StatusUnknown
)

// VersionInfo contains lifecycle information for a Go version
type VersionInfo struct {
	Version      string        // e.g., "1.21"
	ReleaseDate  time.Time     // When version was released
	EOLDate      time.Time     // When support ends
	Status       SupportStatus // Current support status
	Recommended  string        // Recommended upgrade version
	SecurityOnly bool          // Whether only security updates are provided
}

var (
	// versionLifecycle stores the dynamically loaded lifecycle data
	versionLifecycle map[string]VersionInfo
	// lifecycleMutex protects versionLifecycle from concurrent access
	lifecycleMutex sync.RWMutex
	// lifecycleInitialized tracks if we've loaded lifecycle data
	lifecycleInitialized bool
)

// InitializeLifecycleData loads lifecycle data from API/cache/embedded fallback
func InitializeLifecycleData() error {
	lifecycleMutex.Lock()
	defer lifecycleMutex.Unlock()

	if lifecycleInitialized {
		return nil // Already initialized
	}

	// Determine cache directory (use GOENV_ROOT or HOME/.goenv)
	goenvRoot := os.Getenv("GOENV_ROOT")
	if goenvRoot == "" {
		home, err := os.UserHomeDir()
		if err == nil {
			goenvRoot = filepath.Join(home, ".goenv")
		}
	}

	// Create fetcher with cache
	fetcher := NewFetcherWithCache(goenvRoot)

	// Fetch with fallback to cache and embedded data
	data, err := fetcher.FetchWithFallback(goenvRoot)
	if err != nil {
		// If fetching fails, use embedded data as ultimate fallback
		versionLifecycle = EmbeddedLifecycleData
	} else {
		versionLifecycle = data
	}

	lifecycleInitialized = true
	return nil
}

// getLifecycleData returns the lifecycle data, initializing if needed
func getLifecycleData() map[string]VersionInfo {
	lifecycleMutex.RLock()
	initialized := lifecycleInitialized
	lifecycleMutex.RUnlock()

	if !initialized {
		// Initialize on first access
		_ = InitializeLifecycleData()
	}

	lifecycleMutex.RLock()
	defer lifecycleMutex.RUnlock()
	return versionLifecycle
}

// parseDate is a helper to parse dates in lifecycle data
func parseDate(date string) time.Time {
	t, _ := time.Parse("2006-01-02", date)
	return t
}

// GetVersionInfo returns lifecycle information for a Go version
func GetVersionInfo(version string) (VersionInfo, bool) {
	// Extract major.minor from version string
	// Handle formats: 1.21, 1.21.5, 1.21.0, 1.21-rc1, etc.
	majorMinor := utils.ExtractMajorMinor(version)
	if majorMinor == "" {
		return VersionInfo{}, false
	}

	// Get lifecycle data (initializes on first call)
	lifecycleData := getLifecycleData()

	info, found := lifecycleData[majorMinor]
	if !found {
		return VersionInfo{
			Version: majorMinor,
			Status:  StatusUnknown,
		}, false
	}

	// Update status dynamically based on current date
	info.Status = calculateStatus(info)

	// Calculate recommended version and security-only status
	info = calculateDynamicFields(info, lifecycleData)

	return info, true
}

// calculateStatus determines current status based on dates
func calculateStatus(info VersionInfo) SupportStatus {
	now := time.Now()

	// If EOL date is in the past, it's EOL
	if now.After(info.EOLDate) {
		return StatusEOL
	}

	// If EOL is within 3 months, it's near EOL
	threeMonthsFromNow := now.AddDate(0, 3, 0)
	if info.EOLDate.Before(threeMonthsFromNow) {
		return StatusNearEOL
	}

	return StatusCurrent
}

// calculateDynamicFields determines recommended version and security-only status
func calculateDynamicFields(info VersionInfo, allVersions map[string]VersionInfo) VersionInfo {
	// Find the latest stable version to recommend
	var latestVersion string
	var latestReleaseDate time.Time

	for ver, verInfo := range allVersions {
		status := calculateStatus(verInfo)
		// Only recommend current (non-EOL, non-near-EOL) versions
		if status == StatusCurrent {
			if verInfo.ReleaseDate.After(latestReleaseDate) {
				latestVersion = ver
				latestReleaseDate = verInfo.ReleaseDate
			}
		}
	}

	// Set recommended version for EOL and near-EOL versions
	if info.Status == StatusEOL || info.Status == StatusNearEOL {
		info.Recommended = latestVersion
	}

	// Set security-only flag for near-EOL versions
	if info.Status == StatusNearEOL {
		info.SecurityOnly = true
	}

	return info
}

// IsSupported returns true if the version is currently supported
func IsSupported(version string) bool {
	info, found := GetVersionInfo(version)
	if !found {
		// Unknown versions are assumed to be supported (could be newer)
		return true
	}
	return info.Status == StatusCurrent || info.Status == StatusNearEOL
}

// IsEOL returns true if the version is end-of-life
func IsEOL(version string) bool {
	info, found := GetVersionInfo(version)
	if !found {
		return false
	}
	return info.Status == StatusEOL
}

// IsNearEOL returns true if the version is approaching end-of-life
func IsNearEOL(version string) bool {
	info, found := GetVersionInfo(version)
	if !found {
		return false
	}
	return info.Status == StatusNearEOL
}

// GetRecommendedVersion returns the recommended upgrade version
func GetRecommendedVersion(version string) string {
	info, found := GetVersionInfo(version)
	if !found {
		// Unknown version - suggest latest
		return "latest stable version"
	}
	// Return the recommended version (may be empty for current versions)
	return info.Recommended
}

// FormatWarning formats a user-friendly warning message for an outdated version
func FormatWarning(version string) string {
	info, found := GetVersionInfo(version)
	if !found {
		return ""
	}

	majorMinor := utils.ExtractMajorMinor(version)
	switch info.Status {
	case StatusEOL:
		return fmt.Sprintf(
			"Go %s is no longer supported (EOL: %s)\nConsider upgrading to %s for security updates and bug fixes.",
			majorMinor,
			info.EOLDate.Format("2006-01-02"),
			info.Recommended,
		)
	case StatusNearEOL:
		return fmt.Sprintf(
			"Go %s support ends soon (%s)\nConsider planning an upgrade to %s.",
			majorMinor,
			info.EOLDate.Format("2006-01-02"),
			info.Recommended,
		)
	default:
		return ""
	}
}
