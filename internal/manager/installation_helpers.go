package manager

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

// IsVersionCorrupted checks if an installed version is missing its go binary
func (m *Manager) IsVersionCorrupted(version string) bool {
	if version == "system" {
		return false
	}

	versionDir := filepath.Join(m.config.VersionsDir(), version)

	// Check if version directory exists
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return false // Not installed, not corrupted
	}

	// Check for go binary
	goBinaryName := "go"
	if runtime.GOOS == "windows" {
		goBinaryName = "go.exe"
	}
	goBinary := filepath.Join(versionDir, "bin", goBinaryName)

	_, err := os.Stat(goBinary)
	return err != nil // Corrupted if binary doesn't exist
}

// VersionInstallStatus represents the installation status of a version
type VersionInstallStatus struct {
	Installed bool
	Corrupted bool
	Version   string // The resolved/canonical version
}

// CheckVersionStatus checks if a version is installed and not corrupted
func (m *Manager) CheckVersionStatus(version string) (*VersionInstallStatus, error) {
	// Resolve aliases
	resolved, err := m.ResolveAlias(version)
	if err != nil {
		// If alias resolution fails, use original
		resolved = version
	}

	status := &VersionInstallStatus{
		Version:   resolved,
		Installed: m.IsVersionInstalled(resolved),
		Corrupted: false,
	}

	if status.Installed {
		status.Corrupted = m.IsVersionCorrupted(resolved)
	}

	return status, nil
}

// PromptForInstall asks the user if they want to install a version
// Returns true if user wants to install, false otherwise
func PromptForInstall(version string, reason string) bool {
	if reason != "" {
		fmt.Printf("\n%s\n", reason)
	}
	fmt.Printf("Go %s is not installed. Install it now? [Y/n]: ", version)

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "y" || response == "yes"
}

// PromptForReinstall asks the user if they want to reinstall a corrupted version
func PromptForReinstall(version string) bool {
	fmt.Printf("\n⚠️  Go %s installation is CORRUPTED (missing go binary)\n", version)
	fmt.Printf("Reinstall it now? [Y/n]: ")

	reader := bufio.NewReader(os.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	response = strings.TrimSpace(strings.ToLower(response))
	return response == "" || response == "y" || response == "yes"
}

// GetInstallHint returns a helpful message about how to install a version
func GetInstallHint(version string, command string) string {
	return fmt.Sprintf("\nTo install: goenv install %s\nThen run:   goenv %s %s", version, command, version)
}
