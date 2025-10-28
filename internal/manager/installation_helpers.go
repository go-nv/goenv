package manager

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/utils"
	"golang.org/x/term"
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

	// Check for go binary (handles .exe and .bat on Windows)
	goBinaryBase := filepath.Join(versionDir, "bin", "go")
	_, err := pathutil.FindExecutable(goBinaryBase)
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
	// Check if stdin is a TTY (interactive terminal)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-interactive environment (CI/CD, piped input, etc.)
		fmt.Fprintf(os.Stderr, "\n%sError: Go %s is not installed\n", utils.Emoji("❌ "), version)
		fmt.Fprintf(os.Stderr, "\nRunning in non-interactive mode. Cannot prompt for installation.\n")
		fmt.Fprintf(os.Stderr, "To auto-install without prompts, use:\n")
		fmt.Fprintf(os.Stderr, "  goenv use %s --yes\n", version)
		fmt.Fprintf(os.Stderr, "\nOr install manually:\n")
		fmt.Fprintf(os.Stderr, "  goenv install %s\n\n", version)
		return false
	}

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
	// Check if stdin is a TTY (interactive terminal)
	if !term.IsTerminal(int(os.Stdin.Fd())) {
		// Non-interactive environment (CI/CD, piped input, etc.)
		fmt.Fprintf(os.Stderr, "\n%sError: Go %s installation is CORRUPTED (missing go binary)\n", utils.Emoji("❌ "), version)
		fmt.Fprintf(os.Stderr, "\nRunning in non-interactive mode. Cannot prompt for reinstallation.\n")
		fmt.Fprintf(os.Stderr, "To force reinstall without prompts, use:\n")
		fmt.Fprintf(os.Stderr, "  goenv install %s --force\n", version)
		fmt.Fprintf(os.Stderr, "\nOr with goenv use:\n")
		fmt.Fprintf(os.Stderr, "  goenv use %s --yes --force\n\n", version)
		return false
	}

	fmt.Printf("\n%sGo %s installation is CORRUPTED (missing go binary)\n", utils.Emoji("⚠️  "), version)
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
