package errors

import (
	"fmt"
	"strings"

	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
)

// VersionNotInstalled creates a simple error for when a Go version is not installed.
func VersionNotInstalled(version, source string) error {
	if source != "" {
		return fmt.Errorf("goenv: version '%s' is not installed (set by %s)", version, source)
	}
	return fmt.Errorf("goenv: version '%s' is not installed", version)
}

// VersionNotInstalledDetailed creates an enhanced error message with helpful suggestions
// for when a Go version is not installed. This includes installation instructions and
// suggestions for alternative versions.
func VersionNotInstalledDetailed(version, source string, mgr *manager.Manager) error {
	var sb strings.Builder

	// Main error message
	if source != "" {
		sb.WriteString(fmt.Sprintf("goenv: version '%s' is not installed (set by %s)\n", version, source))
	} else {
		sb.WriteString(fmt.Sprintf("goenv: version '%s' is not installed\n", version))
	}

	sb.WriteString("\n")

	// Suggest installation
	sb.WriteString("To install this version:\n")
	sb.WriteString(fmt.Sprintf("  goenv install %s\n", version))

	sb.WriteString("\n")

	// Suggest alternatives
	sb.WriteString("Or use a different version:\n")

	// Get latest installed version if available
	installed, err := mgr.ListInstalledVersions()
	if err == nil && len(installed) > 0 {
		// Find the latest installed version
		latestInstalled := ""
		for _, v := range installed {
			if latestInstalled == "" || utils.CompareGoVersions(v, latestInstalled) > 0 {
				latestInstalled = v
			}
		}
		if latestInstalled != "" {
			sb.WriteString(fmt.Sprintf("  goenv local %s          # Use %s in this directory\n", latestInstalled, latestInstalled))
		}
	}

	sb.WriteString("  goenv local system          # Use system Go\n")
	sb.WriteString("  goenv local --unset         # Remove .go-version file\n")

	// Return error without the trailing newline
	return fmt.Errorf("%s", strings.TrimRight(sb.String(), "\n"))
}

// CommandNotFound creates an error for when a command is not found in any installed version.
func CommandNotFound(command string) error {
	return fmt.Errorf("goenv: '%s' command not found", command)
}

// ExecutableNotFound creates an error for when an executable is not found.
func ExecutableNotFound(name string) error {
	return fmt.Errorf("executable '%s' not found", name)
}

// FailedTo creates a wrapped error with a "failed to <action>" prefix.
// This provides consistent error formatting for operation failures.
func FailedTo(action string, err error) error {
	return fmt.Errorf("failed to %s: %w", action, err)
}
