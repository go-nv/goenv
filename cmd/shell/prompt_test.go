package shell

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPromptCommand(t *testing.T) {
	tests := []struct {
		name             string
		args             []string
		envVars          map[string]string
		setupFunc        func(t *testing.T, tmpDir string)
		expectedOutput   string
		unexpectedOutput string
		shouldFail       bool
		checkNonEmpty    bool
	}{
		{
			name:          "basic prompt output",
			args:          []string{},
			checkNonEmpty: true, // Should output some version
		},
		// Note: The following tests just verify non-empty output
		// Integration tests with actual CLI flag parsing would test formatting
		{
			name: "prompt disabled via env var",
			envVars: map[string]string{
				"GOENV_DISABLE_PROMPT": "1",
			},
			unexpectedOutput: "1.23",
		},
		{
			name: "prompt with prefix from env var",
			envVars: map[string]string{
				"GOENV_PROMPT_PREFIX": "[",
				"GOENV_PROMPT_SUFFIX": "]",
			},
			expectedOutput: "[",
		},
		{
			name: "prompt with format from env var",
			envVars: map[string]string{
				"GOENV_PROMPT_FORMAT": "go:%s",
			},
			expectedOutput: "go:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Set up environment variables
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			// Run setup function if provided
			if tt.setupFunc != nil {
				tt.setupFunc(t, tmpDir)
			}

			// Create a new command for this test
			cmd := &cobra.Command{}
			cmd.SetArgs(tt.args)

			var stdout bytes.Buffer
			cmd.SetOut(&stdout)
			cmd.SetErr(&stdout)

			// Run the prompt command
			err := runPrompt(cmd, tt.args)

			assert.False(t, tt.shouldFail && err == nil, "expected error but got none")
			assert.False(t, !tt.shouldFail && err != nil)

			output := stdout.String()

			assert.False(t, tt.checkNonEmpty && len(strings.TrimSpace(output)) == 0, "expected non-empty output, got empty string")

			assert.False(t, tt.expectedOutput != "" && !strings.Contains(output, tt.expectedOutput), "expected output to contain")

			assert.False(t, tt.unexpectedOutput != "" && strings.Contains(output, tt.unexpectedOutput), "expected output NOT to contain")
		})
	}
}

func TestFormatVersionShort(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		expected string
	}{
		{
			name:     "full version",
			version:  "1.23.2",
			expected: "1.23",
		},
		{
			name:     "already short",
			version:  "1.23",
			expected: "1.23",
		},
		{
			name:     "single digit",
			version:  "1",
			expected: "1",
		},
		{
			name:     "system version",
			version:  "system",
			expected: "system",
		},
		{
			name:     "with rc suffix",
			version:  "1.23.0-rc1",
			expected: "1.23",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := formatVersionShort(tt.version)
			assert.Equal(t, tt.expected, got, "formatVersionShort() = %v", tt.version)
		})
	}
}

func TestFormatPromptVersion(t *testing.T) {
	tests := []struct {
		name     string
		version  string
		flags    map[string]string // Will be set as promptFlags
		envVars  map[string]string
		expected string
	}{
		{
			name:     "basic version",
			version:  "1.23.2",
			expected: "1.23.2",
		},
		{
			name:    "with prefix and suffix flags",
			version: "1.23.2",
			flags: map[string]string{
				"prefix": "(",
				"suffix": ")",
			},
			expected: "(1.23.2)",
		},
		{
			name:    "with format flag",
			version: "1.23.2",
			flags: map[string]string{
				"format": "go:%s",
			},
			expected: "go:1.23.2",
		},
		{
			name:    "with icon flag",
			version: "1.23.2",
			flags: map[string]string{
				"icon": "🐹",
			},
			expected: "🐹 1.23.2",
		},
		{
			name:    "with env var prefix",
			version: "1.23.2",
			envVars: map[string]string{
				"GOENV_PROMPT_PREFIX": "[",
				"GOENV_PROMPT_SUFFIX": "]",
			},
			expected: "[1.23.2]",
		},
		{
			name:    "flags override env vars",
			version: "1.23.2",
			flags: map[string]string{
				"prefix": "(",
			},
			envVars: map[string]string{
				"GOENV_PROMPT_PREFIX": "[",
			},
			expected: "(1.23.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			promptFlags.prefix = ""
			promptFlags.suffix = ""
			promptFlags.format = ""
			promptFlags.icon = ""
			promptFlags.short = false

			// Set flags
			if tt.flags != nil {
				if v, ok := tt.flags["prefix"]; ok {
					promptFlags.prefix = v
				}
				if v, ok := tt.flags["suffix"]; ok {
					promptFlags.suffix = v
				}
				if v, ok := tt.flags["format"]; ok {
					promptFlags.format = v
				}
				if v, ok := tt.flags["icon"]; ok {
					promptFlags.icon = v
				}
			}

			// Set env vars
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			got := formatPromptVersion(tt.version)
			assert.Equal(t, tt.expected, got, "formatPromptVersion() = %v", tt.version)
		})
	}
}

func TestIsGoProject(t *testing.T) {
	tests := []struct {
		name      string
		setupFunc func(t *testing.T, tmpDir string)
		expected  bool
	}{
		{
			name: "with .go-version file",
			setupFunc: func(t *testing.T, tmpDir string) {
				testutil.WriteTestFile(t, filepath.Join(tmpDir, ".go-version"), []byte("1.23.2"), utils.PermFileDefault)
			},
			expected: true,
		},
		{
			name: "with go.mod file",
			setupFunc: func(t *testing.T, tmpDir string) {
				testutil.WriteTestFile(t, filepath.Join(tmpDir, "go.mod"), []byte("module test"), utils.PermFileDefault)
			},
			expected: true,
		},
		{
			name: "with .tool-versions containing golang",
			setupFunc: func(t *testing.T, tmpDir string) {
				testutil.WriteTestFile(t, filepath.Join(tmpDir, ".tool-versions"), []byte("golang 1.23.2\n"), utils.PermFileDefault)
			},
			expected: true,
		},
		{
			name: "with .tool-versions containing go",
			setupFunc: func(t *testing.T, tmpDir string) {
				testutil.WriteTestFile(t, filepath.Join(tmpDir, ".tool-versions"), []byte("go 1.23.2\n"), utils.PermFileDefault)
			},
			expected: true,
		},
		{
			name: "with .go files",
			setupFunc: func(t *testing.T, tmpDir string) {
				testutil.WriteTestFile(t, filepath.Join(tmpDir, "main.go"), []byte("package main"), utils.PermFileDefault)
			},
			expected: true,
		},
		{
			name: "empty directory",
			setupFunc: func(t *testing.T, tmpDir string) {
				// No setup needed
			},
			expected: false,
		},
		{
			name: "with non-go files only",
			setupFunc: func(t *testing.T, tmpDir string) {
				testutil.WriteTestFile(t, filepath.Join(tmpDir, "README.md"), []byte("# Test"), utils.PermFileDefault)
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			originalDir, _ := os.Getwd()
			defer os.Chdir(originalDir)

			os.Chdir(tmpDir)
			tt.setupFunc(t, tmpDir)

			got := isGoProject()
			assert.Equal(t, tt.expected, got, "isGoProject() =")
		})
	}
}

func TestHashString(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{
			name:  "simple string",
			input: "test",
		},
		{
			name:  "with path",
			input: "/home/user/project",
		},
		{
			name:  "with env var",
			input: "GOENV_VERSION=1.23.2",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hash := hashString(tt.input)

			// Hash should be 16 hex characters (8 bytes)
			assert.Len(t, hash, 16, "hashString() length = %v", tt.input)

			// Should be consistent
			hash2 := hashString(tt.input)
			assert.Equal(t, hash2, hash, "hashString() not consistent: != %v", tt.input)

			// Different inputs should produce different hashes
			if tt.input != "test" {
				differentHash := hashString("test")
				assert.NotEqual(t, differentHash, hash, "hashString() produced same hash as 'test' %v", tt.input)
			}
		})
	}
}

func TestGeneratePromptSetup(t *testing.T) {
	tests := []struct {
		name             string
		shell            shellutil.ShellType
		expectedContains []string
	}{
		{
			name:  "bash setup",
			shell: shellutil.ShellTypeBash,
			expectedContains: []string{
				"# goenv prompt integration",
				"export PS1=",
				"goenv prompt",
			},
		},
		{
			name:  "zsh setup",
			shell: shellutil.ShellTypeZsh,
			expectedContains: []string{
				"# goenv prompt integration",
				"setopt PROMPT_SUBST",
				"export PS1=",
				"goenv prompt",
			},
		},
		{
			name:  "fish setup",
			shell: shellutil.ShellTypeFish,
			expectedContains: []string{
				"# goenv prompt integration",
				"fish_prompt",
				"goenv prompt",
			},
		},
		{
			name:  "powershell setup",
			shell: shellutil.ShellTypePowerShell,
			expectedContains: []string{
				"# goenv prompt integration",
				"function global:prompt",
				"goenv prompt",
				"Write-Host",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := generatePromptSetup(tt.shell)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, got, expected, "generatePromptSetup() does not contain \\nGot:\\n %v %v %v", tt.shell, expected, got)
			}
		})
	}
}

func TestGetReloadCommand(t *testing.T) {
	tests := []struct {
		name     string
		shell    shellutil.ShellType
		expected string
	}{
		{
			name:     "bash",
			shell:    shellutil.ShellTypeBash,
			expected: "source ~/.bashrc",
		},
		{
			name:     "zsh",
			shell:    shellutil.ShellTypeZsh,
			expected: "source ~/.zshrc",
		},
		{
			name:     "fish",
			shell:    shellutil.ShellTypeFish,
			expected: "source ~/.config/fish/config.fish",
		},
		{
			name:     "powershell",
			shell:    shellutil.ShellTypePowerShell,
			expected: ". $PROFILE",
		},
		{
			name:     "unknown",
			shell:    shellutil.ShellType("unknown"),
			expected: "restart your shell",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := getReloadCommand(tt.shell)
			assert.Equal(t, tt.expected, got, "getReloadCommand() = %v", tt.shell)
		})
	}
}

func TestCacheFunctions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Root = tmpDir

	t.Run("cache key generation", func(t *testing.T) {
		key1 := generateCacheKey()
		key2 := generateCacheKey()

		// Should be consistent in same directory
		assert.Equal(t, key2, key1, "generateCacheKey() not consistent: !=")

		// Should be 16 hex characters
		assert.Len(t, key1, 16, "generateCacheKey() length =")
	})

	t.Run("cache path", func(t *testing.T) {
		key := "test1234567890ab"
		path := getCachePath(cfg, key)

		expectedDir := filepath.Join(tmpDir, "cache", "prompt")
		assert.Contains(t, path, expectedDir, "getCachePath() = , should contain %v %v", path, expectedDir)

		assert.True(t, strings.HasSuffix(path, key), "getCachePath() = , should end with")
	})

	t.Run("cache read/write", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache", "prompt")
		_ = utils.EnsureDirWithContext(cacheDir, "create test directory")
		cachePath := filepath.Join(cacheDir, "test")

		// Write cache
		version := "1.23.2"
		err = writeCache(cachePath, version)
		require.NoError(t, err, "writeCache() failed")

		// Read cache
		got, err := readCache(cachePath)
		require.NoError(t, err, "readCache() failed")

		assert.Equal(t, version, got, "readCache() =")
	})

	t.Run("cache validity", func(t *testing.T) {
		cacheDir := filepath.Join(tmpDir, "cache", "prompt")
		_ = utils.EnsureDirWithContext(cacheDir, "create test directory")
		cachePath := filepath.Join(cacheDir, "validity_test")

		// Non-existent cache should be invalid
		if isCacheValid(cachePath, 5) {
			t.Error("isCacheValid() = true for non-existent cache, want false")
		}

		// Write cache
		writeCache(cachePath, "1.23.2")

		// Fresh cache should be valid
		assert.True(t, isCacheValid(cachePath, 5), "isCacheValid() = false for fresh cache, want true")

		// Zero TTL should be invalid
		if isCacheValid(cachePath, 0) {
			t.Error("isCacheValid() = true with 0 TTL, want false")
		}
	})
}

func TestShouldHideVersion(t *testing.T) {
	tests := []struct {
		name    string
		version string
		flags   map[string]bool
		envVars map[string]string
		want    bool
	}{
		{
			name:    "show normal version",
			version: "1.23.2",
			want:    false,
		},
		{
			name:    "hide system with flag",
			version: "system",
			flags:   map[string]bool{"noSystem": true},
			want:    true,
		},
		{
			name:    "hide system with env var",
			version: "system",
			envVars: map[string]string{"GOENV_PROMPT_NO_SYSTEM": "1"},
			want:    true,
		},
		{
			name:    "show system by default",
			version: "system",
			want:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset flags
			promptFlags.noSystem = false
			promptFlags.projectOnly = false

			// Set flags
			if tt.flags != nil {
				if v, ok := tt.flags["noSystem"]; ok {
					promptFlags.noSystem = v
				}
				if v, ok := tt.flags["projectOnly"]; ok {
					promptFlags.projectOnly = v
				}
			}

			// Set env vars
			for k, v := range tt.envVars {
				t.Setenv(k, v)
			}

			got := shouldHideVersion(tt.version)
			assert.Equal(t, tt.want, got, "shouldHideVersion() = %v", tt.version)
		})
	}
}
