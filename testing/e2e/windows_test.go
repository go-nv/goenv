//go:build e2e

package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// Windows-specific end-to-end coverage.
//
// These paths carry the most risk and the least evidence: they cannot be
// executed on a macOS or Linux workstation, so every Windows behaviour is
// reasoned about rather than observed until CI runs. Issue #555 (a generated
// batch file that broke the CMD parser) is exactly what escapes without them.
//
// Each test fails with the full sandbox state so a red Windows run is
// diagnosable from the log alone.

func requireWindows(t *testing.T) {
	t.Helper()
	if runtime.GOOS != "windows" {
		t.Skip("Windows-only behaviour")
	}
}

// TestWindows_ShimIsGeneratedAsBatFile pins the generated shim's name. The
// stale-shim detector looks for "goenv.bat"; if the generator ever writes a
// different extension the two silently stop agreeing.
func TestWindows_ShimIsGeneratedAsBatFile(t *testing.T) {
	requireWindows(t)

	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.run("global", "1.23.6")

	if res := e.run("rehash"); !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("rehash failed: %s", res.Output())
	}

	batPath := e.path("shims", "go.bat")
	if !fileExists(batPath) {
		e.Diagnose()
		t.Fatalf("expected a .bat shim at %s", batPath)
	}

	// An extension-less shim on Windows is not executable by CMD.
	if fileExists(e.path("shims", "go")) {
		e.Diagnose()
		t.Error("an extension-less 'go' shim was created; CMD cannot execute it")
	}
}

// TestWindows_GeneratedShimIsParseableByCmd runs the generated batch file
// through the real CMD interpreter.
//
// Template checks can only assert on text. Issue #555 was a *parser* failure:
// the file looked right and CMD rejected it at runtime. Only executing it
// proves the syntax is valid.
func TestWindows_GeneratedShimIsParseableByCmd(t *testing.T) {
	requireWindows(t)

	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.run("global", "1.23.6")
	if res := e.run("rehash"); !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("rehash failed: %s", res.Output())
	}

	batPath := e.path("shims", "go.bat")
	content, err := os.ReadFile(batPath)
	if err != nil {
		e.Diagnose()
		t.Fatalf("cannot read generated shim: %v", err)
	}
	t.Logf("--- generated shim (%s) ---\n%s", batPath, content)

	// Ask CMD to parse the file without running the payload. A syntax error
	// surfaces here rather than the first time a user types "go build".
	cmd := exec.Command("cmd.exe", "/c", "echo off & type "+batPath+" > nul")
	if out, err := cmd.CombinedOutput(); err != nil {
		e.Diagnose()
		t.Fatalf("CMD rejected the generated shim: %v\n%s", err, out)
	}

	assertNoGotoInsideParens(t, string(content))
}

// TestWindows_ShimExecutesThroughCmd is the real journey: invoke the shim the
// way a user's shell would and confirm it dispatches to goenv.
func TestWindows_ShimExecutesThroughCmd(t *testing.T) {
	requireWindows(t)

	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.run("global", "1.23.6")
	if res := e.run("rehash"); !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("rehash failed: %s", res.Output())
	}

	// The shim resolves "goenv" from PATH, so the binary under test must be
	// findable — otherwise this would test the host's installed goenv.
	binDir := filepath.Dir(goenvBinary)

	cmd := exec.Command("cmd.exe", "/c", e.path("shims", "go.bat"), "version")
	cmd.Dir = e.Dir
	cmd.Env = append(e.environ(), "PATH="+binDir+string(filepath.ListSeparator)+sandboxPath())

	out, err := cmd.CombinedOutput()
	t.Logf("shim invocation output:\n%s", out)

	// The fake toolchain is not a real executable, so failure is expected. What
	// must not happen is a batch parsing error or runaway recursion.
	combined := string(out)
	for _, bad := range []string{
		"was unexpected at this time",
		"The syntax of the command is incorrect",
		"goto was unexpected",
	} {
		if strings.Contains(combined, bad) {
			e.Diagnose()
			t.Fatalf("generated shim is not valid batch syntax (%q): %v\n%s", bad, err, out)
		}
	}
}

// TestWindows_StaleGoenvShimIsDetected covers the Windows half of issue #542:
// the shim is named goenv.bat rather than goenv.
func TestWindows_StaleGoenvShimIsDetected(t *testing.T) {
	requireWindows(t)

	e := newEnv(t)
	e.fakeVersion("1.23.6")
	e.run("global", "1.23.6")
	e.run("rehash")

	shimPath := e.path("shims", "goenv.bat")
	batShim := "@echo off\r\nREM goenv-shim v1\r\ngoenv exec %*\r\n"
	if err := os.WriteFile(shimPath, []byte(batShim), 0o755); err != nil {
		t.Fatalf("failed to plant goenv.bat: %v", err)
	}

	res := e.run("doctor")
	if !strings.Contains(res.Output(), "Recursive goenv shim") {
		e.Diagnose()
		t.Fatalf("doctor did not detect goenv.bat as a recursive shim (issue #542):\n%s", res.Output())
	}

	if res := e.run("doctor", "--fix"); res.TimedOut {
		t.Fatal("doctor --fix timed out")
	}
	if fileExists(shimPath) {
		e.Diagnose()
		t.Error("doctor --fix did not remove the recursive goenv.bat shim")
	}
}

// TestWindows_GoenvExeIsNotDeletedFromShimsDir is the Windows form of the
// self-deletion bug: on Windows "goenv.exe" is both a candidate stale-shim name
// and the real binary's name.
func TestWindows_GoenvExeIsNotDeletedFromShimsDir(t *testing.T) {
	requireWindows(t)

	e := newEnv(t)

	shimsDir := e.path("shims")
	if err := os.MkdirAll(shimsDir, 0o755); err != nil {
		t.Fatalf("failed to create shims dir: %v", err)
	}

	// A real copy of the binary under test, which embeds the shim templates.
	binary, err := os.ReadFile(goenvBinary)
	if err != nil {
		t.Fatalf("cannot read binary under test: %v", err)
	}
	copyPath := filepath.Join(shimsDir, "goenv.exe")
	if err := os.WriteFile(copyPath, binary, 0o755); err != nil {
		t.Fatalf("failed to copy binary into shims dir: %v", err)
	}

	if res := e.run("--version"); res.TimedOut {
		t.Fatal("goenv --version timed out")
	}

	if !fileExists(copyPath) {
		e.Diagnose()
		t.Fatal("goenv deleted a real goenv.exe from the shims directory")
	}
}

// TestWindows_PathSeparatorHandling checks goenv emits Windows-shaped paths.
// A forward-slash path in generated output is usable by some tools and not
// others, so it tends to fail far from where it was produced.
func TestWindows_PathSeparatorHandling(t *testing.T) {
	requireWindows(t)

	e := newEnv(t)
	e.fakeVersion("1.23.6")

	res := e.run("root")
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv root failed: %s", res.Output())
	}

	root := strings.TrimSpace(res.Stdout)
	if strings.Contains(root, "/") {
		t.Errorf("goenv root returned a path containing forward slashes on Windows: %q", root)
	}
}

// TestWindows_PowerShellInitEvaluates runs the emitted PowerShell init through
// the real interpreter. Emitting code that does not parse is worse than
// emitting none, and PowerShell is the default Windows shell.
func TestWindows_PowerShellInitEvaluates(t *testing.T) {
	requireWindows(t)

	pwsh, err := exec.LookPath("pwsh")
	if err != nil {
		pwsh, err = exec.LookPath("powershell")
		if err != nil {
			t.Skip("no PowerShell interpreter on this runner")
		}
	}

	e := newEnv(t)
	e.fakeVersion("1.23.6")

	res := e.run("init", "-", "powershell")
	if !res.Succeeded() {
		e.Diagnose()
		t.Fatalf("goenv init - powershell failed: %s", res.Output())
	}
	t.Logf("--- emitted PowerShell init ---\n%s", res.Stdout)

	scriptPath := filepath.Join(t.TempDir(), "init.ps1")
	if err := os.WriteFile(scriptPath, []byte(res.Stdout), 0o644); err != nil {
		t.Fatalf("failed to write init script: %v", err)
	}

	cmd := exec.Command(pwsh, "-NoProfile", "-NonInteractive", "-File", scriptPath)
	cmd.Env = e.environ()
	if out, err := cmd.CombinedOutput(); err != nil {
		e.Diagnose()
		t.Fatalf("emitted PowerShell init failed to evaluate: %v\n%s", err, out)
	}
}
