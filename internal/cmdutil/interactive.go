package cmdutil

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

// InteractionLevel controls how interactive a command should be
type InteractionLevel int

const (
	// InteractionNone - no prompts, fail fast (--yes, --force, or CI mode)
	InteractionNone InteractionLevel = iota

	// InteractionMinimal - only critical confirmations (default)
	InteractionMinimal

	// InteractionGuided - helpful prompts and suggestions (--interactive)
	InteractionGuided
)

// InteractiveContext holds the interaction state for a command execution
type InteractiveContext struct {
	Level     InteractionLevel
	AssumeYes bool
	Quiet     bool
	cmd       *cobra.Command

	// IO streams (injectable for testing)
	Reader    io.Reader
	Writer    io.Writer
	ErrWriter io.Writer
}

// NewInteractiveContext creates an interactive context from command flags
func NewInteractiveContext(cmd *cobra.Command) *InteractiveContext {
	// Detect from flags and environment
	assumeYes := utils.GoenvEnvVarAssumeYes.IsTrue()
	quiet := false

	// Check for --quiet flag
	if cmd.Flags().Changed("quiet") {
		quietFlag, _ := cmd.Flags().GetBool("quiet")
		quiet = quietFlag
	}

	// Check for --yes flag
	if cmd.Flags().Changed("yes") {
		yesFlag, _ := cmd.Flags().GetBool("yes")
		if yesFlag {
			assumeYes = true
		}
	}

	// Check for --force flag
	if cmd.Flags().Changed("force") {
		forceFlag, _ := cmd.Flags().GetBool("force")
		if forceFlag {
			assumeYes = true
		}
	}

	// Determine interaction level
	level := InteractionMinimal
	if assumeYes || quiet || IsCI() {
		level = InteractionNone
	}

	// Check for --interactive flag
	if cmd.Flags().Changed("interactive") {
		interactiveFlag, _ := cmd.Flags().GetBool("interactive")
		if interactiveFlag && !IsCI() {
			level = InteractionGuided
		}
	}

	return &InteractiveContext{
		Level:     level,
		AssumeYes: assumeYes,
		Quiet:     quiet,
		cmd:       cmd,
		Reader:    os.Stdin,
		Writer:    os.Stdout,
		ErrWriter: os.Stderr,
	}
}

// IsInteractive returns true if the context allows interactive prompts
func (ctx *InteractiveContext) IsInteractive() bool {
	return ctx.Level != InteractionNone
}

// IsGuided returns true if the context is in guided interaction mode
func (ctx *InteractiveContext) IsGuided() bool {
	return ctx.Level == InteractionGuided
}

// Confirm asks for yes/no confirmation based on interaction level
// Returns true if user confirms, false otherwise
// In non-interactive mode, returns defaultValue
func (ctx *InteractiveContext) Confirm(question string, defaultYes bool) bool {
	// If assume-yes is set, always confirm
	if ctx.AssumeYes {
		return true
	}

	// If non-interactive, return default
	if ctx.Level == InteractionNone {
		return defaultYes
	}

	// Use existing prompt infrastructure
	return utils.PromptYesNo(utils.PromptConfig{
		Question:   question,
		DefaultYes: defaultYes,
		Reader:     ctx.Reader,
		Writer:     ctx.Writer,
		ErrWriter:  ctx.ErrWriter,
	})
}

// OfferRepair prompts user to fix an issue and returns whether to proceed
// This is a specialized version of Confirm with repair-specific messaging
func (ctx *InteractiveContext) OfferRepair(problem string, repairDescription string) bool {
	// If assume-yes is set, always attempt repair
	if ctx.AssumeYes {
		if !ctx.Quiet {
			fmt.Fprintf(ctx.Writer, "%s Problem Detected\n", utils.Emoji("⚠️ "))
			fmt.Fprintf(ctx.Writer, "%s\n", problem)
			fmt.Fprintf(ctx.Writer, "%s Attempting repair...\n", utils.Emoji("🔧"))
		}
		return true
	}

	// If non-interactive, don't offer repair
	if ctx.Level == InteractionNone {
		if !ctx.Quiet {
			fmt.Fprintf(ctx.ErrWriter, "%s Problem Detected\n", utils.Emoji("⚠️ "))
			fmt.Fprintf(ctx.ErrWriter, "%s\n", problem)
			fmt.Fprintln(ctx.ErrWriter, "\nRunning in non-interactive mode. Cannot prompt for repair.")
			fmt.Fprintf(ctx.ErrWriter, "  Repair: %s\n", repairDescription)
			fmt.Fprintln(ctx.ErrWriter, "  Try: Use --yes to auto-confirm repairs")
		}
		return false
	}

	// Interactive mode - show problem and offer repair
	fmt.Fprintf(ctx.Writer, "%s Problem Detected\n", utils.Emoji("⚠️ "))
	fmt.Fprintf(ctx.Writer, "%s\n", problem)

	question := "Would you like goenv to attempt to fix this?"
	if ctx.Confirm(question, true) {
		if !ctx.Quiet {
			fmt.Fprintf(ctx.Writer, "%s Attempting repair...\n", utils.Emoji("🔧"))
		}
		return true
	}

	return false
}

// Select prompts user to select from multiple options
// Returns the selected option (1-indexed) or 0 if cancelled/error
func (ctx *InteractiveContext) Select(question string, options []string) int {
	// If assume-yes is set, select first option
	if ctx.AssumeYes {
		return 1
	}

	// If non-interactive, cannot select
	if ctx.Level == InteractionNone {
		if !ctx.Quiet {
			fmt.Fprintf(ctx.ErrWriter, "\n%s\n", question)
			fmt.Fprintln(ctx.ErrWriter, "Running in non-interactive mode. Cannot prompt for selection.")
			fmt.Fprintln(ctx.ErrWriter, "Available options:")
			for i, opt := range options {
				fmt.Fprintf(ctx.ErrWriter, "  %d) %s\n", i+1, opt)
			}
			fmt.Fprintln(ctx.ErrWriter)
		}
		return 0
	}

	// Show question and options
	fmt.Fprintf(ctx.Writer, "\n%s\n", question)
	for i, opt := range options {
		fmt.Fprintf(ctx.Writer, "  %d) %s\n", i+1, opt)
	}
	fmt.Fprint(ctx.Writer, "\nEnter selection: ")

	// Read response
	reader := bufio.NewReader(ctx.Reader)
	response, err := reader.ReadString('\n')
	if err != nil {
		return 0
	}

	// Parse response
	response = strings.TrimSpace(response)
	if response == "" {
		return 0
	}

	// Convert to integer
	selection, err := strconv.Atoi(response)
	if err != nil || selection < 1 || selection > len(options) {
		fmt.Fprintf(ctx.ErrWriter, "Invalid selection: %s\n", response)
		return 0
	}

	return selection
}

// WaitForUser pauses execution until user presses Enter
// Only waits in interactive mode; in non-interactive mode (CI/tests), returns immediately
// Use this for "press Enter to continue" patterns where user needs time to read/act on information
func (ctx *InteractiveContext) WaitForUser(message string) {
	// Skip in non-interactive mode (CI, tests, --yes, --quiet)
	if ctx.Level == InteractionNone {
		return
	}

	// Show message if provided
	if message != "" && !ctx.Quiet {
		fmt.Fprintln(ctx.Writer, message)
	}

	// Wait for Enter key
	reader := bufio.NewReader(ctx.Reader)
	_, _ = reader.ReadString('\n')
}

// Printf writes formatted output (respects quiet mode)
func (ctx *InteractiveContext) Printf(format string, a ...interface{}) {
	if !ctx.Quiet {
		fmt.Fprintf(ctx.Writer, format, a...)
	}
}

// Println writes a line of output (respects quiet mode)
func (ctx *InteractiveContext) Println(a ...interface{}) {
	if !ctx.Quiet {
		fmt.Fprintln(ctx.Writer, a...)
	}
}

// ErrorPrintf writes formatted error output (ignores quiet mode)
func (ctx *InteractiveContext) ErrorPrintf(format string, a ...interface{}) {
	fmt.Fprintf(ctx.ErrWriter, format, a...)
}

// IsCI detects if running in a CI/CD environment
// This is used to automatically disable interactive features
func IsCI() bool {
	// Check common CI environment variables
	ciEnvVars := []string{
		utils.EnvVarGitHubActions,
		utils.EnvVarGitLabCI,
		utils.EnvVarCircleCI,
		utils.EnvVarTravisCI,
		"CI",               // Generic CI variable
		"BUILD_ID",         // Jenkins
		"BUILDKITE",        // Buildkite
		"DRONE",            // Drone CI
		"TEAMCITY_VERSION", // TeamCity
	}

	for _, envVar := range ciEnvVars {
		if os.Getenv(envVar) != "" {
			return true
		}
	}

	return false
}

// IsTTY checks if the given file is a terminal
func IsTTY(file *os.File) bool {
	fileInfo, err := file.Stat()
	if err != nil {
		return false
	}
	return (fileInfo.Mode() & os.ModeCharDevice) != 0
}
