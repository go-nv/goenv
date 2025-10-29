package shell

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestCompletionsCommand(t *testing.T) {
	if runtime.GOOS == "windows" {
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
			goenvRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Create command if specified
			if tt.createCommand {
				binDir := filepath.Join(goenvRoot, "bin")
				if err := os.MkdirAll(binDir, 0755); err != nil {
					t.Fatalf("Failed to create bin directory: %v", err)
				}

				cmdName := tt.args[0]
				cmdPath := filepath.Join(binDir, "goenv-"+cmdName)
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

			err := cmd.Execute()
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			got := output.String()

			// Check exact output if specified
			if tt.expectedEquals != "" {
				if got != tt.expectedEquals {
					t.Errorf("Expected output %q, got %q", tt.expectedEquals, got)
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
