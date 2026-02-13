package integrations

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCISetupPowerShellQuoting(t *testing.T) {
	var err error
	tests := []struct {
		name         string
		goenvRoot    string
		expectQuoted bool
		skipOnNonWin bool
	}{
		{
			name:         "GOENV_ROOT with spaces",
			goenvRoot:    filepath.Join(t.TempDir(), "path with spaces"),
			expectQuoted: true,
			skipOnNonWin: true,
		},
		{
			name:         "GOENV_ROOT without spaces",
			goenvRoot:    filepath.Join(t.TempDir(), "normalpath"),
			expectQuoted: false,
			skipOnNonWin: false,
		},
		{
			name:         "GOENV_ROOT with special chars",
			goenvRoot:    filepath.Join(t.TempDir(), "path-with_special.chars"),
			expectQuoted: false,
			skipOnNonWin: false,
		},
		{
			name:         "GOENV_ROOT with parentheses",
			goenvRoot:    filepath.Join(t.TempDir(), "path (with) parens"),
			expectQuoted: true,
			skipOnNonWin: true,
		},
		{
			name:         "GOENV_ROOT with single quote",
			goenvRoot:    filepath.Join(t.TempDir(), "path'with'quote"),
			expectQuoted: true,
			skipOnNonWin: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.skipOnNonWin && !utils.IsWindows() {
				t.Skip("Skipping Windows-only test on non-Windows platform")
			}

			// Create test directory
			err = utils.EnsureDirWithContext(tt.goenvRoot, "create test directory")
			require.NoError(t, err, "Failed to create test directory")
			defer os.RemoveAll(tt.goenvRoot)

			// Create config with test root
			cfg := &config.Config{
				Root: tt.goenvRoot,
			}

			// Capture output
			cmd := ciSetupCmd
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			// Set shell to powershell
			ciShell = "powershell"

			// Execute
			outputPowerShell(cmd, cfg)

			output := buf.String()

			// Verify GOENV_ROOT is set correctly
			assert.Contains(t, output, "$env:GOENV_ROOT = '", "Output missing GOENV_ROOT assignment:\\n %v", output)

			// Verify PATH contains properly quoted paths
			assert.Contains(t, output, "$env:PATH = \"", "Output missing PATH assignment:\\n %v", output)

			// Check for proper quote escaping in GOENV_ROOT
			// Single quotes should be doubled for PowerShell
			if strings.Contains(tt.goenvRoot, "'") {
				escapedRoot := strings.ReplaceAll(tt.goenvRoot, "'", "''")
				assert.Contains(t, output, escapedRoot, "Single quotes not properly escaped in GOENV_ROOT:\\nExpected: \\nGot:\\n %v %v", escapedRoot, output)
			}

			// Check for proper backtick escaping in PATH
			// Double quotes should be backtick-escaped
			if strings.Contains(tt.goenvRoot, "\"") {
				escapedPath := strings.ReplaceAll(tt.goenvRoot, "\"", "`\"")
				assert.Contains(t, output, escapedPath, "Double quotes not properly escaped in PATH:\\nExpected: \\nGot:\\n %v %v", escapedPath, output)
			}

			t.Logf("PowerShell output for %s:\n%s", tt.name, output)
		})
	}
}

func TestCISetupPowerShellExecution(t *testing.T) {
	var err error
	if !utils.IsWindows() {
		t.Skip("Skipping PowerShell execution test on non-Windows platform")
	}

	// Check if PowerShell is available
	if _, err := exec.LookPath("powershell.exe"); err != nil {
		t.Skip("PowerShell not available, skipping execution test")
	}

	tests := []struct {
		name      string
		goenvRoot string
	}{
		{
			name:      "Path with spaces",
			goenvRoot: filepath.Join(t.TempDir(), "go env test"),
		},
		{
			name:      "Path with parentheses",
			goenvRoot: filepath.Join(t.TempDir(), "goenv (test)"),
		},
		{
			name:      "Normal path",
			goenvRoot: filepath.Join(t.TempDir(), "goenv-test"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory structure
			err = utils.EnsureDir(filepath.Join(tt.goenvRoot, "bin"))
			require.NoError(t, err, "Failed to create bin directory")
			err = utils.EnsureDir(filepath.Join(tt.goenvRoot, "shims"))
			require.NoError(t, err, "Failed to create shims directory")
			defer os.RemoveAll(tt.goenvRoot)

			// Create config with test root
			cfg := &config.Config{
				Root: tt.goenvRoot,
			}

			// Generate PowerShell script
			cmd := ciSetupCmd
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			ciShell = "powershell"
			outputPowerShell(cmd, cfg)

			script := buf.String()

			// Add test commands to verify the paths work
			script += "\n# Test that paths are accessible\n"
			script += "Write-Host \"GOENV_ROOT: $env:GOENV_ROOT\"\n"
			script += "Write-Host \"PATH contains bin: $($env:PATH -like '*bin*')\"\n"
			script += "Write-Host \"PATH contains shims: $($env:PATH -like '*shims*')\"\n"
			script += "Test-Path $env:GOENV_ROOT\n"

			// Write script to temp file
			scriptFile := filepath.Join(t.TempDir(), "test-script.ps1")
			testutil.WriteTestFile(t, scriptFile, []byte(script), utils.PermFileDefault)

			// Execute PowerShell script
			cmdExec := exec.Command("powershell.exe", "-NoProfile", "-ExecutionPolicy", "Bypass", "-File", scriptFile)
			output, err := cmdExec.CombinedOutput()
			assert.NoError(t, err, "PowerShell execution failed: \\nOutput:\\n\\nScript:\\n")

			outputStr := string(output)
			t.Logf("PowerShell execution output:\n%s", outputStr)

			// Verify the script executed successfully
			assert.Contains(t, outputStr, "GOENV_ROOT:", "GOENV_ROOT not echoed in output:\\n %v", outputStr)
			assert.Contains(t, outputStr, "True", "Test-Path did not return True, path may not be accessible:\\n %v", outputStr)
			assert.Contains(t, outputStr, "PATH contains bin: True", "PATH does not contain bin directory:\\n %v", outputStr)
			assert.Contains(t, outputStr, "PATH contains shims: True", "PATH does not contain shims directory:\\n %v", outputStr)
		})
	}
}

func TestCISetupShellFormats(t *testing.T) {
	tests := []struct {
		name         string
		shell        string
		expectEnvVar string
		expectPath   string
	}{
		{
			name:         "bash format",
			shell:        "bash",
			expectEnvVar: "export GOENV_ROOT=",
			expectPath:   "export PATH=",
		},
		{
			name:         "zsh format",
			shell:        "zsh",
			expectEnvVar: "export GOENV_ROOT=",
			expectPath:   "export PATH=",
		},
		{
			name:         "fish format",
			shell:        "fish",
			expectEnvVar: "set -gx GOENV_ROOT",
			expectPath:   "set -gx PATH",
		},
		{
			name:         "powershell format",
			shell:        "powershell",
			expectEnvVar: "$env:GOENV_ROOT = '",
			expectPath:   "$env:PATH = \"",
		},
		{
			name:         "github actions format",
			shell:        "github",
			expectEnvVar: "echo \"GOENV_ROOT=",
			expectPath:   ">> $GITHUB_PATH",
		},
		{
			name:         "gitlab ci format",
			shell:        "gitlab",
			expectEnvVar: "export GOENV_ROOT=",
			expectPath:   "export PATH=",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test directory
			testRoot := t.TempDir()
			cfg := &config.Config{
				Root: testRoot,
			}

			// Capture output
			cmd := ciSetupCmd
			var buf bytes.Buffer
			cmd.SetOut(&buf)

			// Set shell format
			ciShell = tt.shell

			// Execute based on shell type
			switch tt.shell {
			case "powershell":
				outputPowerShell(cmd, cfg)
			case "github":
				outputGitHubActions(cmd, cfg)
			case "gitlab":
				outputGitLabCI(cmd, cfg)
			case "fish":
				outputFish(cmd, cfg)
			default:
				outputBash(cmd, cfg)
			}

			output := buf.String()

			// Verify expected patterns
			assert.Contains(t, output, tt.expectEnvVar, "Expected environment variable pattern not found in output:\\n %v %v", tt.expectEnvVar, output)
			assert.Contains(t, output, tt.expectPath, "Expected PATH pattern not found in output:\\n %v %v", tt.expectPath, output)

			// Verify root path is in output
			assert.True(t, strings.Contains(output, testRoot) || strings.Contains(output, "\\bin"), "Expected root path or bin reference not found in output:\\n")

			t.Logf("%s output:\n%s", tt.shell, output)
		})
	}
}

func TestCISetupPowerShellSpecialCharacters(t *testing.T) {
	tests := []struct {
		name          string
		goenvRoot     string
		expectEscaped string
		description   string
	}{
		{
			name:          "single quote in path",
			goenvRoot:     "C:\\Users\\john's folder",
			expectEscaped: "john''s folder",
			description:   "Single quotes should be doubled in PowerShell single-quoted strings",
		},
		{
			name:          "double quote in path (theoretical)",
			goenvRoot:     "C:\\Users\\test\\folder",
			expectEscaped: "\\folder",
			description:   "Double quotes should be backtick-escaped in PATH assignment",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Root: tt.goenvRoot,
			}

			// Capture output
			cmd := ciSetupCmd
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			ciShell = "powershell"

			outputPowerShell(cmd, cfg)

			output := buf.String()

			// Verify escaping
			assert.Contains(t, output, tt.expectEscaped, "Expected escaped pattern not found in output:\\n\\nDescription %v %v %v", tt.expectEscaped, output, tt.description)

			t.Logf("PowerShell output for %s:\n%s", tt.description, output)
		})
	}
}
