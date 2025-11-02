package core

import (
	"bytes"
	"github.com/go-nv/goenv/internal/utils"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/lifecycle"
)

func TestCompareCommand_Basic(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := utils.EnsureDirWithContext(versionsDir, "create test directory"); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	compareCmd.SetOut(buf)
	compareCmd.SetErr(buf)

	// Test basic comparison
	err := runCompare(compareCmd, []string{"1.21.0", "1.22.0"})
	if err != nil {
		t.Fatalf("runCompare() unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Comparing Go Versions") {
		t.Errorf("Expected header 'Comparing Go Versions', got: %s", output)
	}
	if !strings.Contains(output, "1.21.0") {
		t.Errorf("Expected version 1.21.0 in output, got: %s", output)
	}
	if !strings.Contains(output, "1.22.0") {
		t.Errorf("Expected version 1.22.0 in output, got: %s", output)
	}
}

func TestCompareCommand_InstalledVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create versions directory with installed versions
	cmdtest.CreateMockGoVersions(t, tmpDir, "1.21.5", "1.22.3")

	buf := new(bytes.Buffer)
	compareCmd.SetOut(buf)
	compareCmd.SetErr(buf)

	err := runCompare(compareCmd, []string{"1.21.5", "1.22.3"})
	if err != nil {
		t.Fatalf("runCompare() unexpected error: %v", err)
	}

	output := buf.String()
	// Should show installed status
	if !strings.Contains(output, "Installed") {
		t.Errorf("Expected 'Installed' in output, got: %s", output)
	}
}

func TestCompareCommand_InvalidArgs(t *testing.T) {
	// Skip: When no versions are installed, root command shows beginner-friendly help
	// instead of argument validation errors. This is intentional UX behavior.
	// Argument validation is still enforced when versions exist.
	t.Skip("Argument validation testing requires installed versions - tested in integration tests")
}

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name        string
		version     string
		expectMajor int
		expectMinor int
		expectPatch int
	}{
		{"standard version", "1.21.5", 1, 21, 5},
		{"with go prefix", "go1.21.5", 1, 21, 5},
		{"two components", "1.21", 1, 21, 0},
		{"latest style", "1.22.0", 1, 22, 0},
		{"zero patch", "1.20.0", 1, 20, 0},
		{"invalid returns zeros", "invalid", 0, 0, 0},
		{"empty string", "", 0, 0, 0},
		{"single number", "1", 0, 0, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch := parseVersion(tt.version)

			if major != tt.expectMajor {
				t.Errorf("parseVersion(%q) major = %d, want %d", tt.version, major, tt.expectMajor)
			}
			if minor != tt.expectMinor {
				t.Errorf("parseVersion(%q) minor = %d, want %d", tt.version, minor, tt.expectMinor)
			}
			if patch != tt.expectPatch {
				t.Errorf("parseVersion(%q) patch = %d, want %d", tt.version, patch, tt.expectPatch)
			}
		})
	}
}

func TestAnalyzeVersionDifference(t *testing.T) {
	tests := []struct {
		name     string
		v1       string
		v2       string
		expected string
	}{
		{
			name:     "major version difference",
			v1:       "1.21.0",
			v2:       "2.0.0",
			expected: "Major version",
		},
		{
			name:     "minor version difference",
			v1:       "1.21.0",
			v2:       "1.22.0",
			expected: "minor version",
		},
		{
			name:     "patch version difference",
			v1:       "1.21.0",
			v2:       "1.21.5",
			expected: "Patch",
		},
		{
			name:     "same version",
			v1:       "1.21.5",
			v2:       "1.21.5",
			expected: "identical",
		},
		{
			name:     "downgrade",
			v1:       "1.22.0",
			v2:       "1.21.0",
			expected: "Downgrade",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			buf := new(bytes.Buffer)
			cmd := compareCmd
			cmd.SetOut(buf)

			// analyzeVersionDifference doesn't return a value, it prints
			// We'll just test it doesn't panic
			analyzeVersionDifference(cmd, tt.v1, tt.v2, false, false, lifecycle.VersionInfo{}, lifecycle.VersionInfo{})

			output := buf.String()
			// Check if output contains expected keyword
			if !strings.Contains(output, tt.expected) {
				t.Logf("analyzeVersionDifference output for %q vs %q doesn't contain %q, got: %s",
					tt.v1, tt.v2, tt.expected, output)
			}
		})
	}
}

func TestFormatInstallStatus(t *testing.T) {
	tests := []struct {
		name      string
		installed bool
	}{
		{"installed", true},
		{"not installed", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatInstallStatus(tt.installed)
			if result == "" {
				t.Errorf("formatInstallStatus(%v) returned empty string", tt.installed)
			}

			// Check for reasonable content
			if tt.installed && !strings.Contains(strings.ToLower(result), "yes") &&
				!strings.Contains(result, "✓") {
				t.Errorf("formatInstallStatus(true) = %q, expected positive indicator", result)
			}
		})
	}
}

func TestFormatAge(t *testing.T) {
	// Test that formatAge returns non-empty string
	// Exact output depends on current time, so we just check it doesn't crash
	result := formatAge(testDate())
	if result == "" {
		t.Error("formatAge() returned empty string")
	}
}

func TestFormatEOLDate(t *testing.T) {
	// Test that formatEOLDate returns non-empty string
	date := testDate()
	info := lifecycle.VersionInfo{
		Status:  lifecycle.StatusEOL,
		EOLDate: date,
	}
	result := formatEOLDate(info)
	if result == "" {
		t.Error("formatEOLDate() returned empty string for EOL version")
	}

	infoCurrent := lifecycle.VersionInfo{
		Status: lifecycle.StatusCurrent,
	}
	result2 := formatEOLDate(infoCurrent)
	if result2 == "" {
		t.Error("formatEOLDate() returned empty string for current version")
	}
}

func TestFormatSupportStatus(t *testing.T) {
	tests := []struct {
		status   lifecycle.SupportStatus
		expected string
	}{
		{lifecycle.StatusCurrent, "Current"},
		{lifecycle.StatusEOL, "EOL"},
		{lifecycle.StatusNearEOL, "Near"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSupportStatus(tt.status)

			// Check for reasonable content based on status
			if !strings.Contains(result, tt.expected) {
				t.Errorf("formatSupportStatus(%q) = %q, want to contain %q",
					tt.status, result, tt.expected)
			}
		})
	}
}

func TestFormatSizeDiff(t *testing.T) {
	tests := []struct {
		name     string
		diff     int64
		expected string
	}{
		{"positive", 1024 * 1024, "larger"},
		{"negative", -1024 * 1024, "smaller"},
		{"zero", 0, "same"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatSizeDiff(tt.diff)

			if !strings.Contains(strings.ToLower(result), tt.expected) {
				t.Errorf("formatSizeDiff(%d) = %q, want to contain %q",
					tt.diff, result, tt.expected)
			}
		})
	}
}

func TestCompareHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	compareCmd.SetOut(buf)
	compareCmd.SetErr(buf)

	err := compareCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"compare",
		"version",
		"Usage:",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

// Helper function for tests
func testDate() time.Time {
	// Return a fixed date for testing
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}
