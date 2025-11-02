package tools

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/testing/testutil"
)

// mockVersionManager implements VersionManager for testing
type mockVersionManager struct {
	versions []string
	err      error
}

func (m *mockVersionManager) ListInstalledVersions() ([]string, error) {
	if m.err != nil {
		return nil, m.err
	}
	return m.versions, nil
}

func TestListForVersion(t *testing.T) {
	// Create temporary directory structure
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create test tool binaries
	tools := []string{"goimports", "gopls", "dlv"}
	for _, tool := range tools {
		toolPath := filepath.Join(binPath, tool)
		testutil.WriteTestFile(t, toolPath, []byte("#!/bin/bash\necho test"), utils.PermFileExecutable)
	}

	// Test listing tools
	result, err := ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	if len(result) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(result))
	}

	// Verify tool metadata
	for _, tool := range result {
		if tool.Name == "" {
			t.Error("Tool name should not be empty")
		}
		if tool.BinaryPath == "" {
			t.Error("Tool binary path should not be empty")
		}
		if tool.GoVersion != version {
			t.Errorf("Expected GoVersion %s, got %s", version, tool.GoVersion)
		}
		if tool.ModTime.IsZero() {
			t.Error("Tool ModTime should not be zero")
		}
	}
}

func TestListForVersion_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Test with non-existent version
	result, err := ListForVersion(cfg, "1.99.0")
	if err != nil {
		t.Fatalf("ListForVersion should not error on missing directory: %v", err)
	}

	if result != nil {
		t.Errorf("Expected nil result for non-existent version, got %v", result)
	}
}

func TestListForVersion_EmptyDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Test with empty directory
	result, err := ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected 0 tools in empty directory, got %d", len(result))
	}
}

func TestListForVersion_PlatformVariants(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create tool with multiple platform variants
	variants := []string{"goimports", "goimports.exe", "goimports.darwin"}
	for _, variant := range variants {
		toolPath := filepath.Join(binPath, variant)
		testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)
	}

	// Test deduplication
	result, err := ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	// Should only return one entry despite multiple platform variants
	if len(result) != 1 {
		t.Errorf("Expected 1 tool (deduplicated), got %d", len(result))
	}

	if result[0].Name != "goimports" {
		t.Errorf("Expected tool name 'goimports', got %s", result[0].Name)
	}
}

func TestListForVersion_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create hidden file
	hiddenPath := filepath.Join(binPath, ".hidden")
	testutil.WriteTestFile(t, hiddenPath, []byte("test"), utils.PermFileDefault)

	// Create normal tool
	toolPath := filepath.Join(binPath, "goimports")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	// Test that hidden files are excluded
	result, err := ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	if len(result) != 1 {
		t.Errorf("Expected 1 tool (hidden file excluded), got %d", len(result))
	}

	if result[0].Name != "goimports" {
		t.Errorf("Expected tool name 'goimports', got %s", result[0].Name)
	}
}

func TestListAll(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Create tools for multiple versions
	versions := []string{"1.20.0", "1.21.0"}
	for _, version := range versions {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
			t.Fatalf("Failed to create bin directory: %v", err)
		}

		toolPath := filepath.Join(binPath, "goimports")
		testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)
	}

	mgr := &mockVersionManager{versions: versions}

	// Test listing all tools
	result, err := ListAll(cfg, mgr)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(result) != 2 {
		t.Errorf("Expected tools for 2 versions, got %d", len(result))
	}

	for _, version := range versions {
		tools, ok := result[version]
		if !ok {
			t.Errorf("Expected tools for version %s", version)
			continue
		}
		if len(tools) != 1 {
			t.Errorf("Expected 1 tool for version %s, got %d", version, len(tools))
		}
	}
}

func TestListAll_EmptyVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	mgr := &mockVersionManager{versions: []string{}}

	result, err := ListAll(cfg, mgr)
	if err != nil {
		t.Fatalf("ListAll failed: %v", err)
	}

	if len(result) != 0 {
		t.Errorf("Expected empty result, got %d versions", len(result))
	}
}

func TestIsInstalled(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create test tool
	toolPath := filepath.Join(binPath, "goimports")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	tests := []struct {
		name     string
		version  string
		toolName string
		expected bool
	}{
		{
			name:     "installed tool",
			version:  version,
			toolName: "goimports",
			expected: true,
		},
		{
			name:     "not installed tool",
			version:  version,
			toolName: "gopls",
			expected: false,
		},
		{
			name:     "non-existent version",
			version:  "1.99.0",
			toolName: "goimports",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := IsInstalled(cfg, tt.version, tt.toolName)
			if result != tt.expected {
				t.Errorf("IsInstalled(%s, %s) = %v, want %v",
					tt.version, tt.toolName, result, tt.expected)
			}
		})
	}
}

func TestIsInstalled_PlatformVariants(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create tool with .exe extension (Windows)
	toolPath := filepath.Join(binPath, "goimports.exe")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	// Should find tool even without extension
	if !IsInstalled(cfg, version, "goimports") {
		t.Error("IsInstalled should find tool with .exe extension")
	}
}

func TestGetToolInfo(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.21.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create test tool
	toolPath := filepath.Join(binPath, "goimports")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	// Test getting tool info
	tool, err := GetToolInfo(cfg, version, "goimports")
	if err != nil {
		t.Fatalf("GetToolInfo failed: %v", err)
	}

	if tool == nil {
		t.Fatal("Expected tool info, got nil")
	}

	if tool.Name != "goimports" {
		t.Errorf("Expected tool name 'goimports', got %s", tool.Name)
	}

	if tool.GoVersion != version {
		t.Errorf("Expected GoVersion %s, got %s", version, tool.GoVersion)
	}

	// Test non-existent tool
	tool, err = GetToolInfo(cfg, version, "nonexistent")
	if err != nil {
		t.Fatalf("GetToolInfo should not error for missing tool: %v", err)
	}

	if tool != nil {
		t.Errorf("Expected nil for non-existent tool, got %v", tool)
	}
}

func TestCollectUniqueTools(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	// Create same tools across multiple versions
	versions := []string{"1.20.0", "1.21.0"}
	for _, version := range versions {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
			t.Fatalf("Failed to create bin directory: %v", err)
		}

		// Both versions have goimports
		toolPath := filepath.Join(binPath, "goimports")
		testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)
	}

	// Add version-specific tool
	binPath := filepath.Join(tmpDir, "versions", "1.21.0", "gopath", "bin")
	toolPath := filepath.Join(binPath, "gopls")
	testutil.WriteTestFile(t, toolPath, []byte("test"), utils.PermFileExecutable)

	mgr := &mockVersionManager{versions: versions}

	// Test collecting unique tools
	result, err := CollectUniqueTools(cfg, mgr)
	if err != nil {
		t.Fatalf("CollectUniqueTools failed: %v", err)
	}

	// Should have 2 unique tools: goimports and gopls
	if len(result) != 2 {
		t.Errorf("Expected 2 unique tools, got %d: %v", len(result), result)
	}

	// Check that both tools are present
	hasGoimports := false
	hasGopls := false
	for _, name := range result {
		if name == "goimports" {
			hasGoimports = true
		}
		if name == "gopls" {
			hasGopls = true
		}
	}

	if !hasGoimports {
		t.Error("Expected 'goimports' in unique tools")
	}
	if !hasGopls {
		t.Error("Expected 'gopls' in unique tools")
	}
}
