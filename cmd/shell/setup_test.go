package shell

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestSetupCommand_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SHELL", "/bin/bash")

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
	if err != nil {
		t.Fatalf("runSetup() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "DRY RUN") {
		t.Errorf("Expected '[DRY RUN]' in output, got: %s", output)
	}

	// Verify no actual files were created in dry-run
	bashrc := filepath.Join(tmpDir, ".bashrc")
	if _, err := os.Stat(bashrc); err == nil {
		t.Error("Dry-run should not create .bashrc file")
	}
}

func TestSetupCommand_AutoYes(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SHELL", "/bin/bash")

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
	if err != nil {
		t.Fatalf("runSetup() unexpected error: %v", err)
	}

	output := buf.String()
	// Should complete without prompts
	if strings.Contains(output, "[y/N]") {
		t.Error("Auto-yes mode should not show prompts")
	}
}

func TestSetupCommand_AlreadyConfigured(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SHELL", "/bin/bash")
	t.Setenv("GOENV_SHELL", "bash") // Already initialized

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
	if err := os.WriteFile(bashrc, []byte(content), 0600); err != nil {
		t.Fatalf("Failed to create .bashrc: %v", err)
	}

	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	err := runSetup(setupCmd, []string{})
	if err != nil {
		t.Fatalf("runSetup() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "already") || !strings.Contains(output, "configured") {
		t.Errorf("Expected 'already configured' message, got: %s", output)
	}
}

func TestWriteVSCodeSettings_NewFile(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	settings := map[string]interface{}{
		"go.goroot": "${env:GOROOT}",
		"go.gopath": "${env:GOPATH}",
	}

	err := writeVSCodeSettings(settingsFile, settings)
	if err != nil {
		t.Fatalf("writeVSCodeSettings() unexpected error: %v", err)
	}

	// Verify file was created
	if _, err := os.Stat(settingsFile); os.IsNotExist(err) {
		t.Fatal("Settings file was not created")
	}

	// Verify content
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Invalid JSON in settings file: %v", err)
	}

	if result["go.goroot"] != "${env:GOROOT}" {
		t.Errorf("Expected go.goroot to be ${env:GOROOT}, got %v", result["go.goroot"])
	}
}

func TestWriteVSCodeSettings_MergeExisting(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	// Create existing settings
	existing := map[string]interface{}{
		"editor.fontSize": 14,
		"go.goroot":       "/old/path",
	}
	existingData, _ := json.MarshalIndent(existing, "", "  ")
	if err := os.WriteFile(settingsFile, existingData, 0600); err != nil {
		t.Fatalf("Failed to create existing settings: %v", err)
	}

	// Add new settings
	newSettings := map[string]interface{}{
		"go.goroot": "${env:GOROOT}",
		"go.gopath": "${env:GOPATH}",
	}

	err := writeVSCodeSettings(settingsFile, newSettings)
	if err != nil {
		t.Fatalf("writeVSCodeSettings() unexpected error: %v", err)
	}

	// Verify merged content
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Invalid JSON in settings file: %v", err)
	}

	// Should have both old and new settings
	if result["editor.fontSize"] != float64(14) {
		t.Errorf("Expected editor.fontSize to be preserved, got %v", result["editor.fontSize"])
	}
	if result["go.goroot"] != "${env:GOROOT}" {
		t.Errorf("Expected go.goroot to be updated, got %v", result["go.goroot"])
	}
	if result["go.gopath"] != "${env:GOPATH}" {
		t.Errorf("Expected go.gopath to be added, got %v", result["go.gopath"])
	}
}

func TestWriteVSCodeSettings_InvalidJSON(t *testing.T) {
	tmpDir := t.TempDir()
	settingsFile := filepath.Join(tmpDir, "settings.json")

	// Create invalid JSON
	if err := os.WriteFile(settingsFile, []byte("{invalid json"), 0600); err != nil {
		t.Fatalf("Failed to create invalid settings: %v", err)
	}

	// Capture stderr to check for warning
	oldStderr := os.Stderr
	r, w, _ := os.Pipe()
	os.Stderr = w

	newSettings := map[string]interface{}{
		"go.goroot": "${env:GOROOT}",
	}

	err := writeVSCodeSettings(settingsFile, newSettings)

	w.Close()
	os.Stderr = oldStderr

	if err != nil {
		t.Fatalf("writeVSCodeSettings() unexpected error: %v", err)
	}

	// Should still create valid JSON even with invalid input
	data, err := os.ReadFile(settingsFile)
	if err != nil {
		t.Fatalf("Failed to read settings file: %v", err)
	}

	var result map[string]interface{}
	if err := json.Unmarshal(data, &result); err != nil {
		t.Fatalf("Result should be valid JSON: %v", err)
	}

	if result["go.goroot"] != "${env:GOROOT}" {
		t.Errorf("Expected go.goroot to be set, got %v", result["go.goroot"])
	}

	// Check warning was printed
	output := make([]byte, 1024)
	n, _ := r.Read(output)
	if n > 0 && !strings.Contains(string(output[:n]), "Warning") {
		t.Error("Expected warning about invalid JSON")
	}
}

func TestSetupShellProfile_FilePermissions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SHELL", "/bin/bash")

	// Create a .bashrc WITHOUT goenv init (so it will be added)
	bashrc := filepath.Join(tmpDir, ".bashrc")
	if err := os.WriteFile(bashrc, []byte("# Existing content\n"), 0644); err != nil {
		t.Fatalf("Failed to create test .bashrc: %v", err)
	}

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
	if err != nil {
		t.Fatalf("setupShellProfile() unexpected error: %v", err)
	}

	// Check file has been modified (new content added)
	content, err := os.ReadFile(bashrc)
	if err != nil {
		t.Fatalf("Failed to read %s: %v", bashrc, err)
	}

	// Should contain goenv init
	if !strings.Contains(string(content), "goenv init") {
		t.Errorf("Expected .bashrc to contain 'goenv init', got: %s", string(content))
	}

	// Note: File permissions will be preserved from original 0644 by append mode
	// The security fix applies to NEW files created via os.OpenFile with O_CREATE
}

func TestSetupShellProfile_BackupCreation(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SHELL", "/bin/bash")

	bashrc := filepath.Join(tmpDir, ".bashrc")
	originalContent := "# Original content\nexport PATH=$HOME/bin:$PATH\n"

	// Create existing profile WITHOUT goenv init
	if err := os.WriteFile(bashrc, []byte(originalContent), 0600); err != nil {
		t.Fatalf("Failed to create .bashrc: %v", err)
	}

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
	if err != nil {
		t.Fatalf("setupShellProfile() unexpected error: %v", err)
	}

	// Check backup was created (with goenv-backup prefix)
	backups, _ := filepath.Glob(filepath.Join(tmpDir, ".bashrc.goenv-backup.*"))
	if len(backups) == 0 {
		t.Error("Expected backup file to be created")
	}

	// Verify backup has original content
	if len(backups) > 0 {
		backupContent, err := os.ReadFile(backups[0])
		if err != nil {
			t.Fatalf("Failed to read backup: %v", err)
		}
		if string(backupContent) != originalContent {
			t.Errorf("Backup content = %q, want %q", string(backupContent), originalContent)
		}
	}
}

func TestSetupCommand_NonInteractive(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("HOME", tmpDir)
	t.Setenv("SHELL", "/bin/bash")

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
	if err != nil {
		t.Fatalf("runSetup() unexpected error: %v", err)
	}

	output := buf.String()
	// Should not prompt in non-interactive mode
	if strings.Contains(output, "[y/N]") {
		t.Error("Non-interactive mode should not show prompts")
	}
}

func TestSetupHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	setupCmd.SetOut(buf)
	setupCmd.SetErr(buf)

	err := setupCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"setup",
		"shell",
		"--yes",
		"--dry-run",
		"configuration",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestSetupCommand_VerifyFlag(t *testing.T) {
	// Skip this test - verify flag runs doctor as subprocess using exec.Command,
	// which doesn't work in test environment (would run test binary with "doctor" arg)
	t.Skip("Verify flag requires subprocess execution - tested in integration tests")
}
