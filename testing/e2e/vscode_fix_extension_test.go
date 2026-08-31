//go:build e2e

package e2e

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// 'goenv vscode fix-extension' is the remedy goenv points at from 'goenv
// doctor', from 'goenv vscode init', from the --env-vars deprecation notice and
// from docs/user-guide/VSCODE_INTEGRATION.md. It is the answer to issue #367,
// where an exported GOROOT makes the Go extension ignore 'goenv local'.
//
// It was also, for a while, entirely unreachable: fixVSCodeExtensionInteractive
// was written in full but never registered as a cobra command, so every one of
// those recommendations named a command that did not exist. Nothing detected
// that — Go does not warn about unused functions, and unit tests exercising the
// internals still passed because the internals were fine. Only running the real
// binary shows whether a command can actually be invoked.
//
// That is why these live in the e2e suite rather than as unit tests.

// vscodeUserSettingsPath mirrors internal/vscode.GetUserSettingsPath for the
// sandboxed HOME the harness sets up. Kept deliberately independent of the
// production helper: if that helper changed to write somewhere outside the
// sandbox, a test sharing its logic would follow it silently, while this one
// fails and says so.
func vscodeUserSettingsPath(t *testing.T, home string) string {
	t.Helper()
	switch runtime.GOOS {
	case "windows":
		return filepath.Join(home, "AppData", "Roaming", "Code", "User", "settings.json")
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Code", "User", "settings.json")
	default:
		return filepath.Join(home, ".config", "Code", "User", "settings.json")
	}
}

func readJSONMap(t *testing.T, path string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read %s: %v", path, err)
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatalf("settings at %s are not valid JSON: %v\n%s", path, err, data)
	}
	return out
}

// TestVSCodeFixExtension_IsReachable is the regression test for the missing
// registration. A non-zero exit, or cobra's "unknown command" help dump, means
// the command is not wired up.
func TestVSCodeFixExtension_IsReachable(t *testing.T) {
	e := newEnv(t)

	res := e.run("vscode", "fix-extension")

	if !res.Succeeded() {
		t.Fatalf("'goenv vscode fix-extension' failed (exit %d).\n"+
			"If this reports an unknown command, the subcommand has been unregistered again "+
			"while goenv, doctor and the docs still recommend it.\n--- output ---\n%s",
			res.ExitCode, truncate(res.Output(), 2000))
	}

	// Cobra prints the parent's help for an unknown subcommand and can still
	// exit 0, so confirm the command actually did its work.
	if !strings.Contains(res.Output(), "alternateTools") {
		t.Errorf("output does not mention the setting it is supposed to write; "+
			"the invocation may have fallen through to parent help:\n%s",
			truncate(res.Output(), 2000))
	}
}

// TestVSCodeFixExtension_WritesGoenvDelegation checks the settings it produces
// are the ones that actually fix #367: go.goroot and go.gopath cleared so the
// extension stops injecting fixed paths, and go.alternateTools.go delegating to
// goenv so the version is resolved per invocation.
func TestVSCodeFixExtension_WritesGoenvDelegation(t *testing.T) {
	e := newEnv(t)

	res := e.run("vscode", "fix-extension")
	if !res.Succeeded() {
		t.Fatalf("command failed (exit %d)\n%s", res.ExitCode, truncate(res.Output(), 2000))
	}

	settingsPath := vscodeUserSettingsPath(t, e.Home)
	if _, err := os.Stat(settingsPath); err != nil {
		t.Fatalf("expected user settings at %s, but it was not created: %v\n"+
			"(if this path is wrong the command may have written outside the sandbox)",
			settingsPath, err)
	}

	settings := readJSONMap(t, settingsPath)

	// go.goroot / go.gopath must be cleared: a non-empty value here is what
	// leaks a fixed toolchain into the terminal PATH.
	for _, key := range []string{"go.goroot", "go.gopath"} {
		value, ok := settings[key]
		if !ok {
			t.Errorf("%s is absent; it should be present and empty so it overrides any earlier value", key)
			continue
		}
		if str, isString := value.(string); !isString || str != "" {
			t.Errorf("%s = %v, want an empty string", key, value)
		}
	}

	// The delegation itself.
	alternate, ok := settings["go.alternateTools"].(map[string]any)
	if !ok {
		t.Fatalf("go.alternateTools missing or not an object: %v", settings["go.alternateTools"])
	}
	goTool, _ := alternate["go"].(string)
	if !strings.Contains(goTool, "goenv") {
		t.Errorf("go.alternateTools.go = %q, want it to route through goenv", goTool)
	}
}

// TestVSCodeFixExtension_IsIdempotent guards against the command mangling
// settings when run twice — plausible for anyone who runs it, reloads, and runs
// it again after not seeing an immediate change.
func TestVSCodeFixExtension_IsIdempotent(t *testing.T) {
	e := newEnv(t)

	if res := e.run("vscode", "fix-extension"); !res.Succeeded() {
		t.Fatalf("first run failed (exit %d)\n%s", res.ExitCode, truncate(res.Output(), 2000))
	}
	settingsPath := vscodeUserSettingsPath(t, e.Home)
	first, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings after first run: %v", err)
	}

	res := e.run("vscode", "fix-extension")
	if !res.Succeeded() {
		t.Fatalf("second run failed (exit %d)\n%s", res.ExitCode, truncate(res.Output(), 2000))
	}

	second, err := os.ReadFile(settingsPath)
	if err != nil {
		t.Fatalf("failed to read settings after second run: %v", err)
	}

	if string(first) != string(second) {
		t.Errorf("running fix-extension twice changed the settings.\n--- after first ---\n%s\n--- after second ---\n%s",
			first, second)
	}
}

// TestVSCodeFixExtension_PreservesUnrelatedSettings makes sure the command
// confines itself to Go keys. It edits the user's real VS Code settings, so
// clobbering an unrelated preference would be a serious misstep.
func TestVSCodeFixExtension_PreservesUnrelatedSettings(t *testing.T) {
	e := newEnv(t)

	settingsPath := vscodeUserSettingsPath(t, e.Home)
	if err := os.MkdirAll(filepath.Dir(settingsPath), 0o755); err != nil {
		t.Fatalf("failed to create settings dir: %v", err)
	}
	existing := `{
  "editor.fontSize": 17,
  "workbench.colorTheme": "Solarized Dark",
  "go.goroot": "/somewhere/stale"
}`
	if err := os.WriteFile(settingsPath, []byte(existing), 0o644); err != nil {
		t.Fatalf("failed to seed settings: %v", err)
	}

	res := e.run("vscode", "fix-extension")
	if !res.Succeeded() {
		t.Fatalf("command failed (exit %d)\n%s", res.ExitCode, truncate(res.Output(), 2000))
	}

	settings := readJSONMap(t, settingsPath)

	if got := settings["editor.fontSize"]; got != float64(17) {
		t.Errorf("editor.fontSize = %v, want 17 preserved", got)
	}
	if got := settings["workbench.colorTheme"]; got != "Solarized Dark" {
		t.Errorf("workbench.colorTheme = %v, want it preserved", got)
	}
	if got, _ := settings["go.goroot"].(string); got != "" {
		t.Errorf("go.goroot = %q, want the stale value cleared", got)
	}
}
