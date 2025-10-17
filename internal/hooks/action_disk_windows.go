//go:build windows
// +build windows

package hooks

import (
	"fmt"
	"syscall"
	"unsafe"
)

var (
	kernel32         = syscall.NewLazyDLL("kernel32.dll")
	getDiskFreeSpace = kernel32.NewProc("GetDiskFreeSpaceExW")
)

// getDiskSpace returns free and total disk space in MB for the given path (Windows)
func getDiskSpace(path string) (freeMB, totalMB int64, err error) {
	// Convert path to UTF16 for Windows API
	pathPtr, err := syscall.UTF16PtrFromString(path)
	if err != nil {
		return 0, 0, fmt.Errorf("invalid path: %w", err)
	}

	var freeBytesAvailable, totalBytes, totalFreeBytes int64

	// Call GetDiskFreeSpaceEx
	r1, _, err := getDiskFreeSpace.Call(
		uintptr(unsafe.Pointer(pathPtr)),
		uintptr(unsafe.Pointer(&freeBytesAvailable)),
		uintptr(unsafe.Pointer(&totalBytes)),
		uintptr(unsafe.Pointer(&totalFreeBytes)),
	)

	if r1 == 0 {
		return 0, 0, fmt.Errorf("failed to get disk space: %w", err)
	}

	// Convert to MB
	freeMB = freeBytesAvailable / (1024 * 1024)
	totalMB = totalBytes / (1024 * 1024)

	return freeMB, totalMB, nil
}
