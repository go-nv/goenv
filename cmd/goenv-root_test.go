package cmd

import (
	"os"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestGoenvRootCommand(t *testing.T) {
	tests := []struct {
		name           string
		goenvRoot      string
		expectedOutput string
	}{
		{
			name:           "returns current GOENV_ROOT",
			goenvRoot:      "/tmp/whatiexpect",
			expectedOutput: "/tmp/whatiexpect",
		},
		{
			name:           "returns GOENV_ROOT from environment",
			goenvRoot:      "/custom/goenv/root",
			expectedOutput: "/custom/goenv/root",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set GOENV_ROOT
			originalRoot := os.Getenv("GOENV_ROOT")
			defer os.Setenv("GOENV_ROOT", originalRoot)
			os.Setenv("GOENV_ROOT", tt.goenvRoot)

			// Execute command
			cmd := &cobra.Command{
				Use: "root",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runGoenvRoot(cmd, args)
				},
				Args:         cobra.NoArgs,
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs([]string{})

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check output
			got := strings.TrimSpace(output.String())
			if got != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, got)
			}
		})
	}
}
