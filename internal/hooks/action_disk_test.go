package hooks

import (
	"strings"
	"testing"
)

func TestCheckDiskSpaceAction_Name(t *testing.T) {
	action := &CheckDiskSpaceAction{}
	if action.Name() != ActionCheckDiskSpace {
		t.Errorf("Name() = %v, want %v", action.Name(), ActionCheckDiskSpace)
	}
}

func TestCheckDiskSpaceAction_Description(t *testing.T) {
	action := &CheckDiskSpaceAction{}
	desc := action.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

func TestCheckDiskSpaceAction_Validate(t *testing.T) {
	action := &CheckDiskSpaceAction{}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Missing path parameter",
			params:  map[string]interface{}{},
			wantErr: true,
			errMsg:  "path",
		},
		{
			name: "Empty path parameter",
			params: map[string]interface{}{
				"path": "",
			},
			wantErr: true,
			errMsg:  "path",
		},
		{
			name: "Non-string path parameter",
			params: map[string]interface{}{
				"path": 123,
			},
			wantErr: true,
			errMsg:  "path",
		},
		{
			name: "Missing min_free_mb parameter",
			params: map[string]interface{}{
				"path": "/tmp",
			},
			wantErr: true,
			errMsg:  "min_free_mb",
		},
		{
			name: "Non-numeric min_free_mb parameter",
			params: map[string]interface{}{
				"path":        "/tmp",
				"min_free_mb": "not a number",
			},
			wantErr: true,
			errMsg:  "min_free_mb",
		},
		{
			name: "Negative min_free_mb parameter",
			params: map[string]interface{}{
				"path":        "/tmp",
				"min_free_mb": -100,
			},
			wantErr: true,
			errMsg:  "non-negative",
		},
		{
			name: "Valid parameters with int",
			params: map[string]interface{}{
				"path":        "/tmp",
				"min_free_mb": 100,
			},
			wantErr: false,
		},
		{
			name: "Valid parameters with float",
			params: map[string]interface{}{
				"path":        "/tmp",
				"min_free_mb": 100.0,
			},
			wantErr: false,
		},
		{
			name: "Valid with warn action",
			params: map[string]interface{}{
				"path":            "/tmp",
				"min_free_mb":     100,
				"on_insufficient": "warn",
			},
			wantErr: false,
		},
		{
			name: "Valid with error action",
			params: map[string]interface{}{
				"path":            "/tmp",
				"min_free_mb":     100,
				"on_insufficient": "error",
			},
			wantErr: false,
		},
		{
			name: "Invalid on_insufficient value",
			params: map[string]interface{}{
				"path":            "/tmp",
				"min_free_mb":     100,
				"on_insufficient": "invalid",
			},
			wantErr: true,
			errMsg:  "on_insufficient",
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

func TestCheckDiskSpaceAction_Execute(t *testing.T) {
	action := &CheckDiskSpaceAction{}

	// Create a temp directory to test against
	tempDir := t.TempDir()

	tests := []struct {
		name      string
		params    map[string]interface{}
		variables map[string]string
		wantErr   bool
	}{
		{
			name: "Sufficient disk space",
			params: map[string]interface{}{
				"path":        tempDir,
				"min_free_mb": 1, // Very small requirement - should always pass
			},
			variables: map[string]string{},
			wantErr:   false,
		},
		{
			name: "Insufficient disk space with error",
			params: map[string]interface{}{
				"path":            tempDir,
				"min_free_mb":     999999999, // Huge requirement - should fail
				"on_insufficient": "error",
			},
			variables: map[string]string{},
			wantErr:   true,
		},
		{
			name: "Insufficient disk space with warn",
			params: map[string]interface{}{
				"path":            tempDir,
				"min_free_mb":     999999999, // Huge requirement
				"on_insufficient": "warn",
			},
			variables: map[string]string{},
			wantErr:   false, // Should not error, just warn
		},
		{
			name: "Path with variable interpolation",
			params: map[string]interface{}{
				"path":        "{install_path}",
				"min_free_mb": 1,
			},
			variables: map[string]string{
				"install_path": tempDir,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := &HookContext{
				Command:   "install",
				Variables: tt.variables,
			}

			err := action.Execute(ctx, tt.params)
			if tt.wantErr {
				if err == nil {
					t.Error("Execute() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("Execute() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestCheckDiskSpaceAction_ExecuteFloatConversion(t *testing.T) {
	action := &CheckDiskSpaceAction{}

	tempDir := t.TempDir()

	// Test that float64 min_free_mb works (common when parsing YAML)
	ctx := &HookContext{
		Command:   "install",
		Variables: map[string]string{},
	}

	params := map[string]interface{}{
		"path":        tempDir,
		"min_free_mb": 1.5, // Float value
	}

	if err := action.Execute(ctx, params); err != nil {
		t.Errorf("Execute() with float min_free_mb failed: %v", err)
	}
}
