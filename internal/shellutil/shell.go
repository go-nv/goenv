// Package shellutil provides canonical implementations for shell detection,
// profile path resolution, and initialization command generation.
//
// This is the low-level package that other packages should import from rather
// than duplicating shell-related logic. For high-level profile management
// operations (backup, modification, issue detection), see internal/shell/profile.
//
// Architecture:
//   - internal/shellutil: Canonical source for shell detection and basic utilities
//   - internal/shell/profile: High-level profile management built on shellutil
package shellutil

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// ShellType represents the type of shell
type ShellType string

const (
	ShellTypeBash       ShellType = "bash"
	ShellTypeZsh        ShellType = "zsh"
	ShellTypeFish       ShellType = "fish"
	ShellTypePowerShell ShellType = "powershell"
	ShellTypeCmd        ShellType = "cmd"
	ShellTypeKsh        ShellType = "ksh"
	ShellTypeUnknown    ShellType = ""
)

// String returns the string representation of the shell type
func (s ShellType) String() string {
	return string(s)
}

// ParseShellType converts a string to a ShellType
// Returns ShellTypeUnknown for unrecognized shells
func ParseShellType(shell string) ShellType {
	switch shell {
	case "bash":
		return ShellTypeBash
	case "zsh":
		return ShellTypeZsh
	case "fish":
		return ShellTypeFish
	case "powershell":
		return ShellTypePowerShell
	case "cmd":
		return ShellTypeCmd
	case "ksh":
		return ShellTypeKsh
	case "sh":
		return ShellTypeBash // Treat sh as bash
	default:
		return ShellTypeUnknown
	}
}

// DetectShell determines the current shell from environment
func DetectShell() ShellType {
	// Check GOENV_SHELL first (set by goenv init)
	if shell := utils.GoenvEnvVarShell.UnsafeValue(); shell != "" {
		return ShellType(shell)
	}

	// Check SHELL environment variable
	shell := os.Getenv(utils.EnvVarShell)
	if shell != "" {
		// Extract shell name from path
		shellName := filepath.Base(shell)

		// Map shell names to shell types
		switch shellName {
		case "bash":
			return ShellTypeBash
		case "zsh":
			return ShellTypeZsh
		case "fish":
			return ShellTypeFish
		case "ksh":
			return ShellTypeKsh
		}
	}

	// Check for PowerShell on Windows (case-insensitive check)
	if os.Getenv(utils.EnvVarPSModulePath) != "" {
		return ShellTypePowerShell
	}

	// Check for specific shell environment variables
	if os.Getenv(utils.EnvVarZshVersion) != "" {
		return ShellTypeZsh
	}
	if os.Getenv(utils.EnvVarFishVersion) != "" {
		return ShellTypeFish
	}
	if os.Getenv(utils.EnvVarBashVersion) != "" {
		return ShellTypeBash
	}

	// Default to PowerShell on Windows, bash elsewhere
	if utils.IsWindows() || os.Getenv(utils.EnvVarComspec) != "" {
		return ShellTypePowerShell
	}

	// Fallback to bash
	return ShellTypeBash
}

// GetProfilePath returns the shell profile file path for a given shell
// This intelligently chooses between alternatives (e.g., .bashrc vs .bash_profile)
func GetProfilePath(shell ShellType) string {
	home, _ := os.UserHomeDir()

	switch shell {
	case ShellTypeBash:
		// Prefer .bashrc if it exists and .bash_profile doesn't
		// (because .bash_profile overrides .bashrc on login shells)
		bashrc := filepath.Join(home, ".bashrc")
		bashProfile := filepath.Join(home, ".bash_profile")

		// If both exist, prefer .bash_profile (login shell file)
		if utils.FileExists(bashProfile) {
			return bashProfile
		}

		// If only .bashrc exists, use it
		if utils.FileExists(bashrc) {
			return bashrc
		}

		// Neither exists - prefer .bash_profile for new files
		return bashProfile
	case ShellTypeZsh:
		return filepath.Join(home, ".zshrc")
	case ShellTypeFish:
		return filepath.Join(home, ".config", "fish", "config.fish")
	case ShellTypePowerShell:
		// PowerShell profile location
		if profile := os.Getenv(utils.EnvVarProfile); profile != "" {
			return profile
		}
		// Fallback to typical location
		if utils.IsWindows() {
			docs := os.Getenv(utils.EnvVarUserProfile)
			return filepath.Join(docs, "Documents", "WindowsPowerShell", "Microsoft.PowerShell_profile.ps1")
		}
		return "$PROFILE"
	case ShellTypeCmd:
		if utils.IsWindows() {
			userProfile := os.Getenv(utils.EnvVarUserProfile)
			return filepath.Join(userProfile, "autorun.cmd")
		}
		return "%USERPROFILE%\\autorun.cmd"
	default:
		return ""
	}
}

// GetProfilePathDisplay returns a user-friendly display path (may include ~)
func GetProfilePathDisplay(shell ShellType) string {
	switch shell {
	case ShellTypeBash:
		return "~/.bashrc or ~/.bash_profile"
	case ShellTypeZsh:
		return "~/.zshrc"
	case ShellTypeFish:
		return "~/.config/fish/config.fish"
	case ShellTypePowerShell:
		return "$PROFILE"
	case ShellTypeCmd:
		return "%USERPROFILE%\\autorun.cmd"
	default:
		return "your shell profile"
	}
}

// GetInitLine returns the shell-specific command to initialize goenv
func GetInitLine(shell ShellType) string {
	switch shell {
	case ShellTypeFish:
		return "status --is-interactive; and source (goenv init -|psub)"
	case ShellTypePowerShell:
		return "Invoke-Expression (goenv init - | Out-String)"
	case ShellTypeCmd:
		return "FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i"
	default:
		return "eval \"$(goenv init -)\""
	}
}

// HasGoenvInProfile checks if a profile file already contains goenv initialization
func HasGoenvInProfile(profilePath string) (bool, error) {
	content, err := os.ReadFile(profilePath)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, err
	}

	contentStr := string(content)
	// Check for common goenv markers
	return containsAny(contentStr, []string{
		"goenv init",
		utils.GoenvEnvVarRoot.String(),
		"goenv/shims",
	}), nil
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if strings.Contains(s, substr) {
			return true
		}
	}
	return false
}
