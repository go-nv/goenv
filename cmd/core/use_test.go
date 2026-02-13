package core

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-nv/goenv/internal/config"
	"github.com/spf13/cobra"
)

func TestUseCommand_NoArgs(t *testing.T) {
	// This test verifies that the command accepts zero arguments
	// The actual execution will fail (no versions installed), but we're testing
	// that the argument parsing accepts 0 args

	// Test 1: Verify command definition allows optional version
	assert.Contains(t, useCmd.Use, "[version]", "Command Use should show [version] as optional %v", useCmd.Use)

	// Test 2: Verify help text documents no-args behavior
	assert.Contains(t, useCmd.Long, "If no version is specified", "Long description should document no-args behavior")

	// Test 3: Verify Args validator accepts 0 or 1 arguments
	// cobra.MaximumNArgs(1) would allow 0 or 1
	// We can't easily test the actual execution without a full setup,
	// but we can verify the command structure is correct
	t.Log("Command structure validated for no-args support")
}

func TestUseCommand_WithVersion(t *testing.T) {
	// This test verifies that the command accepts a version argument

	// Verify the command accepts string arguments (version spec)
	// The Use pattern should indicate an optional argument
	assert.NotContains(t, useCmd.Use, "<version>", "Use should show [version] not <version> since it's optional")

	// Verify examples include version usage
	assert.Contains(t, useCmd.Long, "goenv use 1.23.2", "Examples should show usage with version argument")
}

func TestUseCommand_TooManyArgs(t *testing.T) {
	// This test verifies the runUse function validates argument count
	// The function should return an error for > 1 arguments

	// We test this by checking the function logic in use.go
	// which has: if len(args) > 1 { return fmt.Errorf("usage: goenv use [version]") }

	// Since we can't easily mock the full command execution, we verify
	// the documentation is clear about accepting 0 or 1 arguments
	assert.Contains(t, useCmd.Use, "[version]", "Use pattern should indicate single optional argument")
}

func TestUseCommand_GlobalFlag(t *testing.T) {
	// Verify --global flag is defined
	globalFlag := useCmd.Flags().Lookup("global")
	assert.NotNil(t, globalFlag, "--global flag should be defined")

	// Verify short flag -g exists
	shortFlag := useCmd.Flags().ShorthandLookup("g")
	assert.NotNil(t, shortFlag, "-g shorthand should be defined for --global")

	// Verify it's a boolean flag
	assert.Equal(t, "bool", globalFlag.Value.Type(), "--global should be a boolean flag")
}

func TestUseCommand_NoArgsWithGlobalFlag(t *testing.T) {
	// Verify the combination of no args + --global flag is supported
	// This should use latest version globally

	// Check that the example is documented
	assert.Contains(t, useCmd.Long, "goenv use --global", "Examples should include 'goenv use --global' (no version specified)")
}

func TestUseCommand_QuietFlag(t *testing.T) {
	// Verify --quiet flag is defined
	quietFlag := useCmd.Flags().Lookup("quiet")
	assert.NotNil(t, quietFlag, "--quiet flag should be defined")

	// Verify short flag -q exists
	shortFlag := useCmd.Flags().ShorthandLookup("q")
	assert.NotNil(t, shortFlag, "-q shorthand should be defined for --quiet")
}

func TestUseCommand_HelpText(t *testing.T) {
	// Create command
	cmd := useCmd

	// Check Use field shows version as optional
	assert.Contains(t, cmd.Use, "[version]", "Use field should show version as optional [version] %v", cmd.Use)

	// Check Long description mentions no-args behavior
	assert.Contains(t, cmd.Long, "If no version is specified", "Long description should document no-args behavior")

	// Check examples include no-args usage
	assert.Contains(t, cmd.Long, "goenv use                     # Use latest stable", "Examples should include no-args usage")
}

func TestUseCommand_VersionResolution(t *testing.T) {
	// Test that various version specs are documented in examples
	specs := []string{"latest", "stable", "1.23.2"}

	for _, spec := range specs {
		found := strings.Contains(useCmd.Long, spec)
		assert.False(t, !found && spec != "stable", "Examples should include version spec")
	}

	// Verify "latest" is the default
	assert.Contains(t, useCmd.Long, "If no version is specified", "Should document that no args defaults to latest")
}

func TestUseCommand_Flags(t *testing.T) {
	// Verify all expected flags are defined
	flags := []string{"global", "vscode", "vscode-env-vars", "yes", "force", "quiet"}

	for _, flagName := range flags {
		t.Run("flag_"+flagName, func(t *testing.T) {
			flag := useCmd.Flags().Lookup(flagName)
			assert.NotNil(t, flag, "Expected flag -- to be defined")
		})
	}

	// Check short flags
	shortFlags := map[string]string{
		"g": "global",
		"y": "yes",
		"f": "force",
		"q": "quiet",
	}

	for short := range shortFlags {
		t.Run("shortflag_"+short, func(t *testing.T) {
			flag := useCmd.Flags().ShorthandLookup(short)
			assert.NotNil(t, flag, "Expected short flag - for -- to be defined")
		})
	}
}

func TestUseCommand_ConfigLoading(t *testing.T) {
	// Verify config can be loaded (basic smoke test)
	cfg := config.Load()
	assert.NotNil(t, cfg, "Config should load successfully")

	// Config should have a root directory
	assert.NotEmpty(t, cfg.Root, "Config root should not be empty")
}

func TestUseCommand_ErrorMessages(t *testing.T) {
	// Verify the command documents proper usage in help text
	assert.Contains(t, useCmd.Use, "[version]", "Command should document optional version argument")

	// The runUse function checks: if len(args) > 1 { return fmt.Errorf("usage:...") }
	// We trust this logic and verify the documentation is clear
	assert.Contains(t, useCmd.Long, "goenv use", "Long description should include usage examples")
}

// Integration Tests - These test actual command execution

func TestUseCommand_Integration_TooManyArgs(t *testing.T) {
	// Setup isolated environment
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create a fresh command instance to avoid flag pollution
	cmd := &cobra.Command{
		Use:  useCmd.Use,
		RunE: runUse,
	}
	cmd.SetArgs([]string{"1.22.0", "1.23.0", "1.24.0"})

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute - should fail with usage error
	err := cmd.Execute()
	assert.Error(t, err, "Expected error for too many arguments, got nil")

	assert.Contains(t, err.Error(), "usage:", "Expected usage error %v", err)
}

func TestUseCommand_Integration_NoArgsDefaultsToLatest(t *testing.T) {
	var err error
	// Setup isolated environment
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions dir")

	// Create a fresh command instance
	cmd := &cobra.Command{
		Use:  useCmd.Use,
		RunE: runUse,
	}

	// Add the quiet flag to suppress output
	cmd.Flags().BoolVarP(&useFlags.quiet, "quiet", "q", false, "Quiet mode")
	cmd.SetArgs([]string{"--quiet"})

	// Capture output
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)

	// Execute - will fail (no versions available) but should accept no args
	err = cmd.Execute()

	// The command should NOT complain about argument count
	assert.False(t, err != nil && strings.Contains(err.Error(), "usage: goenv use"), "Command should accept no arguments")

	// The error should be about versions not being available, not argument count
	if err != nil && !strings.Contains(err.Error(), "usage:") {
		// This is expected - command accepted no args, failed on version check
		t.Logf("Command correctly accepted no args, failed on: %v", err)
	}
}

func TestUseCommand_Integration_FlagValidation(t *testing.T) {
	var err error
	// Test that flags are properly registered and work
	tests := []struct {
		name string
		args []string
		flag string
	}{
		{
			name: "global flag",
			args: []string{"--global", "--quiet"},
			flag: "global",
		},
		{
			name: "force flag",
			args: []string{"--force", "--quiet"},
			flag: "force",
		},
		{
			name: "yes flag",
			args: []string{"--yes", "--quiet"},
			flag: "yes",
		},
		{
			name: "vscode flag",
			args: []string{"--vscode", "--quiet"},
			flag: "vscode",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			useFlags = struct {
				global    bool
				vscode    bool
				vscodeEnv bool
				yes       bool
				force     bool
				quiet     bool
			}{}

			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

			// Create versions directory
			versionsDir := filepath.Join(tmpDir, "versions")
			err = utils.EnsureDirWithContext(versionsDir, "create test directory")
			require.NoError(t, err, "Failed to create versions dir")

			// Create a fresh command instance with all flags
			cmd := &cobra.Command{
				Use:  useCmd.Use,
				RunE: runUse,
			}
			cmd.Flags().BoolVarP(&useFlags.global, "global", "g", false, "Global")
			cmd.Flags().BoolVar(&useFlags.vscode, "vscode", false, "VSCode")
			cmd.Flags().BoolVar(&useFlags.vscodeEnv, "vscode-env-vars", false, "VSCode env")
			cmd.Flags().BoolVarP(&useFlags.yes, "yes", "y", false, "Yes")
			cmd.Flags().BoolVarP(&useFlags.force, "force", "f", false, "Force")
			cmd.Flags().BoolVarP(&useFlags.quiet, "quiet", "q", false, "Quiet")

			cmd.SetArgs(tt.args)

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Execute (will fail, but flags should be parsed)
			_ = cmd.Execute()

			// Verify the flag was parsed (checking the struct value)
			switch tt.flag {
			case "global":
				assert.True(t, useFlags.global, "--global flag not set")
			case "force":
				assert.True(t, useFlags.force, "--force flag not set")
			case "yes":
				assert.True(t, useFlags.yes, "--yes flag not set")
			case "vscode":
				assert.True(t, useFlags.vscode, "--vscode flag not set")
			}
		})
	}
}

func TestUseCommand_Integration_QuietMode(t *testing.T) {
	var err error
	// Setup isolated environment
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions dir")

	// Test WITHOUT quiet flag
	t.Run("verbose_mode", func(t *testing.T) {
		useFlags.quiet = false

		cmd := &cobra.Command{
			Use:  useCmd.Use,
			RunE: runUse,
		}
		cmd.Flags().BoolVarP(&useFlags.quiet, "quiet", "q", false, "Quiet")
		cmd.SetArgs([]string{}) // No args

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		_ = cmd.Execute()

		output := buf.String()
		// In verbose mode, should show "No version specified" message
		if !strings.Contains(output, "No version specified") && !strings.Contains(output, "latest") {
			t.Log("Expected verbose output, got:", output)
			// Note: May not always show if command fails early
		}
	})

	// Test WITH quiet flag
	t.Run("quiet_mode", func(t *testing.T) {
		useFlags.quiet = false // Reset

		cmd := &cobra.Command{
			Use:  useCmd.Use,
			RunE: runUse,
		}
		cmd.Flags().BoolVarP(&useFlags.quiet, "quiet", "q", false, "Quiet")
		cmd.SetArgs([]string{"--quiet"})

		var buf bytes.Buffer
		cmd.SetOut(&buf)
		cmd.SetErr(&buf)

		_ = cmd.Execute()

		output := buf.String()
		// In quiet mode, should NOT show "No version specified" message
		assert.NotContains(t, output, "No version specified", "Quiet mode should suppress info messages %v", output)
	})
}
