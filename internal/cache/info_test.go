package cache

import (
	"path/filepath"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
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
			if result != tt.expected {
				t.Errorf("CacheKind.String() = %q, want %q", result, tt.expected)
			}
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
				if err == nil {
					t.Errorf("DetectCacheKind(%q) expected error, got nil", tt.path)
				}
				return
			}
			if err != nil {
				t.Errorf("DetectCacheKind(%q) unexpected error: %v", tt.path, err)
				return
			}
			if result != tt.expected {
				t.Errorf("DetectCacheKind(%q) = %q, want %q", tt.path, result, tt.expected)
			}
		})
	}
}

func TestGetCacheInfo(t *testing.T) {
	// Create a temporary test cache directory
	tempDir := t.TempDir()
	goenvRoot := filepath.Join(tempDir, "goenv")
	versionPath := filepath.Join(goenvRoot, "versions", "1.23.2", "pkg")
	if err := utils.EnsureDirWithContext(versionPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create test directory: %v", err)
	}

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
				if err := utils.EnsureDirWithContext(path, "create test directory"); err != nil {
					t.Fatal(err)
				}
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
				if err := utils.EnsureDirWithContext(path, "create test directory"); err != nil {
					t.Fatal(err)
				}
				testFile := filepath.Join(path, "cache", "test.mod")
				if err := utils.EnsureDirWithContext(filepath.Dir(testFile), "create test directory"); err != nil {
					t.Fatal(err)
				}
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
			if err != nil {
				t.Fatalf("GetCacheInfo() error: %v", err)
			}

			// Verify basic fields
			if info.Kind != tt.kind {
				t.Errorf("Kind = %v, want %v", info.Kind, tt.kind)
			}
			if info.Path != cachePath {
				t.Errorf("Path = %q, want %q", info.Path, cachePath)
			}
			if info.GoVersion != tt.expectVersion {
				t.Errorf("GoVersion = %q, want %q", info.GoVersion, tt.expectVersion)
			}

			// Verify old format detection
			if info.OldFormat != tt.expectOldFmt {
				t.Errorf("OldFormat = %v, want %v", info.OldFormat, tt.expectOldFmt)
			}

			// Verify target info
			if tt.expectTarget {
				if info.Target == nil {
					t.Error("Expected Target info, got nil")
				} else {
					if info.Target.GOOS == "" {
						t.Error("Expected GOOS to be set")
					}
					if info.Target.GOARCH == "" {
						t.Error("Expected GOARCH to be set")
					}
				}
			} else {
				if tt.kind == CacheKindBuild && info.Target != nil && !info.OldFormat {
					t.Errorf("Expected Target to be nil for mod cache or old format, got %+v", info.Target)
				}
			}

			// Verify size is calculated
			if info.SizeBytes < 0 {
				t.Errorf("SizeBytes should be non-negative, got %d", info.SizeBytes)
			}

			// Verify file count behavior
			if tt.fast {
				if info.Files != -1 {
					t.Errorf("Fast mode should return Files=-1, got %d", info.Files)
				}
			}
		})
	}
}

func TestGetCacheInfoNonExistent(t *testing.T) {
	_, err := GetCacheInfo("/non/existent/path", CacheKindBuild, false)
	if err == nil {
		t.Error("GetCacheInfo() with non-existent path should return error")
	}
}

func TestGetCacheStatus(t *testing.T) {
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
		versionPath := filepath.Join(goenvRoot, "versions", v.version, "pkg")
		if err := utils.EnsureDirWithContext(versionPath, "create test directory"); err != nil {
			t.Fatalf("Failed to create version directory: %v", err)
		}

		// Create build cache
		buildPath := filepath.Join(versionPath, v.buildCache)
		if err := utils.EnsureDirWithContext(buildPath, "create test directory"); err != nil {
			t.Fatalf("Failed to create build cache: %v", err)
		}
		// Add a test file
		testFile := filepath.Join(buildPath, "test.o")
		testutil.WriteTestFile(t, testFile, []byte("compiled object"), utils.PermFileDefault, "Failed to write test file")

		// Create mod cache if specified
		if v.hasModCache {
			modPath := filepath.Join(versionPath, "mod")
			if err := utils.EnsureDirWithContext(modPath, "create test directory"); err != nil {
				t.Fatalf("Failed to create mod cache: %v", err)
			}
			modFile := filepath.Join(modPath, "test.mod")
			testutil.WriteTestFile(t, modFile, []byte("module cache"), utils.PermFileDefault, "Failed to write mod file")
		}
	}

	// Get cache status
	status, err := GetCacheStatus(goenvRoot, false)
	if err != nil {
		t.Fatalf("GetCacheStatus() error: %v", err)
	}

	// Verify build caches
	if len(status.BuildCaches) != 3 {
		t.Errorf("Expected 3 build caches, got %d", len(status.BuildCaches))
	}

	// Verify mod caches
	if len(status.ModCaches) != 2 {
		t.Errorf("Expected 2 mod caches, got %d", len(status.ModCaches))
	}

	// Verify total size is positive
	if status.TotalSize <= 0 {
		t.Errorf("Expected positive TotalSize, got %d", status.TotalSize)
	}

	// Verify version grouping
	if len(status.ByVersion) != 3 {
		t.Errorf("Expected 3 versions in ByVersion, got %d", len(status.ByVersion))
	}

	// Verify specific version has correct caches
	if v122, exists := status.ByVersion["1.22.0"]; exists {
		if len(v122.BuildCaches) != 1 {
			t.Errorf("Version 1.22.0 should have 1 build cache, got %d", len(v122.BuildCaches))
		}
		if v122.ModCache == nil {
			t.Error("Version 1.22.0 should have mod cache")
		}
		if v122.BuildCaches[0].OldFormat != true {
			t.Error("Version 1.22.0 build cache should be old format")
		}
	} else {
		t.Error("Expected version 1.22.0 in ByVersion map")
	}

	if v123, exists := status.ByVersion["1.23.2"]; exists {
		if v123.ModCache != nil {
			t.Error("Version 1.23.2 should not have mod cache")
		}
		if len(v123.BuildCaches) != 1 {
			t.Errorf("Version 1.23.2 should have 1 build cache, got %d", len(v123.BuildCaches))
		}
	}
}

func TestGetCacheStatusEmpty(t *testing.T) {
	// Test with empty goenv root
	tempDir := t.TempDir()

	status, err := GetCacheStatus(tempDir, false)
	if err != nil {
		t.Fatalf("GetCacheStatus() with empty root should not error: %v", err)
	}

	if len(status.BuildCaches) != 0 {
		t.Errorf("Expected 0 build caches, got %d", len(status.BuildCaches))
	}
	if len(status.ModCaches) != 0 {
		t.Errorf("Expected 0 mod caches, got %d", len(status.ModCaches))
	}
	if status.TotalSize != 0 {
		t.Errorf("Expected TotalSize=0, got %d", status.TotalSize)
	}
}

func TestGetVersionCaches(t *testing.T) {
	// Create a temporary test structure
	tempDir := t.TempDir()
	goenvRoot := tempDir
	version := "1.23.2"

	versionPath := filepath.Join(goenvRoot, "versions", version, "pkg")
	if err := utils.EnsureDirWithContext(versionPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}

	// Create multiple build caches
	buildCaches := []string{"go-build", "go-build-darwin-arm64", "go-build-linux-amd64"}
	for _, cacheName := range buildCaches {
		cachePath := filepath.Join(versionPath, cacheName)
		if err := utils.EnsureDirWithContext(cachePath, "create test directory"); err != nil {
			t.Fatalf("Failed to create cache: %v", err)
		}
		// Add test file
		testFile := filepath.Join(cachePath, "test.o")
		testutil.WriteTestFile(t, testFile, []byte("test"), utils.PermFileDefault, "Failed to write test file")
	}

	// Create mod cache
	modPath := filepath.Join(versionPath, "mod")
	if err := utils.EnsureDirWithContext(modPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create mod cache: %v", err)
	}

	// Get caches
	caches, err := GetVersionCaches(goenvRoot, version, false)
	if err != nil {
		t.Fatalf("GetVersionCaches() error: %v", err)
	}

	// Should have 3 build + 1 mod = 4 total
	if len(caches) != 4 {
		t.Errorf("Expected 4 caches, got %d", len(caches))
	}

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

	if buildCount != 3 {
		t.Errorf("Expected 3 build caches, got %d", buildCount)
	}
	if modCount != 1 {
		t.Errorf("Expected 1 mod cache, got %d", modCount)
	}
}

func TestGetVersionCachesNonExistent(t *testing.T) {
	tempDir := t.TempDir()

	_, err := GetVersionCaches(tempDir, "99.99.99", false)
	if err == nil {
		t.Error("GetVersionCaches() with non-existent version should return error")
	}
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

	if target.GOOS != "linux" {
		t.Errorf("GOOS = %q, want %q", target.GOOS, "linux")
	}
	if target.GOARCH != "amd64" {
		t.Errorf("GOARCH = %q, want %q", target.GOARCH, "amd64")
	}
	if target.ABI["GOAMD64"] != "v3" {
		t.Errorf("ABI[GOAMD64] = %q, want %q", target.ABI["GOAMD64"], "v3")
	}
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

	if cgoInfo.CC != "/usr/bin/gcc" {
		t.Errorf("CC = %q, want %q", cgoInfo.CC, "/usr/bin/gcc")
	}
	if len(cgoInfo.CFLAGS) != 2 {
		t.Errorf("Expected 2 CFLAGS, got %d", len(cgoInfo.CFLAGS))
	}
}

func TestCacheInfoModTime(t *testing.T) {
	// Create test cache and verify ModTime is set
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "goenv", "versions", "1.23.2", "pkg", "go-build")
	if err := utils.EnsureDirWithContext(cachePath, "create test directory"); err != nil {
		t.Fatalf("Failed to create cache: %v", err)
	}

	// Write a file and wait a tiny bit to ensure time difference
	testFile := filepath.Join(cachePath, "test")
	testutil.WriteTestFile(t, testFile, []byte("test"), utils.PermFileDefault, "Failed to write test file")

	time.Sleep(10 * time.Millisecond)

	info, err := GetCacheInfo(cachePath, CacheKindBuild, false)
	if err != nil {
		t.Fatalf("GetCacheInfo() error: %v", err)
	}

	if info.ModTime.IsZero() {
		t.Error("Expected ModTime to be set, got zero time")
	}

	// ModTime should be recent (within last minute)
	if time.Since(info.ModTime) > time.Minute {
		t.Errorf("ModTime seems too old: %v", info.ModTime)
	}
}
