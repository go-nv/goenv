//go:build !windows
// +build !windows

package utils

import (
	"os"
)

// copyFilePermissions sets file permissions on Unix systems.
// On Unix, this preserves the full permission mode including executable bits.
func copyFilePermissions(path string, mode os.FileMode) error {
	return os.Chmod(path, mode)
}
