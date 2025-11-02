//go:build !windows
// +build !windows

package hooks

import (
	"syscall"

	"github.com/go-nv/goenv/internal/errors"
)

// getDiskSpace returns free and total disk space in MB for the given path (Unix)
func getDiskSpace(path string) (freeMB, totalMB int64, err error) {
	var stat syscall.Statfs_t
	if err := syscall.Statfs(path, &stat); err != nil {
		return 0, 0, errors.FailedTo("get filesystem stats", err)
	}

	// Calculate available space in MB
	// stat.Bavail = available blocks for unprivileged users
	// stat.Bsize = block size in bytes
	freeMB = int64(stat.Bavail) * int64(stat.Bsize) / (1024 * 1024)
	totalMB = int64(stat.Blocks) * int64(stat.Bsize) / (1024 * 1024)

	return freeMB, totalMB, nil
}
