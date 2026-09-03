package sbom

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// requireGo skips the test when no Go toolchain is available on PATH.
func requireGo(t *testing.T) string {
	t.Helper()
	goBin, err := exec.LookPath("go")
	if err != nil {
		t.Skip("go toolchain not available on PATH")
	}
	return goBin
}

// writeModule writes a minimal, buildable Go module into dir. The provided
// goMod and mainGo contents let individual tests exercise replace/retract
// directives and specific imports. All modules are stdlib-only so the toolchain
// commands run fully offline (no proxy access).
func writeModule(t *testing.T, dir, goMod, mainGo string) {
	t.Helper()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "go.mod"), []byte(goMod), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "main.go"), []byte(mainGo), 0o644))
}

const stdlibMain = `package main

import (
	"crypto/tls"
	"fmt"
	"net/http"
)

func main() {
	_ = tls.VersionTLS13
	_ = http.MethodGet
	fmt.Println("ok")
}
`

// newToolchainAt returns a Toolchain bound to dir using the go binary on PATH.
func newToolchainAt(dir string, offline bool) *Toolchain {
	return NewToolchain(nil, nil, dir, offline)
}

func TestE2E_Toolchain_Available(t *testing.T) {
	requireGo(t)
	tc := newToolchainAt(t.TempDir(), false)
	require.True(t, tc.Available(), "toolchain should resolve go on PATH")
	assert.NotEmpty(t, tc.GoBinary())
}

func TestE2E_Toolchain_NilSafety(t *testing.T) {
	var tc *Toolchain
	assert.False(t, tc.Available(), "nil toolchain must report unavailable, not panic")
	assert.Empty(t, tc.GoBinary())
}

func TestE2E_Toolchain_GoEnv(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/env\n\ngo 1.22\n", stdlibMain)

	tc := newToolchainAt(dir, false)
	env, err := tc.GoEnv("GOOS", "GOARCH", "CGO_ENABLED")
	require.NoError(t, err)
	assert.NotEmpty(t, env["GOOS"])
	assert.NotEmpty(t, env["GOARCH"])
	// CGO_ENABLED is 0 or 1 depending on the host; just assert it's present.
	_, ok := env["CGO_ENABLED"]
	assert.True(t, ok)
}

func TestE2E_Toolchain_StdlibPackages(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/std\n\ngo 1.22\n", stdlibMain)

	tc := newToolchainAt(dir, false)
	pkgs, err := tc.StdlibPackages()
	require.NoError(t, err)
	require.NotEmpty(t, pkgs, "expected stdlib packages from go list -deps")

	set := make(map[string]bool)
	for _, p := range pkgs {
		set[p] = true
	}
	for _, want := range []string{"net/http", "crypto/tls", "fmt"} {
		assert.True(t, set[want], "expected stdlib package %q in %v", want, pkgs)
	}
	// Pseudo-packages must be filtered out.
	assert.False(t, set["unsafe"], "unsafe should be filtered")
	assert.False(t, set["C"], "C should be filtered")
}

func TestE2E_Toolchain_ModEdit_Replaces(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	goMod := `module example.com/replaces

go 1.22

require (
	github.com/example/lib v1.2.3
	github.com/other/pkg v1.0.0
)

replace github.com/example/lib => ../local-fork

replace github.com/other/pkg v1.0.0 => github.com/fork/pkg v1.1.0
`
	writeModule(t, dir, goMod, "package main\n\nfunc main() {}\n")

	tc := newToolchainAt(dir, false)
	mod, err := tc.ModEdit()
	require.NoError(t, err)
	assert.Equal(t, "example.com/replaces", mod.Module.Path)
	require.Len(t, mod.Replace, 2)

	directives := classifyReplaces(mod.Replace)
	require.Len(t, directives, 2)

	// First: local path replacement -> high risk.
	assert.Equal(t, "local-path", directives[0].Type)
	assert.Equal(t, "high", directives[0].RiskLevel)

	// Second: fork replacement -> medium risk.
	assert.Equal(t, "fork", directives[1].Type)
	assert.Equal(t, "medium", directives[1].RiskLevel)
}

func TestE2E_Toolchain_ModEdit_Retract(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	goMod := `module example.com/retract

go 1.22

retract (
	v1.0.0 // bug
	v1.1.0 // security
)
`
	writeModule(t, dir, goMod, "package main\n\nfunc main() {}\n")

	tc := newToolchainAt(dir, false)
	mod, err := tc.ModEdit()
	require.NoError(t, err)
	require.Len(t, mod.Retract, 2)
	assert.Equal(t, "v1.0.0", mod.Retract[0].Low)
	assert.Equal(t, "v1.1.0", mod.Retract[1].Low)
}

func TestE2E_Toolchain_Modules_StdlibOnly(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/mods\n\ngo 1.22\n", stdlibMain)

	tc := newToolchainAt(dir, false)
	mods, err := tc.Modules()
	require.NoError(t, err)
	require.NotEmpty(t, mods)
	// The main module must be present and marked Main.
	var foundMain bool
	for _, m := range mods {
		if m.Path == "example.com/mods" {
			foundMain = m.Main
		}
	}
	assert.True(t, foundMain, "main module should be reported by go list -m")
}

func TestE2E_Toolchain_OfflineRetractionsError(t *testing.T) {
	requireGo(t)
	dir := t.TempDir()
	writeModule(t, dir, "module example.com/off\n\ngo 1.22\n", stdlibMain)

	tc := newToolchainAt(dir, true) // offline
	_, err := tc.Retractions()
	assert.Error(t, err, "offline retraction lookup must return an error, not silently succeed")
}

func TestE2E_ReadBinaryProvenance_WithVCS(t *testing.T) {
	goBin := requireGo(t)
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}

	dir := t.TempDir()
	writeModule(t, dir, "module example.com/vcsprov\n\ngo 1.22\n", "package main\n\nfunc main() {}\n")

	// Initialize a git repo so the build embeds VCS provenance.
	runGit := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=t", "GIT_AUTHOR_EMAIL=t@t",
			"GIT_COMMITTER_NAME=t", "GIT_COMMITTER_EMAIL=t@t",
		)
		require.NoError(t, cmd.Run(), "git %v", args)
	}
	runGit("init")
	runGit("add", "-A")
	runGit("commit", "-m", "init")

	bin := filepath.Join(dir, "app")
	build := exec.Command(goBin, "build", "-o", bin, ".")
	build.Dir = dir
	build.Env = append(os.Environ(), "CGO_ENABLED=0")
	out, err := build.CombinedOutput()
	require.NoError(t, err, "go build failed: %s", out)

	prov, err := ReadBinaryProvenance(bin)
	require.NoError(t, err)
	assert.NotEmpty(t, prov.GoVersion)
	assert.Len(t, prov.VCSRevision, 40, "expected a full git SHA in vcs.revision")
	assert.False(t, prov.VCSModified, "clean tree should report vcs_modified=false")
	assert.Equal(t, "0", prov.Settings["CGO_ENABLED"])
}
