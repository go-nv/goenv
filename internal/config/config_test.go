package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
)

func TestDefaultRoot(t *testing.T) {
	// Save original env var
	originalRoot := utils.GoenvEnvVarRoot.UnsafeValue()
	defer utils.GoenvEnvVarRoot.Set(originalRoot)

	// Test with GOENV_ROOT set
	utils.GoenvEnvVarRoot.Set("/custom/root")
	root := DefaultRoot()
	// Normalize paths for cross-platform comparison
	assert.Equal(t, "/custom/root", filepath.ToSlash(root), "Expected /custom/root %v", root)

	// Test without GOENV_ROOT set
	t.Setenv(utils.GoenvEnvVarRoot.String(), "")
	root = DefaultRoot()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".goenv")
	assert.Equal(t, expected, root)
}

func TestConfigLoad(t *testing.T) {
	cfg := Load()

	assert.NotEmpty(t, cfg.Root, "Expected Root to be set")

	assert.NotEmpty(t, cfg.CurrentDir, "Expected CurrentDir to be set")
}

func TestConfigDirectories(t *testing.T) {
	cfg := &Config{Root: "/test/goenv"}

	versionsDir := cfg.VersionsDir()
	expected := "/test/goenv/versions"
	// Normalize paths for cross-platform comparison
	assert.Equal(t, expected, filepath.ToSlash(versionsDir))

	shimsDir := cfg.ShimsDir()
	expected = "/test/goenv/shims"
	// Normalize paths for cross-platform comparison
	assert.Equal(t, expected, filepath.ToSlash(shimsDir))

	globalFile := cfg.GlobalVersionFile()
	expected = "/test/goenv/version"
	// Normalize paths for cross-platform comparison
	assert.Equal(t, expected, filepath.ToSlash(globalFile))

	localFile := cfg.LocalVersionFile()
	expected = ".go-version"
	assert.Equal(t, expected, localFile)
}
