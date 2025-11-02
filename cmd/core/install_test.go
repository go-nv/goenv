package core

import (
	"bytes"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
)

func TestInstallCommand_FlagValidation(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		flags         map[string]string
		expectedError string
	}{
		{
			name: "ipv4 and ipv6 together",
			args: []string{"1.21.0"},
			flags: map[string]string{
				"ipv4": "true",
				"ipv6": "true",
			},
			expectedError: "cannot specify both --ipv4 and --ipv6",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Reset flags
			installCmd.ResetFlags()
			installCmd.Flags().BoolVarP(&installFlags.force, "force", "f", false, "")
			installCmd.Flags().BoolVarP(&installFlags.skipExisting, "skip-existing", "s", false, "")
			installCmd.Flags().BoolVarP(&installFlags.list, "list", "l", false, "")
			installCmd.Flags().BoolVarP(&installFlags.keep, "keep", "k", false, "")
			installCmd.Flags().BoolVarP(&installFlags.verbose, "verbose", "v", false, "")
			installCmd.Flags().BoolVarP(&installFlags.quiet, "quiet", "q", false, "")
			installCmd.Flags().BoolVarP(&installFlags.ipv4, "ipv4", "4", false, "")
			installCmd.Flags().BoolVarP(&installFlags.ipv6, "ipv6", "6", false, "")
			installCmd.Flags().BoolVarP(&installFlags.debug, "debug", "g", false, "")
			installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "")

			// Set flags
			for key, value := range tt.flags {
				installCmd.Flags().Set(key, value)
			}

			// Capture output
			buf := new(bytes.Buffer)
			installCmd.SetOut(buf)
			installCmd.SetErr(buf)

			// Execute
			err := runInstall(installCmd, tt.args)

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
			} else if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Reset flags
			installFlags.force = false
			installFlags.skipExisting = false
			installFlags.list = false
			installFlags.keep = false
			installFlags.verbose = false
			installFlags.quiet = false
			installFlags.ipv4 = false
			installFlags.ipv6 = false
			installFlags.debug = false
			installFlags.complete = false
		})
	}
}

func TestInstallHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	cmd := installCmd
	cmd.SetOut(buf)
	cmd.SetErr(buf)

	// Get help text
	err := cmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()

	// Check for key help text elements (custom help text via helptext package)
	expectedStrings := []string{
		"install",
		"Usage:",
		"--force",
		"--skip-existing",
		"--list",
		"--keep",
		"--verbose",
		"--quiet",
		"Keep source tree",
		"Verbose mode",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestInstallCommand_SkipExisting(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create a proper mock installed version with go binary
	cmdtest.CreateMockGoVersion(t, tmpDir, "1.21.0")

	// Reset flags
	installCmd.ResetFlags()
	installCmd.Flags().BoolVarP(&installFlags.skipExisting, "skip-existing", "s", false, "")
	installCmd.Flags().BoolVar(&installFlags.complete, "complete", false, "")
	installCmd.Flags().Set("skip-existing", "true")

	// Capture output
	buf := new(bytes.Buffer)
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	// Execute - should skip silently since version already exists
	err := runInstall(installCmd, []string{"1.21.0"})

	// Should not error when skipping
	if err != nil {
		t.Errorf("Unexpected error with skip-existing: %v", err)
	}

	// Reset flags
	installFlags.skipExisting = false
	installFlags.complete = false
}
