package helptext

import (
	"fmt"
	"strings"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

// CommandHelp represents the help text for a command
type CommandHelp struct {
	Usage       string   // Usage line(s)
	Summary     string   // Short description
	Description string   // Detailed description
	Examples    []string // Optional examples
}

// Format returns the formatted help text matching bash version style
// If withVersionSubstitution is true, it will replace version examples with actual installed versions
func (h *CommandHelp) Format() string {
	return h.format(true)
}

// FormatWithoutVersions returns formatted help without version substitution
func (h *CommandHelp) FormatWithoutVersions() string {
	return h.format(false)
}

func (h *CommandHelp) format(substituteVersions bool) string {
	var parts []string

	// Usage section
	usage := h.Usage
	description := h.Description
	if description == "" {
		description = h.Summary
	}

	// Substitute version examples if requested
	if substituteVersions {
		latestVersion := getLatestInstalledVersion()
		if latestVersion != "" {
			// Add version examples to description where bash shows them
			usage = strings.ReplaceAll(usage, "$LATEST_VERSION", latestVersion)
			description = strings.ReplaceAll(description, "$LATEST_VERSION", latestVersion)
			description = addVersionExamples(description, latestVersion)
		}
	}

	if usage != "" {
		parts = append(parts, "Usage: "+usage)
	}

	// Description section (can be multi-line)
	if description != "" {
		parts = append(parts, "", description)
	}

	// Examples section
	if len(h.Examples) > 0 {
		parts = append(parts, "")
		for _, example := range h.Examples {
			parts = append(parts, example)
		}
	}

	// Add trailing newline to match bash version
	return strings.Join(parts, "\n") + "\n"
}

// getLatestInstalledVersion returns the latest installed Go version or empty string if none
func getLatestInstalledVersion() string {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)

	installed, err := mgr.ListInstalledVersions()
	if err != nil || len(installed) == 0 {
		return ""
	}

	// Get latest version
	latest := ""
	for _, v := range installed {
		if latest == "" || compareVersions(v, latest) > 0 {
			latest = v
		}
	}
	return latest
}

// addVersionExamples adds version examples like bash does: (1.23.4)
func addVersionExamples(description, latestVersion string) string {
	if latestVersion == "" {
		return description
	}

	// Add version examples to specific patterns found in bash help
	replacements := map[string]string{
		// For global/local commands
		"`latest` sets the latest installed version.":             fmt.Sprintf("`latest` sets the latest installed version (%s).", latestVersion),
		"`1` sets the latest installed major version.":            fmt.Sprintf("`1` sets the latest installed major version (%s).", latestVersion),
		"`23` or `1.23` sets the latest installed minor version.": fmt.Sprintf("`23` or `1.23` sets the latest installed minor version (%s).", latestVersion),
		"`1.23.4` sets this installed version.":                   fmt.Sprintf("`1.23.4` sets this installed version (%s).", latestVersion),
		// For prefix command (different wording: "displays" instead of "sets")
		"`latest` is given, displays the latest installed version.":   fmt.Sprintf("`latest` is given, displays the latest installed version (%s).", latestVersion),
		"`1` displays the latest installed major version.":            fmt.Sprintf("`1` displays the latest installed major version (%s).", latestVersion),
		"`23` or `1.23` displays the latest installed minor version.": fmt.Sprintf("`23` or `1.23` displays the latest installed minor version (%s).", latestVersion),
		"`1.23.4` displays this installed version.":                   fmt.Sprintf("`1.23.4` displays this installed version (%s).", latestVersion),
		// For version-file-write command (different wording: "writes")
		"`latest` writes the latest installed version.":             fmt.Sprintf("`latest` writes the latest installed version (%s).", latestVersion),
		"`1` writes the latest installed major version.":            fmt.Sprintf("`1` writes the latest installed major version (%s).", latestVersion),
		"`23` or `1.23` writes the latest installed minor version.": fmt.Sprintf("`23` or `1.23` writes the latest installed minor version (%s).", latestVersion),
		"`1.23.4` writes this installed version.":                   fmt.Sprintf("`1.23.4` writes this installed version (%s).", latestVersion),
	}

	result := description
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}
	return result
}

// compareVersions compares two version strings (returns >0 if v1 > v2)
func compareVersions(v1, v2 string) int {
	// Strip "go" prefix if present
	v1 = strings.TrimPrefix(v1, "go")
	v2 = strings.TrimPrefix(v2, "go")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	for i := 0; i < len(parts1) && i < len(parts2); i++ {
		// Simple string comparison works for numbers
		if parts1[i] != parts2[i] {
			if parts1[i] > parts2[i] {
				return 1
			}
			return -1
		}
	}

	// Longer version is greater
	if len(parts1) > len(parts2) {
		return 1
	} else if len(parts1) < len(parts2) {
		return -1
	}
	return 0
}

// Registry holds all command help text
var Registry = map[string]*CommandHelp{
	"version": {
		Usage: "goenv version",
		Description: `Shows the currently selected Go version and how it was
selected. To obtain only the version string, use ` + "`goenv version-name'." + ``,
	},
	"version-name": {
		Usage:   "goenv version-name",
		Summary: "Show the current Go version",
	},
	"version-origin": {
		Usage:   "goenv version-origin",
		Summary: "Explain how the current Go version is set",
	},
	"version-file": {
		Usage:   "goenv version-file [<dir>]",
		Summary: "Detect the file that sets the current goenv version",
	},
	"version-file-read": {
		Usage:   "goenv version-file-read <file>",
		Summary: "Reads specified version file if it exists",
	},
	"version-file-write": {
		Usage:       "goenv version-file-write <file> <version>...",
		Description: "If a specified version is not installed, only display an error message and abort.\nIf only a single <version> `system` is specified and installed, display previous version (if any) and remove file (similar to --unset).\n\n<version> should be a string matching a Go version known to goenv.\n<version> `latest` writes the latest installed version.\n<version> `1` writes the latest installed major version.\n<version> `23` or `1.23` writes the latest installed minor version.\n<version> `1.23.4` writes this installed version.\nRun `goenv versions` for a list of available Go versions.",
	},
	"versions": {
		Usage:       "goenv versions [--bare] [--skip-aliases]",
		Description: "Lists all Go versions found in `$GOENV_ROOT/versions/*'.",
	},
	"whence": {
		Usage:   "goenv whence [--path] <command>",
		Summary: "List all Go versions that contain the given executable",
	},
	"which": {
		Usage:       "goenv which <command>",
		Description: "Displays the full path to the executable that goenv will invoke when\nyou run the given command.",
	},
	"local": {
		Usage:       "goenv local [<version>]\n       goenv local --unset",
		Description: "Sets the local application-specific Go version by writing the\nversion name to a file named `.go-version'.\n\nWhen you run a Go command, goenv will look for a `.go-version'\nfile in the current directory and each parent directory. If no such\nfile is found in the tree, goenv will use the global Go version\nspecified with `goenv global'. A version specified with the\n`GOENV_VERSION' environment variable takes precedence over local\nand global versions.\n\n<version> should be a string matching a Go version known to goenv.\nIf no <version> is given, displays the local version if configured.\n<version> `system` unsets the previous version and displays it if configured.\n<version> `latest` sets the latest installed version.\n<version> `1` sets the latest installed major version.\n<version> `23` or `1.23` sets the latest installed minor version.\n<version> `1.23.4` sets this installed version.\nIf no version can be found or no versions are installed or configured, an error message will be displayed.\nRun `goenv versions` for a list of available Go versions.",
	},
	"global": {
		Usage:       "goenv global [<version>]",
		Description: "Sets the global Go version. You can override the global version at\nany time by setting a directory-specific version with `goenv local'\nor by setting the `GOENV_VERSION' environment variable.\n\n<version> should be a string matching a Go version known to goenv.\nIf no <version> is given, displays the global version if configured.\n<version> `system` unsets the previous version and displays it if configured.\n<version> `latest` sets the latest installed version.\n<version> `1` sets the latest installed major version.\n<version> `23` or `1.23` sets the latest installed minor version.\n<version> `1.23.4` sets this installed version.\nIf no version can be found or no versions are installed or configured, an error message will be displayed.\nRun `goenv versions` for a list of available Go versions.",
	},
	"shell": {
		Usage:       "goenv shell <version>\n       goenv shell --unset",
		Description: "Sets a shell-specific Go version by setting the `GOENV_VERSION'\nenvironment variable in your shell. This version overrides local\napplication-specific versions and the global version.\n\n<version> should be a string matching a Go version known to goenv.\nThe special version string `system' will use your default system Go.\nRun `goenv versions' for a list of available Go versions.",
	},
	"install": {
		Usage: `goenv install [-f] [-s] [-kvpq] [<version>|latest|unstable]
       goenv install -l|--list
       goenv install --version`,
		Description: `  -l/--list          List all available versions
  -f/--force         Install even if the version appears to be installed already
  -s/--skip-existing Skip if the version appears to be installed already

  If no version is specified, ` + "`goenv local`" + ` will be used to determine the
  desired version.

  go-build options:

  -k/--keep          Keep source tree in $GOENV_BUILD_ROOT after installation
                     (defaults to $GOENV_ROOT/sources)
  -p/--patch         Apply a patch from stdin before building
  -v/--verbose       Verbose mode: print compilation status to stdout
  -q/--quiet         Disable Progress Bar
  --version          Show version of go-build
  -g/--debug         Build a debug version

For detailed information on installing Go versions with
go-build, including a list of environment variables for adjusting
compilation, see: https://github.com/go-nv/goenv#readme`,
	},
	"uninstall": {
		Usage:       "goenv uninstall [-f|--force] <version>",
		Description: "   -f  Attempt to remove the specified version without prompting\n       for confirmation. Still displays error message if version does not exist.\n\nSee `goenv versions` for a complete list of installed versions.",
	},
	"rehash": {
		Usage:   "goenv rehash",
		Summary: "Rehash goenv shims (run this after installing executables)",
	},
	"root": {
		Usage:   "goenv root",
		Summary: "Display the root directory where versions and shims are kept",
	},
	"prefix": {
		Usage:       "goenv prefix [<version>]",
		Description: "Displays the directory where a Go version is installed.\nIf no <version> is given, displays the location of the currently selected version.\n<version> `latest` is given, displays the latest installed version.\n<version> `system` displays the system Go location if installed.\n<version> `1` displays the latest installed major version.\n<version> `23` or `1.23` displays the latest installed minor version.\n<version> `1.23.4` displays this installed version.\nIf no version can be found or no versions are installed, an error message will be displayed.\nRun `goenv versions` for a list of available Go versions.",
	},
	"exec": {
		Usage:       "goenv exec <command> [arg1 arg2...]",
		Description: "Runs an executable by first preparing PATH so that the selected\nGo version's `bin' directory is at the front.",
	},
	"shims": {
		Usage:   "goenv shims [--short]",
		Summary: "List existing goenv shims",
	},
	"init": {
		Usage:       "eval \"$(goenv init - [--no-rehash] [<shell>])\"",
		Description: "Configure the shell environment for goenv",
	},
	"completions": {
		Usage:       "goenv completions <command> [arg1 arg2...]",
		Description: "Provides auto-completion for itself and other commands by calling them with `--complete`.",
	},
	"commands": {
		Usage:   "goenv commands [--sh|--no-sh]",
		Summary: "List all available commands of goenv",
	},
}

// Get returns the help text for a command
func Get(commandName string) *CommandHelp {
	if help, ok := Registry[commandName]; ok {
		return help
	}
	return nil
}

// SetCustomHelp allows commands to override their help text at runtime
func SetCustomHelp(commandName string, help *CommandHelp) {
	Registry[commandName] = help
}

// GenerateUsageTemplate returns a Cobra usage template that uses our help text
func GenerateUsageTemplate(commandName string) string {
	help := Get(commandName)
	if help == nil {
		return "" // Use default Cobra help
	}
	return help.Format()
}

// UsageFunc returns a function that can be used as Cobra's UsageFunc
func UsageFunc(commandName string) func(cmd interface{ Println(...interface{}) }) error {
	return func(cmd interface{ Println(...interface{}) }) error {
		help := Get(commandName)
		if help != nil {
			cmd.Println(help.Format())
		} else {
			cmd.Println(fmt.Sprintf("Help text not available for command: %s", commandName))
		}
		return nil
	}
}

// SetCommandHelp configures a Cobra command to use our custom help text
// Usage:
//
//	func init() {
//	    rootCmd.AddCommand(yourCmd)
//	    helptext.SetCommandHelp(yourCmd)
//	}
func SetCommandHelp(cmd *cobra.Command) {
	commandName := cmd.Name()
	help := Get(commandName)
	if help != nil {
		helpFunc := func(c *cobra.Command, args []string) {
			c.Println(help.Format())
		}
		cmd.SetHelpFunc(helpFunc)
		cmd.SetUsageFunc(func(c *cobra.Command) error {
			c.Println(help.Format())
			return nil
		})
	}
}
