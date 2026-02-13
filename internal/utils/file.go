package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// Common file permission constants
const (
	// File permissions
	PermFileDefault    = 0644 // rw-r--r-- (regular files)
	PermFileSecure     = 0600 // rw------- (secure files: configs, caches)
	PermFileExecutable = 0755 // rwxr-xr-x (executable files, scripts)

	// Directory permissions
	PermDirDefault = 0755 // rwxr-xr-x (regular directories)
	PermDirSecure  = 0700 // rwx------ (secure directories)

	// Special permissions
	PermExecutableBit = 0111 // --x--x--x (executable bit mask)
)

// FileExists checks if a file exists and is not a directory
func FileExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return !info.IsDir()
}

// DirExists checks if a directory exists
func DirExists(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// PathExists checks if a path exists (file or directory)
func PathExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// GetFileInfo returns file info and a boolean indicating if the path exists
// This is useful when you need both existence check and file info
func GetFileInfo(path string) (os.FileInfo, bool) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, false
	}
	return info, true
}

// CopyFile copies a file from src to dst, preserving file permissions
// This is useful for backup operations and file installations where
// preserving the original permissions is important (e.g., executable bits)
func CopyFile(src, dst string) error {
	// Open source file
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	// Create destination file
	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	// Copy contents
	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	// Copy permissions from source (platform-specific implementation)
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return copyFilePermissions(dst, srcInfo.Mode())
}

// CalculateDirectorySize calculates the total size of a directory and all its contents.
// Returns the total size in bytes.
func CalculateDirectorySize(path string) (int64, error) {
	var size int64
	err := filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if !info.IsDir() {
			size += info.Size()
		}
		return nil
	})
	return size, err
}

// FormatBytes formats a byte count into a human-readable string.
// Examples: 1024 → "1.0 KB", 1536 → "1.5 KB", 1048576 → "1.0 MB"
func FormatBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// IsPathInPATH checks if a given directory path exists in the PATH environment variable.
// On Windows, this performs case-insensitive comparison (matching default Windows behavior,
// even though NTFS can support case-sensitivity in specific configurations like WSL).
// On Unix, it's case-sensitive. It also normalizes path separators to handle both
// forward and backslashes on Windows.
func IsPathInPATH(dirPath, pathEnv string) bool {
	if pathEnv == "" || dirPath == "" {
		return false
	}

	// Normalize the directory path
	normalizedDir := filepath.Clean(dirPath)

	// Split PATH into individual directories
	paths := filepath.SplitList(pathEnv)

	for _, p := range paths {
		// Normalize each path entry
		normalizedPath := filepath.Clean(p)

		// Compare paths (case-insensitive on Windows, case-sensitive on Unix)
		if IsWindows() {
			if strings.EqualFold(normalizedPath, normalizedDir) {
				return true
			}
		} else {
			if normalizedPath == normalizedDir {
				return true
			}
		}
	}

	return false
}

// EnsureDir creates a directory and all parent directories if they don't exist.
// Equivalent to mkdir -p. Uses 0755 permissions.
func EnsureDir(path string) error {
	return os.MkdirAll(path, PermDirDefault)
}

// EnsureDirWithContext creates a directory and all parent directories with contextual error handling.
// This is a convenience wrapper around os.MkdirAll that provides consistent error messages
// using the errors.FailedTo() pattern. The context parameter describes what operation
// required the directory creation (e.g., "create cache directory", "create config directory").
//
// Example:
//
//	if err := EnsureDirWithContext(cacheDir, "create cache directory"); err != nil {
//	    return err
//	}
func EnsureDirWithContext(path string, context string) error {
	if err := os.MkdirAll(path, PermDirDefault); err != nil {
		return fmt.Errorf("failed to %s: %w", context, err)
	}
	return nil
}

// WriteFileWithContext writes data to a file with contextual error handling.
// This helper standardizes file writing across the codebase with consistent error messages.
func WriteFileWithContext(path string, data []byte, perm os.FileMode, context string) error {
	if err := os.WriteFile(path, data, perm); err != nil {
		return fmt.Errorf("failed to %s: %w", context, err)
	}
	return nil
}

// ReadFileWithContext reads a file with contextual error handling.
// This helper standardizes file reading across the codebase with consistent error messages.
// The wrapped error preserves os.IsNotExist checks via errors.Is().
func ReadFileWithContext(path string, context string) ([]byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to %s: %w", context, err)
	}
	return data, nil
}

// EnsureDirForFile creates the parent directory for a file path if it doesn't exist.
// Useful when creating files to ensure the directory structure is in place.
func EnsureDirForFile(filePath string) error {
	return os.MkdirAll(filepath.Dir(filePath), PermDirDefault)
}

// FileNotExists checks if a file does not exist.
// Returns true if the file doesn't exist, false if it exists or if there's a different error.
func FileNotExists(path string) bool {
	_, err := os.Stat(path)
	return os.IsNotExist(err)
}

// IsExecutableFile checks if a path exists, is a file (not directory), and is executable.
// On Windows, all files are considered executable if they exist.
// On Unix, checks for the executable bit in file permissions.
func IsExecutableFile(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}
	if info.IsDir() {
		return false
	}
	return IsWindows() || HasExecutableBit(info)
}

// StatWithExistence performs os.Stat and returns both the FileInfo and existence status.
// This is useful when you need the FileInfo but also want to distinguish between
// "doesn't exist" and other errors.
// Returns:
//   - info: FileInfo if stat succeeded, nil otherwise
//   - exists: true if path exists (stat succeeded)
//   - err: the error from os.Stat if it's not os.IsNotExist, nil otherwise
func StatWithExistence(path string) (info os.FileInfo, exists bool, err error) {
	info, err = os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, false, nil
		}
		return nil, false, err
	}
	return info, true, nil
}

// GetFileSize returns the size of a file in bytes.
// Returns 0 if the file doesn't exist or there's an error.
func GetFileSize(path string) int64 {
	info, err := os.Stat(path)
	if err != nil {
		return 0
	}
	return info.Size()
}

// GetFileModTime returns the modification time of a file.
// Returns zero time if the file doesn't exist or there's an error.
func GetFileModTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Time{}
	}
	return info.ModTime()
}

// GetFileSizeAndModTime returns both the size and modification time of a file.
// This is more efficient than calling GetFileSize and GetFileModTime separately
// since it only performs one os.Stat call.
// Returns (0, zero time) if the file doesn't exist or there's an error.
func GetFileSizeAndModTime(path string) (size int64, modTime time.Time) {
	info, err := os.Stat(path)
	if err != nil {
		return 0, time.Time{}
	}
	return info.Size(), info.ModTime()
}

// IsDir checks if a path exists and is a directory.
// Returns true only if the path exists and is a directory.
// This is an alias for DirExists() for consistency with filepath package naming.
func IsDir(path string) bool {
	return DirExists(path)
}

// IsFile checks if a path exists and is a regular file (not a directory).
// Returns true only if the path exists and is a file.
// This is an alias for FileExists() for consistency with filepath package naming.
func IsFile(path string) bool {
	return FileExists(path)
}
