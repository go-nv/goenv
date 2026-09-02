package legacy

import (
	"strings"
	"testing"
)

// 'goenv local --vscode-env-vars' forwards to 'goenv vscode init --env-vars',
// which selects an inert configuration: the Go extension documents go.goroot as
// "the GOROOT to use when no environment variable is set", so "${env:GOROOT}"
// is ignored whenever GOROOT is exported, and has nothing to expand to when it
// is not. The working alternative is go.alternateTools -> "goenv exec go", via
// 'goenv vscode fix-extension'. See issue #367.
//
// This pins the deprecation so the flag cannot quietly become an ordinary one
// again. It asserts on the deprecation mechanism, not on the exact prose.
func TestVSCodeEnvVarsFlagIsDeprecated(t *testing.T) {
	flag := localCmd.Flags().Lookup("vscode-env-vars")
	if flag == nil {
		t.Fatal("--vscode-env-vars flag is missing; it must remain registered so existing scripts keep working")
	}

	if flag.Deprecated == "" {
		t.Error("--vscode-env-vars must be marked deprecated: it forwards to the inert --env-vars mode")
	}

	if !strings.Contains(flag.Deprecated, "fix-extension") {
		t.Errorf("deprecation message must point at 'goenv vscode fix-extension', got: %q", flag.Deprecated)
	}
}
