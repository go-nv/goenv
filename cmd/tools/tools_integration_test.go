package tools

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
)

// TestMultiVersionToolManagement is a comprehensive integration test
// that verifies tool management across multiple Go versions
func TestMultiVersionToolManagement(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create 3 Go versions
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		versionPath := filepath.Join(tmpDir, "versions", v)
		goRoot := filepath.Join(versionPath, "go", "bin")
		gopath := filepath.Join(versionPath, "gopath", "bin")
		if err := os.MkdirAll(goRoot, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := os.MkdirAll(gopath, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Install different tools in different versions
	toolInstallations := map[string][]string{
		"1.21.0": {"gopls", "staticcheck"},
		"1.22.0": {"gopls"},
		"1.23.0": {"gopls", "staticcheck", "gofmt"},
	}

	for version, tools := range toolInstallations {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		for _, tool := range tools {
			if err := os.WriteFile(filepath.Join(binPath, tool), []byte("fake binary"), 0755); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
	}

	// Test 1: Verify getInstalledVersions
	installedVersions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(installedVersions) != 3 {
		t.Errorf("expected length %v, got %v", 3, len(installedVersions))
	}
	if !utils.SlicesEqual(versions, installedVersions) {
		t.Errorf("slices not equal: expected %v, got %v", versions, installedVersions)
	}

	// Test 2: Verify tools for each version
	for version, expectedTools := range toolInstallations {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !utils.SlicesEqual(expectedTools, tools) {
			t.Errorf("Tools for version %s don't match: expected %v, got %v", version, expectedTools, tools)
		}
	}

	// Test 3: Collect all unique tools
	allTools := make(map[string]bool)
	for _, version := range versions {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, tool := range tools {
			allTools[tool] = true
		}
	}
	if len(allTools) != 3 {
		t.Errorf("Should have 3 unique tools: expected length %v, got %v", 3, len(allTools))
	}
	if !allTools["gopls"] {
		t.Errorf("expected gopls in all tools")
	}
	if !allTools["staticcheck"] {
		t.Errorf("expected staticcheck in all tools")
	}
	if !allTools["gofmt"] {
		t.Errorf("expected gofmt in all tools")
	}
}
