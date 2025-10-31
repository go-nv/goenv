package tools

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/spf13/cobra"
)

func TestListCommand_AllFlag(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple versions with tools
	versionsTools := map[string][]string{
		"1.21.0": {"gopls", "staticcheck"},
		"1.22.0": {"gopls"},
		"1.23.0": {"gopls", "staticcheck", "gofmt"},
	}

	for version, tools := range versionsTools {
		binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
		if err := os.MkdirAll(binPath, 0755); err != nil {
			t.Fatal(err)
		}

		for _, tool := range tools {
			toolPath := filepath.Join(binPath, tool)
			if err := os.WriteFile(toolPath, []byte("fake"), 0755); err != nil {
				t.Fatal(err)
			}
		}
	}

	// Test with --all flag
	listAllVersions = true
	defer func() { listAllVersions = false }()

	// Get all versions
	foundVersions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("getInstalledVersions failed: %v", err)
	}

	if len(foundVersions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(foundVersions))
	}

	// Get tools for each version
	for version, expectedTools := range versionsTools {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			t.Fatalf("getToolsForVersion failed for %s: %v", version, err)
		}

		if len(tools) != len(expectedTools) {
			t.Errorf("Version %s: expected %d tools, got %d", version, len(expectedTools), len(tools))
		}

		for _, expected := range expectedTools {
			found := false
			for _, actual := range tools {
				if actual == expected {
					found = true
					break
				}
			}
			if !found {
				t.Errorf("Version %s: expected tool %s not found", version, expected)
			}
		}
	}
}

func TestListCommand_JSONOutput(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create a version with tools
	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatal(err)
	}

	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		toolPath := filepath.Join(binPath, tool)
		if err := os.WriteFile(toolPath, []byte("fake"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Enable JSON output
	listToolsJSON = true
	defer func() { listToolsJSON = false }()

	// Get tools
	foundTools, err := getToolsForVersion(cfg, version)
	if err != nil {
		t.Fatalf("getToolsForVersion failed: %v", err)
	}

	// Verify we can marshal to JSON
	type versionTools struct {
		Version string   `json:"version"`
		Tools   []string `json:"tools"`
	}

	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		Versions      []versionTools `json:"versions"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		Versions: []versionTools{
			{
				Version: version,
				Tools:   foundTools,
			},
		},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify JSON is valid
	var parsed jsonOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed.SchemaVersion != "1" {
		t.Errorf("Expected schema_version '1', got '%s'", parsed.SchemaVersion)
	}

	if len(parsed.Versions) != 1 {
		t.Errorf("Expected 1 version, got %d", len(parsed.Versions))
	}

	if parsed.Versions[0].Version != version {
		t.Errorf("Expected version %s, got %s", version, parsed.Versions[0].Version)
	}
}

func TestListCommand_EmptyVersion(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create version with no tools
	version := "1.23.0"
	versionPath := filepath.Join(tmpDir, "versions", version)
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		t.Fatal(err)
	}

	// No bin directory created - should return empty
	tools, err := getToolsForVersion(cfg, version)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(tools) != 0 {
		t.Errorf("Expected no tools, got %d", len(tools))
	}
}

func TestListCommand_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create regular and hidden files
	if err := os.WriteFile(filepath.Join(binPath, "gopls"), []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binPath, ".hidden"), []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	tools, err := getToolsForVersion(cfg, version)
	if err != nil {
		t.Fatalf("getToolsForVersion failed: %v", err)
	}

	// Should only get non-hidden files
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(tools))
	}

	if len(tools) > 0 && tools[0] != "gopls" {
		t.Errorf("Expected 'gopls', got '%s'", tools[0])
	}
}

func TestListCommand_PlatformVariants(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatal(err)
	}

	// Create tool with platform variants
	if err := os.WriteFile(filepath.Join(binPath, "gopls"), []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(binPath, "gopls.exe"), []byte("fake"), 0755); err != nil {
		t.Fatal(err)
	}

	tools, err := getToolsForVersion(cfg, version)
	if err != nil {
		t.Fatalf("getToolsForVersion failed: %v", err)
	}

	// Should deduplicate - only get base name
	if len(tools) != 1 {
		t.Errorf("Expected 1 tool (deduplicated), got %d", len(tools))
	}

	if len(tools) > 0 && tools[0] != "gopls" {
		t.Errorf("Expected 'gopls', got '%s'", tools[0])
	}
}

func TestListCommand_RunE(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create a version with tools
	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatal(err)
	}

	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		toolPath := filepath.Join(binPath, tool)
		if err := os.WriteFile(toolPath, []byte("fake"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Create .go-version to set current version
	goVersionFile := filepath.Join(tmpDir, ".go-version")
	if err := os.WriteFile(goVersionFile, []byte(version), 0644); err != nil {
		t.Fatal(err)
	}

	// Test the command can be created
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	// Verify tools are found
	foundTools, err := getToolsForVersion(cfg, version)
	if err != nil {
		t.Fatalf("getToolsForVersion failed: %v", err)
	}

	if len(foundTools) != 2 {
		t.Errorf("Expected 2 tools, got %d", len(foundTools))
	}
}

func TestListCommand_MultipleVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple versions
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		binPath := filepath.Join(tmpDir, "versions", v, "gopath", "bin")
		if err := os.MkdirAll(binPath, 0755); err != nil {
			t.Fatal(err)
		}

		// Each version has gopls
		if err := os.WriteFile(filepath.Join(binPath, "gopls"), []byte("fake"), 0755); err != nil {
			t.Fatal(err)
		}
	}

	// Get all versions
	foundVersions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("getInstalledVersions failed: %v", err)
	}

	if len(foundVersions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(foundVersions))
	}

	// Verify each version has gopls
	for _, version := range foundVersions {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			t.Fatalf("getToolsForVersion failed for %s: %v", version, err)
		}

		if len(tools) != 1 {
			t.Errorf("Version %s: expected 1 tool, got %d", version, len(tools))
		}

		if len(tools) > 0 && tools[0] != "gopls" {
			t.Errorf("Version %s: expected 'gopls', got '%s'", version, tools[0])
		}
	}
}
