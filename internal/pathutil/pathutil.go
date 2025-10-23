package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
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
// On Windows, it checks for both .exe (production) and .bat (tests) files.
// On Unix, it returns the path as-is.
// Returns the full path if the executable exists, or an error if not found.
func FindExecutable(basePath string) (string, error) {
	if runtime.GOOS != "windows" {
		// On Unix, just check if the file exists
		if _, err := os.Stat(basePath); err != nil {
			return "", err
		}
		return basePath, nil
	}

	// On Windows, try .exe first (production), then .bat (tests)
	exePath := basePath + ".exe"
	if _, err := os.Stat(exePath); err == nil {
		return exePath, nil
	}

	batPath := basePath + ".bat"
	if _, err := os.Stat(batPath); err == nil {
		return batPath, nil
	}

	// Neither exists, return error for .exe (expected in production)
	_, err := os.Stat(exePath)
	return "", err
}
