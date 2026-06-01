package resolver

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeExecutable creates a file at path and (on Unix) marks it executable.
// On Windows, it appends ".exe" to satisfy the executable-extension check.
func writeExecutable(t *testing.T, dir, name string) string {
	t.Helper()
	require.NoError(t, os.MkdirAll(dir, 0o755))

	fileName := name
	if runtime.GOOS == "windows" {
		fileName = name + ".exe"
	}
	path := filepath.Join(dir, fileName)
	require.NoError(t, os.WriteFile(path, []byte("#!/bin/sh\necho ok\n"), 0o755))
	return path
}

// newTestResolver builds a Resolver pointed at the given root, with GOPATH
// enabled.
func newTestResolver(root string) *Resolver {
	cfg := &config.Config{Root: root}
	env := &utils.GoenvEnvironment{}
	return New(cfg, env)
}

// TestResolveBinary_FindsLegacyHomeGopathBin verifies that ResolveBinary picks
// up binaries installed by the shims into "$HOME/go/{version}/bin", which is
// the path the v3 shims actually write to today.
func TestResolveBinary_FindsLegacyHomeGopathBin(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	root := filepath.Join(tmp, "goenv")
	require.NoError(t, os.MkdirAll(home, 0o755))
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	const version = "1.21.0"
	const tool = "sometool"

	legacyBinDir := filepath.Join(home, "go", version, "bin")
	writeExecutable(t, legacyBinDir, tool)

	r := newTestResolver(root)

	got, err := r.ResolveBinary(tool, version, "")
	require.NoError(t, err)

	expected := filepath.Join(legacyBinDir, tool)
	if runtime.GOOS == "windows" {
		expected += ".exe"
	}
	assert.Equal(t, expected, got)
}

// TestGetBinaryDirectories_IncludesLegacyHomeGopathBin verifies that rehash
// (via GetBinaryDirectories) scans the legacy "$HOME/go/{version}/bin" path.
func TestGetBinaryDirectories_IncludesLegacyHomeGopathBin(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	root := filepath.Join(tmp, "goenv")
	require.NoError(t, os.MkdirAll(home, 0o755))
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	const version = "1.21.0"

	r := newTestResolver(root)
	dirs := r.GetBinaryDirectories(version)

	legacyBinDir := filepath.Join(home, "go", version, "bin")
	assert.Contains(t, dirs, legacyBinDir,
		"GetBinaryDirectories should include legacy $HOME/go/{version}/bin so rehash sees shim-installed tools")

	// Sanity: the v3 path should still be there too (this is the additive fix).
	cfg := &config.Config{Root: root}
	assert.Contains(t, dirs, cfg.VersionGopathBin(version),
		"GetBinaryDirectories should still include the v3 VersionGopathBin")
	assert.Contains(t, dirs, cfg.VersionBinDir(version),
		"GetBinaryDirectories should still include VersionBinDir")
}

// TestGetBinaryDirectories_OmitsLegacyWhenGopathDisabled verifies that when
// GOENV_DISABLE_GOPATH=true, the legacy path is NOT scanned (matching the
// existing behaviour for VersionGopathBin).
func TestGetBinaryDirectories_OmitsLegacyWhenGopathDisabled(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	root := filepath.Join(tmp, "goenv")
	require.NoError(t, os.MkdirAll(home, 0o755))
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	const version = "1.21.0"

	cfg := &config.Config{Root: root}
	env := &utils.GoenvEnvironment{DisableGopath: true}
	r := New(cfg, env)

	dirs := r.GetBinaryDirectories(version)

	legacyBinDir := filepath.Join(home, "go", version, "bin")
	assert.NotContains(t, dirs, legacyBinDir,
		"legacy bin should not be scanned when GOPATH is disabled")
	assert.NotContains(t, dirs, cfg.VersionGopathBin(version),
		"version gopath bin should not be scanned when GOPATH is disabled")
}

// TestFindVersionsWithBinary_FindsLegacyHomeGopathBin verifies that
// FindVersionsWithBinary considers the legacy "$HOME/go/{version}/bin" path.
func TestFindVersionsWithBinary_FindsLegacyHomeGopathBin(t *testing.T) {
	tmp := t.TempDir()
	home := filepath.Join(tmp, "home")
	root := filepath.Join(tmp, "goenv")
	require.NoError(t, os.MkdirAll(home, 0o755))
	t.Setenv("HOME", home)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", home)
	}

	const version = "1.21.0"
	const tool = "sometool"

	writeExecutable(t, filepath.Join(home, "go", version, "bin"), tool)

	r := newTestResolver(root)
	got, err := r.FindVersionsWithBinary(tool, []string{version})
	require.NoError(t, err)
	assert.Equal(t, []string{version}, got)
}

// TestConfig_LegacyHomeGopathBin spot-checks the new config helper.
func TestConfig_LegacyHomeGopathBin(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("HOME", tmp)
	if runtime.GOOS == "windows" {
		t.Setenv("USERPROFILE", tmp)
	}

	cfg := &config.Config{Root: filepath.Join(tmp, "goenv")}
	got := cfg.LegacyHomeGopathBin("1.21.0")
	assert.Equal(t, filepath.Join(tmp, "go", "1.21.0", "bin"), got)
}
