package cache

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewAtomicWriter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goenv-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("creates cache directory", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache")
		writer, err := NewAtomicWriter(cacheDir)
		if err != nil {
			t.Fatalf("Failed to create atomic writer: %v", err)
		}
		defer writer.Close()

		// Verify cache directory was created
		if _, err := os.Stat(cacheDir); os.IsNotExist(err) {
			t.Error("Cache directory was not created")
		}

		// Verify lock file was created
		lockPath := filepath.Join(cacheDir, ".goenv-cache.lock")
		if _, err := os.Stat(lockPath); os.IsNotExist(err) {
			t.Error("Lock file was not created")
		}

		// Verify temp directory was created
		tmpPath := filepath.Join(cacheDir, ".tmp")
		if _, err := os.Stat(tmpPath); os.IsNotExist(err) {
			t.Error("Temp directory was not created")
		}
	})

	t.Run("prevents concurrent access", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache-concurrent")
		writer1, err := NewAtomicWriter(cacheDir)
		if err != nil {
			t.Fatalf("Failed to create first writer: %v", err)
		}
		defer writer1.Close()

		// Try to create second writer - should fail because of lock
		writer2, err := NewAtomicWriter(cacheDir)
		if err == nil {
			writer2.Close()
			t.Fatal("Expected error when creating second writer, got nil")
		}

		if !isLockError(err) {
			t.Errorf("Expected lock error, got: %v", err)
		}
	})
}

func TestAtomicWriter_WriteFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goenv-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cacheDir := filepath.Join(tmpDir, "cache")
	writer, err := NewAtomicWriter(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create atomic writer: %v", err)
	}
	defer writer.Close()

	t.Run("writes file atomically", func(t *testing.T) {
		targetPath := filepath.Join(cacheDir, "test.txt")
		data := []byte("test data")

		err := writer.WriteFile(targetPath, data, 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Verify file was written
		readData, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(readData) != string(data) {
			t.Errorf("Expected %q, got %q", string(data), string(readData))
		}
	})

	t.Run("creates target directory if needed", func(t *testing.T) {
		targetPath := filepath.Join(cacheDir, "subdir", "nested", "test.txt")
		data := []byte("nested data")

		err := writer.WriteFile(targetPath, data, 0644)
		if err != nil {
			t.Fatalf("Failed to write file: %v", err)
		}

		// Verify nested directories were created
		if _, err := os.Stat(filepath.Dir(targetPath)); os.IsNotExist(err) {
			t.Error("Target directory was not created")
		}

		// Verify file was written
		readData, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(readData) != string(data) {
			t.Errorf("Expected %q, got %q", string(data), string(readData))
		}
	})

	t.Run("overwrites existing file", func(t *testing.T) {
		targetPath := filepath.Join(cacheDir, "overwrite.txt")

		// Write initial data
		initialData := []byte("initial")
		err := writer.WriteFile(targetPath, initialData, 0644)
		if err != nil {
			t.Fatalf("Failed to write initial file: %v", err)
		}

		// Overwrite with new data
		newData := []byte("overwritten")
		err = writer.WriteFile(targetPath, newData, 0644)
		if err != nil {
			t.Fatalf("Failed to overwrite file: %v", err)
		}

		// Verify new data
		readData, err := os.ReadFile(targetPath)
		if err != nil {
			t.Fatalf("Failed to read file: %v", err)
		}

		if string(readData) != string(newData) {
			t.Errorf("Expected %q, got %q", string(newData), string(readData))
		}
	})

	t.Run("handles concurrent writes with unique temp files", func(t *testing.T) {
		// Write multiple files concurrently to test temp file uniqueness
		done := make(chan bool)
		errors := make(chan error, 3)

		for i := 0; i < 3; i++ {
			go func(n int) {
				targetPath := filepath.Join(cacheDir, "concurrent", "file.txt")
				data := []byte("concurrent write")

				err := writer.WriteFile(targetPath, data, 0644)
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
	tmpDir, err := os.MkdirTemp("", "goenv-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cacheDir := filepath.Join(tmpDir, "cache")
	writer, err := NewAtomicWriter(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create atomic writer: %v", err)
	}
	defer writer.Close()

	t.Run("creates directory", func(t *testing.T) {
		dirPath := filepath.Join(cacheDir, "testdir")
		err := writer.CreateDirectory(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create directory: %v", err)
		}

		// Verify directory was created
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Path is not a directory")
		}
	})

	t.Run("creates nested directories", func(t *testing.T) {
		dirPath := filepath.Join(cacheDir, "nested", "sub", "dir")
		err := writer.CreateDirectory(dirPath, 0755)
		if err != nil {
			t.Fatalf("Failed to create nested directory: %v", err)
		}

		// Verify directory was created
		info, err := os.Stat(dirPath)
		if err != nil {
			t.Fatalf("Nested directory was not created: %v", err)
		}

		if !info.IsDir() {
			t.Error("Path is not a directory")
		}
	})
}

func TestAtomicWriter_Close(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goenv-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	cacheDir := filepath.Join(tmpDir, "cache")
	writer, err := NewAtomicWriter(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create atomic writer: %v", err)
	}

	// Close the writer
	err = writer.Close()
	if err != nil {
		t.Fatalf("Failed to close writer: %v", err)
	}

	// After closing, another writer should be able to acquire the lock
	writer2, err := NewAtomicWriter(cacheDir)
	if err != nil {
		t.Fatalf("Failed to create second writer after close: %v", err)
	}
	defer writer2.Close()

	// Multiple closes will return an error (file already closed), which is acceptable
	// We just verify it doesn't crash
	_ = writer.Close()
}

func TestTryNewAtomicWriter(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "goenv-cache-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	t.Run("succeeds when cache is available", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache-try")
		writer, err := TryNewAtomicWriter(cacheDir)
		if err != nil {
			t.Fatalf("TryNewAtomicWriter failed: %v", err)
		}
		defer writer.Close()

		if writer == nil {
			t.Error("Expected non-nil writer")
		}
	})

	t.Run("returns nil when cache is locked", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache-try-locked")
		writer1, err := NewAtomicWriter(cacheDir)
		if err != nil {
			t.Fatalf("Failed to create first writer: %v", err)
		}
		defer writer1.Close()

		// Try to create second writer - should return nil without error
		writer2, err := TryNewAtomicWriter(cacheDir)
		if err != nil {
			t.Fatalf("TryNewAtomicWriter returned error: %v", err)
		}

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
			want: false,           // This specific error doesn't contain "locked"
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

func Test_contains(t *testing.T) {
	tests := []struct {
		name   string
		s      string
		substr string
		want   bool
	}{
		{
			name:   "exact match",
			s:      "locked by another process",
			substr: "locked by another process",
			want:   true,
		},
		{
			name:   "substring present",
			s:      "error: locked by another process",
			substr: "locked by another process",
			want:   true,
		},
		{
			name:   "substring at start",
			s:      "locked at the beginning",
			substr: "locked",
			want:   true,
		},
		{
			name:   "substring at end",
			s:      "this is locked",
			substr: "locked",
			want:   true,
		},
		{
			name:   "substring not present",
			s:      "some other error",
			substr: "locked",
			want:   false,
		},
		{
			name:   "empty substring",
			s:      "any string",
			substr: "",
			want:   true,
		},
		{
			name:   "empty string",
			s:      "",
			substr: "locked",
			want:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := contains(tt.s, tt.substr); got != tt.want {
				t.Errorf("contains() = %v, want %v", got, tt.want)
			}
		})
	}
}
