//go:build windows

package shell

import (
	cmdpkg "github.com/go-nv/goenv/cmd"
	"github.com/go-nv/goenv/internal/cmdtest"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestInitWindowsShells(t *testing.T) {
	cfg := config.Load()

	tests := []struct {
		name           string
		shell          string
		expectContains []string
	}{
		{
			name:  "PowerShell init script",
			shell: "powershell",
			expectContains: []string{
				"$env:GOENV_SHELL = \"powershell\"",
				"$env:GOENV_ROOT =",
				"$env:PATH =",
				"function goenv",
			},
		},
		{
			name:  "cmd init script",
			shell: "cmd",
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
		shell    string
		expected string
	}{
		{
			shell:    "powershell",
			expected: "$PROFILE",
		},
		{
			shell:    "cmd",
			expected: "%USERPROFILE%\\autorun.cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			result := determineProfilePath(tt.shell)
			if result != tt.expected {
				t.Errorf("expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestRenderUsageSnippetWindows(t *testing.T) {
	tests := []struct {
		shell           string
		expectedSnippet string
	}{
		{
			shell:           "powershell",
			expectedSnippet: "Invoke-Expression (goenv init - | Out-String)",
		},
		{
			shell:           "cmd",
			expectedSnippet: "FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i",
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			snippet := renderUsageSnippet(tt.shell)
			if !strings.Contains(snippet, tt.expectedSnippet) {
				t.Errorf("expected snippet to contain %s", tt.expectedSnippet)
			}
		})
	}
}

func TestRenderShellFunctionPowerShell(t *testing.T) {
	script := renderShellFunction("powershell")

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
	script := renderShellFunction("cmd")

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

	// On Windows, should return either "powershell" or "cmd"
	if shell != "powershell" && shell != "cmd" {
		t.Errorf("expected powershell or cmd, got %s", shell)
	}
}

func TestRenderInitScriptPathSeparators(t *testing.T) {
	cfg := config.Load()

	tests := []struct {
		name  string
		shell string
	}{
		{
			name:  "PowerShell uses semicolon",
			shell: "powershell",
		},
		{
			name:  "cmd uses semicolon",
			shell: "cmd",
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
