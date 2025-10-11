package build

import (
	"fmt"
	"os"
	"path/filepath"
)

// Builder coordinates the installation process for Go versions.
type Builder struct {
	GOENVRoot     string
	Platform      *Platform
	KeepBuildPath bool
	Verbose       bool
	Debug         bool
}

// NewBuilder creates a new Builder with the given GOENV_ROOT.
func NewBuilder(goenvRoot string) (*Builder, error) {
	if goenvRoot == "" {
		return nil, fmt.Errorf("GOENV_ROOT is not set")
	}

	platform, err := DetectPlatform()
	if err != nil {
		return nil, fmt.Errorf("failed to detect platform: %w", err)
	}

	if !platform.IsSupported() {
		return nil, fmt.Errorf("platform %s is not supported", platform.String())
	}

	return &Builder{
		GOENVRoot: goenvRoot,
		Platform:  platform,
	}, nil
}

// VersionsDir returns the directory where Go versions are installed.
func (b *Builder) VersionsDir() string {
	return filepath.Join(b.GOENVRoot, "versions")
}

// VersionDir returns the installation directory for a specific version.
func (b *Builder) VersionDir(version string) string {
	return filepath.Join(b.VersionsDir(), version)
}

// IsInstalled checks if a version is already installed.
func (b *Builder) IsInstalled(version string) bool {
	versionDir := b.VersionDir(version)
	info, err := os.Stat(versionDir)
	if err != nil {
		return false
	}
	return info.IsDir()
}

// BuildPath returns the temporary build directory path for a version.
func (b *Builder) BuildPath(version string) string {
	return filepath.Join(os.TempDir(), fmt.Sprintf("goenv-build-%s", version))
}

// Install downloads and installs a Go version.
func (b *Builder) Install(version string) error {
	// Check if already installed
	if b.IsInstalled(version) {
		return fmt.Errorf("version %s is already installed", version)
	}

	// Create versions directory if it doesn't exist
	versionsDir := b.VersionsDir()
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		return fmt.Errorf("failed to create versions directory: %w", err)
	}

	if b.Verbose {
		fmt.Printf("Installing Go %s for %s...\n", version, b.Platform.String())
	}

	// TODO: Implement download and extraction
	// This is a placeholder for Phase 3B
	return fmt.Errorf("installation not yet implemented")
}

// Uninstall removes an installed Go version.
func (b *Builder) Uninstall(version string) error {
	// Check if installed
	if !b.IsInstalled(version) {
		return fmt.Errorf("version %s is not installed", version)
	}

	versionDir := b.VersionDir(version)

	if b.Verbose {
		fmt.Printf("Uninstalling Go %s...\n", version)
	}

	// Remove the version directory
	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	if b.Verbose {
		fmt.Printf("Successfully uninstalled Go %s\n", version)
	}

	return nil
}

// ListInstalled returns a list of installed Go versions.
func (b *Builder) ListInstalled() ([]string, error) {
	versionsDir := b.VersionsDir()

	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			versions = append(versions, entry.Name())
		}
	}

	return versions, nil
}
