// Package profile provides high-level shell profile management operations.
//
// This package builds on top of internal/shellutil, which provides the canonical
// implementations for shell detection, profile path resolution, and initialization
// command generation. The ProfileManager in this package adds advanced functionality
// like backup creation, issue detection, and profile modification.
//
// Architecture:
//   - internal/shellutil: Low-level shell detection and utilities (canonical source)
//   - internal/shell/profile: High-level profile management operations
package profile

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
)

// ShellType is an alias for shellutil.ShellType for backward compatibility
type ShellType = shellutil.ShellType

// Shell type constants (re-exported from shellutil for convenience)
const (
	ShellTypeBash       = shellutil.ShellTypeBash
	ShellTypeZsh        = shellutil.ShellTypeZsh
	ShellTypeFish       = shellutil.ShellTypeFish
	ShellTypePowerShell = shellutil.ShellTypePowerShell
	ShellTypeCmd        = shellutil.ShellTypeCmd
	ShellTypeKsh        = shellutil.ShellTypeKsh
)

// Profile represents a shell profile file
type Profile struct {
	Path         string
	Shell        ShellType
	Exists       bool
	HasGoenv     bool
	Content      string
	LastModified time.Time
}

// ProfileIssue represents a detected issue with profile configuration
type ProfileIssue struct {
	Type        IssueType
	Severity    Severity
	File        string
	Line        int
	Description string
	Suggestion  string
}

// IssueType categorizes different types of profile issues
type IssueType string

const (
	IssueTypePathReset   IssueType = "path_reset"
	IssueTypeConflict    IssueType = "conflict"
	IssueTypeDuplicate   IssueType = "duplicate"
	IssueTypeMissingInit IssueType = "missing_init"
	IssueTypeWrongOrder  IssueType = "wrong_order"
	IssueTypePermission  IssueType = "permission"
)

// Severity indicates how serious an issue is
type Severity string

const (
	SeverityError   Severity = "error"
	SeverityWarning Severity = "warning"
	SeverityInfo    Severity = "info"
)

// ProfileManager handles shell profile detection and modification
type ProfileManager struct {
	shell ShellType
}

// NewProfileManager creates a new profile manager for the given shell
func NewProfileManager(shell ShellType) *ProfileManager {
	return &ProfileManager{
		shell: shell,
	}
}

// NewProfileManagerForCurrentShell creates a profile manager for the detected shell
func NewProfileManagerForCurrentShell() *ProfileManager {
	return &ProfileManager{
		shell: shellutil.DetectShell(),
	}
}

// GetProfile returns detailed information about a shell profile
func (pm *ProfileManager) GetProfile() (*Profile, error) {
	profilePath := pm.getProfilePath()

	profile := &Profile{
		Path:   profilePath,
		Shell:  pm.shell,
		Exists: false,
	}

	// Check if file exists and read content
	info, exists, err := utils.StatWithExistence(profilePath)
	if !exists {
		return profile, nil
	}
	if err != nil {
		return nil, fmt.Errorf("failed to stat profile %s: %w", profilePath, err)
	}

	profile.Exists = true
	profile.LastModified = info.ModTime()

	content, err := os.ReadFile(profilePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read profile %s: %w", profilePath, err)
	}

	profile.Content = string(content)
	profile.HasGoenv = pm.hasGoenvInit(profile.Content)

	return profile, nil
}

// GetAllProfiles returns all potential profile files for the current shell
// For example, zsh checks .zshrc, .zprofile, and .zshenv
func (pm *ProfileManager) GetAllProfiles() ([]*Profile, error) {
	paths := pm.getAllProfilePaths()
	profiles := make([]*Profile, 0, len(paths))

	for _, path := range paths {
		profile := &Profile{
			Path:   path,
			Shell:  pm.shell,
			Exists: false,
		}

		info, exists, err := utils.StatWithExistence(path)
		if !exists {
			profiles = append(profiles, profile)
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("failed to stat profile %s: %w", path, err)
		}

		profile.Exists = true
		profile.LastModified = info.ModTime()

		content, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read profile %s: %w", path, err)
		}

		profile.Content = string(content)
		profile.HasGoenv = pm.hasGoenvInit(profile.Content)

		profiles = append(profiles, profile)
	}

	return profiles, nil
}

// getProfilePath returns the primary profile path for the shell
// Uses shellutil.GetProfilePath for consistent path resolution
func (pm *ProfileManager) getProfilePath() string {
	return shellutil.GetProfilePath(pm.shell)
}

// getAllProfilePaths returns all potential profile paths for the shell
func (pm *ProfileManager) getAllProfilePaths() []string {
	home, _ := os.UserHomeDir()

	switch pm.shell {
	case ShellTypeBash:
		return []string{
			filepath.Join(home, ".bash_profile"),
			filepath.Join(home, ".bashrc"),
			filepath.Join(home, ".profile"),
		}

	case ShellTypeZsh:
		return []string{
			filepath.Join(home, ".zshrc"),
			filepath.Join(home, ".zprofile"),
			filepath.Join(home, ".zshenv"),
		}

	case ShellTypeFish:
		return []string{
			filepath.Join(home, ".config", "fish", "config.fish"),
		}

	default:
		// For other shells, just return the primary profile
		return []string{pm.getProfilePath()}
	}
}

// GetInitLine returns the shell-specific initialization command
// Uses shellutil.GetInitLine for consistent initialization commands
func (pm *ProfileManager) GetInitLine() string {
	return shellutil.GetInitLine(pm.shell)
}

// GetInitBlock returns a full initialization block with comments
func (pm *ProfileManager) GetInitBlock() string {
	initLine := pm.GetInitLine()

	switch pm.shell {
	case ShellTypeFish:
		return fmt.Sprintf("# goenv initialization\n%s\n", initLine)
	case ShellTypePowerShell:
		return fmt.Sprintf("# goenv initialization\n%s\n", initLine)
	default:
		return fmt.Sprintf("# goenv initialization\n%s\n", initLine)
	}
}

// hasGoenvInit checks if content contains goenv initialization
// Uses the same logic as shellutil.HasGoenvInProfile but operates on content string
func (pm *ProfileManager) hasGoenvInit(content string) bool {
	// Check for common goenv markers (same as shellutil.HasGoenvInProfile)
	markers := []string{
		"goenv init",
		utils.GoenvEnvVarRoot.String(),
		"goenv/shims",
	}

	for _, marker := range markers {
		if strings.Contains(content, marker) {
			return true
		}
	}
	return false
}

// AddInitialization adds goenv initialization to the profile
func (pm *ProfileManager) AddInitialization(createBackup bool) error {
	profile, err := pm.GetProfile()
	if err != nil {
		return errors.FailedTo("get profile", err)
	}

	// Check if already initialized
	if profile.HasGoenv {
		return errors.ProfileAlreadyInitialized(profile.Path)
	}

	// Create backup if requested
	if createBackup && profile.Exists {
		if err := pm.createBackup(profile.Path); err != nil {
			return errors.FailedTo("create backup", err)
		}
	}

	// Prepare new content
	initBlock := pm.GetInitBlock()
	newContent := profile.Content
	if profile.Exists && len(profile.Content) > 0 {
		// Append to existing content
		if !strings.HasSuffix(profile.Content, "\n") {
			newContent += "\n"
		}
		newContent += "\n" + initBlock
	} else {
		// New file
		newContent = initBlock
	}

	// Ensure parent directory exists
	if err := utils.EnsureDirWithContext(filepath.Dir(profile.Path), "create profile directory"); err != nil {
		return err
	}

	// Write the file
	if err := utils.WriteFileWithContext(profile.Path, []byte(newContent), utils.PermFileDefault, "write"); err != nil {
		return errors.ProfileModificationFailed("write", err)
	}

	return nil
}

// createBackup creates a timestamped backup of the profile
func (pm *ProfileManager) createBackup(profilePath string) error {
	timestamp := time.Now().Format("20060102-150405")
	backupPath := fmt.Sprintf("%s.goenv-backup.%s", profilePath, timestamp)

	content, err := os.ReadFile(profilePath)
	if err != nil {
		return err
	}

	return utils.WriteFileWithContext(backupPath, content, utils.PermFileDefault, "write file")
}

// AnalyzeIssues performs deep analysis to detect profile configuration issues
func (pm *ProfileManager) AnalyzeIssues() ([]ProfileIssue, error) {
	issues := make([]ProfileIssue, 0)

	profiles, err := pm.GetAllProfiles()
	if err != nil {
		return nil, errors.FailedTo("get profiles", err)
	}

	// Check for PATH resets
	pathResetIssues := pm.detectPathResets(profiles)
	issues = append(issues, pathResetIssues...)

	// Check for conflicts between profile files
	conflictIssues := pm.detectConflicts(profiles)
	issues = append(issues, conflictIssues...)

	// Check for duplicate initialization
	duplicateIssues := pm.detectDuplicates(profiles)
	issues = append(issues, duplicateIssues...)

	return issues, nil
}

// detectPathResets finds PATH assignments that might override goenv
func (pm *ProfileManager) detectPathResets(profiles []*Profile) []ProfileIssue {
	issues := make([]ProfileIssue, 0)

	var goenvFile string
	var resetFile string
	hasGoenv := false
	hasReset := false

	for _, profile := range profiles {
		if !profile.Exists {
			continue
		}

		lines := strings.Split(profile.Content, "\n")
		for i, line := range lines {
			line = strings.TrimSpace(line)

			// Skip comments and empty lines
			if strings.HasPrefix(line, "#") || line == "" {
				continue
			}

			// Check for goenv init
			if strings.Contains(line, "goenv init") {
				hasGoenv = true
				goenvFile = filepath.Base(profile.Path)
			}

			// Check for PATH reset patterns
			// Match: PATH="/something" or export PATH="/something" where something doesn't contain $PATH
			if (strings.Contains(line, "PATH=") || strings.Contains(line, "PATH =")) &&
				!strings.Contains(line, "goenv") {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					value := strings.TrimSpace(parts[1])
					// If PATH is set but doesn't reference previous PATH, it's a reset
					if !strings.Contains(value, "$PATH") &&
						!strings.Contains(value, "${PATH}") &&
						!strings.Contains(value, "goenv") {
						hasReset = true
						resetFile = filepath.Base(profile.Path)

						issues = append(issues, ProfileIssue{
							Type:        IssueTypePathReset,
							Severity:    SeverityWarning,
							File:        profile.Path,
							Line:        i + 1,
							Description: "PATH is reset without preserving previous value",
							Suggestion:  "Use PATH=\"/new/path:$PATH\" to preserve existing PATH entries",
						})
					}
				}
			}
		}
	}

	// Check for the common pattern: goenv in one file, reset in another
	if hasGoenv && hasReset && goenvFile != resetFile {
		issues = append(issues, ProfileIssue{
			Type:        IssueTypeWrongOrder,
			Severity:    SeverityError,
			File:        resetFile,
			Description: fmt.Sprintf("PATH reset in %s after goenv init in %s", resetFile, goenvFile),
			Suggestion:  fmt.Sprintf("Move goenv init to %s after the PATH reset, or remove the PATH reset", resetFile),
		})
	}

	return issues
}

// detectConflicts finds conflicting configurations across profile files
func (pm *ProfileManager) detectConflicts(profiles []*Profile) []ProfileIssue {
	issues := make([]ProfileIssue, 0)

	goenvFiles := make([]string, 0)
	for _, profile := range profiles {
		if profile.Exists && profile.HasGoenv {
			goenvFiles = append(goenvFiles, filepath.Base(profile.Path))
		}
	}

	// Multiple files with goenv init can cause issues
	if len(goenvFiles) > 1 {
		issues = append(issues, ProfileIssue{
			Type:        IssueTypeConflict,
			Severity:    SeverityWarning,
			Description: fmt.Sprintf("Multiple profile files contain goenv init: %s", strings.Join(goenvFiles, ", ")),
			Suggestion:  "Consider keeping goenv init in only one profile file to avoid conflicts",
		})
	}

	return issues
}

// detectDuplicates finds multiple goenv init calls in the same file
func (pm *ProfileManager) detectDuplicates(profiles []*Profile) []ProfileIssue {
	issues := make([]ProfileIssue, 0)

	for _, profile := range profiles {
		if !profile.Exists {
			continue
		}

		count := strings.Count(profile.Content, "goenv init")
		if count > 1 {
			issues = append(issues, ProfileIssue{
				Type:        IssueTypeDuplicate,
				Severity:    SeverityWarning,
				File:        profile.Path,
				Description: fmt.Sprintf("Found %d instances of 'goenv init' in %s", count, filepath.Base(profile.Path)),
				Suggestion:  "Remove duplicate goenv init lines",
			})
		}
	}

	return issues
}

// GetProfilePathDisplay returns a user-friendly display path
func (pm *ProfileManager) GetProfilePathDisplay() string {
	switch pm.shell {
	case ShellTypeBash:
		return "~/.bashrc or ~/.bash_profile"
	case ShellTypeZsh:
		return "~/.zshrc, ~/.zprofile, or ~/.zshenv"
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

// RemoveInitialization removes goenv initialization from the profile
func (pm *ProfileManager) RemoveInitialization(createBackup bool) error {
	profile, err := pm.GetProfile()
	if err != nil {
		return errors.FailedTo("get profile", err)
	}

	if !profile.Exists {
		return errors.NotFound("profile file")
	}

	if !profile.HasGoenv {
		return errors.ProfileNotInitialized(profile.Path)
	}

	// Create backup if requested
	if createBackup {
		if err := pm.createBackup(profile.Path); err != nil {
			return errors.FailedTo("create backup", err)
		}
	}

	// Remove lines containing goenv references
	lines := strings.Split(profile.Content, "\n")
	newLines := make([]string, 0, len(lines))

	skipNext := false
	for i, line := range lines {
		// Skip comment lines that are about goenv
		if strings.Contains(line, "goenv") && strings.HasPrefix(strings.TrimSpace(line), "#") {
			// Check if next line might be the init line
			if i+1 < len(lines) && strings.Contains(lines[i+1], "goenv init") {
				skipNext = true
			}
			continue
		}

		// Skip the actual goenv init line
		if skipNext || strings.Contains(line, "goenv init") || strings.Contains(line, utils.GoenvEnvVarRoot.String()) {
			skipNext = false
			continue
		}

		newLines = append(newLines, line)
	}

	newContent := strings.Join(newLines, "\n")

	// Write the file
	if err := utils.WriteFileWithContext(profile.Path, []byte(newContent), utils.PermFileDefault, "write"); err != nil {
		return errors.ProfileModificationFailed("write", err)
	}

	return nil
}
