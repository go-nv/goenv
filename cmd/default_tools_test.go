package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/defaulttools"
	"github.com/spf13/cobra"
)

func TestDefaultToolsList_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	buf := new(bytes.Buffer)
	defaultToolsListCmd.SetOut(buf)
	defaultToolsListCmd.SetErr(buf)

	err := defaultToolsListCmd.RunE(defaultToolsListCmd, []string{})
	if err != nil {
		t.Fatalf("List command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No default tools configuration found") {
		t.Errorf("Expected 'No default tools configuration found' message, got: %s", output)
	}
	if !strings.Contains(output, "goenv default-tools init") {
		t.Errorf("Expected init suggestion, got: %s", output)
	}
}

func TestDefaultToolsList_WithConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	buf := new(bytes.Buffer)
	defaultToolsListCmd.SetOut(buf)
	defaultToolsListCmd.SetErr(buf)

	err := defaultToolsListCmd.RunE(defaultToolsListCmd, []string{})
	if err != nil {
		t.Fatalf("List command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"Default Tools Configuration",
		"Status:",
		"Config:",
		"Configured Tools",
		"gopls",
		"golangci-lint",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected %q in output, got: %s", expected, output)
		}
	}
}

func TestDefaultToolsInit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	buf := new(bytes.Buffer)
	defaultToolsInitCmd.SetOut(buf)
	defaultToolsInitCmd.SetErr(buf)

	err := defaultToolsInitCmd.RunE(defaultToolsInitCmd, []string{})
	if err != nil {
		t.Fatalf("Init command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Created default tools configuration") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify config file was created
	configPath := defaulttools.ConfigPath(tmpDir)
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		t.Errorf("Config file was not created at %s", configPath)
	}

	// Verify config contents
	config, err := defaulttools.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load created config: %v", err)
	}
	if !config.Enabled {
		t.Errorf("Expected config to be enabled by default")
	}
	if len(config.Tools) == 0 {
		t.Errorf("Expected default tools to be configured")
	}
}

func TestDefaultToolsInit_AlreadyExists(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create config first
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create initial config: %v", err)
	}

	buf := new(bytes.Buffer)
	defaultToolsInitCmd.SetOut(buf)
	defaultToolsInitCmd.SetErr(buf)

	err := defaultToolsInitCmd.RunE(defaultToolsInitCmd, []string{})
	if err == nil {
		t.Fatal("Expected error when config already exists")
	}

	output := buf.String()
	if !strings.Contains(output, "already exists") {
		t.Errorf("Expected 'already exists' message, got: %s", output)
	}
}

func TestDefaultToolsEnable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create disabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = false
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	buf := new(bytes.Buffer)
	enableCmd.SetOut(buf)
	enableCmd.SetErr(buf)

	err := enableCmd.RunE(enableCmd, []string{})
	if err != nil {
		t.Fatalf("Enable command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Default tools enabled") {
		t.Errorf("Expected success message, got: %s", output)
	}

	// Verify config was updated
	updatedConfig, err := defaulttools.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if !updatedConfig.Enabled {
		t.Errorf("Expected config to be enabled")
	}
}

func TestDefaultToolsEnable_AlreadyEnabled(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create enabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = true
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	buf := new(bytes.Buffer)
	enableCmd.SetOut(buf)
	enableCmd.SetErr(buf)

	err := enableCmd.RunE(enableCmd, []string{})
	if err != nil {
		t.Fatalf("Enable command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "already enabled") {
		t.Errorf("Expected 'already enabled' message, got: %s", output)
	}
}

func TestDefaultToolsDisable(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create enabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = true
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	buf := new(bytes.Buffer)
	disableCmd.SetOut(buf)
	disableCmd.SetErr(buf)

	err := disableCmd.RunE(disableCmd, []string{})
	if err != nil {
		t.Fatalf("Disable command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Default tools disabled") {
		t.Errorf("Expected success message, got: %s", output)
	}
	if !strings.Contains(output, "Configuration file preserved") {
		t.Errorf("Expected preservation message, got: %s", output)
	}

	// Verify config was updated
	updatedConfig, err := defaulttools.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}
	if updatedConfig.Enabled {
		t.Errorf("Expected config to be disabled")
	}
}

func TestDefaultToolsDisable_AlreadyDisabled(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create disabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = false
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	buf := new(bytes.Buffer)
	disableCmd.SetOut(buf)
	disableCmd.SetErr(buf)

	err := disableCmd.RunE(disableCmd, []string{})
	if err != nil {
		t.Fatalf("Disable command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "already disabled") {
		t.Errorf("Expected 'already disabled' message, got: %s", output)
	}
}

func TestDefaultToolsInstall_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	buf := new(bytes.Buffer)
	installToolsCmd.SetOut(buf)
	installToolsCmd.SetErr(buf)

	err := installToolsCmd.RunE(installToolsCmd, []string{"1.21.0"})
	if err == nil {
		t.Fatal("Expected error when config doesn't exist")
	}
}

func TestDefaultToolsInstall_EmptyToolsList(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create config with no tools
	configPath := defaulttools.ConfigPath(tmpDir)
	config := &defaulttools.Config{
		Enabled: true,
		Tools:   []defaulttools.Tool{},
	}
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	buf := new(bytes.Buffer)
	installToolsCmd.SetOut(buf)
	installToolsCmd.SetErr(buf)

	err := installToolsCmd.RunE(installToolsCmd, []string{"1.21.0"})
	if err != nil {
		t.Fatalf("Install command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No tools configured") {
		t.Errorf("Expected 'No tools configured' message, got: %s", output)
	}
}

func TestDefaultToolsInstall_DisabledConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create disabled config with tools
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = false
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create a fake Go version directory structure
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create a mock go binary
	goBinary := filepath.Join(binDir, "go")
	var content string
	if runtime.GOOS == "windows" {
		goBinary += ".bat"
		content = "@echo off\necho 1.21.0\n"
	} else {
		content = "#!/bin/bash\necho 1.21.0\n"
	}
	if err := os.WriteFile(goBinary, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock go binary: %v", err)
	}

	buf := new(bytes.Buffer)
	installToolsCmd.SetOut(buf)
	installToolsCmd.SetErr(buf)

	// This should show warning but proceed (note: actual install will fail without real Go)
	_ = installToolsCmd.RunE(installToolsCmd, []string{"1.21.0"})

	// We expect this to fail because we don't have a real Go installation
	// but we want to verify the warning message appears
	output := buf.String()
	if !strings.Contains(output, "disabled") {
		t.Errorf("Expected disabled warning, got: %s", output)
	}
}

func TestDefaultToolsVerify_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create version directory (otherwise verify will fail for other reasons)
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	buf := new(bytes.Buffer)
	verifyCmd.SetOut(buf)
	verifyCmd.SetErr(buf)

	// Should succeed - LoadConfig returns default config when file doesn't exist
	err := verifyCmd.RunE(verifyCmd, []string{"1.21.0"})
	if err != nil {
		t.Fatalf("Verify command should succeed with default config: %v", err)
	}

	output := buf.String()
	// Should show default tools being checked
	if !strings.Contains(output, "Checking default tools") {
		t.Errorf("Expected checking message, got: %s", output)
	}
}

func TestDefaultToolsVerify_EmptyToolsList(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create config with no tools
	configPath := defaulttools.ConfigPath(tmpDir)
	config := &defaulttools.Config{
		Enabled: true,
		Tools:   []defaulttools.Tool{},
	}
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	buf := new(bytes.Buffer)
	verifyCmd.SetOut(buf)
	verifyCmd.SetErr(buf)

	err := verifyCmd.RunE(verifyCmd, []string{"1.21.0"})
	if err != nil {
		t.Fatalf("Verify command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No tools configured") {
		t.Errorf("Expected 'No tools configured' message, got: %s", output)
	}
}

func TestDefaultToolsVerify_WithTools(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	if err := defaulttools.SaveConfig(configPath, config); err != nil {
		t.Fatalf("Failed to create config: %v", err)
	}

	// Create version directory structure
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create one mock tool binary (gopls)
	toolBinary := filepath.Join(binDir, "gopls")
	if err := os.WriteFile(toolBinary, []byte("#!/bin/bash\necho gopls\n"), 0755); err != nil {
		t.Fatalf("Failed to create mock tool binary: %v", err)
	}

	buf := new(bytes.Buffer)
	verifyCmd.SetOut(buf)
	verifyCmd.SetErr(buf)

	err := verifyCmd.RunE(verifyCmd, []string{"1.21.0"})
	if err != nil {
		t.Fatalf("Verify command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"Checking default tools",
		"1.21.0",
		"Summary:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Expected %q in output, got: %s", expected, output)
		}
	}
}

func TestDefaultToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	defaultToolsCmd.SetOut(buf)
	defaultToolsCmd.SetErr(buf)

	err := defaultToolsCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"default-tools",
		"Manages the list of tools",
		"list",
		"init",
		"enable",
		"disable",
		"install",
		"verify",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestDefaultToolsSubcommandHelp(t *testing.T) {
	tests := []struct {
		name    string
		cmd     *cobra.Command
		cmdName string
	}{
		{"list help", defaultToolsListCmd, "list"},
		{"init help", defaultToolsInitCmd, "init"},
		{"enable help", enableCmd, "enable"},
		{"disable help", disableCmd, "disable"},
		{"install help", installToolsCmd, "install"},
		{"verify help", verifyCmd, "verify"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			tt.cmd.SetOut(buf)
			tt.cmd.SetErr(buf)

			err := tt.cmd.Help()
			if err != nil {
				t.Fatalf("Help command failed: %v", err)
			}

			output := buf.String()
			if !strings.Contains(output, tt.cmdName) {
				t.Errorf("Help output should contain %q, got: %s", tt.cmdName, output)
			}
		})
	}
}
