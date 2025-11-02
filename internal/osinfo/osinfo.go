// Package osinfo provides zero-dependency operating system and architecture detection.
// This is the foundational package that wraps runtime.GOOS and runtime.GOARCH.
// It has no dependencies on any other goenv packages to avoid import cycles.
package osinfo

import "runtime"

// OS returns the operating system (runtime.GOOS).
// This is the canonical way to check the OS throughout the codebase.
func OS() string {
	return runtime.GOOS
}

// Arch returns the architecture (runtime.GOARCH).
// This is the canonical way to check the architecture throughout the codebase.
func Arch() string {
	return runtime.GOARCH
}

// IsWindows returns true if running on Windows.
func IsWindows() bool {
	return runtime.GOOS == "windows"
}

// IsMacOS returns true if running on macOS/Darwin.
func IsMacOS() bool {
	return runtime.GOOS == "darwin"
}

// IsLinux returns true if running on Linux.
func IsLinux() bool {
	return runtime.GOOS == "linux"
}

// IsUnix returns true if running on a Unix-like system (Linux, macOS, BSD, etc.).
func IsUnix() bool {
	return runtime.GOOS != "windows"
}
