package hooks

// HookPoint represents a valid hook execution point in the goenv lifecycle
type HookPoint string

// Standard hook points available in goenv
const (
	// PreInstall runs before installing a Go version
	PreInstall HookPoint = "pre_install"

	// PostInstall runs after installing a Go version
	PostInstall HookPoint = "post_install"

	// PreUninstall runs before uninstalling a Go version
	PreUninstall HookPoint = "pre_uninstall"

	// PostUninstall runs after uninstalling a Go version
	PostUninstall HookPoint = "post_uninstall"

	// PreExec runs before executing a Go command
	PreExec HookPoint = "pre_exec"

	// PostExec runs after executing a Go command
	PostExec HookPoint = "post_exec"

	// PreRehash runs before regenerating shims
	PreRehash HookPoint = "pre_rehash"

	// PostRehash runs after regenerating shims
	PostRehash HookPoint = "post_rehash"
)

// String returns the string representation of the hook point
func (h HookPoint) String() string {
	return string(h)
}

// AllHookPoints returns all valid hook points
func AllHookPoints() []HookPoint {
	return []HookPoint{
		PreInstall,
		PostInstall,
		PreUninstall,
		PostUninstall,
		PreExec,
		PostExec,
		PreRehash,
		PostRehash,
	}
}

// IsValid checks if a string represents a valid hook point
func IsValidHookPoint(s string) bool {
	for _, hp := range AllHookPoints() {
		if string(hp) == s {
			return true
		}
	}
	return false
}
