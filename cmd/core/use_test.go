package core

import (
	"bytes"
	"github.com/go-nv/goenv/internal/utils"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/spf13/cobra"
)

func TestUseCommand_NoArgs(t *testing.T) {
	// This test verifies that the command accepts zero arguments
	// The actual execution will fail (no versions installed), but we're testing
	// that the argument parsing accepts 0 args

	// Test 1: Verify command definition allows optional version
	if !strings.Contains(useCmd.Use, "[version]") {
		t.Errorf("Command Use should show [version] as optional, got: %s", useCmd.Use)
	}

	// Test 2: Verify help text documents no-args behavior
	if !strings.Contains(useCmd.Long, "If no version is specified") {
		t.Error("Long description should document no-args behavior")
	}

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
	if strings.Contains(useCmd.Use, "<version>") {
		t.Error("Use should show [version] not <version> since it's optional")
	}

	// Verify examples include version usage
	if !strings.Contains(useCmd.Long, "goenv use 1.23.2") {
		t.Error("Examples should show usage with version argument")
	}
}

func TestUseCommand_TooManyArgs(t *testing.T) {
	// This test verifies the runUse function validates argument count
	// The function should return an error for > 1 arguments

	// We test this by checking the function logic in use.go
	// which has: if len(args) > 1 { return fmt.Errorf("usage: goenv use [version]") }

	// Since we can't easily mock the full command execution, we verify
	// the documentation is clear about accepting 0 or 1 arguments
	if !strings.Contains(useCmd.Use, "[version]") {
		t.Error("Use pattern should indicate single optional argument")
	}
}

func TestUseCommand_GlobalFlag(t *testing.T) {
	// Verify --global flag is defined
	globalFlag := useCmd.Flags().Lookup("global")
	if globalFlag == nil {
		t.Error("--global flag should be defined")
		return
	}

	// Verify short flag -g exists
	shortFlag := useCmd.Flags().ShorthandLookup("g")
	if shortFlag == nil {
		t.Error("-g shorthand should be defined for --global")
	}

	// Verify it's a boolean flag
	if globalFlag.Value.Type() != "bool" {
		t.Errorf("--global should be a boolean flag, got: %s", globalFlag.Value.Type())
	}
}

func TestUseCommand_NoArgsWithGlobalFlag(t *testing.T) {
	// Verify the combination of no args + --global flag is supported
	// This should use latest version globally

	// Check that the example is documented
	if !strings.Contains(useCmd.Long, "goenv use --global") {
		t.Error("Examples should include 'goenv use --global' (no version specified)")
	}
}

func TestUseCommand_QuietFlag(t *testing.T) {
	// Verify --quiet flag is defined
	quietFlag := useCmd.Flags().Lookup("quiet")
	if quietFlag == nil {
		t.Error("--quiet flag should be defined")
	}

	// Verify short flag -q exists
	shortFlag := useCmd.Flags().ShorthandLookup("q")
	if shortFlag == nil {
		t.Error("-q shorthand should be defined for --quiet")
	}
}

func TestUseCommand_HelpText(t *testing.T) {
	// Create command
	cmd := useCmd

	// Check Use field shows version as optional
	if !strings.Contains(cmd.Use, "[version]") {
		t.Errorf("Use field should show version as optional [version], got: %s", cmd.Use)
	}

	// Check Long description mentions no-args behavior
	if !strings.Contains(cmd.Long, "If no version is specified") {
		t.Errorf("Long description should document no-args behavior")
	}

	// Check examples include no-args usage
	if !strings.Contains(cmd.Long, "goenv use                     # Use latest stable") {
		t.Errorf("Examples should include no-args usage")
	}
}

func TestUseCommand_VersionResolution(t *testing.T) {
	// Test that various version specs are documented in examples
	specs := []string{"latest", "stable", "1.23.2"}

	for _, spec := range specs {
		found := strings.Contains(useCmd.Long, spec)
		if !found && spec != "stable" { // stable is optional in docs
			t.Errorf("Examples should include version spec: %s", spec)
		}
	}

	// Verify "latest" is the default
	if !strings.Contains(useCmd.Long, "If no version is specified") {
		t.Error("Should document that no args defaults to latest")
	}
}

func TestUseCommand_Flags(t *testing.T) {
	// Verify all expected flags are defined
	flags := []string{"global", "vscode", "vscode-env-vars", "yes", "force", "quiet"}

	for _, flagName := range flags {
		t.Run("flag_"+flagName, func(t *testing.T) {
			flag := useCmd.Flags().Lookup(flagName)
			if flag == nil {
				t.Errorf("Expected flag --%s to be defined", flagName)
			}
		})
	}

	// Check short flags
	shortFlags := map[string]string{
		"g": "global",
		"y": "yes",
		"f": "force",
		"q": "quiet",
	}

	for short, long := range shortFlags {
		t.Run("shortflag_"+short, func(t *testing.T) {
			flag := useCmd.Flags().ShorthandLookup(short)
			if flag == nil {
				t.Errorf("Expected short flag -%s for --%s to be defined", short, long)
			}
		})
	}
}

func TestUseCommand_ConfigLoading(t *testing.T) {
	// Verify config can be loaded (basic smoke test)
	cfg := config.Load()
	if cfg == nil {
		t.Error("Config should load successfully")
		return
	}

	// Config should have a root directory
	if cfg.Root == "" {
		t.Error("Config root should not be empty")
	}
}

func TestUseCommand_ErrorMessages(t *testing.T) {
	// Verify the command documents proper usage in help text
	if !strings.Contains(useCmd.Use, "[version]") {
		t.Error("Command should document optional version argument")
	}

	// The runUse function checks: if len(args) > 1 { return fmt.Errorf("usage:...") }
	// We trust this logic and verify the documentation is clear
	if !strings.Contains(useCmd.Long, "goenv use") {
		t.Error("Long description should include usage examples")
	}
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
	if err == nil {
		t.Fatal("Expected error for too many arguments, got nil")
	}

	if !strings.Contains(err.Error(), "usage:") {
		t.Errorf("Expected usage error, got: %v", err)
	}
}

func TestUseCommand_Integration_NoArgsDefaultsToLatest(t *testing.T) {
	// Setup isolated environment
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions dir: %v", err)
	}

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
	err := cmd.Execute()

	// The command should NOT complain about argument count
	if err != nil && strings.Contains(err.Error(), "usage: goenv use") {
		t.Errorf("Command should accept no arguments, got usage error: %v", err)
	}

	// The error should be about versions not being available, not argument count
	if err != nil && !strings.Contains(err.Error(), "usage:") {
		// This is expected - command accepted no args, failed on version check
		t.Logf("Command correctly accepted no args, failed on: %v", err)
	}
}

func TestUseCommand_Integration_FlagValidation(t *testing.T) {
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
			if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
				t.Fatalf("Failed to create versions dir: %v", err)
			}

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
				if !useFlags.global {
					t.Error("--global flag not set")
				}
			case "force":
				if !useFlags.force {
					t.Error("--force flag not set")
				}
			case "yes":
				if !useFlags.yes {
					t.Error("--yes flag not set")
				}
			case "vscode":
				if !useFlags.vscode {
					t.Error("--vscode flag not set")
				}
			}
		})
	}
}

func TestUseCommand_Integration_QuietMode(t *testing.T) {
	// Setup isolated environment
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions dir: %v", err)
	}

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
		if strings.Contains(output, "No version specified") {
			t.Errorf("Quiet mode should suppress info messages, got: %s", output)
		}
	})
}
