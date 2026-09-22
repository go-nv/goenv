// Package cache provides utilities and operations for managing Go build and module caches.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/cgo"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/pathutil"
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

// SharedCacheLabel is the value placed in CacheInfo.GoVersion for the module
// cache that is shared across all installed Go versions.
//
// The shared cache belongs to no single version, so it is deliberately absent
// from CacheStatus.ByVersion. Anything that renders cache information must
// handle it separately or it will be silently omitted while still being
// counted in totals (issue #578).
const SharedCacheLabel = "shared"

// CacheStatus contains aggregate cache statistics for all Go versions.
type CacheStatus struct {
	BuildCaches []CacheInfo               // All build caches
	ModCaches   []CacheInfo               // All module caches
	TotalSize   int64                     // Total size of all caches
	TotalFiles  int                       // Total number of files (-1 if any cache timed out)
	ByVersion   map[string]*VersionCaches // Caches grouped by Go version (excludes the shared cache)
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

// buildCacheBaseDirs returns every directory that may contain per-version build
// caches laid out as <dir>/<version>/go-build-*.
//
// That is always <goenvRoot>/versions. When GOENV_GOCACHE_DIR is set it is ALSO
// that directory: `goenv exec` points GOCACHE at
// <GOENV_GOCACHE_DIR>/<version>/go-build-* (see cmd/shims/exec.go), entirely
// off-root. The reader MUST consult the same override the writer does, or caches
// created under a custom dir are invisible to status/clean/info and are leaked
// on uninstall — the same writer/reader path divergence that hid the
// architecture-suffixed caches.
func buildCacheBaseDirs(goenvRoot string) []string {
	versionsDir := filepath.Join(goenvRoot, "versions")
	dirs := []string{versionsDir}
	// Only add the override when it is set AND distinct from versions/. If a user
	// points GOENV_GOCACHE_DIR at $GOENV_ROOT/versions (or it otherwise resolves
	// to the same place), scanning it twice would double-count sizes and try to
	// remove every cache twice.
	if custom := CustomBuildCacheDir(); custom != "" && filepath.Clean(custom) != filepath.Clean(versionsDir) {
		dirs = append(dirs, custom)
	}
	return dirs
}

// CustomBuildCacheDir returns the expanded GOENV_GOCACHE_DIR — the off-root base
// where `goenv exec` places build caches when that variable is set — or "" when
// unset. It is the single source of truth for readers (the status/clean no-work
// guards and the scanner) that must honour the same override the writer does.
func CustomBuildCacheDir() string {
	return pathutil.ExpandPath(utils.GoenvEnvVarGocacheDir.UnsafeValue())
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

	// Build caches live under <base>/<version>/go-build-*. That base is normally
	// <goenvRoot>/versions, but GOENV_GOCACHE_DIR relocates it off-root and
	// `goenv exec` honours that override — so the scanner has to read the SAME
	// override or every cache under a custom dir is invisible (see
	// buildCacheBaseDirs).
	//
	// A missing base directory is not a reason to stop: the shared module cache
	// lives outside these bases and outlives every version, so returning here
	// would report zero bytes while gigabytes sit on disk (issue #578).
	for _, base := range buildCacheBaseDirs(goenvRoot) {
		if !utils.DirExists(base) {
			continue
		}
		entries, err := os.ReadDir(base)
		if err != nil {
			return nil, errors.FailedTo("read cache directory", err)
		}
		if err := collectVersionCaches(status, base, entries, fast); err != nil {
			return nil, err
		}
	}

	// Check for shared module cache (v3+). Deliberately outside the versions
	// walk above — see the comment there.
	sharedModCachePath := config.SharedModCacheDir(goenvRoot)
	if utils.DirExists(sharedModCachePath) {
		cacheInfo, err := GetCacheInfo(sharedModCachePath, CacheKindMod, fast)
		if err == nil {
			cacheInfo.GoVersion = SharedCacheLabel // Mark as shared across versions
			status.ModCaches = append(status.ModCaches, *cacheInfo)
			status.TotalSize += cacheInfo.SizeBytes
			if status.TotalFiles >= 0 && cacheInfo.Files >= 0 {
				status.TotalFiles += cacheInfo.Files
			} else {
				status.TotalFiles = -1 // Mark as approximate
			}
		}
	}

	return status, nil
}

// collectVersionCaches walks each version subdirectory of baseDir and records
// its build and module caches into status. baseDir is <goenvRoot>/versions or a
// GOENV_GOCACHE_DIR override (see buildCacheBaseDirs).
func collectVersionCaches(status *CacheStatus, baseDir string, entries []os.DirEntry, fast bool) error {
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		version := entry.Name()
		versionPath := filepath.Join(baseDir, version)

		// Build caches live DIRECTLY in the version directory, named
		// go-build-{GOOS}-{GOARCH}[-cgo][-abi] (or the legacy bare "go-build").
		// That is exactly where `goenv exec` points GOCACHE, so it is the only
		// place a build cache can appear.
		//
		// A previous implementation scanned versionPath/pkg/ for the new format
		// and only recognised a bare "go-build" directly in the version dir.
		// Every architecture-suffixed cache (e.g. go-build-host-host-cgo) fell
		// through both checks, so status/clean/info reported "No caches found"
		// while gigabytes of build cache sat on disk. Scan the version directory
		// itself, which covers both the legacy and architecture-aware names.
		if buildEntries, err := os.ReadDir(versionPath); err == nil {
			for _, buildEntry := range buildEntries {
				if !buildEntry.IsDir() || !strings.HasPrefix(buildEntry.Name(), "go-build") {
					continue
				}

				cachePath := filepath.Join(versionPath, buildEntry.Name())
				cacheInfo, err := GetCacheInfo(cachePath, CacheKindBuild, fast)
				if err != nil {
					// Skip caches we can't read
					continue
				}
				// GetCacheInfo derives the version from a ".../versions/<v>/..."
				// path segment, which is absent when the cache lives under a custom
				// GOENV_GOCACHE_DIR. Attribute it explicitly so off-root caches are
				// grouped under the right version rather than "".
				cacheInfo.GoVersion = version

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

	return nil
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

	// Get build caches. These live directly in the version directory
	// (go-build-{GOOS}-{GOARCH}[-cgo] or the legacy bare "go-build") — the same
	// location `goenv exec` sets GOCACHE to. See collectVersionCaches.
	if buildEntries, err := os.ReadDir(versionPath); err == nil {
		for _, buildEntry := range buildEntries {
			if !buildEntry.IsDir() || !strings.HasPrefix(buildEntry.Name(), "go-build") {
				continue
			}

			cachePath := filepath.Join(versionPath, buildEntry.Name())
			cacheInfo, err := GetCacheInfo(cachePath, CacheKindBuild, fast)
			if err != nil {
				continue
			}
			caches = append(caches, *cacheInfo)
		}
	}

	// Also scan any GOENV_GOCACHE_DIR override, where `goenv exec` may have
	// written this version's build caches (<custom>/<version>/go-build-*).
	if custom := pathutil.ExpandPath(utils.GoenvEnvVarGocacheDir.UnsafeValue()); custom != "" {
		customVersionPath := filepath.Join(custom, version)
		if buildEntries, err := os.ReadDir(customVersionPath); err == nil {
			for _, buildEntry := range buildEntries {
				if !buildEntry.IsDir() || !strings.HasPrefix(buildEntry.Name(), "go-build") {
					continue
				}
				cachePath := filepath.Join(customVersionPath, buildEntry.Name())
				cacheInfo, err := GetCacheInfo(cachePath, CacheKindBuild, fast)
				if err != nil {
					continue
				}
				cacheInfo.GoVersion = version
				caches = append(caches, *cacheInfo)
			}
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
