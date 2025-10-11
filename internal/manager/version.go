package manager

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
)

// Manager handles version management operations
type Manager struct {
	config *config.Config
}

// UnsetLocalVersion removes the local version file in the current directory, if it exists
func (m *Manager) UnsetLocalVersion() error {
	localFile := filepath.Join(m.workingDir(), m.config.LocalVersionFile())
	if err := os.Remove(localFile); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to unset local version: %w", err)
	}
	return nil
}

// FindVersionFile finds the version file that sets the Go version
// If targetDir is provided, searches from that directory
// Otherwise searches from GOENV_DIR or current directory
// Returns empty string and no error if no local version file is found (defaults to global)
func (m *Manager) FindVersionFile(targetDir string) (string, error) {
	var searchDir string

	if targetDir != "" {
		// Target directory specified
		searchDir = targetDir
	} else {
		// No target - check GOENV_DIR first, then PWD
		searchDir = os.Getenv("GOENV_DIR")
		if searchDir == "" {
			var err error
			searchDir, err = os.Getwd()
			if err != nil {
				searchDir = m.config.CurrentDir
			}
		}
	}

	// Check for go.mod support
	gomodEnabled := os.Getenv("GOENV_GOMOD_VERSION_ENABLE") == "1"
	localFile := m.config.LocalVersionFile()

	// Walk up the directory tree
	currentDir := searchDir
	for {
		// Check for .go-version file
		versionFile := filepath.Join(currentDir, localFile)
		if _, err := os.Stat(versionFile); err == nil {
			return versionFile, nil
		}

		// Check for go.mod if enabled
		if gomodEnabled {
			gomodFile := filepath.Join(currentDir, "go.mod")
			if _, err := os.Stat(gomodFile); err == nil {
				return gomodFile, nil
			}
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached filesystem root
		}
		currentDir = parent
	}

	// If target directory was specified and no file found, return error
	if targetDir != "" {
		return "", fmt.Errorf("no version file found")
	}

	// If no target directory specified, try PWD if different from GOENV_DIR
	if os.Getenv("GOENV_DIR") != "" {
		pwdDir, err := os.Getwd()
		if err == nil && pwdDir != searchDir {
			// Search from PWD
			currentDir := pwdDir
			for {
				versionFile := filepath.Join(currentDir, localFile)
				if _, err := os.Stat(versionFile); err == nil {
					return versionFile, nil
				}

				if gomodEnabled {
					gomodFile := filepath.Join(currentDir, "go.mod")
					if _, err := os.Stat(gomodFile); err == nil {
						return gomodFile, nil
					}
				}

				parent := filepath.Dir(currentDir)
				if parent == currentDir {
					break
				}
				currentDir = parent
			}
		}
	}

	// No local version file found - will use global
	return "", nil
}

// ResolveVersionSpec resolves a version specification to an actual version NewManager creates a new version manager
func NewManager(cfg *config.Config) *Manager {
	return &Manager{
		config: cfg,
	}
}

// ListInstalledVersions returns all installed Go versions
func (m *Manager) ListInstalledVersions() ([]string, error) {
	versionsDir := m.config.VersionsDir()

	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil // No versions installed yet
		}
		return nil, fmt.Errorf("failed to read versions directory: %w", err)
	}

	var versions []string
	for _, entry := range entries {
		if entry.IsDir() {
			// Verify this is a valid Go installation
			goExe := filepath.Join(versionsDir, entry.Name(), "bin", "go")
			if _, err := os.Stat(goExe); err == nil {
				versions = append(versions, entry.Name())
			}
		}
	}

	return versions, nil
}

// GetCurrentVersion returns the currently active Go version
func (m *Manager) GetCurrentVersion() (string, string, error) {
	// Check GOENV_VERSION environment variable first (highest precedence)
	if envVersion := os.Getenv("GOENV_VERSION"); envVersion != "" {
		return envVersion, "GOENV_VERSION environment variable", nil
	}

	// Check local version file
	localVersion, err := m.getLocalVersion()
	if err == nil && localVersion != "" {
		localFile := m.findLocalVersionFile()
		if localFile != "" {
			return localVersion, localFile, nil
		}
	}

	// Check global version file
	globalVersion, err := m.getGlobalVersion()
	if err == nil && globalVersion != "" {
		return globalVersion, "global", nil
	}

	return "", "", fmt.Errorf("no version set")
}

// GetLocalVersion reads version from local .go-version file
func (m *Manager) GetLocalVersion() (string, error) {
	return m.readVersionFile(m.findLocalVersionFile())
}

// getLocalVersion reads version from local .go-version file (internal helper)
func (m *Manager) getLocalVersion() (string, error) {
	return m.GetLocalVersion()
}

// GetGlobalVersion reads version from global version file
func (m *Manager) GetGlobalVersion() (string, error) {
	// Check for version files in order: version, global, default
	globalFile := m.config.GlobalVersionFile()

	// Try primary global version file
	if version, err := m.readVersionFile(globalFile); err == nil {
		return version, nil
	}

	// Try legacy 'global' file
	globalLegacyFile := filepath.Join(m.config.Root, "global")
	if version, err := m.readVersionFile(globalLegacyFile); err == nil {
		return version, nil
	}

	// Try legacy 'default' file
	defaultFile := filepath.Join(m.config.Root, "default")
	if version, err := m.readVersionFile(defaultFile); err == nil {
		return version, nil
	}

	// Default to "system" if no global version is set
	return "system", nil
}

// getGlobalVersion reads version from global version file (internal helper)
func (m *Manager) getGlobalVersion() (string, error) {
	return m.GetGlobalVersion()
}

// findLocalVersionFile walks up the directory tree looking for .go-version file
func (m *Manager) findLocalVersionFile() string {
	// Get current working directory dynamically
	currentDir, err := os.Getwd()
	if err != nil {
		currentDir = m.config.CurrentDir // fallback to config
	}

	localFile := m.config.LocalVersionFile()

	for {
		versionFile := filepath.Join(currentDir, localFile)
		if _, err := os.Stat(versionFile); err == nil {
			return versionFile
		}

		parent := filepath.Dir(currentDir)
		if parent == currentDir {
			break // Reached filesystem root
		}
		currentDir = parent
	}

	return ""
}

// readVersionFile reads a version from a file
func (m *Manager) readVersionFile(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("no version file")
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		version := strings.TrimSpace(scanner.Text())
		if version != "" {
			return version, nil
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	return "", fmt.Errorf("empty version file")
}

// ReadVersionFile reads and parses version(s) from a file (public API for version-file-read command)
// Supports:
// - Regular .go-version files (single or multi-line)
// - go.mod files (extracts Go version from "go X.Y" or "toolchain goX.Y.Z")
// - Multi-line version files (returns colon-separated versions)
// - Skips relative path traversal (lines starting with ".." or containing "./")
// - Trims whitespace and handles various line endings (\n, \r\n, \r)
func (m *Manager) ReadVersionFile(filename string) (string, error) {
	if filename == "" {
		return "", fmt.Errorf("no version file specified")
	}

	file, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer file.Close()

	// Check if it's a go.mod file
	isGoMod := filepath.Base(filename) == "go.mod"

	var versions []string
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()
		// Remove carriage returns and trim whitespace
		line = strings.TrimRight(line, "\r")
		line = strings.TrimSpace(line)

		if line == "" {
			continue
		}

		if isGoMod {
			// Parse go.mod file
			// Look for "toolchain go1.11.4" first (takes precedence)
			if strings.HasPrefix(line, "toolchain ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 && strings.HasPrefix(parts[1], "go") {
					version := strings.TrimPrefix(parts[1], "go")
					return version, nil
				}
			}
			// Look for "go 1.11" line
			if strings.HasPrefix(line, "go ") {
				parts := strings.Fields(line)
				if len(parts) >= 2 {
					// Store but continue looking for toolchain
					versions = append(versions, parts[1])
				}
			}
		} else {
			// Regular version file - skip relative path traversal
			if strings.HasPrefix(line, "..") || strings.Contains(line, "./") {
				continue
			}
			versions = append(versions, line)
		}
	}

	if err := scanner.Err(); err != nil {
		return "", err
	}

	if len(versions) == 0 {
		return "", fmt.Errorf("no version found")
	}

	// Return colon-separated versions
	return strings.Join(versions, ":"), nil
}

// writeVersionFile writes a version to a file
func (m *Manager) writeVersionFile(filename, version string) error {
	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(filename), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(filename)
	if err != nil {
		return fmt.Errorf("failed to create version file: %w", err)
	}
	defer file.Close()

	if _, err := fmt.Fprintln(file, version); err != nil {
		return fmt.Errorf("failed to write version: %w", err)
	}

	return nil
}

// WriteVersionFile writes version(s) to a file (public API for version-file-write command)
// Supports writing single or multiple versions (separated by newlines in the version string)
func (m *Manager) WriteVersionFile(filename, version string) error {
	return m.writeVersionFile(filename, version)
}

// UnsetVersionFile removes a version file (public API for version-file-write command with system)
func (m *Manager) UnsetVersionFile(filename string) error {
	if err := os.Remove(filename); err != nil {
		if os.IsNotExist(err) {
			return nil // Not an error if file doesn't exist
		}
		return fmt.Errorf("failed to remove version file: %w", err)
	}
	return nil
}

// SetGlobalVersion sets the global Go version
func (m *Manager) SetGlobalVersion(version string) error {
	if err := m.ValidateVersion(version); err != nil {
		return err
	}

	return m.writeVersionFile(m.config.GlobalVersionFile(), version)
}

// SetLocalVersion sets the local Go version for current directory
func (m *Manager) SetLocalVersion(version string) error {
	if err := m.ValidateVersion(version); err != nil {
		return err
	}

	localFile := filepath.Join(m.workingDir(), m.config.LocalVersionFile())
	return m.writeVersionFile(localFile, version)
}

// ResolveVersionSpec resolves a user-provided version specifier to an installed version
func (m *Manager) ResolveVersionSpec(spec string) (string, error) {
	if spec == "" {
		return "", fmt.Errorf("goenv: version '%s' not installed", spec)
	}

	if spec == "system" {
		return "system", nil
	}

	installed, err := m.ListInstalledVersions()
	if err != nil {
		return "", err
	}

	if len(installed) == 0 {
		if spec == "latest" {
			return "", fmt.Errorf("goenv: version 'latest' not installed")
		}
		return "", fmt.Errorf("goenv: version '%s' not installed", spec)
	}

	if spec == "latest" {
		resolved := maxVersion(installed)
		if resolved == "" {
			return "", fmt.Errorf("goenv: version 'latest' not installed")
		}
		return resolved, nil
	}

	// Exact match
	for _, version := range installed {
		if version == spec {
			return version, nil
		}
	}

	trimmedSpec := strings.TrimPrefix(spec, "go")
	specParts := strings.Split(trimmedSpec, ".")

	if len(specParts) == 1 {
		// Try matching major version first
		majorMatches := filterVersions(installed, func(v string) bool {
			parts := strings.Split(strings.TrimPrefix(v, "go"), ".")
			return len(parts) > 0 && parts[0] == specParts[0]
		})
		if len(majorMatches) > 0 {
			return maxVersion(majorMatches), nil
		}

		// Fallback to matching minor version anywhere
		minorMatches := filterVersions(installed, func(v string) bool {
			parts := strings.Split(strings.TrimPrefix(v, "go"), ".")
			return len(parts) > 1 && parts[1] == specParts[0]
		})
		if len(minorMatches) > 0 {
			return maxVersion(minorMatches), nil
		}
	} else {
		// Match prefix for major.minor (or longer) specs
		prefix := trimmedSpec + "."
		prefixMatches := filterVersions(installed, func(v string) bool {
			trimmed := strings.TrimPrefix(v, "go")
			return trimmed == trimmedSpec || strings.HasPrefix(trimmed, prefix)
		})
		if len(prefixMatches) > 0 {
			return maxVersion(prefixMatches), nil
		}
	}

	return "", fmt.Errorf("goenv: version '%s' not installed", spec)
}

// ValidateVersion checks if a version is installed or is "system"
func (m *Manager) ValidateVersion(version string) error {
	if version == "system" {
		return nil // "system" is always valid
	}

	versionDir := filepath.Join(m.config.VersionsDir(), version)
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("goenv: version '%s' not installed", version)
	}

	return nil
}

// IsVersionInstalled checks if a version is installed
func (m *Manager) IsVersionInstalled(version string) bool {
	if version == "system" {
		return true
	}

	versionDir := filepath.Join(m.config.VersionsDir(), version)
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return false
	}

	return true
}

// GetVersionPath returns the path to a specific Go version
func (m *Manager) GetVersionPath(version string) (string, error) {
	if version == "system" {
		return "", nil // System version uses default PATH
	}

	versionDir := filepath.Join(m.config.VersionsDir(), version)
	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return "", fmt.Errorf("goenv: version '%s' not installed", version)
	}

	return versionDir, nil
}

// GetGoBinaryPath returns the path to the go binary for a specific version
func (m *Manager) GetGoBinaryPath(version string) (string, error) {
	if version == "system" {
		return "go", nil // Use system go from PATH
	}

	versionPath, err := m.GetVersionPath(version)
	if err != nil {
		return "", err
	}

	return filepath.Join(versionPath, "bin", "go"), nil
}

// workingDir returns the directory to use for local version operations
func (m *Manager) workingDir() string {
	if dir, err := os.Getwd(); err == nil && dir != "" {
		return dir
	}
	if m.config.CurrentDir != "" {
		return m.config.CurrentDir
	}
	return "."
}

func filterVersions(versions []string, predicate func(string) bool) []string {
	var filtered []string
	for _, v := range versions {
		if predicate(v) {
			filtered = append(filtered, v)
		}
	}
	return filtered
}

func maxVersion(versions []string) string {
	if len(versions) == 0 {
		return ""
	}

	max := versions[0]
	for _, v := range versions[1:] {
		if compareGoVersions(v, max) > 0 {
			max = v
		}
	}
	return max
}

func compareGoVersions(v1, v2 string) int {
	v1 = strings.TrimPrefix(v1, "go")
	v2 = strings.TrimPrefix(v2, "go")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		p1 := "0"
		if i < len(parts1) {
			p1 = parts1[i]
		}

		p2 := "0"
		if i < len(parts2) {
			p2 = parts2[i]
		}

		p1HasPre := strings.Contains(p1, "beta") || strings.Contains(p1, "rc")
		p2HasPre := strings.Contains(p2, "beta") || strings.Contains(p2, "rc")

		if p1HasPre && !p2HasPre {
			return -1
		}
		if !p1HasPre && p2HasPre {
			return 1
		}
		if p1HasPre && p2HasPre {
			p1IsRC := strings.Contains(p1, "rc")
			p2IsRC := strings.Contains(p2, "rc")
			if p1IsRC && !p2IsRC {
				return 1
			}
			if !p1IsRC && p2IsRC {
				return -1
			}
		}

		var n1, n2 int
		fmt.Sscanf(p1, "%d", &n1)
		fmt.Sscanf(p2, "%d", &n2)

		if n1 > n2 {
			return 1
		}
		if n1 < n2 {
			return -1
		}
	}

	return 0
}

// HasSystemGo checks if system Go is available in PATH
func (m *Manager) HasSystemGo() bool {
	// Try to find 'go' in PATH
	// This is equivalent to the BATS test helper stub_system_go check
	if goBinary, err := m.GetGoBinaryPath("system"); err == nil {
		// Check if the system go actually exists
		if _, err := os.Stat(goBinary); err == nil {
			return true
		}

		// For system go, GetGoBinaryPath returns "go", so we need to check PATH
		// Simple check using which-like logic
		pathEnv := os.Getenv("PATH")
		if pathEnv == "" {
			return false
		}

		pathDirs := strings.Split(pathEnv, string(os.PathListSeparator))
		for _, dir := range pathDirs {
			if dir == "" {
				continue
			}
			goPath := filepath.Join(dir, "go")
			if _, err := os.Stat(goPath); err == nil {
				return true
			}
		}
	}

	return false
}

// GetSystemGoDir returns the directory containing the system Go binary
func (m *Manager) GetSystemGoDir() (string, error) {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return "", fmt.Errorf("system go not found in PATH")
	}

	pathDirs := strings.Split(pathEnv, string(os.PathListSeparator))
	for _, dir := range pathDirs {
		if dir == "" {
			continue
		}
		goPath := filepath.Join(dir, "go")
		if _, err := os.Stat(goPath); err == nil {
			// Return the parent directory (not the bin dir)
			// For /usr/bin/go, return /usr
			return filepath.Dir(dir), nil
		}
	}

	return "", fmt.Errorf("system go not found in PATH")
}
