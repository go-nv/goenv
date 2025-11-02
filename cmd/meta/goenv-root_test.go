package meta

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			originalRoot := utils.GoenvEnvVarRoot.UnsafeValue()
			defer utils.GoenvEnvVarRoot.Set(originalRoot)
			utils.GoenvEnvVarRoot.Set(tt.goenvRoot)

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
			require.NoError(t, err)

			// Check output (normalize paths for cross-platform comparison)
			got := filepath.ToSlash(strings.TrimSpace(output.String()))
			expected := filepath.ToSlash(tt.expectedOutput)
			assert.Equal(t, expected, got, "Expected output")
		})
	}
}
