//go:build windows
// +build windows

package utils

import (
	"os"
)

// copyFilePermissions is a no-op on Windows.
// Windows file permissions work through ACLs rather than Unix-style permission bits.
// Executable status is determined by file extension (.exe, .bat, .cmd, .com).
// The os.Chmod function on Windows only handles the read-only bit and ignores
// executable and group/other permissions.
func copyFilePermissions(_ string, _ os.FileMode) error {
	// On Windows, we don't need to (and can't effectively) copy Unix-style permissions.
	// Files inherit ACLs from their parent directory, which is the expected behavior.
	// Executability is determined by file extension, not permission bits.
	return nil
}
