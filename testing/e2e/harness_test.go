//go:build e2e

// Package e2e contains end-to-end tests that exercise a real goenv binary
// against a real, isolated GOENV_ROOT on the host operating system.
//
// These tests are the outer ring of the test pyramid: unit tests verify
// functions, these verify user journeys. They exist because several shipped
// regressions (see the issue references on individual tests) were invisible to
// unit tests — they only appeared when the real binary ran against a real
// filesystem, PATH and shell.
//
// Run with:
//
//	go test -tags e2e ./testing/e2e/...
//
// Tests that need to download a Go toolchain are additionally gated behind
// GOENV_E2E_NETWORK=1 so the default suite stays fast and hermetic.
package e2e

import (
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"
)

// goenvBinary is the absolute path to the binary under test, built once by
// TestMain and shared by every test.
var goenvBinary string

// defaultTimeout bounds every command. It must stay short enough that a
// runaway process (the infinite-loop class of bug in issues #542 and #572)
// fails the test rather than hanging the CI job.
const defaultTimeout = 90 * time.Second

func TestMain(m *testing.M) {
	binary, cleanup, err := buildGoenv()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: failed to build goenv: %v\n", err)
		os.Exit(1)
	}
	goenvBinary = binary

	printSuiteDiagnostics()

	code := m.Run()
	cleanup()
	os.Exit(code)
}

// printSuiteDiagnostics records the environment the suite is running in.
//
// When this suite fails on a runner the author cannot reproduce — Windows in
// particular — the first questions are always "which OS, which shell, what was
// on PATH". Printing it once up front means the answer is already in the log
// instead of requiring another CI round-trip to find out.
func printSuiteDiagnostics() {
	fmt.Printf("=== goenv e2e suite ===\n")
	fmt.Printf("GOOS/GOARCH  : %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Printf("Go version   : %s\n", runtime.Version())
	fmt.Printf("binary       : %s\n", goenvBinary)

	if info, err := os.Stat(goenvBinary); err == nil {
		fmt.Printf("binary size  : %d bytes, mode %s\n", info.Size(), info.Mode())
	}

	// The sandbox strips goenv directories from PATH; show what survived so a
	// "command not found" in a shim test is immediately explainable.
	fmt.Printf("sandbox PATH :\n")
	for _, dir := range filepath.SplitList(sandboxPath()) {
		fmt.Printf("  %s\n", dir)
	}

	for _, name := range []string{"bash", "zsh", "pwsh", "cmd"} {
		if path, err := exec.LookPath(name); err == nil {
			fmt.Printf("shell %-5s  : %s\n", name, path)
		} else {
			fmt.Printf("shell %-5s  : (not found)\n", name)
		}
	}

	fmt.Printf("network tests: %v\n", os.Getenv("GOENV_E2E_NETWORK") == "1")
	fmt.Printf("=======================\n")
}

// buildGoenv compiles the goenv binary from the repository root.
func buildGoenv() (string, func(), error) {
	repoRoot, err := findRepoRoot()
	if err != nil {
		return "", nil, err
	}

	tmpDir, err := os.MkdirTemp("", "goenv-e2e-bin-")
	if err != nil {
		return "", nil, err
	}
	cleanup := func() { os.RemoveAll(tmpDir) }

	name := "goenv"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	out := filepath.Join(tmpDir, name)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	defer cancel()

	cmd := exec.CommandContext(ctx, "go", "build", "-o", out, ".")
	cmd.Dir = repoRoot
	if output, err := cmd.CombinedOutput(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("go build failed: %w\n%s", err, output)
	}

	return out, cleanup, nil
}

// findRepoRoot walks up from the working directory looking for go.mod.
func findRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", errors.New("could not locate repository root (no go.mod found)")
		}
		dir = parent
	}
}

// env is an isolated goenv installation plus a working directory, scoped to a
// single test.
type env struct {
	t    *testing.T
	Root string // GOENV_ROOT
	Dir  string // working directory for commands
	Home string // isolated HOME/USERPROFILE
	// allowNetwork opts out of GOENV_OFFLINE for tests that must reach the
	// network. Off by default so the suite is hermetic in fact, not just in
	// its documentation.
	allowNetwork bool
	// Extra environment variables layered on top of the isolated defaults.
	Extra map[string]string
}

// newEnv creates a fresh, isolated goenv installation.
//
// Every GOENV_* variable inherited from the developer's shell or the CI runner
// is stripped. Without this, a contributor's own goenv setup leaks into the
// tests and makes results irreproducible — which is the same class of problem
// issue #570 describes.
//
// HOME is redirected too, and that matters more than it looks: goenv resolves
// tool binaries from "$HOME/go/<version>/bin" (see Config.LegacyHomeGopathBin).
// With the real HOME, a developer's own installed tools silently satisfy
// assertions and tests pass for the wrong reason.
func newEnv(t *testing.T) *env {
	t.Helper()

	base := t.TempDir()
	root := filepath.Join(base, "goenv-root")
	dir := filepath.Join(base, "work")
	home := filepath.Join(base, "home")

	for _, d := range []string{root, dir, home} {
		if err := os.MkdirAll(d, 0o755); err != nil {
			t.Fatalf("failed to create %s: %v", d, err)
		}
	}

	return &env{
		t:     t,
		Root:  root,
		Dir:   dir,
		Home:  home,
		Extra: map[string]string{},
	}
}

// Set adds an environment variable for subsequent commands.
func (e *env) Set(key, value string) *env {
	e.Extra[key] = value
	return e
}

// AllowNetwork lets a test reach the network, disabling GOENV_OFFLINE.
func (e *env) AllowNetwork() *env {
	e.allowNetwork = true
	return e
}

// sandboxPath returns the host PATH with any goenv directories removed.
//
// Generated shims dispatch through a bare "goenv exec", resolved via PATH. On a
// developer machine PATH contains ~/.goenv/shims and ~/.goenv/bin, so a shim
// executed by a test would run the host's installed goenv against the host's
// configuration instead of the binary under test — passing or failing for
// reasons that have nothing to do with the change being tested.
func sandboxPath() string {
	hostPath := os.Getenv("PATH")
	if hostPath == "" {
		return ""
	}

	home, err := os.UserHomeDir()
	var goenvRoot string
	if err == nil {
		goenvRoot = filepath.Join(home, ".goenv")
	}
	if env := os.Getenv("GOENV_ROOT"); env != "" {
		goenvRoot = env
	}

	kept := make([]string, 0, 16)
	for _, dir := range filepath.SplitList(hostPath) {
		if goenvRoot != "" && strings.HasPrefix(filepath.Clean(dir), filepath.Clean(goenvRoot)) {
			continue
		}
		kept = append(kept, dir)
	}

	return strings.Join(kept, string(filepath.ListSeparator))
}

// result is the outcome of running a goenv command.
type result struct {
	Stdout   string
	Stderr   string
	Combined string
	ExitCode int
	Err      error
	TimedOut bool
	// Args and Duration exist for diagnostics: a CI log needs to show what was
	// run and how long it took, not just what came back.
	Args     []string
	Duration time.Duration
}

// Output returns stdout and stderr joined, which is what most assertions want
// since goenv writes advice to stderr and data to stdout.
func (r result) Output() string { return r.Combined }

// Succeeded reports whether the command exited zero.
func (r result) Succeeded() bool { return r.ExitCode == 0 && !r.TimedOut }

// run executes the goenv binary with the given arguments.
func (e *env) run(args ...string) result {
	e.t.Helper()
	return e.runWithTimeout(defaultTimeout, args...)
}

// runWithTimeout executes goenv, killing it after the supplied duration.
//
// A timeout is treated as a test-visible outcome rather than a hang so that
// runaway-recursion regressions produce a clear failure.
func (e *env) runWithTimeout(timeout time.Duration, args ...string) result {
	e.t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, goenvBinary, args...)
	cmd.Dir = e.Dir
	cmd.Env = e.environ()

	var stdout, stderr, combined syncBuffer
	cmd.Stdout = multiWriter(&stdout, &combined)
	cmd.Stderr = multiWriter(&stderr, &combined)

	started := time.Now()
	err := cmd.Run()
	elapsed := time.Since(started)

	res := result{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Combined: combined.String(),
		Err:      err,
		Args:     args,
		Duration: elapsed,
	}

	if ctx.Err() == context.DeadlineExceeded {
		res.TimedOut = true
		res.ExitCode = -1
		e.logResult(res)
		return res
	}

	var exitErr *exec.ExitError
	switch {
	case err == nil:
		res.ExitCode = 0
	case errors.As(err, &exitErr):
		res.ExitCode = exitErr.ExitCode()
	default:
		res.ExitCode = -1
	}

	e.logResult(res)
	return res
}

// logResult records every command under -v.
//
// These tests are frequently diagnosed from a CI log on an operating system the
// author cannot reproduce locally — Windows above all. A failure that only says
// "expected X, got Y" forces a blind guess-and-push cycle, so the full
// invocation, timing, exit status and both streams are recorded for every
// command, not only failing ones. The preceding commands are usually what
// explain the failure.
func (e *env) logResult(res result) {
	e.t.Helper()

	if !testing.Verbose() {
		return
	}

	e.t.Logf("$ goenv %s  (exit=%d, %s)", strings.Join(res.Args, " "), res.ExitCode, res.Duration.Round(time.Millisecond))
	if res.TimedOut {
		e.t.Logf("  !! TIMED OUT")
	}
	if out := strings.TrimRight(res.Stdout, "\n"); out != "" {
		e.t.Logf("  stdout | %s", strings.ReplaceAll(out, "\n", "\n  stdout | "))
	}
	if errOut := strings.TrimRight(res.Stderr, "\n"); errOut != "" {
		e.t.Logf("  stderr | %s", strings.ReplaceAll(errOut, "\n", "\n  stderr | "))
	}
}

// Diagnose dumps the state of the sandbox. Call it from a failure path to make
// a CI log actionable without a local reproduction.
func (e *env) Diagnose() {
	e.t.Helper()

	var b strings.Builder
	fmt.Fprintf(&b, "\n===== e2e sandbox diagnostics =====\n")
	fmt.Fprintf(&b, "GOOS/GOARCH : %s/%s\n", runtime.GOOS, runtime.GOARCH)
	fmt.Fprintf(&b, "binary      : %s\n", goenvBinary)
	fmt.Fprintf(&b, "GOENV_ROOT  : %s\n", e.Root)
	fmt.Fprintf(&b, "workdir     : %s\n", e.Dir)
	fmt.Fprintf(&b, "HOME        : %s\n", e.Home)

	if len(e.Extra) > 0 {
		fmt.Fprintf(&b, "extra env   :\n")
		keys := make([]string, 0, len(e.Extra))
		for k := range e.Extra {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			fmt.Fprintf(&b, "  %s=%s\n", k, e.Extra[k])
		}
	}

	fmt.Fprintf(&b, "--- tree of GOENV_ROOT ---\n")
	writeTree(&b, e.Root)
	fmt.Fprintf(&b, "--- tree of workdir ---\n")
	writeTree(&b, e.Dir)
	fmt.Fprintf(&b, "==================================")

	e.t.Log(b.String())
}

// writeTree lists a directory tree with sizes and modes, which is what
// identifies a missing shim, a wrong extension, or a non-executable file.
func writeTree(w io.Writer, root string) {
	entries := 0
	err := filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			fmt.Fprintf(w, "  !! %s: %v\n", path, err)
			return nil
		}
		entries++
		if entries > 200 {
			return filepath.SkipAll
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		if rel == "." {
			return nil
		}
		if info.IsDir() {
			fmt.Fprintf(w, "  %s%c\n", rel, filepath.Separator)
			return nil
		}
		fmt.Fprintf(w, "  %-52s %7d  %s\n", rel, info.Size(), info.Mode())
		return nil
	})
	if err != nil {
		fmt.Fprintf(w, "  !! walk failed: %v\n", err)
	}
	if entries == 0 {
		fmt.Fprintf(w, "  (empty or missing)\n")
	}
}

// environ builds a deliberately minimal environment: enough of the host
// environment for a process to run, plus only the GOENV_* variables this test
// asked for. HOME is redirected into the test's sandbox.
func (e *env) environ() []string {
	// Deliberately excludes HOME, USERPROFILE and PATH; those are set below.
	// LOCALAPPDATA/APPDATA are also redirected into the sandbox on Windows,
	// since anything writing under %APPDATA% would otherwise escape it.
	passthrough := []string{"TMPDIR", "TEMP", "TMP",
		"SystemRoot", "COMSPEC", "PATHEXT", "SystemDrive", "windir",
		"ProgramData", "ProgramFiles", "ProgramFiles(x86)",
		"USERNAME", "NUMBER_OF_PROCESSORS", "PROCESSOR_ARCHITECTURE"}

	out := make([]string, 0, len(passthrough)+len(e.Extra)+12)
	for _, key := range passthrough {
		if value, ok := os.LookupEnv(key); ok {
			out = append(out, key+"="+value)
		}
	}

	// PATH is handled separately: goenv directories are stripped from it.
	out = append(out, "PATH="+sandboxPath())

	out = append(out, "GOENV_ROOT="+e.Root)
	out = append(out, "HOME="+e.Home)
	out = append(out, "USERPROFILE="+e.Home)
	out = append(out, "LOCALAPPDATA="+filepath.Join(e.Home, "AppData", "Local"))
	out = append(out, "APPDATA="+filepath.Join(e.Home, "AppData", "Roaming"))
	// Keep output deterministic and parseable.
	out = append(out, "NO_COLOR=1")

	// Keep Go's own caches inside the sandbox. Without these, a test that shells
	// out to a toolchain writes into the developer's real ~/go and ~/Library/Caches.
	out = append(out, "GOPATH="+filepath.Join(e.Home, "go"))
	out = append(out, "GOCACHE="+filepath.Join(e.Home, "go-build"))
	out = append(out, "GOMODCACHE="+filepath.Join(e.Home, "go", "pkg", "mod"))
	// XDG paths, so anything following the spec also stays inside the sandbox.
	out = append(out, "XDG_CACHE_HOME="+filepath.Join(e.Home, ".cache"))
	out = append(out, "XDG_CONFIG_HOME="+filepath.Join(e.Home, ".config"))
	out = append(out, "XDG_DATA_HOME="+filepath.Join(e.Home, ".local", "share"))

	// Offline by default. Several commands (install, and anything that resolves
	// a version spec) otherwise reach api/golang.org, which makes the suite
	// slow, flaky, and dependent on the network despite being advertised as
	// hermetic. Tests that genuinely need the network opt in via networkEnv.
	if !e.allowNetwork {
		out = append(out, "GOENV_OFFLINE=1")
	}

	for key, value := range e.Extra {
		out = append(out, key+"="+value)
	}

	return out
}

// path joins a path relative to the isolated GOENV_ROOT.
func (e *env) path(parts ...string) string {
	return filepath.Join(append([]string{e.Root}, parts...)...)
}

// workPath joins a path relative to the working directory.
func (e *env) workPath(parts ...string) string {
	return filepath.Join(append([]string{e.Dir}, parts...)...)
}

// writeFile writes a file inside the working directory.
func (e *env) writeFile(name, content string) string {
	e.t.Helper()
	full := e.workPath(name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		e.t.Fatalf("failed to create directory for %s: %v", name, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		e.t.Fatalf("failed to write %s: %v", name, err)
	}
	return full
}

// fakeVersion creates a version directory that looks installed enough for
// shim generation, version listing and resolution, without downloading Go.
func (e *env) fakeVersion(version string, extraBinaries ...string) string {
	e.t.Helper()

	binDir := e.path("versions", version, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		e.t.Fatalf("failed to create version dir: %v", err)
	}

	for _, name := range append([]string{"go", "gofmt"}, extraBinaries...) {
		writeFakeExecutable(e.t, filepath.Join(binDir, name))
	}

	return e.path("versions", version)
}

// writeFakeExecutable creates a stub executable appropriate for the platform.
func writeFakeExecutable(t *testing.T, path string) {
	t.Helper()

	content := "#!/usr/bin/env bash\necho fake\n"
	mode := os.FileMode(0o755)
	if runtime.GOOS == "windows" {
		path += ".exe"
		// Content is irrelevant on Windows; goenv only checks for existence
		// and an executable extension.
		content = "fake"
		mode = 0o644
	}

	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatalf("failed to write fake executable %s: %v", path, err)
	}
}

// exeName returns the platform-specific file name for a shim or binary.
func exeName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".bat"
	}
	return name
}

// fileExists is a small assertion helper.
func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// requireContains fails the test when needle is absent from haystack, printing
// the full output to make CI failures diagnosable without a re-run.
func requireContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if !strings.Contains(haystack, needle) {
		t.Fatalf("%s\nexpected output to contain %q\n--- actual output ---\n%s", context, needle, haystack)
	}
}

// requireNotContains fails the test when needle is present in haystack.
func requireNotContains(t *testing.T, haystack, needle, context string) {
	t.Helper()
	if strings.Contains(haystack, needle) {
		t.Fatalf("%s\nexpected output NOT to contain %q\n--- actual output ---\n%s", context, needle, haystack)
	}
}

// skipWithoutNetwork skips tests that download a Go toolchain unless explicitly
// enabled, keeping the default suite fast and offline.
func skipWithoutNetwork(t *testing.T) {
	t.Helper()
	if os.Getenv("GOENV_E2E_NETWORK") != "1" {
		t.Skip("set GOENV_E2E_NETWORK=1 to run tests that download a Go toolchain")
	}
}
