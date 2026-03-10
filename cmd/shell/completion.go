package shell

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/completions"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate shell completion script",
	Long: `Generate shell completion script for goenv.

If no shell is specified, the current shell is auto-detected.

Examples:
  # Auto-detect shell and output completion
  goenv completion

  # Auto-detect and install
  goenv completion --install

  # Output bash completion to stdout
  goenv completion bash

  # Install bash completion
  goenv completion bash --install
  # or manually:
  goenv completion bash >> ~/.bashrc

  # Install zsh completion
  goenv completion zsh > "${fpath[1]}/_goenv"

  # Install fish completion
  goenv completion fish > ~/.config/fish/completions/goenv.fish

  # Install PowerShell completion
  goenv completion powershell >> $PROFILE
`,
	ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
	Args:      cobra.MaximumNArgs(1),
	RunE:      runCompletion,
}

var completionFlags struct {
	install bool
}

func init() {
	cmdpkg.RootCmd.AddCommand(completionCmd)
	completionCmd.Flags().BoolVarP(&completionFlags.install, "install", "i", false, "Install completion script automatically")
}

func runCompletion(cmd *cobra.Command, args []string) error {
	// Auto-detect shell if not specified
	var shell shellutil.ShellType
	if len(args) == 0 {
		shell = shellutil.DetectShell()
		fmt.Fprintf(cmd.OutOrStderr(), "Auto-detected shell: %s\n\n", shell)
	} else {
		shell = shellutil.ParseShellType(args[0])
	}

	var script string
	var installPath string
	var installInstructions string

	switch shell {
	case shellutil.ShellTypeBash:
		script = completions.Bash
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.FailedTo("determine home directory", err)
		}
		installPath = filepath.Join(home, ".bashrc")
		installInstructions = `
Bash completion script generated.

To install manually:
  goenv completion bash >> ~/.bashrc
  source ~/.bashrc

Or install system-wide (requires sudo):
  sudo goenv completion bash > /etc/bash_completion.d/goenv
`

	case shellutil.ShellTypeZsh:
		script = completions.Zsh
		// Try to find zsh fpath
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.FailedTo("determine home directory", err)
		}
		// Use user's home directory for zsh completions (cross-platform)
		fpath := filepath.Join(home, ".zsh", "completions")
		installPath = filepath.Join(fpath, "_goenv")
		installInstructions = `
Zsh completion script generated.

To install manually:
  mkdir -p ~/.zsh/completions
  goenv completion zsh > ~/.zsh/completions/_goenv
  
  # Add to ~/.zshrc if not already there:
  fpath=(~/.zsh/completions $fpath)
  autoload -U compinit && compinit
`

	case shellutil.ShellTypeFish:
		script = completions.Fish
		home, err := os.UserHomeDir()
		if err != nil {
			return errors.FailedTo("determine home directory", err)
		}
		installPath = filepath.Join(home, ".config", "fish", "completions", "goenv.fish")
		installInstructions = `
Fish completion script generated.

To install:
  mkdir -p ~/.config/fish/completions
  goenv completion fish > ~/.config/fish/completions/goenv.fish
`

	case shellutil.ShellTypePowerShell:
		script = completions.PowerShell
		installInstructions = `
PowerShell completion script generated.

To install:
  # Add to your PowerShell profile
  goenv completion powershell >> $PROFILE
  
  # Then reload
  . $PROFILE

To find your profile location:
  $PROFILE
`

	default:
		return fmt.Errorf("unsupported shell: %s (supported: bash, zsh, fish, powershell)", shell)
	}

	// If --install flag is set, try to install automatically
	if completionFlags.install {
		return installCompletion(shell, script, installPath)
	}

	// Otherwise, just output the script
	fmt.Fprint(cmd.OutOrStdout(), script)

	// Show instructions to stderr so they don't interfere with piping
	if !completionFlags.install {
		fmt.Fprint(cmd.OutOrStderr(), installInstructions)
	}

	return nil
}

func installCompletion(shell shellutil.ShellType, script, installPath string) error {
	switch shell {
	case shellutil.ShellTypeBash:
		// Append to .bashrc
		return appendToFile(installPath, "\n# goenv shell completion\n"+script)

	case shellutil.ShellTypeZsh:
		// Create directory and write file
		dir := filepath.Dir(installPath)
		if err := utils.EnsureDirWithContext(dir, "create directory"); err != nil {
			return err
		}
		return utils.WriteFileWithContext(installPath, []byte(script), utils.PermFileDefault, "write file")

	case shellutil.ShellTypeFish:
		// Create directory and write file
		dir := filepath.Dir(installPath)
		if err := utils.EnsureDirWithContext(dir, "create directory"); err != nil {
			return err
		}
		return utils.WriteFileWithContext(installPath, []byte(script), utils.PermFileDefault, "write file")

	case shellutil.ShellTypePowerShell:
		// Would need to find $PROFILE on Windows
		return fmt.Errorf("automatic installation not supported for PowerShell, use: goenv completion powershell >> $PROFILE")

	default:
		return fmt.Errorf("unsupported shell: %s", shell)
	}
}

func appendToFile(path, content string) error {
	// Check if content already exists
	if data, err := os.ReadFile(path); err == nil {
		if strings.Contains(string(data), "goenv shell completion") {
			return fmt.Errorf("completion already installed in %s", path)
		}
	}

	// Append content
	f, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, utils.PermFileDefault)
	if err != nil {
		return errors.FailedTo("open file", err)
	}
	defer f.Close()

	if _, err := f.WriteString(content); err != nil {
		return errors.FailedTo("write completion", err)
	}

	fmt.Printf("%sCompletion installed to %s\n", utils.Emoji("✅ "), path)
	fmt.Println("Run 'source " + path + "' to activate")
	return nil
}
