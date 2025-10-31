package vscode

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

func TestCheckSettings_EnvVars(t *testing.T) {
	// Create a temporary settings file
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	content := `{
	"go.goroot": "${env:GOROOT}",
	"go.toolsGopath": "~/go/tools"
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckSettings(settingsPath, "1.23.2")

	if !result.HasSettings {
		t.Error("Expected HasSettings to be true")
	}
	if !result.UsesEnvVars {
		t.Error("Expected UsesEnvVars to be true")
	}
	if result.Mismatch {
		t.Error("Expected no mismatch when using env vars")
	}
	if result.ConfiguredVersion != "" {
		t.Errorf("Expected empty ConfiguredVersion, got %q", result.ConfiguredVersion)
	}
}

func TestCheckSettings_EnvHomeWithVersion(t *testing.T) {
	// Test the specific case: ${env:HOME} with hardcoded version
	// This should NOT be considered "using env vars" because the version is hardcoded
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	content := `{
	"go.goroot": "${env:HOME}/.goenv/versions/1.24.1",
	"go.gopath": "${env:HOME}/go/1.24.1",
	"go.toolsGopath": "${env:HOME}/go/tools"
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Current version is 1.24.3, but settings have 1.24.1
	result := CheckSettings(settingsPath, "1.24.3")

	if !result.HasSettings {
		t.Error("Expected HasSettings to be true")
	}
	if result.UsesEnvVars {
		t.Error("Expected UsesEnvVars to be false - hardcoded version in path needs updating")
	}
	if result.ConfiguredVersion != "1.24.1" {
		t.Errorf("Expected ConfiguredVersion to be '1.24.1', got %q", result.ConfiguredVersion)
	}
	if !result.Mismatch {
		t.Error("Expected Mismatch to be true")
	}
}

func TestCheckSettings_UnixPath(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Unix-style path
	content := `{
	"go.goroot": "/Users/adam/.goenv/versions/1.23.2"
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckSettings(settingsPath, "1.23.2")

	if !result.HasSettings {
		t.Error("Expected HasSettings to be true")
	}
	if result.UsesEnvVars {
		t.Error("Expected UsesEnvVars to be false")
	}
	if result.ConfiguredVersion != "1.23.2" {
		t.Errorf("Expected ConfiguredVersion to be '1.23.2', got %q", result.ConfiguredVersion)
	}
	if result.Mismatch {
		t.Error("Expected no mismatch")
	}
}

func TestCheckSettings_WindowsPath(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Windows-style path with backslashes (escaped in JSON)
	content := `{
	"go.goroot": "C:\\Users\\adam\\.goenv\\versions\\1.23.2"
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckSettings(settingsPath, "1.23.2")

	if !result.HasSettings {
		t.Error("Expected HasSettings to be true")
	}
	if result.UsesEnvVars {
		t.Error("Expected UsesEnvVars to be false")
	}
	if result.ConfiguredVersion != "1.23.2" {
		t.Errorf("Expected ConfiguredVersion to be '1.23.2', got %q", result.ConfiguredVersion)
	}
	if result.Mismatch {
		t.Error("Expected no mismatch")
	}
}

func TestCheckSettings_VersionMismatch_Unix(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	content := `{
	"go.goroot": "/Users/adam/.goenv/versions/1.22.0"
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckSettings(settingsPath, "1.23.2")

	if !result.Mismatch {
		t.Error("Expected Mismatch to be true")
	}
	if result.ConfiguredVersion != "1.22.0" {
		t.Errorf("Expected ConfiguredVersion to be '1.22.0', got %q", result.ConfiguredVersion)
	}
	if result.ExpectedVersion != "1.23.2" {
		t.Errorf("Expected ExpectedVersion to be '1.23.2', got %q", result.ExpectedVersion)
	}
}

func TestCheckSettings_VersionMismatch_Windows(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Windows path with different version
	content := `{
	"go.goroot": "C:\\Users\\adam\\.goenv\\versions\\1.22.0"
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckSettings(settingsPath, "1.23.2")

	if !result.Mismatch {
		t.Error("Expected Mismatch to be true")
	}
	if result.ConfiguredVersion != "1.22.0" {
		t.Errorf("Expected ConfiguredVersion to be '1.22.0', got %q", result.ConfiguredVersion)
	}
}

func TestCheckSettings_NoSettings(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Settings file with no Go configuration
	content := `{
	"editor.fontSize": 14
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckSettings(settingsPath, "1.23.2")

	if result.HasSettings {
		t.Error("Expected HasSettings to be false")
	}
	if result.UsesEnvVars {
		t.Error("Expected UsesEnvVars to be false")
	}
	if result.Mismatch {
		t.Error("Expected Mismatch to be false")
	}
}

func TestCheckSettings_NonExistentFile(t *testing.T) {
	result := CheckSettings("/nonexistent/path/settings.json", "1.23.2")

	if result.HasSettings {
		t.Error("Expected HasSettings to be false for non-existent file")
	}
	if result.Mismatch {
		t.Error("Expected Mismatch to be false for non-existent file")
	}
}

func TestCheckSettings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	settingsPath := filepath.Join(tmpDir, "settings.json")

	// Invalid JSON
	content := `{
	"go.goroot": "/path/to/go",
	invalid json here
}`

	if err := os.WriteFile(settingsPath, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	result := CheckSettings(settingsPath, "1.23.2")

	// Should gracefully handle invalid JSON
	if result.HasSettings {
		t.Error("Expected HasSettings to be false for invalid JSON")
	}
}

func TestDetectIndentation(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected int
	}{
		{
			name: "2 spaces",
			content: `{
  "key": "value"
}`,
			expected: 2,
		},
		{
			name: "4 spaces",
			content: `{
    "key": "value"
}`,
			expected: 4,
		},
		{
			name: "tabs",
			content: `{
	"key": "value"
}`,
			expected: 4, // Tabs converted to 4 spaces
		},
		{
			name: "no indentation",
			content: `{
"key": "value"
}`,
			expected: 2, // Default to 2
		},
		{
			name:     "empty file",
			content:  "",
			expected: 2, // Default to 2
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := DetectIndentation(tt.content)
			if result != tt.expected {
				t.Errorf("Expected indentation %d, got %d", tt.expected, result)
			}
		})
	}
}

func TestEscapeJSONKey(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"go.goroot", `go\.goroot`},
		{"go.toolsGopath", `go\.toolsGopath`},
		{"simple", "simple"},
		{"multiple.dots.here", `multiple\.dots\.here`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := EscapeJSONKey(tt.input)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestUpdateJSONKeys(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "settings.json")

	// Initial content
	initial := `{
  "editor.fontSize": 14,
  "go.goroot": "/old/path"
}`

	if err := os.WriteFile(testFile, []byte(initial), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Update keys
	updates := map[string]any{
		"go.goroot":      "/new/path",
		"go.toolsGopath": "~/go/tools",
	}

	if err := UpdateJSONKeys(testFile, updates); err != nil {
		t.Fatalf("UpdateJSONKeys failed: %v", err)
	}

	// Read back and verify
	settings, err := ReadExistingSettings(testFile)
	if err != nil {
		t.Fatalf("Failed to read updated settings: %v", err)
	}

	if settings["go.goroot"] != "/new/path" {
		t.Errorf("Expected go.goroot to be '/new/path', got %v", settings["go.goroot"])
	}

	if settings["go.toolsGopath"] != "~/go/tools" {
		t.Errorf("Expected go.toolsGopath to be '~/go/tools', got %v", settings["go.toolsGopath"])
	}

	if settings["editor.fontSize"].(float64) != 14 {
		t.Error("Expected editor.fontSize to be preserved")
	}
}

func TestUpdateJSONKeys_PreservesIndentation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "settings.json")

	// Content with 4-space indentation
	initial := `{
    "go.goroot": "/old/path"
}`

	if err := os.WriteFile(testFile, []byte(initial), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	updates := map[string]any{
		"go.goroot": "/new/path",
	}

	if err := UpdateJSONKeys(testFile, updates); err != nil {
		t.Fatalf("UpdateJSONKeys failed: %v", err)
	}

	// Read back and check indentation
	data, err := os.ReadFile(testFile)
	if err != nil {
		t.Fatalf("Failed to read file: %v", err)
	}

	content := string(data)
	if !containsFourSpaceIndent(content) {
		t.Error("Expected 4-space indentation to be preserved")
	}
}

// Helper function to check for 4-space indentation
func containsFourSpaceIndent(content string) bool {
	lines := splitLines(content)
	for _, line := range lines {
		if len(line) >= 4 && line[:4] == "    " {
			return true
		}
	}
	return false
}

func splitLines(s string) []string {
	var lines []string
	start := 0
	for i := 0; i < len(s); i++ {
		if s[i] == '\n' {
			lines = append(lines, s[start:i])
			start = i + 1
		}
	}
	if start < len(s) {
		lines = append(lines, s[start:])
	}
	return lines
}

func TestReadExistingSettings_JSONC(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "settings.json")

	// JSONC content with comments and trailing commas
	content := `{
  // This is a comment
  "go.goroot": "/path/to/go",
  "go.toolsGopath": "~/go/tools", // trailing comma
}`

	if err := os.WriteFile(testFile, []byte(content), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	settings, err := ReadExistingSettings(testFile)
	if err != nil {
		t.Fatalf("ReadExistingSettings failed: %v", err)
	}

	if settings["go.goroot"] != "/path/to/go" {
		t.Errorf("Expected go.goroot to be '/path/to/go', got %v", settings["go.goroot"])
	}

	if settings["go.toolsGopath"] != "~/go/tools" {
		t.Errorf("Expected go.toolsGopath to be '~/go/tools', got %v", settings["go.toolsGopath"])
	}
}

func TestFindSettingsFile(t *testing.T) {
	tmpDir := t.TempDir()
	vscodeDir := filepath.Join(tmpDir, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	settingsPath, err := FindSettingsFile(tmpDir)
	if err != nil {
		t.Fatalf("FindSettingsFile failed: %v", err)
	}

	expectedPath := filepath.Join(tmpDir, ".vscode", "settings.json")
	if settingsPath != expectedPath {
		t.Errorf("Expected settings path %q, got %q", expectedPath, settingsPath)
	}
}

func TestHasVSCodeDirectory(t *testing.T) {
	tmpDir := t.TempDir()

	// Should not exist initially
	if HasVSCodeDirectory(tmpDir) {
		t.Error("Expected HasVSCodeDirectory to be false")
	}

	// Create .vscode directory
	vscodeDir := filepath.Join(tmpDir, ".vscode")
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	// Should exist now
	if !HasVSCodeDirectory(tmpDir) {
		t.Error("Expected HasVSCodeDirectory to be true")
	}
}

func TestWriteJSONFile_PreservesIndentation(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "extensions.json")

	tests := []struct {
		name             string
		existingFile     string
		expectedWidth    int
		expectedNewlines int
	}{
		{
			name: "preserves 2-space indentation (no trailing newline)",
			existingFile: `{
  "recommendations": [
    "ms-vscode.vscode-typescript-next"
  ]
}`,
			expectedWidth:    2,
			expectedNewlines: 0, // No newline after closing }
		},
		{
			name: "preserves 4-space indentation (no trailing newline)",
			existingFile: `{
    "recommendations": [
        "ms-vscode.vscode-typescript-next"
    ]
}`,
			expectedWidth:    4,
			expectedNewlines: 0, // No newline after closing }
		},
		{
			name: "converts tabs to 4 spaces (no trailing newline)",
			existingFile: `{
	"recommendations": [
		"ms-vscode.vscode-typescript-next"
	]
}`,
			expectedWidth:    4, // Tabs are converted to 4 spaces (Go json package limitation)
			expectedNewlines: 0, // No newline after closing }
		},
		{
			name: "preserves double trailing newline",
			existingFile: `{
  "recommendations": [
    "ms-vscode.vscode-typescript-next"
  ]
}

`,
			expectedWidth:    2,
			expectedNewlines: 2,
		},
		{
			name: "preserves no trailing newline",
			existingFile: `{
  "recommendations": [
    "ms-vscode.vscode-typescript-next"
  ]
}`,
			expectedWidth:    2,
			expectedNewlines: 0,
		},
		{
			name:             "defaults to 2 spaces and single newline for new file",
			existingFile:     "",
			expectedWidth:    2,
			expectedNewlines: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create existing file if specified
			if tt.existingFile != "" {
				if err := os.WriteFile(testFile, []byte(tt.existingFile), 0644); err != nil {
					t.Fatalf("Failed to write existing file: %v", err)
				}
			} else {
				// Remove file if it exists
				os.Remove(testFile)
			}

			// Write new content
			extensions := &Extensions{
				Recommendations: []string{"golang.go", "dbaeumer.vscode-eslint"},
			}

			if err := WriteJSONFile(testFile, extensions); err != nil {
				t.Fatalf("WriteJSONFile failed: %v", err)
			}

			// Read back and verify indentation
			content, err := os.ReadFile(testFile)
			if err != nil {
				t.Fatalf("Failed to read written file: %v", err)
			}

			detectedWidth := DetectIndentation(string(content))
			if detectedWidth != tt.expectedWidth {
				t.Errorf("Expected indentation width %d, got %d\nContent:\n%s",
					tt.expectedWidth, detectedWidth, string(content))
			}

			// Verify trailing newlines
			contentStr := string(content)
			trailingCount := 0
			for i := len(contentStr) - 1; i >= 0 && contentStr[i] == '\n'; i-- {
				trailingCount++
			}
			if trailingCount != tt.expectedNewlines {
				t.Errorf("Expected %d trailing newlines, got %d\nContent ends with: %q",
					tt.expectedNewlines, trailingCount, contentStr[len(contentStr)-10:])
			}

			// Verify content is valid JSON (strip trailing newlines for validation)
			var result Extensions
			trimmed := []byte(strings.TrimSpace(contentStr))
			if err := json.Unmarshal(trimmed, &result); err != nil {
				t.Fatalf("Written file is not valid JSON: %v", err)
			}

			if len(result.Recommendations) != 2 {
				t.Errorf("Expected 2 recommendations, got %d", len(result.Recommendations))
			}
		})
	}
}

func TestValidateSettingsKeys(t *testing.T) {
	tests := []struct {
		name         string
		keys         map[string]any
		wantErr      bool
		wantWarnings int
		errContains  string
	}{
		{
			name: "valid go keys",
			keys: map[string]any{
				"go.goroot":      "/path/to/go",
				"go.gopath":      "/path/to/gopath",
				"go.toolsGopath": "/path/to/tools",
			},
			wantErr:      false,
			wantWarnings: 0,
		},
		{
			name: "valid gopls keys",
			keys: map[string]any{
				"gopls": map[string]any{
					"formatting.gofumpt": true,
				},
				"gopls.ui.completion.usePlaceholders": true,
			},
			wantErr:      false,
			wantWarnings: 0,
		},
		{
			name: "invalid non-go key",
			keys: map[string]any{
				"editor.fontSize": 14,
				"go.goroot":       "/path/to/go",
			},
			wantErr:     true,
			errContains: "refusing to modify non-Go setting: editor.fontSize",
		},
		{
			name: "deprecated key",
			keys: map[string]any{
				"go.useLanguageServer": true,
			},
			wantErr:      false,
			wantWarnings: 1,
		},
		{
			name: "multiple deprecated keys",
			keys: map[string]any{
				"go.useLanguageServer":                  true,
				"go.languageServerExperimentalFeatures": map[string]bool{},
			},
			wantErr:      false,
			wantWarnings: 2,
		},
		{
			name: "mixed valid and deprecated",
			keys: map[string]any{
				"go.goroot":            "/path/to/go",
				"go.useLanguageServer": true,
			},
			wantErr:      false,
			wantWarnings: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			warnings, err := ValidateSettingsKeys(tt.keys)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("Expected error containing %q, got %q", tt.errContains, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			if len(warnings) != tt.wantWarnings {
				t.Errorf("Expected %d warnings, got %d: %v", tt.wantWarnings, len(warnings), warnings)
			}
		})
	}
}

func TestIsGoRelatedKey(t *testing.T) {
	tests := []struct {
		key  string
		want bool
	}{
		{"go.goroot", true},
		{"go.gopath", true},
		{"go.toolsGopath", true},
		{"gopls", true},
		{"gopls.formatting.gofumpt", true},
		{"gopls.ui.completion.usePlaceholders", true},
		{"editor.fontSize", false},
		{"python.pythonPath", false},
		{"terminal.integrated.shell.linux", false},
		{"files.autoSave", false},
	}

	for _, tt := range tests {
		t.Run(tt.key, func(t *testing.T) {
			got := IsGoRelatedKey(tt.key)
			if got != tt.want {
				t.Errorf("IsGoRelatedKey(%q) = %v, want %v", tt.key, got, tt.want)
			}
		})
	}
}

func TestFilterGoKeys(t *testing.T) {
	input := map[string]any{
		"go.goroot":                       "/path/to/go",
		"gopls.formatting.gofumpt":        true,
		"editor.fontSize":                 14,
		"python.pythonPath":               "/usr/bin/python3",
		"go.toolsGopath":                  "/path/to/tools",
		"terminal.integrated.shell.linux": "/bin/bash",
	}

	result := FilterGoKeys(input)

	// Should only have Go-related keys
	if len(result) != 3 {
		t.Errorf("Expected 3 keys, got %d: %v", len(result), result)
	}

	expectedKeys := []string{"go.goroot", "gopls.formatting.gofumpt", "go.toolsGopath"}
	for _, key := range expectedKeys {
		if _, exists := result[key]; !exists {
			t.Errorf("Expected key %q to be in result", key)
		}
	}

	// Should NOT have non-Go keys
	unexpectedKeys := []string{"editor.fontSize", "python.pythonPath", "terminal.integrated.shell.linux"}
	for _, key := range unexpectedKeys {
		if _, exists := result[key]; exists {
			t.Errorf("Did not expect key %q to be in result", key)
		}
	}
}

func TestConvertToWorkspacePaths(t *testing.T) {
	// Use platform-appropriate paths
	var workspaceRoot string
	var externalPath1, externalPath2 string

	if utils.IsWindows() {
		workspaceRoot = `C:\Users\adam\projects\myapp`
		externalPath1 = `C:\Program Files\Go`
		externalPath2 = `C:\Users\adam\go\tools`
	} else {
		workspaceRoot = "/Users/adam/projects/myapp"
		externalPath1 = "/usr/local/go"
		externalPath2 = "/Users/adam/go/tools"
	}

	tests := []struct {
		name     string
		settings map[string]any
		expected map[string]any
	}{
		{
			name: "convert absolute paths",
			settings: map[string]any{
				"go.goroot": filepath.Join(workspaceRoot, "sdk", "go"),
				"go.gopath": filepath.Join(workspaceRoot, "gopath"),
			},
			expected: map[string]any{
				"go.goroot": "${workspaceFolder}/sdk/go",
				"go.gopath": "${workspaceFolder}/gopath",
			},
		},
		{
			name: "leave external paths unchanged",
			settings: map[string]any{
				"go.goroot":      externalPath1,
				"go.toolsGopath": externalPath2,
			},
			expected: map[string]any{
				"go.goroot":      externalPath1,
				"go.toolsGopath": externalPath2,
			},
		},
		{
			name: "handle nested maps",
			settings: map[string]any{
				"gopls": map[string]any{
					"build.env": map[string]any{
						"GOROOT": filepath.Join(workspaceRoot, "sdk", "go"),
					},
				},
			},
			expected: map[string]any{
				"gopls": map[string]any{
					"build.env": map[string]any{
						"GOROOT": "${workspaceFolder}/sdk/go",
					},
				},
			},
		},
		{
			name: "preserve non-string values",
			settings: map[string]any{
				"go.goroot":                     filepath.Join(workspaceRoot, "go"),
				"go.toolsManagement.autoUpdate": true,
				"gopls": map[string]any{
					"formatting.gofumpt": true,
				},
			},
			expected: map[string]any{
				"go.goroot":                     "${workspaceFolder}/go",
				"go.toolsManagement.autoUpdate": true,
				"gopls": map[string]any{
					"formatting.gofumpt": true,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ConvertToWorkspacePaths(tt.settings, workspaceRoot)

			// Compare results
			resultJSON, _ := json.MarshalIndent(result, "", "  ")
			expectedJSON, _ := json.MarshalIndent(tt.expected, "", "  ")

			if string(resultJSON) != string(expectedJSON) {
				t.Errorf("Result mismatch:\nGot:\n%s\n\nExpected:\n%s", resultJSON, expectedJSON)
			}
		})
	}
}
