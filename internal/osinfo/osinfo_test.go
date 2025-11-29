package osinfo

import (
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOS(t *testing.T) {
	result := OS()
	assert.NotEmpty(t, result, "OS() should not return empty string")
	assert.Equal(t, runtime.GOOS, result, "OS() =")
}

func TestArch(t *testing.T) {
	result := Arch()
	assert.NotEmpty(t, result, "Arch() should not return empty string")
	assert.Equal(t, runtime.GOARCH, result, "Arch() =")
}

func TestIsWindows(t *testing.T) {
	result := IsWindows()
	expected := runtime.GOOS == "windows"
	assert.Equal(t, expected, result, "IsWindows() =")
}

func TestIsMacOS(t *testing.T) {
	result := IsMacOS()
	expected := runtime.GOOS == "darwin"
	assert.Equal(t, expected, result, "IsMacOS() =")
}

func TestIsLinux(t *testing.T) {
	result := IsLinux()
	expected := runtime.GOOS == "linux"
	assert.Equal(t, expected, result, "IsLinux() =")
}

func TestIsUnix(t *testing.T) {
	result := IsUnix()
	expected := runtime.GOOS != "windows"
	assert.Equal(t, expected, result, "IsUnix() =")
}
