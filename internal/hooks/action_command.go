package hooks

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
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
		if shell != "auto" && shell != "bash" && shell != "sh" && shell != "cmd" && shell != "powershell" {
			return fmt.Errorf("invalid shell: %s (must be auto, bash, sh, cmd, or powershell)", shell)
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

	// Execute command
	cmdToRun, cmdArgs := prepareCommand(command, args, shellType)

	execCmd := exec.Command(cmdToRun, cmdArgs...)
	if workDir != "" {
		execCmd.Dir = workDir
	}

	// Setup output capture
	var stdout, stderr bytes.Buffer
	if captureOutput || logOutput {
		execCmd.Stdout = &stdout
		execCmd.Stderr = &stderr
	}

	// Start the command
	if err := execCmd.Start(); err != nil {
		return fmt.Errorf("failed to start command: %v", err)
	}

	// Wait for completion or timeout
	done := make(chan error, 1)
	go func() {
		done <- execCmd.Wait()
	}()

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case err := <-done:
		// Command completed normally
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

	case <-timer.C:
		// Timeout - kill the process
		if execCmd.Process != nil {
			execCmd.Process.Kill()
		}

		// Wait for the process to finish being killed
		<-done

		errMsg := fmt.Sprintf("command timed out after %s", timeout)
		if logOutput {
			logError(errMsg)
		}

		if failOnError {
			return fmt.Errorf("%s", errMsg)
		}
		return nil
	}
}

// prepareCommand prepares the command and arguments based on the shell type
func prepareCommand(command string, args []string, shellType string) (string, []string) {
	// Auto-detect shell based on platform
	if shellType == "auto" {
		if runtime.GOOS == "windows" {
			shellType = "cmd"
		} else {
			shellType = "sh"
		}
	}

	// If no args, we need to run the command through a shell
	if len(args) == 0 {
		switch shellType {
		case "bash":
			return "bash", []string{"-c", command}
		case "sh":
			return "sh", []string{"-c", command}
		case "cmd":
			return "cmd", []string{"/C", command}
		case "powershell":
			return "powershell", []string{"-Command", command}
		default:
			// Default to sh on Unix, cmd on Windows
			if runtime.GOOS == "windows" {
				return "cmd", []string{"/C", command}
			}
			return "sh", []string{"-c", command}
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
