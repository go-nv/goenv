package shell

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSetupCommand_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.EnvVarHome, tmpDir)
	t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME
	t.Setenv(utils.EnvVarShell, "/bin/bash")

	// Set dry-run flag
	setupFlags.dryRun = true
	setupFlags.skipVSCode = true
	setupFlags.yes = true
	defer func() {
		setupFlags.dryRun = false
		setupFlags.skipVSCode = false
		setupFlags.yes = false
	}()

	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	err := runSetup(setupCmd, []string{})
	require.NoError(t, err, "runSetup() unexpected error")

	output := buf.String()
	assert.Contains(t, output, "DRY RUN", "Expected '[DRY RUN]' in output %v", output)

	// Verify no actual files were created in dry-run
	bashrc := filepath.Join(tmpDir, ".bashrc")
	if utils.PathExists(bashrc) {
		t.Error("Dry-run should not create .bashrc file")
	}
}

func TestSetupCommand_AutoYes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.EnvVarHome, tmpDir)
	t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME
	t.Setenv(utils.EnvVarShell, "/bin/bash")

	setupFlags.yes = true
	setupFlags.skipVSCode = true
	defer func() {
		setupFlags.yes = false
		setupFlags.skipVSCode = false
	}()

	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	err := runSetup(setupCmd, []string{})
	require.NoError(t, err, "runSetup() unexpected error")

	output := buf.String()
	// Should complete without prompts
	assert.NotContains(t, output, "[y/N]", "Auto-yes mode should not show prompts")
}

func TestSetupCommand_AlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.EnvVarHome, tmpDir)
	t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME
	t.Setenv(utils.EnvVarShell, "/bin/bash")
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash") // Already initialized

	setupFlags.yes = true
	setupFlags.skipVSCode = true
	defer func() {
		setupFlags.yes = false
		setupFlags.skipVSCode = false
	}()

	// Create .bashrc with goenv init
	bashrc := filepath.Join(tmpDir, ".bashrc")
	content := `# goenv initialization
eval "$(goenv init -)"
`
	testutil.WriteTestFile(t, bashrc, []byte(content), utils.PermFileSecure)

	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	err := runSetup(setupCmd, []string{})
	require.NoError(t, err, "runSetup() unexpected error")

	output := buf.String()
	assert.False(t, !strings.Contains(output, "already") || !strings.Contains(output, "configured"), "Expected 'already configured' message")
}

func TestWriteVSCodeSettings_NewFile(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	settings := map[string]interface{}{
		"go.goroot": "${env:GOROOT}",
		"go.gopath": "${env:GOPATH}",
	}

	err = writeVSCodeSettings(settingsFile, settings)
	require.NoError(t, err, "writeVSCodeSettings() unexpected error")

	// Verify file was created
	if utils.FileNotExists(settingsFile) {
		t.Fatal("Settings file was not created")
	}

	// Verify content
	data, err := os.ReadFile(settingsFile)
	require.NoError(t, err, "Failed to read settings file")

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err, "Invalid JSON in settings file")

	assert.Equal(t, "${env:GOROOT}", result["go.goroot"], "Expected go.goroot to be ${env:GOROOT}")
}

func TestWriteVSCodeSettings_MergeExisting(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	// Create existing settings
	existing := map[string]interface{}{
		"editor.fontSize": 14,
		"go.goroot":       "/old/path",
	}
	existingData, _ := json.MarshalIndent(existing, "", "  ")
	testutil.WriteTestFile(t, settingsFile, existingData, utils.PermFileSecure)

	// Add new settings
	newSettings := map[string]interface{}{
		"go.goroot": "${env:GOROOT}",
		"go.gopath": "${env:GOPATH}",
	}

	err = writeVSCodeSettings(settingsFile, newSettings)
	require.NoError(t, err, "writeVSCodeSettings() unexpected error")

	// Verify merged content
	data, err := os.ReadFile(settingsFile)
	require.NoError(t, err, "Failed to read settings file")

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err, "Invalid JSON in settings file")

	// Should have both old and new settings
	assert.Equal(t, float64(14), result["editor.fontSize"], "Expected editor.fontSize to be preserved")
	assert.Equal(t, "${env:GOROOT}", result["go.goroot"], "Expected go.goroot to be updated")
	assert.Equal(t, "${env:GOPATH}", result["go.gopath"], "Expected go.gopath to be added")
}

func TestWriteVSCodeSettings_InvalidJSON(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	// Create invalid JSON
	testutil.WriteTestFile(t, settingsFile, []byte("{invalid json"), utils.PermFileSecure)

	// Capture stderr to check for warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	newSettings := map[string]interface{}{
		"go.goroot": "${env:GOROOT}",
	}

	err = writeVSCodeSettings(settingsFile, newSettings)

	w.Close()
	os.Stderr = oldStderr

	require.NoError(t, err, "writeVSCodeSettings() unexpected error")

	// Should still create valid JSON even with invalid input
	data, err := os.ReadFile(settingsFile)
	require.NoError(t, err, "Failed to read settings file")

	var result map[string]interface{}
	err = json.Unmarshal(data, &result)
	require.NoError(t, err, "Result should be valid JSON")

	assert.Equal(t, "${env:GOROOT}", result["go.goroot"], "Expected go.goroot to be set")

	// Check warning was printed
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	assert.False(t, n > 0 && !strings.Contains(string(output[:n]), "Warning"), "Expected warning about invalid JSON")
}

func TestSetupShellProfile_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.EnvVarHome, tmpDir)
	t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME
	t.Setenv(utils.EnvVarShell, "/bin/bash")

	// Create a .bashrc WITHOUT goenv init (so it will be added)
	bashrc := filepath.Join(tmpDir, ".bashrc")
	testutil.WriteTestFile(t, bashrc, []byte("# Existing content\n"), utils.PermFileDefault)

	setupFlags.shell = "bash"
	setupFlags.yes = true
	defer func() {
		setupFlags.shell = ""
		setupFlags.yes = false
	}()

	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	// Create config
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Call setupShellProfile directly
	_, err := setupShellProfile(setupCmd, cfg)
	require.NoError(t, err, "setupShellProfile() unexpected error")

	// Check file has been modified (new content added)
	content, err := os.ReadFile(bashrc)
	require.NoError(t, err, "Failed to read")

	// Should contain goenv init
	assert.Contains(t, string(content), "goenv init", "Expected .bashrc to contain 'goenv init' %v", string(content))

	// Note: File permissions will be preserved from original utils.PermFileDefault by append mode
	// The security fix applies to NEW files created via os.OpenFile with O_CREATE
}

func TestSetupShellProfile_BackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.EnvVarHome, tmpDir)
	t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME
	t.Setenv(utils.EnvVarShell, "/bin/bash")

	bashrc := filepath.Join(tmpDir, ".bashrc")
	originalContent := "# Original content\nexport PATH=$HOME/bin:$PATH\n"

	// Create existing profile WITHOUT goenv init
	testutil.WriteTestFile(t, bashrc, []byte(originalContent), utils.PermFileSecure)

	setupFlags.shell = "bash"
	setupFlags.yes = true
	defer func() {
		setupFlags.shell = ""
		setupFlags.yes = false
	}()

	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	cfg := &config.Config{
		Root: tmpDir,
	}

	_, err := setupShellProfile(setupCmd, cfg)
	require.NoError(t, err, "setupShellProfile() unexpected error")

	// Check backup was created (with goenv-backup prefix)
	backups, _ := filepath.Glob(filepath.Join(tmpDir, ".bashrc.goenv-backup.*"))
	assert.NotEqual(t, 0, len(backups), "Expected backup file to be created")

	// Verify backup has original content
	if len(backups) > 0 {
		backupContent, err := os.ReadFile(backups[0])
		require.NoError(t, err, "Failed to read backup")
		assert.Equal(t, originalContent, string(backupContent), "Backup content =")
	}
}

func TestSetupCommand_NonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.EnvVarHome, tmpDir)
	t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME
	t.Setenv(utils.EnvVarShell, "/bin/bash")

	setupFlags.nonInteractive = true
	setupFlags.skipVSCode = true
	defer func() {
		setupFlags.nonInteractive = false
		setupFlags.skipVSCode = false
	}()

	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	err := runSetup(setupCmd, []string{})
	require.NoError(t, err, "runSetup() unexpected error")

	output := buf.String()
	// Should not prompt in non-interactive mode
	assert.NotContains(t, output, "[y/N]", "Non-interactive mode should not show prompts")
}

func TestSetupHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	err := setupCmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()
	expectedStrings := []string{
		"setup",
		"shell",
		"--yes",
		"--dry-run",
		"configuration",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
	}
}

func TestSetupCommand_VerifyFlag(t *testing.T) {
	// Skip this test - verify flag runs doctor as subprocess using exec.Command,
	// which doesn't work in test environment (would run test binary with "doctor" arg)
	t.Skip("Verify flag requires subprocess execution - tested in integration tests")
}
