package hooks

import (
	"os"
	"strings"
	"testing"
)

func TestSetEnvAction_Name(t *testing.T) {
	action := &SetEnvAction{}
	if action.Name() != ActionSetEnv {
		t.Errorf("Name() = %v, want %v", action.Name(), ActionSetEnv)
	}
}

func TestSetEnvAction_Description(t *testing.T) {
	action := &SetEnvAction{}
	desc := action.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

func TestSetEnvAction_Validate(t *testing.T) {
	action := &SetEnvAction{}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Missing variables parameter",
			params:  map[string]interface{}{},
			wantErr: true,
			errMsg:  "variables",
		},
		{
			name: "Empty variables map",
			params: map[string]interface{}{
				"variables": map[string]interface{}{},
			},
			wantErr: true,
			errMsg:  "variables",
		},
		{
			name: "Non-map variables parameter",
			params: map[string]interface{}{
				"variables": "not a map",
			},
			wantErr: true,
			errMsg:  "variables",
		},
		{
			name: "Valid single variable",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"MY_VAR": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid multiple variables",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"VAR1": "value1",
					"VAR2": "value2",
					"VAR3": "value3",
				},
			},
			wantErr: false,
		},
		{
			name: "Invalid variable name - starts with digit",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"1VAR": "value",
				},
			},
			wantErr: true,
			errMsg:  "must start with a letter or underscore",
		},
		{
			name: "Invalid variable name - contains space",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"MY VAR": "value",
				},
			},
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name: "Invalid variable name - contains hyphen",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"MY-VAR": "value",
				},
			},
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name: "Invalid variable name - reserved PATH",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"PATH": "/some/path",
				},
			},
			wantErr: true,
			errMsg:  "reserved variable",
		},
		{
			name: "Invalid variable name - reserved HOME",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"HOME": "/home/user",
				},
			},
			wantErr: true,
			errMsg:  "reserved variable",
		},
		{
			name: "Non-string variable value",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"MY_VAR": 123,
				},
			},
			wantErr: true,
			errMsg:  "must be a string",
		},
		{
			name: "Valid variable with underscore",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"_PRIVATE_VAR": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid variable with numbers",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"VAR123": "value",
				},
			},
			wantErr: false,
		},
		{
			name: "Valid with hook scope",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"MY_VAR": "value",
				},
				"scope": "hook",
			},
			wantErr: false,
		},
		{
			name: "Valid with process scope",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"MY_VAR": "value",
				},
				"scope": "process",
			},
			wantErr: false,
		},
		{
			name: "Invalid scope",
			params: map[string]interface{}{
				"variables": map[string]interface{}{
					"MY_VAR": "value",
				},
				"scope": "invalid",
			},
			wantErr: true,
			errMsg:  "invalid scope",
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

func TestSetEnvAction_ExecuteHookScope(t *testing.T) {
	action := &SetEnvAction{}

	ctx := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"existing": "value",
		},
	}

	params := map[string]interface{}{
		"variables": map[string]interface{}{
			"NEW_VAR":     "new_value",
			"ANOTHER_VAR": "another_value",
		},
		"scope": "hook",
	}

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify variables were added to context
	if ctx.Variables["NEW_VAR"] != "new_value" {
		t.Errorf("NEW_VAR not set in context, got %q", ctx.Variables["NEW_VAR"])
	}
	if ctx.Variables["ANOTHER_VAR"] != "another_value" {
		t.Errorf("ANOTHER_VAR not set in context, got %q", ctx.Variables["ANOTHER_VAR"])
	}

	// Verify existing variable is still there
	if ctx.Variables["existing"] != "value" {
		t.Error("Existing variable was modified")
	}
}

func TestSetEnvAction_ExecuteProcessScope(t *testing.T) {
	action := &SetEnvAction{}

	// Use unique variable names to avoid conflicts
	varName := "GOENV_TEST_VAR_PROCESS"
	varValue := "test_value_123"

	// Clean up before and after
	os.Unsetenv(varName)
	defer os.Unsetenv(varName)

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	params := map[string]interface{}{
		"variables": map[string]interface{}{
			varName: varValue,
		},
		"scope": "process",
	}

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify variable was set in process environment
	if got := os.Getenv(varName); got != varValue {
		t.Errorf("Process env %s = %q, want %q", varName, got, varValue)
	}
}

func TestSetEnvAction_ExecuteDefaultScope(t *testing.T) {
	action := &SetEnvAction{}

	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	params := map[string]interface{}{
		"variables": map[string]interface{}{
			"MY_VAR": "value",
		},
		// No scope specified - should default to "hook"
	}

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify variable was added to context (default scope is hook)
	if ctx.Variables["MY_VAR"] != "value" {
		t.Errorf("MY_VAR not set in context with default scope")
	}
}

func TestSetEnvAction_ExecuteInterpolation(t *testing.T) {
	action := &SetEnvAction{}

	ctx := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"version": "1.21.0",
			"prefix":  "/usr/local",
		},
	}

	params := map[string]interface{}{
		"variables": map[string]interface{}{
			"INSTALL_PATH": "{prefix}/go/{version}",
			"GO_VERSION":   "go{version}",
		},
		"scope": "hook",
	}

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify interpolation worked
	if got := ctx.Variables["INSTALL_PATH"]; got != "/usr/local/go/1.21.0" {
		t.Errorf("INSTALL_PATH = %q, want %q", got, "/usr/local/go/1.21.0")
	}
	if got := ctx.Variables["GO_VERSION"]; got != "go1.21.0" {
		t.Errorf("GO_VERSION = %q, want %q", got, "go1.21.0")
	}
}

func TestValidateEnvVarName(t *testing.T) {
	tests := []struct {
		name    string
		varName string
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Empty name",
			varName: "",
			wantErr: true,
			errMsg:  "cannot be empty",
		},
		{
			name:    "Valid uppercase",
			varName: "MY_VAR",
			wantErr: false,
		},
		{
			name:    "Valid lowercase",
			varName: "my_var",
			wantErr: false,
		},
		{
			name:    "Valid with numbers",
			varName: "VAR123",
			wantErr: false,
		},
		{
			name:    "Valid starts with underscore",
			varName: "_PRIVATE",
			wantErr: false,
		},
		{
			name:    "Invalid starts with number",
			varName: "1VAR",
			wantErr: true,
			errMsg:  "must start with a letter or underscore",
		},
		{
			name:    "Invalid contains space",
			varName: "MY VAR",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "Invalid contains hyphen",
			varName: "MY-VAR",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "Invalid contains dot",
			varName: "MY.VAR",
			wantErr: true,
			errMsg:  "invalid character",
		},
		{
			name:    "Reserved PATH",
			varName: "PATH",
			wantErr: true,
			errMsg:  "reserved variable",
		},
		{
			name:    "Reserved path (lowercase)",
			varName: "path",
			wantErr: true,
			errMsg:  "reserved variable",
		},
		{
			name:    "Reserved HOME",
			varName: "HOME",
			wantErr: true,
			errMsg:  "reserved variable",
		},
		{
			name:    "Reserved USER",
			varName: "USER",
			wantErr: true,
			errMsg:  "reserved variable",
		},
		{
			name:    "Reserved SHELL",
			varName: "SHELL",
			wantErr: true,
			errMsg:  "reserved variable",
		},
		{
			name:    "Reserved TERM",
			varName: "TERM",
			wantErr: true,
			errMsg:  "reserved variable",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateEnvVarName(tt.varName)
			if tt.wantErr {
				if err == nil {
					t.Errorf("validateEnvVarName() expected error containing %q, got nil", tt.errMsg)
				} else if !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("validateEnvVarName() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("validateEnvVarName() unexpected error: %v", err)
				}
			}
		})
	}
}
