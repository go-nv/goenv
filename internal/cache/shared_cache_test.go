package cache

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeSizedFile creates a file of a known size so cache sizing assertions are
// meaningful without needing a real module cache.
func writeSizedFile(t *testing.T, path string, size int) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, make([]byte, size), 0o644))
}

// TestGetCacheStatus_FindsSharedCacheWithoutVersionsDir is a regression test
// for issue #578.
//
// GetCacheStatus used to return early when versions/ was absent, before it ever
// looked at the shared module cache. Because the shared cache lives outside
// versions/ and outlives every installed version, goenv reported zero bytes
// while gigabytes sat on disk — which is why the reporter had to find it by
// hand with docker history.
func TestGetCacheStatus_FindsSharedCacheWithoutVersionsDir(t *testing.T) {
	root := t.TempDir()

	// Deliberately no versions/ directory.
	const size = 4096
	writeSizedFile(t, filepath.Join(root, "shared", "go-mod", "cache", "blob.bin"), size)

	status, err := GetCacheStatus(root, false)
	require.NoError(t, err)

	require.Len(t, status.ModCaches, 1,
		"the shared module cache must be reported even with no installed versions")
	assert.Equal(t, SharedCacheLabel, status.ModCaches[0].GoVersion)
	assert.GreaterOrEqual(t, status.TotalSize, int64(size),
		"shared cache bytes must be counted in the total")
}

// TestGetCacheStatus_SharedCacheIsNotAttributedToAVersion documents why the
// display layer has to handle the shared cache separately: it is intentionally
// absent from ByVersion, so anything rendering only ByVersion will omit it.
func TestGetCacheStatus_SharedCacheIsNotAttributedToAVersion(t *testing.T) {
	root := t.TempDir()

	writeSizedFile(t, filepath.Join(root, "shared", "go-mod", "cache", "blob.bin"), 2048)
	// A real version with its own module cache, so both kinds are present.
	writeSizedFile(t, filepath.Join(root, "versions", "1.23.6", "pkg", "mod", "v.bin"), 1024)

	status, err := GetCacheStatus(root, false)
	require.NoError(t, err)

	assert.NotContains(t, status.ByVersion, SharedCacheLabel,
		"the shared cache must not masquerade as an installed version")

	var sawShared bool
	for _, c := range status.ModCaches {
		if c.GoVersion == SharedCacheLabel {
			sawShared = true
		}
	}
	assert.True(t, sawShared, "shared cache should still be present in ModCaches")

	// Totals must account for both, otherwise the summary silently under-reports.
	assert.GreaterOrEqual(t, status.TotalSize, int64(2048+1024))
}
