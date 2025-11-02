package hooks

import (
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRunCommandAction_Name(t *testing.T) {
	action := &RunCommandAction{}
	assert.Equal(t, ActionRunCommand, action.Name(), "Name() =")
}

func TestRunCommandAction_Description(t *testing.T) {
	action := &RunCommandAction{}
	desc := action.Description()
	assert.NotEmpty(t, desc, "Description() returned empty string")
}

func TestRunCommandAction_Validate(t *testing.T) {
	action := &RunCommandAction{}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Missing command parameter",
			params:  map[string]interface{}{},
			wantErr: true,
			errMsg:  "command parameter is required",
		},
		{
			name: "Empty command",
			params: map[string]interface{}{
				"command": "",
			},
			wantErr: true,
			errMsg:  "command parameter is required",
		},
		{
			name: "Valid simple command",
			params: map[string]interface{}{
				"command": "echo hello",
			},
			wantErr: false,
		},
		{
			name: "Valid command with args",
			params: map[string]interface{}{
				"command": "echo",
				"args":    []interface{}{"hello", "world"},
			},
			wantErr: false,
		},
		{
			name: "Invalid args - not array",
			params: map[string]interface{}{
				"command": "echo",
				"args":    "not an array",
			},
			wantErr: true,
			errMsg:  "args parameter must be an array",
		},
		{
			name: "Invalid args - non-string element",
			params: map[string]interface{}{
				"command": "echo",
				"args":    []interface{}{"valid", 123},
			},
			wantErr: true,
			errMsg:  "must be a string",
		},
		{
			name: "Valid working_dir",
			params: map[string]interface{}{
				"command":     "pwd",
				"working_dir": "/tmp",
			},
			wantErr: false,
		},
		{
			name: "Invalid working_dir - path traversal",
			params: map[string]interface{}{
				"command":     "pwd",
				"working_dir": "/tmp/../etc",
			},
			wantErr: true,
			errMsg:  "path traversal",
		},
		{
			name: "Valid timeout",
			params: map[string]interface{}{
				"command": "sleep 1",
				"timeout": "30s",
			},
			wantErr: false,
		},
		{
			name: "Invalid timeout format",
			params: map[string]interface{}{
				"command": "echo test",
				"timeout": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid timeout format",
		},
		{
			name: "Valid capture_output",
			params: map[string]interface{}{
				"command":        "echo test",
				"capture_output": true,
			},
			wantErr: false,
		},
		{
			name: "Invalid capture_output type",
			params: map[string]interface{}{
				"command":        "echo test",
				"capture_output": "yes",
			},
			wantErr: true,
			errMsg:  "capture_output parameter must be a boolean",
		},
		{
			name: "Valid log_output",
			params: map[string]interface{}{
				"command":    "echo test",
				"log_output": true,
			},
			wantErr: false,
		},
		{
			name: "Invalid log_output type",
			params: map[string]interface{}{
				"command":    "echo test",
				"log_output": "yes",
			},
			wantErr: true,
			errMsg:  "log_output parameter must be a boolean",
		},
		{
			name: "Valid fail_on_error",
			params: map[string]interface{}{
				"command":       "echo test",
				"fail_on_error": false,
			},
			wantErr: false,
		},
		{
			name: "Invalid fail_on_error type",
			params: map[string]interface{}{
				"command":       "echo test",
				"fail_on_error": "no",
			},
			wantErr: true,
			errMsg:  "fail_on_error parameter must be a boolean",
		},
		{
			name: "Valid shell",
			params: map[string]interface{}{
				"command": "echo test",
				"shell":   "bash",
			},
			wantErr: false,
		},
		{
			name: "Invalid shell",
			params: map[string]interface{}{
				"command": "echo test",
				"shell":   "invalid",
			},
			wantErr: true,
			errMsg:  "invalid shell",
		},
		{
			name: "All valid parameters combined",
			params: map[string]interface{}{
				"command":        "echo",
				"args":           []interface{}{"test"},
				"working_dir":    "/tmp",
				"timeout":        "5s",
				"capture_output": true,
				"log_output":     false,
				"fail_on_error":  true,
				"shell":          "sh",
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := action.Validate(tt.params)
			if tt.wantErr {
				if err == nil {
					t.Errorf("Validate() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Validate() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				assert.NoError(t, err, "Validate() unexpected error")
			}
		})
	}
}

func TestRunCommandAction_Execute_Simple(t *testing.T) {
	var err error
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	// Simple echo command (cross-platform)
	var params map[string]interface{}
	if utils.IsWindows() {
		params = map[string]interface{}{
			"command": "echo hello",
			"shell":   "cmd",
		}
	} else {
		params = map[string]interface{}{
			"command": "echo hello",
			"shell":   "sh",
		}
	}

	err = action.Execute(ctx, params)
	require.NoError(t, err, "Execute() failed")
}

func TestRunCommandAction_Execute_WithArgs(t *testing.T) {
	var err error
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	// Use echo with arguments
	var cmdName string
	if utils.IsWindows() {
		cmdName = "cmd"
	} else {
		cmdName = "echo"
	}

	params := map[string]interface{}{
		"command": cmdName,
		"args":    []interface{}{"hello", "world"},
	}

	// On Windows, we need to adjust the command
	if utils.IsWindows() {
		params["args"] = []interface{}{"/C", "echo", "hello", "world"}
	}

	err = action.Execute(ctx, params)
	require.NoError(t, err, "Execute() failed")
}

func TestRunCommandAction_Execute_CaptureOutput(t *testing.T) {
	var err error
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	var params map[string]interface{}
	if utils.IsWindows() {
		params = map[string]interface{}{
			"command":        "echo test output",
			"capture_output": true,
			"shell":          "cmd",
		}
	} else {
		params = map[string]interface{}{
			"command":        "echo test output",
			"capture_output": true,
			"shell":          "sh",
		}
	}

	err = action.Execute(ctx, params)
	require.NoError(t, err, "Execute() failed")

	// Check that output was captured
	if _, exists := ctx.Variables["command_stdout"]; !exists {
		t.Error("command_stdout was not set in context")
	}

	// Verify the output contains our test string
	stdout := ctx.Variables["command_stdout"]
	assert.Contains(t, stdout, "test output", "command_stdout = %v", stdout)
}

func TestRunCommandAction_Execute_FailOnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	// Use a command that will fail
	var params map[string]interface{}
	if utils.IsWindows() {
		params = map[string]interface{}{
			"command":       "exit 1",
			"fail_on_error": true,
			"shell":         "cmd",
		}
	} else {
		params = map[string]interface{}{
			"command":       "exit 1",
			"fail_on_error": true,
			"shell":         "sh",
		}
	}

	err := action.Execute(ctx, params)
	assert.Error(t, err, "Execute() expected error for failing command, got nil")
}

func TestRunCommandAction_Execute_NoFailOnError(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	// Use a command that will fail, but with fail_on_error=false
	var params map[string]interface{}
	if utils.IsWindows() {
		params = map[string]interface{}{
			"command":       "exit 1",
			"fail_on_error": false,
			"shell":         "cmd",
		}
	} else {
		params = map[string]interface{}{
			"command":       "exit 1",
			"fail_on_error": false,
			"shell":         "sh",
		}
	}

	err := action.Execute(ctx, params)
	assert.NoError(t, err, "Execute() with fail_on_error=false should not error")
}

func TestRunCommandAction_Execute_Interpolation(t *testing.T) {
	var err error
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"version": "1.21.0",
			"text":    "hello",
		},
	}

	var params map[string]interface{}
	if utils.IsWindows() {
		params = map[string]interface{}{
			"command":        "echo {text} {version}",
			"capture_output": true,
			"shell":          "cmd",
		}
	} else {
		params = map[string]interface{}{
			"command":        "echo {text} {version}",
			"capture_output": true,
			"shell":          "sh",
		}
	}

	err = action.Execute(ctx, params)
	require.NoError(t, err, "Execute() failed")

	stdout := ctx.Variables["command_stdout"]
	assert.Contains(t, stdout, "hello", "command_stdout should contain interpolated 'hello' %v", stdout)
	assert.Contains(t, stdout, "1.21.0", "command_stdout should contain interpolated '1.21.0' %v", stdout)
}

func TestRunCommandAction_Execute_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	// Command that takes longer than timeout
	var params map[string]interface{}
	if utils.IsWindows() {
		params = map[string]interface{}{
			"command":       "Start-Sleep -Seconds 10",
			"timeout":       "1s",
			"fail_on_error": true,
			"shell":         "powershell",
		}
	} else {
		params = map[string]interface{}{
			"command":       "sleep 10",
			"timeout":       "1s",
			"fail_on_error": true,
			"shell":         "sh",
		}
	}

	err := action.Execute(ctx, params)
	assert.Error(t, err, "Execute() expected timeout error, got nil")
	assert.True(t, strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "timed out"), "Execute() error should mention timeout")
}

func TestPrepareCommand(t *testing.T) {
	tests := []struct {
		name      string
		command   string
		args      []string
		shellType string
		wantCmd   string
		wantArgs  []string
	}{
		{
			name:      "No args - sh",
			command:   "echo hello",
			args:      []string{},
			shellType: "sh",
			wantCmd:   "sh",
			wantArgs:  []string{"-c", "echo hello"},
		},
		{
			name:      "No args - bash",
			command:   "echo hello",
			args:      []string{},
			shellType: "bash",
			wantCmd:   "bash",
			wantArgs:  []string{"-c", "echo hello"},
		},
		{
			name:      "With args - direct execution",
			command:   "echo",
			args:      []string{"hello", "world"},
			shellType: "sh",
			wantCmd:   "echo",
			wantArgs:  []string{"hello", "world"},
		},
		{
			name:      "No args - cmd (Windows)",
			command:   "echo hello",
			args:      []string{},
			shellType: "cmd",
			wantCmd:   "cmd",
			wantArgs:  []string{"/C", "echo hello"},
		},
		{
			name:      "No args - powershell",
			command:   "Get-Date",
			args:      []string{},
			shellType: "powershell",
			wantCmd:   "powershell",
			wantArgs:  []string{"-Command", "Get-Date"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gotCmd, gotArgs := prepareCommand(tt.command, tt.args, tt.shellType)

			assert.Equal(t, tt.wantCmd, gotCmd, "prepareCommand() cmd =")

			assert.Len(t, gotArgs, len(tt.wantArgs), "prepareCommand() args length =")

			for i := range gotArgs {
				assert.Equal(t, tt.wantArgs[i], gotArgs[i], "prepareCommand() args[] =")
			}
		})
	}
}

func TestPrepareCommand_Auto(t *testing.T) {
	// Test auto-detection
	cmd, args := prepareCommand("echo test", []string{}, "auto")

	if utils.IsWindows() {
		assert.Equal(t, "cmd", cmd, "Auto-detect on Windows: cmd =")
		assert.False(t, len(args) != 2 || args[0] != "/C", "Auto-detect on Windows: args =")
	} else {
		assert.Equal(t, "sh", cmd, "Auto-detect on Unix: cmd =")
		assert.False(t, len(args) != 2 || args[0] != "-c", "Auto-detect on Unix: args =")
	}
}
