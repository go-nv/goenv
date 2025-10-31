package manager

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/install"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/utils"
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
	if reason != "" {
		fmt.Printf("\n%s\n", reason)
	}

	return utils.PromptYesNo(utils.PromptConfig{
		Question:            fmt.Sprintf("Go %s is not installed. Install it now?", version),
		DefaultYes:          true,
		NonInteractiveError: fmt.Sprintf("%sError: Go %s is not installed", utils.Emoji("❌ "), version),
		NonInteractiveHelp: []string{
			"To auto-install without prompts, use:",
			fmt.Sprintf("  goenv use %s --yes", version),
			"",
			"Or install manually:",
			fmt.Sprintf("  goenv install %s", version),
		},
	})
}

// PromptForReinstall asks the user if they want to reinstall a corrupted version
func PromptForReinstall(version string) bool {
	fmt.Printf("\n%sGo %s installation is CORRUPTED (missing go binary)\n", utils.Emoji("⚠️  "), version)

	return utils.PromptYesNo(utils.PromptConfig{
		Question:            "Reinstall it now?",
		DefaultYes:          true,
		NonInteractiveError: fmt.Sprintf("%sError: Go %s installation is CORRUPTED (missing go binary)", utils.Emoji("❌ "), version),
		NonInteractiveHelp: []string{
			"To force reinstall without prompts, use:",
			fmt.Sprintf("  goenv install %s --force", version),
			"",
			"Or with goenv use:",
			fmt.Sprintf("  goenv use %s --yes --force", version),
		},
	})
}

// GetInstallHint returns a helpful message about how to install a version
func GetInstallHint(version string, command string) string {
	return fmt.Sprintf("\nTo install: goenv install %s\nThen run:   goenv %s %s", version, command, version)
}

// InstallOptions configures how HandleVersionInstallation behaves
type InstallOptions struct {
	// Config is the goenv configuration
	Config *config.Config

	// Manager is the version manager
	Manager *Manager

	// Version to install/check
	Version string

	// AutoInstall if true, installs without prompting
	AutoInstall bool

	// Force reinstall even if version is already installed
	Force bool

	// Quiet suppresses output messages
	Quiet bool

	// Reason shown to user when prompting for installation (e.g., "Setting global version to X")
	Reason string

	// Writer for standard output
	Writer io.Writer
}

// InstallResult contains the outcome of HandleVersionInstallation
type InstallResult struct {
	// Installed is true if a new installation was performed
	Installed bool

	// Reinstalled is true if an existing version was reinstalled
	Reinstalled bool

	// AlreadyInstalled is true if the version was already installed and no action was needed
	AlreadyInstalled bool

	// Cancelled is true if user declined to install
	Cancelled bool
}

// HandleVersionInstallation checks version status and handles installation/reinstallation
// This consolidates the common pattern across use, global, and local commands
func HandleVersionInstallation(opts InstallOptions) (*InstallResult, error) {
	result := &InstallResult{}

	// Check version status
	status, err := opts.Manager.CheckVersionStatus(opts.Version)
	if err != nil {
		return nil, fmt.Errorf("failed to check version status: %w", err)
	}

	needsReinstall := false

	// Determine what action is needed
	if !status.Installed {
		if !opts.Quiet && opts.Writer != nil {
			fmt.Fprintf(opts.Writer, "%sGo %s is not installed\n", utils.Emoji("❌ "), opts.Version)
		}
	} else if status.Corrupted {
		needsReinstall = true
		if !opts.Quiet && opts.Writer != nil {
			fmt.Fprintf(opts.Writer, "%sGo %s is CORRUPTED (missing go binary)\n", utils.Emoji("⚠️  "), opts.Version)
		}
	} else if opts.Force {
		needsReinstall = true
		if !opts.Quiet && opts.Writer != nil {
			fmt.Fprintf(opts.Writer, "%sForce reinstall requested\n", utils.Emoji("🔄 "))
		}
	} else {
		// Version is already installed and not corrupted
		result.AlreadyInstalled = true
		if !opts.Quiet && opts.Writer != nil {
			fmt.Fprintf(opts.Writer, "%sGo %s is installed\n", utils.Emoji("✅ "), opts.Version)
		}
		return result, nil
	}

	// Determine if we should proceed with installation
	shouldInstall := opts.AutoInstall

	if !shouldInstall {
		// Prompt user
		if needsReinstall {
			shouldInstall = PromptForReinstall(opts.Version)
		} else {
			shouldInstall = PromptForInstall(opts.Version, opts.Reason)
		}
	}

	if !shouldInstall {
		result.Cancelled = true
		return result, fmt.Errorf("installation cancelled")
	}

	// Perform installation
	if !opts.Quiet && opts.Writer != nil {
		if needsReinstall {
			fmt.Fprintf(opts.Writer, "\n%sReinstalling Go %s...\n", utils.Emoji("📦 "), opts.Version)
		} else {
			fmt.Fprintf(opts.Writer, "\n%sInstalling Go %s...\n", utils.Emoji("📦 "), opts.Version)
		}
	}

	// Create and configure installer
	installer := install.NewInstaller(opts.Config)
	installer.Quiet = opts.Quiet
	installer.Verbose = !opts.Quiet

	// Perform the installation
	if err := installer.Install(opts.Version, needsReinstall || opts.Force); err != nil {
		return nil, fmt.Errorf("installation failed: %w", err)
	}

	// Update result
	if needsReinstall {
		result.Reinstalled = true
	} else {
		result.Installed = true
	}

	// Show completion message
	if !opts.Quiet && opts.Writer != nil {
		if needsReinstall {
			fmt.Fprintf(opts.Writer, "%sReinstallation complete\n\n", utils.Emoji("✅ "))
		} else {
			fmt.Fprintf(opts.Writer, "%sInstallation complete\n\n", utils.Emoji("✅ "))
		}
	}

	return result, nil
}
