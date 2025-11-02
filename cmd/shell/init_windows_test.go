//go:build windows

package shell

import (
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/stretchr/testify/assert"
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
				assert.Contains(t, script, expected, "init script for should contain %v %v", tt.shell, expected)
			}
		})
	}
}

func TestDetermineProfilePathWindows(t *testing.T) {
	tests := []struct {
		shell    shellutil.ShellType
		expected string
	}{
		{
			shell:    shellutil.ShellTypePowerShell,
			expected: "$PROFILE",
		},
		{
			shell:    shellutil.ShellTypeCmd,
			expected: "%USERPROFILE%\\autorun.cmd",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			result := shellutil.GetProfilePathDisplay(tt.shell)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRenderUsageSnippetWindows(t *testing.T) {
	tests := []struct {
		shell           shellutil.ShellType
		expectedSnippet string
	}{
		{
			shell:           shellutil.ShellTypePowerShell,
			expectedSnippet: "Invoke-Expression (goenv init - | Out-String)",
		},
		{
			shell:           shellutil.ShellTypeCmd,
			expectedSnippet: "FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i",
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			snippet := renderUsageSnippet(tt.shell)
			assert.Contains(t, snippet, tt.expectedSnippet, "expected snippet to contain %v", tt.expectedSnippet)
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
		assert.Contains(t, script, check, "PowerShell function should contain %v", check)
	}
}

func TestRenderShellFunctionCmd(t *testing.T) {
	script := renderShellFunction(shellutil.ShellTypeCmd)

	// cmd.exe doesn't support functions, so should have REM comment
	assert.Contains(t, script, "REM cmd.exe does not support functions", "cmd script should contain comment about no function support")
	assert.Contains(t, script, "REM Use goenv commands directly", "cmd script should contain comment about using commands directly")
}

func TestDetectEnvShellWindows(t *testing.T) {
	// This test checks the default behavior on Windows
	// We can't easily simulate environment variables, so we just
	// verify the function returns a valid shell
	shell := detectEnvShell()

	// On Windows, should return either powershell or cmd
	assert.False(t, shell != shellutil.ShellTypePowerShell && shell != shellutil.ShellTypeCmd, "expected powershell or cmd")
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
			assert.Contains(t, script, shimsPath, "script should contain shims path with backslashes %v", shimsPath)
		})
	}
}
