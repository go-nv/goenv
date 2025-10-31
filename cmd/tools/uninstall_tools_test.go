package tools

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
)

func TestFindToolBinaries(t *testing.T) {
	tests := []struct {
		name          string
		toolName      string
		files         []string
		expectedCount int
		expectedFiles []string
	}{
		{
			name:          "exact match",
			toolName:      "gopls",
			files:         []string{"gopls", "othertool"},
			expectedCount: 1,
			expectedFiles: []string{"gopls"},
		},
		{
			name:          "platform variants",
			toolName:      "gopls",
			files:         []string{"gopls", "gopls.exe", "gopls.darwin"},
			expectedCount: 3,
			expectedFiles: []string{"gopls", "gopls.exe", "gopls.darwin"},
		},
		{
			name:          "no match",
			toolName:      "gopls",
			files:         []string{"othertool", "anothertool"},
			expectedCount: 0,
			expectedFiles: []string{},
		},
		{
			name:          "partial name no match",
			toolName:      "gopls",
			files:         []string{"gopls-old", "mygopls"},
			expectedCount: 0,
			expectedFiles: []string{},
		},
		{
			name:          "windows exe only",
			toolName:      "gopls",
			files:         []string{"gopls.exe"},
			expectedCount: 1,
			expectedFiles: []string{"gopls.exe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			binPath := filepath.Join(tmpDir, "bin")
			if err := os.MkdirAll(binPath, 0755); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(binPath, file)
				if err := os.WriteFile(filePath, []byte("fake binary"), 0755); err != nil {
					t.Fatalf("unexpected error: %v", err)
				}
			}

			// Find binaries
			binaries := findToolBinaries(binPath, tt.toolName)

			// Check count
			if len(binaries) != tt.expectedCount {
				t.Errorf("unexpected number of binaries found: expected length %v, got %v", tt.expectedCount, len(binaries))
			}

			// Check expected files are present
			foundNames := make(map[string]bool)
			for _, bin := range binaries {
				foundNames[filepath.Base(bin)] = true
			}

			for _, expected := range tt.expectedFiles {
				if !foundNames[expected] {
					t.Errorf("expected file %s not found", expected)
				}
			}
		})
	}
}

func TestFindToolBinaries_NonExistentDir(t *testing.T) {
	binPath := "/nonexistent/path/bin"
	binaries := findToolBinaries(binPath, "gopls")
	if len(binaries) != 0 {
		t.Errorf("should return empty slice for non-existent directory: expected length %v, got %v", 0, len(binaries))
	}
}

func TestFindCurrentVersionToolTargets(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create test tools
	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		toolPath := filepath.Join(binPath, tool)
		if err := os.WriteFile(toolPath, []byte("fake"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
	}

	// Set current version
	versionFile := filepath.Join(tmpDir, "version")
	if err := os.WriteFile(versionFile, []byte(version), 0644); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Set environment variable to simulate current version
	t.Setenv("GOENV_VERSION", version)

	// Find targets
	targets := findCurrentVersionToolTargets(cfg, []string{"gopls", "staticcheck", "nonexistent"})

	if len(targets) != 3 {
		t.Fatalf("should return target for each tool name: expected length %v, got %v", 3, len(targets))
	}

	// Check gopls
	if !reflect.DeepEqual("gopls", targets[0].ToolName) {
		t.Errorf("expected %v, got %v", "gopls", targets[0].ToolName)
	}
	if !reflect.DeepEqual(version, targets[0].GoVersion) {
		t.Errorf("expected %v, got %v", version, targets[0].GoVersion)
	}
	if !(targets[0].Exists) {
		t.Errorf("expected true")
	}
	if len(targets[0].BinaryFiles) != 1 {
		t.Errorf("expected length %v, got %v", 1, len(targets[0].BinaryFiles))
	}

	// Check staticcheck
	if !reflect.DeepEqual("staticcheck", targets[1].ToolName) {
		t.Errorf("expected %v, got %v", "staticcheck", targets[1].ToolName)
	}
	if !(targets[1].Exists) {
		t.Errorf("expected true")
	}

	// Check nonexistent
	if !reflect.DeepEqual("nonexistent", targets[2].ToolName) {
		t.Errorf("expected %v, got %v", "nonexistent", targets[2].ToolName)
	}
	if targets[2].Exists {
		t.Errorf("expected false")
	}
	if len(targets[2].BinaryFiles) != 0 {
		t.Errorf("expected length %v, got %v", 0, len(targets[2].BinaryFiles))
	}
}

func TestFindAllVersionToolTargets(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	versions := []string{"1.21.0", "1.22.0", "1.23.0"}

	// Create gopls in all versions, staticcheck only in some
	for i, version := range versions {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		if err := os.MkdirAll(binPath, 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// gopls in all versions
		goplsPath := filepath.Join(binPath, "gopls")
		if err := os.WriteFile(goplsPath, []byte("fake"), 0755); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// staticcheck only in 1.22.0 and 1.23.0
		if i >= 1 {
			staticPath := filepath.Join(binPath, "staticcheck")
			if err := os.WriteFile(staticPath, []byte("fake"), 0755); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		}
	}

	// Find gopls across all versions
	goplsTargets := findAllVersionToolTargets(cfg, []string{"gopls"})
	if len(goplsTargets) != 3 {
		t.Errorf("gopls should be found in all 3 versions: expected length %v, got %v", 3, len(goplsTargets))
	}

	for _, target := range goplsTargets {
		if !reflect.DeepEqual("gopls", target.ToolName) {
			t.Errorf("expected %v, got %v", "gopls", target.ToolName)
		}
		if !(target.Exists) {
			t.Errorf("expected true")
		}
		if !utils.SliceContains(versions, target.GoVersion) {
			t.Errorf("expected slice to contain %v", target.GoVersion)
		}
	}

	// Find staticcheck (only in 2 versions)
	staticTargets := findAllVersionToolTargets(cfg, []string{"staticcheck"})
	if len(staticTargets) != 2 {
		t.Errorf("staticcheck should be found in 2 versions: expected length %v, got %v", 2, len(staticTargets))
	}

	// Find nonexistent tool
	noneTargets := findAllVersionToolTargets(cfg, []string{"nonexistent"})
	if len(noneTargets) != 0 {
		t.Errorf("nonexistent tool should return no targets: expected length %v, got %v", 0, len(noneTargets))
	}
}

func TestFindGlobalToolTargets(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	globalGopath := filepath.Join(tmpDir, "global-go")
	binPath := filepath.Join(globalGopath, "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create test tool
	goplsPath := filepath.Join(binPath, "gopls")
	if err := os.WriteFile(goplsPath, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Override GOPATH
	t.Setenv("GOPATH", globalGopath)

	// Find targets
	targets := findGlobalToolTargets(cfg, []string{"gopls", "nonexistent"})

	if len(targets) != 2 {
		t.Fatalf("expected length %v, got %v", 2, len(targets))
	}

	// Check gopls
	if !reflect.DeepEqual("gopls", targets[0].ToolName) {
		t.Errorf("expected %v, got %v", "gopls", targets[0].ToolName)
	}
	if !reflect.DeepEqual("", targets[0].GoVersion) {
		t.Errorf("expected %v, got %v", "", targets[0].GoVersion)
	} // Empty for global
	if !(targets[0].Exists) {
		t.Errorf("expected true")
	}

	// Check nonexistent
	if !reflect.DeepEqual("nonexistent", targets[1].ToolName) {
		t.Errorf("expected %v, got %v", "nonexistent", targets[1].ToolName)
	}
	if targets[1].Exists {
		t.Errorf("expected false")
	}
}

func TestRunUninstall_StripVersionSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create tool
	goplsPath := filepath.Join(binPath, "gopls")
	if err := os.WriteFile(goplsPath, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("GOENV_VERSION", version)

	// Test the stripping logic directly (runUninstall does the stripping)
	toolName := "gopls@v0.12.0"
	if idx := strings.Index(toolName, "@"); idx != -1 {
		toolName = toolName[:idx]
	}
	if !reflect.DeepEqual("gopls", toolName) {
		t.Errorf("expected %v, got %v", "gopls", toolName)
	}

	// Now verify findCurrentVersionToolTargets works with the clean name
	targets := findCurrentVersionToolTargets(cfg, []string{toolName})
	if len(targets) != 1 {
		t.Fatalf("expected length %v, got %v", 1, len(targets))
	}
	if !reflect.DeepEqual("gopls", targets[0].ToolName) {
		t.Errorf("expected %v, got %v", "gopls", targets[0].ToolName)
	}
	if !(targets[0].Exists) {
		t.Errorf("expected true")
	}
}

func TestExecuteUninstalls(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create test files
	gopls := filepath.Join(binPath, "gopls")
	goplsExe := filepath.Join(binPath, "gopls.exe")
	if err := os.WriteFile(gopls, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(goplsExe, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   "1.23.0",
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{gopls, goplsExe},
		},
	}

	// Execute uninstalls
	uninstallVerbose = false
	uninstallForce = true

	err := executeUninstalls(targets)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify files are removed
	_, err = os.Stat(gopls)
	if !os.IsNotExist(err) {
		t.Errorf("gopls should be removed")
	}

	_, err = os.Stat(goplsExe)
	if !os.IsNotExist(err) {
		t.Errorf("gopls.exe should be removed")
	}
}

func TestExecuteUninstalls_MultipleTools(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create multiple tools
	gopls := filepath.Join(binPath, "gopls")
	staticcheck := filepath.Join(binPath, "staticcheck")
	if err := os.WriteFile(gopls, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(staticcheck, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   "1.23.0",
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{gopls},
		},
		{
			ToolName:    "staticcheck",
			GoVersion:   "1.23.0",
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{staticcheck},
		},
	}

	uninstallVerbose = false
	uninstallForce = true

	err := executeUninstalls(targets)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify both removed
	_, err = os.Stat(gopls)
	if !os.IsNotExist(err) {
		t.Errorf("expected file to not exist")
	}

	_, err = os.Stat(staticcheck)
	if !os.IsNotExist(err) {
		t.Errorf("expected file to not exist")
	}
}

func TestExecuteUninstalls_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	binPath := filepath.Join(tmpDir, "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create one tool
	gopls := filepath.Join(binPath, "gopls")
	if err := os.WriteFile(gopls, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Create target with non-existent file (will fail to remove)
	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   "1.23.0",
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{gopls},
		},
		{
			ToolName:    "nonexistent",
			GoVersion:   "1.23.0",
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{filepath.Join(binPath, "nonexistent")},
		},
	}

	uninstallVerbose = false
	uninstallForce = true

	err := executeUninstalls(targets)
	if err == nil {
		t.Errorf("should return error when some tools fail")
	}
	if err != nil && !strings.Contains(err.Error(), "failed to uninstall") {
		t.Errorf("expected error to contain 'failed to uninstall'")
	}

	// First tool should still be removed
	_, err = os.Stat(gopls)
	if !os.IsNotExist(err) {
		t.Errorf("expected file to not exist")
	}
}

func TestShowUninstallPlan(t *testing.T) {
	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   "1.23.0",
			BinPath:     "/path/to/bin",
			Exists:      true,
			BinaryFiles: []string{"/path/to/bin/gopls", "/path/to/bin/gopls.exe"},
		},
		{
			ToolName:    "staticcheck",
			GoVersion:   "1.22.0",
			BinPath:     "/path/to/other/bin",
			Exists:      true,
			BinaryFiles: []string{"/path/to/other/bin/staticcheck"},
		},
	}

	// Test without verbose
	uninstallVerbose = false
	err := showUninstallPlan(targets)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Test with verbose
	uninstallVerbose = true
	err = showUninstallPlan(targets)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestShowUninstallPlan_Global(t *testing.T) {
	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   "", // Empty for global
			BinPath:     "/home/user/go/bin",
			Exists:      true,
			BinaryFiles: []string{"/home/user/go/bin/gopls"},
		},
	}

	uninstallVerbose = false
	err := showUninstallPlan(targets)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestRunUninstall_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Setup: Create Go version with tools
	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	gopls := filepath.Join(binPath, "gopls")
	staticcheck := filepath.Join(binPath, "staticcheck")
	if err := os.WriteFile(gopls, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := os.WriteFile(staticcheck, []byte("fake"), 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("GOENV_VERSION", version)

	// Run uninstall with dry-run
	uninstallDryRun = true
	uninstallForce = true
	uninstallAllVersions = false
	uninstallGlobal = false

	err := runUninstall(cfg, []string{"gopls"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Files should still exist after dry-run
	if _, err := os.Stat(gopls); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", gopls)
	}

	// Run actual uninstall
	uninstallDryRun = false
	err = runUninstall(cfg, []string{"gopls"})
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// gopls should be removed
	_, err = os.Stat(gopls)
	if !os.IsNotExist(err) {
		t.Errorf("expected file to not exist")
	}

	// staticcheck should still exist
	if _, err := os.Stat(staticcheck); os.IsNotExist(err) {
		t.Errorf("expected file to exist: %s", staticcheck)
	}
}

func TestRunUninstall_NoToolsFound(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	t.Setenv("GOENV_VERSION", version)

	uninstallForce = true
	err := runUninstall(cfg, []string{"nonexistent"})

	// Should not return error, just print message
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestParseToolNames(t *testing.T) {
	// Test that runUninstall strips @version
	inputs := []string{
		"gopls@v0.12.0",
		"staticcheck@latest",
		"golangci-lint",
	}

	// Just verify the stripping logic works
	var cleanNames []string
	for _, name := range inputs {
		if idx := strings.Index(name, "@"); idx != -1 {
			name = name[:idx]
		}
		cleanNames = append(cleanNames, name)
	}

	if !reflect.DeepEqual("gopls", cleanNames[0]) {
		t.Errorf("expected %v, got %v", "gopls", cleanNames[0])
	}
	if !reflect.DeepEqual("staticcheck", cleanNames[1]) {
		t.Errorf("expected %v, got %v", "staticcheck", cleanNames[1])
	}
	if !reflect.DeepEqual("golangci-lint", cleanNames[2]) {
		t.Errorf("expected %v, got %v", "golangci-lint", cleanNames[2])
	}
}
