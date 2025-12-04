package shims

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/resolver"
	"github.com/go-nv/goenv/internal/utils"
)

// Manager handles shim operations
type ShimManager struct {
	config   *config.Config
	resolver *resolver.Resolver
}

// NewShimManager creates a new shim manager
func NewShimManager(cfg *config.Config) *ShimManager {
	return &ShimManager{
		config:   cfg,
		resolver: resolver.New(cfg),
	}
}

// Rehash creates or updates shims for all installed Go versions
func (s *ShimManager) Rehash() error {
	shimsDir := s.config.ShimsDir()

	// Ensure shims directory exists
	if err := utils.EnsureDirWithContext(shimsDir, "create shims directory"); err != nil {
		return err
	}

	// Clear existing shims
	if err := s.clearShims(); err != nil {
		return errors.FailedTo("clear existing shims", err)
	}

	// Get all installed versions
	mgr := manager.NewManager(s.config)
	versions, err := mgr.ListInstalledVersions()
	if err != nil {
		return errors.FailedTo("list installed versions", err)
	}

	// Collect all unique binaries across versions
	binaries := make(map[string]bool)

	// Scan each version's directories
	for _, version := range versions {
		dirs := s.resolver.GetBinaryDirectories(version)
		versionBinaries, _ := resolver.CollectBinaries(dirs)
		for binary := range versionBinaries {
			binaries[binary] = true
		}
	}

	// Scan host bin directory (tools shared across all versions)
	if s.resolver.ShouldScanGopath() {
		hostDir := s.resolver.GetHostBinaryDirectory()
		hostBinaries, _ := resolver.CollectBinaries([]string{hostDir})
		for binary := range hostBinaries {
			binaries[binary] = true
		}
	}

	// Create shims for all binaries (except goenv itself to prevent recursion)
	for binary := range binaries {
		// Skip creating a shim for goenv itself to prevent infinite recursion
		if binary == "goenv" {
			continue
		}
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
		return nil, errors.FailedTo("read shims directory", err)
	}

	var shims []string
	for _, entry := range entries {
		if !entry.IsDir() {
			name := entry.Name()
			// On Windows, strip executable extensions from shim names
			if utils.IsWindows() {
				for _, ext := range utils.WindowsExecutableExtensions() {
					if strings.HasSuffix(name, ext) {
						name = strings.TrimSuffix(name, ext)
						break
					}
				}
			}
			shims = append(shims, name)
		}
	}

	return shims, nil
}

// WhichBinary returns the full path to the binary that would be executed
func (s *ShimManager) WhichBinary(command string) (string, error) {
	mgr := manager.NewManager(s.config)

	// Get current version and its source
	version, source, err := mgr.GetCurrentVersion()
	if err != nil {
		return "", fmt.Errorf("no version set: %w", err)
	}

	if version == manager.SystemVersion {
		// Use system binary
		return command, nil
	}

	// Use resolver to find the binary (passes source to determine if host bin should be checked)
	return s.resolver.ResolveBinary(command, version, source)
}

// WhenceVersions returns all versions that contain the specified command
func (s *ShimManager) WhenceVersions(command string) ([]string, error) {
	mgr := manager.NewManager(s.config)
	allVersions, err := mgr.ListInstalledVersions()
	if err != nil {
		return nil, errors.FailedTo("list versions", err)
	}

	// Use resolver to find which versions have the binary
	return s.resolver.FindVersionsWithBinary(command, allVersions)
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
	if utils.IsWindows() {
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

# For go commands, detect file arguments
if [[ "$program" = "go"* ]]; then
  for arg; do
    case "$arg" in
    -c* | -- ) break ;;
    */* )
      if [ -f "$arg" ]; then
        export GOENV_FILE_ARG="$arg"
        break
      fi
      ;;
    esac
  done
fi

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

	if err := utils.WriteFileWithContext(shimPath, []byte(shimContent), utils.PermFileExecutable, "write shim file"); err != nil {
		return err
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

REM For go commands, detect file arguments (simplified for batch)
if "%%program:~0,2%%"=="go" (
  for %%%%a in (%%*) do (
    if exist "%%%%a" (
      set "GOENV_FILE_ARG=%%%%a"
      goto :found_file
    )
  )
  :found_file
)

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

	if err := utils.WriteFileWithContext(shimPath, []byte(shimContent), 0666, "write shim file"); err != nil {
		return err
	}

	return nil
}
