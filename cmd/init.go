package cmd

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:   "init [shell]",
	Short: "Configure the shell environment for goenv",
	Long: `Configure the shell environment for goenv.
	
This command should be evaluated in your shell:
  eval "$(goenv init -)"

Or add it to your shell's startup file (.bashrc, .zshrc, etc.)`,
	RunE: runInit,
}

var initFlags struct {
	noRehash bool
	complete bool
}

func init() {
	rootCmd.AddCommand(initCmd)
	initCmd.SilenceUsage = true
	initCmd.Flags().BoolVar(&initFlags.noRehash, "no-rehash", false, "Skip rehashing shims")
	initCmd.Flags().BoolVar(&initFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = initCmd.Flags().MarkHidden("complete")
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	if initFlags.complete {
		for _, option := range []string{"-", "--no-rehash", "bash", "fish", "ksh", "zsh"} {
			cmd.Println(option)
		}
		return nil
	}

	dashMode := false
	var explicitShell string
	for _, arg := range args {
		if arg == "-" {
			dashMode = true
			continue
		}
		if explicitShell == "" {
			explicitShell = arg
		}
	}

	shell := resolveShell(explicitShell, dashMode)

	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Initializing for %s shell\n", shell)
	}

	if !dashMode {
		cmd.Print(renderUsageSnippet(shell))
		return nil
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to prepare goenv directories: %w", err)
	}

	cmd.Print(renderInitScript(shell, cfg, initFlags.noRehash))
	return nil
}

func resolveShell(explicit string, dashMode bool) string {
	if explicit != "" {
		return explicit
	}

	// Check GOENV_SHELL first (used by sh-shell command)
	if goenvShell := os.Getenv("GOENV_SHELL"); goenvShell != "" {
		return goenvShell
	}

	if dashMode {
		if detected := detectParentShell(); detected != "" {
			return detected
		}
	}

	return detectEnvShell()
}

func detectEnvShell() string {
	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		if base := filepath.Base(shellPath); base != "" {
			return base
		}
	}
	return "bash"
}

func detectParentShell() string {
	ppid := os.Getppid()
	cmd := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "args=")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return ""
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return ""
	}

	shell := strings.TrimPrefix(fields[0], "-")
	if shell == "" {
		return ""
	}

	return filepath.Base(shell)
}

func renderUsageSnippet(shell string) string {
	profile := determineProfilePath(shell)

	var builder strings.Builder
	builder.WriteString("# Load goenv automatically by appending\n")
	builder.WriteString(fmt.Sprintf("# the following to %s:\n\n", profile))

	if shell == "fish" {
		builder.WriteString("status --is-interactive; and source (goenv init -|psub)\n")
	} else {
		builder.WriteString("eval \"$(goenv init -)\"\n")
	}

	builder.WriteString("\n")
	return builder.String()
}

func determineProfilePath(shell string) string {
	switch shell {
	case "bash":
		home := os.Getenv("HOME")
		if home != "" {
			bashrc := filepath.Join(home, ".bashrc")
			bashProfile := filepath.Join(home, ".bash_profile")
			if _, err := os.Stat(bashrc); err == nil {
				if _, err := os.Stat(bashProfile); err != nil {
					return "~/.bashrc"
				}
			}
		}
		return "~/.bash_profile"
	case "zsh":
		return "~/.zshrc"
	case "ksh":
		return "~/.profile"
	case "fish":
		return "~/.config/fish/config.fish"
	default:
		return fmt.Sprintf("<unknown shell: %s, replace with your profile path>", shell)
	}
}

func renderInitScript(shell string, cfg *config.Config, noRehash bool) string {
	var builder strings.Builder

	switch shell {
	case "fish":
		fmt.Fprintf(&builder, "set -gx GOENV_SHELL %s\n", shell)
		fmt.Fprintf(&builder, "set -gx GOENV_ROOT %s\n", cfg.Root)
		builder.WriteString("if test -z $GOENV_RC_FILE\n")
		builder.WriteString("  set GOENV_RC_FILE $HOME/.goenvrc\n")
		builder.WriteString("end\n")
		builder.WriteString("if test -e $GOENV_RC_FILE\n")
		builder.WriteString("  source $GOENV_RC_FILE\n")
		builder.WriteString("end\n")
		builder.WriteString("if not contains $GOENV_ROOT/shims $PATH\n")
		builder.WriteString("  if test \"$GOENV_PATH_ORDER\" = \"front\"\n")
		builder.WriteString("    set -gx PATH $GOENV_ROOT/shims $PATH\n")
		builder.WriteString("  else\n")
		builder.WriteString("    set -gx PATH $PATH $GOENV_ROOT/shims\n")
		builder.WriteString("  end\n")
		builder.WriteString("end\n")
	default:
		fmt.Fprintf(&builder, "export GOENV_SHELL=%s\n", shell)
		fmt.Fprintf(&builder, "export GOENV_ROOT=%s\n", cfg.Root)
		builder.WriteString("if [ -z \"${GOENV_RC_FILE:-}\" ]; then\n")
		builder.WriteString("  GOENV_RC_FILE=\"${HOME}/.goenvrc\"\n")
		builder.WriteString("fi\n")
		builder.WriteString("if [ -e \"${GOENV_RC_FILE:-}\" ]; then\n")
		builder.WriteString("  source \"${GOENV_RC_FILE}\"\n")
		builder.WriteString("fi\n")
		builder.WriteString("if [ \"${PATH#*$GOENV_ROOT/shims}\" = \"${PATH}\" ]; then\n")
		builder.WriteString("  if [ \"${GOENV_PATH_ORDER:-}\" = \"front\" ] ; then\n")
		builder.WriteString("    export PATH=\"${GOENV_ROOT}/shims:${PATH}\"\n")
		builder.WriteString("  else\n")
		builder.WriteString("    export PATH=\"${PATH}:${GOENV_ROOT}/shims\"\n")
		builder.WriteString("  fi\n")
		builder.WriteString("fi\n")
	}

	if completion := findCompletionPath(shell); completion != "" {
		fmt.Fprintf(&builder, "source '%s'\n", completion)
	}

	if !noRehash {
		builder.WriteString("command goenv rehash 2>/dev/null\n")
	}

	builder.WriteString(renderShellFunction(shell))

	if !noRehash {
		builder.WriteString("  goenv rehash --only-manage-paths\n")
	}

	return builder.String()
}

func renderShellFunction(shell string) string {
	specialCommands := []string{"rehash", "shell"}

	var builder strings.Builder

	switch shell {
	case "fish":
		builder.WriteString("function goenv\n")
		builder.WriteString("  set command $argv[1]\n")
		builder.WriteString("  set -e argv[1]\n\n")
		builder.WriteString("  switch \"$command\"\n")
		builder.WriteString("  case " + strings.Join(specialCommands, " ") + "\n")
		builder.WriteString("    source (goenv \"sh-$command\" $argv|psub)\n")
		builder.WriteString("  case '*'\n")
		builder.WriteString("    command goenv \"$command\" $argv\n")
		builder.WriteString("  end\n")
		builder.WriteString("end\n")
	case "ksh":
		builder.WriteString("function goenv {\n")
		builder.WriteString("  typeset command\n")
		builder.WriteString("  command=\"$1\"\n")
		builder.WriteString("  if [ \"$#\" -gt 0 ]; then\n")
		builder.WriteString("    shift\n")
		builder.WriteString("  fi\n\n")
		builder.WriteString("  case \"$command\" in\n")
		builder.WriteString("  " + strings.Join(specialCommands, "|") + ")\n")
		builder.WriteString("    eval \"$(goenv \"sh-$command\" \"$@\")\";;\n")
		builder.WriteString("  *)\n")
		builder.WriteString("    command goenv \"$command\" \"$@\";;\n")
		builder.WriteString("  esac\n")
		builder.WriteString("}\n")
	default:
		builder.WriteString("goenv() {\n")
		builder.WriteString("  local command\n")
		builder.WriteString("  command=\"$1\"\n")
		builder.WriteString("  if [ \"$#\" -gt 0 ]; then\n")
		builder.WriteString("    shift\n")
		builder.WriteString("  fi\n\n")
		builder.WriteString("  case \"$command\" in\n")
		builder.WriteString("  " + strings.Join(specialCommands, "|") + ")\n")
		builder.WriteString("    eval \"$(goenv \"sh-$command\" \"$@\")\";;\n")
		builder.WriteString("  *)\n")
		builder.WriteString("    command goenv \"$command\" \"$@\";;\n")
		builder.WriteString("  esac\n")
		builder.WriteString("}\n")
	}

	return builder.String()
}

func findCompletionPath(shell string) string {
	root := getInstallRoot()
	if root == "" {
		return ""
	}

	completion := filepath.Join(root, "completions", fmt.Sprintf("goenv.%s", shell))
	if _, err := os.Stat(completion); err == nil {
		return completion
	}

	return ""
}

var (
	installRootOnce sync.Once
	installRoot     string
)

func getInstallRoot() string {
	installRootOnce.Do(func() {
		if envRoot := os.Getenv("GOENV_INSTALL_ROOT"); envRoot != "" {
			installRoot = envRoot
			return
		}

		if exe, err := os.Executable(); err == nil {
			dir := filepath.Dir(exe)
			candidate := filepath.Clean(filepath.Join(dir, ".."))
			if _, err := os.Stat(filepath.Join(candidate, "completions")); err == nil {
				installRoot = candidate
				return
			}
		}

		if _, file, _, ok := runtime.Caller(0); ok {
			candidate := filepath.Clean(filepath.Join(filepath.Dir(file), ".."))
			installRoot = candidate
			return
		}

		installRoot = ""
	})

	return installRoot
}

// Helper commands for shell integration
var shShellCmd = &cobra.Command{
	Use:    "sh-shell [version]",
	Short:  "Set or show the shell-specific Go version",
	Hidden: true,
	RunE:   runShShell,
}

func init() {
	rootCmd.AddCommand(shShellCmd)
}

func runShShell(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	// Handle completion request
	if len(args) == 1 && args[0] == "--complete" {
		fmt.Fprintln(cmd.OutOrStdout(), "--unset")
		fmt.Fprintln(cmd.OutOrStdout(), "system")
		// Print all installed versions
		mgr := manager.NewManager(cfg)
		versions, _ := mgr.ListInstalledVersions()
		for _, v := range versions {
			fmt.Fprintln(cmd.OutOrStdout(), v)
		}
		return nil
	}

	// Determine shell type
	shell := resolveShell("", true)

	// No arguments - print current GOENV_VERSION
	if len(args) == 0 {
		currentVersion := os.Getenv("GOENV_VERSION")
		if currentVersion == "" {
			return fmt.Errorf("goenv: no shell-specific version configured")
		}
		fmt.Fprintln(cmd.OutOrStdout(), `echo "$GOENV_VERSION"`)
		return nil
	}

	// Handle --unset flag
	if args[0] == "--unset" {
		if shell == "fish" {
			fmt.Fprintln(cmd.OutOrStdout(), "set -e GOENV_VERSION")
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "unset GOENV_VERSION")
		}
		return nil
	}

	// Set version - validate it exists first
	versionStr := strings.Join(args, ":")

	// Check if version exists (unless it's "system")
	if versionStr != "system" {
		versions := strings.Split(versionStr, ":")
		for _, v := range versions {
			versionPath := filepath.Join(cfg.Root, "versions", v)
			if _, err := os.Stat(versionPath); os.IsNotExist(err) {
				fmt.Fprintf(cmd.ErrOrStderr(), "goenv: version '%s' not installed\n", v)
				fmt.Fprintln(cmd.OutOrStdout(), "false")
				return fmt.Errorf("version not installed")
			}
		}
	}

	// Print shell-specific export command
	if shell == "fish" {
		fmt.Fprintf(cmd.OutOrStdout(), "set -gx GOENV_VERSION \"%s\"\n", versionStr)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "export GOENV_VERSION=\"%s\"\n", versionStr)
	}

	return nil
}
