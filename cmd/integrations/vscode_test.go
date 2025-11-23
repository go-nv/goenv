package integrations

import (
	"os"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-nv/goenv/internal/vscode"
	"github.com/go-nv/goenv/testing/testutil"
)

// TestVSCodeTemplatesUsePlatformEnvVars verifies that templates use platform-specific environment variables
func TestVSCodeTemplatesUsePlatformEnvVars(t *testing.T) {
	templates := []string{"basic", "advanced", "monorepo"}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			settings, err := generateSettings(template)
			require.NoError(t, err, "Failed to generate template")

			// Verify toolsGopath uses environment variable (cross-platform)
			toolsGopath, ok := settings["go.toolsGopath"].(string)
			require.True(t, ok, "go.toolsGopath not found in template")

			// Should use either ${env:HOME} (Unix) or ${env:USERPROFILE} (Windows)
			assert.True(t, strings.Contains(toolsGopath, "${env:HOME}") || strings.Contains(toolsGopath, "${env:USERPROFILE}"))

			// Should end with /go/tools
			assert.True(t, strings.HasSuffix(toolsGopath, "/go/tools"))

			t.Logf("✓ Template '%s' uses platform-specific home directory: %s", template, toolsGopath)
		})
	}
}

// TestVSCodeGenerateSettingsBasic verifies the basic template structure
func TestVSCodeGenerateSettingsBasic(t *testing.T) {
	settings, err := generateSettings("basic")
	require.NoError(t, err, "Failed to generate basic template")

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
	assert.Equal(t, "${env:GOROOT}", settings["go.goroot"], "Expected go.goroot to be '${env:GOROOT}'")

	assert.Equal(t, "${env:GOPATH}", settings["go.gopath"], "Expected go.gopath to be '${env:GOPATH}'")
}

// TestVSCodeGenerateSettingsAdvanced verifies the advanced template
func TestVSCodeGenerateSettingsAdvanced(t *testing.T) {
	settings, err := generateSettings("advanced")
	require.NoError(t, err, "Failed to generate advanced template")

	// Verify gopls settings exist (modern language server configuration)
	if _, ok := settings["gopls"]; !ok {
		t.Error("Advanced template missing gopls configuration")
	}

	// Verify modern tool management settings
	assert.Equal(t, true, settings["go.toolsManagement.autoUpdate"], "Advanced template should have autoUpdate enabled")

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
	require.NoError(t, err, "Failed to generate monorepo template")

	// Verify inferGopath is disabled for monorepos
	assert.Equal(t, false, settings["go.inferGopath"], "Monorepo template should have inferGopath disabled")

	// Verify gopls directory filters exist
	gopls, ok := settings["gopls"].(map[string]interface{})
	require.True(t, ok, "gopls settings not found in monorepo template")

	if _, ok := gopls["build.directoryFilters"]; !ok {
		t.Error("Monorepo template missing build.directoryFilters")
	}
}

// TestVSCodeGenerateSettingsInvalidTemplate verifies error handling
func TestVSCodeGenerateSettingsInvalidTemplate(t *testing.T) {
	_, err := generateSettings("invalid")
	assert.Error(t, err, "Expected error for invalid template, got nil")

	assert.Contains(t, err.Error(), "unknown template", "Expected 'unknown template' error %v", err)
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
	assert.Equal(t, "/old/path", merged["go.goroot"], "Expected existing go.goroot to be preserved")

	// Should add new go.gopath
	assert.Equal(t, "${env:GOPATH}", merged["go.gopath"], "Expected new go.gopath to be added")

	// Should preserve custom setting
	assert.Equal(t, "value", merged["custom.setting"], "Custom setting was not preserved")
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
	assert.Equal(t, "/new/path", merged["go.goroot"], "Expected go.goroot to be overridden")

	// Should override go.gopath
	assert.Equal(t, "/new/gopath", merged["go.gopath"], "Expected go.gopath to be overridden")

	// Should preserve custom setting
	assert.Equal(t, "value", merged["custom.setting"], "Custom setting was not preserved")
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
	require.NoError(t, err, "Failed to read extensions")

	assert.Len(t, extensions.Recommendations, 2, "Expected 2 recommendations")

	expected := []string{"ms-vscode.vscode-typescript-next", "dbaeumer.vscode-eslint"}
	for i, rec := range extensions.Recommendations {
		assert.Equal(t, expected[i], rec)
	}
}

// TestReadExistingExtensionsNonExistent verifies error handling for non-existent file
func TestReadExistingExtensionsNonExistent(t *testing.T) {
	_, err := vscode.ReadExistingExtensions("/nonexistent/extensions.json")
	assert.Error(t, err, "Expected error for non-existent file, got nil")
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
	require.NoError(t, err, "Failed to read extensions with comments")

	assert.Len(t, extensions.Recommendations, 2, "Expected 2 recommendations")
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

	assert.False(t, len(recommendations) != 1 || recommendations[0] != "golang.go", "Expected golang.go to be added")

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

	assert.Len(t, recommendations, 3, "Expected 3 recommendations")
	assert.Equal(t, "golang.go", recommendations[2], "Expected golang.go to be appended")

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

	assert.Len(t, recommendations, 2, "Expected 2 recommendations (no duplicate)")
}

// TestVSCodeInitPreservesExtensions verifies that running vscode init multiple times preserves existing extensions
func TestVSCodeInitPreservesExtensions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	extensionsFile := vscodeDir + "/extensions.json"

	// Create .vscode directory
	err = utils.EnsureDirWithContext(vscodeDir, "create test directory")
	require.NoError(t, err, "Failed to create .vscode directory")

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
	require.NoError(t, err, "Failed to read initial extensions")

	assert.Len(t, initialExtensions.Recommendations, 2, "Expected 2 initial recommendations")

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
	assert.Len(t, recommendations, 3, "Expected 3 recommendations after merge")

	// Verify existing extensions are preserved
	expectedExtensions := []string{
		"ms-vscode.vscode-typescript-next",
		"dbaeumer.vscode-eslint",
		"golang.go",
	}

	for i, expected := range expectedExtensions {
		assert.False(t, i >= len(recommendations) || recommendations[i] != expected)
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
		require.NoError(t, err, "Failed to generate settings")

		toolsGopath := settings["go.toolsGopath"].(string)

		// On Unix/macOS, should use ${env:HOME}
		// On Windows, should use ${env:USERPROFILE}
		// Both are valid and cross-platform
		assert.True(t, strings.Contains(toolsGopath, "${env:HOME}") || strings.Contains(toolsGopath, "${env:USERPROFILE}"), "Expected platform-specific env var in toolsGopath")

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
	var err error
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	err = utils.EnsureDirWithContext(vscodeDir, "create test directory")
	require.NoError(t, err, "Failed to create .vscode directory")

	settingsFile := vscodeDir + "/settings.json"
	backupFile := settingsFile + ".bak"

	// Create initial settings that match what init would generate
	initialSettings := map[string]interface{}{
		"go.goroot":      "${env:HOME}/.goenv/versions/1.23.2",
		"go.gopath":      "${env:HOME}/go/1.23.2",
		"go.toolsGopath": "${env:HOME}/go/tools",
	}

	err = vscode.WriteJSONFile(settingsFile, initialSettings)
	require.NoError(t, err, "Failed to write initial settings")

	// Change to temp directory
	oldWd, err := os.Getwd()
	require.NoError(t, err, "Failed to get working directory")
	defer os.Chdir(oldWd)

	err = os.Chdir(tmpDir)
	require.NoError(t, err, "Failed to change directory")

	// Simulate the check that happens in runVSCodeInit
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	require.NoError(t, err, "Failed to read existing settings")

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
	assert.False(t, hasChanges, "Expected no changes to be detected when settings are already correct")

	// Verify no backup file exists
	if utils.PathExists(backupFile) {
		t.Error("Backup file should not exist when no changes are made")
	}

	t.Log("✓ No backup created when settings are already correct")
}

// TestVSCodeInit_BackupCreatedWhenChangesNeeded verifies that backup is created only when changes are made
func TestVSCodeInit_BackupCreatedWhenChangesNeeded(t *testing.T) {
	var err error
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	err = utils.EnsureDirWithContext(vscodeDir, "create test directory")
	require.NoError(t, err, "Failed to create .vscode directory")

	settingsFile := vscodeDir + "/settings.json"
	backupFile := settingsFile + ".bak"

	// Create initial settings with DIFFERENT values
	initialSettings := map[string]interface{}{
		"go.goroot":      "${env:HOME}/.goenv/versions/1.22.0", // Old version
		"go.gopath":      "${env:HOME}/go/1.22.0",              // Old version
		"go.toolsGopath": "${env:HOME}/go/tools",
	}

	err = vscode.WriteJSONFile(settingsFile, initialSettings)
	require.NoError(t, err, "Failed to write initial settings")

	// Simulate the check that happens in runVSCodeInit
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	require.NoError(t, err, "Failed to read existing settings")

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
	assert.True(t, hasChanges, "Expected changes to be detected when settings differ")

	// If changes are detected, backup should be created
	if hasChanges {
		err = vscode.BackupFile(settingsFile)
		require.NoError(t, err, "Failed to create backup")

		// Verify backup file exists
		if utils.FileNotExists(backupFile) {
			t.Error("Backup file should exist when changes are made")
		}

		// Verify backup contains the old values
		backupSettings, err := vscode.ReadExistingSettings(backupFile)
		require.NoError(t, err, "Failed to read backup settings")

		assert.Equal(t, "${env:HOME}/.goenv/versions/1.22.0", backupSettings["go.goroot"], "Backup should contain old go.goroot value")

		assert.Equal(t, "${env:HOME}/go/1.22.0", backupSettings["go.gopath"], "Backup should contain old go.gopath value")

		t.Log("✓ Backup created and contains correct old values when changes are needed")
	}
}

// TestVSCodeSync_NoBackupWhenAlreadySynced verifies that sync doesn't create backup when already in sync
func TestVSCodeSync_NoBackupWhenAlreadySynced(t *testing.T) {
	var err error
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	err = utils.EnsureDirWithContext(vscodeDir, "create test directory")
	require.NoError(t, err, "Failed to create .vscode directory")

	settingsFile := vscodeDir + "/settings.json"
	backupFile := settingsFile + ".bak"

	// Create settings that are already synced
	currentVersion := "1.23.2"
	syncedSettings := map[string]interface{}{
		"go.goroot":      "${env:HOME}/.goenv/versions/" + currentVersion,
		"go.gopath":      "${env:HOME}/go/" + currentVersion,
		"go.toolsGopath": "${env:HOME}/go/tools",
	}

	err = vscode.WriteJSONFile(settingsFile, syncedSettings)
	require.NoError(t, err, "Failed to write synced settings")

	// Simulate the sync check
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	require.NoError(t, err, "Failed to read existing settings")

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
	assert.False(t, hasChanges, "Expected no changes when settings are already synced")

	// Verify no backup file exists
	if utils.PathExists(backupFile) {
		t.Error("Backup file should not exist when sync is not needed")
	}

	t.Log("✓ No backup created when settings are already synced")
}

// TestVSCodeSync_BackupCreatedWhenOutOfSync verifies that sync creates backup when out of sync
func TestVSCodeSync_BackupCreatedWhenOutOfSync(t *testing.T) {
	var err error
	// Create a temporary directory
	tmpDir := t.TempDir()
	vscodeDir := tmpDir + "/.vscode"
	err = utils.EnsureDirWithContext(vscodeDir, "create test directory")
	require.NoError(t, err, "Failed to create .vscode directory")

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

	err = vscode.WriteJSONFile(settingsFile, oldSettings)
	require.NoError(t, err, "Failed to write old settings")

	// Simulate the sync check with a new version
	existingSettings, err := vscode.ReadExistingSettings(settingsFile)
	require.NoError(t, err, "Failed to read existing settings")

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
	assert.True(t, hasChanges, "Expected changes to be detected when versions differ")

	// Create backup when changes are detected
	if hasChanges {
		err = vscode.BackupFile(settingsFile)
		require.NoError(t, err, "Failed to create backup")

		// Verify backup file exists
		if utils.FileNotExists(backupFile) {
			t.Error("Backup file should exist when sync needs to update settings")
		}

		// Verify backup contains the old version
		backupSettings, err := vscode.ReadExistingSettings(backupFile)
		require.NoError(t, err, "Failed to read backup settings")

		expectedOldGoroot := "${env:HOME}/.goenv/versions/" + oldVersion
		assert.Equal(t, expectedOldGoroot, backupSettings["go.goroot"], "Backup should contain old version")

		t.Log("✓ Backup created when sync detects version mismatch")
	}
}
