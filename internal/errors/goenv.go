package errors

import (
	"fmt"
	"strings"

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
func VersionNotInstalledDetailed(version, source string, installedVersions []string) error {
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
	if len(installedVersions) > 0 {
		// Find the latest installed version
		latestInstalled := ""
		for _, v := range installedVersions {
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
//
// Note: This function is a re-export from internal/errutil to maintain backward
// compatibility. New code that might create import cycles should import
// internal/errutil directly instead.
func FailedTo(action string, err error) error {
	return fmt.Errorf("failed to %s: %w", action, err)
}

// Tool Management Errors

// ToolNotInstalled creates an error for when a tool is not installed for a version.
func ToolNotInstalled(tool, version string) error {
	return fmt.Errorf("tool '%s' is not installed for Go %s", tool, version)
}

// ToolInstallFailed creates an error for when tool installation fails.
func ToolInstallFailed(tool string, err error) error {
	return fmt.Errorf("failed to install tool '%s': %w", tool, err)
}

// ToolUninstallFailed creates an error for when tool uninstallation fails.
func ToolUninstallFailed(tool string, err error) error {
	return fmt.Errorf("failed to uninstall tool '%s': %w", tool, err)
}

// Cache Management Errors

// CacheCleanFailed creates an error for when cache cleaning fails.
func CacheCleanFailed(cachePath string, err error) error {
	return fmt.Errorf("failed to clean cache at %s: %w", cachePath, err)
}

// CacheMigrationFailed creates an error for when cache migration fails.
func CacheMigrationFailed(err error) error {
	return fmt.Errorf("cache migration failed: %w", err)
}

// Configuration Errors

// InvalidConfig creates an error for invalid configuration.
func InvalidConfig(field string, err error) error {
	return fmt.Errorf("invalid configuration for %s: %w", field, err)
}

// ConfigNotFound creates an error for when configuration file is not found.
func ConfigNotFound(path string) error {
	return fmt.Errorf("configuration file not found: %s", path)
}

// Installation Errors

// DownloadFailed creates an error for when version download fails.
func DownloadFailed(version string, err error) error {
	return fmt.Errorf("failed to download Go %s: %w", version, err)
}

// ExtractionFailed creates an error for when archive extraction fails.
func ExtractionFailed(version string, err error) error {
	return fmt.Errorf("failed to extract Go %s: %w", version, err)
}

// VerificationFailed creates an error for when version verification fails.
func VerificationFailed(version string, reason string) error {
	return fmt.Errorf("verification failed for Go %s: %s", version, reason)
}

// NoVersionsInstalled creates an error for when no Go versions are installed.
func NoVersionsInstalled() error {
	return fmt.Errorf("no Go versions installed")
}

// SystemVersionNotFound creates an error when system Go is not found in PATH.
func SystemVersionNotFound() error {
	return fmt.Errorf("goenv: system version not found in PATH")
}

// PleaseRunManually creates an error when a command must be run manually.
func PleaseRunManually() error {
	return fmt.Errorf("please run the command manually")
}

// SomeVersionsNotInstalled creates an error when some versions are not installed.
func SomeVersionsNotInstalled() error {
	return fmt.Errorf("some versions are not installed")
}

// Version Errors

// InvalidVersion creates an error for invalid version strings.
func InvalidVersion(version string) error {
	return fmt.Errorf("invalid version: %s", version)
}

// AmbiguousVersion creates an error when a version spec matches multiple versions.
func AmbiguousVersion(spec string, matches []string) error {
	return fmt.Errorf("ambiguous version '%s' matches multiple installed versions: %v", spec, matches)
}

// Generic Errors

// InvalidInput creates an error for invalid user input.
func InvalidInput(field, value, reason string) error {
	return fmt.Errorf("invalid %s '%s': %s", field, value, reason)
}

// NotSupported creates an error for unsupported features on a platform.
func NotSupported(feature, platform string) error {
	return fmt.Errorf("%s is not supported on %s", feature, platform)
}

// PermissionDenied creates an error for permission issues.
func PermissionDenied(path string) error {
	return fmt.Errorf("permission denied: %s", path)
}

// AlreadyExists creates an error for when something already exists.
func AlreadyExists(item string) error {
	return fmt.Errorf("%s already exists", item)
}

// NotFound creates a generic not found error.
func NotFound(item string) error {
	return fmt.Errorf("%s not found", item)
}

// Profile Management Errors

// ProfileModificationFailed creates an error for when profile modification fails.
func ProfileModificationFailed(action string, err error) error {
	return fmt.Errorf("failed to %s profile: %w", action, err)
}

// ProfileNotInitialized creates an error for when profile is not initialized.
func ProfileNotInitialized(profilePath string) error {
	return fmt.Errorf("profile %s does not contain goenv initialization", profilePath)
}

// ProfileAlreadyInitialized creates an error for when profile is already initialized.
func ProfileAlreadyInitialized(profilePath string) error {
	return fmt.Errorf("profile %s already contains goenv initialization", profilePath)
}
