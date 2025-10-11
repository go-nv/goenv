package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/go-nv/goenv/internal/helptext"
	"github.com/spf13/cobra"
)

var hooksCmd = &cobra.Command{
	Use:          "hooks <command>",
	Short:        "List hook scripts for a given goenv command",
	Long:         "Find and list all hook scripts that should be executed for the given command",
	Args:         cobra.ExactArgs(1),
	RunE:         runHooks,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(hooksCmd)
	hooksCmd.Flags().Bool("complete", false, "Show completion options")
	helptext.SetCommandHelp(hooksCmd)
}

func runHooks(cmd *cobra.Command, args []string) error {
	// Handle completion flag
	if complete, _ := cmd.Flags().GetBool("complete"); complete {
		// List commands that support hooks
		fmt.Fprintln(cmd.OutOrStdout(), "exec")
		fmt.Fprintln(cmd.OutOrStdout(), "rehash")
		fmt.Fprintln(cmd.OutOrStdout(), "version-name")
		fmt.Fprintln(cmd.OutOrStdout(), "version-origin")
		fmt.Fprintln(cmd.OutOrStdout(), "which")
		return nil
	}

	commandName := args[0]

	// Get GOENV_HOOK_PATH, default to $GOENV_ROOT/goenv.d
	hookPath := os.Getenv("GOENV_HOOK_PATH")
	if hookPath == "" {
		goenvRoot := os.Getenv("GOENV_ROOT")
		if goenvRoot == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil
			}
			goenvRoot = filepath.Join(home, ".goenv")
		}
		hookPath = filepath.Join(goenvRoot, "goenv.d")
	}

	var hooks []string

	// Search each path in GOENV_HOOK_PATH
	for _, dir := range filepath.SplitList(hookPath) {
		// Resolve relative paths
		absDir, err := filepath.Abs(dir)
		if err != nil {
			continue
		}

		// Look for hooks for this command
		commandHooksDir := filepath.Join(absDir, commandName)
		entries, err := os.ReadDir(commandHooksDir)
		if err != nil {
			continue
		}

		for _, entry := range entries {
			if entry.IsDir() {
				continue
			}

			name := entry.Name()
			// Only include .bash files
			if !strings.HasSuffix(name, ".bash") {
				continue
			}

			hookPath := filepath.Join(commandHooksDir, name)

			// Resolve symlinks
			resolvedPath, err := filepath.EvalSymlinks(hookPath)
			if err != nil {
				// If can't resolve, use original path
				resolvedPath = hookPath
			}

			hooks = append(hooks, resolvedPath)
		}
	}

	// Sort hooks alphabetically
	sort.Strings(hooks)

	// Print hooks
	for _, hook := range hooks {
		fmt.Fprintln(cmd.OutOrStdout(), hook)
	}

	return nil
}
