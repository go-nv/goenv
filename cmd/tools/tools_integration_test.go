package tools

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
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
		// Create Go binary (required by ListInstalledVersions)
		goBinDir := filepath.Join(versionPath, "bin")
		gopath := filepath.Join(versionPath, "gopath", "bin")
		if err := utils.EnsureDirWithContext(goBinDir, "create test directory"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if err := utils.EnsureDirWithContext(gopath, "create test directory"); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Create Go binary
		goBin := filepath.Join(goBinDir, "go")
		testutil.WriteTestFile(t, goBin, []byte("#!/bin/sh\necho go version"), utils.PermFileExecutable)
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
			testutil.WriteTestFile(t, filepath.Join(binPath, tool), []byte("fake binary"), utils.PermFileExecutable, "unexpected error")
		}
	}

	// Test 1: Verify ListInstalledVersions
	mgr := manager.NewManager(cfg)
	installedVersions, err := mgr.ListInstalledVersions()
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
		toolList, err := toolspkg.ListForVersion(cfg, version)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Extract tool names
		var toolNames []string
		for _, tool := range toolList {
			toolNames = append(toolNames, tool.Name)
		}

		if !utils.SlicesEqual(expectedTools, toolNames) {
			t.Errorf("Tools for version %s don't match: expected %v, got %v", version, expectedTools, toolNames)
		}
	}

	// Test 3: Collect all unique tools
	allTools := make(map[string]bool)
	for _, version := range versions {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, tool := range toolList {
			allTools[tool.Name] = true
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
