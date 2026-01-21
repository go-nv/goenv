// Package tools provides utilities and operations for managing Go tools across versions.
package tools

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// GetLatestVersion queries the Go module proxy for the latest version of a package.
func GetLatestVersion(packagePath string) (string, error) {
	// Use go list to get the latest version
	output, err := utils.RunCommandOutput("go", "list", "-m", "-versions", "-json", packagePath+"@latest")
	if err != nil {
		return "", errors.FailedTo("query latest version", err)
	}

	// Parse JSON output
	var info struct {
		Version string
	}

	if err := json.Unmarshal([]byte(output), &info); err != nil {
		return "", errors.FailedTo("parse version info", err)
	}

	return info.Version, nil
}

// ParseSemver parses a semantic version string into major, minor, and patch components.
// Handles versions with or without 'v' prefix and pre-release suffixes.
// Examples: "v1.2.3" -> (1, 2, 3), "2.0.1-rc1" -> (2, 0, 1)
func ParseSemver(version string) (major, minor, patch int, err error) {
	// Remove 'v' prefix if present
	version = strings.TrimPrefix(version, "v")

	// Remove pre-release/build metadata suffixes (e.g., -rc1, +build123)
	if idx := strings.IndexAny(version, "-+"); idx != -1 {
		version = version[:idx]
	}

	// Split into components
	parts := strings.Split(version, ".")
	if len(parts) < 2 {
		return 0, 0, 0, fmt.Errorf("invalid semver format: %s", version)
	}

	// Parse major
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid major version: %w", err)
	}

	// Parse minor
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("invalid minor version: %w", err)
	}

	// Parse patch (optional - some versions only have major.minor)
	if len(parts) >= 3 {
		patch, err = strconv.Atoi(parts[2])
		if err != nil {
			return 0, 0, 0, fmt.Errorf("invalid patch version: %w", err)
		}
	}

	return major, minor, patch, nil
}

// CompareVersions compares two version strings (semantic versioning).
// Returns: -1 if v1 < v2, 0 if v1 == v2, 1 if v1 > v2
func CompareVersions(v1, v2 string) int {
	// Handle special cases first
	v1Clean := strings.TrimPrefix(v1, "v")
	v2Clean := strings.TrimPrefix(v2, "v")

	if v1Clean == v2Clean {
		return 0
	}
	if v1Clean == "unknown" {
		return -1
	}
	if v2Clean == "unknown" {
		return 1
	}

	// Try proper semver comparison
	major1, minor1, patch1, err1 := ParseSemver(v1)
	major2, minor2, patch2, err2 := ParseSemver(v2)

	// If both parse successfully, use numeric comparison
	if err1 == nil && err2 == nil {
		if major1 != major2 {
			if major1 < major2 {
				return -1
			}
			return 1
		}
		if minor1 != minor2 {
			if minor1 < minor2 {
				return -1
			}
			return 1
		}
		if patch1 != patch2 {
			if patch1 < patch2 {
				return -1
			}
			return 1
		}
		return 0
	}

	// Fallback to string comparison if parsing fails
	if v1Clean < v2Clean {
		return -1
	} else if v1Clean > v2Clean {
		return 1
	}
	return 0
}
