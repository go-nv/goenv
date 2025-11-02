package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLogToFileAction_Name(t *testing.T) {
	action := &LogToFileAction{}
	if action.Name() != ActionLogToFile {
		t.Errorf("Name() = %v, want %v", action.Name(), ActionLogToFile)
	}
}

func TestLogToFileAction_Description(t *testing.T) {
	action := &LogToFileAction{}
	desc := action.Description()
	if desc == "" {
		t.Error("Description() returned empty string")
	}
}

func TestLogToFileAction_Validate(t *testing.T) {
	action := &LogToFileAction{}

	tests := []struct {
		name    string
		params  map[string]interface{}
		wantErr bool
		errMsg  string
	}{
		{
			name:    "Missing file parameter",
			params:  map[string]interface{}{},
			wantErr: true,
			errMsg:  "file",
		},
		{
			name: "Empty file parameter",
			params: map[string]interface{}{
				"file": "",
			},
			wantErr: true,
			errMsg:  "file",
		},
		{
			name: "Non-string file parameter",
			params: map[string]interface{}{
				"file": 123,
			},
			wantErr: true,
			errMsg:  "file",
		},
		{
			name: "Valid file parameter",
			params: map[string]interface{}{
				"file": "/tmp/test.log",
			},
			wantErr: false,
		},
		{
			name: "Valid file with format",
			params: map[string]interface{}{
				"file":   "/tmp/test.log",
				"format": "{timestamp} | {message}",
			},
			wantErr: false,
		},
		{
			name: "Valid file with append",
			params: map[string]interface{}{
				"file":   "/tmp/test.log",
				"append": true,
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

func TestLogToFileAction_Execute(t *testing.T) {
	// Create temp directory for test files
	tempDir := t.TempDir()

	tests := []struct {
		name           string
		params         map[string]interface{}
		variables      map[string]string
		wantErr        bool
		checkContent   bool
		expectedInFile string
	}{
		{
			name: "Basic log write",
			params: map[string]interface{}{
				"file": filepath.Join(tempDir, "basic.log"),
			},
			variables: map[string]string{
				"message": "Test message",
			},
			wantErr:        false,
			checkContent:   true,
			expectedInFile: "Test message",
		},
		{
			name: "Log with custom format",
			params: map[string]interface{}{
				"file":   filepath.Join(tempDir, "formatted.log"),
				"format": "[{hook}] {version} - {message}",
			},
			variables: map[string]string{
				"hook":    "pre_install",
				"version": "1.21.0",
				"message": "Installing Go",
			},
			wantErr:        false,
			checkContent:   true,
			expectedInFile: "[pre_install] 1.21.0 - Installing Go",
		},
		{
			name: "Log with append mode",
			params: map[string]interface{}{
				"file":   filepath.Join(tempDir, "append.log"),
				"append": true,
			},
			variables: map[string]string{
				"message": "First entry",
			},
			wantErr:        false,
			checkContent:   true,
			expectedInFile: "First entry",
		},
		{
			name: "Log with truncate mode",
			params: map[string]interface{}{
				"file":   filepath.Join(tempDir, "truncate.log"),
				"append": false,
			},
			variables: map[string]string{
				"message": "New content",
			},
			wantErr:        false,
			checkContent:   true,
			expectedInFile: "New content",
		},
		{
			name: "Log creates missing directory",
			params: map[string]interface{}{
				"file": filepath.Join(tempDir, "subdir", "nested", "test.log"),
			},
			variables: map[string]string{
				"message": "Nested log",
			},
			wantErr:        false,
			checkContent:   true,
			expectedInFile: "Nested log",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			action := &LogToFileAction{}
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

				if tt.checkContent {
					file := tt.params["file"].(string)
					content, err := os.ReadFile(file)
					if err != nil {
						t.Errorf("Failed to read log file: %v", err)
					} else if !strings.Contains(string(content), tt.expectedInFile) {
						t.Errorf("Log file content = %q, want to contain %q", string(content), tt.expectedInFile)
					}

					// Verify file ends with newline
					if len(content) > 0 && content[len(content)-1] != '\n' {
						t.Error("Log file does not end with newline")
					}
				}
			}
		})
	}
}

func TestLogToFileAction_ExecuteAppendMode(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	action := &LogToFileAction{}
	logFile := filepath.Join(tempDir, "append.log")

	// Write first entry
	ctx1 := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"message": "First entry",
		},
	}
	params := map[string]interface{}{
		"file":   logFile,
		"append": true,
	}

	if err := action.Execute(ctx1, params); err != nil {
		t.Fatalf("First Execute() failed: %v", err)
	}

	// Write second entry
	ctx2 := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"message": "Second entry",
		},
	}

	if err := action.Execute(ctx2, params); err != nil {
		t.Fatalf("Second Execute() failed: %v", err)
	}

	// Verify both entries exist
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "First entry") {
		t.Error("First entry not found in log file")
	}
	if !strings.Contains(contentStr, "Second entry") {
		t.Error("Second entry not found in log file")
	}
}

func TestLogToFileAction_ExecuteTruncateMode(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	action := &LogToFileAction{}
	logFile := filepath.Join(tempDir, "truncate.log")

	// Write first entry
	ctx1 := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"message": "First entry",
		},
	}
	params := map[string]interface{}{
		"file":   logFile,
		"append": false, // Truncate mode
	}

	if err := action.Execute(ctx1, params); err != nil {
		t.Fatalf("First Execute() failed: %v", err)
	}

	// Write second entry (should replace first)
	ctx2 := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"message": "Second entry",
		},
	}

	if err := action.Execute(ctx2, params); err != nil {
		t.Fatalf("Second Execute() failed: %v", err)
	}

	// Verify only second entry exists
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)
	if strings.Contains(contentStr, "First entry") {
		t.Error("First entry should not exist in truncate mode")
	}
	if !strings.Contains(contentStr, "Second entry") {
		t.Error("Second entry not found in log file")
	}
}

func TestLogToFileAction_ExecuteTimestamp(t *testing.T) {
	// Create temp directory
	tempDir := t.TempDir()

	action := &LogToFileAction{}
	logFile := filepath.Join(tempDir, "timestamp.log")

	// Test that timestamp gets interpolated when provided in variables
	ctx := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"message": "Test message",
			"hook":    "pre_install",
		},
	}
	params := map[string]interface{}{
		"file":   logFile,
		"format": "[{hook}] {message}",
	}

	if err := action.Execute(ctx, params); err != nil {
		t.Fatalf("Execute() failed: %v", err)
	}

	// Verify content
	content, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatalf("Failed to read log file: %v", err)
	}

	contentStr := string(content)

	// Should contain interpolated values
	if !strings.Contains(contentStr, "[pre_install]") {
		t.Errorf("Log file does not contain interpolated hook, content: %s", contentStr)
	}
	if !strings.Contains(contentStr, "Test message") {
		t.Errorf("Log file does not contain message, content: %s", contentStr)
	}
	// Verify template variables were interpolated
	if strings.Contains(contentStr, "{hook}") || strings.Contains(contentStr, "{message}") {
		t.Errorf("Template variables were not interpolated, content: %s", contentStr)
	}
}

func TestExpandTilde(t *testing.T) {
	tests := []struct {
		name  string
		path  string
		check func(string) bool
	}{
		{
			name: "Path with tilde",
			path: "~/test/file.log",
			check: func(result string) bool {
				return !strings.HasPrefix(result, "~") && strings.Contains(filepath.ToSlash(result), "test/file.log")
			},
		},
		{
			name: "Path without tilde",
			path: "/absolute/path/file.log",
			check: func(result string) bool {
				return result == "/absolute/path/file.log"
			},
		},
		{
			name: "Relative path",
			path: "relative/path/file.log",
			check: func(result string) bool {
				return result == "relative/path/file.log"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := expandTilde(tt.path)
			if !tt.check(result) {
				t.Errorf("expandTilde(%q) = %q, check failed", tt.path, result)
			}
		})
	}
}
