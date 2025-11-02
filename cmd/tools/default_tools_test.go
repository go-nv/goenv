package tools

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/defaulttools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultToolsList_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	buf := new(bytes.Buffer)
	defaultToolsListCmd.SetOut(buf)
	defaultToolsListCmd.SetErr(buf)

	err := defaultToolsListCmd.RunE(defaultToolsListCmd, []string{})
	require.NoError(t, err, "List command failed")

	output := buf.String()
	assert.Contains(t, output, "No default tools configuration found", "Expected 'No default tools configuration found' message %v", output)
	assert.Contains(t, output, "goenv tools default init", "Expected init suggestion %v", output)
}

func TestDefaultToolsList_WithConfig(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	buf := new(bytes.Buffer)
	defaultToolsListCmd.SetOut(buf)
	defaultToolsListCmd.SetErr(buf)

	err = defaultToolsListCmd.RunE(defaultToolsListCmd, []string{})
	require.NoError(t, err, "List command failed")

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
		assert.Contains(t, output, expected, "Expected in output %v %v", expected, output)
	}
}

func TestDefaultToolsInit(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	buf := new(bytes.Buffer)
	defaultToolsInitCmd.SetOut(buf)
	defaultToolsInitCmd.SetErr(buf)

	err := defaultToolsInitCmd.RunE(defaultToolsInitCmd, []string{})
	require.NoError(t, err, "Init command failed")

	output := buf.String()
	assert.Contains(t, output, "Created default tools configuration", "Expected success message %v", output)

	// Verify config file was created
	configPath := defaulttools.ConfigPath(tmpDir)
	if utils.FileNotExists(configPath) {
		t.Errorf("Config file was not created at %s", configPath)
	}

	// Verify config contents
	config, err := defaulttools.LoadConfig(configPath)
	require.NoError(t, err, "Failed to load created config")
	assert.True(t, config.Enabled, "Expected config to be enabled by default")
	assert.NotEqual(t, 0, len(config.Tools), "Expected default tools to be configured")
}

func TestDefaultToolsInit_AlreadyExists(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create config first
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create initial config")

	buf := new(bytes.Buffer)
	defaultToolsInitCmd.SetOut(buf)
	defaultToolsInitCmd.SetErr(buf)

	err = defaultToolsInitCmd.RunE(defaultToolsInitCmd, []string{})
	assert.Error(t, err, "Expected error when config already exists")

	output := buf.String()
	assert.Contains(t, output, "already exists", "Expected 'already exists' message %v", output)
}

func TestDefaultToolsEnable(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create disabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = false
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	buf := new(bytes.Buffer)
	enableCmd.SetOut(buf)
	enableCmd.SetErr(buf)

	err = enableCmd.RunE(enableCmd, []string{})
	require.NoError(t, err, "Enable command failed")

	output := buf.String()
	assert.Contains(t, output, "Default tools enabled", "Expected success message %v", output)

	// Verify config was updated
	updatedConfig, err := defaulttools.LoadConfig(configPath)
	require.NoError(t, err, "Failed to load config")
	assert.True(t, updatedConfig.Enabled, "Expected config to be enabled")
}

func TestDefaultToolsEnable_AlreadyEnabled(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create enabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = true
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	buf := new(bytes.Buffer)
	enableCmd.SetOut(buf)
	enableCmd.SetErr(buf)

	err = enableCmd.RunE(enableCmd, []string{})
	require.NoError(t, err, "Enable command failed")

	output := buf.String()
	assert.Contains(t, output, "already enabled", "Expected 'already enabled' message %v", output)
}

func TestDefaultToolsDisable(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create enabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = true
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	buf := new(bytes.Buffer)
	disableCmd.SetOut(buf)
	disableCmd.SetErr(buf)

	err = disableCmd.RunE(disableCmd, []string{})
	require.NoError(t, err, "Disable command failed")

	output := buf.String()
	assert.Contains(t, output, "Default tools disabled", "Expected success message %v", output)
	assert.Contains(t, output, "Configuration file preserved", "Expected preservation message %v", output)

	// Verify config was updated
	updatedConfig, err := defaulttools.LoadConfig(configPath)
	require.NoError(t, err, "Failed to load config")
	if updatedConfig.Enabled {
		t.Errorf("Expected config to be disabled")
	}
}

func TestDefaultToolsDisable_AlreadyDisabled(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create disabled config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = false
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	buf := new(bytes.Buffer)
	disableCmd.SetOut(buf)
	disableCmd.SetErr(buf)

	err = disableCmd.RunE(disableCmd, []string{})
	require.NoError(t, err, "Disable command failed")

	output := buf.String()
	assert.Contains(t, output, "already disabled", "Expected 'already disabled' message %v", output)
}

func TestDefaultToolsInstall_NoConfig(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	buf := new(bytes.Buffer)
	installDefaultToolsCmd.SetOut(buf)
	installDefaultToolsCmd.SetErr(buf)

	err := installDefaultToolsCmd.RunE(installDefaultToolsCmd, []string{"1.21.0"})
	assert.Error(t, err, "Expected error when config doesn't exist")
}

func TestDefaultToolsInstall_EmptyToolsList(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create config with empty tools list
	configPath := defaulttools.ConfigPath(tmpDir)
	config := &defaulttools.Config{
		Enabled: true,
		Tools:   []defaulttools.Tool{}, // Empty tools list
	}
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	buf := new(bytes.Buffer)
	installDefaultToolsCmd.SetOut(buf)
	installDefaultToolsCmd.SetErr(buf)

	// Should not error, but should show message about no tools configured
	err = installDefaultToolsCmd.RunE(installDefaultToolsCmd, []string{"1.21.0"})
	require.NoError(t, err, "Unexpected error with empty tools list")

	output := buf.String()
	assert.Contains(t, output, "No tools configured", "Expected 'No tools configured' message %v", output)
}

func TestDefaultToolsInstall_DisabledConfig(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Set GOENV_VERSION to explicitly use the test version
	t.Setenv(utils.GoenvEnvVarVersion.String(), "1.21.0")

	// Create disabled config with tools
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	config.Enabled = false
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	// Create a fake Go version directory structure
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create a mock go binary
	goBinary := filepath.Join(binDir, "go")
	var content string
	if utils.IsWindows() {
		goBinary += ".bat"
		content = "@echo off\necho 1.21.0\n"
	} else {
		content = "#!/bin/bash\necho 1.21.0\n"
	}
	testutil.WriteTestFile(t, goBinary, []byte(content), utils.PermFileExecutable)

	buf := new(bytes.Buffer)
	installDefaultToolsCmd.SetOut(buf)
	installDefaultToolsCmd.SetErr(buf)

	// This should show warning but proceed (note: actual install will fail without real Go)
	_ = installDefaultToolsCmd.RunE(installDefaultToolsCmd, []string{"1.21.0"})

	// We expect this to fail because we don't have a real Go installation
	// but we want to verify the warning message appears
	output := buf.String()
	assert.Contains(t, output, "disabled", "Expected disabled warning %v", output)
}

func TestDefaultToolsVerify_NoConfig(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create version directory (otherwise verify will fail for other reasons)
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	buf := new(bytes.Buffer)
	verifyCmd.SetOut(buf)
	verifyCmd.SetErr(buf)

	// Should succeed - LoadConfig returns default config when file doesn't exist
	err = verifyCmd.RunE(verifyCmd, []string{"1.21.0"})
	require.NoError(t, err, "Verify command should succeed with default config")

	output := buf.String()
	// Should show default tools being checked
	assert.Contains(t, output, "Checking default tools", "Expected checking message %v", output)
}

func TestDefaultToolsVerify_EmptyToolsList(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create config with no tools
	configPath := defaulttools.ConfigPath(tmpDir)
	config := &defaulttools.Config{
		Enabled: true,
		Tools:   []defaulttools.Tool{},
	}
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	buf := new(bytes.Buffer)
	verifyCmd.SetOut(buf)
	verifyCmd.SetErr(buf)

	err = verifyCmd.RunE(verifyCmd, []string{"1.21.0"})
	require.NoError(t, err, "Verify command failed")

	output := buf.String()
	assert.Contains(t, output, "No tools configured", "Expected 'No tools configured' message %v", output)
}

func TestDefaultToolsVerify_WithTools(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create config
	configPath := defaulttools.ConfigPath(tmpDir)
	config := defaulttools.DefaultConfig()
	err = defaulttools.SaveConfig(configPath, config)
	require.NoError(t, err, "Failed to create config")

	// Create version directory structure
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create one mock tool binary (gopls)
	toolBinary := filepath.Join(binDir, "gopls")
	var content string
	if utils.IsWindows() {
		toolBinary += ".exe"
		content = "@echo off\necho gopls\n"
	} else {
		content = "#!/bin/bash\necho gopls\n"
	}
	testutil.WriteTestFile(t, toolBinary, []byte(content), utils.PermFileExecutable)

	buf := new(bytes.Buffer)
	verifyCmd.SetOut(buf)
	verifyCmd.SetErr(buf)

	err = verifyCmd.RunE(verifyCmd, []string{"1.21.0"})
	require.NoError(t, err, "Verify command failed")

	output := buf.String()
	expectedStrings := []string{
		"Checking default tools",
		"1.21.0",
		"Summary:",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Expected in output %v %v", expected, output)
	}
}

func TestDefaultToolsHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	defaultToolsCmd.SetOut(buf)
	defaultToolsCmd.SetErr(buf)

	err := defaultToolsCmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()
	expectedStrings := []string{
		"tools", "default",
		"Manages the list of tools",
		"list",
		"init",
		"enable",
		"disable",
		"install",
		"verify",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
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
			require.NoError(t, err, "Help command failed")

			output := buf.String()
			assert.Contains(t, output, tt.cmdName, "Help output should contain %v %v", tt.cmdName, output)
		})
	}
}
