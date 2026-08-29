//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// --- Issue #438: "goenv latest" must print, never mutate the project ---------

// TestLatest_DoesNotWriteVersionFile guards issue #438.
//
// "goenv latest" used to be swallowed by the version-shorthand rewrite in
// cmd/root.go, which turned it into "goenv local latest" and silently created
// a .go-version file in whatever directory the user happened to be in.
func TestLatest_DoesNotWriteVersionFile(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.fakeVersion("1.24.1")

	res := e.run("latest")

	if !res.Succeeded() {
		t.Fatalf("goenv latest failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	requireContains(t, res.Stdout, "1.24.1", "goenv latest should print the newest installed version")

	if fileExists(e.workPath(".go-version")) {
		t.Fatal("goenv latest created a .go-version file; it must only print (issue #438)")
	}
}

// TestLatest_WithPrefix filters to a version line.
func TestLatest_WithPrefix(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.fakeVersion("1.24.1")

	res := e.run("latest", "1.23")

	if !res.Succeeded() {
		t.Fatalf("goenv latest 1.23 failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	requireContains(t, res.Stdout, "1.23.6", "prefix should select the newest matching installed version")
	requireNotContains(t, res.Stdout, "1.24.1", "prefix must not select outside the requested line")

	if fileExists(e.workPath(".go-version")) {
		t.Fatal("goenv latest <prefix> created a .go-version file (issue #438)")
	}
}

// TestVersionShorthand_StillWritesVersionFile pins the behaviour the #438 fix
// had to preserve: a bare version argument is still shorthand for "goenv local".
func TestVersionShorthand_StillWritesVersionFile(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6")

	res := e.run("1.23.6")

	if !res.Succeeded() {
		t.Fatalf("goenv 1.23.6 failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	if !fileExists(e.workPath(".go-version")) {
		t.Fatal("version shorthand should still write .go-version")
	}
}

// --- Issue #593: "goenv init -" must emit shell completions -----------------

// TestInit_EmitsCompletions guards issue #593, where "eval \"$(goenv init -)\""
// produced no completion code at all on installations without an on-disk
// completions directory (Homebrew being the common case).
func TestInit_EmitsCompletions(t *testing.T) {
	tests := []struct {
		shell    string
		fragment string
	}{
		{"bash", "complete -F _goenv goenv"},
		{"zsh", "compctl -K _goenv_compctl goenv"},
		{"fish", "complete -c goenv"},
		{"powershell", "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			e := newEnv(t)

			res := e.run("init", "-", tt.shell)

			if !res.Succeeded() {
				t.Fatalf("goenv init - %s failed (exit %d):\n%s", tt.shell, res.ExitCode, res.Output())
			}
			requireContains(t, res.Stdout, tt.fragment,
				"goenv init - must emit completion registration (issue #593)")
		})
	}
}

// TestInit_EvaluatesCleanlyInRealShell runs the emitted script through the real
// shell. Emitting completion code that does not parse would be worse than
// emitting none, so this closes the loop on the #593 fix.
func TestInit_EvaluatesCleanlyInRealShell(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("bash/zsh evaluation is not applicable on Windows")
	}

	for _, shell := range []string{"bash", "zsh"} {
		t.Run(shell, func(t *testing.T) {
			if _, err := lookPath(shell); err != nil {
				t.Skipf("%s not available on this runner", shell)
			}

			e := newEnv(t)
			e.fakeVersion("1.23.6")

			res := e.run("init", "-", shell)
			if !res.Succeeded() {
				t.Fatalf("goenv init - %s failed:\n%s", shell, res.Output())
			}

			scriptPath := filepath.Join(t.TempDir(), "init_script")
			if err := os.WriteFile(scriptPath, []byte(res.Stdout), 0o644); err != nil {
				t.Fatalf("failed to write init script: %v", err)
			}

			// -i is required: the zsh completion registration is guarded on an
			// interactive shell, matching how users actually load it.
			out, err := runShell(t, shell, "-i", "-c",
				"set -e; source '"+scriptPath+"'; echo INIT_OK")
			if err != nil {
				t.Fatalf("emitted %s init script failed to evaluate: %v\n%s", shell, err, out)
			}
			requireContains(t, out, "INIT_OK",
				"the init script emitted by goenv must evaluate cleanly in "+shell)
		})
	}
}

// --- Issues #542 / #572: no runaway recursion -------------------------------

// TestAutoInstall_DoesNotLoop guards the infinite loop reported in issues #572
// and #542.
//
// Root cause: the auto-install path called installCmd.Execute(), and cobra's
// Execute() always dispatches from the root command, so the root command
// re-entered its own auto-install branch forever, printing
// "Auto-installing Go x.y.z (from go.mod)..." without end.
func TestAutoInstall_DoesNotLoop(t *testing.T) {
	e := newEnv(t)
	e.Set("GOENV_AUTO_INSTALL", "1")

	// A version that cannot resolve makes the install fail fast, so the test
	// measures re-entrancy rather than download speed.
	e.writeFile("go.mod", "module example.com/looptest\n\ngo 0.0.1\n")

	// No arguments: bare "goenv" is the invocation that enters the smart
	// detection / auto-install branch in cmd/root.go.
	res := e.runWithTimeout(60 * time.Second)

	if res.TimedOut {
		t.Fatalf("goenv did not terminate: the auto-install path is looping (issues #572, #542)\n"+
			"--- output (truncated) ---\n%s", truncate(res.Output(), 2000))
	}

	// Exiting non-zero is fine — the version genuinely cannot be installed.
	// Looping is not. One attempt means one message.
	if got := strings.Count(res.Output(), "Auto-installing Go"); got > 1 {
		t.Fatalf("auto-install ran %d times for a single invocation; expected at most 1 (issues #572, #542)\n"+
			"--- output (truncated) ---\n%s", got, truncate(res.Output(), 2000))
	}
}

// TestRehash_DoesNotCreateGoenvShim guards the recursion trigger behind #542.
//
// A "goenv" shim inside the shims directory shadows the real binary, because
// the shims directory is first on PATH and every shim invokes "goenv exec".
func TestRehash_DoesNotCreateGoenvShim(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6", "goenv")
	e.run("global", "1.23.6")

	res := e.run("rehash")
	if !res.Succeeded() {
		t.Fatalf("goenv rehash failed (exit %d):\n%s", res.ExitCode, res.Output())
	}

	for _, name := range []string{"goenv", exeName("goenv"), "goenv.exe"} {
		if fileExists(e.path("shims", name)) {
			t.Fatalf("rehash created a %q shim; it shadows the real binary and causes "+
				"infinite recursion (issue #542)", name)
		}
	}

	// Sanity check that rehash did its actual job.
	if !fileExists(e.path("shims", exeName("go"))) {
		t.Fatal("rehash did not create the expected 'go' shim")
	}
}

// TestDoctor_DetectsGoenvShim verifies goenv reports the recursion trigger from
// issue #542 rather than leaving users to discover it via a hung terminal.
func TestDoctor_DetectsGoenvShim(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.run("global", "1.23.6")
	e.run("rehash")

	// Plant the offending shim, mimicking a leftover v2 install.
	shimPath := e.path("shims", exeName("goenv"))
	if err := os.MkdirAll(filepath.Dir(shimPath), 0o755); err != nil {
		t.Fatalf("failed to create shims dir: %v", err)
	}
	if err := os.WriteFile(shimPath, []byte("#!/usr/bin/env bash\nexec goenv exec \"$@\"\n"), 0o755); err != nil {
		t.Fatalf("failed to plant goenv shim: %v", err)
	}

	res := e.run("doctor")
	output := res.Output()

	// doctor exits non-zero when it finds problems, which is expected here.
	if !strings.Contains(strings.ToLower(output), "goenv") ||
		!strings.Contains(strings.ToLower(output), "shim") {
		t.Fatalf("doctor output does not mention the goenv shim:\n%s", output)
	}
	if !mentionsRecursionRisk(output) {
		t.Fatalf("doctor did not flag the recursion-causing goenv shim (issue #542):\n%s", output)
	}
}

// --- Issue #581: default-tools verify must look where install writes --------

// TestToolsVerify_FindsInstalledTool guards the false "not installed" reports
// described in issue #581. InstallTools sets GOPATH to the version directory,
// so binaries land in versions/<version>/bin, but verify only looked in
// versions/<version>/gopath/bin.
func TestToolsVerify_FindsInstalledTool(t *testing.T) {
	e := newEnv(t)
	version := "1.23.6"
	e.fakeVersion(version)

	// Configure a single tool so the assertion does not depend on the shipped
	// default list.
	e.writeGoenvFile("default-tools.yaml", ""+
		"enabled: true\n"+
		"tools:\n"+
		"  - name: gopls\n"+
		"    package: golang.org/x/tools/gopls\n"+
		"    binary: gopls\n")

	// Place the binary exactly where "go install" puts it under goenv's own
	// GOPATH choice.
	writeFakeExecutable(t, e.path("versions", version, "bin", "gopls"))

	res := e.run("tools", "default-tools", "verify", version)
	if !res.Succeeded() {
		t.Fatalf("tools verify failed (exit %d):\n%s", res.ExitCode, res.Output())
	}

	requireNotContains(t, res.Output(), "gopls (not installed)",
		"verify reported an installed tool as missing (issue #581)")
	requireContains(t, res.Output(), "1 installed",
		"verify should count the installed tool")
}

// --- Shim generation across operating systems -------------------------------

// TestRehash_GeneratesPlatformAppropriateShims checks the generated shim is
// valid for the host OS. Windows batch shims have parsing constraints that are
// easy to break from a macOS or Linux workstation (issue #555).
func TestRehash_GeneratesPlatformAppropriateShims(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.run("global", "1.23.6")

	if res := e.run("rehash"); !res.Succeeded() {
		t.Fatalf("goenv rehash failed:\n%s", res.Output())
	}

	shimPath := e.path("shims", exeName("go"))
	content, err := os.ReadFile(shimPath)
	if err != nil {
		t.Fatalf("failed to read generated shim %s: %v", shimPath, err)
	}
	shim := string(content)

	requireContains(t, shim, "goenv exec", "every shim must delegate to goenv exec")

	if runtime.GOOS == "windows" {
		requireContains(t, shim, "@echo off", "Windows shims must suppress echo")
		// Issue #555: a goto/label inside a parenthesized if block breaks the
		// CMD parser at runtime.
		if strings.Contains(shim, "goto :") && strings.Contains(shim, ")") {
			assertNoGotoInsideParens(t, shim)
		}
	} else {
		requireContains(t, shim, "#!/usr/bin/env bash", "Unix shims need a shebang")

		// The shim must actually be executable, not merely present.
		info, err := os.Stat(shimPath)
		if err != nil {
			t.Fatalf("failed to stat shim: %v", err)
		}
		if info.Mode().Perm()&0o111 == 0 {
			t.Fatalf("generated shim %s is not executable (mode %v)", shimPath, info.Mode().Perm())
		}
	}
}

// --- Baseline health --------------------------------------------------------

// TestDoctor_CleanRootHasNoErrors ensures a freshly created goenv root does not
// report hard errors. This is the check issue #542 wanted: users should be able
// to trust doctor as the first troubleshooting step.
func TestDoctor_CleanRootHasNoErrors(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6")

	if res := e.run("global", "1.23.6"); !res.Succeeded() {
		t.Fatalf("goenv global failed:\n%s", res.Output())
	}
	if res := e.run("rehash"); !res.Succeeded() {
		t.Fatalf("goenv rehash failed:\n%s", res.Output())
	}

	res := e.run("doctor")

	// Warnings are acceptable (PATH is intentionally not configured in the
	// harness); crashes and panics are not.
	requireNotContains(t, res.Output(), "panic:", "doctor must not panic")
	if res.TimedOut {
		t.Fatal("goenv doctor timed out")
	}
}

// TestCoreCommands_Smoke exercises the commands users touch first. It is a
// cheap guard against a command that fails to run at all on one OS.
func TestCoreCommands_Smoke(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion("1.23.6")

	commands := [][]string{
		{"--version"},
		{"--help"},
		{"versions"},
		{"list"},
		{"root"},
		{"commands"},
		{"init", "-", "bash"},
	}

	for _, args := range commands {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			res := e.run(args...)
			if res.TimedOut {
				t.Fatalf("goenv %s timed out", strings.Join(args, " "))
			}
			requireNotContains(t, res.Output(), "panic:", "command panicked")
			if !res.Succeeded() {
				t.Fatalf("goenv %s failed (exit %d):\n%s",
					strings.Join(args, " "), res.ExitCode, res.Output())
			}
		})
	}
}

// --- Issue #578: the shared module cache must be visible and reclaimable ----

// TestSharedCache_IsVisibleAndReclaimable guards issue #578, where a
// devcontainer image grew by ~2.5GB and the reporter had to dig through the
// filesystem with docker history to find the cause.
//
// Two separate defects made that necessary: "cache status" never displayed the
// shared module cache (it was counted in the total but not listed), and
// "cache clean" refused to do anything when no versions were installed — even
// though the shared cache lives outside versions/ and survives them.
func TestSharedCache_IsVisibleAndReclaimable(t *testing.T) {
	e := newEnv(t)

	// A shared cache with no installed versions: exactly the state a container
	// image lands in after the versions are pruned but the cache is not.
	blob := e.path("shared", "go-mod", "cache", "download", "blob.bin")
	if err := os.MkdirAll(filepath.Dir(blob), 0o755); err != nil {
		t.Fatalf("failed to create shared cache dir: %v", err)
	}
	// 2 MiB is enough to assert on without slowing the suite down.
	if err := os.WriteFile(blob, make([]byte, 2<<20), 0o644); err != nil {
		t.Fatalf("failed to write cache blob: %v", err)
	}

	status := e.run("cache", "status")
	if !status.Succeeded() {
		t.Fatalf("goenv cache status failed (exit %d):\n%s", status.ExitCode, status.Output())
	}

	requireNotContains(t, status.Output(), "No caches found",
		"cache status hid a populated shared module cache (issue #578)")
	requireNotContains(t, status.Output(), "No Go versions installed",
		"cache status must report the shared cache even with no versions installed (issue #578)")
	requireContains(t, status.Output(), "Shared",
		"cache status must list the shared module cache as its own entry (issue #578)")

	// The advice cache status prints must actually work.
	clean := e.run("cache", "clean", "mod", "--force")
	if !clean.Succeeded() {
		t.Fatalf("goenv cache clean mod failed (exit %d):\n%s", clean.ExitCode, clean.Output())
	}

	if fileExists(blob) {
		t.Fatalf("goenv cache clean mod did not reclaim the shared module cache (issue #578)\n"+
			"--- output ---\n%s", clean.Output())
	}
}

// --- Network-gated: the full install journey --------------------------------

// TestInstallAndUse_RealVersion is the complete user journey: install a real Go
// toolchain, select it for a directory, and run it through the shim.
//
// This is the only test that proves version switching actually reaches the
// toolchain, which is what issues #433 and #367 report as broken.
func TestInstallAndUse_RealVersion(t *testing.T) {
	skipWithoutNetwork(t)

	const version = "1.23.6"

	e := newEnv(t)

	res := e.runWithTimeout(10*time.Minute, "install", version)
	if !res.Succeeded() {
		t.Fatalf("goenv install %s failed (exit %d):\n%s", version, res.ExitCode, res.Output())
	}

	if res := e.run("local", version); !res.Succeeded() {
		t.Fatalf("goenv local failed:\n%s", res.Output())
	}

	// version-name is the resolution goenv exposes to shims.
	res = e.run("version-name")
	if !res.Succeeded() {
		t.Fatalf("goenv version-name failed:\n%s", res.Output())
	}
	requireContains(t, res.Stdout, version, "the local version must be what goenv resolves")

	// And the real toolchain must run through goenv exec.
	res = e.runWithTimeout(2*time.Minute, "exec", "go", "version")
	if !res.Succeeded() {
		t.Fatalf("goenv exec go version failed:\n%s", res.Output())
	}
	requireContains(t, res.Stdout, version,
		"goenv exec must run the selected toolchain (issues #433, #367)")
}
