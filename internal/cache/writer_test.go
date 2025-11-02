package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAtomicWriter(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("creates cache directory", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache")
		writer, err := NewAtomicWriter(cacheDir)
		require.NoError(t, err, "Failed to create atomic writer")
		defer writer.Close()

		// Verify cache directory was created
		if utils.FileNotExists(cacheDir) {
			t.Error("Cache directory was not created")
		}

		// Verify lock file was created
		lockPath := filepath.Join(cacheDir, ".goenv-cache.lock")
		if utils.FileNotExists(lockPath) {
			t.Error("Lock file was not created")
		}

		// Verify temp directory was created
		tmpPath := filepath.Join(cacheDir, ".tmp")
		if utils.FileNotExists(tmpPath) {
			t.Error("Temp directory was not created")
		}
	})

	t.Run("prevents concurrent access", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache-concurrent")
		writer1, err := NewAtomicWriter(cacheDir)
		require.NoError(t, err, "Failed to create first writer")
		defer writer1.Close()

		// Try to create second writer - should fail because of lock
		writer2, err := NewAtomicWriter(cacheDir)
		if err == nil {
			writer2.Close()
			t.Fatal("Expected error when creating second writer, got nil")
		}

		assert.True(t, isLockError(err), "Expected lock error")
	})
}

func TestAtomicWriter_WriteFile(t *testing.T) {
	tmpDir := t.TempDir()

	cacheDir := filepath.Join(tmpDir, "cache")
	writer, err := NewAtomicWriter(cacheDir)
	require.NoError(t, err, "Failed to create atomic writer")
	defer writer.Close()

	t.Run("writes file atomically", func(t *testing.T) {
		targetPath := filepath.Join(cacheDir, "test.txt")
		data := []byte("test data")

		err := writer.WriteFile(targetPath, data, utils.PermFileDefault)
		require.NoError(t, err, "Failed to write file")

		// Verify file was written
		readData, err := os.ReadFile(targetPath)
		require.NoError(t, err, "Failed to read file")

		assert.Equal(t, string(data), string(readData))
	})

	t.Run("creates target directory if needed", func(t *testing.T) {
		targetPath := filepath.Join(cacheDir, "subdir", "nested", "test.txt")
		data := []byte("nested data")

		err := writer.WriteFile(targetPath, data, utils.PermFileDefault)
		require.NoError(t, err, "Failed to write file")

		// Verify nested directories were created
		if utils.FileNotExists(filepath.Dir(targetPath)) {
			t.Error("Target directory was not created")
		}

		// Verify file was written
		readData, err := os.ReadFile(targetPath)
		require.NoError(t, err, "Failed to read file")

		assert.Equal(t, string(data), string(readData))
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		targetPath := filepath.Join(cacheDir, "overwrite.txt")

		// Write initial data
		initialData := []byte("initial")
		err := writer.WriteFile(targetPath, initialData, utils.PermFileDefault)
		require.NoError(t, err, "Failed to write initial file")

		// Overwrite with new data
		newData := []byte("overwritten")
		err = writer.WriteFile(targetPath, newData, utils.PermFileDefault)
		require.NoError(t, err, "Failed to overwrite file")

		// Verify new data
		readData, err := os.ReadFile(targetPath)
		require.NoError(t, err, "Failed to read file")

		assert.Equal(t, string(newData), string(readData))
	})

	t.Run("handles concurrent writes with unique temp files", func(t *testing.T) {
		// Write multiple files concurrently to test temp file uniqueness
		done := make(chan bool)
		errors := make(chan error, 3)

		for i := 0; i < 3; i++ {
			go func(n int) {
				targetPath := filepath.Join(cacheDir, "concurrent", "file.txt")
				data := []byte("concurrent write")

				err := writer.WriteFile(targetPath, data, utils.PermFileDefault)
				if err != nil {
					errors <- err
				}
				done <- true
			}(i)
		}

		// Wait for all writes to complete
		for i := 0; i < 3; i++ {
			<-done
		}

		// Check for errors
		select {
		case err := <-errors:
			t.Errorf("Concurrent write failed: %v", err)
		default:
			// No errors
		}
	})
}

func TestAtomicWriter_CreateDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	cacheDir := filepath.Join(tmpDir, "cache")
	writer, err := NewAtomicWriter(cacheDir)
	require.NoError(t, err, "Failed to create atomic writer")
	defer writer.Close()

	t.Run("creates directory", func(t *testing.T) {
		dirPath := filepath.Join(cacheDir, "testdir")
		err := writer.CreateDirectory(dirPath, utils.PermFileExecutable)
		require.NoError(t, err, "Failed to create directory")

		// Verify directory was created
		require.True(t, utils.DirExists(dirPath), "Directory was not created")
	})

	t.Run("creates nested directories", func(t *testing.T) {
		dirPath := filepath.Join(cacheDir, "nested", "sub", "dir")
		err := writer.CreateDirectory(dirPath, utils.PermFileExecutable)
		require.NoError(t, err, "Failed to create nested directory")

		// Verify directory was created
		require.True(t, utils.DirExists(dirPath), "Nested directory was not created")
	})
}

func TestAtomicWriter_Close(t *testing.T) {
	tmpDir := t.TempDir()

	cacheDir := filepath.Join(tmpDir, "cache")
	writer, err := NewAtomicWriter(cacheDir)
	require.NoError(t, err, "Failed to create atomic writer")

	// Close the writer
	err = writer.Close()
	require.NoError(t, err, "Failed to close writer")

	// After closing, another writer should be able to acquire the lock
	writer2, err := NewAtomicWriter(cacheDir)
	require.NoError(t, err, "Failed to create second writer after close")
	defer writer2.Close()

	// Multiple closes will return an error (file already closed), which is acceptable
	// We just verify it doesn't crash
	_ = writer.Close()
}

func TestTryNewAtomicWriter(t *testing.T) {
	tmpDir := t.TempDir()

	t.Run("succeeds when cache is available", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache-try")
		writer, err := TryNewAtomicWriter(cacheDir)
		require.NoError(t, err, "TryNewAtomicWriter failed")
		defer writer.Close()

		assert.NotNil(t, writer, "Expected non-nil writer")
	})

	t.Run("returns nil when cache is locked", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache-try-locked")
		writer1, err := NewAtomicWriter(cacheDir)
		require.NoError(t, err, "Failed to create first writer")
		defer writer1.Close()

		// Try to create second writer - should return nil without error
		writer2, err := TryNewAtomicWriter(cacheDir)
		require.NoError(t, err, "TryNewAtomicWriter returned error")

		if writer2 != nil {
			writer2.Close()
			t.Error("Expected nil writer when cache is locked")
		}
	})
}

func Test_isLockError(t *testing.T) {
	tests := []struct {
		name string
		err  error
		want bool
	}{
		{
			name: "nil error",
			err:  nil,
			want: false,
		},
		{
			name: "lock error",
			err:  os.ErrPermission, // Simulate a lock error for testing purposes
			want: false,            // This specific error doesn't contain "locked"
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := isLockError(tt.err); got != tt.want {
				t.Errorf("isLockError() = %v, want %v", got, tt.want)
			}
		})
	}
}
