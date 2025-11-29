package manager

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
)

// VersionSource represents where a version was discovered from
type VersionSource string

const (
	SourceGoVersion VersionSource = config.VersionFileName
	SourceGoMod     VersionSource = config.GoModFileName
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
// It checks both .go-version and go.mod, with special handling for toolchain directives.
//
// Precedence rules:
// 1. If go.mod has a toolchain directive, it takes precedence (it's a hard requirement)
// 2. If .go-version exists and is >= toolchain (or no toolchain), use .go-version
// 3. If only one exists, use that
//
// Returns nil if no version is found
func DiscoverVersion(dir string) (*DiscoveredVersion, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return nil, fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	// Check for .go-version
	versionFile := filepath.Join(dir, config.VersionFileName)
	var goVersionFileVer string
	if utils.PathExists(versionFile) {
		content, err := os.ReadFile(versionFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read .go-version: %w", err)
		}
		goVersionFileVer = parseVersionContent(string(content))
	}

	// Check for go.mod (with toolchain precedence)
	gomodPath := filepath.Join(dir, config.GoModFileName)
	var goModVer string
	if utils.PathExists(gomodPath) {
		var err error
		goModVer, err = ParseGoModVersion(gomodPath)
		if err != nil {
			return nil, fmt.Errorf("failed to parse go.mod: %w", err)
		}
	}

	// Decision logic based on what we found
	if goModVer != "" && goVersionFileVer != "" {
		// Both exist - go.mod toolchain takes precedence as it's a project requirement
		// But if .go-version is explicitly newer, respect user's choice
		if VersionSatisfies(goVersionFileVer, goModVer) {
			// .go-version is same or newer - use it
			return &DiscoveredVersion{
				Version: goVersionFileVer,
				Source:  SourceGoVersion,
				Path:    versionFile,
			}, nil
		}
		// .go-version is older than go.mod requirement - prefer go.mod
		return &DiscoveredVersion{
			Version: goModVer,
			Source:  SourceGoMod,
			Path:    gomodPath,
		}, nil
	}

	// Only one exists (or neither)
	if goVersionFileVer != "" {
		return &DiscoveredVersion{
			Version: goVersionFileVer,
			Source:  SourceGoVersion,
			Path:    versionFile,
		}, nil
	}

	if goModVer != "" {
		return &DiscoveredVersion{
			Version: goModVer,
			Source:  SourceGoMod,
			Path:    gomodPath,
		}, nil
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
	versionFile := filepath.Join(dir, config.VersionFileName)
	hasGoVersion := false
	if utils.PathExists(versionFile) {
		content, err := os.ReadFile(versionFile)
		if err == nil {
			goVersionVer = parseVersionContent(string(content))
			hasGoVersion = goVersionVer != ""
		}
	}

	// Check for go.mod
	gomodPath := filepath.Join(dir, config.GoModFileName)
	hasGoMod := false
	if utils.PathExists(gomodPath) {
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
