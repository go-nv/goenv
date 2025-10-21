package manager

import (
	"os"
	"strings"
	"testing"
)

func TestValidateVersionString(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		shouldError bool
		errorText   string
	}{
		// Valid versions
		{
			name:        "Valid version 1.21.0",
			version:     "1.21.0",
			shouldError: false,
		},
		{
			name:        "Valid version 1.22.1",
			version:     "1.22.1",
			shouldError: false,
		},
		{
			name:        "Valid version with text 1.21rc1",
			version:     "1.21rc1",
			shouldError: false,
		},
		{
			name:        "Valid system keyword",
			version:     "system",
			shouldError: false,
		},
		{
			name:        "Valid latest keyword",
			version:     "latest",
			shouldError: false,
		},
		{
			name:        "Valid beta version",
			version:     "1.22beta1",
			shouldError: false,
		},

		// Path traversal attacks
		{
			name:        "Path traversal with ..",
			version:     "../etc/passwd",
			shouldError: true,
			errorText:   "path traversal",
		},
		{
			name:        "Path traversal with .. in middle",
			version:     "1.21/../1.22",
			shouldError: true,
			errorText:   "path traversal",
		},
		{
			name:        "Path traversal double dot",
			version:     "1.21..",
			shouldError: true,
			errorText:   "path traversal",
		},

		// Absolute paths
		{
			name:        "Unix absolute path",
			version:     "/usr/local/go",
			shouldError: true,
			errorText:   "absolute path",
		},
		{
			name:        "Windows absolute path",
			version:     "\\Windows\\System32",
			shouldError: true,
			errorText:   "absolute path",
		},
		{
			name:        "Windows drive letter C:",
			version:     "C:\\Go",
			shouldError: true,
			errorText:   "drive letter",
		},
		{
			name:        "Windows drive letter D:",
			version:     "D:\\Programs\\Go",
			shouldError: true,
			errorText:   "drive letter",
		},

		// Path separators
		{
			name:        "Unix path separator",
			version:     "1.21/bin",
			shouldError: true,
			errorText:   "path separators",
		},
		{
			name:        "Windows path separator",
			version:     "1.21\\bin",
			shouldError: true,
			errorText:   "path separators",
		},

		// Hidden files
		{
			name:        "Hidden file .hidden",
			version:     ".hidden",
			shouldError: true,
			errorText:   "cannot start with dot",
		},
		{
			name:        "Hidden file .go-version",
			version:     ".go-version",
			shouldError: true,
			errorText:   "cannot start with dot",
		},

		// Null bytes (path truncation)
		{
			name:        "Null byte attack",
			version:     "1.21\x00/etc/passwd",
			shouldError: true,
			errorText:   "", // Will catch either path separator or null byte, both are valid
		},

		// Control characters
		{
			name:        "Newline character",
			version:     "1.21\n1.22",
			shouldError: true,
			errorText:   "invalid character",
		},
		{
			name:        "Tab character",
			version:     "1.21\t1.22",
			shouldError: true,
			errorText:   "invalid character",
		},
		{
			name:        "Carriage return",
			version:     "1.21\r",
			shouldError: true,
			errorText:   "invalid character",
		},
		{
			name:        "DEL character",
			version:     "1.21\x7F",
			shouldError: true,
			errorText:   "invalid character",
		},

		// Empty string
		{
			name:        "Empty string",
			version:     "",
			shouldError: true,
			errorText:   "cannot be empty",
		},

		// Excessive length
		{
			name:        "Excessive length (256 chars)",
			version:     strings.Repeat("a", 256),
			shouldError: true,
			errorText:   "too long",
		},
		{
			name:        "Maximum valid length (255 chars)",
			version:     strings.Repeat("1", 255),
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVersionString(tt.version)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error for version %q, but got none", tt.version)
				} else if tt.errorText != "" && !strings.Contains(err.Error(), tt.errorText) {
					t.Errorf("Expected error to contain %q, got: %v", tt.errorText, err)
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error for version %q, but got: %v", tt.version, err)
				}
			}
		})
	}
}

func TestValidateVersionStringCVE202235861(t *testing.T) {
	// Specific test cases from CVE-2022-35861
	// The vulnerability allowed path traversal in version files
	maliciousVersions := []string{
		"..",
		"../../../etc/passwd",
		"1.21/../../../root",
		"./local",
		"../parent",
	}

	for _, version := range maliciousVersions {
		t.Run("CVE-2022-35861: "+version, func(t *testing.T) {
			err := validateVersionString(version)
			if err == nil {
				t.Errorf("CVE-2022-35861: Expected error for malicious version %q, but validation passed", version)
			}
		})
	}
}

func TestValidateVersionStringEdgeCases(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		shouldError bool
	}{
		{
			name:        "Version with hyphen",
			version:     "1.21-beta",
			shouldError: false,
		},
		{
			name:        "Version with underscore",
			version:     "1.21_alpha",
			shouldError: false,
		},
		{
			name:        "Version with plus",
			version:     "1.21+patch",
			shouldError: false,
		},
		{
			name:        "Only numbers",
			version:     "12345",
			shouldError: false,
		},
		{
			name:        "Single character",
			version:     "1",
			shouldError: false,
		},
		{
			name:        "Version with space (should fail - control character)",
			version:     "1.21 beta",
			shouldError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateVersionString(tt.version)

			if tt.shouldError && err == nil {
				t.Errorf("Expected error for version %q, but got none", tt.version)
			} else if !tt.shouldError && err != nil {
				t.Errorf("Expected no error for version %q, but got: %v", tt.version, err)
			}
		})
	}
}

func TestParseGoModVersion(t *testing.T) {
	tests := []struct {
		name          string
		goModContent  string
		expected      string
		shouldError   bool
		errorContains string
	}{
		{
			name:         "Standard go.mod",
			goModContent: "module example\n\ngo 1.24.3\n\nrequire (\n\tgithub.com/pkg v1.0.0\n)\n",
			expected:     "1.24.3",
			shouldError:  false,
		},
		{
			name:         "go.mod with toolchain",
			goModContent: "module example\n\ngo 1.24.1\n\ntoolchain go1.24.3\n",
			expected:     "1.24.3", // toolchain takes precedence
			shouldError:  false,
		},
		{
			name:         "go.mod with inline comment",
			goModContent: "module example\n\ngo 1.24.3 // Go version\n",
			expected:     "1.24.3",
			shouldError:  false,
		},
		{
			name:         "go.mod with comment line",
			goModContent: "module example\n\n// Minimum Go version\ngo 1.24.3\n",
			expected:     "1.24.3",
			shouldError:  false,
		},
		{
			name:         "Minimal go.mod",
			goModContent: "go 1.24.3",
			expected:     "1.24.3",
			shouldError:  false,
		},
		{
			name:         "go.mod with extra whitespace",
			goModContent: "module example\n\ngo    1.24.3   \n",
			expected:     "1.24.3",
			shouldError:  false,
		},
		{
			name:          "go.mod without go directive",
			goModContent:  "module example\n\nrequire (\n\tgithub.com/pkg v1.0.0\n)\n",
			expected:      "",
			shouldError:   true,
			errorContains: "no go version directive",
		},
		{
			name:          "Empty go.mod",
			goModContent:  "",
			expected:      "",
			shouldError:   true,
			errorContains: "no go version directive",
		},
		{
			name:          "Invalid go directive",
			goModContent:  "module example\n\ngo\n",
			expected:      "",
			shouldError:   true,
			errorContains: "no go version directive",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp file
			tmpDir := t.TempDir()
			goModPath := tmpDir + "/go.mod"
			if err := os.WriteFile(goModPath, []byte(tt.goModContent), 0644); err != nil {
				t.Fatalf("Failed to create test file: %v", err)
			}

			result, err := ParseGoModVersion(goModPath)

			if tt.shouldError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing %q, got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if result != tt.expected {
					t.Errorf("Expected version %q, got %q", tt.expected, result)
				}
			}
		})
	}
}

func TestVersionSatisfies(t *testing.T) {
	tests := []struct {
		name     string
		current  string
		required string
		expected bool
	}{
		// Equal versions
		{name: "Equal major.minor.patch", current: "1.24.3", required: "1.24.3", expected: true},
		{name: "Equal with v prefix", current: "v1.24.3", required: "v1.24.3", expected: true},
		{name: "Equal mixed prefix", current: "v1.24.3", required: "1.24.3", expected: true},

		// Current is newer (should satisfy)
		{name: "Newer patch", current: "1.24.5", required: "1.24.3", expected: true},
		{name: "Newer minor", current: "1.25.0", required: "1.24.3", expected: true},
		{name: "Newer major", current: "2.0.0", required: "1.24.3", expected: true},
		{name: "Much newer", current: "1.24.10", required: "1.24.3", expected: true},

		// Current is older (should NOT satisfy)
		{name: "Older patch", current: "1.24.1", required: "1.24.3", expected: false},
		{name: "Older minor", current: "1.23.5", required: "1.24.3", expected: false},
		{name: "Older major", current: "0.99.0", required: "1.24.3", expected: false},

		// Missing components (treat as .0)
		{name: "Current missing patch", current: "1.24", required: "1.24.3", expected: false},
		{name: "Required missing patch", current: "1.24.5", required: "1.24", expected: true},
		{name: "Both missing patch equal", current: "1.24", required: "1.24", expected: true},
		{name: "Current missing minor/patch", current: "1", required: "1.24.3", expected: false},
		{name: "Required missing minor/patch", current: "1.24.3", required: "1", expected: true},

		// Version suffixes (should be ignored for comparison)
		{name: "Current with rc suffix", current: "1.24.3-rc1", required: "1.24.3", expected: true},
		{name: "Required with rc suffix", current: "1.24.3", required: "1.24.3-rc1", expected: true},
		{name: "Current with beta", current: "1.24.3beta1", required: "1.24.3", expected: true},
		{name: "Both with suffixes equal", current: "1.24.3-rc1", required: "1.24.3-rc2", expected: true},

		// Edge cases
		{name: "Zero versions", current: "0.0.0", required: "0.0.0", expected: true},
		{name: "Large numbers", current: "1.999.999", required: "1.999.998", expected: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := VersionSatisfies(tt.current, tt.required)
			if result != tt.expected {
				t.Errorf("VersionSatisfies(%q, %q) = %v, expected %v",
					tt.current, tt.required, result, tt.expected)
			}
		})
	}
}
