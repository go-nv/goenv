package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWindowsExecutableExtensions(t *testing.T) {
	exts := WindowsExecutableExtensions()
	expected := []string{".exe", ".bat", ".cmd", ".com"}

	assert.Len(t, exts, len(expected), "Expected extensions")

	for i, ext := range expected {
		assert.Equal(t, ext, exts[i], "Expected extension at index")
	}
}

func TestFindExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name        string
		createFile  string
		makeExec    bool
		searchName  string
		shouldFind  bool
		expectedExt string
	}{
		{
			name:        "finds exe on windows",
			createFile:  "test.exe",
			searchName:  "test",
			shouldFind:  IsWindows(),
			expectedExt: ".exe",
		},
		{
			name:        "finds bat on windows",
			createFile:  "test.bat",
			searchName:  "test",
			shouldFind:  IsWindows(),
			expectedExt: ".bat",
		},
		{
			name:        "finds cmd on windows",
			createFile:  "test.cmd",
			searchName:  "test",
			shouldFind:  IsWindows(),
			expectedExt: ".cmd",
		},
		{
			name:        "finds exact name on unix",
			createFile:  "test",
			makeExec:    true,
			searchName:  "test",
			shouldFind:  !IsWindows(),
			expectedExt: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test file
			filePath := filepath.Join(tmpDir, tt.createFile)
			testutil.WriteTestFile(t, filePath, []byte("test"), PermFileDefault)

			// Make executable on Unix if needed
			if tt.makeExec && !IsWindows() {
				err := os.Chmod(filePath, PermFileExecutable)
				require.NoError(t, err, "Failed to chmod file")
			}

			// Try to find it
			found, err := FindExecutable(tmpDir, tt.searchName)

			if tt.shouldFind {
				assert.NoError(t, err, "Expected to find executable")
				assert.NotEmpty(t, found, "Expected non-empty path")
				assert.False(t, tt.expectedExt != "" && filepath.Ext(found) != tt.expectedExt, "Expected extension")
			} else {
				assert.Error(t, err, "Expected not to find executable on this platform, but found %v", found)
			}

			// Cleanup
			os.Remove(filePath)
		})
	}
}

func TestIsExecutable(t *testing.T) {
	tmpDir := t.TempDir()

	tests := []struct {
		name         string
		filename     string
		makeExec     bool
		shouldBeExec bool
	}{
		{
			name:         "exe file on windows",
			filename:     "test.exe",
			shouldBeExec: IsWindows(),
		},
		{
			name:         "bat file on windows",
			filename:     "test.bat",
			shouldBeExec: IsWindows(),
		},
		{
			name:         "txt file not executable",
			filename:     "test.txt",
			shouldBeExec: false,
		},
		{
			name:         "file with exec bit on unix",
			filename:     "test",
			makeExec:     true,
			shouldBeExec: !IsWindows(),
		},
		{
			name:         "file without exec bit on unix",
			filename:     "test2",
			makeExec:     false,
			shouldBeExec: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filePath := filepath.Join(tmpDir, tt.filename)
			testutil.WriteTestFile(t, filePath, []byte("test"), PermFileDefault)
			defer os.Remove(filePath)

			if tt.makeExec && !IsWindows() {
				err := os.Chmod(filePath, PermFileExecutable)
				require.NoError(t, err, "Failed to chmod file")
			}

			isExec := IsExecutable(filePath)
			assert.Equal(t, tt.shouldBeExec, isExec, "Expected IsExecutable=")
		})
	}
}

func TestHasExecutableExtension(t *testing.T) {
	if IsWindows() {
		// Test Windows-specific behavior
		tests := []struct {
			filename string
			expected bool
		}{
			{"test.exe", true},
			{"test.bat", true},
			{"test.cmd", true},
			{"test.com", true},
			{"test.txt", false},
			{"test", false},
		}

		for _, tt := range tests {
			t.Run(tt.filename, func(t *testing.T) {
				result := HasExecutableExtension(tt.filename)
				assert.Equal(t, tt.expected, result, "HasExecutableExtension() = , expected %v", tt.filename)
			})
		}
	} else {
		// On Unix, any filename is valid (extensions don't matter)
		tests := []string{"test", "test.sh", "test.exe", "test.txt"}
		for _, filename := range tests {
			t.Run(filename, func(t *testing.T) {
				result := HasExecutableExtension(filename)
				assert.True(t, result, "HasExecutableExtension() = false, expected true on Unix")
			})
		}
	}
}
