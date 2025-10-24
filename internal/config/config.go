package config

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"

	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/utils"
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

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		Root:       DefaultRoot(),
		CurrentDir: getCurrentDir(),
		Shell:      utils.GoenvEnvVarShell.UnsafeValue(),
		Debug:      utils.GoenvEnvVarDebug.UnsafeValue() != "",
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
	return ".go-version"
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
// Uses runtime.GOOS and runtime.GOARCH to identify the host architecture
func (c *Config) HostDir() string {
	hostOS := runtime.GOOS
	hostArch := runtime.GOARCH
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
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
