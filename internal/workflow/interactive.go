package workflow

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/vscode"
)

// InteractiveSetup handles the interactive setup workflow when running goenv with no args
// It discovers the required version, offers to install it, and checks VS Code settings
type InteractiveSetup struct {
	Config       *config.Config
	Manager      *manager.Manager
	Stdout       io.Writer
	Stderr       io.Writer
	Stdin        io.Reader
	WorkingDir   string
	VSCodeUpdate func(version string) error // Callback to update VS Code settings with specific version
}

// WorkflowResult represents the outcome of the interactive setup
type WorkflowResult struct {
	VersionFound       bool
	Version            string
	VersionSource      manager.VersionSource
	VersionInstalled   bool
	InstallRequested   bool // User wants to install
	VSCodeChecked      bool
	VSCodeUpdated      bool
	VSCodeUpdateNeeded bool // VS Code needs updating but user hasn't confirmed yet
}

// Run executes the interactive setup workflow
func (s *InteractiveSetup) Run() (*WorkflowResult, error) {
	result := &WorkflowResult{}

	// Discover version from .go-version or go.mod
	discovered, err := manager.DiscoverVersion(s.WorkingDir)
	if err != nil {
		return nil, fmt.Errorf("failed to discover version: %w", err)
	}

	if discovered == nil {
		// No version found
		return result, nil
	}

	result.VersionFound = true
	result.Version = discovered.Version
	result.VersionSource = discovered.Source

	// Print what we found
	switch discovered.Source {
	case manager.SourceGoVersion:
		fmt.Fprintf(s.Stdout, "Found .go-version: %s\n", discovered.Version)
	case manager.SourceGoMod:
		fmt.Fprintf(s.Stdout, "Found go.mod: %s\n", discovered.Version)
	}

	// Check if version is installed
	versionInstalled := s.Manager.IsVersionInstalled(discovered.Version)
	result.VersionInstalled = versionInstalled

	if versionInstalled {
		fmt.Fprintf(s.Stdout, "%sGo %s is installed\n", utils.Emoji("✓ "), discovered.Version)
	} else {
		// Offer to install
		result.InstallRequested = s.promptInstall(discovered.Version)
		if !result.InstallRequested {
			return result, nil
		}
		// Caller will handle actual installation
		// For now, assume it will be installed
	}

	// Check for version mismatch between .go-version and go.mod
	// This may prompt the user to update .go-version
	s.checkVersionMismatch(discovered)

	// Check VS Code settings if version is or will be installed
	needsUpdate := s.checkVSCodeSettings(discovered.Version, versionInstalled || result.InstallRequested)
	result.VSCodeChecked = true
	result.VSCodeUpdateNeeded = needsUpdate

	if needsUpdate && s.promptVSCodeUpdate() {
		if s.VSCodeUpdate != nil {
			if err := s.VSCodeUpdate(discovered.Version); err != nil {
				fmt.Fprintf(s.Stderr, "Failed to update VS Code settings: %v\n", err)
			} else {
				fmt.Fprintf(s.Stdout, "%sVS Code settings updated\n", utils.Emoji("✓ "))
				result.VSCodeUpdated = true
			}
		}
	}

	return result, nil
}

// promptInstall asks the user if they want to install the version
// Returns true if user wants to install, false otherwise
// The actual installation should be handled by the caller
func (s *InteractiveSetup) promptInstall(version string) bool {
	fmt.Fprintf(s.Stdout, "%sGo %s is not installed\n", utils.Emoji("⚠️  "), version)
	fmt.Fprintf(s.Stdout, "Install now? (Y/n) ")

	response := s.readInput()
	if response == "" || response == "y" || response == "yes" {
		return true
	}

	fmt.Fprintf(s.Stdout, "Skipped installation\n")
	fmt.Fprintf(s.Stdout, "\nTo install later, run: goenv install %s\n", version)
	return false
}

// checkVersionMismatch warns if .go-version and go.mod have different versions
// Returns true if the user chose to update .go-version
func (s *InteractiveSetup) checkVersionMismatch(discoveredVersion *manager.DiscoveredVersion) bool {
	mismatch, goVersionVer, goModVer, err := manager.DiscoverVersionMismatch(s.WorkingDir)
	if err != nil || !mismatch {
		return false
	}

	// If discovery chose go.mod over .go-version (because .go-version was older),
	// prompt the user to update .go-version
	if discoveredVersion.Source == manager.SourceGoMod && goVersionVer != "" {
		fmt.Fprintf(s.Stdout, "\n%sYour .go-version (%s) is older than go.mod's toolchain requirement (%s)\n", utils.Emoji("⚠️  "), goVersionVer, goModVer)
		fmt.Fprintf(s.Stdout, "   Using %s as required by go.mod\n\n", goModVer)

		fmt.Fprintf(s.Stdout, "Update .go-version to %s to avoid this warning? (Y/n) ", goModVer)
		response := s.readInput()

		if response == "" || response == "y" || response == "yes" {
			versionFile := filepath.Join(s.WorkingDir, ".go-version")
			if err := os.WriteFile(versionFile, []byte(goModVer+"\n"), 0644); err != nil {
				fmt.Fprintf(s.Stdout, "%sFailed to update .go-version: %v\n", utils.Emoji("⚠️  "), err)
				return false
			}
			fmt.Fprintf(s.Stdout, "%sUpdated .go-version to %s\n\n", utils.Emoji("✅ "), goModVer)
			return true
		}
		fmt.Fprintln(s.Stdout)
		return false
	}

	// Otherwise just show informational message
	fmt.Fprintf(s.Stdout, "\n%sVersion mismatch detected:\n", utils.Emoji("⚠️  "))
	fmt.Fprintf(s.Stdout, "   .go-version: %s\n", goVersionVer)
	fmt.Fprintf(s.Stdout, "   go.mod:      %s\n", goModVer)
	fmt.Fprintf(s.Stdout, "\n%sConsider updating .go-version to match go.mod:\n", utils.Emoji("💡 "))
	fmt.Fprintf(s.Stdout, "   goenv local %s\n\n", goModVer)
	return false
}

// checkVSCodeSettings checks VS Code settings and returns true if they need updating
func (s *InteractiveSetup) checkVSCodeSettings(version string, versionInstalled bool) bool {
	vscodeSettingsPath := filepath.Join(s.WorkingDir, ".vscode", "settings.json")
	if _, err := os.Stat(vscodeSettingsPath); err != nil {
		// No VS Code settings found
		return false
	}

	if !versionInstalled {
		// Don't check VS Code if version isn't installed
		return false
	}

	result := vscode.CheckSettings(vscodeSettingsPath, version)

	// Treat missing Go settings same as mismatch - offer to configure
	if !result.HasSettings {
		fmt.Fprintf(s.Stdout, "%sVS Code settings found but not configured for goenv\n", utils.Emoji("💡 "))
		return true
	}

	if result.Mismatch {
		if result.ConfiguredVersion != "" {
			fmt.Fprintf(s.Stdout, "%sVS Code settings use Go %s but discovered version is %s\n", utils.Emoji("⚠️  "), result.ConfiguredVersion, version)
		} else {
			fmt.Fprintf(s.Stdout, "%sVS Code settings found but not configured for goenv\n", utils.Emoji("💡 "))
		}
		return true
	}

	// No mismatch - check if using env vars
	if result.UsesEnvVars {
		fmt.Fprintf(s.Stdout, "%sVS Code settings using environment variables\n", utils.Emoji("✓ "))
	} else {
		fmt.Fprintf(s.Stdout, "%sVS Code settings are correct\n", utils.Emoji("✓ "))
	}

	return false
}

// promptVSCodeUpdate asks the user if they want to update VS Code settings
func (s *InteractiveSetup) promptVSCodeUpdate() bool {
	fmt.Fprintf(s.Stdout, "Update VS Code settings? (Y/n) ")
	response := s.readInput()

	if response == "" || response == "y" || response == "yes" {
		return true
	}

	fmt.Fprintf(s.Stdout, "Skipped VS Code update\n")
	return false
}

// readInput reads a line from stdin and returns it trimmed and lowercased
func (s *InteractiveSetup) readInput() string {
	reader := bufio.NewReader(s.Stdin)
	response, err := reader.ReadString('\n')
	if err != nil {
		fmt.Fprintf(s.Stderr, "Error reading input: %v\n", err)
		return ""
	}
	return strings.TrimSpace(strings.ToLower(response))
}

// AutoInstallSetup handles automatic installation workflow (GOENV_AUTO_INSTALL=1)
type AutoInstallSetup struct {
	Config          *config.Config
	Manager         *manager.Manager
	Stdout          io.Writer
	Stderr          io.Writer
	WorkingDir      string
	VSCodeUpdate    func(version string) error // Callback to update VS Code settings
	InstallCallback func(version string) error // Callback to install a version
	AdditionalFlags string                     // GOENV_AUTO_INSTALL_FLAGS
}

// AutoInstallResult represents the outcome of auto-install
type AutoInstallResult struct {
	VersionFound     bool
	Version          string
	VersionSource    manager.VersionSource
	AlreadyInstalled bool
	Installed        bool
	VSCodeConfigured bool
	UsedLatest       bool // True if no version discovered, installed latest
}

// Run executes the auto-install workflow
func (s *AutoInstallSetup) Run() (*AutoInstallResult, error) {
	result := &AutoInstallResult{}

	// Discover version from .go-version or go.mod
	discovered, err := manager.DiscoverVersion(s.WorkingDir)
	if err == nil && discovered != nil {
		// We found a version requirement
		result.VersionFound = true
		result.Version = discovered.Version
		result.VersionSource = discovered.Source

		if !s.Manager.IsVersionInstalled(discovered.Version) {
			// Version not installed - auto-install it
			fmt.Fprintf(s.Stdout, "Auto-installing Go %s (from %s)...\n", discovered.Version, discovered.Source)

			if s.InstallCallback != nil {
				if err := s.InstallCallback(discovered.Version); err != nil {
					return nil, fmt.Errorf("auto-install failed: %w", err)
				}
			}
			result.Installed = true

			// After install, configure VS Code
			if s.VSCodeUpdate != nil {
				if err := s.VSCodeUpdate(discovered.Version); err == nil {
					fmt.Fprintf(s.Stdout, "%sConfigured VS Code for Go %s\n", utils.Emoji("✅ "), discovered.Version)
					result.VSCodeConfigured = true
				}
			}
		} else {
			// Version already installed
			result.AlreadyInstalled = true

			// Just verify VS Code settings are correct
			if s.VSCodeUpdate != nil {
				if err := s.VSCodeUpdate(discovered.Version); err == nil {
					fmt.Fprintf(s.Stdout, "%sGo %s already installed and configured\n", utils.Emoji("✅ "), discovered.Version)
					result.VSCodeConfigured = true
				}
			}
		}
	} else {
		// No version discovered - install latest with any additional flags
		result.UsedLatest = true
		fmt.Fprintf(s.Stdout, "No version file found, installing latest stable version...\n")

		if s.InstallCallback != nil {
			// Pass empty string to indicate "latest"
			if err := s.InstallCallback(""); err != nil {
				return nil, fmt.Errorf("auto-install latest failed: %w", err)
			}
		}
		result.Installed = true
	}

	return result, nil
}
