package resolver

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/utils"
)

// Resolver handles binary resolution for goenv
type Resolver struct {
	config *config.Config
}

// New creates a new binary resolver
func New(cfg *config.Config) *Resolver {
	return &Resolver{config: cfg}
}

// ResolveBinary finds the full path to a binary for the given version.
// It searches in order based on version source:
// 1. Version's bin directory (Go binaries like go, gofmt)
// 2. Version-specific GOPATH bin
// 3. Host bin directory (ONLY if using global version, not project-specific)
//
// Returns the full path if found, or an error if not found.
func (r *Resolver) ResolveBinary(command, version, versionSource string) (string, error) {
	versionBinDir := r.config.VersionBinDir(version)
	versionGopathBin := r.config.VersionGopathBin(version)
	gopathDisabled := utils.GoenvEnvVarDisableGopath.IsTrue()

	// Check version bin first
	if path, err := utils.FindExecutable(versionBinDir, command); err == nil {
		return path, nil
	}

	// Check version-specific GOPATH
	if !gopathDisabled {
		if path, err := utils.FindExecutable(versionGopathBin, command); err == nil {
			return path, nil
		}
	}

	// ONLY check host bin if using global version (not project-specific)
	if isGlobalVersion(versionSource) {
		hostBinDir := r.config.HostBinDir()
		if path, err := utils.FindExecutable(hostBinDir, command); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("command not found: %s", command)
}

// isGlobalVersion returns true if the version source indicates global version
func isGlobalVersion(versionSource string) bool {
	// Empty source means default/global behavior
	if versionSource == "" {
		return true
	}
	// Check if source is a global version file (not local .go-version or go.mod)
	return strings.Contains(versionSource, "version") && !strings.Contains(versionSource, ".go-version") && !strings.Contains(versionSource, "go.mod")
}

// FindVersionsWithBinary returns all versions that contain the specified command.
// Only checks version-specific directories (version bin and version GOPATH).
// Host bin is not considered since it's only for global version.
func (r *Resolver) FindVersionsWithBinary(command string, allVersions []string) ([]string, error) {
	gopathDisabled := utils.GoenvEnvVarDisableGopath.IsTrue()

	var versionsWithCommand []string
	for _, version := range allVersions {
		dirs := []string{r.config.VersionBinDir(version)}
		
		if !gopathDisabled {
			dirs = append(dirs, r.config.VersionGopathBin(version))
		}

		if _, err := pathutil.ResolveBinary(command, dirs); err == nil {
			versionsWithCommand = append(versionsWithCommand, version)
		}
	}

	return versionsWithCommand, nil
}

// GetBinaryDirectories returns all directories that should be scanned for binaries
// for the given version. This is used by rehash to discover all available binaries.
func (r *Resolver) GetBinaryDirectories(version string) []string {
	dirs := []string{
		r.config.VersionBinDir(version),
	}

	// Add version-specific GOPATH bin if not disabled
	if !utils.GoenvEnvVarDisableGopath.IsTrue() {
		dirs = append(dirs, r.config.VersionGopathBin(version))
	}

	return dirs
}

// GetHostBinaryDirectory returns the host bin directory that is shared across all versions
func (r *Resolver) GetHostBinaryDirectory() string {
	return r.config.HostBinDir()
}

// ShouldScanGopath returns whether version-specific GOPATH directories should be scanned
func (r *Resolver) ShouldScanGopath() bool {
	return !utils.GoenvEnvVarDisableGopath.IsTrue()
}

// CollectBinaries scans directories and returns a map of unique binary names.
// It handles Windows executable extensions properly.
func CollectBinaries(dirs []string) (map[string]bool, error) {
	binaries := make(map[string]bool)

	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			continue // Skip directories that don't exist or can't be read
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			fullPath := dir + string(os.PathSeparator) + entry.Name()
			if !utils.IsExecutableFile(fullPath) {
				continue
			}

			binaryName := entry.Name()
			
			// On Windows, strip executable extensions from binary name
			if utils.IsWindows() {
				for _, ext := range utils.WindowsExecutableExtensions() {
					if len(binaryName) > len(ext) && binaryName[len(binaryName)-len(ext):] == ext {
						binaryName = binaryName[:len(binaryName)-len(ext)]
						break
					}
				}
			}

			binaries[binaryName] = true
		}
	}

	return binaries, nil
}
