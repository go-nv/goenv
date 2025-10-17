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

	// Check for local version file (including go.mod if enabled)
	localFile, err := m.FindVersionFile("")
	if err == nil && localFile != "" {
		// Read version from the found file
		version, err := m.ReadVersionFile(localFile)
		if err == nil && version != "" {
			return version, localFile, nil
		}
	}

	// Check global version file
	globalVersion, err := m.getGlobalVersion()
	if err == nil && globalVersion != "" {
		// Check if a global version file actually exists
		globalFile := m.config.GlobalVersionFile()
		if _, statErr := os.Stat(globalFile); statErr == nil {
			// Global file exists, return its path
			return globalVersion, globalFile, nil
		}

		// Try legacy files
		globalLegacyFile := filepath.Join(m.config.Root, "global")
		if _, statErr := os.Stat(globalLegacyFile); statErr == nil {
			return globalVersion, globalLegacyFile, nil
		}

		defaultFile := filepath.Join(m.config.Root, "default")
		if _, statErr := os.Stat(defaultFile); statErr == nil {
			return globalVersion, defaultFile, nil
		}

		// No file exists, this is a default fallback to "system"
		// Return empty source to indicate default behavior
		if globalVersion == "system" {
			return "system", "", nil
		}

		// Shouldn't reach here, but return the global file path for compatibility
		return globalVersion, globalFile, nil
	}

	return "", "", fmt.Errorf("no version set")
}

// GetLocalVersion reads version from local .go-version file
func (m *Manager) GetLocalVersion() (string, error) {
	return m.readVersionFile(m.findLocalVersionFile())
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
			// Use Fields to split by any whitespace (spaces, tabs, etc.)
			parts := strings.Fields(line)
			if len(parts) == 0 {
				continue
			}

			// Look for "toolchain go1.11.4" first (takes precedence)
			if parts[0] == "toolchain" && len(parts) >= 2 {
				if strings.HasPrefix(parts[1], "go") {
					version := strings.TrimPrefix(parts[1], "go")
					// Validate version string for path traversal attacks
					if err := validateVersionString(version); err != nil {
						continue // Skip invalid versions
					}
					return version, nil
				}
			}
			// Look for "go 1.11" line
			if parts[0] == "go" && len(parts) >= 2 {
				// Validate version string for path traversal attacks
				if err := validateVersionString(parts[1]); err != nil {
					continue // Skip invalid versions
				}
				// Store but continue looking for toolchain
				versions = append(versions, parts[1])
			}
		} else {
			// Regular version file - validate for path traversal attacks
			// This provides defense-in-depth beyond the basic checks
			if err := validateVersionString(line); err != nil {
				continue // Skip invalid versions
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
// Resolves aliases to their target versions before writing
func (m *Manager) SetGlobalVersion(version string) error {
	// Resolve alias if applicable
	resolved, err := m.ResolveAlias(version)
	if err != nil {
		return err
	}
	version = resolved

	if err := m.ValidateVersion(version); err != nil {
		return err
	}

	return m.writeVersionFile(m.config.GlobalVersionFile(), version)
}

// SetLocalVersion sets the local Go version for current directory
// Resolves aliases to their target versions before writing
func (m *Manager) SetLocalVersion(version string) error {
	// Resolve alias if applicable
	resolved, err := m.ResolveAlias(version)
	if err != nil {
		return err
	}
	version = resolved

	if err := m.ValidateVersion(version); err != nil {
		return err
	}

	localFile := filepath.Join(m.workingDir(), m.config.LocalVersionFile())
	return m.writeVersionFile(localFile, version)
}

// validateVersionString checks if a version string is safe from path traversal attacks
// This provides defense-in-depth protection against CVE-2022-35861 and similar vulnerabilities
func validateVersionString(version string) error {
	if version == "" {
		return fmt.Errorf("version string cannot be empty")
	}

	// Allow "system" and "latest" as special cases
	if version == "system" || version == "latest" {
		return nil
	}

	// Check for path traversal attempts
	if strings.Contains(version, "..") {
		return fmt.Errorf("version string contains path traversal (..): %s", version)
	}

	// Check for absolute paths (Unix and Windows)
	if strings.HasPrefix(version, "/") || strings.HasPrefix(version, "\\") {
		return fmt.Errorf("version string cannot be an absolute path: %s", version)
	}

	// Check for Windows drive letters (C:, D:, etc.)
	if len(version) >= 2 && version[1] == ':' && ((version[0] >= 'A' && version[0] <= 'Z') || (version[0] >= 'a' && version[0] <= 'z')) {
		return fmt.Errorf("version string cannot contain drive letter: %s", version)
	}

	// Check for directory separators (should be a simple version string, not a path)
	if strings.Contains(version, "/") || strings.Contains(version, "\\") {
		return fmt.Errorf("version string cannot contain path separators: %s", version)
	}

	// Check for null bytes (path truncation attack)
	if strings.Contains(version, "\x00") {
		return fmt.Errorf("version string contains null byte")
	}

	// Check for hidden files (starting with .)
	if strings.HasPrefix(version, ".") {
		return fmt.Errorf("version string cannot start with dot: %s", version)
	}

	// Check for excessive length (prevent buffer overflow style attacks)
	if len(version) > 255 {
		return fmt.Errorf("version string too long (max 255 characters): %s", version)
	}

	// Check for control characters and spaces (spaces can cause parsing issues)
	for _, ch := range version {
		if ch <= 32 || ch == 127 {
			return fmt.Errorf("version string contains invalid character: %s", version)
		}
	}

	return nil
}

// ResolveVersionSpec resolves a user-provided version specifier to an installed version
// This handles aliases, "latest", "system", and version prefix matching
func (m *Manager) ResolveVersionSpec(spec string) (string, error) {
	if spec == "" {
		return "", fmt.Errorf("goenv: version '%s' not installed", spec)
	}

	// Resolve aliases first
	resolved, err := m.ResolveAlias(spec)
	if err != nil {
		return "", err
	}
	spec = resolved

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
// This also resolves aliases before checking
func (m *Manager) ValidateVersion(version string) error {
	// First validate the version string for path traversal attacks (defense-in-depth)
	if err := validateVersionString(version); err != nil {
		return err
	}

	// Resolve aliases
	resolved, err := m.ResolveAlias(version)
	if err != nil {
		return err
	}
	version = resolved

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

// ListAliases returns all defined aliases as a map of name -> version
func (m *Manager) ListAliases() (map[string]string, error) {
	aliasesFile := m.config.AliasesFile()

	file, err := os.Open(aliasesFile)
	if err != nil {
		if os.IsNotExist(err) {
			return make(map[string]string), nil // No aliases file yet
		}
		return nil, fmt.Errorf("failed to read aliases file: %w", err)
	}
	defer file.Close()

	aliases := make(map[string]string)
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}

		name := strings.TrimSpace(parts[0])
		version := strings.TrimSpace(parts[1])

		if name != "" && version != "" {
			aliases[name] = version
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading aliases file: %w", err)
	}

	return aliases, nil
}

// ResolveAlias resolves an alias to its target version
// Returns the input if it's not an alias
func (m *Manager) ResolveAlias(nameOrVersion string) (string, error) {
	aliases, err := m.ListAliases()
	if err != nil {
		return "", err
	}

	if target, exists := aliases[nameOrVersion]; exists {
		return target, nil
	}

	// Not an alias, return as-is
	return nameOrVersion, nil
}

// SetAlias creates or updates an alias
func (m *Manager) SetAlias(name, version string) error {
	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	if version == "" {
		return fmt.Errorf("alias version cannot be empty")
	}

	// Validate alias name (no special characters, path separators, etc.)
	if err := validateAliasName(name); err != nil {
		return err
	}

	// Validate target version string
	if err := validateVersionString(version); err != nil {
		return fmt.Errorf("invalid alias target: %w", err)
	}

	// Read existing aliases
	aliases, err := m.ListAliases()
	if err != nil {
		return err
	}

	// Update/add the alias
	aliases[name] = version

	// Write back to file
	return m.writeAliasesFile(aliases)
}

// DeleteAlias removes an alias
func (m *Manager) DeleteAlias(name string) error {
	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	// Read existing aliases
	aliases, err := m.ListAliases()
	if err != nil {
		return err
	}

	// Check if alias exists
	if _, exists := aliases[name]; !exists {
		return fmt.Errorf("alias '%s' not found", name)
	}

	// Remove the alias
	delete(aliases, name)

	// Write back to file
	return m.writeAliasesFile(aliases)
}

// writeAliasesFile writes the aliases map to the aliases file
func (m *Manager) writeAliasesFile(aliases map[string]string) error {
	aliasesFile := m.config.AliasesFile()

	// Ensure directory exists
	if err := os.MkdirAll(filepath.Dir(aliasesFile), 0755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	file, err := os.Create(aliasesFile)
	if err != nil {
		return fmt.Errorf("failed to create aliases file: %w", err)
	}
	defer file.Close()

	// Write header comment
	fmt.Fprintln(file, "# goenv aliases")
	fmt.Fprintln(file, "# Format: alias_name=target_version")

	// Write aliases (sorted for deterministic output)
	var names []string
	for name := range aliases {
		names = append(names, name)
	}

	// Simple sort (bubble sort for small lists)
	for i := 0; i < len(names); i++ {
		for j := i + 1; j < len(names); j++ {
			if names[i] > names[j] {
				names[i], names[j] = names[j], names[i]
			}
		}
	}

	for _, name := range names {
		fmt.Fprintf(file, "%s=%s\n", name, aliases[name])
	}

	return nil
}

// validateAliasName checks if an alias name is valid
func validateAliasName(name string) error {
	if name == "" {
		return fmt.Errorf("alias name cannot be empty")
	}

	// Reserve special keywords
	reserved := []string{"system", "latest"}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("alias name '%s' is reserved", name)
		}
	}

	// Check for invalid characters
	if strings.ContainsAny(name, "=/\\:;\"'` \t\n\r") {
		return fmt.Errorf("alias name contains invalid characters")
	}

	// Check for path traversal
	if strings.Contains(name, "..") || strings.HasPrefix(name, ".") {
		return fmt.Errorf("invalid alias name: %s", name)
	}

	// Check length
	if len(name) > 64 {
		return fmt.Errorf("alias name too long (max 64 characters)")
	}

	return nil
}
