//go:build windows

package shell

import (
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/shellutil"
)

func TestInitWindowsShells(t *testing.T) {
	cfg := config.Load()

	tests := []struct {
		name           string
		shell          shellutil.ShellType
		expectContains []string
	}{
		{
			name:  "PowerShell init script",
			shell: shellutil.ShellTypePowerShell,
			expectContains: []string{
				"$env:GOENV_SHELL = \"powershell\"",
				"$env:GOENV_ROOT =",
				"$env:PATH =",
				"function goenv",
			},
		},
		{
			name:  "cmd init script",
			shell: shellutil.ShellTypeCmd,
			expectContains: []string{
				"SET GOENV_SHELL=cmd",
				"SET GOENV_ROOT=",
				"SET PATH=",
				"REM cmd.exe does not support functions",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := renderInitScript(tt.shell, cfg, false)

			for _, expected := range tt.expectContains {
				if !strings.Contains(script, expected) {
					t.Errorf("init script for %s should contain: %s", tt.shell, expected)
				}
			}
		})
	}
}

func TestDetermineProfilePathWindows(t *testing.T) {
	tests := []struct {
		shell         shellutil.ShellType
		originalShell string
		expected      string
	}{
		{
			shell:         shellutil.ShellTypePowerShell,
			originalShell: "powershell",
			expected:      "$PROFILE",
		},
		{
			shell:         shellutil.ShellTypeCmd,
			originalShell: "cmd",
			expected:      "%USERPROFILE%\\autorun.cmd",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			result := determineProfilePath(tt.shell, tt.originalShell)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRenderUsageSnippetWindows(t *testing.T) {
	tests := []struct {
		shell           shellutil.ShellType
		originalShell   string
		expectedSnippet string
	}{
		{
			shell:           shellutil.ShellTypePowerShell,
			originalShell:   "powershell",
			expectedSnippet: "Invoke-Expression (goenv init - | Out-String)",
		},
		{
			shell:           shellutil.ShellTypeCmd,
			originalShell:   "cmd",
			expectedSnippet: "FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			snippet := renderUsageSnippet(tt.shell, tt.originalShell)
			if !strings.Contains(snippet, tt.expectedSnippet) {
				t.Errorf("expected snippet to contain %s", tt.expectedSnippet)
			}
		})
	}
}

func TestRenderShellFunctionPowerShell(t *testing.T) {
	script := renderShellFunction(shellutil.ShellTypePowerShell)

	checks := []string{
		"function goenv {",
		"$command = $args[0]",
		"switch ($command)",
		"\"rehash\"",
		"\"shell\"",
		"Invoke-Expression",
		"default {",
		"& goenv $command @restArgs",
	}

	for _, check := range checks {
		if !strings.Contains(script, check) {
			t.Errorf("PowerShell function should contain: %s", check)
		}
	}
}

func TestRenderShellFunctionCmd(t *testing.T) {
	script := renderShellFunction(shellutil.ShellTypeCmd)

	// cmd.exe doesn't support functions, so should have REM comment
	if !strings.Contains(script, "REM cmd.exe does not support functions") {
		t.Error("cmd script should contain comment about no function support")
	}
	if !strings.Contains(script, "REM Use goenv commands directly") {
		t.Error("cmd script should contain comment about using commands directly")
	}
}

func TestDetectEnvShellWindows(t *testing.T) {
	// This test checks the default behavior on Windows
	// We can't easily simulate environment variables, so we just
	// verify the function returns a valid shell
	shell := detectEnvShell()

	// On Windows, should return either powershell or cmd
	if shell != shellutil.ShellTypePowerShell && shell != shellutil.ShellTypeCmd {
		t.Errorf("expected powershell or cmd, got %s", shell)
	}
}

func TestRenderInitScriptPathSeparators(t *testing.T) {
	cfg := config.Load()

	tests := []struct {
		name  string
		shell shellutil.ShellType
	}{
		{
			name:  "PowerShell uses semicolon",
			shell: shellutil.ShellTypePowerShell,
		},
		{
			name:  "cmd uses semicolon",
			shell: shellutil.ShellTypeCmd,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := renderInitScript(tt.shell, cfg, false)

			// Windows uses semicolon for PATH separator
			shimsPath := strings.Replace(cfg.ShimsDir(), "/", "\\", -1)
			if !strings.Contains(script, shimsPath) {
				t.Errorf("script should contain shims path with backslashes: %s", shimsPath)
			}
		})
	}
}
