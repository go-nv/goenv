package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestHooksCommand(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		flags            []string
		setupHooks       map[string][]string // command -> list of hook files
		hookPaths        []string            // directories to include in GOENV_HOOK_PATH
		expectedOutput   string
		expectedContains []string
	}{
		{
			name:           "completion support",
			args:           []string{"exec"},
			flags:          []string{"--complete"},
			expectedOutput: "exec\nrehash\nversion-name\nversion-origin\nwhich\n",
		},
		{
			name: "prints list of hooks for given command",
			args: []string{"exec"},
			setupHooks: map[string][]string{
				"exec":  {"hello.bash", "ahoy.bash", "invalid.sh"},
				"which": {"ignored.bash"},
			},
			hookPaths:        []string{"goenv.d"},
			expectedContains: []string{"ahoy.bash", "hello.bash"},
		},
		{
			name: "searches multiple hook paths",
			args: []string{"exec"},
			setupHooks: map[string][]string{
				"exec": {"hello.bash", "ahoy.bash", "bueno.bash"},
			},
			hookPaths:        []string{"goenv.d", "etc/goenv_hooks"},
			expectedContains: []string{"ahoy.bash", "hello.bash", "bueno.bash"},
		},
		{
			name: "handles paths with spaces",
			args: []string{"exec"},
			setupHooks: map[string][]string{
				"exec": {"hello.bash", "ahoy.bash"},
			},
			hookPaths:        []string{"my hooks/goenv.d", "etc/goenv hooks"},
			expectedContains: []string{"hello.bash", "ahoy.bash"},
		},
		{
			name: "only includes bash files",
			args: []string{"exec"},
			setupHooks: map[string][]string{
				"exec": {"valid.bash", "invalid.sh", "nope.txt"},
			},
			hookPaths:        []string{"goenv.d"},
			expectedContains: []string{"valid.bash"},
		},
		{
			name:           "returns nothing when no hooks exist",
			args:           []string{"exec"},
			hookPaths:      []string{"goenv.d"},
			expectedOutput: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goenvRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup hooks
			var hookPathDirs []string
			for _, pathName := range tt.hookPaths {
				hookDir := filepath.Join(goenvRoot, pathName)
				hookPathDirs = append(hookPathDirs, hookDir)

				for command, files := range tt.setupHooks {
					commandDir := filepath.Join(hookDir, command)
					if err := os.MkdirAll(commandDir, 0755); err != nil {
						t.Fatalf("Failed to create command hook dir: %v", err)
					}

					for _, file := range files {
						hookFile := filepath.Join(commandDir, file)
						if err := os.WriteFile(hookFile, []byte("#!/bin/bash\n"), 0755); err != nil {
							t.Fatalf("Failed to create hook file: %v", err)
						}
					}
				}
			}

			// Set GOENV_HOOK_PATH
			originalHookPath := os.Getenv("GOENV_HOOK_PATH")
			defer os.Setenv("GOENV_HOOK_PATH", originalHookPath)
			os.Setenv("GOENV_HOOK_PATH", strings.Join(hookPathDirs, ":"))

			// Execute command
			cmd := &cobra.Command{
				Use: "hooks",
				RunE: func(cmd *cobra.Command, cmdArgs []string) error {
					return runHooks(cmd, cmdArgs)
				},
				Args:         cobra.ExactArgs(1),
				SilenceUsage: true,
			}

			cmd.Flags().Bool("complete", false, "Show completion options")

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs(append(tt.args, tt.flags...))

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
		})
	}
}
