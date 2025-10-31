package shell

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var initCmd = &cobra.Command{
	Use:     "init [shell]",
	Short:   "Configure the shell environment for goenv",
	GroupID: string(cmdpkg.GroupShell),
	Long: `Configure the shell environment for goenv.

The shell type is auto-detected from your environment. You can optionally 
specify it explicitly: bash, zsh, fish, ksh, powershell, or cmd.

Usage:
  eval "$(goenv init -)"            # Auto-detects your shell
  
  Or add it to your shell's startup file (.bashrc, .zshrc, etc.)

This sets up:
  - Environment variables (GOENV_SHELL, PATH, etc.)
  - Shell function for easy 'goenv shell' and 'goenv init' usage
  - Completion scripts (if available)

After setup, you can:
  - Run 'goenv init' to check initialization status
  - Run 'goenv shell 1.21.0' to switch versions in current shell
  - Run 'goenv rehash' to rebuild shims

Shell auto-detection:
  - Checks GOENV_SHELL environment variable
  - Detects parent shell process (with -)
  - Falls back to $SHELL or platform defaults

Tip: Use 'goenv setup' for interactive configuration.`,
	RunE: runInit,
}

var initFlags struct {
	noRehash bool
	complete bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(initCmd)
	initCmd.Flags().BoolVar(&initFlags.noRehash, "no-rehash", false, "Skip rehashing shims")
	initCmd.Flags().BoolVar(&initFlags.complete, "complete", false, "Internal flag for shell completions")
	_ = initCmd.Flags().MarkHidden("complete")
}

func runInit(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	if initFlags.complete {
		for _, option := range []string{"-", "--no-rehash", "bash", "fish", "ksh", "zsh", "powershell", "cmd"} {
			fmt.Fprintln(cmd.OutOrStdout(), option)
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

	shell := ResolveShell(explicitShell, dashMode)

	if cfg.Debug {
		fmt.Fprintf(cmd.ErrOrStderr(), "Debug: Initializing for %s shell\n", shell)
	}

	if !dashMode {
		// Check if shell is already initialized
		isInitialized := checkShellInitialized(cfg)

		if isInitialized {
			fmt.Fprintf(cmd.OutOrStdout(), "%s goenv is already initialized in this shell\n\n", utils.Emoji("✅ "))
			fmt.Fprintln(cmd.OutOrStdout(), "Environment variables:")
			fmt.Fprintf(cmd.OutOrStdout(), "  GOENV_SHELL=%s\n", utils.GoenvEnvVarShell.UnsafeValue())
			fmt.Fprintf(cmd.OutOrStdout(), "  GOENV_ROOT=%s\n", cfg.Root)
			fmt.Fprintf(cmd.OutOrStdout(), "  Shims in PATH: yes\n")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%s goenv is NOT initialized in this shell\n\n", utils.Emoji("⚠️  "))
			fmt.Fprintln(cmd.OutOrStdout(), "To initialize goenv in your current shell session, run:")
			fmt.Fprintf(cmd.OutOrStdout(), "  %s\n\n", getEvalCommand(shell))
			fmt.Fprint(cmd.OutOrStdout(), renderUsageSnippet(shell, explicitShell))
		}
		return nil
	}

	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to prepare goenv directories: %w", err)
	}

	fmt.Fprint(cmd.OutOrStdout(), renderInitScript(shell, cfg, initFlags.noRehash))
	return nil
}

func ResolveShell(explicit string, dashMode bool) shellutil.ShellType {
	if explicit != "" {
		return shellutil.ParseShellType(explicit)
	}

	// Check GOENV_SHELL first (used by sh-shell command)
	if goenvShell := utils.GoenvEnvVarShell.UnsafeValue(); goenvShell != "" {
		return shellutil.ParseShellType(goenvShell)
	}

	if dashMode {
		if detected := detectParentShell(); detected != shellutil.ShellTypeUnknown {
			return detected
		}
	}

	return detectEnvShell()
}

func detectEnvShell() shellutil.ShellType {
	// On Windows, detect PowerShell or cmd
	if utils.IsWindows() {
		// Check if running in PowerShell
		if psVersion := os.Getenv(utils.EnvVarPSModulePath); psVersion != "" {
			return shellutil.ShellTypePowerShell
		}
		// Default to cmd on Windows
		return shellutil.ShellTypeCmd
	}

	shellPath := os.Getenv(utils.EnvVarShell)
	if shellPath != "" {
		if base := filepath.Base(shellPath); base != "" {
			return shellutil.ParseShellType(base)
		}
	}
	return shellutil.ShellTypeBash
}

func detectParentShell() shellutil.ShellType {
	// On Windows, ps command doesn't exist - rely on detectEnvShell instead
	if utils.IsWindows() {
		return shellutil.ShellTypeUnknown
	}

	ppid := os.Getppid()
	cmd := exec.Command("ps", "-p", strconv.Itoa(ppid), "-o", "args=")
	output, err := cmd.Output()
	if err != nil {
		return shellutil.ShellTypeUnknown
	}

	line := strings.TrimSpace(string(output))
	if line == "" {
		return shellutil.ShellTypeUnknown
	}

	fields := strings.Fields(line)
	if len(fields) == 0 {
		return shellutil.ShellTypeUnknown
	}

	shell := strings.TrimPrefix(fields[0], "-")
	if shell == "" {
		return shellutil.ShellTypeUnknown
	}

	return shellutil.ParseShellType(filepath.Base(shell))
}

func renderUsageSnippet(shell shellutil.ShellType, originalShell string) string {
	profile := determineProfilePath(shell, originalShell)

	var builder strings.Builder
	builder.WriteString("# Load goenv automatically by appending\n")
	builder.WriteString(fmt.Sprintf("# the following to %s:\n\n", profile))

	switch shell {
	case shellutil.ShellTypeFish:
		builder.WriteString("status --is-interactive; and source (goenv init -|psub)\n")
	case shellutil.ShellTypePowerShell:
		builder.WriteString("Invoke-Expression (goenv init - | Out-String)\n")
	case shellutil.ShellTypeCmd:
		builder.WriteString("FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i\n")
	default:
		builder.WriteString("eval \"$(goenv init -)\"\n")
	}

	builder.WriteString("\n")
	return builder.String()
}

func determineProfilePath(shell shellutil.ShellType, originalShell string) string {
	switch shell {
	case shellutil.ShellTypeBash:
		home, err := os.UserHomeDir()
		if err != nil || home == "" {
			home = os.Getenv(utils.EnvVarHome) // Fallback
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
	case shellutil.ShellTypeZsh:
		return "~/.zshrc"
	case shellutil.ShellTypeKsh:
		return "~/.profile"
	case shellutil.ShellTypeFish:
		return "~/.config/fish/config.fish"
	case shellutil.ShellTypePowerShell:
		return "$PROFILE"
	case shellutil.ShellTypeCmd:
		return "%USERPROFILE%\\autorun.cmd"
	default:
		// Use originalShell if provided, otherwise use the shell type string
		shellName := originalShell
		if shellName == "" {
			shellName = shell.String()
		}
		return fmt.Sprintf("<unknown shell: %s, replace with your profile path>", shellName)
	}
}

func renderInitScript(shell shellutil.ShellType, cfg *config.Config, noRehash bool) string {
	var builder strings.Builder

	switch shell {
	case shellutil.ShellTypeFish:
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
	case shellutil.ShellTypePowerShell:
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
	case shellutil.ShellTypeCmd:
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
		if shell != shellutil.ShellTypePowerShell && shell != shellutil.ShellTypeCmd {
			fmt.Fprintf(&builder, "source '%s'\n", completion)
		}
	}

	if !noRehash {
		// Use sh-rehash with --only-manage-paths for faster initialization
		// This updates GOPATH/GOROOT without rebuilding shims on every shell start
		switch shell {
		case shellutil.ShellTypePowerShell:
			builder.WriteString("goenv sh-rehash --only-manage-paths 2>$null\n")
		case shellutil.ShellTypeCmd:
			builder.WriteString("goenv sh-rehash --only-manage-paths 2>NUL\n")
		default:
			builder.WriteString("eval \"$(command goenv sh-rehash --only-manage-paths 2>/dev/null)\"\n")
		}
	}

	builder.WriteString(renderShellFunction(shell))

	return builder.String()
}

func renderShellFunction(shell shellutil.ShellType) string {
	specialCommands := []string{"rehash", "shell"}

	var builder strings.Builder

	switch shell {
	case shellutil.ShellTypeFish:
		builder.WriteString("function goenv\n")
		builder.WriteString("  set command $argv[1]\n")
		builder.WriteString("  set -e argv[1]\n\n")
		builder.WriteString("  switch \"$command\"\n")
		builder.WriteString("  case init\n")
		builder.WriteString("    # Handle init command with status check\n")
		builder.WriteString("    if test \"$argv[1]\" = \"-\"\n")
		builder.WriteString("      # Output init script\n")
		builder.WriteString("      command goenv init -\n")
		builder.WriteString("    else if test (count $argv) -eq 0\n")
		builder.WriteString("      # No arguments - show initialization status\n")
		builder.WriteString("      if test -n \"$GOENV_SHELL\"\n")
		builder.WriteString("        echo \"✓ goenv is initialized\"\n")
		builder.WriteString("        echo \"  Shell: $GOENV_SHELL\"\n")
		builder.WriteString("        echo \"  Root: \"(test -n \"$GOENV_ROOT\"; and echo $GOENV_ROOT; or echo \"not set\")\n")
		builder.WriteString("        if command -v goenv >/dev/null 2>&1\n")
		builder.WriteString("          set version (command goenv version-name 2>/dev/null; or echo \"none\")\n")
		builder.WriteString("          echo \"  Version: $version\"\n")
		builder.WriteString("        end\n")
		builder.WriteString("      else\n")
		builder.WriteString("        echo \"✗ goenv is not initialized in this shell\" >&2\n")
		builder.WriteString("        echo \"\" >&2\n")
		builder.WriteString("        echo \"To initialize, run:\" >&2\n")
		builder.WriteString("        echo \"  eval (goenv init -)\" >&2\n")
		builder.WriteString("        echo \"\" >&2\n")
		builder.WriteString("        echo \"Or add to your ~/.config/fish/config.fish:\" >&2\n")
		builder.WriteString("        echo \"  eval (goenv init -)\" >&2\n")
		builder.WriteString("        return 1\n")
		builder.WriteString("      end\n")
		builder.WriteString("    else\n")
		builder.WriteString("      # Other arguments (shell name, --help, etc.)\n")
		builder.WriteString("      command goenv init $argv\n")
		builder.WriteString("    end\n")
		builder.WriteString("  case " + strings.Join(specialCommands, " ") + "\n")
		builder.WriteString("    source (goenv \"sh-$command\" $argv|psub)\n")
		builder.WriteString("  case '*'\n")
		builder.WriteString("    command goenv \"$command\" $argv\n")
		builder.WriteString("  end\n")
		builder.WriteString("end\n")
	case shellutil.ShellTypePowerShell:
		builder.WriteString("function goenv {\n")
		builder.WriteString("  $command = $args[0]\n")
		builder.WriteString("  $restArgs = $args[1..($args.Length)]\n\n")
		builder.WriteString("  switch ($command) {\n")
		builder.WriteString("    \"init\" {\n")
		builder.WriteString("      # Handle init command with status check\n")
		builder.WriteString("      if ($restArgs[0] -eq \"-\") {\n")
		builder.WriteString("        # Output init script\n")
		builder.WriteString("        & goenv init -\n")
		builder.WriteString("      } elseif ($restArgs.Count -eq 0 -or ($restArgs.Count -eq 1 -and [string]::IsNullOrEmpty($restArgs[0]))) {\n")
		builder.WriteString("        # No arguments - show initialization status\n")
		builder.WriteString("        if ($env:GOENV_SHELL) {\n")
		builder.WriteString("          Write-Host \"✓ goenv is initialized\"\n")
		builder.WriteString("          Write-Host \"  Shell: $env:GOENV_SHELL\"\n")
		builder.WriteString("          if ($env:GOENV_ROOT) {\n")
		builder.WriteString("            Write-Host \"  Root: $env:GOENV_ROOT\"\n")
		builder.WriteString("          } else {\n")
		builder.WriteString("            Write-Host \"  Root: not set\"\n")
		builder.WriteString("          }\n")
		builder.WriteString("          if (Get-Command goenv -ErrorAction SilentlyContinue) {\n")
		builder.WriteString("            $version = & goenv version-name 2>$null\n")
		builder.WriteString("            if (-not $version) { $version = \"none\" }\n")
		builder.WriteString("            Write-Host \"  Version: $version\"\n")
		builder.WriteString("          }\n")
		builder.WriteString("        } else {\n")
		builder.WriteString("          Write-Host \"✗ goenv is not initialized in this shell\" -ForegroundColor Red\n")
		builder.WriteString("          Write-Host \"\"\n")
		builder.WriteString("          Write-Host \"To initialize, run:\"\n")
		builder.WriteString("          Write-Host '  Invoke-Expression (& goenv init - | Out-String)'\n")
		builder.WriteString("          Write-Host \"\"\n")
		builder.WriteString("          Write-Host \"Or add to your PowerShell profile:\"\n")
		builder.WriteString("          Write-Host '  Invoke-Expression (& goenv init - | Out-String)'\n")
		builder.WriteString("          exit 1\n")
		builder.WriteString("        }\n")
		builder.WriteString("      } else {\n")
		builder.WriteString("        # Other arguments (shell name, --help, etc.)\n")
		builder.WriteString("        & goenv init @restArgs\n")
		builder.WriteString("      }\n")
		builder.WriteString("    }\n")
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
	case shellutil.ShellTypeCmd:
		// cmd.exe doesn't support functions like bash/PowerShell
		// Users will need to use goenv.bat shim directly
		builder.WriteString("REM cmd.exe does not support functions\n")
		builder.WriteString("REM Use goenv commands directly (e.g., goenv shell 1.21.0)\n")
		builder.WriteString("REM Note: 'goenv init' without dash requires PowerShell for auto-apply\n")
	case shellutil.ShellTypeKsh:
		builder.WriteString("function goenv {\n")
		builder.WriteString("  typeset command\n")
		builder.WriteString("  command=\"$1\"\n")
		builder.WriteString("  if [ \"$#\" -gt 0 ]; then\n")
		builder.WriteString("    shift\n")
		builder.WriteString("  fi\n\n")
		builder.WriteString("  case \"$command\" in\n")
		builder.WriteString("  init)\n")
		builder.WriteString("    # Handle init command with status check\n")
		builder.WriteString("    case \"$1\" in\n")
		builder.WriteString("      -)\n")
		builder.WriteString("        command goenv init -;;\n")
		builder.WriteString("      \"\")\n")
		builder.WriteString("        if [ -n \"${GOENV_SHELL:-}\" ]; then\n")
		builder.WriteString("          echo \"✓ goenv is initialized\"\n")
		builder.WriteString("          echo \"  Shell: ${GOENV_SHELL}\"\n")
		builder.WriteString("          echo \"  Root: ${GOENV_ROOT:-not set}\"\n")
		builder.WriteString("          if command -v goenv >/dev/null 2>&1; then\n")
		builder.WriteString("            typeset version\n")
		builder.WriteString("            version=$(command goenv version-name 2>/dev/null || echo \"none\")\n")
		builder.WriteString("            echo \"  Version: ${version}\"\n")
		builder.WriteString("          fi\n")
		builder.WriteString("        else\n")
		builder.WriteString("          echo \"✗ goenv is not initialized in this shell\" >&2\n")
		builder.WriteString("          echo \"\" >&2\n")
		builder.WriteString("          echo \"To initialize, run:\" >&2\n")
		builder.WriteString("          echo \"  eval \\\"\\$(goenv init -)\\\"\" >&2\n")
		builder.WriteString("          return 1\n")
		builder.WriteString("        fi;;\n")
		builder.WriteString("      *)\n")
		builder.WriteString("        command goenv init \"$@\";;\n")
		builder.WriteString("    esac;;\n")
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
		builder.WriteString("  init)\n")
		builder.WriteString("    # Handle init command with status check\n")
		builder.WriteString("    case \"$1\" in\n")
		builder.WriteString("      -)\n")
		builder.WriteString("        # Output init script\n")
		builder.WriteString("        command goenv init -;;\n")
		builder.WriteString("      \"\")\n")
		builder.WriteString("        # No arguments - show initialization status\n")
		builder.WriteString("        if [ -n \"${GOENV_SHELL:-}\" ]; then\n")
		builder.WriteString("          echo \"✓ goenv is initialized\"\n")
		builder.WriteString("          echo \"  Shell: ${GOENV_SHELL}\"\n")
		builder.WriteString("          echo \"  Root: ${GOENV_ROOT:-not set}\"\n")
		builder.WriteString("          if command -v goenv >/dev/null 2>&1; then\n")
		builder.WriteString("            local version\n")
		builder.WriteString("            version=$(command goenv version-name 2>/dev/null || echo \"none\")\n")
		builder.WriteString("            echo \"  Version: ${version}\"\n")
		builder.WriteString("          fi\n")
		builder.WriteString("        else\n")
		builder.WriteString("          echo \"✗ goenv is not initialized in this shell\" >&2\n")
		builder.WriteString("          echo \"\" >&2\n")
		builder.WriteString("          echo \"To initialize, run:\" >&2\n")
		builder.WriteString("          echo \"  eval \\\"\\$(goenv init -)\\\"\" >&2\n")
		builder.WriteString("          echo \"\" >&2\n")
		builder.WriteString("          echo \"Or add to your shell profile (~/.bashrc, ~/.zshrc, etc.):\" >&2\n")
		builder.WriteString("          echo \"  eval \\\"\\$(goenv init -)\\\"\" >&2\n")
		builder.WriteString("          return 1\n")
		builder.WriteString("        fi;;\n")
		builder.WriteString("      *)\n")
		builder.WriteString("        # Other arguments (shell name, --help, etc.)\n")
		builder.WriteString("        command goenv init \"$@\";;\n")
		builder.WriteString("    esac;;\n")
		builder.WriteString("  " + strings.Join(specialCommands, "|") + ")\n")
		builder.WriteString("    eval \"$(goenv \"sh-$command\" \"$@\")\";;\n")
		builder.WriteString("  *)\n")
		builder.WriteString("    command goenv \"$command\" \"$@\";;\n")
		builder.WriteString("  esac\n")
		builder.WriteString("}\n")
	}

	return builder.String()
}

func findCompletionPath(shell shellutil.ShellType) string {
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
		// Check environment variable override first
		if envRoot := utils.GoenvEnvVarInstallRoot.UnsafeValue(); envRoot != "" {
			installRoot = envRoot
			return
		}

		// Use executable location to find install root
		// This works correctly for distributed binaries
		if exe, err := os.Executable(); err == nil {
			// Binary is typically in <root>/bin/goenv
			// So install root is one level up from the binary directory
			dir := filepath.Dir(exe)
			candidate := filepath.Clean(filepath.Join(dir, ".."))

			// Verify this looks like a goenv installation by checking for completions
			if _, err := os.Stat(filepath.Join(candidate, "completions")); err == nil {
				installRoot = candidate
				return
			}
		}

		// No install root found - completions won't be available
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
	cmdpkg.RootCmd.AddCommand(shShellCmd)
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
	shell := ResolveShell("", true)

	// No arguments - print current GOENV_VERSION
	if len(args) == 0 {
		currentVersion := utils.GoenvEnvVarVersion.UnsafeValue()
		if currentVersion == "" {
			return fmt.Errorf("goenv: no shell-specific version configured")
		}
		fmt.Fprintln(cmd.OutOrStdout(), `echo "$GOENV_VERSION"`)
		return nil
	}

	// Handle --unset flag
	if args[0] == "--unset" {
		if shell == shellutil.ShellTypeFish {
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
	if shell == shellutil.ShellTypeFish {
		fmt.Fprintf(cmd.OutOrStdout(), "set -gx GOENV_VERSION \"%s\"\n", versionStr)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "export GOENV_VERSION=\"%s\"\n", versionStr)
	}

	return nil
}

// checkShellInitialized checks if the current shell has goenv properly initialized
func checkShellInitialized(cfg *config.Config) bool {
	// Check if GOENV_SHELL is set
	if utils.GoenvEnvVarShell.UnsafeValue() == "" {
		return false
	}

	// Check if shims directory is in PATH
	shimsDir := filepath.Join(cfg.Root, "shims")
	path := os.Getenv(utils.EnvVarPath)

	// Platform-specific path separator
	pathSep := ":"
	if utils.IsWindows() {
		pathSep = ";"
	}

	pathDirs := strings.Split(path, pathSep)
	for _, dir := range pathDirs {
		if dir == shimsDir {
			return true
		}
	}

	return false
}

// getEvalCommand returns the shell-specific command to initialize goenv
func getEvalCommand(shell shellutil.ShellType) string {
	switch shell {
	case shellutil.ShellTypeFish:
		return "source (goenv init -|psub)"
	case shellutil.ShellTypePowerShell:
		return "Invoke-Expression (goenv init - | Out-String)"
	case shellutil.ShellTypeCmd:
		return "FOR /f \"tokens=*\" %i IN ('goenv init -') DO @%i"
	default:
		return "eval \"$(goenv init -)\""
	}
}
