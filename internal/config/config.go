package config

import (
	"fmt"
	"os"
	"path/filepath"
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
	if root := os.Getenv("GOENV_ROOT"); root != "" {
		return filepath.Clean(root)
	}

	home, err := os.UserHomeDir()
	if err != nil {
		return filepath.Join("/tmp", ".goenv") // fallback
	}

	return filepath.Join(home, ".goenv")
}

// Load loads configuration from environment variables
func Load() *Config {
	cfg := &Config{
		Root:       DefaultRoot(),
		CurrentDir: getCurrentDir(),
		Shell:      os.Getenv("GOENV_SHELL"),
		Debug:      os.Getenv("GOENV_DEBUG") != "",
	}

	return cfg
}

// getCurrentDir gets the current directory for version resolution
func getCurrentDir() string {
	// Check GOENV_DIR first (from shims)
	if dir := os.Getenv("GOENV_DIR"); dir != "" {
		return dir
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

// EnsureDirectories creates necessary directories if they don't exist
func (c *Config) EnsureDirectories() error {
	dirs := []string{
		c.Root,
		c.VersionsDir(),
		c.ShimsDir(),
	}

	for _, dir := range dirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create directory %s: %w", dir, err)
		}
	}

	return nil
}
