package integrations

import (
	"strings"
	"testing"
)

// The "${env:GOROOT}" configuration mode is inert. The Go extension documents
// go.goroot as "the GOROOT to use when no environment variable is set", so:
//
//   - when GOROOT is exported, the extension reads the variable and ignores
//     go.goroot entirely;
//   - when GOROOT is not exported, "${env:GOROOT}" has nothing to expand to.
//
// Either way the setting selects nothing. What actually pins the toolchain is
// the exported GOROOT, a snapshot of the launching shell's startup directory —
// which is why 'goenv local' could not override 'goenv global' inside VS Code
// (issue #367). The working alternative is go.alternateTools -> "goenv exec go",
// applied by 'goenv vscode fix-extension'.
//
// These tests pin the deprecation so it cannot be reverted to an ordinary flag
// without failing. They assert on the deprecation *mechanism* rather than on
// prose, so rewording the message does not break them.

func TestEnvVarsFlagIsDeprecated(t *testing.T) {
	flag := vscodeInitCmd.Flags().Lookup("env-vars")
	if flag == nil {
		t.Fatal("--env-vars flag is missing; it must remain registered so existing scripts keep working")
	}

	if flag.Deprecated == "" {
		t.Error("--env-vars must be marked deprecated: the mode it selects has no effect")
	}

	if !flag.Hidden {
		t.Error("--env-vars should be hidden from help once deprecated")
	}

	// The whole point of the deprecation is to redirect users somewhere that
	// works, so the message has to name it.
	if !strings.Contains(flag.Deprecated, "fix-extension") {
		t.Errorf("deprecation message must point at 'goenv vscode fix-extension', got: %q", flag.Deprecated)
	}
}
