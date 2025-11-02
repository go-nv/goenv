package utils

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestIsPathInPATH(t *testing.T) {
	tests := []struct {
		name     string
		dirPath  string
		pathEnv  string
		expected bool
	}{
		{
			name:     "empty PATH",
			dirPath:  "/usr/bin",
			pathEnv:  "",
			expected: false,
		},
		{
			name:     "empty dirPath",
			dirPath:  "",
			pathEnv:  "/usr/bin:/usr/local/bin",
			expected: false,
		},
		{
			name:     "exact match",
			dirPath:  "/usr/bin",
			pathEnv:  "/usr/bin:/usr/local/bin",
			expected: true,
		},
		{
			name:     "not found",
			dirPath:  "/opt/bin",
			pathEnv:  "/usr/bin:/usr/local/bin",
			expected: false,
		},
		{
			name:     "path with trailing slash",
			dirPath:  "/usr/bin/",
			pathEnv:  "/usr/bin:/usr/local/bin",
			expected: true,
		},
		{
			name:     "PATH entry with trailing slash",
			dirPath:  "/usr/bin",
			pathEnv:  "/usr/bin/:/usr/local/bin",
			expected: true,
		},
		{
			name:     "middle of PATH",
			dirPath:  "/opt/bin",
			pathEnv:  "/usr/bin:/opt/bin:/usr/local/bin",
			expected: true,
		},
		{
			name:     "last in PATH",
			dirPath:  "/home/user/.local/bin",
			pathEnv:  "/usr/bin:/usr/local/bin:/home/user/.local/bin",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// On Unix, use the test case as-is
			// On Windows, convert to Windows-style paths
			dirPath := tt.dirPath
			pathEnv := tt.pathEnv

			if IsWindows() {
				// Convert Unix-style paths to Windows-style for the test
				dirPath = strings.ReplaceAll(dirPath, "/", "\\")
				if strings.HasPrefix(dirPath, "\\") {
					dirPath = "C:" + dirPath
				}

				// Convert PATH entries
				parts := strings.Split(pathEnv, ":")
				var windowsParts []string
				for _, p := range parts {
					if p != "" {
						wp := strings.ReplaceAll(p, "/", "\\")
						if strings.HasPrefix(wp, "\\") {
							wp = "C:" + wp
						}
						windowsParts = append(windowsParts, wp)
					}
				}
				pathEnv = strings.Join(windowsParts, ";")
			}

			got := IsPathInPATH(dirPath, pathEnv)
			if got != tt.expected {
				t.Errorf("IsPathInPATH(%q, %q) = %v, want %v", dirPath, pathEnv, got, tt.expected)
			}
		})
	}
}

func TestIsPathInPATH_CaseInsensitiveWindows(t *testing.T) {
	if !IsWindows() {
		t.Skip("This test is Windows-specific")
	}

	tests := []struct {
		name     string
		dirPath  string
		pathEnv  string
		expected bool
	}{
		{
			name:     "different case - all uppercase",
			dirPath:  "C:\\PROGRAM FILES\\GO\\BIN",
			pathEnv:  "C:\\Program Files\\Go\\bin;C:\\Windows\\System32",
			expected: true,
		},
		{
			name:     "different case - all lowercase",
			dirPath:  "c:\\program files\\go\\bin",
			pathEnv:  "C:\\Program Files\\Go\\bin;C:\\Windows\\System32",
			expected: true,
		},
		{
			name:     "mixed separators",
			dirPath:  "C:/Program Files/Go/bin",
			pathEnv:  "C:\\Program Files\\Go\\bin;C:\\Windows\\System32",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPathInPATH(tt.dirPath, tt.pathEnv)
			if got != tt.expected {
				t.Errorf("IsPathInPATH(%q, %q) = %v, want %v", tt.dirPath, tt.pathEnv, got, tt.expected)
			}
		})
	}
}

func TestIsPathInPATH_CaseSensitiveUnix(t *testing.T) {
	if IsWindows() {
		t.Skip("This test is Unix-specific")
	}

	tests := []struct {
		name     string
		dirPath  string
		pathEnv  string
		expected bool
	}{
		{
			name:     "different case should not match on Unix",
			dirPath:  "/usr/BIN",
			pathEnv:  "/usr/bin:/usr/local/bin",
			expected: false,
		},
		{
			name:     "exact case should match",
			dirPath:  "/usr/bin",
			pathEnv:  "/usr/bin:/usr/local/bin",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPathInPATH(tt.dirPath, tt.pathEnv)
			if got != tt.expected {
				t.Errorf("IsPathInPATH(%q, %q) = %v, want %v", tt.dirPath, tt.pathEnv, got, tt.expected)
			}
		})
	}
}

func TestIsPathInPATH_RealPaths(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()

	// Create subdirectories
	bin1 := filepath.Join(tmpDir, "bin1")
	bin2 := filepath.Join(tmpDir, "bin2")
	bin3 := filepath.Join(tmpDir, "bin3")

	for _, dir := range []string{bin1, bin2, bin3} {
		if err := EnsureDirWithContext(dir, "create test directory"); err != nil {
			t.Fatal(err)
		}
	}

	// Create a PATH with these directories
	pathEnv := bin1 + string(os.PathListSeparator) + bin2 + string(os.PathListSeparator) + bin3

	tests := []struct {
		name     string
		dirPath  string
		expected bool
	}{
		{
			name:     "first directory",
			dirPath:  bin1,
			expected: true,
		},
		{
			name:     "middle directory",
			dirPath:  bin2,
			expected: true,
		},
		{
			name:     "last directory",
			dirPath:  bin3,
			expected: true,
		},
		{
			name:     "non-existent directory",
			dirPath:  filepath.Join(tmpDir, "bin4"),
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := IsPathInPATH(tt.dirPath, pathEnv)
			if got != tt.expected {
				t.Errorf("IsPathInPATH(%q, %q) = %v, want %v", tt.dirPath, pathEnv, got, tt.expected)
			}
		})
	}
}
