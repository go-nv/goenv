package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestIsFirstRun(t *testing.T) {
	// Save original env
	origShell := os.Getenv("GOENV_SHELL")
	defer func() {
		if origShell != "" {
			os.Setenv("GOENV_SHELL", origShell)
		} else {
			os.Unsetenv("GOENV_SHELL")
		}
	}()

	tests := []struct {
		name           string
		setupFunc      func(dir string)
		shellSet       bool
		expectedResult bool
	}{
		{
			name: "first run - no versions, no shell",
			setupFunc: func(dir string) {
				// Don't create versions directory
			},
			shellSet:       false,
			expectedResult: true,
		},
		{
			name: "first run - empty versions dir, no shell",
			setupFunc: func(dir string) {
				os.MkdirAll(filepath.Join(dir, "versions"), 0755)
			},
			shellSet:       false,
			expectedResult: true,
		},
		{
			name: "not first run - has versions, no shell",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				os.MkdirAll(filepath.Join(versionsDir, "1.21.5"), 0755)
			},
			shellSet:       false,
			expectedResult: false,
		},
		{
			name: "not first run - no versions, shell initialized",
			setupFunc: func(dir string) {
				// Don't create versions directory
			},
			shellSet:       true,
			expectedResult: false,
		},
		{
			name: "not first run - has versions, shell initialized",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				os.MkdirAll(filepath.Join(versionsDir, "1.21.5"), 0755)
			},
			shellSet:       true,
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory for test
			tmpDir := t.TempDir()

			// Setup test conditions
			tt.setupFunc(tmpDir)

			// Set or unset shell
			if tt.shellSet {
				os.Setenv("GOENV_SHELL", "bash")
			} else {
				os.Unsetenv("GOENV_SHELL")
			}

			// Test
			result := IsFirstRun(tmpDir)
			if result != tt.expectedResult {
				t.Errorf("IsFirstRun() = %v, expected %v", result, tt.expectedResult)
			}
		})
	}
}

func TestIsShellInitialized(t *testing.T) {
	// Save original env
	origShell := os.Getenv("GOENV_SHELL")
	defer func() {
		if origShell != "" {
			os.Setenv("GOENV_SHELL", origShell)
		} else {
			os.Unsetenv("GOENV_SHELL")
		}
	}()

	tests := []struct {
		name     string
		shellVal string
		expected bool
	}{
		{
			name:     "shell initialized - bash",
			shellVal: "bash",
			expected: true,
		},
		{
			name:     "shell initialized - zsh",
			shellVal: "zsh",
			expected: true,
		},
		{
			name:     "shell not initialized",
			shellVal: "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.shellVal != "" {
				os.Setenv("GOENV_SHELL", tt.shellVal)
			} else {
				os.Unsetenv("GOENV_SHELL")
			}

			result := IsShellInitialized()
			if result != tt.expected {
				t.Errorf("IsShellInitialized() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestHasAnyVersionsInstalled(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(dir string)
		expected  bool
	}{
		{
			name: "no versions directory",
			setupFunc: func(dir string) {
				// Don't create anything
			},
			expected: false,
		},
		{
			name: "empty versions directory",
			setupFunc: func(dir string) {
				os.MkdirAll(filepath.Join(dir, "versions"), 0755)
			},
			expected: false,
		},
		{
			name: "one version installed",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				os.MkdirAll(filepath.Join(versionsDir, "1.21.5"), 0755)
			},
			expected: true,
		},
		{
			name: "multiple versions installed",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				os.MkdirAll(filepath.Join(versionsDir, "1.21.5"), 0755)
				os.MkdirAll(filepath.Join(versionsDir, "1.22.0"), 0755)
			},
			expected: true,
		},
		{
			name: "only files in versions directory",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				os.MkdirAll(versionsDir, 0755)
				// Create a file, not a directory
				os.WriteFile(filepath.Join(versionsDir, "somefile.txt"), []byte("test"), 0644)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(tmpDir)

			result := HasAnyVersionsInstalled(tmpDir)
			if result != tt.expected {
				t.Errorf("HasAnyVersionsInstalled() = %v, expected %v", result, tt.expected)
			}
		})
	}
}
