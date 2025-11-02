package utils

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/testing/testutil"
)

func TestWindowsExecutableExtensions(t *testing.T) {
	exts := WindowsExecutableExtensions()
	expected := []string{".exe", ".bat", ".cmd", ".com"}

	if len(exts) != len(expected) {
		t.Errorf("Expected %d extensions, got %d", len(expected), len(exts))
	}

	for i, ext := range expected {
		if exts[i] != ext {
			t.Errorf("Expected extension %s at index %d, got %s", ext, i, exts[i])
		}
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
				if err != nil {
					t.Fatalf("Failed to chmod file: %v", err)
				}
			}

			// Try to find it
			found, err := FindExecutable(tmpDir, tt.searchName)

			if tt.shouldFind {
				if err != nil {
					t.Errorf("Expected to find executable, got error: %v", err)
				}
				if found == "" {
					t.Error("Expected non-empty path")
				}
				if tt.expectedExt != "" && filepath.Ext(found) != tt.expectedExt {
					t.Errorf("Expected extension %s, got %s", tt.expectedExt, filepath.Ext(found))
				}
			} else {
				if err == nil {
					t.Errorf("Expected not to find executable on this platform, but found: %s", found)
				}
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
				if err != nil {
					t.Fatalf("Failed to chmod file: %v", err)
				}
			}

			isExec := IsExecutable(filePath)
			if isExec != tt.shouldBeExec {
				t.Errorf("Expected IsExecutable=%v, got %v", tt.shouldBeExec, isExec)
			}
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
				if result != tt.expected {
					t.Errorf("HasExecutableExtension(%q) = %v, expected %v",
						tt.filename, result, tt.expected)
				}
			})
		}
	} else {
		// On Unix, any filename is valid (extensions don't matter)
		tests := []string{"test", "test.sh", "test.exe", "test.txt"}
		for _, filename := range tests {
			t.Run(filename, func(t *testing.T) {
				result := HasExecutableExtension(filename)
				if !result {
					t.Errorf("HasExecutableExtension(%q) = false, expected true on Unix", filename)
				}
			})
		}
	}
}
