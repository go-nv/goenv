package hooks

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
)

// RunCommandAction executes a shell command during hook execution
type RunCommandAction struct{}

func (a *RunCommandAction) Name() ActionName {
	return ActionRunCommand
}

func (a *RunCommandAction) Description() string {
	return "Executes a shell command with optional output capture and logging"
}

func (a *RunCommandAction) Validate(params map[string]interface{}) error {
	// Validate command parameter (required)
	command, ok := params["command"].(string)
	if !ok || command == "" {
		return fmt.Errorf("command parameter is required and must be a non-empty string")
	}

	// Basic security validation - prevent control characters
	if err := ValidateString(command); err != nil {
		return fmt.Errorf("invalid command: %w", err)
	}

	// Validate args parameter (optional)
	if args, ok := params["args"]; ok {
		argsSlice, ok := args.([]interface{})
		if !ok {
			return fmt.Errorf("args parameter must be an array")
		}

		// Validate each argument
		for i, arg := range argsSlice {
			argStr, ok := arg.(string)
			if !ok {
				return fmt.Errorf("args[%d] must be a string", i)
			}
			if err := ValidateString(argStr); err != nil {
				return fmt.Errorf("invalid args[%d]: %w", i, err)
			}
		}
	}

	// Validate working_dir parameter (optional)
	if workDir, ok := params["working_dir"].(string); ok {
		if workDir != "" {
			if err := ValidatePath(workDir); err != nil {
				return fmt.Errorf("invalid working_dir: %w", err)
			}
		}
	}

	// Validate timeout parameter (optional)
	if timeoutStr, ok := params["timeout"].(string); ok {
		if timeoutStr != "" {
			if _, err := time.ParseDuration(timeoutStr); err != nil {
				return fmt.Errorf("invalid timeout format: %w (use format like '30s', '5m')", err)
			}
		}
	}

	// Validate capture_output parameter (optional)
	if capture, ok := params["capture_output"]; ok {
		if _, ok := capture.(bool); !ok {
			return fmt.Errorf("capture_output parameter must be a boolean")
		}
	}

	// Validate log_output parameter (optional)
	if logOut, ok := params["log_output"]; ok {
		if _, ok := logOut.(bool); !ok {
			return fmt.Errorf("log_output parameter must be a boolean")
		}
	}

	// Validate fail_on_error parameter (optional)
	if failOn, ok := params["fail_on_error"]; ok {
		if _, ok := failOn.(bool); !ok {
			return fmt.Errorf("fail_on_error parameter must be a boolean")
		}
	}

	// Validate shell parameter (optional)
	if shell, ok := params["shell"].(string); ok {
		shell = strings.ToLower(shell)
		// Allow "auto" and "sh" as special cases, plus all valid shell types
		if shell != "auto" && shell != "sh" && shellutil.ParseShellType(shell) == shellutil.ShellTypeUnknown {
			return fmt.Errorf("invalid shell: %s (must be auto, bash, zsh, fish, sh, cmd, powershell, or ksh)", shell)
		}
	}

	return nil
}

func (a *RunCommandAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	// Get command
	command := params["command"].(string)
	command = interpolateString(command, ctx.Variables)

	// Get args (optional)
	var args []string
	if argsParam, ok := params["args"].([]interface{}); ok {
		args = make([]string, len(argsParam))
		for i, arg := range argsParam {
			argStr := arg.(string)
			args[i] = interpolateString(argStr, ctx.Variables)
		}
	}

	// Get working directory (optional)
	workDir := ""
	if wd, ok := params["working_dir"].(string); ok {
		workDir = interpolateString(wd, ctx.Variables)
		// Expand environment variables and tilde
		workDir = os.ExpandEnv(workDir)
		workDir = expandTildeForCommand(workDir)
	}

	// Get timeout (default to 2 minutes)
	timeout := 2 * time.Minute
	if timeoutStr, ok := params["timeout"].(string); ok && timeoutStr != "" {
		if parsed, err := time.ParseDuration(timeoutStr); err == nil {
			timeout = parsed
		}
	}

	// Get flags
	captureOutput := false
	if capture, ok := params["capture_output"].(bool); ok {
		captureOutput = capture
	}

	logOutput := false
	if logOut, ok := params["log_output"].(bool); ok {
		logOutput = logOut
	}

	failOnError := true
	if failOn, ok := params["fail_on_error"].(bool); ok {
		failOnError = failOn
	}

	// Get shell type
	shellType := "auto"
	if shell, ok := params["shell"].(string); ok {
		shellType = strings.ToLower(shell)
	}

	// Execute command with timeout context
	// CommandContext will automatically kill the process when the context deadline is exceeded
	cmdCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmdToRun, cmdArgs := prepareCommand(command, args, shellType)

	execCmd := exec.CommandContext(cmdCtx, cmdToRun, cmdArgs...)
	if workDir != "" {
		execCmd.Dir = workDir
	}

	// Setup output capture
	var stdout, stderr bytes.Buffer
	if captureOutput || logOutput {
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr
	}

	// Run the command (Start + Wait)
	err := execCmd.Run()

	// Check context state first to distinguish timeout from normal command failures
	// This is important because a timed-out command may return a non-zero exit code
	if cmdCtx.Err() == context.DeadlineExceeded {
		errMsg := fmt.Sprintf("command timed out after %s", timeout)
		if logOutput {
			logError(errMsg)
		}

		if failOnError {
			return fmt.Errorf("%s", errMsg)
		}
		return nil
	}

	// Check for other errors
	if err != nil {
		errMsg := fmt.Sprintf("command failed: %v", err)
		if stderr.Len() > 0 {
			errMsg += fmt.Sprintf("\nStderr: %s", stderr.String())
		}

		if logOutput {
			logError(errMsg)
		}

		if failOnError {
			return fmt.Errorf("%s", errMsg)
		}
		// If not failing on error, just log and continue
		return nil
	}

	// Success - log output if requested
	if logOutput && stdout.Len() > 0 {
		logError(fmt.Sprintf("Command output:\n%s", stdout.String()))
	}

	// Store output in context if captured
	if captureOutput {
		ctx.Variables["command_stdout"] = stdout.String()
		ctx.Variables["command_stderr"] = stderr.String()
		ctx.Variables["command_exit_code"] = "0"
	}

	return nil
}

// prepareCommand prepares the command and arguments based on the shell type
func prepareCommand(command string, args []string, shellTypeStr string) (string, []string) {
	// If no args, we need to run the command through a shell
	if len(args) == 0 {
		// Handle special cases and auto-detection
		if shellTypeStr == "auto" {
			// Auto-detect: use sh on Unix for POSIX compatibility, cmd on Windows
			if utils.IsWindows() {
				return "cmd", []string{"/C", command}
			}
			return "sh", []string{"-c", command}
		}

		// Handle explicit shell types
		switch shellTypeStr {
		case "sh":
			return "sh", []string{"-c", command}
		case "bash":
			return "bash", []string{"-c", command}
		case "cmd":
			return "cmd", []string{"/C", command}
		case "powershell":
			return "powershell", []string{"-Command", command}
		default:
			// Parse as shellutil type for other cases
			shellType := shellutil.ParseShellType(shellTypeStr)
			switch shellType {
			case shellutil.ShellTypeBash:
				return "bash", []string{"-c", command}
			case shellutil.ShellTypeCmd:
				return "cmd", []string{"/C", command}
			case shellutil.ShellTypePowerShell:
				return "powershell", []string{"-Command", command}
			default:
				// Default to sh on Unix, cmd on Windows
				if utils.IsWindows() {
					return "cmd", []string{"/C", command}
				}
				return "sh", []string{"-c", command}
			}
		}
	}

	// If args are provided, execute directly without shell
	return command, args
}

// expandTildeForCommand expands ~ to home directory (cross-platform)
func expandTildeForCommand(path string) string {
	if strings.HasPrefix(path, "~/") || path == "~" {
		home, err := os.UserHomeDir()
		if err == nil {
			if path == "~" {
				return home
			}
			return filepath.Join(home, path[2:])
		}
	}
	return path
}
