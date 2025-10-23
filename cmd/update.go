package cmd

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:     "update",
	Short:   "Update goenv to the latest version",
	GroupID: "system",
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
	rootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	cfg := config.Load()

	fmt.Fprintln(cmd.OutOrStdout(), "🔄 Checking for goenv updates...")
	fmt.Fprintln(cmd.OutOrStdout())

	// Detect installation method
	installType, installPath, err := detectInstallation(cfg)
	if err != nil {
		return fmt.Errorf("failed to detect installation type: %w", err)
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
		return "", "", fmt.Errorf("cannot determine binary location: %w", err)
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
	if stat, err := os.Stat(gitDir); err == nil && stat.IsDir() {
		// Verify it's actually a git repo
		cmd := exec.Command("git", "rev-parse", "--git-dir")
		cmd.Dir = dir
		return cmd.Run() == nil
	}
	return false
}

// updateGitInstallation updates a git-based installation
func updateGitInstallation(cmd *cobra.Command, cfg *config.Config, gitRoot string) error {
	// Check git is available
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git not found in PATH - cannot update git-based installation")
	}

	// Get current commit
	currentCommit, err := getGitCommit(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to get current git commit: %w", err)
	}

	// Get current branch
	currentBranch, err := getGitBranch(gitRoot)
	if err != nil {
		return fmt.Errorf("failed to get current git branch: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Current commit: %s\n", currentCommit)
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Current branch: %s\n", currentBranch)
	}

	// Fetch latest changes
	fmt.Fprintln(cmd.OutOrStdout(), "📡 Fetching latest changes...")
	if err := runGitCommand(gitRoot, "fetch", "origin"); err != nil {
		return fmt.Errorf("git fetch failed: %w", err)
	}

	// Check if there are updates
	remoteCommit, err := getGitCommit(gitRoot, "origin/"+currentBranch)
	if err != nil {
		return fmt.Errorf("failed to get remote commit: %w", err)
	}

	if currentCommit == remoteCommit && !updateForce {
		fmt.Fprintln(cmd.OutOrStdout(), "✅ goenv is already up-to-date!")
		fmt.Fprintf(cmd.OutOrStdout(), "   Current version: %s\n", currentCommit[:7])
		return nil
	}

	if updateCheckOnly {
		fmt.Fprintln(cmd.OutOrStdout(), "🆕 Update available!")
		fmt.Fprintf(cmd.OutOrStdout(), "   Current:  %s\n", currentCommit[:7])
		fmt.Fprintf(cmd.OutOrStdout(), "   Latest:   %s\n", remoteCommit[:7])
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv update' to install the update.")
		return nil
	}

	// Show what will be updated
	if currentCommit != remoteCommit {
		fmt.Fprintln(cmd.OutOrStdout(), "📝 Changes:")
		if err := showGitLog(cmd, gitRoot, currentCommit, remoteCommit); err != nil {
			fmt.Fprintf(cmd.OutOrStdout(), "   (Unable to show changes: %v)\n", err)
		}
		fmt.Fprintln(cmd.OutOrStdout())
	}

	// Check for uncommitted changes
	if hasUncommittedChanges(gitRoot) {
		fmt.Fprintln(cmd.OutOrStderr(), "⚠️  Warning: You have uncommitted changes in goenv directory")
		fmt.Fprintln(cmd.OutOrStderr(), "   The update may fail or overwrite your changes.")
		fmt.Fprintln(cmd.OutOrStderr())
		if !updateForce {
			fmt.Fprintln(cmd.OutOrStderr(), "Use --force to update anyway, or commit/stash your changes first.")
			return fmt.Errorf("uncommitted changes detected")
		}
	}

	// Perform the update
	fmt.Fprintln(cmd.OutOrStdout(), "⬇️  Updating goenv...")
	if err := runGitCommand(gitRoot, "pull", "origin", currentBranch); err != nil {
		return fmt.Errorf("git pull failed: %w", err)
	}

	// Get new commit
	newCommit, err := getGitCommit(gitRoot)
	if err != nil {
		newCommit = "unknown"
	}

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "✅ goenv updated successfully!")
	fmt.Fprintf(cmd.OutOrStdout(), "   Updated from %s to %s\n", currentCommit[:7], newCommit[:7])
	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "💡 Restart your shell to use the new version:")
	fmt.Fprintln(cmd.OutOrStdout(), "   exec $SHELL")

	return nil
}

// updateBinaryInstallation updates a standalone binary installation
func updateBinaryInstallation(cmd *cobra.Command, cfg *config.Config, binaryPath string) error {
	fmt.Fprintln(cmd.OutOrStdout(), "📦 Detected binary installation")
	fmt.Fprintf(cmd.OutOrStdout(), "   Binary location: %s\n", binaryPath)
	fmt.Fprintln(cmd.OutOrStdout())

	// Get latest release from GitHub
	fmt.Fprintln(cmd.OutOrStdout(), "🔍 Checking GitHub releases...")
	latestVersion, downloadURL, err := getLatestRelease()
	if err != nil {
		return fmt.Errorf("failed to get latest release: %w", err)
	}

	if cfg.Debug {
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Latest version: %s\n", latestVersion)
		fmt.Fprintf(cmd.OutOrStdout(), "Debug: Download URL: %s\n", downloadURL)
	}

	// Check current version
	currentVersion := getCurrentVersion()
	if currentVersion == latestVersion && !updateForce {
		fmt.Fprintln(cmd.OutOrStdout(), "✅ goenv is already up-to-date!")
		fmt.Fprintf(cmd.OutOrStdout(), "   Current version: %s\n", currentVersion)
		return nil
	}

	if updateCheckOnly {
		if currentVersion == latestVersion {
			fmt.Fprintln(cmd.OutOrStdout(), "✅ goenv is up-to-date!")
			fmt.Fprintf(cmd.OutOrStdout(), "   Current version: %s\n", currentVersion)
		} else {
			fmt.Fprintln(cmd.OutOrStdout(), "🆕 Update available!")
			fmt.Fprintf(cmd.OutOrStdout(), "   Current:  %s\n", currentVersion)
			fmt.Fprintf(cmd.OutOrStdout(), "   Latest:   %s\n", latestVersion)
			fmt.Fprintln(cmd.OutOrStdout())
			fmt.Fprintln(cmd.OutOrStdout(), "Run 'goenv update' to install the update.")
		}
		return nil
	}

	// Check write permissions
	if err := checkWritePermission(binaryPath); err != nil {
		return fmt.Errorf("cannot update binary: %w\nTry running with sudo or updating manually", err)
	}

	// Download new binary
	fmt.Fprintf(cmd.OutOrStdout(), "⬇️  Downloading goenv %s...\n", latestVersion)
	tmpFile, err := downloadBinary(downloadURL)
	if err != nil {
		return fmt.Errorf("failed to download update: %w", err)
	}
	defer os.Remove(tmpFile)

	// Backup current binary
	backupPath := binaryPath + ".backup"
	fmt.Fprintln(cmd.OutOrStdout(), "💾 Creating backup...")
	if err := copyFile(binaryPath, backupPath); err != nil {
		return fmt.Errorf("failed to create backup: %w", err)
	}

	// Replace binary
	fmt.Fprintln(cmd.OutOrStdout(), "🔄 Replacing binary...")
	if err := os.Chmod(tmpFile, 0755); err != nil {
		return fmt.Errorf("failed to set permissions: %w", err)
	}

	if err := os.Rename(tmpFile, binaryPath); err != nil {
		// Try to restore backup
		os.Rename(backupPath, binaryPath)
		return fmt.Errorf("failed to replace binary: %w", err)
	}

	// Remove backup
	os.Remove(backupPath)

	fmt.Fprintln(cmd.OutOrStdout())
	fmt.Fprintln(cmd.OutOrStdout(), "✅ goenv updated successfully!")
	fmt.Fprintf(cmd.OutOrStdout(), "   Updated from %s to %s\n", currentVersion, latestVersion)

	return nil
}

// Helper functions for git operations

func getGitCommit(gitRoot string, ref ...string) (string, error) {
	args := []string{"rev-parse", "HEAD"}
	if len(ref) > 0 {
		args = []string{"rev-parse", ref[0]}
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = gitRoot
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func getGitBranch(gitRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = gitRoot
	output, err := cmd.Output()
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(output)), nil
}

func runGitCommand(gitRoot string, args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = gitRoot
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func showGitLog(cmd *cobra.Command, gitRoot string, from, to string) error {
	gitCmd := exec.Command("git", "log", "--oneline", "--no-decorate", from+".."+to)
	gitCmd.Dir = gitRoot
	output, err := gitCmd.Output()
	if err != nil {
		return err
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line != "" {
			fmt.Fprintf(cmd.OutOrStdout(), "   • %s\n", line)
		}
	}
	return nil
}

func hasUncommittedChanges(gitRoot string) bool {
	cmd := exec.Command("git", "status", "--porcelain")
	cmd.Dir = gitRoot
	output, err := cmd.Output()
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(output)) != ""
}

// Helper functions for binary updates

func getLatestRelease() (version string, downloadURL string, err error) {
	// GitHub API endpoint for latest release
	url := "https://api.github.com/repos/go-nv/goenv/releases/latest"

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(url)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("GitHub API returned status %d", resp.StatusCode)
	}

	// Parse JSON response (simplified - in production use encoding/json)
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
		platform := runtime.GOOS
		arch := runtime.GOARCH
		assetName := fmt.Sprintf("goenv_%s_%s_%s", strings.TrimPrefix(version, "v"), platform, arch)

		// Look for this asset in the release
		if strings.Contains(bodyStr, assetName) {
			downloadURL = fmt.Sprintf("https://github.com/go-nv/goenv/releases/download/%s/%s", version, assetName)
		}
	}

	if version == "" || downloadURL == "" {
		return "", "", fmt.Errorf("failed to parse release information")
	}

	return version, downloadURL, nil
}

func getCurrentVersion() string {
	// Try to get version from binary (if built with ldflags)
	// For now, return "unknown" - this would be populated at build time
	if appVersion != "" && appVersion != "dev" {
		return appVersion
	}
	return "unknown"
}

func checkWritePermission(path string) error {
	file, err := os.OpenFile(path, os.O_WRONLY, 0755)
	if err != nil {
		return err
	}
	file.Close()
	return nil
}

func downloadBinary(url string) (string, error) {
	client := &http.Client{Timeout: 60 * time.Second}
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

func copyFile(src, dst string) error {
	source, err := os.Open(src)
	if err != nil {
		return err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer destination.Close()

	if _, err := io.Copy(destination, source); err != nil {
		return err
	}

	// Copy permissions
	srcInfo, err := os.Stat(src)
	if err != nil {
		return err
	}
	return os.Chmod(dst, srcInfo.Mode())
}
