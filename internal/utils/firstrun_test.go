package utils

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIsFirstRun(t *testing.T) {
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
				EnsureDir(filepath.Join(dir, "versions"))
			},
			shellSet:       false,
			expectedResult: true,
		},
		{
			name: "not first run - has versions, no shell",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				EnsureDir(filepath.Join(versionsDir, "1.21.5"))
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
				EnsureDir(filepath.Join(versionsDir, "1.21.5"))
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

			// Set shell environment for this subtest
			if tt.shellSet {
				t.Setenv(GoenvEnvVarShell.String(), "bash")
			} else {
				// Ensure GOENV_SHELL is unset for this subtest
				t.Setenv(GoenvEnvVarShell.String(), "")
			}

			// Test
			result := IsFirstRun(tmpDir)
			assert.Equal(t, tt.expectedResult, result, "IsFirstRun() = , expected")
		})
	}
}

func TestIsShellInitialized(t *testing.T) {
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
				t.Setenv(GoenvEnvVarShell.String(), tt.shellVal)
			} else {
				// Ensure GOENV_SHELL is unset for this subtest
				t.Setenv(GoenvEnvVarShell.String(), "")
			}

			result := IsShellInitialized()
			assert.Equal(t, tt.expected, result, "IsShellInitialized() = , expected")
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
				EnsureDir(filepath.Join(dir, "versions"))
			},
			expected: false,
		},
		{
			name: "one version installed",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				EnsureDir(filepath.Join(versionsDir, "1.21.5"))
			},
			expected: true,
		},
		{
			name: "multiple versions installed",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				EnsureDir(filepath.Join(versionsDir, "1.21.5"))
				EnsureDir(filepath.Join(versionsDir, "1.22.0"))
			},
			expected: true,
		},
		{
			name: "only files in versions directory",
			setupFunc: func(dir string) {
				versionsDir := filepath.Join(dir, "versions")
				_ = EnsureDirWithContext(versionsDir, "create test directory")
				// Create a file, not a directory
				testutil.WriteTestFile(t, filepath.Join(versionsDir, "somefile.txt"), []byte("test"), PermFileDefault)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupFunc(tmpDir)

			result := HasAnyVersionsInstalled(tmpDir)
			assert.Equal(t, tt.expected, result, "HasAnyVersionsInstalled() = , expected")
		})
	}
}
