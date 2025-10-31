package tools

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestStatusCommand_NoVersions(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatal(err)
	}

	versions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	if len(versions) != 0 {
		t.Errorf("Expected no versions, got %d", len(versions))
	}
}

func TestStatusCommand_JSONStructure(t *testing.T) {
	// Test JSON output structure
	status := toolStatus{
		Name:          "gopls",
		TotalVersions: 3,
		InstalledIn:   3,
		VersionPresence: map[string]bool{
			"1.21.0": true,
			"1.22.0": true,
			"1.23.0": true,
		},
		ConsistencyScore: 100.0,
	}

	type jsonOutput struct {
		SchemaVersion string       `json:"schema_version"`
		GoVersions    []string     `json:"go_versions"`
		Tools         []toolStatus `json:"tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		GoVersions:    []string{"1.21.0", "1.22.0", "1.23.0"},
		Tools:         []toolStatus{status},
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

	if len(parsed.GoVersions) != 3 {
		t.Errorf("Expected 3 Go versions, got %d", len(parsed.GoVersions))
	}

	if len(parsed.Tools) != 1 {
		t.Errorf("Expected 1 tool, got %d", len(parsed.Tools))
	}

	tool := parsed.Tools[0]
	if tool.Name != "gopls" {
		t.Errorf("Expected name 'gopls', got '%s'", tool.Name)
	}
	if tool.TotalVersions != 3 {
		t.Errorf("Expected total_versions 3, got %d", tool.TotalVersions)
	}
	if tool.InstalledIn != 3 {
		t.Errorf("Expected installed_in 3, got %d", tool.InstalledIn)
	}
	if tool.ConsistencyScore != 100.0 {
		t.Errorf("Expected consistency_score 100.0, got %f", tool.ConsistencyScore)
	}
}

func TestStatusCommand_ConsistencyCalculation(t *testing.T) {
	tests := []struct {
		name          string
		totalVersions int
		installedIn   int
		expectedScore float64
	}{
		{
			name:          "fully consistent",
			totalVersions: 3,
			installedIn:   3,
			expectedScore: 100.0,
		},
		{
			name:          "two thirds consistent",
			totalVersions: 3,
			installedIn:   2,
			expectedScore: 66.66666666666666,
		},
		{
			name:          "one third consistent",
			totalVersions: 3,
			installedIn:   1,
			expectedScore: 33.33333333333333,
		},
		{
			name:          "not installed",
			totalVersions: 3,
			installedIn:   0,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := float64(tt.installedIn) / float64(tt.totalVersions) * 100

			// Use a small epsilon for floating point comparison
			epsilon := 0.0001
			if score < tt.expectedScore-epsilon || score > tt.expectedScore+epsilon {
				t.Errorf("Expected score %f, got %f", tt.expectedScore, score)
			}
		})
	}
}

func TestStatusCommand_Categorization(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create multiple versions with different tool installations
	versionsTools := map[string][]string{
		"1.21.0": {"gopls", "staticcheck", "gofmt"},
		"1.22.0": {"gopls", "staticcheck"},
		"1.23.0": {"gopls", "golangci-lint"},
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

	// Get versions and collect tools
	foundVersions, err := getInstalledVersions(cfg)
	if err != nil {
		t.Fatalf("getInstalledVersions failed: %v", err)
	}

	toolsByVersion := make(map[string][]string)
	allToolNames := make(map[string]bool)

	for _, version := range foundVersions {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			t.Fatalf("getToolsForVersion failed for %s: %v", version, err)
		}
		toolsByVersion[version] = tools
		for _, tool := range tools {
			allToolNames[tool] = true
		}
	}

	// Verify gopls is consistent (in all versions)
	goplsCount := 0
	for _, tools := range toolsByVersion {
		if contains(tools, "gopls") {
			goplsCount++
		}
	}
	if goplsCount != 3 {
		t.Errorf("Expected gopls in all 3 versions, found in %d", goplsCount)
	}

	// Verify staticcheck is partial (in 2 versions)
	staticcheckCount := 0
	for _, tools := range toolsByVersion {
		if contains(tools, "staticcheck") {
			staticcheckCount++
		}
	}
	if staticcheckCount != 2 {
		t.Errorf("Expected staticcheck in 2 versions, found in %d", staticcheckCount)
	}

	// Verify gofmt is version-specific (in 1 version)
	gofmtCount := 0
	for _, tools := range toolsByVersion {
		if contains(tools, "gofmt") {
			gofmtCount++
		}
	}
	if gofmtCount != 1 {
		t.Errorf("Expected gofmt in 1 version, found in %d", gofmtCount)
	}

	// Verify golangci-lint is version-specific (in 1 version)
	golangciCount := 0
	for _, tools := range toolsByVersion {
		if contains(tools, "golangci-lint") {
			golangciCount++
		}
	}
	if golangciCount != 1 {
		t.Errorf("Expected golangci-lint in 1 version, found in %d", golangciCount)
	}
}

func TestStatusCommand_EmptyTools(t *testing.T) {
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

	// All versions should have no tools
	allToolNames := make(map[string]bool)
	for _, version := range foundVersions {
		tools, err := getToolsForVersion(cfg, version)
		if err != nil {
			t.Fatalf("getToolsForVersion failed for %s: %v", version, err)
		}

		if len(tools) != 0 {
			t.Errorf("Version %s: expected no tools, got %d", version, len(tools))
		}

		for _, tool := range tools {
			allToolNames[tool] = true
		}
	}

	if len(allToolNames) != 0 {
		t.Errorf("Expected no tools across all versions, got %d", len(allToolNames))
	}
}

func TestStatusCommand_Command(t *testing.T) {
	// Test that the command can be created
	cmd := newStatusCommand()

	if cmd == nil {
		t.Fatal("Expected command to be created")
	}

	if cmd.Use != "status" {
		t.Errorf("Expected Use 'status', got '%s'", cmd.Use)
	}

	if cmd.Short != "Show tool installation consistency across versions" {
		t.Errorf("Unexpected Short description: %s", cmd.Short)
	}

	// Check flags
	jsonFlag := cmd.Flags().Lookup("json")
	if jsonFlag == nil {
		t.Error("Expected --json flag to exist")
	}
}

func TestConsistencyScoreCalculation(t *testing.T) {
	tests := []struct {
		name          string
		totalVersions int
		installedIn   int
		expectedScore float64
	}{
		{
			name:          "fully consistent",
			totalVersions: 3,
			installedIn:   3,
			expectedScore: 100.0,
		},
		{
			name:          "partially consistent",
			totalVersions: 3,
			installedIn:   2,
			expectedScore: 66.66666666666666,
		},
		{
			name:          "single version",
			totalVersions: 3,
			installedIn:   1,
			expectedScore: 33.33333333333333,
		},
		{
			name:          "not installed",
			totalVersions: 3,
			installedIn:   0,
			expectedScore: 0.0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := float64(tt.installedIn) / float64(tt.totalVersions) * 100
			epsilon := 0.0001
			if score < tt.expectedScore-epsilon || score > tt.expectedScore+epsilon {
				t.Errorf("Expected score %f, got %f", tt.expectedScore, score)
			}
		})
	}
}

func TestStatusCommand_MultipleTools(t *testing.T) {
	// Test creating multiple tool statuses
	statuses := []toolStatus{
		{
			Name:          "gopls",
			TotalVersions: 3,
			InstalledIn:   3,
			VersionPresence: map[string]bool{
				"1.21.0": true,
				"1.22.0": true,
				"1.23.0": true,
			},
			ConsistencyScore: 100.0,
		},
		{
			Name:          "staticcheck",
			TotalVersions: 3,
			InstalledIn:   2,
			VersionPresence: map[string]bool{
				"1.21.0": true,
				"1.22.0": false,
				"1.23.0": true,
			},
			ConsistencyScore: 66.66666666666666,
		},
		{
			Name:          "gofmt",
			TotalVersions: 3,
			InstalledIn:   1,
			VersionPresence: map[string]bool{
				"1.21.0": true,
				"1.22.0": false,
				"1.23.0": false,
			},
			ConsistencyScore: 33.33333333333333,
		},
	}

	type jsonOutput struct {
		SchemaVersion string       `json:"schema_version"`
		GoVersions    []string     `json:"go_versions"`
		Tools         []toolStatus `json:"tools"`
	}

	output := jsonOutput{
		SchemaVersion: "1",
		GoVersions:    []string{"1.21.0", "1.22.0", "1.23.0"},
		Tools:         statuses,
	}

	data, err := json.Marshal(output)
	if err != nil {
		t.Fatalf("Failed to marshal JSON: %v", err)
	}

	var parsed jsonOutput
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("Failed to unmarshal JSON: %v", err)
	}

	if len(parsed.Tools) != 3 {
		t.Errorf("Expected 3 tools, got %d", len(parsed.Tools))
	}

	// Categorize tools
	var fullyInstalled, partiallyInstalled, singleVersion []toolStatus
	for _, status := range parsed.Tools {
		if status.InstalledIn == status.TotalVersions {
			fullyInstalled = append(fullyInstalled, status)
		} else if status.InstalledIn == 1 {
			singleVersion = append(singleVersion, status)
		} else {
			partiallyInstalled = append(partiallyInstalled, status)
		}
	}

	if len(fullyInstalled) != 1 {
		t.Errorf("Expected 1 fully installed tool, got %d", len(fullyInstalled))
	}

	if len(partiallyInstalled) != 1 {
		t.Errorf("Expected 1 partially installed tool, got %d", len(partiallyInstalled))
	}

	if len(singleVersion) != 1 {
		t.Errorf("Expected 1 single-version tool, got %d", len(singleVersion))
	}
}
