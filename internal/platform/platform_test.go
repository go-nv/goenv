package platform

import (
	"testing"

	"github.com/go-nv/goenv/internal/osinfo"
	"github.com/stretchr/testify/assert"
)

func TestDetect(t *testing.T) {
	info := Detect()

	assert.NotEmpty(t, info.OS, "OS should not be empty")

	assert.NotEmpty(t, info.Arch, "Arch should not be empty")

	// OS should match osinfo.OS()
	assert.Equal(t, osinfo.OS(), info.OS, "OS mismatch")

	// Arch should match osinfo.Arch()
	assert.Equal(t, osinfo.Arch(), info.Arch, "Arch mismatch")
}

func TestIsWindows(t *testing.T) {
	result := IsWindows()
	expected := osinfo.IsWindows()

	assert.Equal(t, expected, result, "IsWindows() =")
}

func TestIsMacOS(t *testing.T) {
	result := IsMacOS()
	expected := osinfo.IsMacOS()

	assert.Equal(t, expected, result, "IsMacOS() =")
}

func TestIsLinux(t *testing.T) {
	result := IsLinux()
	expected := osinfo.IsLinux()

	assert.Equal(t, expected, result, "IsLinux() =")
}

func TestIsUnix(t *testing.T) {
	result := IsUnix()
	expected := !osinfo.IsWindows()

	assert.Equal(t, expected, result, "IsUnix() =")
}

func TestOSName(t *testing.T) {
	name := OSName()

	assert.NotEmpty(t, name, "OSName() should not return empty string")

	// Verify it returns reasonable values
	switch osinfo.OS() {
	case "darwin":
		assert.Equal(t, "macOS", name, "OSName() =")
	case "windows":
		assert.Equal(t, "Windows", name, "OSName() =")
	case "linux":
		assert.Equal(t, "Linux", name, "OSName() =")
	}
}

func TestArchName(t *testing.T) {
	name := ArchName()

	assert.NotEmpty(t, name, "ArchName() should not return empty string")

	// Verify it returns reasonable values
	switch osinfo.Arch() {
	case "amd64":
		assert.Equal(t, "x86_64", name, "ArchName() =")
	case "arm64":
		assert.Equal(t, "ARM64", name, "ArchName() =")
	}
}
