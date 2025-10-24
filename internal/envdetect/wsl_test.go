package envdetect

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsWSL(t *testing.T) {
	// This test can only run meaningfully on Linux
	if runtime.GOOS != "linux" {
		t.Skip("WSL detection only works on Linux")
	}

	// We can't reliably test this without being in WSL,
	// but we can verify it doesn't crash
	result := IsWSL()
	t.Logf("IsWSL returned: %v", result)
}

func TestIsWindowsBinary(t *testing.T) {
	tests := []struct {
		name     string
		content  []byte
		expected bool
	}{
		{
			name:     "Windows PE binary",
			content:  []byte{'M', 'Z', 0x90, 0x00},
			expected: true,
		},
		{
			name:     "ELF binary",
			content:  []byte{0x7f, 'E', 'L', 'F'},
			expected: false,
		},
		{
			name:     "Mach-O binary",
			content:  []byte{0xcf, 0xfa, 0xed, 0xfe},
			expected: false,
		},
		{
			name:     "Text file",
			content:  []byte("#!/bin/bash\necho hello"),
			expected: false,
		},
		{
			name:     "Empty file",
			content:  []byte{},
			expected: false,
		},
		{
			name:     "Single byte",
			content:  []byte{'M'},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temporary file
			tmpDir := t.TempDir()
			tmpFile := filepath.Join(tmpDir, "testbinary")

			if err := os.WriteFile(tmpFile, tt.content, 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result := IsWindowsBinary(tmpFile)
			if result != tt.expected {
				t.Errorf("IsWindowsBinary(%s) = %v, expected %v", tt.name, result, tt.expected)
			}
		})
	}
}

func TestIsWindowsBinary_NonExistentFile(t *testing.T) {
	result := IsWindowsBinary("/nonexistent/path/to/binary")
	if result {
		t.Error("IsWindowsBinary should return false for non-existent files")
	}
}

func TestCheckWSLCrossExecution(t *testing.T) {
	if runtime.GOOS != "linux" {
		// On non-Linux systems, should always return empty
		result := CheckWSLCrossExecution("/any/path")
		if result != "" {
			t.Error("CheckWSLCrossExecution should return empty string on non-Linux systems")
		}
		return
	}

	// On Linux (but not WSL), should return empty for any binary
	// We can't easily test the actual WSL scenario without being in WSL
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testbinary")

	// Create an ELF binary
	elfBinary := []byte{0x7f, 'E', 'L', 'F', 0x01, 0x01, 0x01, 0x00}
	if err := os.WriteFile(tmpFile, elfBinary, 0755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckWSLCrossExecution(tmpFile)
	// On regular Linux (not WSL), this should return empty
	// If we're actually in WSL and the binary is ELF, it should also be empty
	t.Logf("CheckWSLCrossExecution result: %q", result)
}

func TestCheckWSLCrossExecution_WindowsBinary(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("This test only runs on Linux")
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testbinary.exe")

	// Create a Windows PE binary
	peBinary := []byte{'M', 'Z', 0x90, 0x00}
	if err := os.WriteFile(tmpFile, peBinary, 0755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckWSLCrossExecution(tmpFile)

	// If we're in WSL, we should get a warning
	// If we're on regular Linux, we should get empty (not WSL)
	if IsWSL() {
		if result == "" {
			t.Error("Expected warning when running Windows binary in WSL")
		}
		// Verify the warning includes the actual host architecture
		if !contains(result, runtime.GOARCH) {
			t.Errorf("Warning message should include host architecture %q, but got: %q", runtime.GOARCH, result)
		}
		// Verify it suggests the correct GOOS
		if !contains(result, "GOOS=linux") {
			t.Errorf("Warning message should suggest GOOS=linux, but got: %q", result)
		}
		t.Logf("WSL warning message: %s", result)
	}

	if !IsWSL() && result != "" {
		t.Error("Should not warn about Windows binary on regular Linux (not WSL)")
	}
}

// Helper function for substring checking
func contains(s, substr string) bool {
	return len(s) >= len(substr) && findSubstring(s, substr)
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
