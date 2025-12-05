package meta

import (
	"fmt"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/helptext"
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
	cmdpkg.RootCmd.AddCommand(helpCmd)
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
   refresh     Clear caches and fetch fresh version data
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

	// Handle special aliases: shell and rehash have both regular and sh- versions
	// For help purposes, treat them as the regular command names
	lookupName := commandName
	if commandName == "shell" || commandName == "rehash" {
		// Check if help text exists in registry first
		if helptext.Get(commandName) != nil {
			lookupName = commandName // Use original name for help lookup
		}
	}

	// Find the Cobra command
	targetCmd, _, err := cmd.Root().Find([]string{lookupName})
	if err != nil || targetCmd == cmd.Root() {
		// For shell/rehash, check if we have help text even if command not found
		if helptext.Get(commandName) != nil {
			// We have help text, use it directly
			if helpUsage {
				help := helptext.Get(commandName)
				if help != nil && help.Usage != "" {
					cmd.Println("Usage: " + help.Usage)
				}
				return nil
			}
			cmd.Println(helptext.Get(commandName).Format())
			return nil
		}
		// Command not found
		fmt.Fprintf(cmd.ErrOrStderr(), "goenv: no such command `%s'\n", commandName)
		return nil // Return nil to exit with code 0
	}

	if helpUsage {
		// Show only usage line
		help := helptext.Get(commandName)
		if help != nil && help.Usage != "" {
			cmd.Println("Usage: " + help.Usage)
		} else {
			cmd.Println(targetCmd.UseLine())
		}
		return nil
	}

	// Show help using our registry first, fall back to Cobra
	help := helptext.Get(commandName)
	if help != nil {
		cmd.Println(help.Format())
	} else {
		targetCmd.Help()
	}
	return nil
}
