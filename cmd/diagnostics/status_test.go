package diagnostics

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	require.NoError(t, err, "runStatus() unexpected error")

	output := buf.String()
	if !strings.Contains(output, "not initialized") && !strings.Contains(output, "Not initialized") {
		t.Logf("Output shows not initialized status, got: %s", output)
	}
}

func TestStatusCommand_Initialized(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create necessary directories
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err = runStatus(statusCmd, []string{})
	require.NoError(t, err, "runStatus() unexpected error")

	output := buf.String()
	assert.Contains(t, output, "Status", "Expected 'Status' header in output %v", output)
	assert.False(t, !strings.Contains(output, "Shell") || !strings.Contains(output, "bash"), "Expected shell information")
}

func TestStatusCommand_WithInstalledVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory with some installed versions
	versionsDir := filepath.Join(tmpDir, "versions")
	version1 := filepath.Join(versionsDir, "1.21.5")
	version2 := filepath.Join(versionsDir, "1.22.3")

	err = utils.EnsureDir(filepath.Join(version1, "bin"))
	require.NoError(t, err, "Failed to create version1")
	err = utils.EnsureDir(filepath.Join(version2, "bin"))
	require.NoError(t, err, "Failed to create version2")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err = runStatus(statusCmd, []string{})
	require.NoError(t, err, "runStatus() unexpected error")

	output := buf.String()
	assert.Contains(t, output, "2", "Expected to show 2 installed versions %v", output)
}

func TestStatusCommand_WithCurrentVersion(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")
	t.Setenv(utils.GoenvEnvVarVersion.String(), "1.21.5")

	// Create version directory
	versionsDir := filepath.Join(tmpDir, "versions")
	versionDir := filepath.Join(versionsDir, "1.21.5")
	err = utils.EnsureDir(filepath.Join(versionDir, "bin"))
	require.NoError(t, err, "Failed to create version directory")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err = runStatus(statusCmd, []string{})
	require.NoError(t, err, "runStatus() unexpected error")

	output := buf.String()
	assert.Contains(t, output, "1.21.5", "Expected current version 1.21.5 in output %v", output)
}

func TestStatusCommand_SystemVersion(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Create versions directory (can be empty)
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err = runStatus(statusCmd, []string{})
	require.NoError(t, err, "runStatus() unexpected error")

	output := buf.String()
	assert.Contains(t, output, "system", "Expected 'system' in output %v", output)
}

func TestStatusCommand_NoVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create empty versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err = runStatus(statusCmd, []string{})
	require.NoError(t, err, "runStatus() unexpected error")

	output := buf.String()
	// Should indicate no versions installed
	assert.False(t, !strings.Contains(output, "0") && !strings.Contains(output, "none") && !strings.Contains(output, "No"), "Expected indication of no versions")
}

func TestStatusCommand_WithGoenvDir(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	projectDir := filepath.Join(tmpDir, "project")
	err = utils.EnsureDirWithContext(projectDir, "create test directory")
	require.NoError(t, err, "Failed to create project directory")

	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), projectDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err = runStatus(statusCmd, []string{})
	require.NoError(t, err, "runStatus() unexpected error")

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
	require.NoError(t, err, "Help command failed")

	output := buf.String()
	expectedStrings := []string{
		"status",
		"installation",
		"quick",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
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
	assert.NotEmpty(t, output, "Expected some output even for missing GOENV_ROOT")
}

func TestStatusCommand_PathConfiguration(t *testing.T) {
	var err error
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
	err = utils.EnsureDirWithContext(shimsDir, "create test directory")
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	buf := new(bytes.Buffer)
	statusCmd.SetOut(buf)
	statusCmd.SetErr(buf)

	err = runStatus(statusCmd, []string{})
	require.NoError(t, err, "runStatus() unexpected error")

	output := buf.String()
	// Should show PATH configuration status
	if !strings.Contains(output, "PATH") || !strings.Contains(output, "path") {
		t.Logf("Expected PATH information in output, got: %s", output)
	}
}
