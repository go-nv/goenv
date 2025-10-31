package utils

import (
	"fmt"
	"strings"
)

// CompareGoVersions compares two Go version strings.
// Returns: 1 if v1 > v2, -1 if v1 < v2, 0 if equal
//
// This function handles:
//   - Version prefixes (e.g., "go1.21.0" vs "1.21.0")
//   - Semantic versioning (e.g., "1.21.0" vs "1.20.5")
//   - Pre-release versions (e.g., "1.22beta1" vs "1.21.5")
//   - Release candidates (e.g., "1.22rc1" vs "1.22beta1")
//
// Comparison rules:
//   - Stable versions > pre-release versions (1.22.0 > 1.22rc1 > 1.22beta1)
//   - RC versions > beta versions (1.22rc1 > 1.22beta1)
//   - Numeric comparison for version parts (1.21.5 > 1.21.0)
func CompareGoVersions(v1, v2 string) int {
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

		// Handle special suffixes like "beta", "rc", "alpha"
		p1HasPre := strings.Contains(p1, "beta") || strings.Contains(p1, "rc") || strings.Contains(p1, "alpha")
		p2HasPre := strings.Contains(p2, "beta") || strings.Contains(p2, "rc") || strings.Contains(p2, "alpha")

		if p1HasPre && !p2HasPre {
			return -1 // stable version is greater than beta/rc
		} else if !p1HasPre && p2HasPre {
			return 1 // stable version is greater than beta/rc
		} else if p1HasPre && p2HasPre {
			// Both have pre-release, rc > beta > alpha
			p1IsRC := strings.Contains(p1, "rc")
			p2IsRC := strings.Contains(p2, "rc")
			p1IsBeta := strings.Contains(p1, "beta")
			p2IsBeta := strings.Contains(p2, "beta")

			if p1IsRC && !p2IsRC {
				return 1 // rc > beta/alpha
			} else if !p1IsRC && p2IsRC {
				return -1 // rc > beta/alpha
			} else if p1IsBeta && !p2IsBeta && !p2IsRC {
				return 1 // beta > alpha
			} else if !p1IsBeta && p2IsBeta && !p1IsRC {
				return -1 // beta > alpha
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

// ExtractMajorMinor extracts the major.minor version from a version string
// Examples: "1.21.5" -> "1.21", "go1.22.0" -> "1.22", "1.23rc1" -> "1.23"
func ExtractMajorMinor(version string) string {
	// Remove any prefix like "go" or "v"
	version = strings.TrimPrefix(version, "go")
	version = strings.TrimPrefix(version, "v")

	// Split by dots
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return ""
	}

	// Validate major is a number
	var major int
	if _, err := fmt.Sscanf(parts[0], "%d", &major); err != nil {
		return ""
	}

	// Handle minor version with suffixes like "21-rc1" or "21beta1"
	minorStr := parts[1]
	minorParts := strings.FieldsFunc(minorStr, func(r rune) bool {
		return r == '-' || r == '+' || r == '~' || !('0' <= r && r <= '9')
	})

	if len(minorParts) == 0 {
		return ""
	}

	var minor int
	if _, err := fmt.Sscanf(minorParts[0], "%d", &minor); err != nil {
		return ""
	}

	return fmt.Sprintf("%d.%d", major, minor)
}

// SplitVersions splits a colon-delimited version string into individual versions
// Examples: "1.21.0:1.20.5" → ["1.21.0", "1.20.5"], "1.21.0" → ["1.21.0"]
// Empty strings and empty segments are filtered out
func SplitVersions(version string) []string {
	if version == "" {
		return []string{}
	}

	result := []string{}
	current := ""

	for _, ch := range version {
		if ch == ':' {
			if current != "" {
				result = append(result, current)
				current = ""
			}
		} else {
			current += string(ch)
		}
	}

	if current != "" {
		result = append(result, current)
	}

	return result
}

// ParseVersionTuple parses a version string into (major, minor, patch) components.
// Returns an error if the version string is invalid.
// Examples:
//   - "1.21.5" → (1, 21, 5, nil)
//   - "go1.21.5" → (1, 21, 5, nil)
//   - "1.21" → (1, 21, 0, nil)
//   - "invalid" → (0, 0, 0, error)
func ParseVersionTuple(ver string) (major, minor, patch int, err error) {
	// Remove common prefixes
	ver = strings.TrimPrefix(ver, "go")
	ver = strings.TrimPrefix(ver, "v")

	// Try to parse all three components
	n, _ := fmt.Sscanf(ver, "%d.%d.%d", &major, &minor, &patch)

	if n == 0 {
		return 0, 0, 0, fmt.Errorf("invalid version format: %q", ver)
	}

	// Two components is valid (e.g., "1.21")
	if n >= 2 {
		return major, minor, patch, nil
	}

	// Only one component is invalid for Go versions
	return 0, 0, 0, fmt.Errorf("invalid version format: %q (need at least major.minor)", ver)
}
