// Package cache provides utilities and operations for managing Go build and module caches.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// CacheKind represents the type of cache.
type CacheKind string

const (
	// CacheKindBuild represents a Go build cache.
	CacheKindBuild CacheKind = "build"

	// CacheKindMod represents a Go module cache.
	CacheKindMod CacheKind = "mod"
)

// String returns the string representation of the cache kind.
func (k CacheKind) String() string {
	return string(k)
}

// TargetInfo contains platform-specific information extracted from cache names.
//
// Example: "go-build-darwin-arm64" → GOOS="darwin", GOARCH="arm64", ABI=nil
// Example: "go-build-linux-amd64-v3" → GOOS="linux", GOARCH="amd64", ABI={"GOAMD64":"v3"}
type TargetInfo struct {
	GOOS   string            // Operating system (darwin, linux, windows, etc.)
	GOARCH string            // Architecture (amd64, arm64, etc.)
	ABI    map[string]string // Additional ABI info (GOAMD64, GOARM, etc.)
}

// CGOToolchainInfo contains C compiler information used for builds.
type CGOToolchainInfo struct {
	CC       string   // C compiler path
	CXX      string   // C++ compiler path
	CFLAGS   []string // C compiler flags
	CXXFLAGS []string // C++ compiler flags
	LDFLAGS  []string // Linker flags
}

// CacheInfo contains metadata about a cache directory.
type CacheInfo struct {
	Kind      CacheKind         // "build" or "mod"
	Path      string            // Full path to cache directory
	GoVersion string            // Go version (e.g., "1.23.2")
	Target    *TargetInfo       // GOOS/GOARCH info (nil for old format or mod caches)
	SizeBytes int64             // Total size in bytes
	Files     int               // Number of files (-1 if approximate/timed out)
	ModTime   time.Time         // Most recent modification time
	OldFormat bool              // True if old non-architecture-aware format
	CGOInfo   *CGOToolchainInfo // CGO compiler info (build caches only)
}

// CacheStatus contains aggregate cache statistics for all Go versions.
type CacheStatus struct {
	BuildCaches []CacheInfo               // All build caches
	ModCaches   []CacheInfo               // All module caches
	TotalSize   int64                     // Total size of all caches
	TotalFiles  int                       // Total number of files (-1 if any cache timed out)
	ByVersion   map[string]*VersionCaches // Caches grouped by Go version
}

// VersionCaches groups cache information by Go version.
type VersionCaches struct {
	Version     string      // Go version (e.g., "1.23.2")
	BuildCaches []CacheInfo // Build caches for this version
	ModCache    *CacheInfo  // Module cache for this version (only one per version)
	TotalSize   int64       // Total size of all caches for this version
}

// DetectCacheKind determines whether a path is a build or module cache.
//
// Returns:
//   - CacheKindBuild if path contains "go-build"
//   - CacheKindMod if path contains "pkg/mod"
//   - Error if cache kind cannot be determined
func DetectCacheKind(path string) (CacheKind, error) {
	base := filepath.Base(path)

	// Check for build cache patterns
	if strings.HasPrefix(base, "go-build") {
		return CacheKindBuild, nil
	}

	// Check for module cache (usually in pkg/mod)
	if strings.Contains(path, "pkg/mod") || strings.Contains(path, "pkg\\mod") {
		return CacheKindMod, nil
	}

	return "", fmt.Errorf("unknown cache kind for path: %s", path)
}

// GetCacheInfo gathers metadata about a single cache directory.
//
// Parameters:
//   - cachePath: Full path to cache directory
//   - kind: Type of cache (build or mod)
//   - fast: If true, skips file counting for better performance
//
// Returns cache metadata including size, file count, modification time, and platform info.
func GetCacheInfo(cachePath string, kind CacheKind, fast bool) (*CacheInfo, error) {
	// Check if cache directory exists
	if !utils.DirExists(cachePath) {
		return nil, fmt.Errorf("cache directory does not exist: %s", cachePath)
	}

	info := &CacheInfo{
		Kind: kind,
		Path: cachePath,
	}

	// Extract version from path
	// Path format: $GOENV_ROOT/versions/<version>/pkg/...
	pathParts := strings.Split(filepath.ToSlash(cachePath), "/")
	for i, part := range pathParts {
		if part == "versions" && i+1 < len(pathParts) {
			info.GoVersion = pathParts[i+1]
			break
		}
	}

	// Get size and file count
	size, files, err := GetDirSizeWithOptions(cachePath, fast, 10*time.Second)
	if err != nil {
		return nil, errors.FailedTo("get directory size", err)
	}
	info.SizeBytes = size
	info.Files = files

	// Get modification time
	modTime, err := GetCacheModTime(cachePath)
	if err != nil {
		// Non-fatal - use zero time
		modTime = time.Time{}
	}
	info.ModTime = modTime

	// For build caches, extract platform info
	if kind == CacheKindBuild {
		cacheName := filepath.Base(cachePath)

		// Detect old format (just "go-build" without platform suffix)
		if cacheName == "go-build" {
			info.OldFormat = true
		} else {
			// Parse platform info from cache name
			goos, goarch, abi := ParseABIFromCacheName(cacheName)
			if goos != "" && goarch != "" {
				info.Target = &TargetInfo{
					GOOS:   goos,
					GOARCH: goarch,
					ABI:    abi,
				}
			} else {
				// Has suffix but couldn't parse - consider old format
				info.OldFormat = true
			}
		}

		// Try to get CGO toolchain info
		if cgoInfo := detectCGOInfo(cachePath); cgoInfo != nil {
			info.CGOInfo = cgoInfo
		}
	}

	return info, nil
}

// GetCacheStatus gathers metadata about all caches in the GOENV_ROOT.
//
// Parameters:
//   - goenvRoot: Path to GOENV_ROOT directory
//   - fast: If true, skips file counting for better performance
//
// Returns aggregate cache statistics including per-version breakdown.
func GetCacheStatus(goenvRoot string, fast bool) (*CacheStatus, error) {
	status := &CacheStatus{
		BuildCaches: make([]CacheInfo, 0),
		ModCaches:   make([]CacheInfo, 0),
		ByVersion:   make(map[string]*VersionCaches),
	}

	versionsDir := filepath.Join(goenvRoot, "versions")
	if !utils.DirExists(versionsDir) {
		// No versions directory - return empty status
		return status, nil
	}

	// Walk through all version directories
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return nil, errors.FailedTo("read versions directory", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		versionPath := filepath.Join(versionsDir, version)

		// Check for build caches in pkg/ directory (new format)
		buildCacheBase := filepath.Join(versionPath, "pkg")
		if buildEntries, err := os.ReadDir(buildCacheBase); err == nil {
			for _, buildEntry := range buildEntries {
				if !buildEntry.IsDir() || !strings.HasPrefix(buildEntry.Name(), "go-build") {
					continue
				}

				cachePath := filepath.Join(buildCacheBase, buildEntry.Name())
				cacheInfo, err := GetCacheInfo(cachePath, CacheKindBuild, fast)
				if err != nil {
					// Skip caches we can't read
					continue
				}

				status.BuildCaches = append(status.BuildCaches, *cacheInfo)
				status.TotalSize += cacheInfo.SizeBytes
				if status.TotalFiles >= 0 && cacheInfo.Files >= 0 {
					status.TotalFiles += cacheInfo.Files
				} else {
					status.TotalFiles = -1 // Mark as approximate
				}

				// Add to version-specific tracking
				if _, exists := status.ByVersion[version]; !exists {
					status.ByVersion[version] = &VersionCaches{
						Version:     version,
						BuildCaches: make([]CacheInfo, 0),
					}
				}
				status.ByVersion[version].BuildCaches = append(status.ByVersion[version].BuildCaches, *cacheInfo)
				status.ByVersion[version].TotalSize += cacheInfo.SizeBytes
			}
		}

		// Check for old-format build cache directly in version directory
		oldCachePath := filepath.Join(versionPath, "go-build")
		if utils.DirExists(oldCachePath) {
			cacheInfo, err := GetCacheInfo(oldCachePath, CacheKindBuild, fast)
			if err == nil {
				status.BuildCaches = append(status.BuildCaches, *cacheInfo)
				status.TotalSize += cacheInfo.SizeBytes
				if status.TotalFiles >= 0 && cacheInfo.Files >= 0 {
					status.TotalFiles += cacheInfo.Files
				} else {
					status.TotalFiles = -1 // Mark as approximate
				}

				// Add to version-specific tracking
				if _, exists := status.ByVersion[version]; !exists {
					status.ByVersion[version] = &VersionCaches{
						Version:     version,
						BuildCaches: make([]CacheInfo, 0),
					}
				}
				status.ByVersion[version].BuildCaches = append(status.ByVersion[version].BuildCaches, *cacheInfo)
				status.ByVersion[version].TotalSize += cacheInfo.SizeBytes
			}
		}

		// Check for module cache
		modCachePath := filepath.Join(versionPath, "pkg", "mod")
		if utils.DirExists(modCachePath) {
			cacheInfo, err := GetCacheInfo(modCachePath, CacheKindMod, fast)
			if err == nil {
				status.ModCaches = append(status.ModCaches, *cacheInfo)
				status.TotalSize += cacheInfo.SizeBytes
				if status.TotalFiles >= 0 && cacheInfo.Files >= 0 {
					status.TotalFiles += cacheInfo.Files
				} else {
					status.TotalFiles = -1 // Mark as approximate
				}

				// Add to version-specific tracking
				if _, exists := status.ByVersion[version]; !exists {
					status.ByVersion[version] = &VersionCaches{
						Version: version,
					}
				}
				status.ByVersion[version].ModCache = cacheInfo
				status.ByVersion[version].TotalSize += cacheInfo.SizeBytes
			}
		}
	}

	return status, nil
}

// GetVersionCaches returns all caches for a specific Go version.
//
// Parameters:
//   - goenvRoot: Path to GOENV_ROOT directory
//   - version: Go version (e.g., "1.23.2")
//   - fast: If true, skips file counting for better performance
//
// Returns list of cache info for the specified version.
func GetVersionCaches(goenvRoot, version string, fast bool) ([]CacheInfo, error) {
	caches := make([]CacheInfo, 0)
	versionPath := filepath.Join(goenvRoot, "versions", version)

	// Check if version exists
	if !utils.DirExists(versionPath) {
		return nil, fmt.Errorf("version %s is not installed", version)
	}

	// Get build caches
	buildCacheBase := filepath.Join(versionPath, "pkg")
	if buildEntries, err := os.ReadDir(buildCacheBase); err == nil {
		for _, buildEntry := range buildEntries {
			if !buildEntry.IsDir() || !strings.HasPrefix(buildEntry.Name(), "go-build") {
				continue
			}

			cachePath := filepath.Join(buildCacheBase, buildEntry.Name())
			cacheInfo, err := GetCacheInfo(cachePath, CacheKindBuild, fast)
			if err != nil {
				continue
			}
			caches = append(caches, *cacheInfo)
		}
	}

	// Get module cache
	modCachePath := filepath.Join(versionPath, "pkg", "mod")
	if utils.DirExists(modCachePath) {
		cacheInfo, err := GetCacheInfo(modCachePath, CacheKindMod, fast)
		if err == nil {
			caches = append(caches, *cacheInfo)
		}
	}

	return caches, nil
}

// detectCGOInfo attempts to detect CGO compiler information from a build cache.
// Returns nil if CGO info cannot be detected.
func detectCGOInfo(cachePath string) *CGOToolchainInfo {
	// Try to read build info from cache directory
	buildInfo, err := cgo.ReadBuildInfo(cachePath)
	if err != nil {
		// No build.info file or unable to read
		return nil
	}

	if buildInfo.CC == "" {
		// CGO not used for this cache
		return nil
	}

	// Parse flags from strings to slices
	var cflags, cxxflags, ldflags []string
	if buildInfo.CFLAGS != "" {
		cflags = strings.Fields(buildInfo.CFLAGS)
	}
	if buildInfo.CXXFLAGS != "" {
		cxxflags = strings.Fields(buildInfo.CXXFLAGS)
	}
	if buildInfo.LDFLAGS != "" {
		ldflags = strings.Fields(buildInfo.LDFLAGS)
	}

	return &CGOToolchainInfo{
		CC:       buildInfo.CC,
		CXX:      buildInfo.CXX,
		CFLAGS:   cflags,
		CXXFLAGS: cxxflags,
		LDFLAGS:  ldflags,
	}
}
