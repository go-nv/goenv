package pathutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// ExpandPath expands environment variables and tilde in a path
// Handles: $HOME, ${HOME}, %USERPROFILE%, ~/
// This is needed because Go's standard library doesn't expand shell metacharacters
func ExpandPath(path string) string {
	if path == "" {
		return path
	}

	// Expand environment variables first (e.g., $HOME, ${HOME}, %USERPROFILE%)
	path = os.ExpandEnv(path)

	// Expand tilde prefix (e.g., ~/go)
	if strings.HasPrefix(path, "~/") || path == "~" {
		if homeDir, err := os.UserHomeDir(); err == nil {
			if path == "~" {
				path = homeDir
			} else {
				path = filepath.Join(homeDir, path[2:])
			}
		}
	}

	return path
}

// FindExecutable finds an executable file, handling Windows executable extensions.
// On Windows, it checks for .exe, .bat, .cmd, and .com files.
// On Unix, it returns the path as-is.
// Returns the full path if the executable exists, or an error if not found.
func FindExecutable(basePath string) (string, error) {
	if !utils.IsWindows() {
		// On Unix, just check if the file exists
		if !utils.PathExists(basePath) {
			return "", os.ErrNotExist
		}
		return basePath, nil
	}

	// On Windows, try common executable extensions from utils
	for _, ext := range utils.WindowsExecutableExtensions() {
		extPath := basePath + ext
		if utils.PathExists(extPath) {
			return extPath, nil
		}
	}

	// None found, return error for .exe (expected in production)
	return "", os.ErrNotExist
}

// ResolveBinary searches for a binary in the standard goenv search order:
// 1. Version's bin directory (Go binaries like go, gofmt)
// 2. Host bin directory (tools installed with "goenv tools install")
// 3. Version-specific GOPATH bin (if GOPATH not disabled)
//
// Returns the full path to the binary if found, or an error if not found.
func ResolveBinary(command, versionBinDir, hostBinDir, versionGopathBin string, gopathDisabled bool) (string, error) {
	// Try version's bin directory first
	if path, err := utils.FindExecutable(versionBinDir, command); err == nil {
		return path, nil
	}

	// Try host bin directory (shared across versions)
	if path, err := utils.FindExecutable(hostBinDir, command); err == nil {
		return path, nil
	}

	// Try version-specific GOPATH bin if not disabled
	if !gopathDisabled {
		if path, err := utils.FindExecutable(versionGopathBin, command); err == nil {
			return path, nil
		}
	}

	return "", os.ErrNotExist
}
