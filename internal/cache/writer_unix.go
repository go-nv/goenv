//go:build !windows
// +build !windows

package cache

import (
	"os"
	"syscall"
)

// acquireLock acquires an exclusive advisory lock on the file.
// Uses flock on Unix-like systems (Linux, macOS, BSD).
func acquireLock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_EX|syscall.LOCK_NB)
}

// releaseLock releases the advisory lock on the file.
func releaseLock(file *os.File) error {
	return syscall.Flock(int(file.Fd()), syscall.LOCK_UN)
}
