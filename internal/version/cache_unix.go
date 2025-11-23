//go:build !windows
// +build !windows

package version

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// checkPermissions checks if cache file has secure permissions (utils.PermFileSecure) on Unix systems
func (c *Cache) checkPermissions() error {
	info, err := os.Stat(c.cachePath)
	if err != nil {
		return err
	}

	mode := info.Mode().Perm()
	// Check if file is readable/writable by others
	if mode&0077 != 0 {
		return fmt.Errorf("cache file has insecure permissions: %o (should be utils.PermFileSecure)", mode)
	}

	// Check cache directory permissions (should be utils.PermDirSecure)
	cacheDir := filepath.Dir(c.cachePath)
	dirInfo, err := os.Stat(cacheDir)
	if err != nil {
		return err
	}

	dirMode := dirInfo.Mode().Perm()
	if dirMode&0077 != 0 {
		return fmt.Errorf("cache directory has insecure permissions: %o (should be utils.PermDirSecure)", dirMode)
	}

	return nil
}

// ensureSecurePermissions fixes cache file and directory permissions on Unix systems
func (c *Cache) ensureSecurePermissions() error {
	// Fix file permissions
	if err := os.Chmod(c.cachePath, utils.PermFileSecure); err != nil {
		return errors.FailedTo("fix cache file permissions", err)
	}

	// Fix directory permissions
	cacheDir := filepath.Dir(c.cachePath)
	if err := os.Chmod(cacheDir, utils.PermDirSecure); err != nil {
		return errors.FailedTo("fix cache directory permissions", err)
	}

	return nil
}
