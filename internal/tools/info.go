// Package tools provides utilities and operations for managing Go tools across versions.
package tools

import (
	"time"
)

// Tool represents an installed tool binary with comprehensive metadata.
// This is the unified tool representation used throughout goenv.
type Tool struct {
	Name        string    // Binary name (e.g., "goimports", "gopls")
	BinaryPath  string    // Full path to binary
	PackagePath string    // Full package path (e.g., "golang.org/x/tools/cmd/goimports")
	Version     string    // Tool version (e.g., "@latest", "v0.14.1")
	GoVersion   string    // Go version it's installed under (e.g., "1.21.5")
	ModTime     time.Time // Last modified time

	// Extended metadata (populated by tooldetect when available)
	LatestVersion string // Latest available version (if checked)
	IsOutdated    bool   // True if LatestVersion > Version
}

// ToolStatus contains aggregate tool statistics across all Go versions.
type ToolStatus struct {
	ByVersion   map[string][]Tool // Tools grouped by Go version
	AllTools    []Tool            // All tools across versions
	Outdated    []Tool            // Tools with newer versions available
	TotalCount  int               // Total number of tools
	UniqueTools int               // Number of unique tool names
}

// InstallResult contains results of a tool installation operation.
type InstallResult struct {
	Installed []string // Successfully installed tools
	Failed    []string // Failed tool installations
	Errors    []error  // Errors encountered
}

// UninstallResult contains results of a tool uninstallation operation.
type UninstallResult struct {
	Removed []string // Successfully removed tools
	Failed  []string // Failed tool removals
	Errors  []error  // Errors encountered
}
