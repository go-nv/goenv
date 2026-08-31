//go:build e2e

package e2e

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// These tests cover the VS Code auto-detect branch of "goenv use", which had
// two defects that unit tests could not see because both depend on what stdin
// really is at runtime.
//
//  1. The prompt read stdin with fmt.Fscanln and treated an empty response as
//     "yes". On a non-terminal stdin that read returns immediately, so running
//     'goenv use' in CI silently rewrote a tracked .vscode/settings.json —
//     dirtying the working tree and breaking 'git diff --exit-code' gates.
//
//  2. The first fix for (1) announced itself whenever .vscode/settings.json
//     merely existed, so a repo whose settings contain nothing but
//     "editor.tabSize" produced a line of log noise on every CI run.
//
// The contract these tests pin is therefore narrow and behavioural:
//
//   - the file on disk is never modified without explicit consent, and
//   - goenv only speaks when the settings are genuinely stale.
//
// Note that the harness sets no CI variable, so the non-interactive path here
// is reached purely because stdin is not a terminal. That is deliberate: it
// exercises the half of the guard that CI-variable detection cannot cover
// (cron, container builds, piped scripts).

const (
	e2eVSCodeVersion      = "1.23.2" // the version 'goenv use' will select
	e2eVSCodeStaleVersion = "1.22.0" // what the settings file is pinned to
)

// writeVSCodeSettings writes .vscode/settings.json and returns its path plus
// the exact bytes written, so a test can prove the file was left untouched
// rather than merely rewritten to an equivalent value.
func writeVSCodeSettings(t *testing.T, e *env, body string) (string, []byte) {
	t.Helper()
	path := e.writeFile(filepath.Join(".vscode", "settings.json"), body)
	original, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read back settings.json: %v", err)
	}
	return path, original
}

// gorootSetting builds a settings.json pinned to a version inside the sandbox
// root. Forward slashes keep the JSON valid on Windows, and CheckSettings
// normalises separators before parsing.
func gorootSetting(e *env, version string) string {
	goroot := filepath.ToSlash(e.path("versions", version))
	return `{"go.goroot": "` + goroot + `"}`
}

func assertUnchanged(t *testing.T, path string, original []byte, res result) {
	t.Helper()
	current, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read settings.json: %v", err)
	}
	if string(current) != string(original) {
		t.Errorf("settings.json was modified without consent.\n--- before ---\n%s\n--- after ---\n%s\n--- command output ---\n%s",
			original, current, truncate(res.Output(), 2000))
	}
}

// TestUse_NonInteractive_LeavesStaleVSCodeSettingsAlone is the regression test
// for defect (1): stale settings must survive, and goenv must say why.
func TestUse_NonInteractive_LeavesStaleVSCodeSettingsAlone(t *testing.T) {
	e := newEnv(t)
	e.fakeVersion(e2eVSCodeVersion)
	path, original := writeVSCodeSettings(t, e, gorootSetting(e, e2eVSCodeStaleVersion))

	res := e.run("use", e2eVSCodeVersion)

	if !res.Succeeded() {
		t.Fatalf("goenv use failed (exit %d); the VS Code branch must not affect the exit code\n%s",
			res.ExitCode, truncate(res.Output(), 2000))
	}
	assertUnchanged(t, path, original, res)

	// Stale settings are worth exactly one actionable mention.
	if !strings.Contains(res.Output(), "vscode sync") {
		t.Errorf("expected stale settings to be reported with a remedy ('goenv vscode sync'), got:\n%s",
			truncate(res.Output(), 2000))
	}
}

// TestUse_CI_LeavesStaleVSCodeSettingsAlone covers the same contract with an
// explicit CI variable, so the guard holds even where stdin happens to be a
// terminal (some runners allocate one).
func TestUse_CI_LeavesStaleVSCodeSettingsAlone(t *testing.T) {
	e := newEnv(t)
	e.Set("CI", "true")
	e.fakeVersion(e2eVSCodeVersion)
	path, original := writeVSCodeSettings(t, e, gorootSetting(e, e2eVSCodeStaleVersion))

	res := e.run("use", e2eVSCodeVersion)

	if !res.Succeeded() {
		t.Fatalf("goenv use failed (exit %d)\n%s", res.ExitCode, truncate(res.Output(), 2000))
	}
	assertUnchanged(t, path, original, res)
}

// TestUse_NonInteractive_QuietWhenNothingToDo is the regression test for
// defect (2). A settings file with no Go configuration, or one already on the
// right version, must produce no VS Code output at all — otherwise every CI run
// in such a repo gains a spurious line.
func TestUse_NonInteractive_QuietWhenNothingToDo(t *testing.T) {
	tests := []struct {
		name string
		body func(e *env) string
	}{
		{
			name: "settings.json with no Go configuration",
			body: func(*env) string { return `{"editor.tabSize": 7}` },
		},
		{
			name: "settings.json already on the selected version",
			body: func(e *env) string { return gorootSetting(e, e2eVSCodeVersion) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.fakeVersion(e2eVSCodeVersion)
			path, original := writeVSCodeSettings(t, e, tt.body(e))

			res := e.run("use", e2eVSCodeVersion)

			if !res.Succeeded() {
				t.Fatalf("goenv use failed (exit %d)\n%s", res.ExitCode, truncate(res.Output(), 2000))
			}
			assertUnchanged(t, path, original, res)

			if strings.Contains(res.Output(), "VS Code") {
				t.Errorf("goenv mentioned VS Code when there was nothing to do; "+
					"this is log noise on every CI run in such a repo:\n%s",
					truncate(res.Output(), 2000))
			}
		})
	}
}

// TestUse_NonInteractive_OptInsStillApply guards the other direction: the fix
// must not disable the documented ways to opt in to automatic updating, or it
// would break the automations it was meant to protect.
func TestUse_NonInteractive_OptInsStillApply(t *testing.T) {
	tests := []struct {
		name string
		env  map[string]string
		args []string
	}{
		{
			name: "GOENV_VSCODE_AUTO_SYNC=1",
			env:  map[string]string{"GOENV_VSCODE_AUTO_SYNC": "1"},
			args: []string{"use", e2eVSCodeVersion},
		},
		{
			name: "explicit --vscode flag",
			args: []string{"use", e2eVSCodeVersion, "--vscode"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			e := newEnv(t)
			e.Set("CI", "true")
			for k, v := range tt.env {
				e.Set(k, v)
			}
			e.fakeVersion(e2eVSCodeVersion)
			path, original := writeVSCodeSettings(t, e, gorootSetting(e, e2eVSCodeStaleVersion))

			res := e.run(tt.args...)

			if !res.Succeeded() {
				t.Fatalf("goenv use failed (exit %d)\n%s", res.ExitCode, truncate(res.Output(), 2000))
			}

			current, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("failed to read settings.json: %v", err)
			}
			if string(current) == string(original) {
				t.Errorf("settings.json was NOT updated despite an explicit opt-in; "+
					"the non-interactive guard has broken a supported automation path\n--- output ---\n%s",
					truncate(res.Output(), 2000))
			}
		})
	}
}
