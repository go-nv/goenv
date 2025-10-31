package shell

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var setupCmd = &cobra.Command{
	Use:     "setup",
	Short:   "Automatic first-time setup for goenv",
	GroupID: string(cmdpkg.GroupGettingStarted),
	Long: `Automatically configure goenv for first-time use.

This command will:
  - Detect your shell (bash, zsh, fish, PowerShell, cmd)
  - Add goenv initialization to your shell profile
  - Detect VS Code and offer to configure it
  - Create necessary directories
  - Show you what was done

This is safe to run multiple times - it won't duplicate configuration.

Examples:
  goenv setup              # Interactive setup with prompts
  goenv setup --yes        # Auto-accept all prompts
  goenv setup --verify     # Run doctor checks after setup
  goenv setup --shell zsh  # Force specific shell`,
	RunE: runSetup,
}

var setupFlags struct {
	yes            bool
	shell          string
	skipVSCode     bool
	skipShell      bool
	dryRun         bool
	nonInteractive bool
	verify         bool
}

// setupStdin can be overridden in tests
var setupStdin io.Reader = os.Stdin

func init() {
	cmdpkg.RootCmd.AddCommand(setupCmd)
	setupCmd.Flags().BoolVarP(&setupFlags.yes, "yes", "y", false, "Auto-accept all prompts")
	setupCmd.Flags().StringVar(&setupFlags.shell, "shell", "", "Force specific shell (bash, zsh, fish, powershell, cmd)")
	setupCmd.Flags().BoolVar(&setupFlags.skipVSCode, "skip-vscode", false, "Skip VS Code integration setup")
	setupCmd.Flags().BoolVar(&setupFlags.skipShell, "skip-shell", false, "Skip shell profile setup")
	setupCmd.Flags().BoolVar(&setupFlags.dryRun, "dry-run", false, "Show what would be done without making changes")
	setupCmd.Flags().BoolVar(&setupFlags.nonInteractive, "non-interactive", false, "Disable all interactive prompts")
	setupCmd.Flags().BoolVar(&setupFlags.verify, "verify", false, "Run doctor checks after setup to verify configuration")
}

func runSetup(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	fmt.Fprintf(cmd.OutOrStdout(), "%sWelcome to goenv setup!\n", utils.Emoji("🚀 "))
	fmt.Fprintln(cmd.OutOrStdout())

	changes := []string{}

	// Ensure directories exist
	if err := cfg.EnsureDirectories(); err != nil {
		return fmt.Errorf("failed to create goenv directories: %w", err)
	}

	// Step 1: Shell profile setup
	if !setupFlags.skipShell {
		shellChanges, err := setupShellProfile(cmd, cfg)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%sWarning: Shell setup failed: %v\n", utils.Emoji("⚠️  "), err)
		} else {
			changes = append(changes, shellChanges...)
		}
	}

	// Step 2: VS Code setup
	if !setupFlags.skipVSCode {
		vscodeChanges, err := setupVSCode(cmd, cfg)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "%sWarning: VS Code setup failed: %v\n", utils.Emoji("⚠️  "), err)
		} else {
			changes = append(changes, vscodeChanges...)
		}
	}

	// Summary
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintf(cmd.OutOrStdout(), "%sSetup Summary\n", utils.Emoji("📋 "))
	fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	fmt.Fprintln(cmd.OutOrStdout())

	if len(changes) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "%sNo changes needed - goenv is already configured!\n", utils.Emoji("✅ "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%sChanges made:\n", utils.Emoji("✅ "))
		for _, change := range changes {
			fmt.Fprintf(cmd.OutOrStdout(), "  • %s\n", change)
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())

	if !setupFlags.dryRun && !setupFlags.skipShell && len(changes) > 0 {
		// Check if shell profile was modified
		shellModified := false
		for _, change := range changes {
			if strings.Contains(change, "shell profile") || strings.Contains(change, "Added goenv") {
				shellModified = true
				break
			}
		}

		if shellModified {
			fmt.Fprintf(cmd.OutOrStdout(), "%sNext steps:\n", utils.Emoji("🎯 "))
			fmt.Fprintln(cmd.OutOrStdout(), "  1. Restart your shell or run:")

			var shell shellutil.ShellType
			if setupFlags.shell != "" {
				shell = shellutil.ShellType(setupFlags.shell)
			} else {
				shell = shellutil.DetectShell()
			}
			profilePath := shellutil.GetProfilePathDisplay(shell)

			switch shell {
			case shellutil.ShellTypePowerShell:
				fmt.Fprintf(cmd.OutOrStdout(), "     . %s\n", profilePath)
			case shellutil.ShellTypeCmd:
				fmt.Fprintf(cmd.OutOrStdout(), "     (Restart your command prompt)\n")
			case shellutil.ShellTypeFish:
				fmt.Fprintf(cmd.OutOrStdout(), "     source %s\n", profilePath)
			default:
				fmt.Fprintf(cmd.OutOrStdout(), "     source %s\n", profilePath)
			}

			fmt.Fprintln(cmd.OutOrStdout(), "  2. Install a Go version:")
			fmt.Fprintln(cmd.OutOrStdout(), "     goenv install 1.23.2")
			fmt.Fprintln(cmd.OutOrStdout(), "  3. Set it as your default:")
			fmt.Fprintln(cmd.OutOrStdout(), "     goenv global 1.23.2")

			// Pause if --verify will run, to prevent doctor output from burying these commands
			if setupFlags.verify && !setupFlags.dryRun {
				reader := bufio.NewReader(setupStdin)
				utils.PauseForUser(cmd.OutOrStdout(), reader)
			}
		}
	}

	fmt.Fprintln(cmd.OutOrStdout())

	// Run verification if requested
	if setupFlags.verify && !setupFlags.dryRun {
		fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintf(cmd.OutOrStdout(), "%sVerifying Setup\n", utils.Emoji("🔍 "))
		fmt.Fprintln(cmd.OutOrStdout(), "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
		fmt.Fprintln(cmd.OutOrStdout())

		// Run doctor command as subprocess to avoid os.Exit() interference
		// Note: Since we just modified the profile, the current shell
		// environment may not have goenv initialized yet. The doctor
		// checks will detect this and provide appropriate guidance.
		goenvBinary, err := os.Executable()
		if err != nil {
			// Fallback to ./goenv
			goenvBinary = "./goenv"
		}

		doctorCmd := exec.Command(goenvBinary, "doctor")
		doctorCmd.Stdout = cmd.OutOrStdout()
		doctorCmd.Stderr = cmd.ErrOrStderr()

		// Run doctor and capture exit code (but don't fail setup)
		_ = doctorCmd.Run()

		// Print newline for spacing
		fmt.Fprintln(cmd.OutOrStdout())

		// If we just made shell changes, remind user they may need to restart
		if len(changes) > 0 {
			shellModified := false
			for _, change := range changes {
				if strings.Contains(change, "shell profile") || strings.Contains(change, "Added goenv") {
					shellModified = true
					break
				}
			}

			if shellModified {
				fmt.Fprintf(cmd.OutOrStdout(), "%sNote: Shell environment checks may report issues until you restart your shell:\n", utils.Emoji("💡 "))

				var shell shellutil.ShellType
				if setupFlags.shell != "" {
					shell = shellutil.ShellType(setupFlags.shell)
				} else {
					shell = shellutil.DetectShell()
				}

				switch shell {
				case shellutil.ShellTypeCmd:
					fmt.Fprintln(cmd.OutOrStdout(), "  (Restart your command prompt)")
				default:
					fmt.Fprintf(cmd.OutOrStdout(), "  exec %s\n", shell)
				}
				fmt.Fprintln(cmd.OutOrStdout(), "  Then run: goenv doctor --fix")
				fmt.Fprintln(cmd.OutOrStdout())
			}
		}
	}

	if setupFlags.verify {
		fmt.Fprintf(cmd.OutOrStdout(), "%sDone!\n", utils.Emoji("🎉 "))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%sDone! Run 'goenv doctor' to verify your setup.\n", utils.Emoji("🎉 "))
	}

	return nil
}

func setupShellProfile(cmd *cobra.Command, cfg *config.Config) ([]string, error) {
	changes := []string{}

	fmt.Fprintf(cmd.OutOrStdout(), "%sConfiguring shell integration...\n", utils.Emoji("🐚 "))

	// Detect shell
	var shell shellutil.ShellType
	if setupFlags.shell != "" {
		shell = shellutil.ShellType(setupFlags.shell)
	} else {
		shell = shellutil.DetectShell()
	}
	fmt.Fprintf(cmd.OutOrStdout(), "  Detected shell: %s\n", shell)

	// Get profile path
	profilePath := shellutil.GetProfilePath(shell)
	if profilePath == "" {
		return changes, fmt.Errorf("unsupported shell: %s", shell)
	}

	profileDisplay := shellutil.GetProfilePathDisplay(shell)
	fmt.Fprintf(cmd.OutOrStdout(), "  Profile file: %s\n", profileDisplay)

	// Check if already configured
	alreadyConfigured, err := shellutil.HasGoenvInProfile(profilePath)
	if err == nil && alreadyConfigured {
		fmt.Fprintf(cmd.OutOrStdout(), "  %sAlready configured\n", utils.Emoji("✓ "))
		return changes, nil
	}

	// Check if profile exists
	profileExists := false
	if _, err := os.Stat(profilePath); err == nil {
		profileExists = true
	}

	// Build the init line
	initLine := shellutil.GetInitLine(shell)

	// Prompt user
	if !setupFlags.yes && !setupFlags.dryRun && !setupFlags.nonInteractive {
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "  Add this to %s? [Y/n]: ", profileDisplay)
		fmt.Fprintf(cmd.OutOrStdout(), "\n    %s\n", initLine)
		fmt.Fprint(cmd.OutOrStdout(), "  ")

		reader := bufio.NewReader(setupStdin)
		response, err := reader.ReadString('\n')
		if err != nil {
			return changes, err
		}

		response = strings.TrimSpace(strings.ToLower(response))
		if response != "" && response != "y" && response != "yes" {
			fmt.Fprintf(cmd.OutOrStdout(), "  Skipped\n")
			return changes, nil
		}
	}

	if setupFlags.dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s[DRY RUN] Would add to %s\n", utils.Emoji("ℹ️  "), profileDisplay)
		changes = append(changes, fmt.Sprintf("[DRY RUN] Would add goenv init to %s", profileDisplay))
		return changes, nil
	}

	// Create backup
	if profileExists {
		backupPath := profilePath + ".goenv-backup." + time.Now().Format("20060102-150405")
		if err := utils.CopyFile(profilePath, backupPath); err != nil {
			return changes, fmt.Errorf("failed to backup profile: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "  Created backup: %s\n", filepath.Base(backupPath))
		changes = append(changes, fmt.Sprintf("Backed up %s", profileDisplay))
	}

	// Add init line
	// Use 0600 permissions for shell config files (user read/write only)
	file, err := os.OpenFile(profilePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return changes, fmt.Errorf("failed to open profile: %w", err)
	}
	defer file.Close()

	// Add some spacing if file already has content
	if profileExists {
		fmt.Fprintln(file)
	}

	// Write the init block
	fmt.Fprintln(file, "# goenv initialization")
	fmt.Fprintln(file, initLine)

	fmt.Fprintf(cmd.OutOrStdout(), "  %sAdded goenv initialization\n", utils.Emoji("✓ "))
	changes = append(changes, fmt.Sprintf("Added goenv init to %s", profileDisplay))

	return changes, nil
}

func setupVSCode(cmd *cobra.Command, cfg *config.Config) ([]string, error) {
	changes := []string{}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sChecking for VS Code...\n", utils.Emoji("💻 "))

	// Check if VS Code is installed
	cwd, err := os.Getwd()
	if err != nil {
		return changes, nil // Not critical
	}

	vscodeDir := filepath.Join(cwd, ".vscode")
	settingsFile := filepath.Join(vscodeDir, "settings.json")

	// Check if .vscode exists
	if _, err := os.Stat(vscodeDir); os.IsNotExist(err) {
		fmt.Fprintf(cmd.OutOrStdout(), "  No .vscode directory in current folder\n")

		// Ask if they want to set up VS Code for current directory
		if !setupFlags.yes && !setupFlags.dryRun && !setupFlags.nonInteractive {
			fmt.Fprint(cmd.OutOrStdout(), "  Set up VS Code integration for this directory? [y/N]: ")

			reader := bufio.NewReader(setupStdin)
			response, err := reader.ReadString('\n')
			if err != nil {
				return changes, nil
			}

			response = strings.TrimSpace(strings.ToLower(response))
			if response != "y" && response != "yes" {
				fmt.Fprintf(cmd.OutOrStdout(), "  Skipped\n")
				return changes, nil
			}
		} else if !setupFlags.yes {
			// In dry-run or non-interactive, skip
			return changes, nil
		}
	}

	// Check if settings already exist
	if _, err := os.Stat(settingsFile); err == nil {
		content, readErr := os.ReadFile(settingsFile)
		if readErr == nil && strings.Contains(string(content), "go.goroot") {
			fmt.Fprintf(cmd.OutOrStdout(), "  %sVS Code already configured\n", utils.Emoji("✓ "))
			return changes, nil
		}
	}

	if setupFlags.dryRun {
		fmt.Fprintf(cmd.OutOrStdout(), "  %s[DRY RUN] Would configure VS Code\n", utils.Emoji("ℹ️  "))
		changes = append(changes, "[DRY RUN] Would configure VS Code settings")
		return changes, nil
	}

	// Create .vscode directory if needed
	if err := os.MkdirAll(vscodeDir, 0755); err != nil {
		return changes, fmt.Errorf("failed to create .vscode directory: %w", err)
	}

	// Get current Go version if set
	currentVersion := utils.GoenvEnvVarVersion.UnsafeValue()
	if currentVersion == "" {
		// Try to read from .go-version
		goVersionFile := filepath.Join(cwd, ".go-version")
		if content, err := os.ReadFile(goVersionFile); err == nil {
			currentVersion = strings.TrimSpace(string(content))
		}
	}

	// Create VS Code settings
	settings := map[string]interface{}{
		"go.goroot":      "${env:GOROOT}",
		"go.gopath":      "${env:GOPATH}",
		"go.toolsGopath": filepath.Join(cfg.Root, "tools"),
	}

	// Write settings file
	if err := writeVSCodeSettings(settingsFile, settings); err != nil {
		return changes, fmt.Errorf("failed to configure VS Code: %w", err)
	}

	if currentVersion != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "  %sConfigured VS Code (current version: Go %s)\n", utils.Emoji("✓ "), currentVersion)
		changes = append(changes, fmt.Sprintf("Configured VS Code for Go %s", currentVersion))
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "  %sConfigured VS Code to use environment variables\n", utils.Emoji("✓ "))
		changes = append(changes, "Configured VS Code to use environment variables")
	}

	return changes, nil
}

func writeVSCodeSettings(settingsFile string, newSettings map[string]interface{}) error {
	// Read existing settings if any
	existingSettings := make(map[string]interface{})
	if data, err := os.ReadFile(settingsFile); err == nil {
		// Try to parse existing JSON, but continue even if it's malformed
		// This allows fixing corrupted settings files
		if jsonErr := json.Unmarshal(data, &existingSettings); jsonErr != nil {
			// Log warning but continue with empty settings (will overwrite)
			fmt.Fprintf(os.Stderr, "Warning: existing settings.json is invalid JSON, will be overwritten\n")
			existingSettings = make(map[string]interface{})
		}
	}

	// Merge settings
	for k, v := range newSettings {
		existingSettings[k] = v
	}

	// Write back
	data, err := json.MarshalIndent(existingSettings, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(settingsFile, data, 0644)
}
