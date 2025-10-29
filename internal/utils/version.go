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
