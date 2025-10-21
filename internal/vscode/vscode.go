package vscode

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/tidwall/jsonc"
	"github.com/tidwall/pretty"
	"github.com/tidwall/sjson"
)

// CheckResult contains the result of a VS Code settings check
type CheckResult struct {
	// HasSettings indicates if any go.* settings exist
	HasSettings bool

	// UsesEnvVars indicates if settings use ${env:GOROOT} pattern
	UsesEnvVars bool

	// ConfiguredVersion is the version detected from absolute paths (empty if using env vars)
	ConfiguredVersion string

	// Mismatch indicates if the configured version doesn't match expected
	Mismatch bool

	// ExpectedVersion is the version that should be configured
	ExpectedVersion string

	// SettingsPath is the path to the settings.json file
	SettingsPath string
}

// CheckSettings examines VS Code settings.json and validates Go configuration
// Returns a CheckResult with detailed information about the VS Code integration status
func CheckSettings(settingsPath string, expectedVersion string) CheckResult {
	result := CheckResult{
		ExpectedVersion: expectedVersion,
		SettingsPath:    settingsPath,
	}

	data, err := os.ReadFile(settingsPath)
	if err != nil {
		// File doesn't exist or can't be read
		return result
	}

	// Strip comments and trailing commas (VS Code uses JSONC format)
	data = jsonc.ToJSON(data)

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		// Invalid JSON - treat as if no Go settings exist
		return result
	}

	// Check if go.goroot exists at all
	goroot, hasGoRoot := settings["go.goroot"]
	if !hasGoRoot {
		// No Go settings configured
		return result
	}

	result.HasSettings = true

	// Check if using environment variables (these are correct, no mismatch)
	// Only ${env:GOROOT} is truly version-agnostic
	// ${env:HOME} or ${env:USERPROFILE} + hardcoded version still needs updating
	if gorootStr, ok := goroot.(string); ok {
		if strings.Contains(gorootStr, "${env:GOROOT}") {
			result.UsesEnvVars = true
			return result
		}

		// Extract version from GOROOT path
		// Path format: /Users/username/.goenv/versions/1.23.2 (Unix)
		//          or: C:\Users\username\.goenv\versions\1.23.2 (Windows)
		// Normalize backslashes to forward slashes for consistent parsing
		normalizedPath := strings.ReplaceAll(gorootStr, `\`, `/`)
		parts := strings.Split(normalizedPath, "/")
		for i, part := range parts {
			if part == "versions" && i+1 < len(parts) {
				currentVersion := parts[i+1]
				result.ConfiguredVersion = currentVersion
				if currentVersion != expectedVersion {
					result.Mismatch = true
				}
				return result
			}
		}
	}

	return result
}

// DetectIndentation detects the indentation width from a JSON file
// Returns the number of spaces used for indentation (defaults to 2 if can't detect)
func DetectIndentation(content string) int {
	lines := strings.Split(content, "\n")

	// Look for the first indented line (not the opening brace)
	for _, line := range lines {
		if len(line) == 0 {
			continue
		}

		// Skip lines that start with { or }
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || trimmed[0] == '{' || trimmed[0] == '}' {
			continue
		}

		// Count leading spaces
		spaces := 0
		for _, ch := range line {
			if ch == ' ' {
				spaces++
			} else if ch == '\t' {
				// If tabs are used, convert to 4 spaces equivalent
				return 4
			} else {
				break
			}
		}

		// If we found indentation, return it
		if spaces > 0 {
			return spaces
		}
	}

	// Default to 2 spaces if we couldn't detect
	return 2
}

// EscapeJSONKey escapes dots in a key name for sjson
// sjson treats dots as path separators, so "go.goroot" becomes nested
// We need to escape dots with backslashes: "go\.goroot"
func EscapeJSONKey(key string) string {
	return strings.ReplaceAll(key, ".", `\.`)
}

// UpdateJSONKeys updates specific keys in a JSON file while preserving key order
//
// Behavior:
//   - Preserves the order of existing keys
//   - Updates only the specified keys
//   - Adds new keys at the end if they don't exist
//   - Detects and preserves original indentation (2 or 4 spaces)
//   - Strips JSONC features (comments, trailing commas)
//   - May reformat arrays to single line if short
//
// Note: Some formatting changes occur because:
//  1. No Go library supports true JSONC round-tripping
//  2. VS Code itself strips comments/trailing commas when modifying settings via UI
//  3. The key order and indentation preservation are more important for readability
func UpdateJSONKeys(path string, keysToUpdate map[string]interface{}) error {
	// Read original file (may be JSONC)
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}

	// Detect indentation from original file
	indentWidth := DetectIndentation(string(data))

	// Convert JSONC to clean JSON
	jsonData := string(jsonc.ToJSON(data))

	// Apply updates using sjson (preserves key order)
	// Note: Dots in key names must be escaped with backslash
	for key, value := range keysToUpdate {
		// Escape dots in the key name so sjson treats it as a literal key, not a path
		escapedKey := EscapeJSONKey(key)
		jsonData, err = sjson.Set(jsonData, escapedKey, value)
		if err != nil {
			return fmt.Errorf("failed to set %s: %w", key, err)
		}
	}

	// Pretty-print the JSON with the detected indentation
	opts := &pretty.Options{
		Width:    80,
		Prefix:   "",
		Indent:   strings.Repeat(" ", indentWidth),
		SortKeys: false,
	}
	prettyJSON := pretty.PrettyOptions([]byte(jsonData), opts)

	// Write back with proper formatting
	if err := os.WriteFile(path, prettyJSON, 0644); err != nil {
		return err
	}

	return nil
}

// FindSettingsFile returns the path to VS Code settings.json in the current or specified directory
func FindSettingsFile(dir string) (string, error) {
	if dir == "" {
		var err error
		dir, err = os.Getwd()
		if err != nil {
			return "", fmt.Errorf("failed to get current directory: %w", err)
		}
	}

	vscodeDir := filepath.Join(dir, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")

	return settingsFile, nil
}

// HasVSCodeDirectory checks if a .vscode directory exists in the specified directory
func HasVSCodeDirectory(dir string) bool {
	if dir == "" {
		dir, _ = os.Getwd()
	}
	vscodeDir := filepath.Join(dir, ".vscode")
	info, err := os.Stat(vscodeDir)
	return err == nil && info.IsDir()
}

// ReadExistingSettings reads and parses existing settings.json
func ReadExistingSettings(path string) (map[string]interface{}, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Convert JSONC to JSON (handles comments and trailing commas)
	data = jsonc.ToJSON(data)

	var settings map[string]interface{}
	if err := json.Unmarshal(data, &settings); err != nil {
		return nil, fmt.Errorf("invalid JSON in existing settings: %w", err)
	}

	return settings, nil
}

// Extensions represents the VS Code extensions.json structure
type Extensions struct {
	Recommendations []string `json:"recommendations"`
}

// ReadExistingExtensions reads and parses existing extensions.json
func ReadExistingExtensions(path string) (*Extensions, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	// Convert JSONC to JSON (handles comments and trailing commas)
	data = jsonc.ToJSON(data)

	var extensions Extensions
	if err := json.Unmarshal(data, &extensions); err != nil {
		return nil, fmt.Errorf("invalid JSON in existing extensions: %w", err)
	}

	return &extensions, nil
}

// WriteJSONFile writes a JSON file with proper formatting
// Detects and preserves existing indentation and trailing newlines if file exists
// Defaults to 2 spaces and single trailing newline for new files
func WriteJSONFile(path string, data interface{}) error {
	// Detect indentation and trailing newlines from existing file if it exists
	indentWidth := 2      // Default to 2 spaces
	trailingNewlines := 1 // Default to single newline
	if existingData, err := os.ReadFile(path); err == nil {
		indentWidth = DetectIndentation(string(existingData))

		// Count trailing newlines in existing file
		content := string(existingData)
		if len(content) > 0 {
			trailingNewlines = 0
			for i := len(content) - 1; i >= 0 && content[i] == '\n'; i-- {
				trailingNewlines++
			}
		}
	}

	indent := strings.Repeat(" ", indentWidth)
	jsonData, err := json.MarshalIndent(data, "", indent)
	if err != nil {
		return err
	}

	// json.MarshalIndent doesn't add a trailing newline, so we add them based on original
	// Add the appropriate number of trailing newlines
	var output []byte
	if trailingNewlines > 0 {
		output = append(jsonData, []byte(strings.Repeat("\n", trailingNewlines))...)
	} else {
		output = jsonData
	}

	if err := os.WriteFile(path, output, 0644); err != nil {
		return err
	}

	return nil
}
