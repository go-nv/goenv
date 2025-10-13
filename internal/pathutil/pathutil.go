package pathutil

import (
	"os"
	"path/filepath"
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
