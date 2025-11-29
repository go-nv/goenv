package meta

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetStartedCommand_NotInitialized(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Unset GOENV_SHELL to simulate not initialized
	t.Setenv(utils.GoenvEnvVarShell.String(), "")
	t.Setenv(utils.EnvVarShell, "/bin/bash")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should show initialization instructions
	assert.False(t, !strings.Contains(output, "init") || !strings.Contains(output, "eval"), "Expected initialization instructions")
}

func TestGetStartedCommand_Initialized(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory (empty)
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err = runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should show next steps (like installing Go)
	assert.Contains(t, output, "install", "Expected installation instructions %v", output)
}

func TestGetStartedCommand_WithInstalledVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory with an installed version
	versionsDir := filepath.Join(tmpDir, "versions")
	versionDir := filepath.Join(versionsDir, "1.21.5")
	err = utils.EnsureDir(filepath.Join(versionDir, "bin"))
	require.NoError(t, err, "Failed to create version directory")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err = runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should show that setup is complete or show next steps
	if !strings.Contains(output, "1.21.5") && !strings.Contains(output, "use") {
		t.Logf("Expected version or usage instructions, got: %s", output)
	}
}

func TestGetStartedCommand_BashShell(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "")
	t.Setenv(utils.EnvVarShell, "/bin/bash")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should show bash-specific instructions
	assert.True(t, strings.Contains(output, "bashrc") || strings.Contains(output, "bash_profile"), "Expected bash-specific instructions")
}

func TestGetStartedCommand_ZshShell(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "")
	t.Setenv(utils.EnvVarShell, "/bin/zsh")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should show zsh-specific instructions
	assert.Contains(t, output, "zshrc", "Expected zsh-specific instructions %v", output)
}

func TestGetStartedCommand_FishShell(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "")
	t.Setenv(utils.EnvVarShell, "/usr/bin/fish")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should show fish-specific instructions
	assert.True(t, strings.Contains(output, "fish") || strings.Contains(output, "config.fish"), "Expected fish-specific instructions")
}

func TestGetStartedCommand_UnknownShell(t *testing.T) {
	// On Windows, shell detection always defaults to PowerShell
	// This test is for Unix where unknown shells fall back to bash
	if utils.IsWindows() {
		t.Skip("Skipping unknown shell test on Windows - defaults to PowerShell")
	}

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "")
	t.Setenv(utils.EnvVarShell, "/usr/bin/unknown-shell")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should still provide generic instructions
	assert.False(t, !strings.Contains(output, "eval") || !strings.Contains(output, "init"), "Expected generic init instructions")
}

func TestGetStartedHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := getStartedCmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()
	expectedStrings := []string{
		"get-started",
		"guide",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
	}
}

func TestGetStartedCommand_ShowsHelpfulLinks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should mention other helpful commands
	helpfulCommands := []string{"doctor", "help", "install"}
	foundCommands := 0
	for _, cmd := range helpfulCommands {
		if strings.Contains(output, cmd) {
			foundCommands++
		}
	}

	assert.NotEqual(t, 0, foundCommands, "Expected at least one helpful command mentioned")
}

func TestGetStartedCommand_FormattingAndStructure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should have clear structure (sections with headings)
	assert.Contains(t, output, "\n\n", "Output should have paragraph breaks for readability")

	// Should have some numbered or bulleted steps
	hasSteps := strings.Contains(output, "1.") ||
		strings.Contains(output, "2.") ||
		strings.Contains(output, "-") ||
		strings.Contains(output, "*")

	assert.True(t, hasSteps, "Expected step-by-step instructions")
}

func TestGetStartedCommand_EmptyOutput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	require.NoError(t, err, "runGetStarted() unexpected error")

	output := buf.String()

	// Should never have empty output
	assert.NotEmpty(t, output, "Output should not be empty")

	// Should be reasonably long (helpful guide)
	if len(output) < 100 {
		t.Errorf("Output seems too short (%d chars) for a helpful guide", len(output))
	}
}

func TestGetStartedCommand_AdaptiveContent(t *testing.T) {
	// Test that output adapts based on setup state
	tests := []struct {
		name         string
		setupEnv     func(*testing.T, string)
		expectText   string
		unexpectText string
	}{
		{
			name: "not initialized",
			setupEnv: func(t *testing.T, tmpDir string) {
				t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
				t.Setenv(utils.GoenvEnvVarShell.String(), "")
			},
			expectText:   "init",
			unexpectText: "",
		},
		{
			name: "initialized but no versions",
			setupEnv: func(t *testing.T, tmpDir string) {
				t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
				t.Setenv(utils.GoenvEnvVarShell.String(), "bash")
				utils.EnsureDir(filepath.Join(tmpDir, "versions"))
			},
			expectText:   "install",
			unexpectText: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			tt.setupEnv(t, tmpDir)

			buf := new(bytes.Buffer)
			getStartedCmd.SetOut(buf)
			getStartedCmd.SetErr(buf)

			err := runGetStarted(getStartedCmd, []string{})
			require.NoError(t, err, "runGetStarted() unexpected error")

			output := buf.String()

			assert.False(t, tt.expectText != "" && !strings.Contains(output, tt.expectText), "Expected in output")
		})
	}
}
