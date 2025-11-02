package utils

import (
	"os"
	"testing"

	"github.com/go-nv/goenv/internal/osinfo"
	"github.com/stretchr/testify/assert"
)

func TestIsMinGW(t *testing.T) {
	// Save original environment
	originalMSYSTEM := os.Getenv(EnvVarMSYSTEM)
	originalMINGWPrefix := os.Getenv(EnvVarMINGWPrefix)
	originalShell := os.Getenv(EnvVarShell)
	defer func() {
		if originalMSYSTEM != "" {
			os.Setenv(EnvVarMSYSTEM, originalMSYSTEM)
		} else {
			os.Unsetenv(EnvVarMSYSTEM)
		}
		if originalMINGWPrefix != "" {
			os.Setenv(EnvVarMINGWPrefix, originalMINGWPrefix)
		} else {
			os.Unsetenv(EnvVarMINGWPrefix)
		}
		if originalShell != "" {
			os.Setenv(EnvVarShell, originalShell)
		} else {
			os.Unsetenv(EnvVarShell)
		}
	}()

	tests := []struct {
		name           string
		msystem        string
		mingwPrefix    string
		shell          string
		expectedResult bool
	}{
		{
			name:           "MSYSTEM set to MINGW64",
			msystem:        "MINGW64",
			mingwPrefix:    "",
			shell:          "",
			expectedResult: osinfo.IsWindows(),
		},
		{
			name:           "MSYSTEM set to MINGW32",
			msystem:        "MINGW32",
			mingwPrefix:    "",
			shell:          "",
			expectedResult: osinfo.IsWindows(),
		},
		{
			name:           "MINGW_PREFIX set",
			msystem:        "",
			mingwPrefix:    "/mingw64",
			shell:          "",
			expectedResult: osinfo.IsWindows(),
		},
		{
			name:           "SHELL contains bash on Windows",
			msystem:        "",
			mingwPrefix:    "",
			shell:          "/usr/bin/bash",
			expectedResult: osinfo.IsWindows(),
		},
		{
			name:           "SHELL contains sh on Windows",
			msystem:        "",
			mingwPrefix:    "",
			shell:          "/bin/sh",
			expectedResult: osinfo.IsWindows(),
		},
		{
			name:           "No MinGW indicators",
			msystem:        "",
			mingwPrefix:    "",
			shell:          "",
			expectedResult: false,
		},
		{
			name:           "PowerShell on Windows (not MinGW)",
			msystem:        "",
			mingwPrefix:    "",
			shell:          "C:\\Windows\\System32\\WindowsPowerShell\\v1.0\\powershell.exe",
			expectedResult: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variables for this test
			if tt.msystem != "" {
				os.Setenv(EnvVarMSYSTEM, tt.msystem)
			} else {
				os.Unsetenv(EnvVarMSYSTEM)
			}
			if tt.mingwPrefix != "" {
				os.Setenv(EnvVarMINGWPrefix, tt.mingwPrefix)
			} else {
				os.Unsetenv(EnvVarMINGWPrefix)
			}
			if tt.shell != "" {
				os.Setenv(EnvVarShell, tt.shell)
			} else {
				os.Unsetenv(EnvVarShell)
			}

			result := IsMinGW()
			assert.Equal(t, tt.expectedResult, result, "IsMinGW() = %v", osinfo.OS())
		})
	}
}

func TestGetEnvValue(t *testing.T) {
	tests := []struct {
		name     string
		env      []string
		key      string
		expected string
	}{
		{
			name:     "key exists",
			env:      []string{"FOO=bar", "BAZ=qux"},
			key:      "FOO",
			expected: "bar",
		},
		{
			name:     "key with equals in value",
			env:      []string{"KEY=value=with=equals"},
			key:      "KEY",
			expected: "value=with=equals",
		},
		{
			name:     "key doesn't exist",
			env:      []string{"FOO=bar"},
			key:      "MISSING",
			expected: "",
		},
		{
			name:     "empty env slice",
			env:      []string{},
			key:      "FOO",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := GetEnvValue(tt.env, tt.key)
			assert.Equal(t, tt.expected, result, "GetEnvValue() =")
		})
	}
}
