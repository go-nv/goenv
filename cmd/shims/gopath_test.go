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

	t.Run("expands ~, drops non-absolute and managed, keeps custom", func(t *testing.T) {
		raw := strings.Join([]string{"~/go", "relative/path", managedVersion, customAbs}, sep)
		got := sanitizeInheritedGopath(raw, prefix)
		// ~/go expands to the prefix (kept); relative dropped; managed version
		// dropped; custom absolute kept; order preserved.
		assert.Equal(t, []string{prefix, customAbs}, got)
	})

	t.Run("keeps the GOPATH prefix itself", func(t *testing.T) {
		assert.Equal(t, []string{prefix}, sanitizeInheritedGopath(prefix, prefix))
	})

	t.Run("no literal tilde ever survives", func(t *testing.T) {
		for _, p := range sanitizeInheritedGopath("~/go"+sep+"~", prefix) {
			assert.NotContains(t, p, "~", "expanded entry must not contain a tilde")
			assert.True(t, filepath.IsAbs(p), "entry must be absolute")
		}
	})
}
