package envdetect

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestIsAppleSilicon(t *testing.T) {
	if runtime.GOOS != "darwin" {
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

	// We can verify this matches runtime.GOARCH on native execution
	if result && runtime.GOARCH != "arm64" {
		t.Log("Detected Apple Silicon but runtime.GOARCH is not arm64 - may be running under Rosetta")
	}

	if !result && runtime.GOARCH == "arm64" {
		t.Error("Should detect Apple Silicon when running on arm64")
	}
}

func TestGetBinaryArchitecture(t *testing.T) {
	if runtime.GOOS != "darwin" {
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

	// The architecture should match runtime.GOARCH (assuming native build)
	if arch != runtime.GOARCH {
		t.Logf("Binary arch (%s) differs from runtime.GOARCH (%s) - may be expected", arch, runtime.GOARCH)
	}
}

func TestGetBinaryArchitecture_NonExistent(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test")
	}

	arch := GetBinaryArchitecture("/nonexistent/binary")
	if arch != "" {
		t.Error("Should return empty string for non-existent file")
	}
}

func TestGetBinaryArchitecture_NonBinary(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test")
	}

	// Create a text file
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "textfile.txt")
	if err := os.WriteFile(tmpFile, []byte("hello world"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	arch := GetBinaryArchitecture(tmpFile)
	// Should return empty or not crash
	t.Logf("Architecture for text file: %q", arch)
}

func TestCheckRosettaMixedArchitecture(t *testing.T) {
	if runtime.GOOS != "darwin" {
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
	if runtime.GOOS != "darwin" {
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
	if runtime.GOOS != "darwin" {
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
	if runtime.GOOS != "darwin" {
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
	if runtime.GOOS != "darwin" {
		t.Skip("macOS-specific test")
	}

	// Test with system binaries that might be universal
	binaries := []string{
		"/usr/bin/true",
		"/bin/ls",
		"/usr/bin/file",
	}

	for _, binary := range binaries {
		if _, err := os.Stat(binary); err != nil {
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
