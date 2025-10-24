package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

func TestDefaultRoot(t *testing.T) {
	// Save original env var
	originalRoot := utils.GoenvEnvVarRoot.UnsafeValue()
	defer utils.GoenvEnvVarRoot.Set(originalRoot)

	// Test with GOENV_ROOT set
	utils.GoenvEnvVarRoot.Set("/custom/root")
	root := DefaultRoot()
	// Normalize paths for cross-platform comparison
	if filepath.ToSlash(root) != "/custom/root" {
		t.Errorf("Expected /custom/root, got %s", root)
	}

	// Test without GOENV_ROOT set
	os.Unsetenv("GOENV_ROOT")
	root = DefaultRoot()
	homeDir, _ := os.UserHomeDir()
	expected := filepath.Join(homeDir, ".goenv")
	if root != expected {
		t.Errorf("Expected %s, got %s", expected, root)
	}
}

func TestConfigLoad(t *testing.T) {
	cfg := Load()

	if cfg.Root == "" {
		t.Error("Expected Root to be set")
	}

	if cfg.CurrentDir == "" {
		t.Error("Expected CurrentDir to be set")
	}
}

func TestConfigDirectories(t *testing.T) {
	cfg := &Config{Root: "/test/goenv"}

	versionsDir := cfg.VersionsDir()
	expected := "/test/goenv/versions"
	// Normalize paths for cross-platform comparison
	if filepath.ToSlash(versionsDir) != expected {
		t.Errorf("Expected %s, got %s", expected, versionsDir)
	}

	shimsDir := cfg.ShimsDir()
	expected = "/test/goenv/shims"
	// Normalize paths for cross-platform comparison
	if filepath.ToSlash(shimsDir) != expected {
		t.Errorf("Expected %s, got %s", expected, shimsDir)
	}

	globalFile := cfg.GlobalVersionFile()
	expected = "/test/goenv/version"
	if globalFile != expected {
		t.Errorf("Expected %s, got %s", expected, globalFile)
	}

	localFile := cfg.LocalVersionFile()
	expected = ".go-version"
	if localFile != expected {
		t.Errorf("Expected %s, got %s", expected, localFile)
	}
}
