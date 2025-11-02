package version

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
	}{
		{"go1.21.0", "go1.20.0", 1},
		{"go1.20.0", "go1.21.0", -1},
		{"go1.21.0", "go1.21.0", 0},
		{"1.21.0", "1.20.0", 1},
		{"1.21.5", "1.21.0", 1},
		{"1.21.0", "1.21.5", -1},
		{"1.21.0", "1.21.0rc1", 1},      // stable > rc
		{"1.21.0rc1", "1.21.0beta1", 1}, // rc > beta
	}

	for _, test := range tests {
		result := utils.CompareGoVersions(test.v1, test.v2)
		assert.Equal(t, test.expected, result, "utils.CompareGoVersions(, ) = , expected %v %v", test.v1, test.v2)
	}
}

func TestSortVersions(t *testing.T) {
	releases := []GoRelease{
		{Version: "go1.20.0", Stable: true},
		{Version: "go1.21.0", Stable: true},
		{Version: "go1.19.0", Stable: true},
		{Version: "go1.21.5", Stable: true},
	}

	SortVersions(releases)

	expected := []string{"go1.21.5", "go1.21.0", "go1.20.0", "go1.19.0"}
	for i, release := range releases {
		assert.Equal(t, expected[i], release.Version, "After sorting, position : got , expected")
	}
}

func TestGetFileForPlatform(t *testing.T) {
	var err error
	release := GoRelease{
		Version: "go1.21.0",
		Files: []GoFile{
			{Filename: "go1.21.0.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive"},
			{Filename: "go1.21.0.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Kind: "archive"},
			{Filename: "go1.21.0.windows-amd64.zip", OS: "windows", Arch: "amd64", Kind: "archive"},
		},
	}

	file, err := release.GetFileForPlatform("linux", "amd64")
	require.NoError(t, err, "Expected to find file for linux/amd64")

	assert.Equal(t, "go1.21.0.linux-amd64.tar.gz", file.Filename, "Expected linux-amd64 file")

	// Test non-existent platform
	_, err = release.GetFileForPlatform("nonexistent", "arch")
	assert.Error(t, err, "Expected error for non-existent platform, got nil")
}

// TestETagSupport tests ETag-based conditional requests
func TestETagSupport(t *testing.T) {
	callCount := 0
	etag := `"test-etag-123"`

	// Mock server that supports ETags
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		callCount++

		// Check for If-None-Match header
		ifNoneMatch := r.Header.Get("If-None-Match")

		if ifNoneMatch == etag {
			// Return 304 Not Modified
			w.WriteHeader(http.StatusNotModified)
			return
		}

		// Return full response with ETag
		w.Header().Set("ETag", etag)
		w.Header().Set("Content-Type", "application/json")
		releases := []GoRelease{
			{Version: "go1.21.0", Stable: true},
		}
		json.NewEncoder(w).Encode(releases)
	}))
	defer server.Close()

	fetcher := NewFetcher()
	fetcher.baseURL = server.URL + "/"

	// First request - should get full response
	releases1, etag1, err := fetcher.FetchAllReleasesWithETag("")
	require.NoError(t, err, "First fetch failed")
	assert.NotEqual(t, 0, len(releases1), "Expected releases, got empty slice")
	assert.Equal(t, etag, etag1, "Expected ETag")
	assert.Equal(t, 1, callCount, "Expected 1 server call")

	// Second request with ETag - should get 304
	releases2, etag2, err := fetcher.FetchAllReleasesWithETag(etag1)
	require.NoError(t, err, "Second fetch failed")
	assert.Nil(t, releases2, "Expected nil releases on 304, got data")
	assert.Empty(t, etag2, "Expected empty ETag on 304")
	assert.Equal(t, 2, callCount, "Expected 2 server calls")
}

// TestSHA256Verification tests cache integrity verification
func TestSHA256Verification(t *testing.T) {
	var err error
	tmpDir := t.TempDir()

	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
		{Version: "go1.20.0", Stable: true},
	}

	// Write cache with SHA256
	err = cache.Set(releases)
	require.NoError(t, err, "Failed to set cache")

	// Read cache - should verify SHA256
	cached, err := cache.Get()
	require.NoError(t, err, "Failed to get cache")

	assert.NotEmpty(t, cached.SHA256, "Expected SHA256 to be set")

	assert.Len(t, cached.Releases, len(releases), "Expected releases")

	// Now corrupt the cache by modifying the releases but keeping the old SHA256
	cachePath := filepath.Join(tmpDir, "releases-cache.json")
	data, _ := os.ReadFile(cachePath)
	var corrupted CachedData
	json.Unmarshal(data, &corrupted)

	// Add an extra release to corrupt the data
	corrupted.Releases = append(corrupted.Releases, GoRelease{Version: "go1.22.0", Stable: true})
	// Keep the old SHA256 - this should cause verification to fail

	corruptedData, _ := json.MarshalIndent(corrupted, "", "  ")
	testutil.WriteTestFile(t, cachePath, corruptedData, utils.PermFileSecure)

	// Try to read corrupted cache - should fail verification
	_, err = cache.Get()
	assert.Error(t, err, "Expected error on corrupted cache, got nil")
	assert.False(t, err != nil && err.Error() == "", "Expected non-empty error message")
}

// TestCachePermissions tests that cache files have secure permissions
func TestCachePermissions(t *testing.T) {
	var err error
	// Skip on Windows - permissions work differently
	if utils.IsWindows() {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
	}

	err = cache.Set(releases)
	require.NoError(t, err, "Failed to set cache")

	// Check file permissions (should be utils.PermFileSecure)
	cachePath := filepath.Join(tmpDir, "releases-cache.json")
	info, err := os.Stat(cachePath)
	require.NoError(t, err, "Failed to stat cache file")

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(utils.PermFileSecure)
	assert.Equal(t, expectedMode, mode, "Expected cache file mode")

	// Check directory permissions (should be utils.PermDirSecure)
	dirInfo, err := os.Stat(tmpDir)
	require.NoError(t, err, "Failed to stat cache dir")

	dirMode := dirInfo.Mode().Perm()
	expectedDirMode := os.FileMode(utils.PermDirSecure)
	assert.Equal(t, expectedDirMode, dirMode, "Expected cache dir mode")
}

// TestPermissionFixing tests that insecure permissions are auto-fixed
func TestPermissionFixing(t *testing.T) {
	var err error
	// Skip on Windows
	if utils.IsWindows() {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
	}

	// Write cache normally
	err = cache.Set(releases)
	require.NoError(t, err, "Failed to set cache")

	// Manually set insecure permissions
	cachePath := filepath.Join(tmpDir, "releases-cache.json")
	os.Chmod(cachePath, utils.PermFileDefault) // World-readable
	os.Chmod(tmpDir, utils.PermFileExecutable) // World-readable dir

	// Try to read - should auto-fix permissions
	_, err = cache.Get()
	require.NoError(t, err, "Failed to get cache after permission fix")

	// Verify permissions were fixed
	info, _ := os.Stat(cachePath)
	assert.Equal(t, os.FileMode(utils.PermFileSecure), info.Mode().Perm(), "Expected permissions to be fixed to utils.PermFileSecure")

	dirInfo, _ := os.Stat(tmpDir)
	assert.Equal(t, os.FileMode(utils.PermDirSecure), dirInfo.Mode().Perm(), "Expected dir permissions to be fixed to utils.PermDirSecure")
}

// TestCacheWithETag tests the full cache flow with ETag support
func TestCacheWithETag(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
	}
	etag := `"test-etag"`

	// Set cache with ETag
	err = cache.SetWithETag(releases, etag)
	require.NoError(t, err, "Failed to set cache with ETag")

	// Get cache and verify ETag is preserved
	cached, err := cache.Get()
	require.NoError(t, err, "Failed to get cache")

	assert.Equal(t, etag, cached.ETag, "Expected ETag")
}

// TestComputeSHA256 tests the SHA256 computation helper
func TestComputeSHA256(t *testing.T) {
	data1 := []byte("test data")
	data2 := []byte("test data")
	data3 := []byte("different data")

	hash1 := computeSHA256(data1)
	hash2 := computeSHA256(data2)
	hash3 := computeSHA256(data3)

	// Same data should produce same hash
	assert.Equal(t, hash2, hash1, "Same data produced different hashes")

	// Different data should produce different hash
	assert.NotEqual(t, hash3, hash1, "Different data produced same hash")

	// Hash should be 64 hex characters (SHA256 = 32 bytes = 64 hex chars)
	assert.Len(t, hash1, 64, "Expected hash length 64")
}

// TestBackgroundRefresh tests background cache refresh (basic smoke test)
func TestBackgroundRefresh(t *testing.T) {
	var err error
	// This is a smoke test - actual background behavior is hard to test
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
	}

	// Set initial cache
	err = cache.SetWithETag(releases, "etag1")
	require.NoError(t, err, "Failed to set cache")

	// Create a mock server for background refresh
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Return 304 for simplicity
		w.WriteHeader(http.StatusNotModified)
	}))
	defer server.Close()

	fetcher := NewFetcher()
	fetcher.baseURL = server.URL + "/"

	// Trigger background refresh (won't actually do anything useful in test)
	go fetcher.backgroundRefresh(cache, tmpDir)

	// Give it a moment to start (not wait for completion)
	time.Sleep(10 * time.Millisecond)

	// If we got here without panic, the basic mechanism works
}
