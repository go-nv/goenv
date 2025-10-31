package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestOutdatedCommand_NoVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Get installed versions - should be empty
	versions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("Expected no versions, got %d", len(versions))
	}
}

func TestOutdatedCommand_JSONStructure(t *testing.T) {
	// Test JSON output structure
	outdated := outdatedTool{
		Name:           "gopls",
		GoVersion:      "1.23.0",
		CurrentVersion: "v0.12.0",
		LatestVersion:  "v0.13.2",
		PackagePath:    "golang.org/x/tools/gopls",
	}

	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		OutdatedTools []outdatedTool `json:"outdated_tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		OutdatedTools: []outdatedTool{outdated},
	}

	// Verify we can marshal to JSON
	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	// Verify we can unmarshal
	var parsed jsonOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if parsed.SchemaVersion != "1" {
		t.Errorf("Expected schema_version '1', got '%s'", parsed.SchemaVersion)
	}

	if len(parsed.OutdatedTools) != 1 {
		t.Errorf("Expected 1 outdated tool, got %d", len(parsed.OutdatedTools))
	}

	tool := parsed.OutdatedTools[0]
	if tool.Name != "gopls" {
		t.Errorf("Expected name 'gopls', got '%s'", tool.Name)
	}
	if tool.GoVersion != "1.23.0" {
		t.Errorf("Expected go_version '1.23.0', got '%s'", tool.GoVersion)
	}
	if tool.CurrentVersion != "v0.12.0" {
		t.Errorf("Expected current_version 'v0.12.0', got '%s'", tool.CurrentVersion)
	}
	if tool.LatestVersion != "v0.13.2" {
		t.Errorf("Expected latest_version 'v0.13.2', got '%s'", tool.LatestVersion)
	}
	if tool.PackagePath != "golang.org/x/tools/gopls" {
		t.Errorf("Expected package_path 'golang.org/x/tools/gopls', got '%s'", tool.PackagePath)
	}
}

func TestOutdatedCommand_EmptyResult(t *testing.T) {
	// Test empty outdated tools result
	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		OutdatedTools []outdatedTool `json:"outdated_tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		OutdatedTools: []outdatedTool{},
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var parsed jsonOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(parsed.OutdatedTools) != 0 {
		t.Errorf("Expected 0 outdated tools, got %d", len(parsed.OutdatedTools))
	}
}

func TestOutdatedCommand_MultipleTools(t *testing.T) {
	// Test multiple outdated tools
	tools := []outdatedTool{
		{
			Name:           "gopls",
			GoVersion:      "1.21.0",
			CurrentVersion: "v0.12.0",
			LatestVersion:  "v0.13.2",
			PackagePath:    "golang.org/x/tools/gopls",
		},
		{
			Name:           "staticcheck",
			GoVersion:      "1.21.0",
			CurrentVersion: "v0.4.0",
			LatestVersion:  "v0.4.6",
			PackagePath:    "honnef.co/go/tools/cmd/staticcheck",
		},
		{
			Name:           "gopls",
			GoVersion:      "1.23.0",
			CurrentVersion: "v0.12.0",
			LatestVersion:  "v0.13.2",
			PackagePath:    "golang.org/x/tools/gopls",
		},
	}

	type jsonOutput struct {
		SchemaVersion string         `json:"schema_version"`
		OutdatedTools []outdatedTool `json:"outdated_tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		OutdatedTools: tools,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var parsed jsonOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(parsed.OutdatedTools) != 3 {
		t.Errorf("Expected 3 outdated tools, got %d", len(parsed.OutdatedTools))
	}

	// Verify grouping by Go version
	versionGroups := make(map[string][]outdatedTool)
	for _, tool := range parsed.OutdatedTools {
		versionGroups[tool.GoVersion] = append(versionGroups[tool.GoVersion], tool)
	}

	if len(versionGroups) != 2 {
		t.Errorf("Expected 2 Go versions, got %d", len(versionGroups))
	}

	if len(versionGroups["1.21.0"]) != 2 {
		t.Errorf("Expected 2 outdated tools for Go 1.21.0, got %d", len(versionGroups["1.21.0"]))
	}

	if len(versionGroups["1.23.0"]) != 1 {
		t.Errorf("Expected 1 outdated tool for Go 1.23.0, got %d", len(versionGroups["1.23.0"]))
	}
}

func TestOutdatedCommand_Command(t *testing.T) {
	// Test that the command can be created
	cmd := newOutdatedCommand()

	if cmd == nil {
		t.Fatal("Expected command to be created")
	}

	if cmd.Use != "outdated" {
		t.Errorf("Expected Use 'outdated', got '%s'", cmd.Use)
	}

	if cmd.Short != "Show outdated tools across all Go versions" {
		t.Errorf("Unexpected Short description: %s", cmd.Short)
	}

	// Check flags
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("Expected --json flag to exist")
	}
}

func TestOutdatedCommand_WithVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create versions but no tools
	versions := []string{"1.21.0", "1.22.0", "1.23.0"}
	for _, v := range versions {
		versionPath := filepath.Join(tmpDir, "versions", v)
		if err := os.MkdirAll(versionPath, 0755); err != nil {
			t.Fatal(err)
		}
	}

	foundVersions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("getInstalledVersions failed: %v", err)
	}

	if len(foundVersions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(foundVersions))
	}

	// No tools installed - outdated should be empty
	for _, version := range foundVersions {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			t.Fatalf("getToolsForVersion failed for %s: %v", version, err)
		}

		if len(tools) != 0 {
			t.Errorf("Version %s: expected no tools, got %d", version, len(tools))
		}
	}
}
