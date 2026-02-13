package envdetect

import (
	"fmt"
	"github.com/go-nv/goenv/internal/osinfo"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// EnvironmentType represents the type of environment goenv is running in
type EnvironmentType string

const (
	EnvTypeNative    EnvironmentType = "native"
	EnvTypeContainer EnvironmentType = "container"
	EnvTypeWSL       EnvironmentType = "wsl"
)

// FilesystemType represents the type of filesystem
type FilesystemType string

const (
	FSTypeLocal   FilesystemType = "local"
	FSTypeNFS     FilesystemType = "nfs"
	FSTypeSMB     FilesystemType = "smb"
	FSTypeBind    FilesystemType = "bind"
	FSTypeFUSE    FilesystemType = "fuse"
	FSTypeUnknown FilesystemType = "unknown"
)

// EnvironmentInfo contains information about the runtime environment
type EnvironmentInfo struct {
	Type           EnvironmentType
	IsContainer    bool
	IsWSL          bool
	ContainerType  string // docker, podman, kubernetes, etc.
	WSLVersion     string // "1" or "2"
	WSLDistro      string
	FilesystemType FilesystemType
	FilesystemPath string
	Warnings       []string
}

// Detect detects the current runtime environment
func Detect() *EnvironmentInfo {
	info := &EnvironmentInfo{
		Type:     EnvTypeNative,
		Warnings: make([]string, 0),
	}

	// Check for WSL
	if osinfo.IsLinux() {
		if wslVersion, distro := detectWSL(); wslVersion != "" {
			info.IsWSL = true
			info.Type = EnvTypeWSL
			info.WSLVersion = wslVersion
			info.WSLDistro = distro

			// Add WSL-specific warnings
			if wslVersion == "1" {
				info.Warnings = append(info.Warnings, "WSL 1 detected: Performance may be slower than WSL 2, especially for I/O operations")
			}
		}
	}

	// Check for container environments
	if containerType := detectContainer(); containerType != "" {
		info.IsContainer = true
		info.Type = EnvTypeContainer
		info.ContainerType = containerType

		// Add container-specific warnings
		info.Warnings = append(info.Warnings, fmt.Sprintf("Running in %s container: Ensure volumes are properly mounted", containerType))
	}

	return info
}

// DetectFilesystem detects the filesystem type for a given path
func DetectFilesystem(path string) *EnvironmentInfo {
	info := Detect()
	info.FilesystemPath = path

	if osinfo.IsLinux() {
		info.FilesystemType = detectLinuxFilesystem(path)
	} else if osinfo.IsMacOS() {
		info.FilesystemType = detectDarwinFilesystem(path)
	} else if osinfo.IsWindows() {
		info.FilesystemType = detectWindowsFilesystem(path)
	} else {
		info.FilesystemType = FSTypeLocal
	}

	// Add filesystem-specific warnings
	switch info.FilesystemType {
	case FSTypeNFS:
		info.Warnings = append(info.Warnings, "NFS filesystem detected: File locking and permissions may behave differently. Build caches may have issues.")
	case FSTypeSMB:
		info.Warnings = append(info.Warnings, "SMB/CIFS filesystem detected: Symbolic links and file permissions may not work correctly")
	case FSTypeBind:
		info.Warnings = append(info.Warnings, "Bind mount detected: Ensure the mount has correct permissions and is persistent")
	case FSTypeFUSE:
		info.Warnings = append(info.Warnings, "FUSE filesystem detected: Performance may be impacted. Consider using a local filesystem for better performance.")
	}

	return info
}

// detectWSL checks if running in Windows Subsystem for Linux
func detectWSL() (version string, distro string) {
	// Check /proc/version for WSL
	if data, err := os.ReadFile("/proc/version"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			// Determine WSL version
			// WSL 2 uses a real Linux kernel, WSL 1 has "Microsoft" in the kernel version
			if strings.Contains(content, "wsl2") {
				version = "2"
			} else if strings.Contains(content, "microsoft-standard") {
				version = "2"
			} else {
				version = "1"
			}

			// Try to get distro name
			if data, err := os.ReadFile("/etc/os-release"); err == nil {
				lines := strings.Split(string(data), "\n")
				for _, line := range lines {
					if strings.HasPrefix(line, "NAME=") {
						distro = strings.Trim(strings.TrimPrefix(line, "NAME="), "\"")
						break
					}
				}
			}

			return version, distro
		}
	}

	// Alternative: Check for WSL_DISTRO_NAME environment variable
	if distroName := os.Getenv(utils.EnvVarWSLDistroName); distroName != "" {
		// WSL_DISTRO_NAME is set in WSL 2
		return "2", distroName
	}

	// Check /proc/sys/kernel/osrelease for WSL
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "microsoft") || strings.Contains(content, "wsl") {
			return "2", "" // Likely WSL 2
		}
	}

	return "", ""
}

// detectContainer checks if running in a container
func detectContainer() string {
	// Check for /.dockerenv file (Docker)
	if utils.PathExists("/.dockerenv") {
		return "docker"
	}

	// Check cgroup for container indicators
	if data, err := os.ReadFile("/proc/1/cgroup"); err == nil {
		content := strings.ToLower(string(data))
		if strings.Contains(content, "docker") {
			return "docker"
		}
		if strings.Contains(content, "kubepods") || strings.Contains(content, "kube") {
			return "kubernetes"
		}
		if strings.Contains(content, "podman") {
			return "podman"
		}
		if strings.Contains(content, "lxc") {
			return "lxc"
		}
	}

	// Check for container environment variables
	if os.Getenv(utils.EnvVarKubernetesServiceHost) != "" {
		return "kubernetes"
	}
	if os.Getenv(utils.EnvVarContainer) == "podman" {
		return "podman"
	}
	if os.Getenv(utils.EnvVarContainerUpper) == "podman" {
		return "podman"
	}

	// Check /run/.containerenv (Podman)
	if utils.PathExists("/run/.containerenv") {
		return "podman"
	}

	// Check for buildkit
	if os.Getenv(utils.EnvVarBuildkitSandboxHostname) != "" {
		return "buildkit"
	}

	return ""
}

// detectLinuxFilesystem detects filesystem type on Linux
func detectLinuxFilesystem(path string) FilesystemType {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return FSTypeUnknown
	}

	// Read /proc/mounts
	data, err := os.ReadFile("/proc/mounts")
	if err != nil {
		return FSTypeUnknown
	}

	// Parse mounts to find the mount point for our path
	lines := strings.Split(string(data), "\n")
	var bestMatchLen int
	var bestMatchType string

	for _, line := range lines {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			continue
		}

		mountPoint := fields[1]
		fsType := fields[2]

		// Check if path is under this mount point
		if strings.HasPrefix(absPath, mountPoint) && len(mountPoint) > bestMatchLen {
			bestMatchLen = len(mountPoint)
			bestMatchType = fsType
		}
	}

	if bestMatchType == "" {
		return FSTypeLocal
	}

	// Classify filesystem type
	fsType := strings.ToLower(bestMatchType)
	switch {
	case fsType == "nfs" || fsType == "nfs4":
		return FSTypeNFS
	case fsType == "cifs" || fsType == "smb" || fsType == "smbfs":
		return FSTypeSMB
	case strings.HasPrefix(fsType, "fuse"):
		return FSTypeFUSE
	case fsType == "overlay" || fsType == "aufs":
		return FSTypeBind
	case fsType == "ext4" || fsType == "ext3" || fsType == "xfs" || fsType == "btrfs":
		return FSTypeLocal
	default:
		return FSTypeUnknown
	}
}

// detectDarwinFilesystem detects filesystem type on macOS
func detectDarwinFilesystem(path string) FilesystemType {
	// Get absolute path
	absPath, err := filepath.Abs(path)
	if err != nil {
		return FSTypeUnknown
	}

	// Common indicators on macOS:
	// - /Volumes/ prefix often indicates external or network volumes
	// - NFS mounts typically show up in mount output

	if strings.HasPrefix(absPath, "/Volumes/") {
		// Could be network mount or external drive
		// This is a heuristic - more detailed checking would require parsing mount output
		return FSTypeUnknown
	}

	// Default to local for macOS unless we have specific indicators
	return FSTypeLocal
}

// detectWindowsFilesystem detects filesystem type on Windows
func detectWindowsFilesystem(path string) FilesystemType {
	// Check for UNC paths BEFORE normalizing (they start with \\ or //)
	if strings.HasPrefix(path, "\\\\") || strings.HasPrefix(path, "//") {
		return FSTypeSMB
	}

	// Check for WSL paths BEFORE normalizing (filepath.Abs on Windows converts Unix paths)
	if strings.HasPrefix(path, "/mnt/") {
		// This is a WSL mount of a Windows drive
		// These are typically much slower than native Linux filesystem
		return FSTypeFUSE // FUSE-like performance characteristics
	}

	// On Windows, we need to check for:
	// - UNC paths (\\server\share) - SMB/CIFS (checked above)
	// - Mapped network drives (future: could check drive letters)

	// Default to local for Windows drives
	return FSTypeLocal
}

// IsProblematicEnvironment checks if the environment may cause issues
func (info *EnvironmentInfo) IsProblematicEnvironment() bool {
	return len(info.Warnings) > 0
}

// GetWarnings returns all warnings about the environment
func (info *EnvironmentInfo) GetWarnings() []string {
	return info.Warnings
}

// String returns a human-readable description of the environment
func (info *EnvironmentInfo) String() string {
	var parts []string

	switch info.Type {
	case EnvTypeContainer:
		parts = append(parts, fmt.Sprintf("Container (%s)", info.ContainerType))
	case EnvTypeWSL:
		if info.WSLDistro != "" {
			parts = append(parts, fmt.Sprintf("WSL %s (%s)", info.WSLVersion, info.WSLDistro))
		} else {
			parts = append(parts, fmt.Sprintf("WSL %s", info.WSLVersion))
		}
	default:
		parts = append(parts, "Native")
	}

	if info.FilesystemType != "" && info.FilesystemType != FSTypeLocal {
		parts = append(parts, fmt.Sprintf("Filesystem: %s", info.FilesystemType))
	}

	return strings.Join(parts, ", ")
}
