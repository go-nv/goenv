//go:build e2e

package e2e

import (
	"strings"
	"testing"
	"time"
)

// TestLocalSync_DoesNotLoop guards the same defect as TestAutoInstall_DoesNotLoop
// (issues #572, #542) on a second code path that was missed the first time.
//
// "goenv local --sync" installs the version named in .go-version when it is
// missing. It located the install command and called installCmd.Execute(), and
// cobra's Execute() always dispatches from the root command, so the whole
// "goenv local --sync" invocation re-ran, found the version still missing, and
// tried to install it again.
func TestLocalSync_DoesNotLoop(t *testing.T) {
	e := newEnv(t)

	// A version that cannot be installed makes this fail fast, so the test
	// measures re-entrancy rather than download speed.
	e.writeFile(".go-version", "0.0.1\n")

	res := e.runWithTimeout(60*time.Second, "local", "--sync")

	if res.TimedOut {
		t.Fatalf("goenv local --sync did not terminate: the install path is looping "+
			"(same root cause as issues #572, #542)\n--- output (truncated) ---\n%s",
			truncate(res.Output(), 2000))
	}

	// One invocation means at most one install attempt.
	if got := strings.Count(res.Output(), "Installing Go"); got > 1 {
		t.Fatalf("goenv local --sync attempted %d installs for a single invocation; expected at most 1\n"+
			"--- output (truncated) ---\n%s", got, truncate(res.Output(), 2000))
	}
}
