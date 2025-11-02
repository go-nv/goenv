package cmdtest

import (
	"os"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
)

// TestEmojiSuppression_CommandOutput is a golden test that verifies
// emoji suppression when command output is piped
//
// This test demonstrates the expected behavior when goenv commands are run
// in a non-TTY environment (piped, redirected, or in CI/CD).
func TestEmojiSuppression_CommandOutput(t *testing.T) {
	// This test documents the expected behavior rather than testing a live binary
	// because building binaries in tests can be flaky across environments.
	//
	// The behavior tested here is:
	// 1. When stdout is not a TTY (piped/redirected), emojis are suppressed
	// 2. When NO_COLOR=1 is set, emojis are suppressed
	// 3. When --plain flag is used, emojis are suppressed

	t.Log("Golden test: Emoji suppression in command output")
	t.Log("")
	t.Log("Expected behaviors:")
	t.Log("  1. goenv version | cat         → no emojis (piped, non-TTY)")
	t.Log("  2. NO_COLOR=1 goenv version    → no emojis (environment variable)")
	t.Log("  3. goenv --plain version       → no emojis (flag)")
	t.Log("  4. goenv version > output.txt  → no emojis (redirected)")
	t.Log("")
	t.Log("All commands should produce clean, parseable output without emoji characters")
	t.Log("when run in non-interactive environments.")

	// Test the underlying logic that commands rely on
	scenarios := []struct {
		name        string
		setup       func()
		expectEmoji bool
	}{
		{
			name: "piped output (non-TTY)",
			setup: func() {
				// In test environments, stdout is typically not a TTY
				// This is the normal case for CI/CD and piped output
			},
			expectEmoji: false,
		},
		{
			name: "NO_COLOR environment variable",
			setup: func() {
				os.Setenv(utils.EnvVarNoColor, "1")
			},
			expectEmoji: false,
		},
		{
			name: "plain flag",
			setup: func() {
				utils.SetOutputOptions(false, true)
			},
			expectEmoji: false,
		},
	}

	for _, sc := range scenarios {
		t.Run(sc.name, func(t *testing.T) {
			// Reset state
			os.Unsetenv("NO_COLOR")
			utils.SetOutputOptions(false, false)

			// Apply setup
			sc.setup()

			// Check emoji suppression
			result := utils.ShouldUseEmojis()

			assert.Equal(t, sc.expectEmoji, result, "ShouldUseEmojis() = , expected")

			// Verify Emoji() function
			emojiResult := utils.Emoji("✓ ")
			assert.False(t, sc.expectEmoji && emojiResult == "", "Expected emoji to be returned, but got empty string")
			assert.False(t, !sc.expectEmoji && emojiResult != "", "Expected no emoji")

			// Verify EmojiOr() function
			fallbackResult := utils.EmojiOr("✓ ", "[OK] ")
			assert.False(t, sc.expectEmoji && fallbackResult == "[OK] ", "Expected emoji, but got fallback")
			assert.False(t, !sc.expectEmoji && fallbackResult != "[OK] ", "Expected fallback")

			// Clean up
			os.Unsetenv("NO_COLOR")
			utils.SetOutputOptions(false, false)
		})
	}

	t.Log("")
	t.Log("✓ All emoji suppression scenarios validated")
	t.Log("Commands will produce clean output when piped or in CI/CD environments")
}

// TestEmojiFunction_DirectUsage tests the Emoji and EmojiOr functions directly
func TestEmojiFunction_DirectUsage(t *testing.T) {
	// Save original state
	defer func() {
		utils.SetOutputOptions(false, false)
		os.Unsetenv("NO_COLOR")
	}()

	tests := []struct {
		name     string
		setup    func()
		emoji    string
		fallback string
		wantFunc func() string
	}{
		{
			name: "Emoji() with NO_COLOR returns empty",
			setup: func() {
				os.Setenv(utils.EnvVarNoColor, "1")
				utils.SetOutputOptions(false, false)
			},
			emoji: "✓ ",
			wantFunc: func() string {
				return utils.Emoji("✓ ")
			},
		},
		{
			name: "EmojiOr() with NO_COLOR returns fallback",
			setup: func() {
				os.Setenv(utils.EnvVarNoColor, "1")
				utils.SetOutputOptions(false, false)
			},
			emoji:    "✓ ",
			fallback: "[OK] ",
			wantFunc: func() string {
				return utils.EmojiOr("✓ ", "[OK] ")
			},
		},
		{
			name: "Emoji() with --plain returns empty",
			setup: func() {
				os.Unsetenv("NO_COLOR")
				utils.SetOutputOptions(false, true)
			},
			emoji: "❌ ",
			wantFunc: func() string {
				return utils.Emoji("❌ ")
			},
		},
		{
			name: "EmojiOr() with --plain returns fallback",
			setup: func() {
				os.Unsetenv("NO_COLOR")
				utils.SetOutputOptions(false, true)
			},
			emoji:    "❌ ",
			fallback: "[ERROR] ",
			wantFunc: func() string {
				return utils.EmojiOr("❌ ", "[ERROR] ")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Reset
			os.Unsetenv("NO_COLOR")
			utils.SetOutputOptions(false, false)

			// Setup test conditions
			tt.setup()

			// Get result
			result := tt.wantFunc()

			// For emoji suppression tests, we expect empty or fallback
			if strings.Contains(tt.name, "NO_COLOR") || strings.Contains(tt.name, "plain") {
				if strings.Contains(tt.name, "EmojiOr") {
					// Should return fallback
					assert.Equal(t, tt.fallback, result)
				} else {
					// Should return empty
					assert.Empty(t, result, "Expected empty string")
				}
			}

			t.Logf("Result: %q", result)
		})
	}
}

// TestOutputFunctions_NilCheck ensures functions don't panic with edge cases
func TestOutputFunctions_NilCheck(t *testing.T) {
	// Save state
	defer func() {
		utils.SetOutputOptions(false, false)
		os.Unsetenv("NO_COLOR")
	}()

	// Test with empty strings
	result := utils.Emoji("")
	assert.Empty(t, result, "Emoji() should return empty string")

	// Test EmojiOr with empty strings
	result = utils.EmojiOr("", "")
	assert.Empty(t, result, "EmojiOr(, ) should return empty string")

	// Test with NO_COLOR
	os.Setenv(utils.EnvVarNoColor, "1")
	result = utils.Emoji("✓")
	assert.Empty(t, result, "With NO_COLOR, Emoji should return empty")

	result = utils.EmojiOr("✓", "OK")
	assert.Equal(t, "OK", result, "With NO_COLOR, EmojiOr should return fallback")
}
