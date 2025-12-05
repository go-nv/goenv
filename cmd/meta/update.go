package meta

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	cmdpkg "github.com/go-nv/goenv/cmd"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update goenv to the latest version",
	GroupID: string(cmdpkg.GroupMeta),
	Long: `Updates goenv to the latest version.

For git-based installations (recommended):
  - Runs 'git pull' in GOENV_ROOT directory
  - Shows changes and new version

For binary installations:
  - Downloads latest release from GitHub
  - Replaces current binary
  - Requires write permission to binary location

Use --check to see if an update is available without installing.`,
	RunE: runUpdate,
}

var (
	updateCheckOnly bool
	updateForce     bool
)

func init() {
	updateCmd.Flags().BoolVarP(&updateCheckOnly, "check", "c", false, "Check for updates without installing")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Force update even if already up-to-date")
	cmdpkg.RootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	cfg, _ := cmdutil.SetupContext()

	fmt.Fprintf(cmd.OutOrStdout(), "%sChecking for goenv updates...\n", utils.Emoji("🔄 "))
	fmt.Fprintln(cmd.OutOrStdout())

	// Detect installation method
	installType, installPath, err := detectInstallation(cfg)
	if err != nil {
		return errors.FailedTo("detect installation type", err)
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Installation type: %s\n", installType)
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Installation path: %s\n", installPath)
	}

	switch installType {
	case "git":
		return updateGitInstallation(cmd, cfg, installPath)
	case "binary":
		return updateBinaryInstallation(cmd, cfg, installPath)
	default:
		return fmt.Errorf("unknown installation type: %s", installType)
	}
}

// detectInstallation determines how goenv was installed
func detectInstallation(cfg *config.Config) (string, string, error) {
	// First, check if we're running from within a git repository
	// This handles the case where someone is developing/testing goenv
	execPath, err := os.Executable()
	if err != nil {
		return "", "", errors.FailedTo("determine binary location", err)
	}

	// Resolve symlinks
	realPath, err := filepath.EvalSymlinks(execPath)
	if err != nil {
		realPath = execPath
	}

	// Check if the binary is in a git repository
	execDir := filepath.Dir(realPath)
	if isGitRepo(execDir) {
		return "git", execDir, nil
	}

	// Check if GOENV_ROOT is a git repository (standard installation)
	if isGitRepo(cfg.Root) {
		return "git", cfg.Root, nil
	}

	// Binary installation
	return "binary", realPath, nil
}

// isGitRepo checks if a directory is a git repository
func isGitRepo(dir string) bool {
	gitDir := filepath.Join(dir, ".git")
	if utils.DirExists(gitDir) {
		// Verify it's actually a git repo
		return utils.RunCommandInDir(dir, "git", "rev-parse", "--git-dir") == nil
	}
	return false
}

// updateGitInstallation updates a git-based installation
func updateGitInstallation(cmd *cobra.Command, cfg *config.Config, gitRoot string) error {
	// Check git is available
	if _, err := exec.LookPath("git"); err != nil {
		errMsg := "git not found in PATH - cannot update git-based installation\n\n"
		errMsg += "To fix this:\n"
		if utils.IsWindows() {
			errMsg += "  • Install Git for Windows: https://git-scm.com/download/win\n"
			errMsg += "  • Or install via winget: winget install Git.Git\n"
		} else if platform.IsMacOS() {
			errMsg += "  • Install Xcode Command Line Tools: xcode-select --install\n"
			errMsg += "  • Or install via Homebrew: brew install git\n"
		} else {
			errMsg += "  • Install git using your package manager (apt-get, yum, pacman, etc.)\n"
		}
		errMsg += "\nAlternatively, if you don't have write permissions to update:\n"
		errMsg += "  • Download the latest binary from: https://github.com/go-nv/goenv/releases"
		return fmt.Errorf("%s", errMsg)
	}

	// Get current commit
	currentCommit, err := getGitCommit(gitRoot)
	if err != nil {
		return errors.FailedTo("get current git commit", err)
	}

	// Get current branch
	currentBranch, err := getGitBranch(gitRoot)
	if err != nil {
		return errors.FailedTo("get current git branch", err)
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Current commit: %s\n", currentCommit)
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Current branch: %s\n", currentBranch)
	}

	// Fetch latest changes
	fmt.Fprintf(cmd.OutOrStdout(), "%sFetching latest changes...\n", utils.Emoji("📡 "))
	if err := runGitCommand(gitRoot, "fetch", "origin"); err != nil {
		return errors.FailedTo("fetch git updates", err)
	}

	// Check if there are updates
	remoteCommit, err := getGitCommit(gitRoot, "origin/"+currentBranch)
	if err != nil {
		return errors.FailedTo("get remote commit", err)
	}

	if currentCommit == remoteCommit && !updateForce {
		fmt.Fprintf(cmd.OutOrStdout(), "%sgoenv is already up-to-date!\n", utils.Emoji("✅ "))
		fmt.Fprintf(cmd.OutOrStdout(), "   Current version: %s\n", currentCommit[:7])
		return nil
	}

	if updateCheckOnly {
		fmt.Fprintf(cmd.OutOrStdout(), "%sUpdate available!\n", utils.Emoji("🆕 "))
		fmt.Fprintf(cmd.OutOrStdout(), "   Current:  %s\n", currentCommit[:7])
		fmt.Fprintf(cmd.OutOrStdout(), "   Latest:   %s\n", remoteCommit[:7])
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv update' to install the update.")
		return nil
	}

	// Show what will be updated
	if currentCommit != remoteCommit {
		fmt.Fprintf(cmd.OutOrStdout(), "%sChanges:\n", utils.Emoji("📝 "))
		if err := showGitLog(cmd, gitRoot, currentCommit, remoteCommit); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "   (Unable to show changes: %v)\n", err)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Check for uncommitted changes
	if hasUncommittedChanges(gitRoot) {
		fmt.Fprintf(cmd.OutOrStderr(), "%sWarning: You have uncommitted changes in goenv directory\n", utils.Emoji("⚠️  "))
		fmt.Fprintln(cmd.OutOrStderr(), "   The update may fail or overwrite your changes.")
		fmt.Fprintln(cmd.OutOrStderr())
		if !updateForce {
			fmt.Fprintln(cmd.OutOrStderr(), "Use --force to update anyway, or commit/stash your changes first.")
			return fmt.Errorf("uncommitted changes detected")
		}
	}

	// Perform the update
	fmt.Fprintf(cmd.OutOrStdout(), "%sUpdating goenv...\n", utils.Emoji("⬇️  "))
	if err := runGitCommand(gitRoot, "pull", "origin", currentBranch); err != nil {
		return errors.FailedTo("pull git updates", err)
	}

	// Get new commit
	newCommit, err := getGitCommit(gitRoot)
	if err != nil {
		newCommit = "unknown"
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sgoenv updated successfully!\n", utils.Emoji("✅ "))
	fmt.Fprintf(cmd.OutOrStdout(), "   Updated from %s to %s\n", currentCommit[:7], newCommit[:7])
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sRestart your shell to use the new version:\n", utils.Emoji("💡 "))
	fmt.Fprintln(cmd.OutOrStdout(), "   exec $SHELL")

	return nil
}

// updateBinaryInstallation updates a standalone binary installation
func updateBinaryInstallation(cmd *cobra.Command, cfg *config.Config, binaryPath string) error {
	fmt.Fprintf(cmd.OutOrStdout(), "%sDetected binary installation\n", utils.Emoji("📦 "))
	fmt.Fprintf(cmd.OutOrStdout(), "   Binary location: %s\n", binaryPath)
	fmt.Fprintln(cmd.OutOrStdout())

	// Get latest release from GitHub
	fmt.Fprintf(cmd.OutOrStdout(), "%sChecking GitHub releases...\n", utils.Emoji("🔍 "))
	latestVersion, downloadURL, err := getLatestRelease()
	if err != nil {
		return errors.FailedTo("get latest release", err)
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Latest version: %s\n", latestVersion)
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Download URL: %s\n", downloadURL)
	}

	// Check current version
	currentVersion := getCurrentVersion()
	if currentVersion == latestVersion && !updateForce {
		fmt.Fprintf(cmd.OutOrStdout(), "%sgoenv is already up-to-date!\n", utils.Emoji("✅ "))
		fmt.Fprintf(cmd.OutOrStdout(), "   Current version: %s\n", currentVersion)
		return nil
	}

	if updateCheckOnly {
		if currentVersion == latestVersion {
			fmt.Fprintf(cmd.OutOrStdout(), "%sgoenv is up-to-date!\n", utils.Emoji("✅ "))
			fmt.Fprintf(cmd.OutOrStdout(), "   Current version: %s\n", currentVersion)
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "%sUpdate available!\n", utils.Emoji("🆕 "))
			fmt.Fprintf(cmd.OutOrStdout(), "   Current:  %s\n", currentVersion)
			fmt.Fprintf(cmd.OutOrStdout(), "   Latest:   %s\n", latestVersion)
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv update' to install the update.")
		}
		return nil
	}

	// Check write permissions
	if err := checkWritePermission(binaryPath); err != nil {
		errMsg := fmt.Sprintf("cannot update binary: %v\n\n", err)
		errMsg += "To fix this:\n"
		if utils.IsWindows() {
			errMsg += "  • Run PowerShell as Administrator, or\n"
			errMsg += "  • Install goenv to a user-writeable path like %LOCALAPPDATA%\\goenv\n"
			errMsg += "    (e.g., C:\\Users\\YourName\\AppData\\Local\\goenv)\n"
		} else {
			errMsg += "  • Run with elevated permissions: sudo goenv update\n"
			errMsg += "  • Or install goenv to a user-writeable path (e.g., ~/.local/bin/)\n"
		}
		errMsg += "\nAlternatively, download and install manually:\n"
		errMsg += "  • https://github.com/go-nv/goenv/releases"
		return fmt.Errorf("%s", errMsg)
	}

	// Download new binary
	fmt.Fprintf(cmd.OutOrStdout(), "%sDownloading goenv %s...\n", utils.Emoji("⬇️  "), latestVersion)
	tmpFile, err := downloadBinary(downloadURL)
	if err != nil {
		return errors.FailedTo("download update", err)
	}
	defer os.Remove(tmpFile)

	// Verify checksum if available
	fmt.Fprintf(cmd.OutOrStdout(), "%sVerifying checksum...\n", utils.Emoji("🔐 "))
	checksumURL := fmt.Sprintf("https://github.com/go-nv/goenv/releases/download/%s/SHA256SUMS", latestVersion)
	if err := verifyChecksum(tmpFile, checksumURL, filepath.Base(downloadURL)); err != nil {
		if cfg.Debug {
			fmt.Fprintf(cmd.OutOrStdout(), "Debug: Checksum verification: %v\n", err)
		}
		// Warn but don't block (for older releases without checksums)
		fmt.Fprintf(cmd.OutOrStdout(), "%sWarning: Could not verify checksum (proceeding anyway)\n", utils.Emoji("⚠️  "))
		fmt.Fprintln(cmd.OutOrStdout(), "   This may indicate the release doesn't have checksums published")
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "%sChecksum verified\n", utils.Emoji("✅ "))
	}

	// Backup current binary
	backupPath := binaryPath + ".backup"
	fmt.Fprintf(cmd.OutOrStdout(), "%sCreating backup...\n", utils.Emoji("💾 "))
	if err := utils.CopyFile(binaryPath, backupPath); err != nil {
		return errors.FailedTo("create backup", err)
	}

	// Replace binary
	fmt.Fprintf(cmd.OutOrStdout(), "%sReplacing binary...\n", utils.Emoji("🔄 "))

	// Make executable on Unix (Windows uses file extension for executability)
	if !utils.IsWindows() {
		if err := os.Chmod(tmpFile, utils.PermFileExecutable); err != nil {
			return errors.FailedTo("set permissions", err)
		}
	}

	// On Windows, we cannot replace a running executable directly
	// Instead, we use a two-step process:
	// 1. Rename the new binary to the target name with .new extension
	// 2. Create a batch script that waits, renames, and restarts
	if utils.IsWindows() {
		return replaceWindowsBinary(cmd, tmpFile, binaryPath, backupPath, currentVersion, latestVersion)
	}

	// On Unix, we can replace the binary directly
	if err := os.Rename(tmpFile, binaryPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, binaryPath)
		return errors.FailedTo("replace binary", err)
	}

	// Remove backup
	os.Remove(backupPath)

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sgoenv updated successfully!\n", utils.Emoji("✅ "))
	fmt.Fprintf(cmd.OutOrStdout(), "   Updated from %s to %s\n", currentVersion, latestVersion)

	return nil
}

// Helper functions for git operations

func getGitCommit(gitRoot string, ref ...string) (string, error) {
	args := []string{"rev-parse", "HEAD"}
	if len(ref) > 0 {
		args = []string{"rev-parse", ref[0]}
	}

	return utils.RunCommandOutputInDir(gitRoot, "git", args...)
}

func getGitBranch(gitRoot string) (string, error) {
	return utils.RunCommandOutputInDir(gitRoot, "git", "rev-parse", "--abbrev-ref", "HEAD")
}

func runGitCommand(gitRoot string, args ...string) error {
	return utils.RunCommandWithIOInDir(gitRoot, "git", args, os.Stdout, os.Stderr)
}

func showGitLog(cmd *cobra.Command, gitRoot string, from, to string) error {
	output, err := utils.RunCommandOutputInDir(gitRoot, "git", "log", "--oneline", "--no-decorate", from+".."+to)
	if err != nil {
		return err
	}

	lines := strings.Split(output, "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   • %s\n", line)
		}
	}
	return nil
}

func hasUncommittedChanges(gitRoot string) bool {
	output, err := utils.RunCommandOutputInDir(gitRoot, "git", "status", "--porcelain")
	if err != nil {
		return false
	}
	return output != ""
}

// Helper functions for binary updates

func getLatestRelease() (version string, downloadURL string, err error) {
	// GitHub API endpoint for latest release
	apiURL := "https://api.github.com/repos/go-nv/goenv/releases/latest"

	// Try to load cached ETag
	cfg, _ := cmdutil.SetupContext()
	etagFile := filepath.Join(cfg.Root, "cache", "update-etag")
	cachedETag, _ := os.ReadFile(etagFile)

	// Create request with ETag support
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return "", "", errors.FailedTo("create HTTP request", err)
	}

	// Set headers
	req.Header.Set("Accept", "application/vnd.github.v3+json")
	if len(cachedETag) > 0 {
		req.Header.Set("If-None-Match", strings.TrimSpace(string(cachedETag)))
	}

	// Check for GitHub token for higher rate limits
	if token := os.Getenv(utils.EnvVarGitHubToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "Using GitHub token for higher rate limits\n")
		}
	}

	// Make request with retries for rate limiting
	client := utils.NewHTTPClient(10 * time.Second)
	var resp *http.Response

	maxRetries := 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err = client.Do(req)
		if err != nil {
			return "", "", errors.FailedTo("execute HTTP request", err)
		}

		// Handle rate limiting
		if resp.StatusCode == 403 || resp.StatusCode == 429 {
			resp.Body.Close()

			// Check rate limit headers
			remaining := resp.Header.Get("X-RateLimit-Remaining")
			resetTime := resp.Header.Get("X-RateLimit-Reset")

			if remaining == "0" && resetTime != "" {
				if reset, err := strconv.ParseInt(resetTime, 10, 64); err == nil {
					resetAt := time.Unix(reset, 0)
					waitDuration := time.Until(resetAt)
					if waitDuration > 0 && waitDuration < 5*time.Minute {
						return "", "", fmt.Errorf("GitHub API rate limit exceeded. Resets at %s (in %v)",
							resetAt.Format(time.RFC3339), waitDuration.Round(time.Second))
					}
				}
				return "", "", fmt.Errorf("GitHub API rate limit exceeded. Try again later")
			}

			// Handle 429 with Retry-After
			if resp.StatusCode == 429 {
				retryAfter := resp.Header.Get("Retry-After")
				if retryAfter != "" {
					if seconds, err := strconv.Atoi(retryAfter); err == nil && seconds < 60 {
						if attempt < maxRetries-1 {
							time.Sleep(time.Duration(seconds) * time.Second)
							continue
						}
					}
				}
			}

			// Exponential backoff
			if attempt < maxRetries-1 {
				backoff := time.Duration(1<<uint(attempt)) * time.Second
				time.Sleep(backoff)
				continue
			}

			return "", "", fmt.Errorf("GitHub API rate limit exceeded after %d attempts", maxRetries)
		}

		break
	}
	defer resp.Body.Close()

	// Handle 304 Not Modified
	if resp.StatusCode == http.StatusNotModified {
		return "", "", fmt.Errorf("no updates available (cached)")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		return "", "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(body))
	}

	// Save ETag for next request
	if etag := resp.Header.Get("ETag"); etag != "" {
		// Ensure cache directory exists with secure permissions
		if err := utils.EnsureDirWithContext(filepath.Dir(etagFile), "create cache directory"); err != nil {
			// Non-fatal: log but continue
			if cfg.Debug {
				fmt.Fprintf(os.Stderr, "Warning: failed to create cache directory: %v\n", err)
			}
		} else {
			// Write ETag file with secure permissions
			if err := utils.WriteFileWithContext(etagFile, []byte(etag), utils.PermFileSecure, "save etag cache"); err != nil {
				// Non-fatal: log but continue
				if cfg.Debug {
					fmt.Fprintf(os.Stderr, "Warning: failed to save ETag cache: %v\n", err)
				}
			}
		}
	}

	// Parse JSON response
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", "", err
	}

	bodyStr := string(body)

	// Extract version (tag_name)
	if idx := strings.Index(bodyStr, `"tag_name"`); idx != -1 {
		start := strings.Index(bodyStr[idx:], `"`) + idx + 1
		start = strings.Index(bodyStr[start:], `"`) + start + 1
		end := strings.Index(bodyStr[start:], `"`) + start
		version = bodyStr[start:end]
	}

	// Build download URL for current platform
	if version != "" {
		osName := platform.OS()
		arch := platform.Arch()
		assetName := fmt.Sprintf("goenv_%s_%s_%s", strings.TrimPrefix(version, "v"), osName, arch)

		// Look for this asset in the release
		if strings.Contains(bodyStr, assetName) {
			downloadURL = fmt.Sprintf("https://github.com/go-nv/goenv/releases/download/%s/%s", version, assetName)
		}
	}

	if version == "" || downloadURL == "" {
		return "", "", errors.FailedTo("parse release information", fmt.Errorf("incomplete release data"))
	}

	return version, downloadURL, nil
}

func getCurrentVersion() string {
	// Try to get version from binary (if built with ldflags)
	// For now, return "unknown" - this would be populated at build time
	if cmdpkg.AppVersion != "" && cmdpkg.AppVersion != "dev" {
		return cmdpkg.AppVersion
	}
	return "unknown"
}

func checkWritePermission(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY, utils.PermFileExecutable)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func downloadBinary(url string) (string, error) {
	client := utils.NewHTTPClient(60 * time.Second)
	resp, err := client.Get(url)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("download failed with status %d", resp.StatusCode)
	}

	// Create temporary file
	tmpFile, err := os.CreateTemp("", "goenv-update-*")
	if err != nil {
		return "", err
	}
	defer tmpFile.Close()

	// Copy downloaded content
	if _, err := io.Copy(tmpFile, resp.Body); err != nil {
		os.Remove(tmpFile.Name())
		return "", err
	}

	return tmpFile.Name(), nil
}

// verifyChecksum downloads SHA256SUMS and verifies the binary matches
func verifyChecksum(binaryPath, checksumURL, filename string) error {
	// Download checksum file
	client := utils.NewHTTPClient(10 * time.Second)
	resp, err := client.Get(checksumURL)
	if err != nil {
		return errors.FailedTo("download checksums", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return fmt.Errorf("checksums file not found (release may not have published checksums)")
	}

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("checksum download failed with status %d", resp.StatusCode)
	}

	checksums, err := io.ReadAll(resp.Body)
	if err != nil {
		return errors.FailedTo("read checksums", err)
	}

	// Parse SHA256SUMS file format: "<hash>  <filename>"
	lines := strings.Split(string(checksums), "\n")
	var expectedHash string
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.Fields(line)
		if len(parts) >= 2 && parts[1] == filename {
			expectedHash = parts[0]
			break
		}
	}

	if expectedHash == "" {
		return fmt.Errorf("checksum not found for %s", filename)
	}

	// Calculate actual hash of downloaded binary
	file, err := os.Open(binaryPath)
	if err != nil {
		return errors.FailedTo("open binary", err)
	}
	defer file.Close()

	hasher := sha256.New()
	if _, err := io.Copy(hasher, file); err != nil {
		return errors.FailedTo("calculate hash", err)
	}

	actualHash := hex.EncodeToString(hasher.Sum(nil))

	// Compare hashes
	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s, got %s (possible tampering!)", expectedHash, actualHash)
	}

	return nil
}

// replaceWindowsBinary handles binary replacement on Windows using a helper batch script
// Windows locks running executables, so we can't replace the binary directly.
// Instead, we move the new binary to .new, create a batch script that:
// 1. Waits for goenv.exe to exit
// 2. Moves old binary to .backup
// 3. Moves new binary to final location
// 4. Cleans up
func replaceWindowsBinary(cmd *cobra.Command, tmpFile, binaryPath, backupPath, currentVersion, latestVersion string) error {
	// Move new binary next to the old one with .new extension
	newPath := binaryPath + ".new"
	if err := os.Rename(tmpFile, newPath); err != nil {
		return errors.FailedTo("move new binary", err)
	}

	// Create batch script to complete the update after goenv exits
	updateScript := binaryPath + ".update.bat"
	scriptContent := fmt.Sprintf(`@echo off
REM goenv Windows Update Helper Script
REM This script completes the update after goenv.exe exits

echo Waiting for goenv.exe to exit...
:WAIT
timeout /t 1 /nobreak >nul 2>&1
tasklist /FI "IMAGENAME eq %s" 2>nul | find /I "%s" >nul
if not errorlevel 1 goto WAIT

echo Replacing binary...
move /Y "%s" "%s" >nul 2>&1
move /Y "%s" "%s" >nul 2>&1
del /Q "%s" >nul 2>&1

echo.
echo goenv updated successfully!
echo    Updated from %s to %s
echo.
echo Update complete. You can close this window.

REM Self-delete the update script
del "%%~f0" >nul 2>&1
`, filepath.Base(binaryPath), filepath.Base(binaryPath),
		binaryPath, backupPath,
		newPath, binaryPath,
		updateScript,
		currentVersion, latestVersion)

	if err := utils.WriteFileWithContext(updateScript, []byte(scriptContent), utils.PermFileDefault, "create update script"); err != nil {
		os.Remove(newPath)
		return err
	}

	// Start the batch script in a new console window
	startCmd := exec.Command("cmd", "/C", "start", "goenv Update", updateScript)
	startCmd.SysProcAttr = nil // Let it run detached
	if err := startCmd.Start(); err != nil {
		os.Remove(newPath)
		os.Remove(updateScript)
		return errors.FailedTo("start update script", err)
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "%sUpdate downloaded and prepared!\n", utils.Emoji("✅ "))
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "A batch script has been started to complete the update.")
	fmt.Fprintln(cmd.OutOrStdout(), "The update will finish automatically when goenv exits.")
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintf(cmd.OutOrStdout(), "   Updating from %s to %s\n", currentVersion, latestVersion)

	return nil
}
