//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"testing"
	"time"
)

// writeGoenvFile writes a file directly into GOENV_ROOT (config files such as
// default-tools.yaml live there rather than in the working directory).
func (e *env) writeGoenvFile(name, content string) string {
	e.t.Helper()

	full := e.path(name)
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		e.t.Fatalf("failed to create directory for %s: %v", name, err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		e.t.Fatalf("failed to write %s: %v", name, err)
	}
	return full
}

// lookPath reports whether an executable is available on this runner.
func lookPath(name string) (string, error) {
	return exec.LookPath(name)
}

// runShell executes a shell with the supplied arguments and returns its
// combined output.
func runShell(t *testing.T, shell string, args ...string) (string, error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, shell, args...)
	// Keep the user's real profile out of the picture so failures are ours.
	cmd.Env = append(os.Environ(), "NO_COLOR=1")

	out, err := cmd.CombinedOutput()
	return string(out), err
}

// truncate shortens long command output for failure messages.
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max] + "\n... (truncated, " + strconv.Itoa(len(s)-max) + " more bytes)"
}

// gotoInsideParensPattern finds a "goto :label" that sits inside a
// parenthesized if block, which breaks the CMD parser (issue #555).
//
// The `(?s)` flag matters: the construct this exists to catch spans multiple
// lines —
//
//	if "%x%"=="y" (
//	  goto :label
//	)
//
// so the character class between the paren and the goto must be able to cross
// newlines. An earlier version used [^\r\n]*, which could only ever match a
// single-line form that issue #555 was not about — a guard that could not fire.
var gotoInsideParensPattern = regexp.MustCompile(`(?is)\bif\b[^(\r\n]*\([^)]*\bgoto\s*:`)

// assertNoGotoInsideParens fails when a generated Windows shim contains the
// batch construct that issue #555 was filed for.
func assertNoGotoInsideParens(t *testing.T, shim string) {
	t.Helper()
	if gotoInsideParensPattern.MatchString(shim) {
		t.Fatalf("generated Windows shim contains a goto inside a parenthesized if block, "+
			"which breaks CMD parsing (issue #555):\n%s", shim)
	}
}
