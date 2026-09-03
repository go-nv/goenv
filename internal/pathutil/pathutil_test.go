package pathutil
package pathutil

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExpandPath(t *testing.T) {
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	t.Run("empty stays empty", func(t *testing.T) {
		assert.Equal(t, "", ExpandPath(""))
	})

	t.Run("leading ~/ expands to home", func(t *testing.T) {
		assert.Equal(t, filepath.Join(home, "go"), ExpandPath("~/go"))
	})

	t.Run("bare ~ expands to home", func(t *testing.T) {
		assert.Equal(t, home, ExpandPath("~"))
	})

	t.Run("$VAR is expanded", func(t *testing.T) {
		t.Setenv("GOENV_TEST_EXP", "/some/dir")
		assert.Equal(t, "/some/dir/x", ExpandPath("$GOENV_TEST_EXP/x"))
	})

	t.Run("absolute path is unchanged", func(t *testing.T) {
		abs := filepath.Join(home, "already", "absolute")
		assert.Equal(t, abs, ExpandPath(abs))
	})

	t.Run("a non-leading ~ is left alone", func(t *testing.T) {
		// A mid-path tilde is a literal filename character, not shell sugar.
		assert.Equal(t, "/a/~b", ExpandPath("/a/~b"))
	})
}

// TestExpandPath_Windows covers the native Windows forms (~\ and %VAR%), which
// only expand on Windows. Gated so the assertions run on the Windows CI runner.
func TestExpandPath_Windows(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows-specific path forms")
	}
	home, err := os.UserHomeDir()
	require.NoError(t, err)

	assert.Equal(t, filepath.Join(home, "go"), ExpandPath(`~\go`), "~\\ should expand to home")

	t.Setenv("USERPROFILE", home)
	assert.Equal(t, filepath.Join(home, "go"), ExpandPath(`%USERPROFILE%\go`), "%USERPROFILE% should expand")

	// An unknown %VAR% is left untouched rather than blanked out.
	assert.Equal(t, `%NO_SUCH_VAR%\x`, ExpandPath(`%NO_SUCH_VAR%\x`))
}
