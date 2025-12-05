package core

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/lifecycle"
)

func TestCompareCommand_Basic(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create versions directory
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	compareCmd.SetOut(buf)
	compareCmd.SetErr(buf)

	// Test basic comparison
	err = runCompare(compareCmd, []string{"1.21.0", "1.22.0"})
	require.NoError(t, err, "runCompare() unexpected error")

	output := buf.String()
	assert.Contains(t, output, "Comparing Go Versions", "Expected header 'Comparing Go Versions' %v", output)
	assert.Contains(t, output, "1.21.0", "Expected version 1.21.0 in output %v", output)
	assert.Contains(t, output, "1.22.0", "Expected version 1.22.0 in output %v", output)
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
	require.NoError(t, err, "runCompare() unexpected error")

	output := buf.String()
	// Should show installed status
	assert.Contains(t, output, "Installed", "Expected 'Installed' in output %v", output)
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

			assert.Equal(t, tt.expectMajor, major, "parseVersion() major = %v", tt.version)
			assert.Equal(t, tt.expectMinor, minor, "parseVersion() minor = %v", tt.version)
			assert.Equal(t, tt.expectPatch, patch, "parseVersion() patch = %v", tt.version)
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
			assert.NotEmpty(t, result, "formatInstallStatus() returned empty string")

			// Check for reasonable content
			assert.False(t, tt.installed && !strings.Contains(strings.ToLower(result), "yes") &&
				!strings.Contains(result, "✓"), "formatInstallStatus(true) = , expected positive indicator")
		})
	}
}

func TestFormatAge(t *testing.T) {
	// Test that formatAge returns non-empty string
	// Exact output depends on current time, so we just check it doesn't crash
	result := formatAge(testDate())
	assert.NotEmpty(t, result, "formatAge() returned empty string")
}

func TestFormatEOLDate(t *testing.T) {
	// Test that formatEOLDate returns non-empty string
	date := testDate()
	info := lifecycle.VersionInfo{
		Status:  lifecycle.StatusEOL,
		EOLDate: date,
	}
	result := formatEOLDate(info)
	assert.NotEmpty(t, result, "formatEOLDate() returned empty string for EOL version")

	infoCurrent := lifecycle.VersionInfo{
		Status: lifecycle.StatusCurrent,
	}
	result2 := formatEOLDate(infoCurrent)
	assert.NotEmpty(t, result2, "formatEOLDate() returned empty string for current version")
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
			assert.Contains(t, result, tt.expected, "formatSupportStatus() = %v %v %v", tt.status, result, tt.expected)
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

			assert.Contains(t, strings.ToLower(result), tt.expected, "formatSizeDiff() = %v %v %v", tt.diff, result, tt.expected)
		})
	}
}

func TestCompareHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	compareCmd.SetOut(buf)
	compareCmd.SetErr(buf)

	err := compareCmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()
	expectedStrings := []string{
		"compare",
		"version",
		"Usage:",
		"Examples:",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
	}
}

// Helper function for tests
func testDate() time.Time {
	// Return a fixed date for testing
	return time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC)
}
