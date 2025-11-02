package envdetect

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/osinfo"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
)

func TestIsAppleSilicon(t *testing.T) {
	if !osinfo.IsMacOS() {
		// Should return false on non-macOS
		result := IsAppleSilicon()
		if result {
			t.Error("IsAppleSilicon should return false on non-macOS systems")
		}
		return
	}

	// On macOS, check if we can detect Apple Silicon
	result := IsAppleSilicon()
	t.Logf("IsAppleSilicon returned: %v", result)

	// We can verify this matches osinfo.Arch() on native execution
	if result && osinfo.Arch() != "arm64" {
		t.Log("Detected Apple Silicon but osinfo.Arch() is not arm64 - may be running under Rosetta")
	}

	if !result && osinfo.Arch() == "arm64" {
		t.Error("Should detect Apple Silicon when running on arm64")
	}
}

func TestGetBinaryArchitecture(t *testing.T) {
	if !osinfo.IsMacOS() {
		// Should return empty on non-macOS
		result := GetBinaryArchitecture("/any/path")
		if result != "" {
			t.Error("GetBinaryArchitecture should return empty string on non-macOS systems")
		}
		return
	}

	// Test with the current executable
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}

	arch := GetBinaryArchitecture(exe)
	if arch == "" {
		t.Error("Should detect architecture for current executable")
	}

	t.Logf("Current executable architecture: %s", arch)

	// The architecture should match osinfo.Arch() (assuming native build)
	if arch != osinfo.Arch() {
		t.Logf("Binary arch (%s) differs from osinfo.Arch() (%s) - may be expected", arch, osinfo.Arch())
	}
}

func TestGetBinaryArchitecture_NonExistent(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("macOS-specific test")
	}

	arch := GetBinaryArchitecture("/nonexistent/binary")
	if arch != "" {
		t.Error("Should return empty string for non-existent file")
	}
}

func TestGetBinaryArchitecture_NonBinary(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("macOS-specific test")
	}

	// Create a text file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "textfile.txt")
	testutil.WriteTestFile(t, tmpFile, []byte("hello world"), utils.PermFileDefault, "Failed to create test file")

	arch := GetBinaryArchitecture(tmpFile)
	// Should return empty or not crash
	t.Logf("Architecture for text file: %q", arch)
}

func TestCheckRosettaMixedArchitecture(t *testing.T) {
	if !osinfo.IsMacOS() {
		// Should return empty on non-macOS
		result := CheckRosettaMixedArchitecture("/any/path")
		if result != "" {
			t.Error("CheckRosettaMixedArchitecture should return empty string on non-macOS")
		}
		return
	}

	if !IsAppleSilicon() {
		t.Skip("Not on Apple Silicon - test requires Apple Silicon Mac")
	}

	// Get current executable
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}

	// Check with itself - should not warn if both are same architecture
	result := CheckRosettaMixedArchitecture(exe)

	goenvArch := GetBinaryArchitecture(exe)
	t.Logf("Goenv architecture: %s", goenvArch)
	t.Logf("Check result: %q", result)

	// If both goenv and tool are the same, should not warn
	// (unless both are x86_64, which gets an info message)
	if result != "" {
		t.Logf("Received warning/info: %s", result)
	}
}

func TestCheckRosettaMixedArchitecture_DifferentArchs(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("macOS-specific test")
	}

	if !IsAppleSilicon() {
		t.Skip("Not on Apple Silicon")
	}

	// We can't easily create binaries of different architectures in a test,
	// but we can test the logic with /usr/bin/true which is usually universal
	result := CheckRosettaMixedArchitecture("/usr/bin/true")

	// Should not error out
	t.Logf("Check result for /usr/bin/true: %q", result)
}

func TestCheckRosettaMixedArchitecture_NonExistent(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("macOS-specific test")
	}

	// Should handle non-existent files gracefully
	result := CheckRosettaMixedArchitecture("/nonexistent/tool")

	// Should return empty (can't determine architecture)
	if result != "" {
		t.Errorf("Should return empty for non-existent file, got: %q", result)
	}
}

func TestCheckRosettaMixedArchitecture_IntelMac(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("macOS-specific test")
	}

	if IsAppleSilicon() {
		t.Skip("Test is for Intel Macs only")
	}

	// On Intel Mac, should always return empty
	exe, err := os.Executable()
	if err != nil {
		t.Fatalf("Failed to get executable path: %v", err)
	}

	result := CheckRosettaMixedArchitecture(exe)
	if result != "" {
		t.Errorf("Should not warn on Intel Mac, got: %q", result)
	}
}

func TestGetBinaryArchitecture_MultiArchBinary(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("macOS-specific test")
	}

	// Test with system binaries that might be universal
	binaries := []string{
		"/usr/bin/true",
		"/bin/ls",
		"/usr/bin/file",
	}

	for _, binary := range binaries {
		if utils.FileNotExists(binary) {
			continue // Binary doesn't exist, skip
		}

		arch := GetBinaryArchitecture(binary)
		t.Logf("%s architecture: %s", binary, arch)

		// Should detect at least one architecture
		// (universal binaries might show both, or just one depending on 'file' output)
		if arch == "" {
			t.Logf("Warning: Could not detect architecture for %s", binary)
		}
	}
}
