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
		for _, option := range []string{"-", "--no-rehash", "bash", "fish", "ksh", "zsh", "powershell", "cmd"} {
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
	// On Windows, detect PowerShell or cmd
	if runtime.GOOS == "windows" {
		// Check if running in PowerShell
		if psVersion := os.Getenv("PSModulePath"); psVersion != "" {
			return "powershell"
		}
		// Default to cmd on Windows
		return "cmd"
	}

	shellPath := os.Getenv("SHELL")
	if shellPath != "" {
		if base := filepath.Base(shellPath); base != "" {
			return base
		}
	}
	return "bash"
}

func detectParentShell() string {
	// On Windows, ps command doesn't exist - rely on detectEnvShell instead
	if runtime.GOOS == "windows" {
		return ""
	}

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

	switch shell {
	case "fish":
		builder.WriteString("status --is-interactive; and source (goenv init -|psub)\n")
	case "powershell":
		builder.WriteString("Invoke-Expression (goenv init - | Out-String)\n")
	case "cmd":
		builder.WriteString("FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i\n")
	default:
		builder.WriteString("eval \"$(goenv init -)\"\n")
	}

	builder.WriteString("\n")
	return builder.String()
}

func determineProfilePath(shell string) string {
	switch shell {
	case "bash":
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv("HOME") // Fallback
		}
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
	case "powershell":
		return "$PROFILE"
	case "cmd":
		return "%USERPROFILE%\\autorun.cmd"
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
	case "powershell":
		fmt.Fprintf(&builder, "$env:GOENV_SHELL = \"%s\"\n", shell)
		fmt.Fprintf(&builder, "$env:GOENV_ROOT = \"%s\"\n", cfg.Root)
		builder.WriteString("if (-not $env:GOENV_RC_FILE) {\n")
		builder.WriteString("  $env:GOENV_RC_FILE = Join-Path $env:USERPROFILE \".goenvrc.ps1\"\n")
		builder.WriteString("}\n")
		builder.WriteString("if (Test-Path $env:GOENV_RC_FILE) {\n")
		builder.WriteString("  . $env:GOENV_RC_FILE\n")
		builder.WriteString("}\n")
		shimsDir := filepath.Join(cfg.Root, "shims")
		builder.WriteString(fmt.Sprintf("if ($env:PATH -notlike '*%s*') {\n", shimsDir))
		builder.WriteString("  if ($env:GOENV_PATH_ORDER -eq 'front') {\n")
		builder.WriteString(fmt.Sprintf("    $env:PATH = \"%s;$env:PATH\"\n", shimsDir))
		builder.WriteString("  } else {\n")
		builder.WriteString(fmt.Sprintf("    $env:PATH = \"$env:PATH;%s\"\n", shimsDir))
		builder.WriteString("  }\n")
		builder.WriteString("}\n")
	case "cmd":
		fmt.Fprintf(&builder, "SET GOENV_SHELL=%s\n", shell)
		fmt.Fprintf(&builder, "SET GOENV_ROOT=%s\n", cfg.Root)
		builder.WriteString("IF NOT DEFINED GOENV_RC_FILE (\n")
		builder.WriteString("  SET GOENV_RC_FILE=%USERPROFILE%\\.goenvrc.cmd\n")
		builder.WriteString(")\n")
		builder.WriteString("IF EXIST \"%GOENV_RC_FILE%\" (\n")
		builder.WriteString("  CALL \"%GOENV_RC_FILE%\"\n")
		builder.WriteString(")\n")
		shimsDir := filepath.Join(cfg.Root, "shims")
		builder.WriteString(fmt.Sprintf("IF \"%%PATH:%s=%%\" == \"%%PATH%%\" (\n", shimsDir))
		builder.WriteString("  IF \"%GOENV_PATH_ORDER%\" == \"front\" (\n")
		builder.WriteString(fmt.Sprintf("    SET PATH=%s;%%PATH%%\n", shimsDir))
		builder.WriteString("  ) ELSE (\n")
		builder.WriteString(fmt.Sprintf("    SET PATH=%%PATH%%;%s\n", shimsDir))
		builder.WriteString("  )\n")
		builder.WriteString(")\n")
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
		// Completion sourcing only for Unix shells
		if shell != "powershell" && shell != "cmd" {
			fmt.Fprintf(&builder, "source '%s'\n", completion)
		}
	}

	if !noRehash {
		// Rehash with shell-appropriate error redirection
		switch shell {
		case "powershell":
			builder.WriteString("goenv rehash 2>$null\n")
		case "cmd":
			builder.WriteString("goenv rehash 2>NUL\n")
		default:
			builder.WriteString("command goenv rehash 2>/dev/null\n")
		}
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
	case "powershell":
		builder.WriteString("function goenv {\n")
		builder.WriteString("  $command = $args[0]\n")
		builder.WriteString("  $restArgs = $args[1..($args.Length)]\n\n")
		builder.WriteString("  switch ($command) {\n")
		for _, cmd := range specialCommands {
			builder.WriteString(fmt.Sprintf("    \"%s\" {\n", cmd))
			builder.WriteString(fmt.Sprintf("      Invoke-Expression (& goenv sh-%s @restArgs | Out-String)\n", cmd))
			builder.WriteString("    }\n")
		}
		builder.WriteString("    default {\n")
		builder.WriteString("      & goenv $command @restArgs\n")
		builder.WriteString("    }\n")
		builder.WriteString("  }\n")
		builder.WriteString("}\n")
	case "cmd":
		// cmd.exe doesn't support functions like bash/PowerShell
		// Users will need to use goenv.bat shim directly
		builder.WriteString("REM cmd.exe does not support functions\n")
		builder.WriteString("REM Use goenv commands directly\n")
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
	Use:                "sh-shell [version]",
	Short:              "Set or show the shell-specific Go version",
	Hidden:             true,
	DisableFlagParsing: true, // Allow --unset as argument not flag
	RunE:               runShShell,
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
