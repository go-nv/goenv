package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var completionsCmd = &cobra.Command{
	Use:          "completions <command> [arg1 arg2...]",
	Short:        "List available completions for a command",
	Long:         "Query commands for their completion suggestions",
	Args:         cobra.MinimumNArgs(1),
	RunE:         runCompletions,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(completionsCmd)
	helptext.SetCommandHelp(completionsCmd)
}

func runCompletions(cmd *cobra.Command, args []string) error {
	commandName := args[0]
	completionArgs := args[1:]

	// Always print --help as first option
	fmt.Fprintln(cmd.OutOrStdout(), "--help")

	// Try to find the command
	cfg := config.Load()

	// Check if it's a Cobra registered command first
	for _, subCmd := range rootCmd.Commands() {
		if subCmd.Name() == commandName {
			// Try to call it with --complete flag
			if subCmd.Flags().Lookup("complete") != nil {
				// Create a new command instance to capture output
				testCmd := &cobra.Command{
					Use:  subCmd.Use,
					RunE: subCmd.RunE,
				}
				testCmd.SetArgs(append([]string{"--complete"}, completionArgs...))
				// Copy flags
				subCmd.Flags().VisitAll(func(flag *pflag.Flag) {
					testCmd.Flags().AddFlag(flag)
				})

				// Execute and capture output
				testCmd.SetOut(cmd.OutOrStdout())
				if err := testCmd.Execute(); err == nil {
					return nil
				}
			}
		}
	}

	// Try to find command in PATH
	commandPath := ""
	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		candidate := filepath.Join(dir, "goenv-"+commandName)
		// On Windows, all files are "executable"; on Unix, check the executable bit
		if info, err := os.Stat(candidate); err == nil && (runtime.GOOS == "windows" || info.Mode()&0111 != 0) {
			commandPath = candidate
			break
		}
	}

	// If not found in PATH, try libexec
	if commandPath == "" {
		libexecDir := filepath.Join(cfg.Root, "libexec")
		candidate := filepath.Join(libexecDir, "goenv-"+commandName)
		// On Windows, all files are "executable"; on Unix, check the executable bit
		if info, err := os.Stat(candidate); err == nil && (runtime.GOOS == "windows" || info.Mode()&0111 != 0) {
			commandPath = candidate
		}
	}

	if commandPath == "" {
		// Command not found, just return --help
		return nil
	}

	// Check if command supports completions by looking for "# provide goenv completions" comment
	content, err := os.ReadFile(commandPath)
	if err != nil {
		return nil
	}

	// Case-insensitive check for completion support
	lines := strings.ToLower(string(content))
	if !strings.Contains(lines, "# provide goenv completions") {
		// No completion support, just --help already printed
		return nil
	}

	// Execute command with --complete flag
	execCmd := exec.Command(commandPath, append([]string{"--complete"}, completionArgs...)...)
	execCmd.Stdout = cmd.OutOrStdout()
	execCmd.Stderr = cmd.ErrOrStderr()
	execCmd.Env = os.Environ()

	if err := execCmd.Run(); err != nil {
		// Command failed, but we already printed --help
		return nil
	}

	return nil
}
