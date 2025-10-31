package utils

import (
	"encoding/json"
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

// ANSI color codes
const (
	colorReset  = "\x1b[0m"
	colorRed    = "\x1b[31m"
	colorGreen  = "\x1b[32m"
	colorYellow = "\x1b[33m"
	colorBlue   = "\x1b[34m"
	colorGray   = "\x1b[90m"
	colorCyan   = "\x1b[36m"

	colorBoldRed    = "\x1b[1;31m"
	colorBoldGreen  = "\x1b[1;32m"
	colorBoldYellow = "\x1b[1;33m"
	colorBoldBlue   = "\x1b[1;34m"
)

// Color wraps the given string in ANSI color codes
func Color(text, code string) string {
	if !ShouldUseColor() {
		return text
	}
	return code + text + colorReset
}

// Red returns text in red
func Red(text string) string {
	return Color(text, colorRed)
}

// Green returns text in green
func Green(text string) string {
	return Color(text, colorGreen)
}

// Yellow returns text in yellow
func Yellow(text string) string {
	return Color(text, colorYellow)
}

// Blue returns text in blue
func Blue(text string) string {
	return Color(text, colorBlue)
}

// Gray returns text in gray
func Gray(text string) string {
	return Color(text, colorGray)
}

// Cyan returns text in cyan
func Cyan(text string) string {
	return Color(text, colorCyan)
}

// BoldRed returns text in bold red
func BoldRed(text string) string {
	return Color(text, colorBoldRed)
}

// BoldGreen returns text in bold green
func BoldGreen(text string) string {
	return Color(text, colorBoldGreen)
}

// BoldYellow returns text in bold yellow
func BoldYellow(text string) string {
	return Color(text, colorBoldYellow)
}

// BoldBlue returns text in bold blue
func BoldBlue(text string) string {
	return Color(text, colorBoldBlue)
}

// BoldCyan returns text in bold cyan
func BoldCyan(text string) string {
	return Color(text, "\x1b[1;36m")
}

// BoldWhite returns text in bold white
func BoldWhite(text string) string {
	return Color(text, "\x1b[1;37m")
}

// PrintJSON encodes and prints JSON with proper indentation
func PrintJSON(w interface{ Write([]byte) (int, error) }, v interface{}) error {
	encoder := json.NewEncoder(w)
	encoder.SetIndent("", "  ")
	return encoder.Encode(v)
}
