//go:build !windows
// +build !windows

package version

import (
	"fmt"
	"os"
	"path/filepath"
)

// checkPermissions checks if cache file has secure permissions (0600) on Unix systems
func (c *Cache) checkPermissions() error {
	info, err := os.Stat(c.cachePath)
	if err != nil {
		return err
	}

	mode := info.Mode().Perm()
	// Check if file is readable/writable by others
	if mode&0077 != 0 {
		return fmt.Errorf("cache file has insecure permissions: %o (should be 0600)", mode)
	}

	// Check cache directory permissions (should be 0700)
	cacheDir := filepath.Dir(c.cachePath)
	dirInfo, err := os.Stat(cacheDir)
	if err != nil {
		return err
	}

	dirMode := dirInfo.Mode().Perm()
	if dirMode&0077 != 0 {
		return fmt.Errorf("cache directory has insecure permissions: %o (should be 0700)", dirMode)
	}

	return nil
}

// ensureSecurePermissions fixes cache file and directory permissions on Unix systems
func (c *Cache) ensureSecurePermissions() error {
	// Fix file permissions
	if err := os.Chmod(c.cachePath, 0600); err != nil {
		return fmt.Errorf("failed to fix cache file permissions: %w", err)
	}

	// Fix directory permissions
	cacheDir := filepath.Dir(c.cachePath)
	if err := os.Chmod(cacheDir, 0700); err != nil {
		return fmt.Errorf("failed to fix cache directory permissions: %w", err)
	}

	return nil
}
