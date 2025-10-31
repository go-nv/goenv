package utils

import (
	"bytes"
	"strings"
	"testing"
)

func TestPromptYesNo_AutoConfirmField(t *testing.T) {
	tests := []struct {
		name        string
		autoConfirm bool
		want        bool
	}{
		{"AutoConfirm true", true, true},
		{"AutoConfirm false", false, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reader := strings.NewReader("") // No input should be read
			writer := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			config := PromptConfig{
				Question:    "Test question?",
				AutoConfirm: tt.autoConfirm,
				DefaultYes:  false, // Should be ignored when AutoConfirm is true
				Reader:      reader,
				Writer:      writer,
				ErrWriter:   errWriter,
			}

			got := PromptYesNo(config)
			if got != tt.want {
				t.Errorf("PromptYesNo() = %v, want %v", got, tt.want)
			}

			// When AutoConfirm is true, prompt should NOT be displayed
			if tt.autoConfirm {
				output := writer.String()
				if output != "" {
					t.Errorf("Prompt should not be displayed when AutoConfirm=true, got: %s", output)
				}
			}
		})
	}
}

func TestPromptYesNo_EnvironmentVariable(t *testing.T) {
	tests := []struct {
		name   string
		envVal string
		want   bool
	}{
		{"GOENV_ASSUME_YES=1", "1", true},
		{"GOENV_ASSUME_YES=true", "true", true},
		{"GOENV_ASSUME_YES=yes", "yes", true},
		{"GOENV_ASSUME_YES=0", "0", false},
		{"GOENV_ASSUME_YES=false", "false", false},
		{"GOENV_ASSUME_YES=no", "no", false},
		{"GOENV_ASSUME_YES empty", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			if tt.envVal != "" {
				t.Setenv("GOENV_ASSUME_YES", tt.envVal)
			}

			reader := strings.NewReader("n\n") // Input "no", but env var should take precedence
			writer := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			config := PromptConfig{
				Question:   "Test question?",
				DefaultYes: false,
				Reader:     reader,
				Writer:     writer,
				ErrWriter:  errWriter,
			}

			got := PromptYesNo(config)
			if got != tt.want {
				t.Errorf("PromptYesNo() with GOENV_ASSUME_YES=%s = %v, want %v", tt.envVal, got, tt.want)
			}

			// When GOENV_ASSUME_YES=1/true/yes, prompt should NOT be displayed
			output := writer.String()
			if tt.want && output != "" {
				t.Errorf("Prompt should not be displayed when GOENV_ASSUME_YES=%s, got: %s", tt.envVal, output)
			}
		})
	}
}

func TestPromptYesNo_AutoConfirmFieldTakesPrecedence(t *testing.T) {
	// AutoConfirm flag should take precedence over environment variable
	t.Setenv("GOENV_ASSUME_YES", "0") // Try to disable via env var

	reader := strings.NewReader("n\n")
	writer := &bytes.Buffer{}
	errWriter := &bytes.Buffer{}

	config := PromptConfig{
		Question:    "Test question?",
		AutoConfirm: true, // Flag should win
		DefaultYes:  false,
		Reader:      reader,
		Writer:      writer,
		ErrWriter:   errWriter,
	}

	got := PromptYesNo(config)
	if !got {
		t.Errorf("PromptYesNo() with AutoConfirm=true should return true regardless of env var")
	}

	// Prompt should not be displayed
	output := writer.String()
	if output != "" {
		t.Errorf("Prompt should not be displayed when AutoConfirm=true, got: %s", output)
	}
}

func TestPromptYesNo_EnvironmentVariableInvalidValues(t *testing.T) {
	tests := []struct {
		name         string
		envVal       string
		inputAnswer  string
		expectedBool bool
		shouldPrompt bool
	}{
		// Invalid env values should be ignored, falling back to normal prompting
		{"GOENV_ASSUME_YES=Yes (capitalized)", "Yes", "yes\n", true, true},
		{"GOENV_ASSUME_YES=TRUE (uppercase)", "TRUE", "no\n", false, true},
		{"GOENV_ASSUME_YES=2", "2", "yes\n", true, true},
		{"GOENV_ASSUME_YES=random", "random", "yes\n", true, true},
		{"GOENV_ASSUME_YES=on", "on", "no\n", false, true},
		{"GOENV_ASSUME_YES=off", "off", "yes\n", true, true},
		{"GOENV_ASSUME_YES=y (single char, invalid)", "y", "yes\n", true, true},
		{"GOENV_ASSUME_YES=n (single char, invalid)", "n", "no\n", false, true},
		{"GOENV_ASSUME_YES with spaces", " true ", "yes\n", true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment variable
			t.Setenv("GOENV_ASSUME_YES", tt.envVal)

			reader := strings.NewReader(tt.inputAnswer)
			writer := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			config := PromptConfig{
				Question:   "Test question?",
				DefaultYes: false,
				Reader:     reader,
				Writer:     writer,
				ErrWriter:  errWriter,
			}

			got := PromptYesNo(config)
			if got != tt.expectedBool {
				t.Errorf("PromptYesNo() with invalid GOENV_ASSUME_YES=%q = %v, want %v (should use input)",
					tt.envVal, got, tt.expectedBool)
			}

			// Should show prompt since env var is invalid
			output := writer.String()
			if tt.shouldPrompt && output == "" {
				t.Errorf("Prompt should be displayed when GOENV_ASSUME_YES has invalid value %q", tt.envVal)
			}
		})
	}
}

func TestPromptYesNo_DefaultYesIgnoredWhenAutoConfirm(t *testing.T) {
	tests := []struct {
		name        string
		autoConfirm bool
		defaultYes  bool
		envValue    string
		want        bool
	}{
		// When AutoConfirm is true, DefaultYes should be completely ignored
		{"AutoConfirm true, DefaultYes true", true, true, "", true},
		{"AutoConfirm true, DefaultYes false", true, false, "", true},

		// When env var enables auto-confirm, DefaultYes should be ignored
		{"Env var yes, DefaultYes true", false, true, "1", true},
		{"Env var yes, DefaultYes false", false, false, "1", true},
		{"Env var yes, DefaultYes true", false, true, "true", true},
		{"Env var yes, DefaultYes false", false, false, "yes", true},

		// When AutoConfirm is false and no env var, DefaultYes matters (tested elsewhere)
		// These cases would need actual input, so we skip them here
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("GOENV_ASSUME_YES", tt.envValue)
			}

			reader := strings.NewReader("") // No input needed
			writer := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			config := PromptConfig{
				Question:    "Test question?",
				AutoConfirm: tt.autoConfirm,
				DefaultYes:  tt.defaultYes,
				Reader:      reader,
				Writer:      writer,
				ErrWriter:   errWriter,
			}

			got := PromptYesNo(config)
			if got != tt.want {
				t.Errorf("PromptYesNo(AutoConfirm=%v, DefaultYes=%v, env=%q) = %v, want %v",
					tt.autoConfirm, tt.defaultYes, tt.envValue, got, tt.want)
			}

			// Prompt should never be displayed when auto-confirming
			output := writer.String()
			if output != "" {
				t.Errorf("Prompt should not be displayed when auto-confirming, got: %s", output)
			}
		})
	}
}

func TestPromptYesNo_EnvVarOnlyWhenFieldNotSet(t *testing.T) {
	// This test verifies that GOENV_ASSUME_YES is checked
	// only when AutoConfirm field is not explicitly set to true
	tests := []struct {
		name         string
		autoConfirm  bool
		envValue     string
		inputAnswer  string
		want         bool
		shouldPrompt bool
	}{
		// Field NOT set (false), env var SHOULD work
		{"Field false, env 1", false, "1", "", true, false},
		{"Field false, env true", false, "true", "", true, false},
		{"Field false, env yes", false, "yes", "", true, false},
		{"Field false, env 0", false, "0", "yes\n", true, true},
		{"Field false, env false", false, "false", "yes\n", true, true},
		{"Field false, env no", false, "no", "yes\n", true, true},
		{"Field false, env empty", false, "", "no\n", false, true},

		// Field explicitly set to true, env var should be ignored
		{"Field true, env 0 (field wins)", true, "0", "", true, false},
		{"Field true, env false (field wins)", true, "false", "", true, false},
		{"Field true, env no (field wins)", true, "no", "", true, false},
		{"Field true, env empty (field wins)", true, "", "", true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.envValue != "" {
				t.Setenv("GOENV_ASSUME_YES", tt.envValue)
			}

			reader := strings.NewReader(tt.inputAnswer)
			writer := &bytes.Buffer{}
			errWriter := &bytes.Buffer{}

			config := PromptConfig{
				Question:    "Test question?",
				AutoConfirm: tt.autoConfirm,
				DefaultYes:  false,
				Reader:      reader,
				Writer:      writer,
				ErrWriter:   errWriter,
			}

			got := PromptYesNo(config)
			if got != tt.want {
				t.Errorf("PromptYesNo(AutoConfirm=%v, env=%q) = %v, want %v",
					tt.autoConfirm, tt.envValue, got, tt.want)
			}

			// Check if prompt was displayed as expected
			output := writer.String()
			if tt.shouldPrompt && output == "" {
				t.Errorf("Expected prompt to be displayed, but it wasn't")
			}
			if !tt.shouldPrompt && output != "" {
				t.Errorf("Expected no prompt, but got: %s", output)
			}
		})
	}
}

func TestPromptYesNo_CompleteEnvVarPrecedenceRules(t *testing.T) {
	// This test documents the complete precedence rules:
	// 1. AutoConfirm field (if explicitly true)
	// 2. GOENV_ASSUME_YES environment variable (if valid: 1/true/yes)
	// 3. Normal prompting with DefaultYes behavior

	t.Run("Precedence level 1: AutoConfirm field", func(t *testing.T) {
		t.Setenv("GOENV_ASSUME_YES", "0") // Try to disable

		config := PromptConfig{
			Question:    "Test?",
			AutoConfirm: true, // This should win
			DefaultYes:  false,
			Reader:      strings.NewReader("no\n"),
			Writer:      &bytes.Buffer{},
			ErrWriter:   &bytes.Buffer{},
		}

		if !PromptYesNo(config) {
			t.Error("AutoConfirm field should take precedence over everything")
		}
	})

	t.Run("Precedence level 2: GOENV_ASSUME_YES env var", func(t *testing.T) {
		t.Setenv("GOENV_ASSUME_YES", "1")

		config := PromptConfig{
			Question:    "Test?",
			AutoConfirm: false, // Not explicitly set
			DefaultYes:  false,
			Reader:      strings.NewReader("no\n"), // User would say no, but env var wins
			Writer:      &bytes.Buffer{},
			ErrWriter:   &bytes.Buffer{},
		}

		if !PromptYesNo(config) {
			t.Error("GOENV_ASSUME_YES should take precedence over user input")
		}
	})

	t.Run("Precedence level 3: Normal prompting", func(t *testing.T) {
		// No env var set, AutoConfirm false

		writer := &bytes.Buffer{}
		config := PromptConfig{
			Question:    "Test?",
			AutoConfirm: false,
			DefaultYes:  true,
			Reader:      strings.NewReader("yes\n"),
			Writer:      writer,
			ErrWriter:   &bytes.Buffer{},
		}

		if !PromptYesNo(config) {
			t.Error("Should respect user input when no auto-confirm is active")
		}

		// Verify prompt was actually shown
		if !strings.Contains(writer.String(), "Test?") {
			t.Error("Prompt should be displayed in normal mode")
		}
	})
}
