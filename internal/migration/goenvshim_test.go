package migration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/go-nv/goenv/internal/shims"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeShim(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	require.NoError(t, os.WriteFile(path, []byte(content), 0o755))
	return path
}

const (
	v2GoenvShim = `#!/usr/bin/env bash
set -e
exec "/opt/homebrew/Cellar/goenv/2.2.38_1/libexec/goenv" "$@"
`
	v3GoenvShim = `#!/usr/bin/env bash
# goenv-shim v1
# goenv shim for goenv
set -e
program="${0##*/}"
exec goenv exec "$program" "$@"
`
)

// TestFindGoenvShims_IgnoresTheGoenvBinary is a regression test for a
// self-deletion bug introduced while fixing issue #542.
//
// Detection originally matched the substring "goenv exec" against a candidate
// file's raw bytes. The compiled goenv binary embeds the shim templates, so it
// contains that sequence too — meaning a goenv binary placed in the shims
// directory (which packagers and users do) was classified as a stale shim and
// deleted on the next invocation.
//
// Detection gates a destructive operation, so it must positively identify a
// shim rather than merely fail to rule one out.
func TestFindGoenvShims_IgnoresTheGoenvBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim file naming differs on Windows; covered by the name-list test")
	}

	dir := t.TempDir()

	// Stand in for the real binary: ELF-ish header, NUL bytes, and the shim
	// template text that the real binary genuinely embeds.
	fakeBinary := append([]byte{0x7f, 'E', 'L', 'F', 0x00, 0x00, 0x00},
		[]byte("exec goenv exec \"$program\" \"$@\"\n# goenv shim for %s\x00")...)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "goenv"), fakeBinary, 0o755))

	assert.Empty(t, FindGoenvShims(dir),
		"a compiled goenv binary must never be classified as a stale shim")
}

// TestFindGoenvShims_IgnoresUnrelatedScripts ensures a user's own script in the
// shims directory is left alone.
func TestFindGoenvShims_IgnoresUnrelatedScripts(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim file naming differs on Windows; covered by the name-list test")
	}

	dir := t.TempDir()
	writeShim(t, dir, "goenv", "#!/usr/bin/env bash\n# my own wrapper\nexec goenv \"$@\"\n")

	assert.Empty(t, FindGoenvShims(dir),
		"a hand-written wrapper without a goenv shim marker must not be deleted")
}

// TestGeneratedShimsCarryTheMarker pins the contract between the shim generator
// and this package: if internal/shims stops emitting the marker, stale-shim
// detection silently stops working. The constants are asserted here so the two
// packages cannot drift apart unnoticed.
func TestGeneratedShimsCarryTheMarker(t *testing.T) {
	assert.True(t, v3ShimPattern.MatchString("#!/usr/bin/env bash\n"+shims.UnixShimMarker+"\n"),
		"the Unix shim marker emitted by internal/shims must be recognised")
	assert.True(t, v3ShimPattern.MatchString("@echo off\r\n"+shims.WindowsShimMarker+"\r\n"),
		"the Windows shim marker emitted by internal/shims must be recognised")
}

// TestGoenvShimNamesFor_Windows exercises the Windows naming rules from any
// runner. The generated shim is ".bat"; the rest cover leftovers.
func TestGoenvShimNamesFor_Windows(t *testing.T) {
	names := goenvShimNamesFor("windows")

	assert.Contains(t, names, "goenv.bat", "rehash generates .bat shims on Windows")
	assert.Contains(t, names, "goenv", "a extension-less leftover must still be considered")

	// The generated-shim name must match what the shim generator actually
	// writes, otherwise a recursive shim is created under one name and looked
	// for under another.
	assert.Equal(t, "goenv"+shims.WindowsShimExtension, "goenv.bat",
		"shim extension must match internal/shims.createWindowsShim")
}

// TestGoenvShimNamesFor_Unix keeps the Unix list minimal: anything broader
// risks deleting a user's own file.
func TestGoenvShimNamesFor_Unix(t *testing.T) {
	for _, goos := range []string{"linux", "darwin", "freebsd"} {
		assert.Equal(t, []string{"goenv"}, goenvShimNamesFor(goos),
			"%s should only consider an extension-less goenv shim", goos)
	}
}

// TestFindGoenvShims_IgnoresBinaryNamedGoenvExe covers the Windows case of the
// self-deletion bug: on Windows "goenv.exe" is a candidate name, and it is
// exactly what the real binary is called.
func TestFindGoenvShims_IgnoresBinaryNamedGoenvExe(t *testing.T) {
	dir := t.TempDir()

	fakeBinary := append([]byte{'M', 'Z', 0x90, 0x00, 0x00},
		[]byte("exec goenv exec \"$program\" \"$@\"\x00")...)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "goenv.exe"), fakeBinary, 0o755))

	assert.Empty(t, FindGoenvShims(dir),
		"a compiled goenv.exe must never be classified as a stale shim")
}

// TestFindGoenvShims_DetectsV3FormatShim covers the gap that left issue #542
// reproducible: RemoveStaleV2Shim only matched the v2 "libexec/goenv"
// fingerprint, so a v3-format goenv shim kept recursing forever.
func TestFindGoenvShims_DetectsV3FormatShim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim file naming differs on Windows; covered by the name-list test")
	}

	shimsDir := t.TempDir()
	want := writeShim(t, shimsDir, "goenv", v3GoenvShim)

	found := FindGoenvShims(shimsDir)

	assert.Equal(t, []string{want}, found)
	assert.Empty(t, mustRemoveErr(t, shimsDir), "removal should succeed")
	assert.NoFileExists(t, want)
}

func TestFindGoenvShims_DetectsV2FormatShim(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim file naming differs on Windows")
	}

	shimsDir := t.TempDir()
	want := writeShim(t, shimsDir, "goenv", v2GoenvShim)

	assert.Equal(t, []string{want}, FindGoenvShims(shimsDir))
}

// TestFindGoenvShims_LeavesUnrelatedFilesAlone makes sure a user's own file
// named "goenv" is not deleted just because of its name.
func TestFindGoenvShims_LeavesUnrelatedFilesAlone(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shim file naming differs on Windows")
	}

	shimsDir := t.TempDir()
	path := writeShim(t, shimsDir, "goenv", "#!/bin/sh\necho hello\n")

	assert.Empty(t, FindGoenvShims(shimsDir))

	removed, err := RemoveGoenvShims(shimsDir)
	require.NoError(t, err)
	assert.Empty(t, removed)
	assert.FileExists(t, path, "an unrelated file must not be removed")
}

func TestFindGoenvShims_EmptyDirectory(t *testing.T) {
	assert.Empty(t, FindGoenvShims(t.TempDir()))
}

func TestFindGoenvShims_MissingDirectory(t *testing.T) {
	assert.Empty(t, FindGoenvShims(filepath.Join(t.TempDir(), "does-not-exist")))
}

// TestGoenvShimNames_CoversWindowsBatchShims asserts the Windows shim
// extension is included, since rehash writes ".bat" files there.
func TestGoenvShimNames_CoversWindowsBatchShims(t *testing.T) {
	names := goenvShimNames()

	if runtime.GOOS == "windows" {
		assert.Contains(t, names, "goenv.bat")
	} else {
		assert.Equal(t, []string{"goenv"}, names)
	}
}

func mustRemoveErr(t *testing.T, shimsDir string) string {
	t.Helper()
	_, err := RemoveGoenvShims(shimsDir)
	if err != nil {
		return err.Error()
	}
	return ""
}
