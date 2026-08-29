//go:build e2e

package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
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
	return s[:max] + "\n... (truncated, " + itoa(len(s)-max) + " more bytes)"
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// mentionsRecursionRisk reports whether doctor output warns about a goenv shim
// that would cause recursion. Matching on intent rather than an exact string
// keeps the test from breaking on wording changes.
func mentionsRecursionRisk(output string) bool {
	lower := strings.ToLower(output)
	for _, phrase := range []string{"recursion", "recursive", "shadow", "infinite loop", "remove"} {
		if strings.Contains(lower, phrase) {
			return true
		}
	}
	return false
}

// gotoInsideParensPattern finds a "goto :label" that sits inside a
// parenthesized if block, which breaks the CMD parser (issue #555).
var gotoInsideParensPattern = regexp.MustCompile(`(?is)\bif\b[^\r\n]*\(\s*[^)]*goto\s*:`)

// assertNoGotoInsideParens fails when a generated Windows shim contains the
// batch construct that issue #555 was filed for.
func assertNoGotoInsideParens(t *testing.T, shim string) {
	t.Helper()
	if gotoInsideParensPattern.MatchString(shim) {
		t.Fatalf("generated Windows shim contains a goto inside a parenthesized if block, "+
			"which breaks CMD parsing (issue #555):\n%s", shim)
	}
}
