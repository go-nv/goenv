package hooks

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// LogToFileAction writes formatted log messages to a file
type LogToFileAction struct{}

func (a *LogToFileAction) Name() ActionName {
	return ActionLogToFile
}

func (a *LogToFileAction) Description() string {
	return "Write log entry to file"
}

func (a *LogToFileAction) Validate(params map[string]interface{}) error {
	// Required: file
	file, ok := params["file"].(string)
	if !ok || file == "" {
		return fmt.Errorf("'file' parameter is required and must be a string")
	}

	// Validate path
	if err := ValidatePath(file); err != nil {
		return err
	}

	// Optional: format
	if format, ok := params["format"].(string); ok {
		if err := ValidateString(format); err != nil {
			return fmt.Errorf("invalid format string: %w", err)
		}
	}

	return nil
}

func (a *LogToFileAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	// Get parameters
	file := params["file"].(string)
	format, _ := params["format"].(string)
	if format == "" {
		format = "{timestamp} | {message}"
	}
	append := true
	if appendParam, ok := params["append"].(bool); ok {
		append = appendParam
	}

	// Expand path
	file = os.ExpandEnv(file)
	file = expandTilde(file)

	// Interpolate format string
	message := interpolateString(format, ctx.Variables)

	// Add timestamp if not present
	if !strings.Contains(message, "{timestamp}") {
		timestamp := time.Now().Format("2006-01-02 15:04:05")
		ctx.Variables["timestamp"] = timestamp
		message = interpolateString(format, ctx.Variables)
	}

	// Ensure message ends with newline
	if !strings.HasSuffix(message, "\n") {
		message += "\n"
	}

	// Create directory if needed
	dir := filepath.Dir(file)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Open file
	flags := os.O_CREATE | os.O_WRONLY
	if append {
		flags |= os.O_APPEND
	} else {
		flags |= os.O_TRUNC
	}

	f, err := os.OpenFile(file, flags, 0644)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}
	defer f.Close()

	// Check file size (limit to 100MB)
	info, err := f.Stat()
	if err == nil && info.Size() > 100*1024*1024 {
		return fmt.Errorf("log file exceeds maximum size (100MB)")
	}

	// Write message
	if _, err := f.WriteString(message); err != nil {
		return fmt.Errorf("failed to write to log file: %w", err)
	}

	return nil
}

// expandTilde expands ~ to home directory
func expandTilde(path string) string {
	if strings.HasPrefix(path, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
