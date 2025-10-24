package utils

import (
	"os"

	"golang.org/x/term"
)

// OutputOptions contains global output formatting preferences
type OutputOptions struct {
	NoColor bool
	Plain   bool
}

// Global output options (set by root command flags)
var globalOptions = OutputOptions{}

// SetOutputOptions sets the global output options from command line flags
func SetOutputOptions(noColor, plain bool) {
	globalOptions.NoColor = noColor
	globalOptions.Plain = plain
}

// ShouldUseColor returns true if colored output should be used
func ShouldUseColor() bool {
	// Check --plain flag (disables both color and emojis)
	if globalOptions.Plain {
		return false
	}

	// Check --no-color flag
	if globalOptions.NoColor {
		return false
	}

	// Check NO_COLOR environment variable (https://no-color.org/)
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a terminal
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}

	return true
}

// ShouldUseEmojis returns true if emojis should be used in output
func ShouldUseEmojis() bool {
	// Check --plain flag (disables both color and emojis)
	if globalOptions.Plain {
		return false
	}

	// Check NO_COLOR environment variable (https://no-color.org/)
	// Many parsers that break on emojis also set NO_COLOR
	if os.Getenv("NO_COLOR") != "" {
		return false
	}

	// Check if stdout is a terminal
	// When piped or redirected, don't use emojis
	if !term.IsTerminal(int(os.Stdout.Fd())) {
		return false
	}

	return true
}

// Emoji returns the emoji string if emojis should be used, otherwise empty string
// Usage: fmt.Printf("%sChecking...\n", utils.Emoji("🔍 "))
func Emoji(emoji string) string {
	if ShouldUseEmojis() {
		return emoji
	}
	return ""
}

// EmojiOr returns the emoji if emojis should be used, otherwise returns the fallback string
// Usage: fmt.Printf("%sError occurred\n", utils.EmojiOr("❌ ", "[ERROR] "))
func EmojiOr(emoji, fallback string) string {
	if ShouldUseEmojis() {
		return emoji
	}
	return fallback
}
