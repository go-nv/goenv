package shell

import (
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/completions"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestRenderCompletion_EmitsEmbeddedScriptWithoutOnDiskFiles is a regression
// test for issue #593: "goenv init -" produced no completion code at all on
// installations without an <install-root>/completions directory (Homebrew being
// the common case), so "eval \"$(goenv init -)\"" silently gave no completions.
func TestRenderCompletion_EmitsEmbeddedScriptWithoutOnDiskFiles(t *testing.T) {
	// Force findCompletionPath to find nothing, mirroring a Homebrew install.
	installRootOnce.Do(func() {})
	original := installRoot
	installRoot = ""
	t.Cleanup(func() { installRoot = original })

	tests := []struct {
		shell shellutil.ShellType
		// A fragment that only appears if real completion code was emitted.
		wantFragment string
	}{
		{shellutil.ShellTypeBash, "complete -F _goenv goenv"},
		{shellutil.ShellTypeZsh, "compctl -K _goenv_compctl goenv"},
		{shellutil.ShellTypeFish, "complete -c goenv"},
		{shellutil.ShellTypePowerShell, "Register-ArgumentCompleter"},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			got := renderCompletion(tt.shell)

			require.NotEmpty(t, got,
				"renderCompletion(%s) returned nothing; completions would be silently unavailable", tt.shell)
			assert.Contains(t, got, tt.wantFragment,
				"renderCompletion(%s) must register a completion handler", tt.shell)
		})
	}
}

// TestRenderCompletion_CmdHasNoCompletion documents that cmd.exe is
// intentionally skipped — it has no completion mechanism to hook into.
func TestRenderCompletion_CmdHasNoCompletion(t *testing.T) {
	assert.Empty(t, renderCompletion(shellutil.ShellTypeCmd))
}

// TestZshInitCompletionAvoidsCompdef guards the reason a separate zsh variant
// exists: compdef is only defined once compinit has run, and "goenv init -" is
// routinely eval'd from .zshrc before that happens. compctl is a builtin and
// always works.
func TestZshInitCompletionAvoidsCompdef(t *testing.T) {
	assert.NotContains(t, stripShellComments(completions.ZshInit), "compdef",
		"the init-time zsh completion must not depend on compdef")
	assert.Contains(t, completions.ZshInit, "compctl",
		"the init-time zsh completion should register via the compctl builtin")

	// The fpath variant still needs compdef.
	assert.Contains(t, completions.Zsh, "compdef")
}

// stripShellComments removes whole-line "#" comments so assertions apply to the
// executable part of a shell script only.
func stripShellComments(script string) string {
	var kept []string
	for _, line := range strings.Split(script, "\n") {
		if strings.HasPrefix(strings.TrimSpace(line), "#") {
			continue
		}
		kept = append(kept, line)
	}
	return strings.Join(kept, "\n")
}
