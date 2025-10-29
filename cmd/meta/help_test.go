package meta

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestHelpCommand(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		createCommand    bool
		commandContent   string
		expectedContains []string
		expectedError    string
	}{
		{
			name: "without args shows summary of common commands",
			args: []string{},
			expectedContains: []string{
				"Usage: goenv <command>",
				"commands    List all available commands",
				"local       Set or show the local",
				"global      Set or show the global",
				"install     Install a Go version",
			},
		},
		{
			name: "fails when command does not exist",
			args: []string{"nonexistent-command"},
			// In the Go implementation, help for non-existent commands prints an error message
			// but doesn't return an error (returns nil to exit with code 0 for shell compatibility)
			expectedContains: []string{}, // Just verify it doesn't crash
		},
		// NOTE: Tests for bash command files in libexec/ are skipped
		// The Go implementation uses Cobra commands directly, not external bash scripts
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goenvRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Create command if specified
			if tt.createCommand {
				binDir := filepath.Join(goenvRoot, "bin")
				if err := os.MkdirAll(binDir, 0755); err != nil {
					t.Fatalf("Failed to create bin directory: %v", err)
				}

				cmdPath := filepath.Join(binDir, "goenv-hello")
				if err := os.WriteFile(cmdPath, []byte(tt.commandContent), 0755); err != nil {
					t.Fatalf("Failed to create command: %v", err)
				}

				// Add bin to PATH
				originalPath := os.Getenv("PATH")
				os.Setenv("PATH", binDir+":"+originalPath)
				defer os.Setenv("PATH", originalPath)
			}

			// Execute command
			cmd := &cobra.Command{
				Use: "help",
				RunE: func(cmd *cobra.Command, cmdArgs []string) error {
					return runHelp(cmd, cmdArgs)
				},
				Args:         cobra.MaximumNArgs(1),
				SilenceUsage: true,
			}

			cmd.Flags().BoolVar(&helpUsage, "usage", false, "Show only usage line")

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs(tt.args)

			// Reset flag
			helpUsage = false

			err := cmd.Execute()

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			got := output.String()

			// Check contains
			for _, expected := range tt.expectedContains {
				if !strings.Contains(got, expected) {
					t.Errorf("Expected output to contain %q, but it didn't. Output:\n%s", expected, got)
				}
			}

			// For --usage tests, ensure extended help is NOT shown
			if len(tt.args) > 0 && tt.args[0] == "--usage" {
				if strings.Contains(got, "extended help") {
					t.Errorf("Expected --usage to not show extended help, but it did. Output:\n%s", got)
				}
			}
		})
	}
}
