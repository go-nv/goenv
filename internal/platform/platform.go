package platform

import (
	"github.com/go-nv/goenv/internal/osinfo"

	"github.com/go-nv/goenv/internal/binarycheck"
	"github.com/go-nv/goenv/internal/envdetect"
)

// Info contains comprehensive platform information
type Info struct {
	OS   string
	Arch string

	// Environment detection
	IsWSL       bool
	IsRosetta   bool
	IsContainer bool

	// OS-specific details
	MacOS   *binarycheck.PlatformInfo
	Windows *binarycheck.WindowsInfo
	Linux   *binarycheck.LinuxInfo
}

// Detect returns comprehensive platform information
func Detect() *Info {
	info := &Info{
		OS:          osinfo.OS(),
		Arch:        osinfo.Arch(),
		IsWSL:       envdetect.IsWSL(),
		IsRosetta:   envdetect.IsRosetta(),
		IsContainer: envdetect.IsInContainer(),
	}

	// Get OS-specific details
	platformInfo := binarycheck.GetPlatformInfo()
	if platformInfo != nil {
		switch osinfo.OS() {
		case "darwin":
			info.MacOS = platformInfo
		case "linux":
			linuxInfo, _ := binarycheck.CheckLinuxKernelVersion()
			info.Linux = linuxInfo
		case "windows":
			winInfo, _ := binarycheck.CheckWindowsCompiler()
			info.Windows = winInfo
		}
	}

	return info
}

// IsWindows returns true if running on Windows
func IsWindows() bool {
	return osinfo.OS() == "windows"
}

// IsMacOS returns true if running on macOS
func IsMacOS() bool {
	return osinfo.OS() == "darwin"
}

// IsLinux returns true if running on Linux
func IsLinux() bool {
	return osinfo.OS() == "linux"
}

// IsUnix returns true if running on a Unix-like system (macOS, Linux, BSD, etc.)
func IsUnix() bool {
	return osinfo.OS() != "windows"
}

// OSName returns a human-readable OS name
func OSName() string {
	switch osinfo.OS() {
	case "darwin":
		return "macOS"
	case "windows":
		return "Windows"
	case "linux":
		return "Linux"
	case "freebsd":
		return "FreeBSD"
	case "openbsd":
		return "OpenBSD"
	case "netbsd":
		return "NetBSD"
	case "dragonfly":
		return "DragonFly BSD"
	default:
		return osinfo.OS()
	}
}

// ArchName returns a human-readable architecture name
func ArchName() string {
	switch osinfo.Arch() {
	case "amd64":
		return "x86_64"
	case "386":
		return "x86"
	case "arm64":
		return "ARM64"
	case "arm":
		return "ARM"
	default:
		return osinfo.Arch()
	}
}

// OS returns the operating system (osinfo.OS()).
// Use this instead of osinfo.OS() to centralize platform detection.
func OS() string {
	return osinfo.OS()
}

// Arch returns the architecture (osinfo.Arch()).
// Use this instead of osinfo.Arch() to centralize platform detection.
func Arch() string {
	return osinfo.Arch()
}
