package cmd

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

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
}

func runHooks(cmd *cobra.Command, args []string) error {
	// Handle completion flag
	if complete, _ := cmd.Flags().GetBool("complete"); complete {
		// List commands that support hooks
		cmd.Println("exec")
		cmd.Println("rehash")
		cmd.Println("version-name")
		cmd.Println("version-origin")
		cmd.Println("which")
		return nil
	}

	commandName := args[0]

	// Get GOENV_HOOK_PATH
	hookPath := os.Getenv("GOENV_HOOK_PATH")
	if hookPath == "" {
		// No hooks configured
		return nil
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
		cmd.Println(hook)
	}

	return nil
}
