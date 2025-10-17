package defaulttools

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config == nil {
		t.Fatal("DefaultConfig returned nil")
	}

	if !config.Enabled {
		t.Error("Expected default config to be enabled")
	}

	if len(config.Tools) == 0 {
		t.Error("Expected default config to have tools")
	}

	// Check for expected default tools
	expectedTools := []string{"gopls", "golangci-lint", "staticcheck", "delve"}
	foundTools := make(map[string]bool)

	for _, tool := range config.Tools {
		foundTools[tool.Name] = true

		// Verify tool has required fields
		if tool.Package == "" {
			t.Errorf("Tool %s has empty package", tool.Name)
		}
	}

	for _, expected := range expectedTools {
		if !foundTools[expected] {
			t.Errorf("Expected default tool %q not found", expected)
		}
	}
}

func TestLoadConfig_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	config, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed for non-existent file: %v", err)
	}

	// Should return default config
	if config == nil {
		t.Fatal("LoadConfig returned nil for non-existent file")
	}

	if !config.Enabled {
		t.Error("Expected default config when file doesn't exist")
	}
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create a custom config
	originalConfig := &Config{
		Enabled: true,
		Tools: []Tool{
			{
				Name:    "gopls",
				Package: "golang.org/x/tools/gopls",
				Version: "@v0.14.0",
				Binary:  "gopls",
			},
			{
				Name:    "custom-tool",
				Package: "example.com/custom",
				Version: "@latest",
			},
		},
	}

	// Save config
	err := SaveConfig(configPath, originalConfig)
	if err != nil {
		t.Fatalf("SaveConfig failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	loadedConfig, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}

	// Verify loaded config matches original
	if loadedConfig.Enabled != originalConfig.Enabled {
		t.Errorf("Enabled mismatch: got %v, want %v", loadedConfig.Enabled, originalConfig.Enabled)
	}

	if len(loadedConfig.Tools) != len(originalConfig.Tools) {
		t.Fatalf("Tools count mismatch: got %d, want %d", len(loadedConfig.Tools), len(originalConfig.Tools))
	}

	for i, tool := range loadedConfig.Tools {
		orig := originalConfig.Tools[i]
		if tool.Name != orig.Name {
			t.Errorf("Tool[%d] Name mismatch: got %q, want %q", i, tool.Name, orig.Name)
		}
		if tool.Package != orig.Package {
			t.Errorf("Tool[%d] Package mismatch: got %q, want %q", i, tool.Package, orig.Package)
		}
		if tool.Version != orig.Version {
			t.Errorf("Tool[%d] Version mismatch: got %q, want %q", i, tool.Version, orig.Version)
		}
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	err := os.WriteFile(configPath, []byte("not: valid: yaml: content"), 0644)
	if err != nil {
		t.Fatalf("Failed to write invalid YAML: %v", err)
	}

	_, err = LoadConfig(configPath)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}

	if !strings.Contains(err.Error(), "failed to parse") {
		t.Errorf("Expected parse error, got: %v", err)
	}
}

func TestConfigPath(t *testing.T) {
	goenvRoot := "/test/goenv"
	path := ConfigPath(goenvRoot)

	expected := filepath.Join(goenvRoot, "default-tools.yaml")
	if path != expected {
		t.Errorf("ConfigPath mismatch: got %q, want %q", path, expected)
	}
}

func TestInstallTools_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled: false,
		Tools: []Tool{
			{Name: "test", Package: "example.com/test"},
		},
	}

	err := InstallTools(config, "1.21.0", tmpDir, true)
	if err != nil {
		t.Errorf("InstallTools should not error when disabled: %v", err)
	}
}

func TestInstallTools_NoTools(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled: true,
		Tools:   []Tool{},
	}

	err := InstallTools(config, "1.21.0", tmpDir, true)
	if err != nil {
		t.Errorf("InstallTools should not error with no tools: %v", err)
	}
}

func TestInstallTools_GoNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled: true,
		Tools: []Tool{
			{Name: "test", Package: "example.com/test"},
		},
	}

	err := InstallTools(config, "1.21.0", tmpDir, false)
	if err == nil {
		t.Error("Expected error when Go binary not found")
	}

	if !strings.Contains(err.Error(), "Go binary not found") {
		t.Errorf("Expected 'Go binary not found' error, got: %v", err)
	}
}

func TestInstallTools_WithMockGo(t *testing.T) {
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	goRoot := filepath.Join(versionPath, "go")
	goBinDir := filepath.Join(goRoot, "bin")
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	if err := os.MkdirAll(goBinDir, 0755); err != nil {
		t.Fatalf("Failed to create go bin directory: %v", err)
	}
	if err := os.MkdirAll(gopathBin, 0755); err != nil {
		t.Fatalf("Failed to create gopath bin directory: %v", err)
	}

	// Create mock go binary that succeeds
	goBinary := filepath.Join(goBinDir, "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	mockScript := "#!/bin/sh\nexit 0"
	if runtime.GOOS == "windows" {
		mockScript = "@echo off\nexit 0"
	}
	if err := os.WriteFile(goBinary, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	config := &Config{
		Enabled: true,
		Tools: []Tool{
			{
				Name:    "test-tool",
				Package: "example.com/test",
				Version: "@latest",
				Binary:  "test-tool",
			},
		},
	}

	err := InstallTools(config, goVersion, tmpDir, false)
	if err != nil {
		t.Errorf("InstallTools failed: %v", err)
	}
}

func TestInstallTools_Failure(t *testing.T) {
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	goRoot := filepath.Join(versionPath, "go")
	goBinDir := filepath.Join(goRoot, "bin")

	if err := os.MkdirAll(goBinDir, 0755); err != nil {
		t.Fatalf("Failed to create go bin directory: %v", err)
	}

	// Create mock go binary that fails
	goBinary := filepath.Join(goBinDir, "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	mockScript := "#!/bin/sh\nexit 1"
	if runtime.GOOS == "windows" {
		mockScript = "@echo off\nexit 1"
	}
	if err := os.WriteFile(goBinary, []byte(mockScript), 0755); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	config := &Config{
		Enabled: true,
		Tools: []Tool{
			{
				Name:    "failing-tool",
				Package: "example.com/fail",
			},
		},
	}

	err := InstallTools(config, goVersion, tmpDir, false)
	if err == nil {
		t.Error("Expected error when tool installation fails")
	}

	if !strings.Contains(err.Error(), "failed to install") {
		t.Errorf("Expected 'failed to install' error, got: %v", err)
	}
}

func TestVerifyTools(t *testing.T) {
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	if err := os.MkdirAll(gopathBin, 0755); err != nil {
		t.Fatalf("Failed to create gopath bin directory: %v", err)
	}

	// Create some tool binaries
	tool1Binary := filepath.Join(gopathBin, "gopls")
	tool2Binary := filepath.Join(gopathBin, "dlv")
	if runtime.GOOS == "windows" {
		tool1Binary += ".exe"
		tool2Binary += ".exe"
	}

	if err := os.WriteFile(tool1Binary, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("Failed to create tool1 binary: %v", err)
	}
	if err := os.WriteFile(tool2Binary, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("Failed to create tool2 binary: %v", err)
	}

	config := &Config{
		Tools: []Tool{
			{
				Name:    "gopls",
				Package: "golang.org/x/tools/gopls",
				Binary:  "gopls",
			},
			{
				Name:    "delve",
				Package: "github.com/go-delve/delve/cmd/dlv",
				Binary:  "dlv",
			},
			{
				Name:    "missing-tool",
				Package: "example.com/missing",
				Binary:  "missing",
			},
		},
	}

	results, err := VerifyTools(config, goVersion, tmpDir)
	if err != nil {
		t.Fatalf("VerifyTools failed: %v", err)
	}

	// Check results
	if !results["gopls"] {
		t.Error("Expected gopls to be found")
	}

	if !results["delve"] {
		t.Error("Expected delve to be found")
	}

	if results["missing-tool"] {
		t.Error("Expected missing-tool to not be found")
	}
}

func TestVerifyTools_NoBinaryName(t *testing.T) {
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	if err := os.MkdirAll(gopathBin, 0755); err != nil {
		t.Fatalf("Failed to create gopath bin directory: %v", err)
	}

	// Create tool binary with name extracted from package path
	toolBinary := filepath.Join(gopathBin, "staticcheck")
	if runtime.GOOS == "windows" {
		toolBinary += ".exe"
	}

	if err := os.WriteFile(toolBinary, []byte("mock binary"), 0755); err != nil {
		t.Fatalf("Failed to create tool binary: %v", err)
	}

	config := &Config{
		Tools: []Tool{
			{
				Name:    "staticcheck",
				Package: "honnef.co/go/tools/cmd/staticcheck",
				// Binary field is empty, should extract "staticcheck" from package
			},
		},
	}

	results, err := VerifyTools(config, goVersion, tmpDir)
	if err != nil {
		t.Fatalf("VerifyTools failed: %v", err)
	}

	if !results["staticcheck"] {
		t.Error("Expected staticcheck to be found (binary name extracted from package)")
	}
}

func TestVerifyTools_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Tools: []Tool{},
	}

	results, err := VerifyTools(config, "1.21.0", tmpDir)
	if err != nil {
		t.Fatalf("VerifyTools failed: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected empty results for empty config, got %d results", len(results))
	}
}
