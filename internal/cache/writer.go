// Package cache provides atomic write operations for cache management.
// This prevents corruption when multiple processes access the cache concurrently.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/utils"
)

// tempFileCounter is used to generate unique temp file names
var tempFileCounter uint64

// AtomicWriter provides atomic file write operations with advisory locking.
// It ensures that cache writes are safe even on networked filesystems.
type AtomicWriter struct {
	lockFile  *os.File
	tmpDir    string
	cacheRoot string
}

// NewAtomicWriter creates an atomic writer for the given cache root.
// It acquires an advisory lock on the cache directory to prevent concurrent modifications.
func NewAtomicWriter(cacheRoot string) (*AtomicWriter, error) {
	// Create cache root if it doesn't exist
	if err := utils.EnsureDirWithContext(cacheRoot, "create cache directory"); err != nil {
		return nil, err
	}

	// Advisory lock per cache root
	lockPath := filepath.Join(cacheRoot, ".goenv-cache.lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, utils.PermFileDefault)
	if err != nil {
		return nil, errors.FailedTo("create lock file", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	if err := acquireLock(lockFile); err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("cache is locked by another process: %w", err)
	}

	// Create temporary directory for atomic writes
	tmpDir := filepath.Join(cacheRoot, ".tmp")
	if err := utils.EnsureDirWithContext(tmpDir, "create temp directory"); err != nil {
		releaseLock(lockFile)
		lockFile.Close()
		return nil, err
	}

	return &AtomicWriter{
		lockFile:  lockFile,
		tmpDir:    tmpDir,
		cacheRoot: cacheRoot,
	}, nil
}

// WriteFile atomically writes data to the target path.
// It writes to a temporary file first, then renames it to the target path.
// This ensures that readers never see partial writes.
func (w *AtomicWriter) WriteFile(targetPath string, data []byte, perm os.FileMode) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := utils.EnsureDirWithContext(targetDir, "create target directory"); err != nil {
		return err
	}

	// Write to temp file with unique name to avoid concurrent write collisions
	// Use atomic counter to guarantee uniqueness even under high concurrency
	counter := atomic.AddUint64(&tempFileCounter, 1)
	tmpPath := filepath.Join(
		w.tmpDir,
		fmt.Sprintf("%s.%d.%d.%d.tmp", filepath.Base(targetPath), os.Getpid(), time.Now().UnixNano(), counter),
	)

	// Write and sync the file in one operation
	// Open with O_RDWR on Windows to allow Sync() to work
	f, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return errors.FailedTo("create temp file", err)
	}

	if _, err := f.Write(data); err != nil {
		f.Close()
		return errors.FailedTo("write temp file", err)
	}

	// Fsync the file before closing
	if err := f.Sync(); err != nil {
		f.Close()
		return errors.FailedTo("sync temp file", err)
	}

	if err := f.Close(); err != nil {
		return errors.FailedTo("close temp file", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return errors.FailedTo("rename temp file", err)
	}

	// Fsync the directory to ensure the rename is durable
	dir, err := os.Open(targetDir)
	if err == nil {
		dir.Sync()
		dir.Close()
	}

	return nil
}

// CreateDirectory atomically creates a directory.
// This is useful for creating cache subdirectories safely.
func (w *AtomicWriter) CreateDirectory(path string, perm os.FileMode) error {
	if err := os.MkdirAll(path, perm); err != nil {
		return errors.FailedTo("create directory", err)
	}
	return nil
}

// Close releases the lock and closes the lock file.
// Always call this when done with the writer.
func (w *AtomicWriter) Close() error {
	if w.lockFile != nil {
		releaseLock(w.lockFile)
		return w.lockFile.Close()
	}
	return nil
}

// TryNewAtomicWriter attempts to create an atomic writer but returns nil if the cache is locked.
// This is useful for non-critical operations that should skip if the cache is busy.
func TryNewAtomicWriter(cacheRoot string) (*AtomicWriter, error) {
	writer, err := NewAtomicWriter(cacheRoot)
	if err != nil {
		// If locked, return nil without error
		if isLockError(err) {
			return nil, nil
		}
		return nil, err
	}
	return writer, nil
}

// isLockError checks if an error is a lock-related error
func isLockError(err error) bool {
	if err == nil {
		return false
	}
	return strings.Contains(err.Error(), "locked by another process")
}
