package meta

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
			goenvRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Setup versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, goenvRoot, version)
			}

			// Setup custom commands if specified
			if len(tt.setupCommands) > 0 {
				libexecDir := filepath.Join(goenvRoot, "libexec")
				_ = utils.EnsureDirWithContext(libexecDir, "create test directory")
				for _, cmdName := range tt.setupCommands {
					cmdPath := filepath.Join(libexecDir, "goenv-"+cmdName)
					var content string
					if utils.IsWindows() {
						cmdPath += ".bat"
						content = "@echo off\n"
					} else {
						content = "#!/bin/sh\n"
					}
					testutil.WriteTestFile(t, cmdPath, []byte(content), utils.PermFileExecutable)
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
			require.NoError(t, err)

			got := output.String()

			// Check exact output if specified
			if tt.expectedOutput != "" {
				assert.Equal(t, tt.expectedOutput, got, "Expected output")
				return
			}

			// Check contains
			for _, expected := range tt.expectedContains {
				assert.Contains(t, got, expected, "Expected output to contain , but it didn't. Output:\\n %v %v", expected, got)
			}

			// Check excludes
			for _, excluded := range tt.expectedExcludes {
				assert.NotContains(t, got, excluded, "Expected output to NOT contain , but it did. Output:\\n %v %v", excluded, got)
			}
		})
	}
}
