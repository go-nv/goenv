package tools

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/testing/testutil"
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
		// Create version with tool directory
		cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)

		// Create individual tool binaries using helper (handles .bat on Windows)
		cfg := &config.Config{Root: tmpDir}
		binPath := cfg.VersionGopathBin(version)

		for _, tool := range tools {
			cmdtest.CreateToolExecutable(t, binPath, tool)
		}
	}

	// Test with --all flag
	listAllVersions = true
	defer func() { listAllVersions = false }()

	// Get all versions
	mgr := manager.NewManager(cfg)
	foundVersions, err := mgr.ListInstalledVersions()
	if err != nil {
		t.Fatalf("ListInstalledVersions failed: %v", err)
	}

	if len(foundVersions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(foundVersions))
	}

	// Get tools for each version
	for version, expectedTools := range versionsTools {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		if err != nil {
			t.Fatalf("ListForVersion failed for %s: %v", version, err)
		}

		// Extract tool names from Tool structs
		var toolNames []string
		for _, tool := range toolList {
			toolNames = append(toolNames, tool.Name)
		}

		if len(toolNames) != len(expectedTools) {
			t.Errorf("Version %s: expected %d tools, got %d", version, len(expectedTools), len(toolNames))
		}

		for _, expected := range expectedTools {
			found := false
			for _, actual := range toolNames {
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
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatal(err)
	}

	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		cmdtest.CreateToolExecutable(t, binPath, tool)
	}

	// Enable JSON output
	listToolsJSON = true
	defer func() { listToolsJSON = false }()

	// Get tools
	toolList, err := toolspkg.ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	// Extract tool names from Tool structs
	var foundTools []string
	for _, tool := range toolList {
		foundTools = append(foundTools, tool.Name)
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
	if err := utils.EnsureDirWithContext(versionPath, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// No bin directory created - should return empty
	toolList, err := toolspkg.ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(toolList) != 0 {
		t.Errorf("Expected no tools, got %d", len(toolList))
	}
}

func TestListCommand_HiddenFiles(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Create regular and hidden files (use helper for gopls to handle .bat on Windows)
	cmdtest.CreateToolExecutable(t, binPath, "gopls")
	testutil.WriteTestFile(t, filepath.Join(binPath, ".hidden"), []byte("fake"), utils.PermFileExecutable)

	toolList, err := toolspkg.ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	// Extract tool names from Tool structs
	var toolNames []string
	for _, tool := range toolList {
		toolNames = append(toolNames, tool.Name)
	}

	// Should only get non-hidden files
	if len(toolNames) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(toolNames))
	}

	if len(toolNames) > 0 && toolNames[0] != "gopls" {
		t.Errorf("Expected 'gopls', got '%s'", toolNames[0])
	}
}

func TestListCommand_PlatformVariants(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	binPath := filepath.Join(tmpDir, "versions", version, "gopath", "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Create tool with platform variants
	testutil.WriteTestFile(t, filepath.Join(binPath, "gopls"), []byte("fake"), utils.PermFileExecutable)
	testutil.WriteTestFile(t, filepath.Join(binPath, "gopls.exe"), []byte("fake"), utils.PermFileExecutable)

	toolList, err := toolspkg.ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	// Extract tool names from Tool structs
	var toolNames []string
	for _, tool := range toolList {
		toolNames = append(toolNames, tool.Name)
	}

	// Should deduplicate - only get base name
	if len(toolNames) != 1 {
		t.Errorf("Expected 1 tool (deduplicated), got %d", len(toolNames))
	}

	if len(toolNames) > 0 && toolNames[0] != "gopls" {
		t.Errorf("Expected 'gopls', got '%s'", toolNames[0])
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
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatal(err)
	}

	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		cmdtest.CreateToolExecutable(t, binPath, tool)
	}

	// Create .go-version to set current version
	goVersionFile := filepath.Join(tmpDir, ".go-version")
	testutil.WriteTestFile(t, goVersionFile, []byte(version), utils.PermFileDefault)

	// Test the command can be created
	cmd := &cobra.Command{}
	var out bytes.Buffer
	cmd.SetOut(&out)
	cmd.SetErr(&out)

	// Verify tools are found
	toolList, err := toolspkg.ListForVersion(cfg, version)
	if err != nil {
		t.Fatalf("ListForVersion failed: %v", err)
	}

	// Extract tool names from Tool structs
	var foundTools []string
	for _, tool := range toolList {
		foundTools = append(foundTools, tool.Name)
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
		// Create Go binary for version using helper (handles .bat on Windows)
		goBinDir := filepath.Join(tmpDir, "versions", v, "bin")
		cmdtest.CreateTestBinary(t, tmpDir, v, "go")

		// Create tools using helper (handles .bat on Windows)
		binPath := filepath.Join(tmpDir, "versions", v, "gopath", "bin")
		if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
			t.Fatal(err)
		}
		cmdtest.CreateToolExecutable(t, binPath, "gopls")
	}

	// Get all versions
	mgr := manager.NewManager(cfg)
	foundVersions, err := mgr.ListInstalledVersions()
	if err != nil {
		t.Fatalf("ListInstalledVersions failed: %v", err)
	}

	if len(foundVersions) != 3 {
		t.Errorf("Expected 3 versions, got %d", len(foundVersions))
	}

	// Verify each version has gopls
	for _, version := range foundVersions {
		toolList, err := toolspkg.ListForVersion(cfg, version)
		if err != nil {
			t.Fatalf("ListForVersion failed for %s: %v", version, err)
		}

		// Extract tool names from Tool structs
		var toolNames []string
		for _, tool := range toolList {
			toolNames = append(toolNames, tool.Name)
		}

		if len(toolNames) != 1 {
			t.Errorf("Version %s: expected 1 tool, got %d", version, len(toolNames))
		}

		if len(toolNames) > 0 && toolNames[0] != "gopls" {
			t.Errorf("Version %s: expected 'gopls', got '%s'", version, toolNames[0])
		}
	}
}
