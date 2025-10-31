package utils

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// IsFirstRun checks if this appears to be the first time goenv is being used
// Returns true if:
// 1. GOENV_ROOT/versions directory doesn't exist or is empty (no versions installed)
// 2. GOENV_SHELL is not set (not initialized in shell)
func IsFirstRun(goenvRoot string) bool {
	// Check if versions directory exists and has contents
	versionsDir := filepath.Join(goenvRoot, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil || len(entries) == 0 {
		// No versions directory or it's empty
		// Check if shell is initialized
		goenvShell := GoenvEnvVarShell.UnsafeValue()
		if goenvShell == "" {
			return true
		}
	}
	return false
}

// ShowFirstRunGuidance displays helpful guidance for first-time users
// Returns true if guidance was shown (indicating first run)
func ShowFirstRunGuidance(w io.Writer, goenvRoot string) bool {
	if !IsFirstRun(goenvRoot) {
		return false
	}

	fmt.Fprintf(w, "%s%s\n\n", Emoji("👋 "), BoldBlue("Welcome to goenv!"))
	fmt.Fprintf(w, "It looks like this is your first time using goenv.\n\n")

	// Check which issue we have
	hasVersions := HasAnyVersionsInstalled(goenvRoot)
	shellInit := IsShellInitialized()

	if !shellInit {
		fmt.Fprintf(w, "%s\n", BoldYellow("Step 1: Initialize goenv in your shell"))
		fmt.Fprintf(w, "To get started, add goenv to your shell by running:\n\n")

		// Provide shell-specific instructions
		shell := detectShell()
		switch shell {
		case "bash":
			fmt.Fprintf(w, "  %s\n", Cyan("echo 'eval \"$(goenv init -)\"' >> ~/.bashrc"))
			fmt.Fprintf(w, "  %s\n\n", Cyan("source ~/.bashrc"))
		case "zsh":
			fmt.Fprintf(w, "  %s\n", Cyan("echo 'eval \"$(goenv init -)\"' >> ~/.zshrc"))
			fmt.Fprintf(w, "  %s\n\n", Cyan("source ~/.zshrc"))
		case "fish":
			fmt.Fprintf(w, "  %s\n", Cyan("echo 'status --is-interactive; and goenv init - | source' >> ~/.config/fish/config.fish"))
			fmt.Fprintf(w, "  %s\n\n", Cyan("source ~/.config/fish/config.fish"))
		case "powershell":
			fmt.Fprintf(w, "  %s\n\n", Cyan("Add-Content $PROFILE 'goenv init - | Invoke-Expression'"))
		default:
			fmt.Fprintf(w, "  %s\n\n", Cyan("eval \"$(goenv init -)\""))
		}

		fmt.Fprintf(w, "Or for just this session:\n")
		fmt.Fprintf(w, "  %s\n\n", Cyan("eval \"$(goenv init -)\""))
	}

	if !hasVersions {
		fmt.Fprintf(w, "%s\n", BoldYellow("Step 2: Install a Go version"))
		fmt.Fprintf(w, "Install your first Go version:\n\n")
		fmt.Fprintf(w, "  %s        %s Install latest stable Go\n", Cyan("goenv install"), Gray("→"))
		fmt.Fprintf(w, "  %s %s Install specific version\n", Cyan("goenv install 1.21.5"), Gray("→"))
		fmt.Fprintf(w, "  %s      %s List all available versions\n\n", Cyan("goenv install -l"), Gray("→"))

		fmt.Fprintf(w, "%s\n", BoldYellow("Step 3: Set your default version"))
		fmt.Fprintf(w, "After installing:\n\n")
		fmt.Fprintf(w, "  %s\n\n", Cyan("goenv global <version>"))
	}

	fmt.Fprintf(w, "%s\n", Gray("For more help, run: goenv --help or goenv doctor"))
	fmt.Fprintln(w)

	return true
}

// detectShell attempts to detect the current shell
func detectShell() string {
	// Check SHELL environment variable
	shell := os.Getenv(EnvVarShell)
	if shell != "" {
		if filepath.Base(shell) == "bash" {
			return "bash"
		}
		if filepath.Base(shell) == "zsh" {
			return "zsh"
		}
		if filepath.Base(shell) == "fish" {
			return "fish"
		}
	}

	// Check for PowerShell on Windows
	if IsWindows() {
		return "powershell"
	}

	// Default to bash
	return "bash"
}

// IsShellInitialized checks if goenv is initialized in the current shell
func IsShellInitialized() bool {
	return GoenvEnvVarShell.UnsafeValue() != ""
}

// HasAnyVersionsInstalled checks if any Go versions are installed
func HasAnyVersionsInstalled(goenvRoot string) bool {
	versionsDir := filepath.Join(goenvRoot, "versions")
	entries, err := os.ReadDir(versionsDir)
	if err != nil {
		return false
	}

	// Check for at least one directory (version folder)
	for _, entry := range entries {
		if entry.IsDir() {
			return true
		}
	}
	return false
}
