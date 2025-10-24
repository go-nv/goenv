// Package cache provides atomic write operations for cache management.
// This prevents corruption when multiple processes access the cache concurrently.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
	"time"
)

// tempFileCounter is used to generate unique temp file names
var tempFileCounter uint64

// AtomicWriter provides atomic file write operations with advisory locking.
// It ensures that cache writes are safe even on networked filesystems.
type AtomicWriter struct {
	lockFile *os.File
	tmpDir   string
	cacheRoot string
}

// NewAtomicWriter creates an atomic writer for the given cache root.
// It acquires an advisory lock on the cache directory to prevent concurrent modifications.
func NewAtomicWriter(cacheRoot string) (*AtomicWriter, error) {
	// Create cache root if it doesn't exist
	if err := os.MkdirAll(cacheRoot, 0755); err != nil {
		return nil, fmt.Errorf("failed to create cache directory: %w", err)
	}

	// Advisory lock per cache root
	lockPath := filepath.Join(cacheRoot, ".goenv-cache.lock")
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return nil, fmt.Errorf("failed to create lock file: %w", err)
	}

	// Try to acquire exclusive lock (non-blocking)
	if err := acquireLock(lockFile); err != nil {
		lockFile.Close()
		return nil, fmt.Errorf("cache is locked by another process: %w", err)
	}

	// Create temporary directory for atomic writes
	tmpDir := filepath.Join(cacheRoot, ".tmp")
	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		releaseLock(lockFile)
		lockFile.Close()
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}

	return &AtomicWriter{
		lockFile: lockFile,
		tmpDir:   tmpDir,
		cacheRoot: cacheRoot,
	}, nil
}

// WriteFile atomically writes data to the target path.
// It writes to a temporary file first, then renames it to the target path.
// This ensures that readers never see partial writes.
func (w *AtomicWriter) WriteFile(targetPath string, data []byte, perm os.FileMode) error {
	// Ensure target directory exists
	targetDir := filepath.Dir(targetPath)
	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("failed to create target directory: %w", err)
	}

	// Write to temp file with unique name to avoid concurrent write collisions
	// Use atomic counter to guarantee uniqueness even under high concurrency
	counter := atomic.AddUint64(&tempFileCounter, 1)
	tmpPath := filepath.Join(
		w.tmpDir,
		fmt.Sprintf("%s.%d.%d.%d.tmp", filepath.Base(targetPath), os.Getpid(), time.Now().UnixNano(), counter),
	)
	if err := os.WriteFile(tmpPath, data, perm); err != nil {
		return fmt.Errorf("failed to write temp file: %w", err)
	}

	// Fsync the file
	f, err := os.Open(tmpPath)
	if err != nil {
		return fmt.Errorf("failed to open temp file for sync: %w", err)
	}
	if err := f.Sync(); err != nil {
		f.Close()
		return fmt.Errorf("failed to sync temp file: %w", err)
	}
	f.Close()

	// Atomic rename
	if err := os.Rename(tmpPath, targetPath); err != nil {
		return fmt.Errorf("failed to rename temp file: %w", err)
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
		return fmt.Errorf("failed to create directory: %w", err)
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
	return contains(err.Error(), "locked by another process")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && s[len(s)-len(substr):] == substr ||
	       len(s) > len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
