//go:build windows
// +build windows

package version

// checkPermissions is a no-op on Windows since permissions work differently (ACLs)
// Windows file permissions are managed through Access Control Lists (ACLs) rather than
// Unix-style permission bits. The os.Chmod() function has limited functionality on Windows.
func (c *Cache) checkPermissions() error {
	// On Windows, we trust the default file permissions
	// Future enhancement: could implement ACL checks using golang.org/x/sys/windows
	return nil
}

// ensureSecurePermissions is a no-op on Windows
// Windows file security is managed through ACLs, and os.Chmod() has limited effect.
// The file will inherit ACLs from the parent directory, which is typically secure
// (only accessible by the current user and administrators).
func (c *Cache) ensureSecurePermissions() error {
	// On Windows, file security is managed by the OS through ACLs
	// The default behavior (inheriting parent directory ACLs) is secure enough
	return nil
}
