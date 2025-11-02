package shell

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

func TestCompletionsCommand(t *testing.T) {
	var err error
	if utils.IsWindows() {
		t.Skip("Skipping Unix shell completion test on Windows")
	}

	tests := []struct {
		name             string
		args             []string
		createCommand    bool
		commandContent   string
		expectedContains []string
		expectedEquals   string
	}{
		{
			name:           "returns --help for command with no completion support",
			args:           []string{"hello"},
			createCommand:  true,
			commandContent: "#!/bin/bash\necho hello\n",
			expectedEquals: "--help\n",
		},
		{
			name:          "returns --help as first argument for command with completion support",
			args:          []string{"hello"},
			createCommand: true,
			commandContent: `#!/bin/bash
# Provide goenv completions
if [[ $1 = --complete ]]; then
  echo not_important
else
  exit 1
fi
`,
			expectedContains: []string{"--help", "not_important"},
		},
		{
			name:          "returns specified command completions",
			args:          []string{"hello"},
			createCommand: true,
			commandContent: `#!/bin/bash
# Provide goenv completions
if [[ $1 = --complete ]]; then
  echo hello
else
  exit 1
fi
`,
			expectedContains: []string{"--help", "hello"},
		},
		{
			name:          "forwards extra arguments to command",
			args:          []string{"hello", "happy", "world"},
			createCommand: true,
			commandContent: `#!/bin/bash
# provide goenv completions
if [[ $1 = --complete ]]; then
  shift 1
  for arg; do echo $arg; done
else
  exit 1
fi
`,
			expectedContains: []string{"--help", "happy", "world"},
		},
		{
			name:           "returns only --help when command not found",
			args:           []string{"nonexistent"},
			expectedEquals: "--help\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Create command if specified
			if tt.createCommand {
				binDir := filepath.Join(tmpDir, "bin")
				err = utils.EnsureDirWithContext(binDir, "create test directory")
				require.NoError(t, err, "Failed to create bin directory")

				cmdName := tt.args[0]
				cmdPath := filepath.Join(binDir, "goenv-"+cmdName)
				testutil.WriteTestFile(t, cmdPath, []byte(tt.commandContent), utils.PermFileExecutable)

				// Add bin to PATH
				originalPath := os.Getenv(utils.EnvVarPath)
				os.Setenv(utils.EnvVarPath, binDir+":"+originalPath)
				defer os.Setenv(utils.EnvVarPath, originalPath)
			}

			// Execute command
			cmd := &cobra.Command{
				Use: "completions",
				RunE: func(cmd *cobra.Command, cmdArgs []string) error {
					return runCompletions(cmd, cmdArgs)
				},
				Args:         cobra.MinimumNArgs(1),
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs(tt.args)

			err = cmd.Execute()
			require.NoError(t, err)

			got := output.String()

			// Check exact output if specified
			if tt.expectedEquals != "" {
				assert.Equal(t, tt.expectedEquals, got, "Expected output")
				return
			}

			// Check contains
			for _, expected := range tt.expectedContains {
				assert.Contains(t, got, expected, "Expected output to contain , but it didn't. Output:\\n %v %v", expected, got)
			}
		})
	}
}
