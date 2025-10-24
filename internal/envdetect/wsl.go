package envdetect

import (
	"os"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// IsWSL detects if we're running in Windows Subsystem for Linux
func IsWSL() bool {
	if runtime.GOOS != "linux" {
		return false
	}

	// Check /proc/version for Microsoft/WSL indicators
	if data, err := os.ReadFile("/proc/version"); err == nil {
		version := strings.ToLower(string(data))
		return strings.Contains(version, "microsoft") || strings.Contains(version, "wsl")
	}

	// Check /proc/sys/kernel/osrelease
	if data, err := os.ReadFile("/proc/sys/kernel/osrelease"); err == nil {
		osrelease := strings.ToLower(string(data))
		return strings.Contains(osrelease, "microsoft") || strings.Contains(osrelease, "wsl")
	}

	return false
}

// IsWindowsBinary checks if a binary is a Windows PE executable
func IsWindowsBinary(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()

	// Read magic bytes
	magic := make([]byte, 2)
	if _, err := file.Read(magic); err != nil {
		return false
	}

	// Check for PE signature (MZ)
	return magic[0] == 'M' && magic[1] == 'Z'
}

// CheckWSLCrossExecution checks if we're trying to run incompatible binaries in WSL
// Returns a warning message if there's a potential issue, empty string otherwise
func CheckWSLCrossExecution(binaryPath string) string {
	if !IsWSL() {
		return ""
	}

	// Check if trying to run a Windows binary in WSL
	if IsWindowsBinary(binaryPath) {
		// Detect actual host architecture for correct rebuild command
		hostArch := runtime.GOARCH
		return utils.Emoji("⚠️  ") + "Running Windows binary in WSL. This may work via Windows interop but could have issues.\n" +
			"   Consider rebuilding for Linux: GOOS=linux GOARCH=" + hostArch + " go install <package>@latest"
	}

	// Could also check for Linux binaries that might not work well in WSL,
	// but that's harder to detect and less common

	return ""
}
