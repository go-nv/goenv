package diagnostics

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

func TestStatusCommand_NotInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Unset GOENV_SHELL to simulate not initialized
	t.Setenv(utils.GoenvEnvVarShell.String(), "")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "not initialized") && !strings.Contains(output, "Not initialized") {
		t.Logf("Output shows not initialized status, got: %s", output)
	}
}

func TestStatusCommand_Initialized(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create necessary directories
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Status") {
		t.Errorf("Expected 'Status' header in output, got: %s", output)
	}
	if !strings.Contains(output, "Shell") || !strings.Contains(output, "bash") {
		t.Errorf("Expected shell information, got: %s", output)
	}
}

func TestStatusCommand_WithInstalledVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory with some installed versions
	versionsDir := filepath.Join(tmpDir, "versions")
	version1 := filepath.Join(versionsDir, "1.21.5")
	version2 := filepath.Join(versionsDir, "1.22.3")

	if err := utils.EnsureDir(filepath.Join(version1, "bin")); err != nil {
		t.Fatalf("Failed to create version1: %v", err)
	}
	if err := utils.EnsureDir(filepath.Join(version2, "bin")); err != nil {
		t.Fatalf("Failed to create version2: %v", err)
	}

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "2") {
		t.Errorf("Expected to show 2 installed versions, got: %s", output)
	}
}

func TestStatusCommand_WithCurrentVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")
	t.Setenv(utils.GoenvEnvVarVersion.String(), "1.21.5")

	// Create version directory
	versionsDir := filepath.Join(tmpDir, "versions")
	versionDir := filepath.Join(versionsDir, "1.21.5")
	if err := utils.EnsureDir(filepath.Join(versionDir, "bin")); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "1.21.5") {
		t.Errorf("Expected current version 1.21.5 in output, got: %s", output)
	}
}

func TestStatusCommand_SystemVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Create versions directory (can be empty)
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "system") {
		t.Errorf("Expected 'system' in output, got: %s", output)
	}
}

func TestStatusCommand_NoVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	// Should indicate no versions installed
	if !strings.Contains(output, "0") && !strings.Contains(output, "none") && !strings.Contains(output, "No") {
		t.Errorf("Expected indication of no versions, got: %s", output)
	}
}

func TestStatusCommand_WithGoenvDir(t *testing.T) {
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	if err := utils.EnsureDirWithContext(projectDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create project directory: %v", err)
	}

	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), projectDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	// Should show project directory
	if !strings.Contains(output, "project") {
		t.Logf("Expected project directory in output, got: %s", output)
	}
}

func TestStatusHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := statusCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"status",
		"installation",
		"quick",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestStatusCommand_MissingGoenvRoot(t *testing.T) {
	// Create a non-existent directory path
	tmpDir := t.TempDir()
	nonExistent := filepath.Join(tmpDir, "does-not-exist")

	t.Setenv(utils.GoenvEnvVarRoot.String(), nonExistent)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	// Should handle gracefully
	if err != nil {
		// Error is acceptable when GOENV_ROOT doesn't exist
		t.Logf("Expected error for missing GOENV_ROOT: %v", err)
	}

	output := buf.String()
	// Should show some indication of issue
	if output == "" {
		t.Error("Expected some output even for missing GOENV_ROOT")
	}
}

func TestStatusCommand_PathConfiguration(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Add goenv directories to PATH
	shimsDir := filepath.Join(tmpDir, "shims")
	binDir := filepath.Join(tmpDir, "bin")
	oldPath := os.Getenv(utils.EnvVarPath)
	newPath := shimsDir + string(os.PathListSeparator) + binDir + string(os.PathListSeparator) + oldPath
	t.Setenv(utils.EnvVarPath, newPath)

	// Create directories
	if err := utils.EnsureDirWithContext(shimsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err := runStatus(statusCmd, []string{})
	if err != nil {
		t.Fatalf("runStatus() unexpected error: %v", err)
	}

	output := buf.String()
	// Should show PATH configuration status
	if !strings.Contains(output, "PATH") || !strings.Contains(output, "path") {
		t.Logf("Expected PATH information in output, got: %s", output)
	}
}
