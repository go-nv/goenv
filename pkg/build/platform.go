package build

import (
	"fmt"
	"runtime"
	"strings"
)

// Platform represents the operating system and architecture information.
type Platform struct {
	OS        string // darwin, linux, freebsd, openbsd, etc.
	Arch      string // amd64, arm64, 386, arm, etc.
	OSVersion string // OS version string (e.g., "10.15" for macOS)
	CPUCores  int    // Number of CPU cores
}

// DetectPlatform detects the current platform's OS and architecture.
func DetectPlatform() (*Platform, error) {
	p := &Platform{
		OS:       runtime.GOOS,
		Arch:     runtime.GOARCH,
		CPUCores: runtime.NumCPU(),
	}

	// Normalize architecture names to match Go's download naming convention
	p.Arch = normalizeArch(p.Arch)

	return p, nil
}

// normalizeArch converts Go's GOARCH values to the naming convention used in download URLs.
func normalizeArch(arch string) string {
	switch arch {
	case "amd64":
		return "amd64"
	case "arm64":
		return "arm64"
	case "386":
		return "386"
	case "arm":
		return "armv6l"
	default:
		return arch
	}
}

// String returns a human-readable platform description.
func (p *Platform) String() string {
	return fmt.Sprintf("%s/%s", p.OS, p.Arch)
}

// IsSupported checks if the platform is supported for Go installations.
func (p *Platform) IsSupported() bool {
	supportedPlatforms := map[string][]string{
		"darwin":  {"amd64", "arm64"},
		"linux":   {"amd64", "arm64", "386", "armv6l", "ppc64le", "s390x"},
		"freebsd": {"amd64", "386"},
		"openbsd": {"amd64", "386"},
		"windows": {"amd64", "386"},
	}

	archs, ok := supportedPlatforms[p.OS]
	if !ok {
		return false
	}

	for _, arch := range archs {
		if arch == p.Arch {
			return true
		}
	}

	return false
}

// DownloadFilename returns the expected tarball filename for a given version.
func (p *Platform) DownloadFilename(version string) string {
	// Ensure version has "go" prefix
	if !strings.HasPrefix(version, "go") {
		version = "go" + version
	}

	ext := ".tar.gz"
	if p.OS == "windows" {
		ext = ".zip"
	}

	return fmt.Sprintf("%s.%s-%s%s", version, p.OS, p.Arch, ext)
}

// DownloadURL returns the official download URL for a given version.
func (p *Platform) DownloadURL(version string) string {
	filename := p.DownloadFilename(version)
	return fmt.Sprintf("https://go.dev/dl/%s", filename)
}

// MirrorURL returns a mirror download URL if mirror base URL is provided.
func (p *Platform) MirrorURL(version, mirrorBase string) string {
	if mirrorBase == "" {
		return p.DownloadURL(version)
	}

	// Remove trailing slash from mirror base
	mirrorBase = strings.TrimSuffix(mirrorBase, "/")
	filename := p.DownloadFilename(version)
	return fmt.Sprintf("%s/%s", mirrorBase, filename)
}
