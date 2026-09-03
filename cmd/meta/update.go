package meta

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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
	updateNoElevate bool
)

func init() {
	updateCmd.Flags().BoolVarP(&updateCheckOnly, "check", "c", false, "Check for updates without installing")
	updateCmd.Flags().BoolVarP(&updateForce, "force", "f", false, "Force update even if already up-to-date")
	updateCmd.Flags().BoolVar(&updateNoElevate, "no-elevate", false, "Never request elevated privileges; print manual instructions instead")
	cmdpkg.RootCmd.AddCommand(updateCmd)
}

func runUpdate(cmd *cobra.Command, args []string) error {
	ctx := cmdutil.GetContexts(cmd)
	cfg := ctx.Config

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

	// An install owned by a package manager must be updated through it, not
	// overwritten in place — see packageManager for why elevating instead would
	// make things worse.
	if manager, upgradeCmd := packageManager(binaryPath); manager != "" {
		return fmt.Errorf("goenv was installed by %s, so 'goenv update' would be undone by the next %s upgrade\n\n"+
			"Update it with:\n  %s", manager, manager, upgradeCmd)
	}

	// Replacing the binary is a rename in its directory, so that is what has to
	// be writable. When it is not, the install step (and only the install step)
	// is escalated.
	elevate := false
	if err := canReplaceBinary(binaryPath); err != nil {
		if updateNoElevate {
			return fmt.Errorf("%s", elevationInstructions(binaryPath))
		}
		if err := confirmElevation(cmd, binaryPath, latestVersion); err != nil {
			return err
		}
		elevate = true
	}

	// Download new release archive
	fmt.Fprintf(cmd.OutOrStdout(), "%sDownloading goenv %s...\n", utils.Emoji("⬇️  "), latestVersion)
	tmpFile, err := downloadBinary(downloadURL)
	if err != nil {
		return errors.FailedTo("download update", err)
	}
	defer os.Remove(tmpFile)

	// Verify the checksum of the downloaded archive before unpacking anything.
	// goreleaser publishes "goenv_<version>_checksums.txt"; the previously used
	// "SHA256SUMS" name has never existed, so verification always fell through
	// to the "proceeding anyway" warning.
	fmt.Fprintf(cmd.OutOrStdout(), "%sVerifying checksum...\n", utils.Emoji("🔐 "))
	checksumURL := fmt.Sprintf("https://github.com/go-nv/goenv/releases/download/%s/goenv_%s_checksums.txt",
		latestVersion, strings.TrimPrefix(latestVersion, "v"))

	switch err := verifyChecksum(tmpFile, checksumURL, filepath.Base(downloadURL)); {
	case err == nil:
		fmt.Fprintf(cmd.OutOrStdout(), "%sChecksum verified\n", utils.Emoji("✅ "))

	case stderrors.Is(err, errChecksumsUnpublished):
		// Releases predating goreleaser checksums genuinely have none. This is
		// the only failure that does not abort: there is nothing to verify
		// against, as opposed to a check that did not pass.
		fmt.Fprintf(cmd.OutOrStderr(), "%sThis release publishes no checksums; the download cannot be verified.\n",
			utils.Emoji("⚠️  "))

	default:
		// Anything else means verification was possible and did not succeed:
		// a mismatch, a missing entry for this artifact, or an unreadable
		// checksums file. Installing regardless would execute an unverified
		// binary with the user's privileges, which is the exact outcome this
		// step exists to prevent.
		return fmt.Errorf("checksum verification failed: %w\n\n"+
			"The downloaded archive does not match its published checksum, so it has NOT been installed.\n"+
			"Retry the update; if it persists, download the release manually from\n"+
			"https://github.com/go-nv/goenv/releases and verify it yourself", err)
	}

	// Release assets are archives, not bare binaries — unpack the goenv
	// executable before it can replace the installed one.
	//
	// Staged in the installed binary's own directory so the replacement below
	// is a same-filesystem rename. Extracting to $TMPDIR would fail with EXDEV
	// wherever /tmp is a separate mount. That trade does not apply when the
	// directory is unwritable: the elevated helper copies rather than renames,
	// and a 0700 temp directory keeps the verified archive out of reach of other
	// users until root reads it.
	stageDir := filepath.Dir(binaryPath)
	if elevate {
		stageDir, err = os.MkdirTemp("", "goenv-update-")
		if err != nil {
			return errors.FailedTo("create staging directory", err)
		}
		defer os.RemoveAll(stageDir)
	}

	newBinary, err := extractGoenvBinary(tmpFile, downloadURL, stageDir)
	if err != nil {
		return errors.FailedTo("extract update", err)
	}
	defer os.Remove(newBinary)

	// Make executable on Unix (Windows uses file extension for executability)
	if !utils.IsWindows() {
		if err := os.Chmod(newBinary, utils.PermFileExecutable); err != nil {
			return errors.FailedTo("set permissions", err)
		}
	}

	backupPath := binaryPath + ".backup"

	if elevate {
		if err := replaceElevated(cmd, newBinary, binaryPath, backupPath, currentVersion, latestVersion); err != nil {
			return err
		}
		if utils.IsWindows() {
			return nil
		}
		fmt.Fprintln(cmd.OutOrStdout())
		fmt.Fprintf(cmd.OutOrStdout(), "%sgoenv updated successfully!\n", utils.Emoji("✅ "))
		fmt.Fprintf(cmd.OutOrStdout(), "   Updated from %s to %s\n", currentVersion, latestVersion)
		return nil
	}

	// Backup current binary
	fmt.Fprintf(cmd.OutOrStdout(), "%sCreating backup...\n", utils.Emoji("💾 "))
	if err := utils.CopyFile(binaryPath, backupPath); err != nil {
		return errors.FailedTo("create backup", err)
	}

	// Replace binary
	fmt.Fprintf(cmd.OutOrStdout(), "%sReplacing binary...\n", utils.Emoji("🔄 "))

	// On Windows, we cannot replace a running executable directly
	// Instead, we use a two-step process:
	// 1. Rename the new binary to the target name with .new extension
	// 2. Create a batch script that waits, renames, and restarts
	if utils.IsWindows() {
		return replaceWindowsBinary(cmd, newBinary, binaryPath, backupPath, currentVersion, latestVersion, false)
	}

	// On Unix, we can replace the binary directly
	if err := os.Rename(newBinary, binaryPath); err != nil {
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

const (
	// releasesAPIURL lists releases rather than using /releases/latest.
	//
	// /releases/latest returns the most recently published release across *all*
	// branches. The v2 maintenance branch publishes documentation-only releases
	// with no binary assets, so /releases/latest regularly points at a release
	// this binary cannot update to (issue #582). GitHub returns releases
	// newest-first, so we take the first one that ships an asset for this
	// platform. A small page keeps the payload cheap while still reaching past
	// any run of assetless v2 releases.
	releasesAPIURL = "https://api.github.com/repos/go-nv/goenv/releases?per_page=10"

	// releasesAtomURL is served by github.com instead of the REST API, so it is
	// not subject to the (60/hour, per-IP) unauthenticated API rate limit.
	releasesAtomURL = "https://github.com/go-nv/goenv/releases.atom"

	releaseDownloadBase = "https://github.com/go-nv/goenv/releases/download"
)

// errNotModified reports that GitHub answered a conditional request with 304,
// meaning the cached payload is still current.
var errNotModified = stderrors.New("release list not modified")

// errRateLimited reports that GitHub refused the request because the caller is
// over its API rate limit.
var errRateLimited = stderrors.New("GitHub API rate limit exceeded")

func getLatestRelease() (version string, downloadURL string, err error) {
	cfg := config.Load()
	cacheDir := filepath.Join(cfg.Root, "cache")
	etagFile := filepath.Join(cacheDir, "update-etag")
	bodyFile := filepath.Join(cacheDir, "update-releases.json")

	cachedBody, cacheErr := os.ReadFile(bodyFile)
	hasCache := cacheErr == nil && len(cachedBody) > 0

	req, err := http.NewRequest("GET", releasesAPIURL, nil)
	if err != nil {
		return "", "", errors.FailedTo("create HTTP request", err)
	}
	req.Header.Set("Accept", "application/vnd.github.v3+json")

	// Only send the validator when the cached payload is available to serve on
	// a 304; otherwise a 304 would leave us with nothing to parse. Conditional
	// requests that return 304 do not count against the API rate limit.
	if hasCache {
		if etag, err := os.ReadFile(etagFile); err == nil && len(etag) > 0 {
			req.Header.Set("If-None-Match", strings.TrimSpace(string(etag)))
		}
	}

	if token := os.Getenv(utils.EnvVarGitHubToken); token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "Using GitHub token for higher rate limits\n")
		}
	}

	body, etag, err := fetchReleases(req)
	switch {
	case err == nil:
		cacheReleases(cacheDir, etagFile, bodyFile, etag, body, cfg.Debug)
	case stderrors.Is(err, errNotModified):
		body = cachedBody
	case hasCache:
		// Rate limited or a transient failure: a slightly stale answer beats no
		// answer, and the caller compares versions anyway.
		if cfg.Debug {
			fmt.Fprintf(os.Stderr, "Warning: using cached release list: %v\n", err)
		}
		body = cachedBody
	default:
		if v, u, atomErr := latestReleaseFromAtom(); atomErr == nil {
			return v, u, nil
		}
		if stderrors.Is(err, errRateLimited) {
			return "", "", fmt.Errorf("%w. Set %s to raise the limit, or retry later",
				err, utils.EnvVarGitHubToken)
		}
		return "", "", err
	}

	version, downloadURL = selectRelease(body, platform.OS(), platform.Arch())
	if version == "" || downloadURL == "" {
		return "", "", errors.FailedTo("parse release information",
			fmt.Errorf("no release found with a %s_%s binary", platform.OS(), platform.Arch()))
	}

	return version, downloadURL, nil
}

// fetchReleases performs req, retrying transient rate limiting, and returns the
// response body together with its ETag. It returns errNotModified when the
// cached payload is still valid and errRateLimited when GitHub refuses to serve
// the request at all.
func fetchReleases(req *http.Request) (body []byte, etag string, err error) {
	client := utils.NewHTTPClient(10 * time.Second)

	const maxRetries = 3
	for attempt := 0; attempt < maxRetries; attempt++ {
		resp, err := client.Do(req)
		if err != nil {
			return nil, "", errors.FailedTo("execute HTTP request", err)
		}

		if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusTooManyRequests {
			resp.Body.Close()

			if wait, ok := rateLimitReset(resp.Header); ok {
				return nil, "", fmt.Errorf("%w. Resets in %v", errRateLimited, wait.Round(time.Second))
			}

			if attempt == maxRetries-1 {
				return nil, "", fmt.Errorf("%w after %d attempts", errRateLimited, maxRetries)
			}

			time.Sleep(retryDelay(resp.Header, attempt))
			continue
		}

		defer resp.Body.Close()

		if resp.StatusCode == http.StatusNotModified {
			return nil, "", errNotModified
		}
		if resp.StatusCode != http.StatusOK {
			snippet, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
			return nil, "", fmt.Errorf("GitHub API returned status %d: %s", resp.StatusCode, string(snippet))
		}

		body, err = io.ReadAll(resp.Body)
		if err != nil {
			return nil, "", err
		}
		return body, resp.Header.Get("ETag"), nil
	}

	return nil, "", fmt.Errorf("%w after %d attempts", errRateLimited, maxRetries)
}

// rateLimitReset reports how long until the primary rate limit resets, if the
// response says the quota is actually exhausted (as opposed to a retryable 403).
func rateLimitReset(header http.Header) (time.Duration, bool) {
	if header.Get("X-RateLimit-Remaining") != "0" {
		return 0, false
	}
	reset, err := strconv.ParseInt(header.Get("X-RateLimit-Reset"), 10, 64)
	if err != nil {
		return 0, true
	}
	wait := time.Until(time.Unix(reset, 0))
	if wait < 0 {
		wait = 0
	}
	return wait, true
}

func retryDelay(header http.Header, attempt int) time.Duration {
	if seconds, err := strconv.Atoi(header.Get("Retry-After")); err == nil && seconds > 0 && seconds < 60 {
		return time.Duration(seconds) * time.Second
	}
	return time.Duration(1<<uint(attempt)) * time.Second
}

// cacheReleases stores the payload and its validator so the next check can be
// answered by a rate-limit-free conditional request (or by the cache itself
// when GitHub is unreachable). Failures here are non-fatal.
func cacheReleases(cacheDir, etagFile, bodyFile, etag string, body []byte, debug bool) {
	if etag == "" {
		return
	}

	warn := func(err error) {
		if debug && err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to cache release list: %v\n", err)
		}
	}

	if err := utils.EnsureDirWithContext(cacheDir, "create cache directory"); err != nil {
		warn(err)
		return
	}
	if err := utils.WriteFileWithContext(bodyFile, body, utils.PermFileSecure, "save release cache"); err != nil {
		warn(err)
		return
	}
	warn(utils.WriteFileWithContext(etagFile, []byte(etag), utils.PermFileSecure, "save etag cache"))
}

// latestReleaseFromAtom is the last-resort lookup used when the REST API is
// unavailable. The feed lists tags only, so the asset URL is derived from the
// release naming convention rather than read from the payload.
func latestReleaseFromAtom() (version string, downloadURL string, err error) {
	client := utils.NewHTTPClient(10 * time.Second)
	resp, err := client.Get(releasesAtomURL)
	if err != nil {
		return "", "", err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", "", fmt.Errorf("releases feed returned status %d", resp.StatusCode)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return "", "", err
	}

	version, downloadURL = selectAtomRelease(body, platform.OS(), platform.Arch(), getCurrentVersion())
	if version == "" {
		return "", "", fmt.Errorf("no release found in feed with a %s_%s binary", platform.OS(), platform.Arch())
	}
	return version, downloadURL, nil
}

// selectAtomRelease picks the newest tag in an atom releases feed that belongs
// to the same major version as currentVersion. The major filter keeps the v2
// maintenance line (which ships no binaries) out of the result; when the
// running version is unknown, the newest tag wins.
func selectAtomRelease(body []byte, osName, arch, currentVersion string) (version string, downloadURL string) {
	var feed struct {
		Entries []struct {
			ID string `xml:"id"`
		} `xml:"entry"`
	}
	if err := xml.Unmarshal(body, &feed); err != nil {
		return "", ""
	}

	wantMajor, _, _ := strings.Cut(strings.TrimPrefix(currentVersion, "v"), ".")

	for _, entry := range feed.Entries {
		// IDs look like "tag:github.com,2008:Repository/12345/3.1.5".
		idx := strings.LastIndex(entry.ID, "/")
		if idx < 0 {
			continue
		}
		tag := entry.ID[idx+1:]
		if tag == "" {
			continue
		}

		major, _, _ := strings.Cut(strings.TrimPrefix(tag, "v"), ".")
		if wantMajor != "" && wantMajor != "unknown" && major != wantMajor {
			continue
		}

		return tag, releaseAssetURL(tag, osName, arch)
	}

	return "", ""
}

func releaseAssetURL(tag, osName, arch string) string {
	ext := ".tar.gz"
	if osName == "windows" {
		ext = ".zip"
	}
	return fmt.Sprintf("%s/%s/goenv_%s_%s_%s%s",
		releaseDownloadBase, tag, strings.TrimPrefix(tag, "v"), osName, arch, ext)
}

// githubRelease is the subset of the GitHub release payload we need.
type githubRelease struct {
	TagName    string `json:"tag_name"`
	Draft      bool   `json:"draft"`
	Prerelease bool   `json:"prerelease"`
	Assets     []struct {
		Name               string `json:"name"`
		BrowserDownloadURL string `json:"browser_download_url"`
	} `json:"assets"`
}

// selectRelease returns the tag and download URL of the newest published
// release in body that ships a binary for the given os/arch. It returns empty
// strings when no such release exists.
func selectRelease(body []byte, osName, arch string) (version string, downloadURL string) {
	var releases []githubRelease
	if err := json.Unmarshal(body, &releases); err != nil {
		return "", ""
	}

	assetPrefix := fmt.Sprintf("_%s_%s", osName, arch)

	for _, release := range releases {
		if release.Draft || release.Prerelease || release.TagName == "" {
			continue
		}

		want := fmt.Sprintf("goenv_%s%s", strings.TrimPrefix(release.TagName, "v"), assetPrefix)
		for _, asset := range release.Assets {
			// Match with or without an archive extension.
			if asset.Name != want && !strings.HasPrefix(asset.Name, want+".") {
				continue
			}
			if asset.BrowserDownloadURL != "" {
				return release.TagName, asset.BrowserDownloadURL
			}
			return release.TagName, fmt.Sprintf(
				"https://github.com/go-nv/goenv/releases/download/%s/%s", release.TagName, asset.Name)
		}
	}

	return "", ""
}

func getCurrentVersion() string {
	// Try to get version from binary (if built with ldflags)
	// For now, return "unknown" - this would be populated at build time
	if cmdpkg.AppVersion != "" && cmdpkg.AppVersion != "dev" {
		return cmdpkg.AppVersion
	}
	return "unknown"
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

// extractGoenvBinary unpacks the goenv executable from a downloaded release
// archive and returns the path to a temporary file containing it.
//
// Release assets are ".tar.gz" (or ".zip" on Windows) archives, so the
// downloaded file cannot be installed as an executable directly.
func extractGoenvBinary(archivePath, sourceURL, destDir string) (string, error) {
	wantNames := []string{"goenv", "goenv.exe"}

	if strings.HasSuffix(sourceURL, ".zip") {
		return extractFromZip(archivePath, wantNames, destDir)
	}
	return extractFromTarGz(archivePath, wantNames, destDir)
}

func extractFromTarGz(archivePath string, wantNames []string, destDir string) (string, error) {
	file, err := os.Open(archivePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return "", err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
		if header.Typeflag != tar.TypeReg {
			continue
		}
		// Match on the base name only; never use the archive's path.
		if !slices.Contains(wantNames, filepath.Base(header.Name)) {
			continue
		}
		return writeTempBinary(io.LimitReader(tr, maxBinarySize), destDir)
	}

	return "", fmt.Errorf("no goenv binary found in archive")
}

func extractFromZip(archivePath string, wantNames []string, destDir string) (string, error) {
	reader, err := zip.OpenReader(archivePath)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	for _, entry := range reader.File {
		if entry.FileInfo().IsDir() {
			continue
		}
		if !slices.Contains(wantNames, filepath.Base(entry.Name)) {
			continue
		}

		rc, err := entry.Open()
		if err != nil {
			return "", err
		}
		path, err := writeTempBinary(io.LimitReader(rc, maxBinarySize), destDir)
		rc.Close()
		return path, err
	}

	return "", fmt.Errorf("no goenv binary found in archive")
}

// maxBinarySize caps how much data is unpacked from a release archive,
// bounding decompression-bomb exposure.
const maxBinarySize = 256 << 20 // 256 MiB

// writeTempBinary writes src to a new file in destDir and returns its path.
//
// destDir must be the directory the binary will finally live in. The caller
// installs the result with os.Rename, and rename cannot cross filesystems:
// writing to $TMPDIR instead fails with EXDEV wherever /tmp is a separate
// mount, which is the systemd default on many Linux distributions. Staging in
// the destination directory also keeps the replacement atomic, and gives the
// new binary its own inode — replacing an executable's contents in place
// invalidates its code signature and macOS then kills it on exec.
func writeTempBinary(src io.Reader, destDir string) (string, error) {
	out, err := os.CreateTemp(destDir, ".goenv-update-bin-*")
	if err != nil {
		// A read-only or otherwise unwritable destination is a legitimate
		// failure to report: the update could not be staged where it must land.
		return "", fmt.Errorf("cannot stage update in %s: %w", destDir, err)
	}
	defer out.Close()

	if _, err := io.Copy(out, src); err != nil {
		os.Remove(out.Name())
		return "", err
	}

	return out.Name(), nil
}

// errChecksumsUnpublished means the release does not publish a checksums file
// at all. Releases predating goreleaser checksums are the only legitimate case,
// so this is the single failure mode that does not abort the update.
var errChecksumsUnpublished = fmt.Errorf("release does not publish checksums")

// verifyChecksum downloads the release checksums file and verifies the
// downloaded artifact matches.
//
// Every failure except errChecksumsUnpublished must be treated as fatal by the
// caller. A verification step that warns and continues is not a security
// control — it is a log line, and it offers no protection against exactly the
// tampered or truncated artifact it exists to catch.
func verifyChecksum(binaryPath, checksumURL, filename string) error {
	// Download checksum file
	client := utils.NewHTTPClient(10 * time.Second)
	resp, err := client.Get(checksumURL)
	if err != nil {
		return errors.FailedTo("download checksums", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return errChecksumsUnpublished
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
//
// When elevated is set the install directory is not writable, so the staged
// binary and the script stay in the caller's private temp directory and the
// script itself is launched through a UAC prompt.
func replaceWindowsBinary(cmd *cobra.Command, tmpFile, binaryPath, backupPath, currentVersion, latestVersion string, elevated bool) error {
	newPath := tmpFile
	updateScript := filepath.Join(filepath.Dir(tmpFile), "goenv-update.bat")

	if !elevated {
		// Move new binary next to the old one with .new extension
		newPath = binaryPath + ".new"
		if err := os.Rename(tmpFile, newPath); err != nil {
			return errors.FailedTo("move new binary", err)
		}
		updateScript = binaryPath + ".update.bat"
	}

	// Create batch script to complete the update after goenv exits
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
	if elevated {
		startCmd = elevatedStartCommand(updateScript)
	}
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
