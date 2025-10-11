package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestInitCommand(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		envVars          map[string]string
		setupFunc        func(t *testing.T, tmpDir string)
		expectedOutput   []string
		unexpectedOutput []string
		checkDirs        []string
		shouldFail       bool
	}{
		{
			name: "has completion support",
			args: []string{"--complete"},
			expectedOutput: []string{
				"-",
				"--no-rehash",
				"bash",
				"fish",
				"ksh",
				"zsh",
			},
		},
		{
			name: "prints usage snippet when no '-' argument is given, but shell given is 'bash'",
			args: []string{"bash"},
			expectedOutput: []string{
				"# Load goenv automatically by appending",
				"# the following to ~/.bash_profile:",
				"eval \"$(goenv init -)\"",
			},
		},
		{
			name: "prints usage snippet when no '-' argument is given, but shell given is 'zsh'",
			args: []string{"zsh"},
			expectedOutput: []string{
				"# Load goenv automatically by appending",
				"# the following to ~/.zshrc:",
				"eval \"$(goenv init -)\"",
			},
		},
		{
			name: "prints usage snippet when no '-' argument is given, but shell given is 'fish'",
			args: []string{"fish"},
			expectedOutput: []string{
				"# Load goenv automatically by appending",
				"# the following to ~/.config/fish/config.fish:",
				"status --is-interactive; and source (goenv init -|psub)",
			},
		},
		{
			name: "prints usage snippet when no '-' argument is given, but shell given is 'ksh'",
			args: []string{"ksh"},
			expectedOutput: []string{
				"# Load goenv automatically by appending",
				"# the following to ~/.profile:",
				"eval \"$(goenv init -)\"",
			},
		},
		{
			name: "prints usage snippet when no '-' argument is given, but shell given is none of the well known ones",
			args: []string{"magicalshell"},
			expectedOutput: []string{
				"# Load goenv automatically by appending",
				"# the following to <unknown shell: magicalshell, replace with your profile path>:",
				"eval \"$(goenv init -)\"",
			},
		},
		{
			name:      "creates shims and versions directories when '-' argument is given",
			args:      []string{"-", "bash"},
			checkDirs: []string{"shims", "versions"},
			expectedOutput: []string{
				"export GOENV_SHELL=bash",
			},
		},
		{
			name: "includes 'goenv rehash' when '-' is specified and '--no-rehash' is not specified",
			args: []string{"-", "bash"},
			expectedOutput: []string{
				"command goenv rehash 2>/dev/null",
			},
		},
		{
			name: "does not include 'goenv rehash' when '-' and '--no-rehash' are specified",
			args: []string{"-", "--no-rehash", "bash"},
			unexpectedOutput: []string{
				"command goenv rehash 2>/dev/null",
			},
		},
		{
			name: "prints bootstrap script for bash",
			args: []string{"-", "bash"},
			expectedOutput: []string{
				"export GOENV_SHELL=bash",
				"export GOENV_ROOT=",
				"if [ -z \"${GOENV_RC_FILE:-}\" ]; then",
				"GOENV_RC_FILE=\"${HOME}/.goenvrc\"",
				"if [ -e \"${GOENV_RC_FILE:-}\" ]; then",
				"source \"${GOENV_RC_FILE}\"",
				"if [ \"${PATH#*$GOENV_ROOT/shims}\" = \"${PATH}\" ]; then",
				"export PATH=\"${GOENV_ROOT}/shims:${PATH}\"",
				"export PATH=\"${PATH}:${GOENV_ROOT}/shims\"",
				"command goenv rehash 2>/dev/null",
				"goenv() {",
				"local command",
				"case \"$command\" in",
				"rehash|shell)",
				"eval \"$(goenv \"sh-$command\" \"$@\")\";;",
				"command goenv \"$command\" \"$@\";;",
			},
		},
		{
			name: "prints bootstrap script for zsh",
			args: []string{"-", "zsh"},
			expectedOutput: []string{
				"export GOENV_SHELL=zsh",
				"export GOENV_ROOT=",
				"if [ -z \"${GOENV_RC_FILE:-}\" ]; then",
				"command goenv rehash 2>/dev/null",
				"goenv() {",
			},
		},
		{
			name: "prints bootstrap script for fish",
			args: []string{"-", "fish"},
			expectedOutput: []string{
				"set -gx GOENV_SHELL fish",
				"set -gx GOENV_ROOT ",
				"if test -z $GOENV_RC_FILE",
				"set GOENV_RC_FILE $HOME/.goenvrc",
				"if test -e $GOENV_RC_FILE",
				"source $GOENV_RC_FILE",
				"if not contains $GOENV_ROOT/shims $PATH",
				"set -gx PATH $GOENV_ROOT/shims $PATH",
				"command goenv rehash 2>/dev/null",
				"function goenv",
				"set command $argv[1]",
				"switch \"$command\"",
				"case rehash shell",
				"source (goenv \"sh-$command\" $argv|psub)",
			},
		},
		{
			name: "prints bootstrap script for ksh",
			args: []string{"-", "ksh"},
			expectedOutput: []string{
				"export GOENV_SHELL=ksh",
				"export GOENV_ROOT=",
				"command goenv rehash 2>/dev/null",
				"function goenv {",
				"typeset command",
			},
		},
		{
			name: "detects parent shell when '-' argument is given only",
			args: []string{"-"},
			envVars: map[string]string{
				"SHELL": "/bin/false",
			},
			expectedOutput: []string{
				"export GOENV_SHELL=bash", // Will detect bash as parent
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir, cleanup := setupTestEnv(t)
			defer cleanup()

			// Run setup function if provided
			if tt.setupFunc != nil {
				tt.setupFunc(t, tmpDir)
			}

			// Set environment variables
			for key, value := range tt.envVars {
				os.Setenv(key, value)
				defer os.Unsetenv(key)
			}

			// Execute command directly via RunE
			outputBuf := &strings.Builder{}
			errorBuf := &strings.Builder{}

			// Create a temporary command with our buffers
			cmd := &cobra.Command{}
			cmd.SetOut(outputBuf)
			cmd.SetErr(errorBuf)

			// Parse flags manually from args
			filteredArgs := []string{}
			initFlags.complete = false
			initFlags.noRehash = false

			for _, arg := range tt.args {
				if arg == "--complete" {
					initFlags.complete = true
				} else if arg == "--no-rehash" {
					initFlags.noRehash = true
				} else {
					filteredArgs = append(filteredArgs, arg)
				}
			}

			err := runInit(cmd, filteredArgs)

			// Check error expectation
			if tt.shouldFail {
				if err == nil {
					t.Fatalf("Expected command to fail, but it succeeded")
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
			}

			// Check output
			output := outputBuf.String()
			for _, expected := range tt.expectedOutput {
				if !strings.Contains(output, expected) {
					t.Errorf("Expected output to contain %q, but it was missing.\nFull output:\n%s", expected, output)
				}
			}

			// Check unexpected output
			for _, unexpected := range tt.unexpectedOutput {
				if strings.Contains(output, unexpected) {
					t.Errorf("Expected output to NOT contain %q, but it was present.\nFull output:\n%s", unexpected, output)
				}
			}

			// Check directories were created
			for _, dir := range tt.checkDirs {
				dirPath := filepath.Join(tmpDir, dir)
				if _, err := os.Stat(dirPath); os.IsNotExist(err) {
					t.Errorf("Expected directory %q to exist, but it doesn't", dir)
				}
			}
		})
	}
}
