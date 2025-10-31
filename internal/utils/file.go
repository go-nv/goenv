package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
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
