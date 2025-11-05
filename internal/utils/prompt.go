package utils

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strings"

	"golang.org/x/term"
)

// PromptConfig configures behavior of a yes/no prompt
type PromptConfig struct {
	// Question is the prompt text shown to the user (e.g., "Install now?")
	Question string

	// DefaultYes if true, treats empty input as "yes", otherwise as "no"
	// Only applies when actually showing a prompt (i.e., when AutoConfirm is false)
	DefaultYes bool

	// AutoConfirm if true, automatically confirms without prompting
	// Can also be controlled via GOENV_ASSUME_YES environment variable
	// Takes precedence over DefaultYes (prompt is never shown)
	AutoConfirm bool

	// NonInteractiveError is shown when running in non-interactive mode
	NonInteractiveError string

	// NonInteractiveHelp provides suggestions for non-interactive usage
	NonInteractiveHelp []string

	// Reader is the input source (defaults to os.Stdin)
	Reader io.Reader

	// Writer is the output destination (defaults to os.Stdout)
	Writer io.Writer

	// ErrWriter is the error output destination (defaults to os.Stderr)
	ErrWriter io.Writer
}

// PromptYesNo displays a yes/no prompt and returns the user's response
// Returns true if user responds yes, false otherwise
// In non-interactive mode, returns false and displays error message
// Respects GOENV_ASSUME_YES environment variable for global non-interactive mode
func PromptYesNo(config PromptConfig) bool {
	// Set defaults
	if config.Reader == nil {
		config.Reader = os.Stdin
	}
	if config.Writer == nil {
		config.Writer = os.Stdout
	}
	if config.ErrWriter == nil {
		config.ErrWriter = os.Stderr
	}

	// Check for GOENV_ASSUME_YES environment variable
	// This provides a global way to skip all prompts (similar to DEBIAN_FRONTEND=noninteractive)
	if !config.AutoConfirm {
		if GoenvEnvVarAssumeYes.IsTrue() {
			config.AutoConfirm = true
		}
	}

	// If AutoConfirm is set (via field or env var), return true immediately without prompting
	if config.AutoConfirm {
		return true
	}

	// Check if stdin is a TTY (interactive terminal)
	// Only check if Reader is os.Stdin
	if file, ok := config.Reader.(*os.File); ok && file == os.Stdin {
		if !term.IsTerminal(int(file.Fd())) {
			// Non-interactive environment (CI/CD, piped input, etc.)
			if config.NonInteractiveError != "" {
				fmt.Fprintf(config.ErrWriter, "\n%s\n", config.NonInteractiveError)
			}

			if len(config.NonInteractiveHelp) > 0 {
				fmt.Fprintf(config.ErrWriter, "\nRunning in non-interactive mode. Cannot prompt for input.\n")
				for _, help := range config.NonInteractiveHelp {
					fmt.Fprintf(config.ErrWriter, "  %s\n", help)
				}
				fmt.Fprintln(config.ErrWriter)
			}
			return false
		}
	}

	// Show prompt with appropriate default indicator
	promptSuffix := "[y/N]"
	if config.DefaultYes {
		promptSuffix = "[Y/n]"
	}
	fmt.Fprintf(config.Writer, "%s %s: ", config.Question, promptSuffix)

	// Read response
	reader := bufio.NewReader(config.Reader)
	response, err := reader.ReadString('\n')
	if err != nil {
		return false
	}

	// Parse response
	response = strings.TrimSpace(strings.ToLower(response))

	// Handle default case (empty input)
	if response == "" {
		return config.DefaultYes
	}

	// Explicit yes/no
	return response == "y" || response == "yes"
}

// PromptYesNoSimple is a convenience wrapper for simple yes/no prompts with default yes
func PromptYesNoSimple(question string) bool {
	return PromptYesNo(PromptConfig{
		Question:   question,
		DefaultYes: true,
	})
}

// PauseForUser displays a "Press Enter to continue" prompt and waits for user input.
// This prevents important information from being immediately buried by subsequent output.
// Returns immediately if the reader encounters an error (useful for testing and non-interactive scenarios).
func PauseForUser(out io.Writer, reader *bufio.Reader) {
	fmt.Fprintln(out)
	fmt.Fprintf(out, "%sPress Enter to continue...", Emoji("⏸️  "))
	_, _ = reader.ReadString('\n')
}
