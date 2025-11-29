// Package lifecycle provides Go version support status information
package lifecycle

import (
	"fmt"
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

// Go version lifecycle data based on Go's support policy:
// Each major Go release is supported until there are two newer major releases.
// For example, Go 1.21 is supported until Go 1.23 is released.
//
// Data sources:
// - https://go.dev/doc/devel/release
// - https://endoflife.date/go
var versionLifecycle = map[string]VersionInfo{
	// Current and recent versions (as of Oct 2025)
	"1.25": {
		Version:      "1.25",
		ReleaseDate:  parseDate("2025-08-01"),
		EOLDate:      parseDate("2026-08-01"), // Estimated: when 1.27 releases
		Status:       StatusCurrent,
		Recommended:  "",
		SecurityOnly: false,
	},
	"1.24": {
		Version:      "1.24",
		ReleaseDate:  parseDate("2025-02-01"),
		EOLDate:      parseDate("2026-02-01"), // Estimated: when 1.26 releases
		Status:       StatusCurrent,
		Recommended:  "",
		SecurityOnly: false,
	},
	"1.23": {
		Version:      "1.23",
		ReleaseDate:  parseDate("2024-08-13"),
		EOLDate:      parseDate("2025-11-30"), // Approaching EOL (within 3 months from now)
		Status:       StatusNearEOL,
		Recommended:  "1.25",
		SecurityOnly: true,
	},
	"1.22": {
		Version:      "1.22",
		ReleaseDate:  parseDate("2024-02-06"),
		EOLDate:      parseDate("2025-02-01"), // When 1.24 was released
		Status:       StatusEOL,
		Recommended:  "1.25",
		SecurityOnly: false,
	},
	"1.21": {
		Version:      "1.21",
		ReleaseDate:  parseDate("2023-08-08"),
		EOLDate:      parseDate("2024-08-13"), // When 1.23 was released
		Status:       StatusEOL,
		Recommended:  "1.25",
		SecurityOnly: false,
	},
	"1.20": {
		Version:      "1.20",
		ReleaseDate:  parseDate("2023-02-01"),
		EOLDate:      parseDate("2024-02-06"), // When 1.22 was released
		Status:       StatusEOL,
		Recommended:  "1.25",
		SecurityOnly: false,
	},
	"1.19": {
		Version:      "1.19",
		ReleaseDate:  parseDate("2022-08-02"),
		EOLDate:      parseDate("2023-08-08"), // When 1.21 was released
		Status:       StatusEOL,
		Recommended:  "1.25",
		SecurityOnly: false,
	},
	"1.18": {
		Version:      "1.18",
		ReleaseDate:  parseDate("2022-03-15"),
		EOLDate:      parseDate("2023-02-01"), // When 1.20 was released
		Status:       StatusEOL,
		Recommended:  "1.25",
		SecurityOnly: false,
	},
	"1.17": {
		Version:      "1.17",
		ReleaseDate:  parseDate("2021-08-16"),
		EOLDate:      parseDate("2022-08-02"), // When 1.19 was released
		Status:       StatusEOL,
		Recommended:  "1.25",
		SecurityOnly: false,
	},
	"1.16": {
		Version:      "1.16",
		ReleaseDate:  parseDate("2021-02-16"),
		EOLDate:      parseDate("2022-03-15"), // When 1.18 was released
		Status:       StatusEOL,
		Recommended:  "1.25",
		SecurityOnly: false,
	},
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

	info, found := versionLifecycle[majorMinor]
	if !found {
		return VersionInfo{
			Version: majorMinor,
			Status:  StatusUnknown,
		}, false
	}

	// Update status dynamically based on current date
	info.Status = calculateStatus(info)
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
