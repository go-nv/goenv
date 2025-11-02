package meta

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExploreCommand_AllCategories(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{})
	require.NoError(t, err, "runExplore() unexpected error")

	output := buf.String()

	// Should show category keywords (actual format uses lowercase with hyphens)
	expectedCategories := []string{
		"getting-started",
		"versions",
		"tools",
		"diagnostics",
		"shell",
		"advanced",
	}

	for _, category := range expectedCategories {
		assert.Contains(t, output, category, "Expected category in output %v %v", category, output)
	}
}

func TestExploreCommand_SpecificCategory(t *testing.T) {
	tests := []struct {
		category         string
		expectedCommands []string
	}{
		{
			category:         "getting-started",
			expectedCommands: []string{"get-started", "setup"},
		},
		{
			category:         "versions",
			expectedCommands: []string{"install", "global", "list"},
		},
		{
			category:         "tools",
			expectedCommands: []string{"tools"},
		},
		{
			category:         "diagnostics",
			expectedCommands: []string{"doctor", "status"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.category, func(t *testing.T) {
			buf := new(bytes.Buffer)
			exploreCmd.SetOut(buf)
			exploreCmd.SetErr(buf)

			err := runExplore(exploreCmd, []string{tt.category})
			require.NoError(t, err, "runExplore() unexpected error")

			output := buf.String()

			for _, cmd := range tt.expectedCommands {
				assert.Contains(t, output, cmd, "Expected command in category output %v %v %v", cmd, tt.category, output)
			}
		})
	}
}

func TestExploreCommand_InvalidCategory(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{"nonexistent-category"})
	assert.Error(t, err, "Expected error for invalid category, got nil")

	// Error should mention the invalid category
	assert.True(t, strings.Contains(err.Error(), "nonexistent-category") || strings.Contains(err.Error(), "unknown"), "Expected error to mention invalid category")
}

func TestExploreCommand_CaseSensitive(t *testing.T) {
	// Test that category names are case-sensitive (lowercase required)
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	// Lowercase should work
	err := runExplore(exploreCmd, []string{"versions"})
	require.NoError(t, err, "runExplore(\\\"versions\\\") unexpected error")

	output := buf.String()
	assert.Contains(t, output, "install", "Expected 'install' command in output %v", output)

	// Uppercase should error
	buf.Reset()
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err = runExplore(exploreCmd, []string{"VERSIONS"})
	assert.Error(t, err, "Expected error for uppercase category, got nil")
}

func TestExploreCommand_PartialMatch(t *testing.T) {
	tests := []struct {
		input            string
		expectedCategory string
	}{
		{"getting", "getting-started"},
		{"started", "getting-started"},
		{"diag", "diagnostics"},
		{"tools", "tools"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			buf := new(bytes.Buffer)
			exploreCmd.SetOut(buf)
			exploreCmd.SetErr(buf)

			err := runExplore(exploreCmd, []string{tt.input})
			// May or may not match depending on implementation
			_ = err

			output := buf.String()
			// Just verify it doesn't crash
			assert.False(t, output == "" && err == nil, "Expected either output or error for partial match")
		})
	}
}

func TestExploreHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := exploreCmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()
	expectedStrings := []string{
		"explore",
		"command",
		"category",
		"discover",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
	}
}

func TestExploreCommand_ShowsUsefulInfo(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{})
	require.NoError(t, err, "runExplore() unexpected error")

	output := buf.String()

	// Should show helpful information
	assert.Contains(t, output, "goenv", "Output should mention 'goenv'")

	// Should have some structure (not just plain text dump)
	assert.Contains(t, output, "\n\n", "Output should have paragraph breaks for readability")
}

func TestExploreCommand_MultipleArgs(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	// Should handle multiple args (probably use first or error)
	err := runExplore(exploreCmd, []string{"version", "tools"})
	// Behavior depends on implementation - just ensure no panic
	_ = err

	// Just verify it doesn't crash
	output := buf.String()
	assert.False(t, output == "" && err == nil, "Expected output or error for multiple args")
}

func TestExploreCommand_FormattingConsistency(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{"versions"})
	require.NoError(t, err, "runExplore() unexpected error")

	output := buf.String()

	// Output should have consistent formatting
	// Look for command names (they should be highlighted/formatted)
	lines := strings.Split(output, "\n")
	commandLineCount := 0
	for _, line := range lines {
		if strings.Contains(line, "goenv") {
			commandLineCount++
		}
	}

	assert.NotEqual(t, 0, commandLineCount, "Expected at least one goenv command in output")
}
