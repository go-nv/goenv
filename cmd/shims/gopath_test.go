package shims

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSanitizeInheritedGopath(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	prefix := filepath.Join(home, "go")              // $GOPATH_PREFIX
	managedVersion := filepath.Join(prefix, "1.2.3") // goenv-managed per-version path
	customAbs := filepath.Join(t.TempDir(), "custom")
	sep := string(os.PathListSeparator)

	t.Run("empty returns nil", func(t *testing.T) {
		assert.Nil(t, sanitizeInheritedGopath("", prefix))
	})

	t.Run("expands ~, drops only managed-version paths", func(t *testing.T) {
		raw := strings.Join([]string{"~/go", managedVersion, customAbs}, sep)
		got := sanitizeInheritedGopath(raw, prefix)
		// ~/go expands to the prefix (== $prefix, kept); the managed per-version
		// path is dropped (goenv prepends its own); custom is kept; order preserved.
		assert.Equal(t, []string{prefix, customAbs}, got)
	})

	t.Run("keeps a still-invalid entry so go can reject it loudly", func(t *testing.T) {
		// Normalization must not silently drop a bad entry and change the answer.
		raw := strings.Join([]string{"~/go", "relative/path", customAbs}, sep)
		assert.Equal(t, []string{prefix, "relative/path", customAbs}, sanitizeInheritedGopath(raw, prefix))
	})

	t.Run("keeps the GOPATH prefix itself", func(t *testing.T) {
		assert.Equal(t, []string{prefix}, sanitizeInheritedGopath(prefix, prefix))
	})

	t.Run("no literal tilde ever survives", func(t *testing.T) {
		for _, p := range sanitizeInheritedGopath("~/go"+sep+"~", prefix) {
			assert.NotContains(t, p, "~", "expanded entry must not contain a tilde")
		}
	})
}

// TestNormalizeGopathEntries guards that normalization (used for the exec
// baseline, the system version and GOENV_DISABLE_GOPATH) expands and validates
// entries but PRESERVES a user's own managed-prefix path — unlike
// sanitizeInheritedGopath, which de-duplicates it only when goenv is actually
// prepending a managed path.
func TestNormalizeGopathEntries(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skipf("no home dir: %v", err)
	}
	prefix := filepath.Join(home, "go")
	managedVersion := filepath.Join(prefix, "1.2.3")
	customAbs := filepath.Join(t.TempDir(), "custom")
	sep := string(os.PathListSeparator)

	assert.Nil(t, normalizeGopathEntries(""))

	raw := strings.Join([]string{"~/go", "relative/path", managedVersion, customAbs}, sep)
	got := normalizeGopathEntries(raw)
	// Expands ~/go -> prefix; keeps EVERYTHING else (relative + managed-version +
	// custom) so goenv never silently substitutes a value go would have rejected.
	assert.Equal(t, []string{prefix, "relative/path", managedVersion, customAbs}, got)

	// Never blank a value that fails to normalize: a relative-only GOPATH is
	// preserved (go rejects it loudly) rather than reduced to "" (which would
	// silently select the default GOPATH). Only truly empty input yields nil.
	assert.Equal(t, []string{"relative-only"}, normalizeGopathEntries("relative-only"))
	assert.Nil(t, normalizeGopathEntries(sep+sep))
}
