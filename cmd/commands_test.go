package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCommandsCommand(t *testing.T) {
	tests := []struct {
		name             string
		flags            []string
		setupVersions    []string
		setupCommands    []string
		expectedContains []string
		expectedExcludes []string
		expectedOutput   string
	}{
		{
			name:           "completion support",
			flags:          []string{"--complete"},
			expectedOutput: "--sh\n--no-sh\n",
		},
		{
			name:          "returns all commands including versions",
			setupVersions: []string{"1.10.1", "1.9.2"},
			expectedContains: []string{
				"1.10.1",
				"1.9.2",
				"commands",
				"exec",
				"global",
				"local",
				"version",
			},
		},
		{
			name:          "filters commands with --sh flag",
			setupVersions: []string{"1.10.1"},
			flags:         []string{"--sh"},
			expectedContains: []string{
				"rehash",
			},
			expectedExcludes: []string{
				"commands",
				"exec",
				"global",
				"local",
				"version",
				"1.10.1",
			},
		},
		{
			name:          "filters commands with --no-sh flag",
			setupVersions: []string{"1.10.1"},
			flags:         []string{"--no-sh"},
			expectedContains: []string{
				"1.10.1",
				"commands",
				"exec",
				"global",
				"local",
				"version",
			},
			expectedExcludes: []string{
				"shell",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goenvRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup versions
			for _, version := range tt.setupVersions {
				createTestVersion(t, goenvRoot, version)
			}

			// Setup custom commands if specified
			if len(tt.setupCommands) > 0 {
				libexecDir := filepath.Join(goenvRoot, "libexec")
				os.MkdirAll(libexecDir, 0755)
				for _, cmdName := range tt.setupCommands {
					cmdPath := filepath.Join(libexecDir, "goenv-"+cmdName)
					os.WriteFile(cmdPath, []byte("#!/bin/sh\n"), 0755)
				}
			}

			// Execute command
			cmd := &cobra.Command{
				Use: "commands",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runCommands(cmd, args)
				},
				Args:         cobra.NoArgs,
				SilenceUsage: true,
			}

			// Add flags
			cmd.Flags().BoolVar(&commandsSh, "sh", false, "")
			cmd.Flags().BoolVar(&commandsNoSh, "no-sh", false, "")
			cmd.Flags().Bool("complete", false, "")

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs(tt.flags)

			// Reset flags before each test
			commandsSh = false
			commandsNoSh = false

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			got := output.String()

			// Check exact output if specified
			if tt.expectedOutput != "" {
				if got != tt.expectedOutput {
					t.Errorf("Expected output %q, got %q", tt.expectedOutput, got)
				}
				return
			}

			// Check contains
			for _, expected := range tt.expectedContains {
				if !strings.Contains(got, expected) {
					t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", expected, got)
				}
			}

			// Check excludes
			for _, excluded := range tt.expectedExcludes {
				if strings.Contains(got, excluded) {
					t.Errorf("Expected output to NOT contain %q, but it did. Output:\n%s", excluded, got)
				}
			}
		})
	}
}
