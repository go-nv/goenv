//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

// TestExecNormalizesInheritedGopath verifies end to end that `goenv exec` never
// hands the target process a GOPATH that `go` would reject. goenv prepends a
// per-version path and re-exports GOPATH on every shimmed call, so a profile
// that left a literal '~' (from a quoted GOPATH="~/go") or a non-absolute entry
// would otherwise break every build with:
//
//	go: GOPATH entry cannot start with shell metacharacter '~'
//
// The shared normalizer is unit-tested cross-platform; this proves the real
// binary actually applies it through the exec path.
func TestExecNormalizesInheritedGopath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("uses a bash stub to echo the GOPATH the exec'd process received")
	}

	e := newEnv(t)
	const version = "1.23.0"
	e.fakeVersion(version) // valid install (go + gofmt stubs)

	// Replace the `go` stub with one that prints the Go path env vars it was
	// invoked with.
	goStub := filepath.Join(e.path("versions", version, "bin"), "go")
	stub := "#!/usr/bin/env bash\nprintf 'GOPATH=[%s] GOBIN=[%s]' \"$GOPATH\" \"$GOBIN\"\n"
	if err := os.WriteFile(goStub, []byte(stub), 0o755); err != nil {
		t.Fatalf("write go stub: %v", err)
	}
	e.writeFile(".go-version", version+"\n")

	// Profile-style Go path env vars with a literal '~' that goenv must expand
	// before handing them to `go`.
	e.Set("GOPATH", "~/go")
	e.Set("GOBIN", "~/mybin")

	res := e.run("exec", "go")
	if !res.Succeeded() {
		t.Fatalf("goenv exec go failed (exit %d):\n%s", res.ExitCode, res.Output())
	}
	got := res.Output()

	requireNotContains(t, got, "~", "goenv exec must expand '~' in every Go path env var")
	// The literal ~/go and ~/mybin must have been expanded to $HOME/... (HOME is
	// the sandbox home).
	requireContains(t, got, filepath.Join(e.Home, "go"), "expanded GOPATH should include $HOME/go")
	requireContains(t, got, filepath.Join(e.Home, "mybin"), "GOBIN '~' should expand to $HOME/mybin")
}
