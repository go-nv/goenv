package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// VersionSource represents where a version was discovered from
type VersionSource string

const (
	SourceGoVersion VersionSource = ".go-version"
	SourceGoMod     VersionSource = "go.mod"
	SourceNone      VersionSource = "none"
)

// parseVersionContent extracts the version from file content
// Returns the first non-empty, non-comment line
func parseVersionContent(content string) string {
	lines := strings.Split(content, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "#") {
			return line
		}
	}
	return ""
}

// DiscoveredVersion represents a version found in the current directory
type DiscoveredVersion struct {
	Version string
	Source  VersionSource
	Path    string // Full path to the file
}

// DiscoverVersion looks for a Go version in the current directory
// It checks .go-version first, then go.mod (with toolchain precedence)
// Returns nil if no version is found
func DiscoverVersion(dir string) (*DiscoveredVersion, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Check for .go-version first (explicit takes precedence)
	versionFile := filepath.Join(dir, ".go-version")
	if _, err := os.Stat(versionFile); err == nil {
		content, err := os.ReadFile(versionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read .go-version: %w", err)
		}
		version := parseVersionContent(string(content))
		if version != "" {
			return &DiscoveredVersion{
				Version: version,
				Source:  SourceGoVersion,
				Path:    versionFile,
			}, nil
		}
	}

	// Check for go.mod (with toolchain precedence)
	gomodPath := filepath.Join(dir, "go.mod")
	if _, err := os.Stat(gomodPath); err == nil {
		version, err := ParseGoModVersion(gomodPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse go.mod: %w", err)
		}
		if version != "" {
			return &DiscoveredVersion{
				Version: version,
				Source:  SourceGoMod,
				Path:    gomodPath,
			}, nil
		}
	}

	return nil, nil // No version found
}

// DiscoverVersionMismatch checks if .go-version and go.mod exist and have different versions
// Returns true if both exist and have different versions, along with the versions
func DiscoverVersionMismatch(dir string) (mismatch bool, goVersionVer string, goModVer string, err error) {
	if dir == "" {
		dir, err = os.Getwd()
		if err != nil {
			return false, "", "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Check for .go-version
	versionFile := filepath.Join(dir, ".go-version")
	hasGoVersion := false
	if _, err := os.Stat(versionFile); err == nil {
		content, err := os.ReadFile(versionFile)
		if err == nil {
			goVersionVer = parseVersionContent(string(content))
			hasGoVersion = goVersionVer != ""
		}
	}

	// Check for go.mod
	gomodPath := filepath.Join(dir, "go.mod")
	hasGoMod := false
	if _, err := os.Stat(gomodPath); err == nil {
		goModVer, err = ParseGoModVersion(gomodPath)
		if err == nil && goModVer != "" {
			hasGoMod = true
		}
	}

	// Only report mismatch if both exist and differ
	if hasGoVersion && hasGoMod && goVersionVer != goModVer {
		return true, goVersionVer, goModVer, nil
	}

	return false, goVersionVer, goModVer, nil
}
