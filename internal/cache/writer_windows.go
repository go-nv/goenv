//go:build windows
// +build windows

package cache

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"
)

var (
	modkernel32      = syscall.NewLazyDLL("kernel32.dll")
	procLockFileEx   = modkernel32.NewProc("LockFileEx")
	procUnlockFileEx = modkernel32.NewProc("UnlockFileEx")
)

const (
	// LOCKFILE_EXCLUSIVE_LOCK = 0x00000002
	lockfileExclusiveLock = 2
	// LOCKFILE_FAIL_IMMEDIATELY = 0x00000001
	lockfileFailImmediately = 1
)

// acquireLock acquires an exclusive lock using Windows LockFileEx.
func acquireLock(file *os.File) error {
	// Prepare overlapped structure
	ol := syscall.Overlapped{}

	// LockFileEx parameters
	flags := uint32(lockfileExclusiveLock | lockfileFailImmediately)
	reserved := uint32(0)
	bytesLow := uint32(1)
	bytesHigh := uint32(0)

	// Call LockFileEx
	r1, _, err := procLockFileEx.Call(
		uintptr(file.Fd()),
		uintptr(flags),
		uintptr(reserved),
		uintptr(bytesLow),
		uintptr(bytesHigh),
		uintptr(unsafe.Pointer(&ol)),
	)

	if r1 == 0 {
		return fmt.Errorf("failed to acquire lock: %w", err)
	}

	return nil
}

// releaseLock releases the lock using Windows UnlockFileEx.
func releaseLock(file *os.File) error {
	// Prepare overlapped structure
	ol := syscall.Overlapped{}

	// UnlockFileEx parameters
	reserved := uint32(0)
	bytesLow := uint32(1)
	bytesHigh := uint32(0)

	// Call UnlockFileEx
	r1, _, err := procUnlockFileEx.Call(
		uintptr(file.Fd()),
		uintptr(reserved),
		uintptr(bytesLow),
		uintptr(bytesHigh),
		uintptr(unsafe.Pointer(&ol)),
	)

	if r1 == 0 {
		return fmt.Errorf("failed to release lock: %w", err)
	}

	return nil
}
