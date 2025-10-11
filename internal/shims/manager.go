package shims

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

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
				binaryName := entry.Name()
				// On Windows, strip .exe extension from shim name
				if runtime.GOOS == "windows" && filepath.Ext(binaryName) == ".exe" {
					binaryName = binaryName[:len(binaryName)-4]
				}
				binaries[binaryName] = true
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
	binaryPath := s.findBinary(versionBinDir, command)

	if binaryPath == "" {
		return "", fmt.Errorf("command not found: %s", command)
	}

	return binaryPath, nil
}

// findBinary searches for a binary, adding .exe on Windows if needed
func (s *ShimManager) findBinary(binDir, command string) string {
	// Try exact name first
	binaryPath := filepath.Join(binDir, command)
	if _, err := os.Stat(binaryPath); err == nil {
		return binaryPath
	}

	// On Windows, try adding .exe extension
	if runtime.GOOS == "windows" {
		exePath := filepath.Join(binDir, command+".exe")
		if _, err := os.Stat(exePath); err == nil {
			return exePath
		}
	}

	return ""
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
		binaryPath := s.findBinary(versionBinDir, command)

		if binaryPath != "" {
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
	if runtime.GOOS == "windows" {
		return s.createWindowsShim(binaryName)
	}
	return s.createUnixShim(binaryName)
}

// createUnixShim creates a Unix/bash shim script
func (s *ShimManager) createUnixShim(binaryName string) error {
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

// createWindowsShim creates a Windows batch file shim
func (s *ShimManager) createWindowsShim(binaryName string) error {
	shimPath := filepath.Join(s.config.ShimsDir(), binaryName+".bat")

	// Create the batch file shim
	shimContent := fmt.Sprintf(`@echo off
REM goenv shim for %s
setlocal

if "%%GOENV_DEBUG%%"=="1" (
  echo on
)

REM Get the script name without path
for %%%%I in ("%%~f0") do set "program=%%%%~nI"

if "%%program%%"=="goenv" (
  if "%%1"=="" (
    echo goenv: no command specified >&2
    exit /b 1
  )
  goenv exec %%*
) else (
  goenv exec "%%program%%" %%*
)
`, binaryName)

	if err := os.WriteFile(shimPath, []byte(shimContent), 0666); err != nil {
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

	// On Windows, check file extension
	if runtime.GOOS == "windows" {
		ext := filepath.Ext(path)
		return ext == ".exe" || ext == ".bat" || ext == ".cmd" || ext == ".com"
	}

	// On Unix, check execute bit
	return info.Mode()&0111 != 0
}
