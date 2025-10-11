package shims

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
)

// Manager handles shim operations
type ShimManager struct {
	config *config.Config
}

// NewShimManager creates a new shim manager
func NewShimManager(cfg *config.Config) *ShimManager {
	return &ShimManager{config: cfg}
}

// Rehash creates or updates shims for all installed Go versions
func (s *ShimManager) Rehash() error {
	shimsDir := s.config.ShimsDir()

	// Ensure shims directory exists
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		return fmt.Errorf("failed to create shims directory: %w", err)
	}

	// Clear existing shims
	if err := s.clearShims(); err != nil {
		return fmt.Errorf("failed to clear existing shims: %w", err)
	}

	// Get all installed versions
	mgr := manager.NewManager(s.config)
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return fmt.Errorf("failed to list installed versions: %w", err)
	}

	// Collect all unique binaries across versions
	binaries := make(map[string]bool)

	for _, version := range versions {
		versionBinDir := filepath.Join(s.config.VersionsDir(), version, "bin")
		entries, err := os.ReadDir(versionBinDir)
		if err != nil {
			continue // Skip if version doesn't have bin directory
		}

		for _, entry := range entries {
			if !entry.IsDir() && isExecutable(filepath.Join(versionBinDir, entry.Name())) {
				binaries[entry.Name()] = true
			}
		}
	}

	// Create shims for all binaries
	for binary := range binaries {
		if err := s.createShim(binary); err != nil {
			return fmt.Errorf("failed to create shim for %s: %w", binary, err)
		}
	}

	return nil
}

// ListShims returns all available shim files
func (s *ShimManager) ListShims() ([]string, error) {
	shimsDir := s.config.ShimsDir()

	entries, err := os.ReadDir(shimsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, fmt.Errorf("failed to read shims directory: %w", err)
	}

	var shims []string
	for _, entry := range entries {
		if !entry.IsDir() {
			shims = append(shims, entry.Name())
		}
	}

	return shims, nil
}

// WhichBinary returns the full path to the binary that would be executed
func (s *ShimManager) WhichBinary(command string) (string, error) {
	mgr := manager.NewManager(s.config)

	// Get current version
	version, _, err := mgr.GetCurrentVersion()
	if err != nil {
		return "", fmt.Errorf("no version set: %w", err)
	}

	if version == "system" {
		// Use system binary
		return command, nil
	}

	// Look in version's bin directory
	versionBinDir := filepath.Join(s.config.VersionsDir(), version, "bin")
	binaryPath := filepath.Join(versionBinDir, command)

	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath, nil
	}

	return "", fmt.Errorf("command not found: %s", command)
}

// WhenceVersions returns all versions that contain the specified command
func (s *ShimManager) WhenceVersions(command string) ([]string, error) {
	mgr := manager.NewManager(s.config)
	allVersions, err := mgr.ListInstalledVersions()
	if err != nil {
		return nil, fmt.Errorf("failed to list versions: %w", err)
	}

	var versionsWithCommand []string

	for _, version := range allVersions {
		versionBinDir := filepath.Join(s.config.VersionsDir(), version, "bin")
		binaryPath := filepath.Join(versionBinDir, command)

		if _, err := os.Stat(binaryPath); err == nil {
			versionsWithCommand = append(versionsWithCommand, version)
		}
	}

	return versionsWithCommand, nil
}

// clearShims removes all existing shim files
func (s *ShimManager) clearShims() error {
	shimsDir := s.config.ShimsDir()

	entries, err := os.ReadDir(shimsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil // Nothing to clear
		}
		return err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			if err := os.Remove(filepath.Join(shimsDir, entry.Name())); err != nil {
				return err
			}
		}
	}

	return nil
}

// createShim creates a shim file for the specified binary
func (s *ShimManager) createShim(binaryName string) error {
	shimPath := filepath.Join(s.config.ShimsDir(), binaryName)

	// Create the shim script
	shimContent := fmt.Sprintf(`#!/usr/bin/env bash
# goenv shim for %s
set -e
[ -n "$GOENV_DEBUG" ] && set -x

program="${0##*/}"
if [ "$program" = "goenv" ]; then
  case "$1" in
  "" )
    echo "goenv: no command specified" >&2
    exit 1
    ;;
  * )
    exec goenv exec "$@"
    ;;
  esac
else
  exec goenv exec "$program" "$@"
fi
`, binaryName)

	if err := os.WriteFile(shimPath, []byte(shimContent), 0755); err != nil {
		return fmt.Errorf("failed to write shim file: %w", err)
	}

	return nil
}

// isExecutable checks if a file is executable
func isExecutable(path string) bool {
	info, err := os.Stat(path)
	if err != nil {
		return false
	}

	return info.Mode()&0111 != 0 // Check if any execute bit is set
}
