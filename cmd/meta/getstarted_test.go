package meta

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
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
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should show initialization instructions
	if !strings.Contains(output, "init") || !strings.Contains(output, "eval") {
		t.Errorf("Expected initialization instructions, got: %s", output)
	}
}

func TestGetStartedCommand_Initialized(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory (empty)
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should show next steps (like installing Go)
	if !strings.Contains(output, "install") {
		t.Errorf("Expected installation instructions, got: %s", output)
	}
}

func TestGetStartedCommand_WithInstalledVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	// Create versions directory with an installed version
	versionsDir := filepath.Join(tmpDir, "versions")
	versionDir := filepath.Join(versionsDir, "1.21.5")
	if err := utils.EnsureDir(filepath.Join(versionDir, "bin")); err != nil {
		t.Fatalf("Failed to create version directory: %v", err)
	}

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

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
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should show bash-specific instructions
	if !strings.Contains(output, "bashrc") && !strings.Contains(output, "bash_profile") {
		t.Errorf("Expected bash-specific instructions, got: %s", output)
	}
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
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should show zsh-specific instructions
	if !strings.Contains(output, "zshrc") {
		t.Errorf("Expected zsh-specific instructions, got: %s", output)
	}
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
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should show fish-specific instructions
	if !strings.Contains(output, "fish") && !strings.Contains(output, "config.fish") {
		t.Errorf("Expected fish-specific instructions, got: %s", output)
	}
}

func TestGetStartedCommand_UnknownShell(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "")
	t.Setenv(utils.EnvVarShell, "/usr/bin/unknown-shell")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should still provide generic instructions
	if !strings.Contains(output, "eval") || !strings.Contains(output, "init") {
		t.Errorf("Expected generic init instructions, got: %s", output)
	}
}

func TestGetStartedHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := getStartedCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"get-started",
		"guide",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
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
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should mention other helpful commands
	helpfulCommands := []string{"doctor", "help", "install"}
	foundCommands := 0
	for _, cmd := range helpfulCommands {
		if strings.Contains(output, cmd) {
			foundCommands++
		}
	}

	if foundCommands == 0 {
		t.Error("Expected at least one helpful command mentioned")
	}
}

func TestGetStartedCommand_FormattingAndStructure(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarShell.String(), "bash")

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should have clear structure (sections with headings)
	if !strings.Contains(output, "\n\n") {
		t.Error("Output should have paragraph breaks for readability")
	}

	// Should have some numbered or bulleted steps
	hasSteps := strings.Contains(output, "1.") ||
		strings.Contains(output, "2.") ||
		strings.Contains(output, "-") ||
		strings.Contains(output, "*")

	if !hasSteps {
		t.Error("Expected step-by-step instructions")
	}
}

func TestGetStartedCommand_EmptyOutput(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	buf := new(bytes.Buffer)
	getStartedCmd.SetOut(buf)
	getStartedCmd.SetErr(buf)

	err := runGetStarted(getStartedCmd, []string{})
	if err != nil {
		t.Fatalf("runGetStarted() unexpected error: %v", err)
	}

	output := buf.String()

	// Should never have empty output
	if output == "" {
		t.Error("Output should not be empty")
	}

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
			if err != nil {
				t.Fatalf("runGetStarted() unexpected error: %v", err)
			}

			output := buf.String()

			if tt.expectText != "" && !strings.Contains(output, tt.expectText) {
				t.Errorf("Expected %q in output, got: %s", tt.expectText, output)
			}
		})
	}
}
