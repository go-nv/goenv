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
