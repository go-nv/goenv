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
		if result != test.expected {
			t.Errorf("utils.CompareGoVersions(%s, %s) = %d, expected %d",
				test.v1, test.v2, result, test.expected)
		}
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
		if release.Version != expected[i] {
			t.Errorf("After sorting, position %d: got %s, expected %s",
				i, release.Version, expected[i])
		}
	}
}

func TestGetFileForPlatform(t *testing.T) {
	release := GoRelease{
		Version: "go1.21.0",
		Files: []GoFile{
			{Filename: "go1.21.0.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive"},
			{Filename: "go1.21.0.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Kind: "archive"},
			{Filename: "go1.21.0.windows-amd64.zip", OS: "windows", Arch: "amd64", Kind: "archive"},
		},
	}

	file, err := release.GetFileForPlatform("linux", "amd64")
	if err != nil {
		t.Fatalf("Expected to find file for linux/amd64, got error: %v", err)
	}

	if file.Filename != "go1.21.0.linux-amd64.tar.gz" {
		t.Errorf("Expected linux-amd64 file, got %s", file.Filename)
	}

	// Test non-existent platform
	_, err = release.GetFileForPlatform("nonexistent", "arch")
	if err == nil {
		t.Error("Expected error for non-existent platform, got nil")
	}
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
	if err != nil {
		t.Fatalf("First fetch failed: %v", err)
	}
	if len(releases1) == 0 {
		t.Error("Expected releases, got empty slice")
	}
	if etag1 != etag {
		t.Errorf("Expected ETag %s, got %s", etag, etag1)
	}
	if callCount != 1 {
		t.Errorf("Expected 1 server call, got %d", callCount)
	}

	// Second request with ETag - should get 304
	releases2, etag2, err := fetcher.FetchAllReleasesWithETag(etag1)
	if err != nil {
		t.Fatalf("Second fetch failed: %v", err)
	}
	if releases2 != nil {
		t.Error("Expected nil releases on 304, got data")
	}
	if etag2 != "" {
		t.Error("Expected empty ETag on 304")
	}
	if callCount != 2 {
		t.Errorf("Expected 2 server calls, got %d", callCount)
	}
}

// TestSHA256Verification tests cache integrity verification
func TestSHA256Verification(t *testing.T) {
	tmpDir := t.TempDir()

	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
		{Version: "go1.20.0", Stable: true},
	}

	// Write cache with SHA256
	if err := cache.Set(releases); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Read cache - should verify SHA256
	cached, err := cache.Get()
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if cached.SHA256 == "" {
		t.Error("Expected SHA256 to be set")
	}

	if len(cached.Releases) != len(releases) {
		t.Errorf("Expected %d releases, got %d", len(releases), len(cached.Releases))
	}

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
	if err == nil {
		t.Error("Expected error on corrupted cache, got nil")
	}
	if err != nil && err.Error() == "" {
		t.Error("Expected non-empty error message")
	}
}

// TestCachePermissions tests that cache files have secure permissions
func TestCachePermissions(t *testing.T) {
	// Skip on Windows - permissions work differently
	if utils.IsWindows() {
		t.Skip("Skipping permission test on Windows")
	}

	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
	}

	if err := cache.Set(releases); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Check file permissions (should be utils.PermFileSecure)
	cachePath := filepath.Join(tmpDir, "releases-cache.json")
	info, err := os.Stat(cachePath)
	if err != nil {
		t.Fatalf("Failed to stat cache file: %v", err)
	}

	mode := info.Mode().Perm()
	expectedMode := os.FileMode(utils.PermFileSecure)
	if mode != expectedMode {
		t.Errorf("Expected cache file mode %o, got %o", expectedMode, mode)
	}

	// Check directory permissions (should be utils.PermDirSecure)
	dirInfo, err := os.Stat(tmpDir)
	if err != nil {
		t.Fatalf("Failed to stat cache dir: %v", err)
	}

	dirMode := dirInfo.Mode().Perm()
	expectedDirMode := os.FileMode(utils.PermDirSecure)
	if dirMode != expectedDirMode {
		t.Errorf("Expected cache dir mode %o, got %o", expectedDirMode, dirMode)
	}
}

// TestPermissionFixing tests that insecure permissions are auto-fixed
func TestPermissionFixing(t *testing.T) {
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
	if err := cache.Set(releases); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

	// Manually set insecure permissions
	cachePath := filepath.Join(tmpDir, "releases-cache.json")
	os.Chmod(cachePath, utils.PermFileDefault) // World-readable
	os.Chmod(tmpDir, utils.PermFileExecutable) // World-readable dir

	// Try to read - should auto-fix permissions
	_, err := cache.Get()
	if err != nil {
		t.Fatalf("Failed to get cache after permission fix: %v", err)
	}

	// Verify permissions were fixed
	info, _ := os.Stat(cachePath)
	if info.Mode().Perm() != utils.PermFileSecure {
		t.Errorf("Expected permissions to be fixed to utils.PermFileSecure, got %o", info.Mode().Perm())
	}

	dirInfo, _ := os.Stat(tmpDir)
	if dirInfo.Mode().Perm() != utils.PermDirSecure {
		t.Errorf("Expected dir permissions to be fixed to utils.PermDirSecure, got %o", dirInfo.Mode().Perm())
	}
}

// TestCacheWithETag tests the full cache flow with ETag support
func TestCacheWithETag(t *testing.T) {
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
	}
	etag := `"test-etag"`

	// Set cache with ETag
	if err := cache.SetWithETag(releases, etag); err != nil {
		t.Fatalf("Failed to set cache with ETag: %v", err)
	}

	// Get cache and verify ETag is preserved
	cached, err := cache.Get()
	if err != nil {
		t.Fatalf("Failed to get cache: %v", err)
	}

	if cached.ETag != etag {
		t.Errorf("Expected ETag %s, got %s", etag, cached.ETag)
	}
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
	if hash1 != hash2 {
		t.Error("Same data produced different hashes")
	}

	// Different data should produce different hash
	if hash1 == hash3 {
		t.Error("Different data produced same hash")
	}

	// Hash should be 64 hex characters (SHA256 = 32 bytes = 64 hex chars)
	if len(hash1) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash1))
	}
}

// TestBackgroundRefresh tests background cache refresh (basic smoke test)
func TestBackgroundRefresh(t *testing.T) {
	// This is a smoke test - actual background behavior is hard to test
	tmpDir := t.TempDir()
	cache := NewCache(tmpDir)

	releases := []GoRelease{
		{Version: "go1.21.0", Stable: true},
	}

	// Set initial cache
	if err := cache.SetWithETag(releases, "etag1"); err != nil {
		t.Fatalf("Failed to set cache: %v", err)
	}

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
