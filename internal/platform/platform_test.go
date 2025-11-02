package platform

import (
	"github.com/go-nv/goenv/internal/osinfo"
	"testing"
)

func TestDetect(t *testing.T) {
	info := Detect()

	if info.OS == "" {
		t.Error("OS should not be empty")
	}

	if info.Arch == "" {
		t.Error("Arch should not be empty")
	}

	// OS should match osinfo.OS()
	if info.OS != osinfo.OS() {
		t.Errorf("OS mismatch: got %s, want %s", info.OS, osinfo.OS())
	}

	// Arch should match osinfo.Arch()
	if info.Arch != osinfo.Arch() {
		t.Errorf("Arch mismatch: got %s, want %s", info.Arch, osinfo.Arch())
	}
}

func TestIsWindows(t *testing.T) {
	result := IsWindows()
	expected := osinfo.IsWindows()

	if result != expected {
		t.Errorf("IsWindows() = %v, want %v", result, expected)
	}
}

func TestIsMacOS(t *testing.T) {
	result := IsMacOS()
	expected := osinfo.IsMacOS()

	if result != expected {
		t.Errorf("IsMacOS() = %v, want %v", result, expected)
	}
}

func TestIsLinux(t *testing.T) {
	result := IsLinux()
	expected := osinfo.IsLinux()

	if result != expected {
		t.Errorf("IsLinux() = %v, want %v", result, expected)
	}
}

func TestIsUnix(t *testing.T) {
	result := IsUnix()
	expected := !osinfo.IsWindows()

	if result != expected {
		t.Errorf("IsUnix() = %v, want %v", result, expected)
	}
}

func TestOSName(t *testing.T) {
	name := OSName()

	if name == "" {
		t.Error("OSName() should not return empty string")
	}

	// Verify it returns reasonable values
	switch osinfo.OS() {
	case "darwin":
		if name != "macOS" {
			t.Errorf("OSName() = %s, want macOS", name)
		}
	case "windows":
		if name != "Windows" {
			t.Errorf("OSName() = %s, want Windows", name)
		}
	case "linux":
		if name != "Linux" {
			t.Errorf("OSName() = %s, want Linux", name)
		}
	}
}

func TestArchName(t *testing.T) {
	name := ArchName()

	if name == "" {
		t.Error("ArchName() should not return empty string")
	}

	// Verify it returns reasonable values
	switch osinfo.Arch() {
	case "amd64":
		if name != "x86_64" {
			t.Errorf("ArchName() = %s, want x86_64", name)
		}
	case "arm64":
		if name != "ARM64" {
			t.Errorf("ArchName() = %s, want ARM64", name)
		}
	}
}
