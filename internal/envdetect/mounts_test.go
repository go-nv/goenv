package envdetect

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

func TestDetectMountType(t *testing.T) {
	tmpDir := t.TempDir()

	info, err := DetectMountType(tmpDir)
	if err != nil {
		t.Fatalf("DetectMountType failed: %v", err)
	}

	if info == nil {
		t.Fatal("DetectMountType returned nil MountInfo")
	}

	if info.Path == "" {
		t.Error("MountInfo.Path should not be empty")
	}

	t.Logf("Detected mount info: Path=%s, Filesystem=%s, IsRemote=%v, IsDocker=%v",
		info.Path, info.Filesystem, info.IsRemote, info.IsDocker)
}

func TestDetectMountType_NonExistentPath(t *testing.T) {
	// Non-existent path should still work (uses abs path)
	info, err := DetectMountType("/nonexistent/path/testing")
	if err != nil {
		t.Fatalf("DetectMountType should handle non-existent paths: %v", err)
	}

	if info == nil {
		t.Fatal("DetectMountType returned nil MountInfo")
	}
}

func TestDetectLinuxMount(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux mount detection only works on Linux")
	}

	// Test with root directory
	info, err := detectLinuxMount("/")
	if err != nil {
		t.Fatalf("detectLinuxMount failed: %v", err)
	}

	if info == nil {
		t.Fatal("detectLinuxMount returned nil")
	}

	if info.Filesystem == "" {
		t.Error("Filesystem type should be detected for root")
	}

	t.Logf("Root filesystem: %s", info.Filesystem)
}

func TestDetectDarwinMount(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin mount detection only works on macOS")
	}

	// Test with a known path
	info, err := detectDarwinMount("/tmp")
	if err != nil {
		t.Fatalf("detectDarwinMount failed: %v", err)
	}

	if info == nil {
		t.Fatal("detectDarwinMount returned nil")
	}

	// Currently returns basic info
	t.Logf("Mount info for /tmp: %+v", info)
}

func TestCheckCacheOnProblemMount(t *testing.T) {
	tmpDir := t.TempDir()
	cachePath := filepath.Join(tmpDir, "cache")

	if err := os.MkdirAll(cachePath, 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}

	warning := CheckCacheOnProblemMount(cachePath)

	// On a normal temp directory, we shouldn't get a warning
	// (unless we're in a weird environment like CI with NFS mounts)
	if warning != "" {
		t.Logf("Received warning (may be expected in CI): %s", warning)
	}
}

func TestCheckCacheOnProblemMount_NonExistent(t *testing.T) {
	// Should handle non-existent paths gracefully
	warning := CheckCacheOnProblemMount("/nonexistent/cache/path")

	// May or may not warn depending on parent mount
	t.Logf("Warning for non-existent path: %q", warning)
}

func TestIsInContainer(t *testing.T) {
	if runtime.GOOS != "linux" {
		// Should return false on non-Linux
		result := IsInContainer()
		if result {
			t.Error("IsInContainer should return false on non-Linux systems")
		}
		return
	}

	// On Linux, check if we're actually in a container
	result := IsInContainer()

	// We can't reliably test this without being in a container,
	// but we can verify it doesn't crash
	t.Logf("IsInContainer returned: %v", result)

	// Check for common container indicators
	hasDockerEnv := false
	if _, err := os.Stat("/.dockerenv"); err == nil {
		hasDockerEnv = true
	}

	hasCgroup := false
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		cgroup := string(data)
		if len(cgroup) > 0 {
			hasCgroup = true
		}
	}

	t.Logf("Container indicators: dockerenv=%v, cgroup=%v", hasDockerEnv, hasCgroup)

	if hasDockerEnv && !result {
		t.Error("Should detect container when /.dockerenv exists")
	}
}

func TestMountInfo_Fields(t *testing.T) {
	info := &MountInfo{
		Path:       "/test/path",
		Filesystem: "ext4",
		IsRemote:   false,
		IsDocker:   false,
	}

	if info.Path != "/test/path" {
		t.Errorf("Path mismatch: got %s", info.Path)
	}

	if info.Filesystem != "ext4" {
		t.Errorf("Filesystem mismatch: got %s", info.Filesystem)
	}

	if info.IsRemote {
		t.Error("IsRemote should be false")
	}

	if info.IsDocker {
		t.Error("IsDocker should be false")
	}
}

func TestDetectMountType_Windows(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows-specific test")
	}

	// On Windows, should return basic info
	tmpDir := t.TempDir()
	info, err := DetectMountType(tmpDir)

	if err != nil {
		t.Fatalf("DetectMountType failed on Windows: %v", err)
	}

	if info == nil {
		t.Fatal("Should return basic MountInfo on Windows")
	}

	if info.Path == "" {
		t.Error("Path should be set")
	}
}
