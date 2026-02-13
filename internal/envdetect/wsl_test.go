package envdetect

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/osinfo"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
)

func TestIsWSL(t *testing.T) {
	// This test can only run meaningfully on Linux
	if !osinfo.IsLinux() {
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

			testutil.WriteTestFile(t, tmpFile, tt.content, utils.PermFileDefault, "Failed to create test file")

			result := IsWindowsBinary(tmpFile)
			assert.Equal(t, tt.expected, result, "IsWindowsBinary() = , expected %v", tt.name)
		})
	}
}

func TestIsWindowsBinary_NonExistentFile(t *testing.T) {
	result := IsWindowsBinary("/nonexistent/path/to/binary")
	assert.False(t, result, "IsWindowsBinary should return false for non-existent files")
}

func TestCheckWSLCrossExecution(t *testing.T) {
	if !osinfo.IsLinux() {
		// On non-Linux systems, should always return empty
		result := CheckWSLCrossExecution("/any/path")
		assert.Empty(t, result, "CheckWSLCrossExecution should return empty string on non-Linux systems")
		return
	}

	// On Linux (but not WSL), should return empty for any binary
	// We can't easily test the actual WSL scenario without being in WSL
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testbinary")

	// Create an ELF binary
	elfBinary := []byte{0x7f, 'E', 'L', 'F', 0x01, 0x01, 0x01, 0x00}
	testutil.WriteTestFile(t, tmpFile, elfBinary, utils.PermFileExecutable, "Failed to create test file")

	result := CheckWSLCrossExecution(tmpFile)
	// On regular Linux (not WSL), this should return empty
	// If we're actually in WSL and the binary is ELF, it should also be empty
	t.Logf("CheckWSLCrossExecution result: %q", result)
}

func TestCheckWSLCrossExecution_WindowsBinary(t *testing.T) {
	if !osinfo.IsLinux() {
		t.Skip("This test only runs on Linux")
	}

	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "testbinary.exe")

	// Create a Windows PE binary
	peBinary := []byte{'M', 'Z', 0x90, 0x00}
	testutil.WriteTestFile(t, tmpFile, peBinary, utils.PermFileExecutable, "Failed to create test file")

	result := CheckWSLCrossExecution(tmpFile)

	// If we're in WSL, we should get a warning
	// If we're on regular Linux, we should get empty (not WSL)
	if IsWSL() {
		assert.NotEmpty(t, result, "Expected warning when running Windows binary in WSL")
		// Verify the warning includes the actual host architecture
		assert.Contains(t, result, osinfo.Arch(), "Warning message should include host architecture , but %v %v", osinfo.Arch(), result)
		// Verify it suggests the correct GOOS
		assert.Contains(t, result, "GOOS=linux", "Warning message should suggest GOOS=linux, but %v", result)
		t.Logf("WSL warning message: %s", result)
	}

	assert.False(t, !IsWSL() && result != "", "Should not warn about Windows binary on regular Linux (not WSL)")
}
