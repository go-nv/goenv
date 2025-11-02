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

		// Provide shell-specific instructions (inlined to avoid import cycles)
		shell := detectShellSimple()
		initLine := getInitLineSimple(shell)
		profileDisplay := getProfilePathDisplaySimple(shell)

		// Show simple instructions
		fmt.Fprintf(w, "  Add to %s:\n", profileDisplay)
		fmt.Fprintf(w, "    %s\n\n", Cyan(initLine))

		fmt.Fprintf(w, "Or for just this session:\n")
		fmt.Fprintf(w, "  %s\n\n", Cyan(initLine))
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

// detectShellSimple is a simplified shell detection for first-run guidance only
// For full shell detection, use shellutil.DetectShell()
func detectShellSimple() string {
	// Check GOENV_SHELL first (if already initialized)
	if shell := GoenvEnvVarShell.UnsafeValue(); shell != "" {
		return shell
	}

	// Check SHELL environment variable
	shell := os.Getenv(EnvVarShell)
	if shell == "" {
		return "bash" // default fallback
	}

	// Extract shell name from path
	if idx := len(shell) - 1; idx >= 0 {
		for i := idx; i >= 0; i-- {
			if shell[i] == '/' || shell[i] == '\\' {
				return shell[i+1:]
			}
		}
	}
	return shell
}

// getInitLineSimple returns the init line for a shell (simplified version)
func getInitLineSimple(shell string) string {
	switch shell {
	case "fish":
		return "status --is-interactive; and source (goenv init -|psub)"
	case "powershell", "pwsh":
		return "Invoke-Expression (goenv init - | Out-String)"
	case "cmd":
		return "FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i"
	default:
		return "eval \"$(goenv init -)\""
	}
}

// getProfilePathDisplaySimple returns a display name for the profile file
func getProfilePathDisplaySimple(shell string) string {
	switch shell {
	case "bash":
		return "~/.bashrc or ~/.bash_profile"
	case "zsh":
		return "~/.zshrc"
	case "fish":
		return "~/.config/fish/config.fish"
	case "powershell", "pwsh":
		return "$PROFILE"
	case "cmd":
		return "%USERPROFILE%\\autorun.cmd"
	default:
		return "your shell profile"
	}
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
