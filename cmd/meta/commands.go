package meta

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/helptext"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var (
	commandsSh   bool
	commandsNoSh bool
)

var commandsCmd = &cobra.Command{
	Use:          "commands",
	Short:        "List all available goenv commands",
	Long:         "List all goenv commands including installed versions",
	Args:         cobra.NoArgs,
	RunE:         runCommands,
	SilenceUsage: true,
}

func init() {
	commandsCmd.Flags().BoolVar(&commandsSh, "sh", false, "List only commands containing 'sh'")
	commandsCmd.Flags().BoolVar(&commandsNoSh, "no-sh", false, "List commands not containing 'sh'")
	commandsCmd.Flags().Bool("complete", false, "Show completion options")
	commandsCmd.Flags().Lookup("complete").Hidden = true

	cmdpkg.RootCmd.AddCommand(commandsCmd)
	helptext.SetCommandHelp(commandsCmd)
}

func runCommands(cmd *cobra.Command, args []string) error {
	// Handle completion flag
	if complete, _ := cmd.Flags().GetBool("complete"); complete {
		fmt.Fprintln(cmd.OutOrStdout(), "--sh")
		fmt.Fprintln(cmd.OutOrStdout(), "--no-sh")
		return nil
	}

	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config
	commands := []string{}

	// Add installed versions
	versionsDir := filepath.Join(cfg.Root, "versions")
	if entries, err := os.ReadDir(versionsDir); err == nil {
		for _, entry := range entries {
			if entry.IsDir() {
				commands = append(commands, entry.Name())
			}
		}
	}

	// Add Cobra registered commands
	for _, subCmd := range cmdpkg.RootCmd.Commands() {
		if subCmd.Use != "" && !subCmd.Hidden {
			commands = append(commands, subCmd.Name())
		}
	}

	// Add standard goenv commands that might not be registered yet
	standardCommands := []string{
		"commands", "completions", "exec", "global", "help",
		"init", "install", "installed", "latest", "local", "prefix",
		"rehash", "root", "shell", "shims", "system", "uninstall",
		"version", "version-file", "version-file-read", "version-file-write",
		"version-name", "version-origin", "versions", "whence", "which",
	}
	commands = append(commands, standardCommands...)

	// Track discovered commands and their paths for duplicate detection
	commandPaths := make(map[string][]string)

	// Check PATH for additional goenv commands
	pathEnv := os.Getenv(utils.EnvVarPath)
	for _, dir := range filepath.SplitList(pathEnv) {
		if entries, err := os.ReadDir(dir); err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					name := entry.Name()
					if strings.HasPrefix(name, "goenv-") {
						cmdName := strings.TrimPrefix(name, "goenv-")
						// Check if executable
						fullPath := filepath.Join(dir, name)
						if utils.IsExecutableFile(fullPath) {
							commands = append(commands, cmdName)
							commandPaths[cmdName] = append(commandPaths[cmdName], fullPath)
						}
					}
				}
			}
		}
	}

	// Check for duplicates and warn (to stderr, doesn't affect output)
	for cmdName, paths := range commandPaths {
		if len(paths) > 1 {
			fmt.Fprintf(os.Stderr, "⚠️  Warning: Multiple 'goenv-%s' commands found in PATH:\n", cmdName)
			for i, path := range paths {
				marker := "  "
				if i == 0 {
					marker = "→ " // First one wins (appears first in PATH)
				}
				fmt.Fprintf(os.Stderr, "  %s%s\n", marker, path)
			}
			fmt.Fprintln(os.Stderr)
		}
	}

	// Deduplicate and sort
	commandSet := make(map[string]bool)
	for _, c := range commands {
		// Skip Cobra built-in commands that don't exist in bash version
		if c == "completion" {
			continue
		}
		commandSet[c] = true
	}

	commands = make([]string, 0, len(commandSet))
	for c := range commandSet {
		commands = append(commands, c)
	}
	slices.Sort(commands)

	// Filter based on flags
	// In bash version, --sh filters commands that have goenv-sh-* executables
	// These are: rehash (has both goenv-rehash and goenv-sh-rehash), shell
	// Note: rehash appears in BOTH --sh and --no-sh because it has both implementations
	shOnlyCommands := map[string]bool{
		"shell": true,
	}
	bothCommands := map[string]bool{
		"rehash": true, // Has both regular and sh- versions
	}

	var filtered []string
	for _, c := range commands {
		isSh := shOnlyCommands[c] || bothCommands[c]
		isRegular := !shOnlyCommands[c] // Everything except shell-only commands

		if commandsSh && !isSh {
			continue
		}
		if commandsNoSh && !isRegular {
			continue
		}
		filtered = append(filtered, c)
	}

	// Print commands to stdout (not stderr)
	for _, c := range filtered {
		fmt.Fprintln(cmd.OutOrStdout(), c)
	}

	return nil
}
