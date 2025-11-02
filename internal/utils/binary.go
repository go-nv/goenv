package utils

import (
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/osinfo"
)

// IsWindows returns true if the current OS is Windows.
// Note: platform.IsWindows() provides the same functionality with other OS checks
// (IsMacOS, IsLinux, IsUnix). Both implementations are kept to avoid import cycles.
func IsWindows() bool {
	return osinfo.IsWindows()
}

// WindowsExecutableExtensions returns the list of valid executable extensions on Windows
func WindowsExecutableExtensions() []string {
	return []string{".exe", ".bat", ".cmd", ".com"}
}

// HasExecutableBit returns true if the file has the executable bit set (Unix permission 0111).
// This is only meaningful on Unix-like systems. On Windows, this always returns true.
func HasExecutableBit(info os.FileInfo) bool {
	return info.Mode()&PermExecutableBit != 0
}

// FindExecutable looks for an executable file in the given directory.
// On Windows, it tries all valid executable extensions (.exe, .bat, .cmd, .com).
// On Unix, it looks for a file with the exact name that has executable permissions.
//
// Returns the full path to the executable if found, or an error if not found.
func FindExecutable(dir, name string) (string, error) {
	if IsWindows() {
		// On Windows, try all executable extensions in order of preference
		for _, ext := range WindowsExecutableExtensions() {
			path := filepath.Join(dir, name+ext)
			if FileExists(path) {
				return path, nil
			}
		}
		return "", os.ErrNotExist
	}

	// On Unix, check for exact name with executable bit
	path := filepath.Join(dir, name)
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !HasExecutableBit(info) {
		return "", os.ErrPermission
	}
	return path, nil
}

// IsExecutable checks if a file is executable on the current platform.
// On Windows, it checks if the file has a valid executable extension.
// On Unix, it checks if the file has the executable bit set.
func IsExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	if IsWindows() {
		// On Windows, check file extension
		ext := filepath.Ext(path)
		for _, validExt := range WindowsExecutableExtensions() {
			if ext == validExt {
				return true
			}
		}
		return false
	}

	// On Unix, check execute bit
	return HasExecutableBit(info)
}

// HasExecutableExtension checks if a filename has a valid executable extension.
// On Windows, checks for .exe, .bat, .cmd, or .com.
// On Unix, returns true (Unix doesn't require extensions).
func HasExecutableExtension(filename string) bool {
	if IsWindows() {
		ext := filepath.Ext(filename)
		for _, validExt := range WindowsExecutableExtensions() {
			if ext == validExt {
				return true
			}
		}
		return false
	}
	// On Unix, any filename is valid
	return true
}
