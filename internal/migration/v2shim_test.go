package migration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestRemoveStaleV2Shim(t *testing.T) {
	// Create a temporary directory for testing
	tmpDir, err := os.MkdirTemp("", "goenv-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	shimsDir := filepath.Join(tmpDir, "shims")
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		t.Fatalf("Failed to create shims dir: %v", err)
	}

	t.Run("removes v2 shim with forward slash", func(t *testing.T) {
		// Create a v2 shim with forward slash
		shimPath := filepath.Join(shimsDir, "goenv")
		v2Content := `#!/usr/bin/env bash
exec "/opt/homebrew/Cellar/goenv/2.2.38_1/libexec/goenv" "$@"
`
		if err := os.WriteFile(shimPath, []byte(v2Content), 0755); err != nil {
			t.Fatalf("Failed to create shim: %v", err)
		}

		removed, err := RemoveStaleV2Shim(shimsDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !removed {
			t.Error("Expected shim to be removed")
		}
		if _, err := os.Stat(shimPath); !os.IsNotExist(err) {
			t.Error("Shim file should not exist")
		}
	})

	t.Run("removes v2 shim with backslash", func(t *testing.T) {
		// Create a v2 shim with backslash (Windows-style)
		shimPath := filepath.Join(shimsDir, "goenv")
		v2Content := `@echo off
"C:\Program Files\goenv\libexec\goenv" %*
`
		if err := os.WriteFile(shimPath, []byte(v2Content), 0755); err != nil {
			t.Fatalf("Failed to create shim: %v", err)
		}

		removed, err := RemoveStaleV2Shim(shimsDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if !removed {
			t.Error("Expected shim to be removed")
		}
		if _, err := os.Stat(shimPath); !os.IsNotExist(err) {
			t.Error("Shim file should not exist")
		}
	})

	t.Run("does not remove non-v2 shim", func(t *testing.T) {
		// Create a non-v2 shim
		shimPath := filepath.Join(shimsDir, "goenv")
		v3Content := `#!/usr/bin/env bash
# This is a v3 shim
exec "goenv" "$@"
`
		if err := os.WriteFile(shimPath, []byte(v3Content), 0755); err != nil {
			t.Fatalf("Failed to create shim: %v", err)
		}

		removed, err := RemoveStaleV2Shim(shimsDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if removed {
			t.Error("Expected shim to NOT be removed")
		}
		if _, err := os.Stat(shimPath); os.IsNotExist(err) {
			t.Error("Shim file should still exist")
		}
	})

	t.Run("handles missing shim file", func(t *testing.T) {
		// Ensure no shim exists
		shimPath := filepath.Join(shimsDir, "goenv")
		os.Remove(shimPath)

		removed, err := RemoveStaleV2Shim(shimsDir)
		if err != nil {
			t.Errorf("Unexpected error: %v", err)
		}
		if removed {
			t.Error("Expected no removal when file doesn't exist")
		}
	})

	t.Run("returns error when removal fails", func(t *testing.T) {
		// Skip on Windows - Unix permission model doesn't apply
		if runtime.GOOS == "windows" {
			t.Skip("Skipping permission test on Windows - different permission model")
		}

		shimPath := filepath.Join(shimsDir, "goenv")
		v2Content := `#!/usr/bin/env bash
exec "/opt/homebrew/Cellar/goenv/2.2.38_1/libexec/goenv" "$@"
`
		if err := os.WriteFile(shimPath, []byte(v2Content), 0755); err != nil {
			t.Fatalf("Failed to create shim: %v", err)
		}

		// Make directory read-only (Unix-like systems only)
		if err := os.Chmod(shimsDir, 0555); err != nil {
			t.Skipf("Cannot test permission error: %v", err)
		}
		defer os.Chmod(shimsDir, 0755) // Restore permissions

		removed, err := RemoveStaleV2Shim(shimsDir)
		if err == nil {
			t.Error("Expected error when removal fails")
		}
		if removed {
			t.Error("Should not report as removed when error occurs")
		}
	})
}

func TestV2ShimPattern(t *testing.T) {
	tests := []struct {
		name    string
		content string
		matches bool
	}{
		{
			name:    "forward slash",
			content: `exec "/opt/homebrew/Cellar/goenv/2.2.38_1/libexec/goenv" "$@"`,
			matches: true,
		},
		{
			name:    "backslash",
			content: `"C:\Program Files\goenv\libexec\goenv" %*`,
			matches: true,
		},
		{
			name:    "no match",
			content: `exec "goenv" "$@"`,
			matches: false,
		},
		{
			name:    "libexec only",
			content: `exec "/usr/local/libexec/goenv-other" "$@"`,
			matches: false,
		},
		{
			name:    "goenv only",
			content: `exec "/usr/local/bin/goenv" "$@"`,
			matches: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			matches := V2ShimPattern.MatchString(tt.content)
			if matches != tt.matches {
				t.Errorf("Expected match=%v, got match=%v for content: %s", tt.matches, matches, tt.content)
			}
		})
	}
}
