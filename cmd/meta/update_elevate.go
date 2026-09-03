package meta

import (
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/cmdutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

// canReplaceBinary reports whether the current user can replace binaryPath.
//
// Replacement is a rename within the parent directory, so directory write
// access is what decides it — the binary's own mode is irrelevant. Probing with
// a real file also covers read-only mounts, ACLs, and immutable-flag cases that
// a mode/uid comparison would miss, and it avoids the ETXTBSY that opening the
// running executable for writing returns on Linux.
func canReplaceBinary(binaryPath string) error {
	probe, err := os.CreateTemp(filepath.Dir(binaryPath), ".goenv-update-probe-*")
	if err != nil {
		return err
	}
	name := probe.Name()
	probe.Close()
	os.Remove(name)
	return nil
}

// packageManager identifies the tool that owns binaryPath, when goenv was
// installed by one.
//
// Overwriting a package manager's file leaves its metadata describing a version
// that is no longer installed: `brew list --versions goenv` keeps reporting the
// old one, and the next `brew upgrade` silently reverts the update. Elevating
// to do it anyway turns a recoverable mistake into a root-owned one, so these
// installs are refused rather than escalated.
func packageManager(binaryPath string) (name, upgradeCmd string) {
	// Not filepath.ToSlash: that is a no-op off Windows, and these paths are
	// also matched in tests running on Unix.
	p := strings.ToLower(strings.ReplaceAll(binaryPath, `\`, "/"))

	switch {
	case strings.Contains(p, "/cellar/"), strings.Contains(p, "/homebrew/"), strings.Contains(p, "/linuxbrew/"):
		return "Homebrew", "brew upgrade goenv"
	case strings.HasPrefix(p, "/nix/store/"):
		return "Nix", "nix profile upgrade goenv"
	case strings.Contains(p, "/scoop/"):
		return "Scoop", "scoop update goenv"
	case strings.Contains(p, "/chocolatey/"):
		return "Chocolatey", "choco upgrade goenv"
	case strings.Contains(p, "/.asdf/"):
		return "asdf", "asdf install goenv latest"
	}

	return "", ""
}

// findElevator returns the privilege escalation helper available on this
// system, or an error naming what was looked for.
func findElevator() (string, error) {
	candidates := []string{"sudo", "doas"}
	if utils.IsWindows() {
		candidates = []string{"powershell", "pwsh"}
	}

	for _, name := range candidates {
		if path, err := exec.LookPath(name); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("none of %s found in PATH", strings.Join(candidates, ", "))
}

// runElevated executes the helper with stdin attached so it can prompt for a
// password or raise a UAC dialog.
func runElevated(elevator string, args []string, stdout, stderr io.Writer) error {
	cmd := exec.Command(elevator, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

// elevatedStartCommand launches script behind a UAC prompt.
//
// Windows has no sudo to exec into: a non-elevated process can only raise
// privileges by asking the shell to start a new one, which is what
// Start-Process -Verb RunAs does.
func elevatedStartCommand(script string) *exec.Cmd {
	shell := "powershell"
	if path, err := exec.LookPath("pwsh"); err == nil {
		shell = path
	}

	// The path is goenv's own, but it can still contain spaces (%TEMP% under a
	// user profile) and PowerShell single quotes escape by doubling.
	quoted := "'\"" + strings.ReplaceAll(script, "'", "''") + "\"'"

	return exec.Command(shell, "-NoProfile", "-NonInteractive", "-Command",
		fmt.Sprintf("Start-Process -FilePath cmd.exe -ArgumentList '/c',%s -Verb RunAs", quoted))
}

// installElevated replaces target with staged using elevated privileges.
//
// Only this copy runs as root. The download, checksum verification, and archive
// extraction all happen as the invoking user, so an unverified release is never
// executed with elevated privileges and root only ever reads a file this process
// created in a 0700 directory it owns.
//
// The copy lands on a temporary name in the destination directory and is then
// renamed over the target, so an interrupted install cannot leave a truncated
// executable in place.
func installElevated(elevator, staged, target string, stdout, stderr io.Writer) error {
	pending := target + ".new"

	if err := runElevated(elevator, []string{"install", "-m", "0755", staged, pending}, stdout, stderr); err != nil {
		return fmt.Errorf("copy new binary into %s: %w", filepath.Dir(target), err)
	}

	if err := runElevated(elevator, []string{"mv", "-f", pending, target}, stdout, stderr); err != nil {
		// Best effort: leaving the staged copy behind would shadow nothing, but
		// it would confuse the next run.
		_ = runElevated(elevator, []string{"rm", "-f", pending}, io.Discard, io.Discard)
		return fmt.Errorf("move new binary into place: %w", err)
	}

	return nil
}

// elevationInstructions is the manual fallback shown when goenv cannot or may
// not elevate on the user's behalf.
func elevationInstructions(binaryPath string) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf("goenv cannot write to %s\n\n", filepath.Dir(binaryPath)))
	b.WriteString("To fix this:\n")

	if utils.IsWindows() {
		b.WriteString("  • Re-run 'goenv update' from a PowerShell session started as Administrator, or\n")
		b.WriteString("  • Install goenv to a user-writeable path like %LOCALAPPDATA%\\goenv\n")
		b.WriteString("    (e.g., C:\\Users\\YourName\\AppData\\Local\\goenv)\n")
	} else {
		b.WriteString("  • Re-run with elevated permissions: sudo goenv update\n")
		b.WriteString("  • Or install goenv to a user-writeable path (e.g., ~/.local/bin/)\n")
	}

	b.WriteString("\nAlternatively, download and install manually:\n")
	b.WriteString("  • https://github.com/go-nv/goenv/releases")

	return b.String()
}

// confirmElevation checks that escalation is possible and that a human agreed
// to it. Escalation is never implicit: without a terminal to answer on, the
// manual instructions are returned instead, so a scripted or CI invocation
// cannot end up writing to a system directory unattended.
func confirmElevation(cmd *cobra.Command, binaryPath, latestVersion string) error {
	elevator, err := findElevator()
	if err != nil {
		return fmt.Errorf("%s\n\n(no elevation helper available: %v)", elevationInstructions(binaryPath), err)
	}

	out := cmd.OutOrStdout()
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%s%s is not writable by the current user.\n", utils.Emoji("🔒 "), filepath.Dir(binaryPath))
	fmt.Fprintf(out, "   goenv %s will be downloaded and verified as you, then installed using %s.\n",
		latestVersion, filepath.Base(elevator))
	fmt.Fprintln(out)

	ictx := cmdutil.NewInteractiveContext(cmd)
	if !cmdutil.CanPrompt(ictx, os.Stdin) {
		return fmt.Errorf("%s", elevationInstructions(binaryPath))
	}

	if !ictx.Confirm(fmt.Sprintf("Install goenv %s to %s with elevated privileges?", latestVersion, binaryPath), false) {
		return fmt.Errorf("update cancelled")
	}

	return nil
}

// replaceElevated performs the privileged half of the update.
func replaceElevated(cmd *cobra.Command, staged, binaryPath, backupPath, currentVersion, latestVersion string) error {
	elevator, err := findElevator()
	if err != nil {
		return fmt.Errorf("%s", elevationInstructions(binaryPath))
	}

	fmt.Fprintf(cmd.OutOrStdout(), "%sReplacing binary (elevated)...\n", utils.Emoji("🔄 "))

	if utils.IsWindows() {
		return replaceWindowsBinary(cmd, staged, binaryPath, backupPath, currentVersion, latestVersion, true)
	}

	return installElevated(elevator, staged, binaryPath, cmd.OutOrStdout(), cmd.OutOrStderr())
}
