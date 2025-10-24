package hooks

import (
	"runtime"
	"strings"
	"testing"
)

func TestRunCommandAction_Name(t *testing.T) {
	action := &RunCommandAction{}
	if action.Name() != ActionRunCommand {
		t.Errorf("Name() = %v, want %v", action.Name(), ActionRunCommand)
	}
}

func TestRunCommandAction_Description(t *testing.T) {
	action := &RunCommandAction{}
	desc := action.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
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
				if err != nil {
					t.Errorf("Validate() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRunCommandAction_Execute_Simple(t *testing.T) {
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
	if runtime.GOOS == "windows" {
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

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestRunCommandAction_Execute_WithArgs(t *testing.T) {
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
	if runtime.GOOS == "windows" {
		cmdName = "cmd"
	} else {
		cmdName = "echo"
	}

	params := map[string]interface{}{
		"command": cmdName,
		"args":    []interface{}{"hello", "world"},
	}

	// On Windows, we need to adjust the command
	if runtime.GOOS == "windows" {
		params["args"] = []interface{}{"/C", "echo", "hello", "world"}
	}

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}
}

func TestRunCommandAction_Execute_CaptureOutput(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping command execution test in short mode")
	}

	action := &RunCommandAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	var params map[string]interface{}
	if runtime.GOOS == "windows" {
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

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Check that output was captured
	if _, exists := ctx.Variables["command_stdout"]; !exists {
		t.Error("command_stdout was not set in context")
	}

	// Verify the output contains our test string
	stdout := ctx.Variables["command_stdout"]
	if !strings.Contains(stdout, "test output") {
		t.Errorf("command_stdout = %q, want to contain 'test output'", stdout)
	}
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
	if runtime.GOOS == "windows" {
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
	if err == nil {
		t.Error("Execute() expected error for failing command, got nil")
	}
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
	if runtime.GOOS == "windows" {
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
	if err != nil {
		t.Errorf("Execute() with fail_on_error=false should not error, got: %v", err)
	}
}

func TestRunCommandAction_Execute_Interpolation(t *testing.T) {
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
	if runtime.GOOS == "windows" {
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

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	stdout := ctx.Variables["command_stdout"]
	if !strings.Contains(stdout, "hello") {
		t.Errorf("command_stdout should contain interpolated 'hello', got: %q", stdout)
	}
	if !strings.Contains(stdout, "1.21.0") {
		t.Errorf("command_stdout should contain interpolated '1.21.0', got: %q", stdout)
	}
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
	if runtime.GOOS == "windows" {
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
	if err == nil {
		t.Error("Execute() expected timeout error, got nil")
	}
	if !strings.Contains(err.Error(), "timeout") && !strings.Contains(err.Error(), "timed out") {
		t.Errorf("Execute() error should mention timeout, got: %v", err)
	}
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

			if gotCmd != tt.wantCmd {
				t.Errorf("prepareCommand() cmd = %q, want %q", gotCmd, tt.wantCmd)
			}

			if len(gotArgs) != len(tt.wantArgs) {
				t.Errorf("prepareCommand() args length = %d, want %d", len(gotArgs), len(tt.wantArgs))
				return
			}

			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("prepareCommand() args[%d] = %q, want %q", i, gotArgs[i], tt.wantArgs[i])
				}
			}
		})
	}
}

func TestPrepareCommand_Auto(t *testing.T) {
	// Test auto-detection
	cmd, args := prepareCommand("echo test", []string{}, "auto")

	if runtime.GOOS == "windows" {
		if cmd != "cmd" {
			t.Errorf("Auto-detect on Windows: cmd = %q, want 'cmd'", cmd)
		}
		if len(args) != 2 || args[0] != "/C" {
			t.Errorf("Auto-detect on Windows: args = %v, want [/C ...]", args)
		}
	} else {
		if cmd != "sh" {
			t.Errorf("Auto-detect on Unix: cmd = %q, want 'sh'", cmd)
		}
		if len(args) != 2 || args[0] != "-c" {
			t.Errorf("Auto-detect on Unix: args = %v, want [-c ...]", args)
		}
	}
}
