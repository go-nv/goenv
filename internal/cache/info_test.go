package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCacheKindString(t *testing.T) {
	tests := []struct {
		kind     CacheKind
		expected string
	}{
		{CacheKindBuild, "build"},
		{CacheKindMod, "mod"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.kind.String()
			assert.Equal(t, tt.expected, result, "CacheKind.String() =")
		})
	}
}

func TestDetectCacheKind(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected CacheKind
		wantErr  bool
	}{
		{
			name:     "build cache old format",
			path:     "/path/to/go-build",
			expected: CacheKindBuild,
			wantErr:  false,
		},
		{
			name:     "build cache with platform",
			path:     "/path/to/go-build-darwin-arm64",
			expected: CacheKindBuild,
			wantErr:  false,
		},
		{
			name:     "build cache with ABI",
			path:     "/path/to/go-build-linux-amd64-v3",
			expected: CacheKindBuild,
			wantErr:  false,
		},
		{
			name:     "module cache unix",
			path:     "/path/to/pkg/mod",
			expected: CacheKindMod,
			wantErr:  false,
		},
		{
			name:     "module cache windows",
			path:     "C:\\path\\to\\pkg\\mod",
			expected: CacheKindMod,
			wantErr:  false,
		},
		{
			name:     "unknown cache",
			path:     "/path/to/something-else",
			expected: "",
			wantErr:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := DetectCacheKind(tt.path)
			if tt.wantErr {
				assert.Error(t, err, "DetectCacheKind() expected error")
				return
			}
			assert.NoError(t, err, "DetectCacheKind() unexpected error")
			assert.Equal(t, tt.expected, result, "DetectCacheKind() = %v", tt.path)
		})
	}
}

func TestGetCacheInfo(t *testing.T) {
	var err error
	// Create a temporary test cache directory
	tempDir := t.TempDir()
	goenvRoot := filepath.Join(tempDir, "goenv")
	versionPath := filepath.Join(goenvRoot, "versions", "1.23.2", "pkg")
	err = utils.EnsureDirWithContext(versionPath, "create test directory")
	require.NoError(t, err, "Failed to create test directory")

	tests := []struct {
		name          string
		cacheName     string
		kind          CacheKind
		setupFunc     func(string)
		fast          bool
		expectOldFmt  bool
		expectTarget  bool
		expectVersion string
	}{
		{
			name:      "old format build cache",
			cacheName: "go-build",
			kind:      CacheKindBuild,
			setupFunc: func(path string) {
				_ = utils.EnsureDirWithContext(path, "create test directory")
			},
			fast:          false,
			expectOldFmt:  true,
			expectTarget:  false,
			expectVersion: "1.23.2",
		},
		{
			name:      "new format build cache",
			cacheName: "go-build-darwin-arm64",
			kind:      CacheKindBuild,
			setupFunc: func(path string) {
				// Create cache dir with a test file
				err = utils.EnsureDirWithContext(path, "create test directory")
				require.NoError(t, err)
				testFile := filepath.Join(path, "test.cache")
				testutil.WriteTestFile(t, testFile, []byte("test data"), utils.PermFileDefault)
			},
			fast:          false,
			expectOldFmt:  false,
			expectTarget:  true,
			expectVersion: "1.23.2",
		},
		{
			name:      "build cache with ABI",
			cacheName: "go-build-linux-amd64-v3",
			kind:      CacheKindBuild,
			setupFunc: func(path string) {
				_ = utils.EnsureDirWithContext(path, "create test directory")
			},
			fast:          true, // Test fast mode
			expectOldFmt:  false,
			expectTarget:  true,
			expectVersion: "1.23.2",
		},
		{
			name:      "module cache",
			cacheName: "mod",
			kind:      CacheKindMod,
			setupFunc: func(path string) {
				// Create cache dir with nested structure
				err = utils.EnsureDirWithContext(path, "create test directory")
				require.NoError(t, err)
				testFile := filepath.Join(path, "cache", "test.mod")
				err = utils.EnsureDirWithContext(filepath.Dir(testFile), "create test directory")
				require.NoError(t, err)
				testutil.WriteTestFile(t, testFile, []byte("module test"), utils.PermFileDefault)
			},
			fast:          false,
			expectOldFmt:  false,
			expectTarget:  false, // Mod caches don't have target info
			expectVersion: "1.23.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cachePath := filepath.Join(versionPath, tt.cacheName)

			// Setup test cache
			tt.setupFunc(cachePath)

			// Get cache info
			info, err := GetCacheInfo(cachePath, tt.kind, tt.fast)
			require.NoError(t, err, "GetCacheInfo() error")

			// Verify basic fields
			assert.Equal(t, tt.kind, info.Kind, "Kind =")
			assert.Equal(t, cachePath, info.Path, "Path =")
			assert.Equal(t, tt.expectVersion, info.GoVersion, "GoVersion =")

			// Verify old format detection
			assert.Equal(t, tt.expectOldFmt, info.OldFormat, "OldFormat =")

			// Verify target info
			if tt.expectTarget {
				if info.Target == nil {
					t.Error("Expected Target info, got nil")
				} else {
					assert.NotEmpty(t, info.Target.GOOS, "Expected GOOS to be set")
					assert.NotEmpty(t, info.Target.GOARCH, "Expected GOARCH to be set")
				}
			} else {
				assert.False(t, tt.kind == CacheKindBuild && info.Target != nil && !info.OldFormat, "Expected Target to be nil for mod cache or old format")
			}

			// Verify size is calculated
			if info.SizeBytes < 0 {
				t.Errorf("SizeBytes should be non-negative, got %d", info.SizeBytes)
			}

			// Verify file count behavior
			if tt.fast {
				assert.Equal(t, -1, info.Files, "Fast mode should return Files=-1")
			}
		})
	}
}

func TestGetCacheInfoNonExistent(t *testing.T) {
	_, err := GetCacheInfo("/non/existent/path", CacheKindBuild, false)
	assert.Error(t, err, "GetCacheInfo() with non-existent path should return error")
}

func TestGetCacheStatus(t *testing.T) {
	var err error
	// Create a temporary test goenv structure
	tempDir := t.TempDir()
	goenvRoot := tempDir

	// Setup multiple versions with different cache configurations
	versions := []struct {
		version     string
		buildCache  string
		hasModCache bool
	}{
		{"1.22.0", "go-build", true},                 // Old format + mod
		{"1.23.0", "go-build-darwin-arm64", true},    // New format + mod
		{"1.23.2", "go-build-linux-amd64-v3", false}, // New format, no mod
	}

	for _, v := range versions {
		versionPath := filepath.Join(goenvRoot, "versions", v.version)
		err = utils.EnsureDirWithContext(versionPath, "create test directory")
		require.NoError(t, err, "Failed to create version directory")

		// Create build cache directly in the version directory — this is where
		// `goenv exec` points GOCACHE, and therefore where the scanner must look.
		buildPath := filepath.Join(versionPath, v.buildCache)
		err = utils.EnsureDirWithContext(buildPath, "create test directory")
		require.NoError(t, err, "Failed to create build cache")
		// Add a test file
		testFile := filepath.Join(buildPath, "test.o")
		testutil.WriteTestFile(t, testFile, []byte("compiled object"), utils.PermFileDefault, "Failed to write test file")

		// Create mod cache if specified (legacy per-version location under pkg/).
		if v.hasModCache {
			modPath := filepath.Join(versionPath, "pkg", "mod")
			err = utils.EnsureDirWithContext(modPath, "create test directory")
			require.NoError(t, err, "Failed to create mod cache")
			modFile := filepath.Join(modPath, "test.mod")
			testutil.WriteTestFile(t, modFile, []byte("module cache"), utils.PermFileDefault, "Failed to write mod file")
		}
	}

	// Get cache status
	status, err := GetCacheStatus(goenvRoot, false)
	require.NoError(t, err, "GetCacheStatus() error")

	// Verify build caches
	assert.Len(t, status.BuildCaches, 3, "Expected 3 build caches")

	// Verify mod caches
	assert.Len(t, status.ModCaches, 2, "Expected 2 mod caches")

	// Verify total size is positive
	if status.TotalSize <= 0 {
		t.Errorf("Expected positive TotalSize, got %d", status.TotalSize)
	}

	// Verify version grouping
	assert.Len(t, status.ByVersion, 3, "Expected 3 versions in ByVersion")

	// Verify specific version has correct caches
	if v122, exists := status.ByVersion["1.22.0"]; exists {
		assert.Len(t, v122.BuildCaches, 1, "Version 1.22.0 should have 1 build cache")
		assert.NotNil(t, v122.ModCache, "Version 1.22.0 should have mod cache")
		assert.Equal(t, true, v122.BuildCaches[0].OldFormat, "Version 1.22.0 build cache should be old format")
	} else {
		t.Error("Expected version 1.22.0 in ByVersion map")
	}

	if v123, exists := status.ByVersion["1.23.2"]; exists {
		assert.Nil(t, v123.ModCache, "Version 1.23.2 should not have mod cache")
		assert.Len(t, v123.BuildCaches, 1, "Version 1.23.2 should have 1 build cache")
	}
}

func TestGetCacheStatusEmpty(t *testing.T) {
	// Test with empty goenv root
	tempDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarGocacheDir.String(), "") // hermetic: ignore any ambient override

	status, err := GetCacheStatus(tempDir, false)
	require.NoError(t, err, "GetCacheStatus() with empty root should not error")

	assert.Len(t, status.BuildCaches, 0, "Expected 0 build caches")
	assert.Len(t, status.ModCaches, 0, "Expected 0 mod caches")
	assert.Equal(t, int64(0), status.TotalSize, "Expected TotalSize=0")
}

// TestGetCacheStatus_FindsArchitectureSuffixedCacheInVersionDir is a regression
// test for the bug where `goenv cache status/clean/info` reported "No caches
// found" while gigabytes of build cache sat on disk.
//
// `goenv exec` sets GOCACHE to versions/{version}/go-build-{GOOS}-{GOARCH}[-cgo]
// — directly in the version directory. The scanner previously looked under
// versions/{version}/pkg/ and only recognised a bare "go-build" in the version
// dir, so every real (architecture-suffixed) cache was invisible. The tests at
// the time seeded the pkg/ layout, so they passed while the tool was broken.
func TestGetCacheStatus_FindsArchitectureSuffixedCacheInVersionDir(t *testing.T) {
	root := t.TempDir()

	// Reproduce the exact on-disk name that triggered the report.
	cacheDir := filepath.Join(root, "versions", "1.27.0", "go-build-host-host-cgo")
	require.NoError(t, utils.EnsureDirWithContext(cacheDir, "create test cache"))
	testutil.WriteTestFile(t, filepath.Join(cacheDir, "a.o"), []byte("compiled object"), utils.PermFileDefault)

	status, err := GetCacheStatus(root, false)
	require.NoError(t, err, "GetCacheStatus() error")

	require.Len(t, status.BuildCaches, 1, "architecture-suffixed build cache in the version dir must be found")
	assert.Equal(t, "1.27.0", status.BuildCaches[0].GoVersion, "GoVersion")
	assert.False(t, status.BuildCaches[0].OldFormat, "suffixed cache is not old format")
	assert.Greater(t, status.TotalSize, int64(0), "TotalSize must include the discovered cache")
	require.Contains(t, status.ByVersion, "1.27.0", "cache must be attributed to its version")
}

// TestGetCacheStatus_FindsCachesUnderGoenvGocacheDir is a regression test for
// the GOENV_GOCACHE_DIR override. `goenv exec` points GOCACHE at
// <custom>/<version>/go-build-* when that variable is set — entirely off the
// GOENV_ROOT tree. If the scanner only looks under <root>/versions, those
// caches are invisible to status/clean/info (and leaked on uninstall): the same
// writer/reader path divergence that hid the architecture-suffixed caches, just
// triggered by a documented env override instead of the default layout.
func TestGetCacheStatus_FindsCachesUnderGoenvGocacheDir(t *testing.T) {
	root := t.TempDir()

	// Deliberately empty under root/versions; everything lives off-root.
	custom := t.TempDir()
	t.Setenv(utils.GoenvEnvVarGocacheDir.String(), custom)

	cacheDir := filepath.Join(custom, "1.27.0", "go-build-host-host-cgo")
	require.NoError(t, utils.EnsureDirWithContext(cacheDir, "create custom cache"))
	testutil.WriteTestFile(t, filepath.Join(cacheDir, "a.o"), []byte("compiled object"), utils.PermFileDefault)

	status, err := GetCacheStatus(root, false)
	require.NoError(t, err, "GetCacheStatus() error")

	require.Len(t, status.BuildCaches, 1, "build cache under GOENV_GOCACHE_DIR must be found")
	assert.Equal(t, "1.27.0", status.BuildCaches[0].GoVersion, "off-root cache must be attributed to its version")
	assert.Greater(t, status.TotalSize, int64(0), "TotalSize must include the off-root cache")
	require.Contains(t, status.ByVersion, "1.27.0", "off-root cache must be grouped under its version")
}

// TestGetCacheStatus_DeduplicatesCustomDirEqualToVersions guards against
// double-counting when GOENV_GOCACHE_DIR resolves to $GOENV_ROOT/versions (the
// default base). Without dedup the same directory is scanned twice, doubling
// reported sizes and making clean try to remove every cache twice.
func TestGetCacheStatus_DeduplicatesCustomDirEqualToVersions(t *testing.T) {
	root := t.TempDir()

	cacheDir := filepath.Join(root, "versions", "1.27.0", "go-build-host-host-cgo")
	require.NoError(t, utils.EnsureDirWithContext(cacheDir, "create cache"))
	testutil.WriteTestFile(t, filepath.Join(cacheDir, "a.o"), []byte("compiled object"), utils.PermFileDefault)

	// Point the override at the default versions dir; it must NOT be scanned twice.
	t.Setenv(utils.GoenvEnvVarGocacheDir.String(), filepath.Join(root, "versions"))

	status, err := GetCacheStatus(root, false)
	require.NoError(t, err, "GetCacheStatus() error")

	require.Len(t, status.BuildCaches, 1, "cache under a dir equal to versions/ must be counted once, not twice")
	require.Contains(t, status.ByVersion, "1.27.0")
	assert.Len(t, status.ByVersion["1.27.0"].BuildCaches, 1, "no duplicate build-cache entries for the version")
}

func TestGetVersionCaches(t *testing.T) {
	var err error
	// Create a temporary test structure
	tempDir := t.TempDir()
	goenvRoot := tempDir
	version := "1.23.2"

	versionPath := filepath.Join(goenvRoot, "versions", version)
	err = utils.EnsureDirWithContext(versionPath, "create test directory")
	require.NoError(t, err, "Failed to create version directory")

	// Create multiple build caches directly in the version directory.
	buildCaches := []string{"go-build", "go-build-darwin-arm64", "go-build-linux-amd64"}
	for _, cacheName := range buildCaches {
		cachePath := filepath.Join(versionPath, cacheName)
		err = utils.EnsureDirWithContext(cachePath, "create test directory")
		require.NoError(t, err, "Failed to create cache")
		// Add test file
		testFile := filepath.Join(cachePath, "test.o")
		testutil.WriteTestFile(t, testFile, []byte("test"), utils.PermFileDefault, "Failed to write test file")
	}

	// Create mod cache (legacy per-version location under pkg/).
	modPath := filepath.Join(versionPath, "pkg", "mod")
	err = utils.EnsureDirWithContext(modPath, "create test directory")
	require.NoError(t, err, "Failed to create mod cache")

	// Get caches
	caches, err := GetVersionCaches(goenvRoot, version, false)
	require.NoError(t, err, "GetVersionCaches() error")

	// Should have 3 build + 1 mod = 4 total
	assert.Len(t, caches, 4, "Expected 4 caches")

	// Count by kind
	buildCount := 0
	modCount := 0
	for _, cache := range caches {
		switch cache.Kind {
		case CacheKindBuild:
			buildCount++
		case CacheKindMod:
			modCount++
		}
	}

	assert.Equal(t, 3, buildCount, "Expected 3 build caches")
	assert.Equal(t, 1, modCount, "Expected 1 mod cache")
}

func TestGetVersionCachesNonExistent(t *testing.T) {
	tempDir := t.TempDir()

	_, err := GetVersionCaches(tempDir, "99.99.99", false)
	assert.Error(t, err, "GetVersionCaches() with non-existent version should return error")
}

func TestTargetInfo(t *testing.T) {
	// Test TargetInfo struct initialization
	target := &TargetInfo{
		GOOS:   "linux",
		GOARCH: "amd64",
		ABI: map[string]string{
			"GOAMD64": "v3",
		},
	}

	assert.Equal(t, "linux", target.GOOS, "GOOS =")
	assert.Equal(t, "amd64", target.GOARCH, "GOARCH =")
	assert.Equal(t, "v3", target.ABI["GOAMD64"], "ABI[GOAMD64] =")
}

func TestCGOToolchainInfo(t *testing.T) {
	// Test CGOToolchainInfo struct initialization
	cgoInfo := &CGOToolchainInfo{
		CC:       "/usr/bin/gcc",
		CXX:      "/usr/bin/g++",
		CFLAGS:   []string{"-O2", "-Wall"},
		CXXFLAGS: []string{"-O2", "-std=c++17"},
		LDFLAGS:  []string{"-lpthread"},
	}

	assert.Equal(t, "/usr/bin/gcc", cgoInfo.CC, "CC =")
	assert.Len(t, cgoInfo.CFLAGS, 2, "Expected 2 CFLAGS")
}

func TestCacheInfoModTime(t *testing.T) {
	var err error
	// Create test cache and verify ModTime is set
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "goenv", "versions", "1.23.2", "pkg", "go-build")
	err = utils.EnsureDirWithContext(cachePath, "create test directory")
	require.NoError(t, err, "Failed to create cache")

	// Write a file and wait a tiny bit to ensure time difference
	testFile := filepath.Join(cachePath, "test")
	testutil.WriteTestFile(t, testFile, []byte("test"), utils.PermFileDefault, "Failed to write test file")

	time.Sleep(10 * time.Millisecond)

	info, err := GetCacheInfo(cachePath, CacheKindBuild, false)
	require.NoError(t, err, "GetCacheInfo() error")

	if info.ModTime.IsZero() {
		t.Error("Expected ModTime to be set, got zero time")
	}

	// ModTime should be recent (within last minute)
	if time.Since(info.ModTime) > time.Minute {
		t.Errorf("ModTime seems too old: %v", info.ModTime)
	}
}
