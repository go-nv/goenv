package osinfo

import (
	"runtime"
	"testing"
)

func TestOS(t *testing.T) {
	result := OS()
	if result == "" {
		t.Error("OS() should not return empty string")
	}
	if result != runtime.GOOS {
		t.Errorf("OS() = %q, want %q", result, runtime.GOOS)
	}
}

func TestArch(t *testing.T) {
	result := Arch()
	if result == "" {
		t.Error("Arch() should not return empty string")
	}
	if result != runtime.GOARCH {
		t.Errorf("Arch() = %q, want %q", result, runtime.GOARCH)
	}
}

func TestIsWindows(t *testing.T) {
	result := IsWindows()
	expected := runtime.GOOS == "windows"
	if result != expected {
		t.Errorf("IsWindows() = %v, want %v", result, expected)
	}
}

func TestIsMacOS(t *testing.T) {
	result := IsMacOS()
	expected := runtime.GOOS == "darwin"
	if result != expected {
		t.Errorf("IsMacOS() = %v, want %v", result, expected)
	}
}

func TestIsLinux(t *testing.T) {
	result := IsLinux()
	expected := runtime.GOOS == "linux"
	if result != expected {
		t.Errorf("IsLinux() = %v, want %v", result, expected)
	}
}

func TestIsUnix(t *testing.T) {
	result := IsUnix()
	expected := runtime.GOOS != "windows"
	if result != expected {
		t.Errorf("IsUnix() = %v, want %v", result, expected)
	}
}
