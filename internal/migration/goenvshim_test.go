package migration

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

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
# goenv shim for goenv
set -e
program="${0##*/}"
exec goenv exec "$program" "$@"
`
)

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
