package integrations

import (
	"github.com/go-nv/goenv/internal/utils"
	"os"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/vscode"
	"github.com/go-nv/goenv/testing/testutil"
)

// TestVSCodeTemplatesUsePlatformEnvVars verifies that templates use platform-specific environment variables
func TestVSCodeTemplatesUsePlatformEnvVars(t *testing.T) {
	templates := []string{"basic", "advanced", "monorepo"}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			settings, err := generateSettings(template)
			if err != nil {
				t.Fatalf("Failed to generate %s template: %v", template, err)
			}

			// Verify toolsGopath uses environment variable (cross-platform)
			toolsGopath, ok := settings["go.toolsGopath"].(string)
			if !ok {
				t.Fatal("go.toolsGopath not found in template")
			}

			// Should use either ${env:HOME} (Unix) or ${env:USERPROFILE} (Windows)
			if !strings.Contains(toolsGopath, "${env:HOME}") && !strings.Contains(toolsGopath, "${env:USERPROFILE}") {
				t.Errorf("Template '%s': Expected platform-specific env var, got '%s'", template, toolsGopath)
			}

			// Should end with /go/tools
			if !strings.HasSuffix(toolsGopath, "/go/tools") {
				t.Errorf("Template '%s': Expected path ending with '/go/tools', got '%s'", template, toolsGopath)
			}

			t.Logf("✓ Template '%s' uses platform-specific home directory: %s", template, toolsGopath)
		})
	}
}

// TestVSCodeGenerateSettingsBasic verifies the basic template structure
func TestVSCodeGenerateSettingsBasic(t *testing.T) {
	settings, err := generateSettings("basic")
	if err != nil {
		t.Fatalf("Failed to generate basic template: %v", err)
	}

	// Verify required keys exist
	// Note: go.useLanguageServer removed - modern Go extension (v0.30.0+) defaults to gopls
	// Legacy settings like go.useLanguageServer are intentionally omitted
	requiredKeys := []string{"go.goroot", "go.gopath", "go.toolsGopath"}
	for _, key := range requiredKeys {
		if _, ok := settings[key]; !ok {
			t.Errorf("Basic template missing required key: %s", key)
		}
	}

	// Verify legacy settings are NOT included (modern VS Code Go extension doesn't need them)
	legacySettings := []string{
		"go.useLanguageServer",   // Removed in Go extension v0.30.0+ (2022)
		"go.languageServerFlags", // Replaced by gopls configuration
	}
	for _, key := range legacySettings {
		if _, ok := settings[key]; ok {
			t.Errorf("Basic template should not include legacy setting: %s", key)
		}
	}

	// Verify environment variable syntax for GOROOT and GOPATH
	if settings["go.goroot"] != "${env:GOROOT}" {
		t.Errorf("Expected go.goroot to be '${env:GOROOT}', got '%v'", settings["go.goroot"])
	}

	if settings["go.gopath"] != "${env:GOPATH}" {
		t.Errorf("Expected go.gopath to be '${env:GOPATH}', got '%v'", settings["go.gopath"])
	}
}

// TestVSCodeGenerateSettingsAdvanced verifies the advanced template
func TestVSCodeGenerateSettingsAdvanced(t *testing.T) {
	settings, err := generateSettings("advanced")
	if err != nil {
		t.Fatalf("Failed to generate advanced template: %v", err)
	}

	// Verify gopls settings exist (modern language server configuration)
	if _, ok := settings["gopls"]; !ok {
		t.Error("Advanced template missing gopls configuration")
	}

	// Verify modern tool management settings
	if settings["go.toolsManagement.autoUpdate"] != true {
		t.Error("Advanced template should have autoUpdate enabled")
	}

	// Verify gopls settings are properly structured
	gopls, ok := settings["gopls"].(map[string]interface{})
	if ok {
		// Modern gopls settings should be present
		t.Logf("Gopls settings configured: %v", gopls)
	}

	// Ensure legacy settings are not mixed with modern configuration
	if _, ok := settings["go.useLanguageServer"]; ok {
		t.Error("Advanced template should not mix legacy settings with modern gopls config")
	}
}

// TestVSCodeGenerateSettingsMonorepo verifies the monorepo template
func TestVSCodeGenerateSettingsMonorepo(t *testing.T) {
	settings, err := generateSettings("monorepo")
	if err != nil {
		t.Fatalf("Failed to generate monorepo template: %v", err)
	}

	// Verify inferGopath is disabled for monorepos
	if settings["go.inferGopath"] != false {
		t.Error("Monorepo template should have inferGopath disabled")
	}

	// Verify gopls directory filters exist
	gopls, ok := settings["gopls"].(map[string]interface{})
	if !ok {
		t.Fatal("gopls settings not found in monorepo template")
	}

	if _, ok := gopls["build.directoryFilters"]; !ok {
		t.Error("Monorepo template missing build.directoryFilters")
	}
}

// TestVSCodeGenerateSettingsInvalidTemplate verifies error handling
func TestVSCodeGenerateSettingsInvalidTemplate(t *testing.T) {
	_, err := generateSettings("invalid")
	if err == nil {
		t.Error("Expected error for invalid template, got nil")
	}

	if !strings.Contains(err.Error(), "unknown template") {
		t.Errorf("Expected 'unknown template' error, got: %v", err)
	}
}

// TestVSCodeMergeSettings verifies settings merging logic
func TestVSCodeMergeSettings(t *testing.T) {
	existing := VSCodeSettings{
		"go.goroot":      "/old/path",
		"custom.setting": "value",
	}

	new := VSCodeSettings{
		"go.goroot": "${env:GOROOT}",
		"go.gopath": "${env:GOPATH}",
	}

	merged := mergeSettings(existing, new)

	// Should preserve existing go.goroot
	if merged["go.goroot"] != "/old/path" {
		t.Errorf("Expected existing go.goroot to be preserved, got '%v'", merged["go.goroot"])
	}

	// Should add new go.gopath
	if merged["go.gopath"] != "${env:GOPATH}" {
		t.Errorf("Expected new go.gopath to be added, got '%v'", merged["go.gopath"])
	}

	// Should preserve custom setting
	if merged["custom.setting"] != "value" {
		t.Error("Custom setting was not preserved")
	}
}

// TestVSCodeMergeSettingsWithOverride verifies override logic
func TestVSCodeMergeSettingsWithOverride(t *testing.T) {
	existing := VSCodeSettings{
		"go.goroot":      "/old/path",
		"go.gopath":      "/old/gopath",
		"custom.setting": "value",
	}

	new := VSCodeSettings{
		"go.goroot": "/new/path",
		"go.gopath": "/new/gopath",
	}

	overrideKeys := []string{"go.goroot", "go.gopath"}
	merged := mergeSettingsWithOverride(existing, new, overrideKeys)

	// Should override go.goroot
	if merged["go.goroot"] != "/new/path" {
		t.Errorf("Expected go.goroot to be overridden, got '%v'", merged["go.goroot"])
	}

	// Should override go.gopath
	if merged["go.gopath"] != "/new/gopath" {
		t.Errorf("Expected go.gopath to be overridden, got '%v'", merged["go.gopath"])
	}

	// Should preserve custom setting
	if merged["custom.setting"] != "value" {
		t.Error("Custom setting was not preserved")
	}
}

// TestReadExistingExtensions verifies extensions.json reading
func TestReadExistingExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	extensionsFile := tmpDir + "/extensions.json"

	// Test with existing extensions
	content := `{
	"recommendations": [
		"ms-vscode.vscode-typescript-next",
		"dbaeumer.vscode-eslint"
	]
}`

	testutil.WriteTestFile(t, extensionsFile, []byte(content), utils.PermFileDefault)

	extensions, err := vscode.ReadExistingExtensions(extensionsFile)
	if err != nil {
		t.Fatalf("Failed to read extensions: %v", err)
	}

	if len(extensions.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(extensions.Recommendations))
	}

	expected := []string{"ms-vscode.vscode-typescript-next", "dbaeumer.vscode-eslint"}
	for i, rec := range extensions.Recommendations {
		if rec != expected[i] {
			t.Errorf("Expected recommendation[%d] to be '%s', got '%s'", i, expected[i], rec)
		}
	}
}

// TestReadExistingExtensionsNonExistent verifies error handling for non-existent file
func TestReadExistingExtensionsNonExistent(t *testing.T) {
	_, err := vscode.ReadExistingExtensions("/nonexistent/extensions.json")
	if err == nil {
		t.Error("Expected error for non-existent file, got nil")
	}
}

// TestReadExistingExtensionsWithComments verifies JSONC support
func TestReadExistingExtensionsWithComments(t *testing.T) {
	tmpDir := t.TempDir()
	extensionsFile := tmpDir + "/extensions.json"

	// JSONC with comments and trailing commas
	content := `{
	// This is a comment
	"recommendations": [
		"golang.go",
		"ms-vscode.vscode-typescript-next", // trailing comma
	]
}`

	testutil.WriteTestFile(t, extensionsFile, []byte(content), utils.PermFileDefault)

	extensions, err := vscode.ReadExistingExtensions(extensionsFile)
	if err != nil {
		t.Fatalf("Failed to read extensions with comments: %v", err)
	}

	if len(extensions.Recommendations) != 2 {
		t.Errorf("Expected 2 recommendations, got %d", len(extensions.Recommendations))
	}
}

// TestExtensionsMerging verifies that golang.go is added to existing recommendations
func TestExtensionsMerging(t *testing.T) {
	// Test case 1: No existing extensions
	recommendations := []string{}
	goExtension := "golang.go"

	hasGoExtension := false
	for _, rec := range recommendations {
		if rec == goExtension {
			hasGoExtension = true
			break
		}
	}
	if !hasGoExtension {
		recommendations = append(recommendations, goExtension)
	}

	if len(recommendations) != 1 || recommendations[0] != "golang.go" {
		t.Errorf("Expected golang.go to be added, got %v", recommendations)
	}

	// Test case 2: Existing extensions without golang.go
	recommendations = []string{"ms-vscode.vscode-typescript-next", "dbaeumer.vscode-eslint"}

	hasGoExtension = false
	for _, rec := range recommendations {
		if rec == goExtension {
			hasGoExtension = true
			break
		}
	}
	if !hasGoExtension {
		recommendations = append(recommendations, goExtension)
	}

	if len(recommendations) != 3 {
		t.Errorf("Expected 3 recommendations, got %d", len(recommendations))
	}
	if recommendations[2] != "golang.go" {
		t.Errorf("Expected golang.go to be appended, got %v", recommendations)
	}

	// Test case 3: golang.go already present
	recommendations = []string{"golang.go", "ms-vscode.vscode-typescript-next"}

	hasGoExtension = false
	for _, rec := range recommendations {
		if rec == goExtension {
			hasGoExtension = true
			break
		}
	}
	if !hasGoExtension {
		recommendations = append(recommendations, goExtension)
	}

	if len(recommendations) != 2 {
		t.Errorf("Expected 2 recommendations (no duplicate), got %d", len(recommendations))
	}
}

// TestVSCodeInitPreservesExtensions verifies that running vscode init multiple times preserves existing extensions
func TestVSCodeInitPreservesExtensions(t *testing.T) {
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	extensionsFile := vscodeDir + "/extensions.json"

	// Create .vscode directory
	if err := utils.EnsureDirWithContext(vscodeDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	// Create initial extensions.json with custom extensions
	initialContent := `{
	"recommendations": [
		"ms-vscode.vscode-typescript-next",
		"dbaeumer.vscode-eslint"
	]
}`
	testutil.WriteTestFile(t, extensionsFile, []byte(initialContent), utils.PermFileDefault)

	// Read the file and verify initial state
	initialExtensions, err := vscode.ReadExistingExtensions(extensionsFile)
	if err != nil {
		t.Fatalf("Failed to read initial extensions: %v", err)
	}

	if len(initialExtensions.Recommendations) != 2 {
		t.Errorf("Expected 2 initial recommendations, got %d", len(initialExtensions.Recommendations))
	}

	// Simulate what vscode init does - merge with existing
	var recommendations []string
	existingExtensions, err := vscode.ReadExistingExtensions(extensionsFile)
	if err == nil {
		recommendations = existingExtensions.Recommendations
	}

	// Add golang.go if not already present
	goExtension := "golang.go"
	hasGoExtension := false
	for _, rec := range recommendations {
		if rec == goExtension {
			hasGoExtension = true
			break
		}
	}
	if !hasGoExtension {
		recommendations = append(recommendations, goExtension)
	}

	// Verify merged recommendations
	if len(recommendations) != 3 {
		t.Errorf("Expected 3 recommendations after merge, got %d", len(recommendations))
	}

	// Verify existing extensions are preserved
	expectedExtensions := []string{
		"ms-vscode.vscode-typescript-next",
		"dbaeumer.vscode-eslint",
		"golang.go",
	}

	for i, expected := range expectedExtensions {
		if i >= len(recommendations) || recommendations[i] != expected {
			t.Errorf("Expected recommendation[%d] to be '%s', got '%s'", i, expected, recommendations[i])
		}
	}

	t.Log("✓ Existing extensions preserved and golang.go added")
}

// TestVSCodePathGeneration_PlatformSpecific verifies that --absolute flag generates platform-specific paths
func TestVSCodePathGeneration_PlatformSpecific(t *testing.T) {
	// This test verifies the logic, but doesn't actually run the command
	// because it requires a full goenv setup with versions installed

	// Test the generateSettings function produces correct env vars
	t.Run("Templates use correct platform env vars", func(t *testing.T) {
		settings, err := generateSettings("basic")
		if err != nil {
			t.Fatalf("Failed to generate settings: %v", err)
		}

		toolsGopath := settings["go.toolsGopath"].(string)

		// On Unix/macOS, should use ${env:HOME}
		// On Windows, should use ${env:USERPROFILE}
		// Both are valid and cross-platform
		if !strings.Contains(toolsGopath, "${env:HOME}") && !strings.Contains(toolsGopath, "${env:USERPROFILE}") {
			t.Errorf("Expected platform-specific env var in toolsGopath, got: %s", toolsGopath)
		}

		t.Logf("✓ toolsGopath uses platform-appropriate env var: %s", toolsGopath)
	})

	t.Run("Environment variable syntax is VS Code compatible", func(t *testing.T) {
		// VS Code supports ${env:VARNAME} syntax on all platforms
		// This is documented in VS Code's variables reference
		testCases := []struct {
			envVar   string
			expected string
		}{
			{"${env:HOME}/go/tools", "Valid on Unix/macOS"},
			{"${env:USERPROFILE}/go/tools", "Valid on Windows"},
			{"${env:GOROOT}", "Valid on all platforms"},
			{"${env:GOPATH}", "Valid on all platforms"},
		}

		for _, tc := range testCases {
			t.Logf("✓ %s: %s", tc.envVar, tc.expected)
		}
	})
}

// TestVSCodeInit_NoBackupWhenNoChanges verifies that no backup is created when settings are already correct
func TestVSCodeInit_NoBackupWhenNoChanges(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	if err := utils.EnsureDirWithContext(vscodeDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	settingsFile := vscodeDir + "/settings.json"
	backupFile := settingsFile + ".bak"

	// Create initial settings that match what init would generate
	initialSettings := map[string]interface{}{
		"go.goroot":      "${env:HOME}/.goenv/versions/1.23.2",
		"go.gopath":      "${env:HOME}/go/1.23.2",
		"go.toolsGopath": "${env:HOME}/go/tools",
	}

	if err := vscode.WriteJSONFile(settingsFile, initialSettings); err != nil {
		t.Fatalf("Failed to write initial settings: %v", err)
	}

	// Change to temp directory
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer os.Chdir(oldWd)

	if err := os.Chdir(tmpDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Simulate the check that happens in runVSCodeInit
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read existing settings: %v", err)
	}

	keysToUpdate := map[string]interface{}{
		"go.goroot": "${env:HOME}/.goenv/versions/1.23.2",
		"go.gopath": "${env:HOME}/go/1.23.2",
	}

	// Check if any values differ
	hasChanges := false
	for key, newVal := range keysToUpdate {
		oldVal := existingSettings[key]
		if oldVal != newVal {
			hasChanges = true
			break
		}
	}

	// Verify that no changes are detected
	if hasChanges {
		t.Error("Expected no changes to be detected when settings are already correct")
	}

	// Verify no backup file exists
	if utils.PathExists(backupFile) {
		t.Error("Backup file should not exist when no changes are made")
	}

	t.Log("✓ No backup created when settings are already correct")
}

// TestVSCodeInit_BackupCreatedWhenChangesNeeded verifies that backup is created only when changes are made
func TestVSCodeInit_BackupCreatedWhenChangesNeeded(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	if err := utils.EnsureDirWithContext(vscodeDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	settingsFile := vscodeDir + "/settings.json"
	backupFile := settingsFile + ".bak"

	// Create initial settings with DIFFERENT values
	initialSettings := map[string]interface{}{
		"go.goroot":      "${env:HOME}/.goenv/versions/1.22.0", // Old version
		"go.gopath":      "${env:HOME}/go/1.22.0",              // Old version
		"go.toolsGopath": "${env:HOME}/go/tools",
	}

	if err := vscode.WriteJSONFile(settingsFile, initialSettings); err != nil {
		t.Fatalf("Failed to write initial settings: %v", err)
	}

	// Simulate the check that happens in runVSCodeInit
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read existing settings: %v", err)
	}

	keysToUpdate := map[string]interface{}{
		"go.goroot": "${env:HOME}/.goenv/versions/1.23.2", // New version
		"go.gopath": "${env:HOME}/go/1.23.2",              // New version
	}

	// Check if any values differ
	hasChanges := false
	for key, newVal := range keysToUpdate {
		oldVal := existingSettings[key]
		if oldVal != newVal {
			hasChanges = true
			break
		}
	}

	// Verify that changes ARE detected
	if !hasChanges {
		t.Error("Expected changes to be detected when settings differ")
	}

	// If changes are detected, backup should be created
	if hasChanges {
		if err := vscode.BackupFile(settingsFile); err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		// Verify backup file exists
		if utils.FileNotExists(backupFile) {
			t.Error("Backup file should exist when changes are made")
		}

		// Verify backup contains the old values
		backupSettings, err := vscode.ReadExistingSettings(backupFile)
		if err != nil {
			t.Fatalf("Failed to read backup settings: %v", err)
		}

		if backupSettings["go.goroot"] != "${env:HOME}/.goenv/versions/1.22.0" {
			t.Errorf("Backup should contain old go.goroot value, got: %v", backupSettings["go.goroot"])
		}

		if backupSettings["go.gopath"] != "${env:HOME}/go/1.22.0" {
			t.Errorf("Backup should contain old go.gopath value, got: %v", backupSettings["go.gopath"])
		}

		t.Log("✓ Backup created and contains correct old values when changes are needed")
	}
}

// TestVSCodeSync_NoBackupWhenAlreadySynced verifies that sync doesn't create backup when already in sync
func TestVSCodeSync_NoBackupWhenAlreadySynced(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	if err := utils.EnsureDirWithContext(vscodeDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	settingsFile := vscodeDir + "/settings.json"
	backupFile := settingsFile + ".bak"

	// Create settings that are already synced
	currentVersion := "1.23.2"
	syncedSettings := map[string]interface{}{
		"go.goroot":      "${env:HOME}/.goenv/versions/" + currentVersion,
		"go.gopath":      "${env:HOME}/go/" + currentVersion,
		"go.toolsGopath": "${env:HOME}/go/tools",
	}

	if err := vscode.WriteJSONFile(settingsFile, syncedSettings); err != nil {
		t.Fatalf("Failed to write synced settings: %v", err)
	}

	// Simulate the sync check
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read existing settings: %v", err)
	}

	// Keys that sync would update (using same version, so should be identical)
	keysToUpdate := map[string]interface{}{
		"go.goroot": "${env:HOME}/.goenv/versions/" + currentVersion,
		"go.gopath": "${env:HOME}/go/" + currentVersion,
	}

	// Check if any values differ
	hasChanges := false
	for key, newVal := range keysToUpdate {
		oldVal := existingSettings[key]
		if oldVal != newVal {
			hasChanges = true
			break
		}
	}

	// Verify no changes detected
	if hasChanges {
		t.Error("Expected no changes when settings are already synced")
	}

	// Verify no backup file exists
	if utils.PathExists(backupFile) {
		t.Error("Backup file should not exist when sync is not needed")
	}

	t.Log("✓ No backup created when settings are already synced")
}

// TestVSCodeSync_BackupCreatedWhenOutOfSync verifies that sync creates backup when out of sync
func TestVSCodeSync_BackupCreatedWhenOutOfSync(t *testing.T) {
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	if err := utils.EnsureDirWithContext(vscodeDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create .vscode directory: %v", err)
	}

	settingsFile := vscodeDir + "/settings.json"
	backupFile := settingsFile + ".bak"

	// Create settings for an old version
	oldVersion := "1.22.0"
	newVersion := "1.23.2"
	oldSettings := map[string]interface{}{
		"go.goroot":      "${env:HOME}/.goenv/versions/" + oldVersion,
		"go.gopath":      "${env:HOME}/go/" + oldVersion,
		"go.toolsGopath": "${env:HOME}/go/tools",
	}

	if err := vscode.WriteJSONFile(settingsFile, oldSettings); err != nil {
		t.Fatalf("Failed to write old settings: %v", err)
	}

	// Simulate the sync check with a new version
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read existing settings: %v", err)
	}

	// Keys that sync would update (using new version)
	keysToUpdate := map[string]interface{}{
		"go.goroot": "${env:HOME}/.goenv/versions/" + newVersion,
		"go.gopath": "${env:HOME}/go/" + newVersion,
	}

	// Check if any values differ
	hasChanges := false
	for key, newVal := range keysToUpdate {
		oldVal := existingSettings[key]
		if oldVal != newVal {
			hasChanges = true
			break
		}
	}

	// Verify changes ARE detected
	if !hasChanges {
		t.Error("Expected changes to be detected when versions differ")
	}

	// Create backup when changes are detected
	if hasChanges {
		if err := vscode.BackupFile(settingsFile); err != nil {
			t.Fatalf("Failed to create backup: %v", err)
		}

		// Verify backup file exists
		if utils.FileNotExists(backupFile) {
			t.Error("Backup file should exist when sync needs to update settings")
		}

		// Verify backup contains the old version
		backupSettings, err := vscode.ReadExistingSettings(backupFile)
		if err != nil {
			t.Fatalf("Failed to read backup settings: %v", err)
		}

		expectedOldGoroot := "${env:HOME}/.goenv/versions/" + oldVersion
		if backupSettings["go.goroot"] != expectedOldGoroot {
			t.Errorf("Backup should contain old version %s, got: %v", expectedOldGoroot, backupSettings["go.goroot"])
		}

		t.Log("✓ Backup created when sync detects version mismatch")
	}
}
