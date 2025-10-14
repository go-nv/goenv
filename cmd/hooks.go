package cmd

import (
	"bufio"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
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
		fmt.Fprintln(cmd.OutOrStdout(), "install")
		fmt.Fprintln(cmd.OutOrStdout(), "uninstall")
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
			// Include all supported hook file types
			if !isValidHookFile(name) {
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

// executeHooks runs all hooks for a given command with the provided environment
func executeHooks(commandName string, env []string) error {
	hooks, err := findHooks(commandName)
	if err != nil {
		return err
	}

	for _, hook := range hooks {
		if err := executeHook(hook, env); err != nil {
			// Log the error but continue with other hooks
			if Debug {
				fmt.Fprintf(os.Stderr, "goenv: hook failed: %s: %v\n", hook, err)
			}
		}
	}

	return nil
}

// findHooks returns a list of hook scripts for the given command
func findHooks(commandName string) ([]string, error) {
	// Get GOENV_HOOK_PATH, default to $GOENV_ROOT/goenv.d
	hookPath := os.Getenv("GOENV_HOOK_PATH")
	if hookPath == "" {
		goenvRoot := os.Getenv("GOENV_ROOT")
		if goenvRoot == "" {
			home, err := os.UserHomeDir()
			if err != nil {
				return nil, err
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
			// Include all supported hook file types
			if !isValidHookFile(name) {
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

	return hooks, nil
}

// executeHook runs a single hook script with the provided environment
func executeHook(hookPath string, env []string) error {
	// Skip if the hook doesn't exist
	info, err := os.Stat(hookPath)
	if err != nil {
		return err
	}

	// On Unix systems, check if executable
	if runtime.GOOS != "windows" && info.Mode()&0111 == 0 {
		if Debug {
			fmt.Fprintf(os.Stderr, "goenv: hook not executable: %s\n", hookPath)
		}
		return nil
	}

	// Determine the interpreter and execute the hook
	interpreter, args := detectInterpreter(hookPath)
	if interpreter == "" {
		if Debug {
			fmt.Fprintf(os.Stderr, "goenv: no suitable interpreter found for hook: %s\n", hookPath)
		}
		return nil
	}

	// Build command with interpreter
	cmdArgs := append(args, hookPath)
	cmd := exec.Command(interpreter, cmdArgs...)
	cmd.Env = append(os.Environ(), env...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	return cmd.Run()
}

// isValidHookFile checks if a file is a valid hook script based on extension or content
func isValidHookFile(filename string) bool {
	// Check common hook file extensions
	validExtensions := []string{".bash", ".sh", ".ps1", ".cmd", ".bat"}
	for _, ext := range validExtensions {
		if strings.HasSuffix(filename, ext) {
			return true
		}
	}

	// Files without extension might have shebang
	if !strings.Contains(filename, ".") {
		return true
	}

	return false
}

// detectInterpreter determines which interpreter to use for a hook script
func detectInterpreter(hookPath string) (string, []string) {
	ext := strings.ToLower(filepath.Ext(hookPath))

	switch ext {
	case ".bash":
		// Try bash first, fall back to sh
		if isCommandAvailable("bash") {
			return "bash", nil
		}
		if isCommandAvailable("sh") {
			return "sh", nil
		}

	case ".sh":
		// Try sh first, fall back to bash
		if isCommandAvailable("sh") {
			return "sh", nil
		}
		if isCommandAvailable("bash") {
			return "bash", nil
		}

	case ".ps1":
		// PowerShell script
		if runtime.GOOS == "windows" {
			if isCommandAvailable("pwsh") {
				// PowerShell Core (cross-platform)
				return "pwsh", []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File"}
			}
			if isCommandAvailable("powershell") {
				// Windows PowerShell
				return "powershell", []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File"}
			}
		} else {
			// On Unix, try PowerShell Core
			if isCommandAvailable("pwsh") {
				return "pwsh", []string{"-NoProfile", "-ExecutionPolicy", "Bypass", "-File"}
			}
		}

	case ".cmd", ".bat":
		// Windows batch script
		if runtime.GOOS == "windows" {
			return "cmd", []string{"/c"}
		}

	case "":
		// No extension - check shebang
		if shebang := readShebang(hookPath); shebang != "" {
			// Parse shebang to get interpreter
			parts := strings.Fields(shebang)
			if len(parts) > 0 {
				interpreter := filepath.Base(parts[0])
				args := parts[1:]
				if isCommandAvailable(interpreter) {
					return interpreter, args
				}
			}
		}
		// Default to bash/sh if available
		if isCommandAvailable("bash") {
			return "bash", nil
		}
		if isCommandAvailable("sh") {
			return "sh", nil
		}
	}

	return "", nil
}

// readShebang reads the shebang line from a script file
func readShebang(filename string) string {
	file, err := os.Open(filename)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	if scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#!") {
			return strings.TrimSpace(line[2:])
		}
	}

	return ""
}

// isCommandAvailable checks if a command is available in the PATH
func isCommandAvailable(command string) bool {
	_, err := exec.LookPath(command)
	return err == nil
}
