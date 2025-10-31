package meta

import (
	"bytes"
	"strings"
	"testing"
)

func TestExploreCommand_AllCategories(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{})
	if err != nil {
		t.Fatalf("runExplore() unexpected error: %v", err)
	}

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
		if !strings.Contains(output, category) {
			t.Errorf("Expected category %q in output, got: %s", category, output)
		}
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
			if err != nil {
				t.Fatalf("runExplore(%q) unexpected error: %v", tt.category, err)
			}

			output := buf.String()

			for _, cmd := range tt.expectedCommands {
				if !strings.Contains(output, cmd) {
					t.Errorf("Expected command %q in category %q output, got: %s",
						cmd, tt.category, output)
				}
			}
		})
	}
}

func TestExploreCommand_InvalidCategory(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{"nonexistent-category"})
	if err == nil {
		t.Error("Expected error for invalid category, got nil")
	}

	// Error should mention the invalid category
	if !strings.Contains(err.Error(), "nonexistent-category") &&
		!strings.Contains(err.Error(), "unknown") {
		t.Errorf("Expected error to mention invalid category, got: %v", err)
	}
}

func TestExploreCommand_CaseSensitive(t *testing.T) {
	// Test that category names are case-sensitive (lowercase required)
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	// Lowercase should work
	err := runExplore(exploreCmd, []string{"versions"})
	if err != nil {
		t.Fatalf("runExplore(\"versions\") unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "install") {
		t.Errorf("Expected 'install' command in output, got: %s", output)
	}

	// Uppercase should error
	buf.Reset()
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err = runExplore(exploreCmd, []string{"VERSIONS"})
	if err == nil {
		t.Error("Expected error for uppercase category, got nil")
	}
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
			if output == "" && err == nil {
				t.Error("Expected either output or error for partial match")
			}
		})
	}
}

func TestExploreHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := exploreCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"explore",
		"command",
		"category",
		"discover",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestExploreCommand_ShowsUsefulInfo(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{})
	if err != nil {
		t.Fatalf("runExplore() unexpected error: %v", err)
	}

	output := buf.String()

	// Should show helpful information
	if !strings.Contains(output, "goenv") {
		t.Error("Output should mention 'goenv'")
	}

	// Should have some structure (not just plain text dump)
	if !strings.Contains(output, "\n\n") {
		t.Error("Output should have paragraph breaks for readability")
	}
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
	if output == "" && err == nil {
		t.Error("Expected output or error for multiple args")
	}
}

func TestExploreCommand_FormattingConsistency(t *testing.T) {
	buf := new(bytes.Buffer)
	exploreCmd.SetOut(buf)
	exploreCmd.SetErr(buf)

	err := runExplore(exploreCmd, []string{"versions"})
	if err != nil {
		t.Fatalf("runExplore() unexpected error: %v", err)
	}

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

	if commandLineCount == 0 {
		t.Error("Expected at least one goenv command in output")
	}
}
