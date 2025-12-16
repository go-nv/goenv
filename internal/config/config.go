package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/utils"
)

// Common version file names used by goenv and Go toolchain
const (
	// VersionFileName is the name of the local version file
	VersionFileName = ".go-version"

	// GoModFileName is the name of the Go module file
	GoModFileName = "go.mod"

	// ToolVersionsFileName is the name of the asdf-compatible version file
	ToolVersionsFileName = ".tool-versions"

	// LegacyGlobalFileName is the name of the legacy global version file
	LegacyGlobalFileName = "global"
)

// Config holds goenv configuration
type Config struct {
	Root       string // GOENV_ROOT - where Go versions are installed
	CurrentDir string // GOENV_DIR - current working directory for version resolution
	Shell      string // GOENV_SHELL - shell type for init command
	Debug      bool   // GOENV_DEBUG - debug mode
}

// DefaultRoot returns the default goenv root directory
func DefaultRoot() string {
	if root := utils.GoenvEnvVarRoot.UnsafeValue(); root != "" {
		// Expand tilde and environment variables in GOENV_ROOT
		root = pathutil.ExpandPath(root)
		return filepath.Clean(root)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		// Fallback to OS temp directory (works on Windows, macOS, Linux)
		return filepath.Join(os.TempDir(), ".goenv")
	}

	return filepath.Join(home, ".goenv")
}

// Load loads configuration from environment variables (legacy method).
// Prefer LoadFromEnvironment when using the new context-based approach.
func Load() *Config {
	cfg := &Config{
		Root:       DefaultRoot(),
		CurrentDir: getCurrentDir(),
		Shell:      utils.GoenvEnvVarShell.UnsafeValue(),
		Debug:      utils.GoenvEnvVarDebug.UnsafeValue() != "",
	}

	return cfg
}

// LoadFromEnvironment creates a Config from a parsed GoenvEnvironment struct.
// This is the preferred method when using the context-based approach.
func LoadFromEnvironment(env *utils.GoenvEnvironment) *Config {
	root := env.GetRoot()
	if root == "" {
		root = DefaultRoot()
	} else {
		root = pathutil.ExpandPath(root)
	}

	dir := env.GetDir()
	if dir == "" {
		dir = getCurrentDir()
	} else {
		dir = pathutil.ExpandPath(dir)
	}

	cfg := &Config{
		Root:       root,
		CurrentDir: dir,
		Shell:      env.GetShell(),
		Debug:      env.GetDebug() != "",
	}

	return cfg
}

// getCurrentDir gets the current directory for version resolution
func getCurrentDir() string {
	// Check GOENV_DIR first (from shims)
	if dir := utils.GoenvEnvVarDir.UnsafeValue(); dir != "" {
		// Expand tilde and environment variables in GOENV_DIR
		return pathutil.ExpandPath(dir)
	}

	// Fall back to current working directory
	pwd, err := os.Getwd()
	if err != nil {
		return "/"
	}

	return pwd
}

// VersionsDir returns the directory where Go versions are installed
func (c *Config) VersionsDir() string {
	return filepath.Join(c.Root, "versions")
}

// ShimsDir returns the directory containing shims
func (c *Config) ShimsDir() string {
	return filepath.Join(c.Root, "shims")
}

// GlobalVersionFile returns the path to the global version file
func (c *Config) GlobalVersionFile() string {
	return filepath.Join(c.Root, "version")
}

// LocalVersionFile returns the name of the local version file
func (c *Config) LocalVersionFile() string {
	return VersionFileName
}

// AliasesFile returns the path to the aliases file
func (c *Config) AliasesFile() string {
	return filepath.Join(c.Root, "aliases")
}

// HostsDir returns the directory containing host-specific data
func (c *Config) HostsDir() string {
	return filepath.Join(c.Root, "hosts")
}

// HostDir returns the directory for the current host (GOOS-GOARCH)
// Uses platform.OS() and platform.Arch() to identify the host architecture
func (c *Config) HostDir() string {
	hostOS := platform.OS()
	hostArch := platform.Arch()
	return filepath.Join(c.HostsDir(), hostOS+"-"+hostArch)
}

// HostGopath returns the GOPATH for the current host
func (c *Config) HostGopath() string {
	return filepath.Join(c.HostDir(), "gopath")
}

// HostBinDir returns the bin directory for the current host
func (c *Config) HostBinDir() string {
	return filepath.Join(c.HostGopath(), "bin")
}

// EnsureDirectories creates necessary directories if they don't exist
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Root,
		c.VersionsDir(),
		c.ShimsDir(),
		c.HostDir(),
		c.HostGopath(),
	}

	for _, dir := range dirs {
		if err := utils.EnsureDirWithContext(dir, fmt.Sprintf("create directory %s", dir)); err != nil {
			return err
		}
	}

	return nil
}

func (c *Config) SafeResolvePath(version string) string {
	// check if version directory exists
	versionDir := c.VersionDir(version)

	_, err := os.Stat(versionDir)
	if err == nil {
		return versionDir
	}

	// fallback to host gopath
	return c.HostGopath()
}

// VersionDir returns the installation directory for a specific Go version
// Example: /Users/user/.goenv/versions/1.21.0
func (c *Config) VersionDir(version string) string {
	return filepath.Join(c.VersionsDir(), version)
}

// VersionBinDir returns the bin directory for a specific Go version
// Example: /Users/user/.goenv/versions/1.21.0/bin
func (c *Config) VersionBinDir(version string) string {
	return filepath.Join(c.VersionDir(version), "bin")
}

// VersionGoBinary returns the path to the Go binary for a specific version
// Returns the expected path (doesn't check if it exists)
// Example: /Users/user/.goenv/versions/1.21.0/bin/go[.exe]
func (c *Config) VersionGoBinary(version string) string {
	goBinary := filepath.Join(c.VersionBinDir(version), "go")
	return goBinary
}

// VersionGopathBin returns the gopath/bin directory for a specific version
// This is where tools installed with "go install" are placed
// Example: /Users/user/.goenv/versions/1.21.0/gopath/bin
func (c *Config) VersionGopathBin(version string) string {
	return filepath.Join(c.VersionDir(version), "gopath", "bin")
}

// FindVersionGoBinary returns the path to the Go binary for a specific version,
// handling platform-specific executable extensions (.exe on Windows)
// Returns an error if the binary doesn't exist
func (c *Config) FindVersionGoBinary(version string) (string, error) {
	return pathutil.FindExecutable(c.VersionGoBinary(version))
}

// IsVersionInstalled checks if a specific Go version is installed
func (c *Config) IsVersionInstalled(version string) bool {
	_, err := c.FindVersionGoBinary(version)
	return err == nil
}
