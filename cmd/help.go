package cmd

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/spf13/cobra"
)

var helpUsage bool

var helpCmd = &cobra.Command{
	Use:          "help [--usage] <command>",
	Short:        "Display help for a command",
	Long:         "Show usage and help information for goenv commands",
	Args:         cobra.MaximumNArgs(1),
	RunE:         runHelp,
	SilenceUsage: true,
}

func init() {
	rootCmd.AddCommand(helpCmd)
	helpCmd.Flags().BoolVar(&helpUsage, "usage", false, "Show only usage line")
}

func runHelp(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		// Show general help
		cmd.Println(`Usage: goenv <command> [<args>]

Some useful goenv commands are:
   commands    List all available commands of goenv
   local       Set or show the local application-specific Go version
   global      Set or show the global Go version
   shell       Set or show the shell-specific Go version
   install     Install a Go version using go-build
   uninstall   Uninstall a specific Go version
   rehash      Rehash goenv shims (run this after installing executables)
   version     Show the current Go version and its origin
   versions    List all Go versions available to goenv
   which       Display the full path to an executable
   whence      List all Go versions that contain the given executable

See 'goenv help <command>' for information on a specific command.
For full documentation, see: https://github.com/go-nv/goenv#readme`)
		return nil
	}

	commandName := args[0]

	// Try to find the command script
	cfg := config.Load()
	var commandPath string

	// Check PATH first
	pathEnv := os.Getenv("PATH")
	for _, dir := range filepath.SplitList(pathEnv) {
		candidate := filepath.Join(dir, "goenv-"+commandName)
		if info, err := os.Stat(candidate); err == nil && info.Mode()&0111 != 0 {
			commandPath = candidate
			break
		}
	}

	// Check libexec
	if commandPath == "" {
		libexecDir := filepath.Join(cfg.Root, "libexec")
		candidate := filepath.Join(libexecDir, "goenv-"+commandName)
		if info, err := os.Stat(candidate); err == nil && info.Mode()&0111 != 0 {
			commandPath = candidate
		}
	}

	if commandPath == "" {
		return fmt.Errorf("goenv: no such command `%s'", commandName)
	}

	// Parse help from script comments
	help, err := parseHelpFromScript(commandPath)
	if err != nil {
		return err
	}

	if helpUsage {
		// Only show usage
		if help.Usage != "" {
			cmd.Println(help.Usage)
		}
		return nil
	}

	// Show full help
	if help.Usage != "" {
		cmd.Println(help.Usage)
		if help.Help != "" {
			cmd.Println()
			cmd.Println(help.Help)
		} else if help.Summary != "" {
			cmd.Println()
			cmd.Println(help.Summary)
		}
	}

	return nil
}

type commandHelp struct {
	Usage   string
	Summary string
	Help    string
}

func parseHelpFromScript(scriptPath string) (*commandHelp, error) {
	file, err := os.Open(scriptPath)
	if err != nil {
		return nil, err
	}
	defer file.Close()

	help := &commandHelp{}
	scanner := bufio.NewScanner(file)

	var usageLines []string
	var helpLines []string
	inHelp := false

	for scanner.Scan() {
		line := scanner.Text()

		// Stop at first non-comment line
		if !strings.HasPrefix(line, "#") {
			break
		}

		// Remove leading "# " or "#"
		content := strings.TrimPrefix(line, "#")
		content = strings.TrimPrefix(content, " ")

		if strings.HasPrefix(line, "# Usage:") || strings.HasPrefix(line, "#Usage:") {
			usageLines = append(usageLines, strings.TrimPrefix(content, "Usage:"))
			usageLines[len(usageLines)-1] = strings.TrimSpace(usageLines[len(usageLines)-1])
			inHelp = false
		} else if strings.HasPrefix(line, "#        ") || strings.HasPrefix(line, "#\t") {
			// Continuation of usage line
			if len(usageLines) > 0 {
				usageLines = append(usageLines, strings.TrimSpace(content))
			}
		} else if strings.HasPrefix(line, "# Summary:") {
			help.Summary = strings.TrimPrefix(content, "Summary:")
			help.Summary = strings.TrimSpace(help.Summary)
			inHelp = true
		} else if inHelp && strings.HasPrefix(line, "#") {
			// Extended help text
			if content == "" {
				helpLines = append(helpLines, "")
			} else {
				helpLines = append(helpLines, content)
			}
		}
	}

	// Format usage
	if len(usageLines) > 0 {
		help.Usage = "Usage: " + usageLines[0]
		for i := 1; i < len(usageLines); i++ {
			help.Usage += "\n       " + usageLines[i]
		}
	}

	// Format help
	if len(helpLines) > 0 {
		help.Help = strings.Join(helpLines, "\n")
		help.Help = strings.TrimSpace(help.Help)
	}

	return help, scanner.Err()
}
