package utils

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPromptYesNo_Yes(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
	}{
		{"explicit yes", "yes\n", true, true},
		{"explicit y", "y\n", true, true},
		{"explicit YES uppercase", "YES\n", true, true},
		{"explicit Y uppercase", "Y\n", true, true},
		{"explicit no", "no\n", true, false},
		{"explicit n", "n\n", true, false},
		{"empty with default yes", "\n", true, true},
		{"empty with default no", "\n", false, false},
		{"whitespace with default yes", "  \n", true, true},
		{"invalid input", "maybe\n", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			config := PromptConfig{
				Question:   "Test question?",
				DefaultYes: tt.defaultYes,
				Reader:     reader,
				Writer:     writer,
				ErrWriter:  errWriter,
			}

			got := PromptYesNo(config)
			assert.Equal(t, tt.want, got, "PromptYesNo() =")

			// Verify prompt was displayed
			output := writer.String()
			assert.Contains(t, output, "Test question?", "Prompt not displayed %v", output)

			// Verify correct default indicator
			assert.False(t, tt.defaultYes && !strings.Contains(output, "[Y/n]"), "Expected [Y/n] indicator")
			assert.True(t, tt.defaultYes || strings.Contains(output, "[y/N]"), "Expected [y/N] indicator")
		})
	}
}

func TestPromptYesNo_NonInteractive(t *testing.T) {
	// This test simulates non-interactive mode by using a non-stdin reader
	// When Reader is not os.Stdin, the TTY check is skipped
	reader := strings.NewReader("yes\n")
	writer := &bytes.Buffer{}
	errWriter := &bytes.Buffer{}

	config := PromptConfig{
		Question:            "Install?",
		DefaultYes:          true,
		Reader:              reader,
		Writer:              writer,
		ErrWriter:           errWriter,
		NonInteractiveError: "Error: Cannot proceed",
		NonInteractiveHelp:  []string{"Try: goenv install --yes"},
	}

	// Since we're using a non-os.Stdin reader, it should proceed normally
	got := PromptYesNo(config)
	assert.True(t, got, "Expected true for 'yes' input, got false")

	// Should not show non-interactive error since reader is not os.Stdin
	assert.NotContains(t, errWriter.String(), "non-interactive", "Should not show non-interactive error for custom reader")
}

func TestPromptYesNo_ReadError(t *testing.T) {
	// Create a reader that returns EOF immediately
	reader := strings.NewReader("")
	writer := &bytes.Buffer{}
	errWriter := &bytes.Buffer{}

	config := PromptConfig{
		Question:   "Continue?",
		DefaultYes: true,
		Reader:     reader,
		Writer:     writer,
		ErrWriter:  errWriter,
	}

	got := PromptYesNo(config)
	assert.False(t, got, "Expected false on read error, got true")
}

func TestPromptYesNoSimple(t *testing.T) {
	// TestPromptYesNoSimple verifies that the convenience function:
	// 1. Uses DefaultYes: true
	// 2. Works with standard yes/no responses
	// 3. Handles empty input as "yes" (due to DefaultYes)
	//
	// Note: We can't directly test the function since it uses os.Stdin,
	// but we verify the wrapper by calling the underlying function with
	// the same config that PromptYesNoSimple would use.

	tests := []struct {
		name  string
		input string
		want  bool
	}{
		{"yes response", "yes\n", true},
		{"y response", "y\n", true},
		{"no response", "no\n", false},
		{"n response", "n\n", false},
		{"empty defaults to yes", "\n", true},
		{"whitespace defaults to yes", "  \n", true},
		{"uppercase YES", "YES\n", true},
		{"mixed case Yes", "Yes\n", true},
		{"invalid input", "maybe\n", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}

			// Simulate what PromptYesNoSimple does
			config := PromptConfig{
				Question:   "Test?",
				DefaultYes: true, // This is what PromptYesNoSimple sets
				Reader:     reader,
				Writer:     writer,
			}

			got := PromptYesNo(config)
			assert.Equal(t, tt.want, got, "PromptYesNoSimple behavior for %v", tt.input)
		})
	}
}

func TestPromptYesNo_DefaultYesVariations(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"Y\n", true},
		{"yes\n", true},
		{"Yes\n", true},
		{"YES\n", true},
		{"n\n", false},
		{"N\n", false},
		{"no\n", false},
		{"No\n", false},
		{"NO\n", false},
		{"\n", true},      // empty defaults to yes
		{"   \n", true},   // whitespace defaults to yes
		{"yep\n", false},  // invalid input = false
		{"nope\n", false}, // invalid input = false
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}

			config := PromptConfig{
				Question:   "Test?",
				DefaultYes: true,
				Reader:     reader,
				Writer:     writer,
			}

			got := PromptYesNo(config)
			assert.Equal(t, tt.want, got, "Input %v", tt.input)
		})
	}
}

func TestPromptYesNo_DefaultNoVariations(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"y\n", true},
		{"yes\n", true},
		{"n\n", false},
		{"no\n", false},
		{"\n", false},      // empty defaults to no
		{"   \n", false},   // whitespace defaults to no
		{"maybe\n", false}, // invalid input = false
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}

			config := PromptConfig{
				Question:   "Test?",
				DefaultYes: false,
				Reader:     reader,
				Writer:     writer,
			}

			got := PromptYesNo(config)
			assert.Equal(t, tt.want, got, "Input %v", tt.input)
		})
	}
}

func TestPromptYesNo_EdgeCases(t *testing.T) {
	tests := []struct {
		name       string
		input      string
		defaultYes bool
		want       bool
	}{
		// Mixed case variations
		{"mixed case yEs", "yEs\n", true, true},
		{"mixed case YeS", "YeS\n", false, true},
		{"mixed case nO", "nO\n", true, false},
		{"mixed case No", "No\n", false, false},

		// Extra whitespace
		{"leading whitespace yes", "  yes\n", true, true},
		{"trailing whitespace yes", "yes  \n", false, true},
		{"surrounding whitespace yes", "  yes  \n", true, true},
		{"leading whitespace no", "  no\n", false, false},
		{"tabs and spaces", "\t  yes  \t\n", true, true},

		// Multiple whitespace in empty input
		{"multiple spaces", "     \n", true, true},
		{"multiple spaces default no", "     \n", false, false},
		{"tabs only", "\t\t\n", true, true},
		{"mixed whitespace", " \t \t \n", false, false},

		// Unusual but valid inputs
		{"y with whitespace", " y \n", true, true},
		{"n with whitespace", " n \n", false, false},
		{"YES with spaces", "  YES  \n", true, true},
		{"NO with spaces", "  NO  \n", false, false},

		// Invalid inputs (should always return false)
		{"yeah", "yeah\n", true, false},
		{"yup", "yup\n", false, false},
		{"nope", "nope\n", true, false},
		{"nah", "nah\n", false, false},
		{"true", "true\n", true, false},
		{"false", "false\n", false, false},
		{"1", "1\n", true, false},
		{"0", "0\n", false, false},
		{"random text", "random\n", true, false},
		{"partial yes", "ye\n", false, false},
		{"partial no", "n o\n", true, false},

		// Special characters (all invalid)
		{"exclamation", "yes!\n", true, false},
		{"question mark", "yes?\n", false, false},
		{"period", "yes.\n", true, false},
		{"comma", "yes,\n", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader(tt.input)
			writer := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			config := PromptConfig{
				Question:   "Test?",
				DefaultYes: tt.defaultYes,
				Reader:     reader,
				Writer:     writer,
				ErrWriter:  errWriter,
			}

			got := PromptYesNo(config)
			assert.Equal(t, tt.want, got, "Input (defaultYes=) %v %v", tt.input, tt.defaultYes)
		})
	}
}

func TestPromptYesNo_QuestionFormatting(t *testing.T) {
	tests := []struct {
		name       string
		question   string
		defaultYes bool
		wantPrompt string
	}{
		{"simple question with default yes", "Install?", true, "Install? [Y/n]:"},
		{"simple question with default no", "Continue?", false, "Continue? [y/N]:"},
		{"long question", "Do you want to proceed with the installation?", true, "Do you want to proceed with the installation? [Y/n]:"},
		{"question with emoji", "🚀 Deploy now?", false, "🚀 Deploy now? [y/N]:"},
		{"empty question with default yes", "", true, " [Y/n]:"},
		{"empty question with default no", "", false, " [y/N]:"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader("yes\n")
			writer := &bytes.Buffer{}

			config := PromptConfig{
				Question:   tt.question,
				DefaultYes: tt.defaultYes,
				Reader:     reader,
				Writer:     writer,
			}

			_ = PromptYesNo(config)

			output := writer.String()
			assert.Equal(t, tt.wantPrompt+" ", output, "Expected prompt %v", strings.TrimSpace(output))
		})
	}
}

func TestPromptYesNo_NonInteractiveHelp(t *testing.T) {
	reader := strings.NewReader("yes\n")
	writer := &bytes.Buffer{}
	errWriter := &bytes.Buffer{}

	config := PromptConfig{
		Question:            "Install?",
		DefaultYes:          true,
		Reader:              reader,
		Writer:              writer,
		ErrWriter:           errWriter,
		NonInteractiveError: "ERROR: Cannot proceed in non-interactive mode",
		NonInteractiveHelp: []string{
			"Use: goenv install --yes",
			"Or set: GOENV_ASSUME_YES=1",
		},
	}

	// This test verifies that the help text fields are accepted
	// and don't cause panics (actual non-interactive detection requires os.Stdin)
	got := PromptYesNo(config)
	assert.True(t, got, "Expected true for 'yes' input, got false")
}
