package envdetect

import (
	"bufio"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// MountInfo represents information about a filesystem mount
type MountInfo struct {
	Path       string
	Filesystem string
	IsRemote   bool
	IsDocker   bool
}

// DetectMountType checks if a path is on a networked/container filesystem
func DetectMountType(path string) (*MountInfo, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return nil, err
	}

	if runtime.GOOS == "linux" {
		return detectLinuxMount(absPath)
	} else if runtime.GOOS == "darwin" {
		return detectDarwinMount(absPath)
	}

	// Windows and others - basic detection
	return &MountInfo{Path: absPath}, nil
}

// detectLinuxMount parses /proc/mounts to detect mount types
func detectLinuxMount(path string) (*MountInfo, error) {
	file, err := os.Open("/proc/mounts")
	if err != nil {
		return &MountInfo{Path: path}, nil // Can't read mounts, assume local
	}
	defer file.Close()

	info := &MountInfo{Path: path}
	longestMatch := ""

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		mountPoint := fields[1]
		fsType := fields[2]

		// Check if path is under this mount point
		if strings.HasPrefix(path, mountPoint) && len(mountPoint) > len(longestMatch) {
			longestMatch = mountPoint
			info.Filesystem = fsType

			// Detect remote filesystems
			switch fsType {
			case "nfs", "nfs4", "cifs", "smb", "smbfs":
				info.IsRemote = true
			case "overlay", "aufs":
				// Docker/container overlay
				info.IsDocker = true
			}

			// Check mount options for bind mounts
			if len(fields) >= 4 {
				options := fields[3]
				if strings.Contains(options, "bind") {
					info.IsDocker = true
				}
			}
		}
	}

	return info, nil
}

// detectDarwinMount uses df to detect mount types on macOS
func detectDarwinMount(path string) (*MountInfo, error) {
	info := &MountInfo{Path: path}

	// Use df to get filesystem type
	// This is a simplified version - full implementation would exec df
	// For now, just return basic info
	return info, nil
}

// CheckCacheOnProblemMount checks if a cache directory is on a filesystem
// that might cause issues with file locking or performance
func CheckCacheOnProblemMount(cachePath string) string {
	info, err := DetectMountType(cachePath)
	if err != nil {
		return ""
	}

	if info.IsRemote {
		return utils.Emoji("⚠️  ") + "Cache directory is on a remote/networked filesystem (" + info.Filesystem + ").\n" +
			"   This may cause:\n" +
			"   - File locking issues (cache corruption)\n" +
			"   - Slow build performance\n" +
			"   - Stale cache problems\n" +
			"   Consider: Using a local cache directory with GOENV_GOCACHE_DIR"
	}

	if info.IsDocker && strings.Contains(cachePath, "/goenv") {
		return utils.Emoji("⚠️  ") + "Cache directory appears to be in a Docker bind mount.\n" +
			"   Consider: Using a Docker volume instead of bind mount for better performance:\n" +
			"   docker run -v goenv-cache:/root/.goenv ..."
	}

	return ""
}

// IsInContainer detects if we're running inside a container
func IsInContainer() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check for /.dockerenv file
	if _, err := os.Stat("/.dockerenv"); err == nil {
		return true
	}

	// Check cgroup for docker/kubepods
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		cgroup := string(data)
		return strings.Contains(cgroup, "docker") ||
			strings.Contains(cgroup, "kubepods") ||
			strings.Contains(cgroup, "containerd")
	}

	return false
}
