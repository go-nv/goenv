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
		fmt.Fprintf(s.Stdout, "✓ Go %s is installed\n", discovered.Version)
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
	s.checkVersionMismatch()

	// Check VS Code settings if version is or will be installed
	needsUpdate := s.checkVSCodeSettings(discovered.Version, versionInstalled || result.InstallRequested)
	result.VSCodeChecked = true
	result.VSCodeUpdateNeeded = needsUpdate

	if needsUpdate && s.promptVSCodeUpdate() {
		if s.VSCodeUpdate != nil {
			if err := s.VSCodeUpdate(discovered.Version); err != nil {
				fmt.Fprintf(s.Stderr, "Failed to update VS Code settings: %v\n", err)
			} else {
				fmt.Fprintf(s.Stdout, "✓ VS Code settings updated\n")
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
	fmt.Fprintf(s.Stdout, "⚠️  Go %s is not installed\n", version)
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
func (s *InteractiveSetup) checkVersionMismatch() {
	mismatch, goVersionVer, goModVer, err := manager.DiscoverVersionMismatch(s.WorkingDir)
	if err != nil || !mismatch {
		return
	}

	fmt.Fprintf(s.Stdout, "\n⚠️  Version mismatch detected:\n")
	fmt.Fprintf(s.Stdout, "   .go-version: %s\n", goVersionVer)
	fmt.Fprintf(s.Stdout, "   go.mod:      %s\n", goModVer)
	fmt.Fprintf(s.Stdout, "\n💡 Consider updating .go-version to match go.mod:\n")
	fmt.Fprintf(s.Stdout, "   goenv local %s\n\n", goModVer)
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
		fmt.Fprintf(s.Stdout, "💡 VS Code settings found but not configured for goenv\n")
		return true
	}

	if result.Mismatch {
		if result.ConfiguredVersion != "" {
			fmt.Fprintf(s.Stdout, "⚠️  VS Code settings use Go %s but discovered version is %s\n", result.ConfiguredVersion, version)
		} else {
			fmt.Fprintf(s.Stdout, "💡 VS Code settings found but not configured for goenv\n")
		}
		return true
	}

	// No mismatch - check if using env vars
	if result.UsesEnvVars {
		fmt.Fprintf(s.Stdout, "✓ VS Code settings using environment variables\n")
	} else {
		fmt.Fprintf(s.Stdout, "✓ VS Code settings are correct\n")
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
