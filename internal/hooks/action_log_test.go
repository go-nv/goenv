package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLogToFileAction_Name(t *testing.T) {
	action := &LogToFileAction{}
	assert.Equal(t, ActionLogToFile, action.Name(), "Name() =")
}

func TestLogToFileAction_Description(t *testing.T) {
	action := &LogToFileAction{}
	desc := action.Description()
	assert.NotEmpty(t, desc, "Description() returned empty string")
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
				assert.NoError(t, err, "Validate() unexpected error")
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
				assert.Error(t, err, "Execute() expected error, got nil")
			} else {
				assert.NoError(t, err, "Execute() unexpected error")

				if tt.checkContent {
					file := tt.params["file"].(string)
					content, err := os.ReadFile(file)
					if err != nil {
						t.Errorf("Failed to read log file: %v", err)
					} else if !strings.Contains(string(content), tt.expectedInFile) {
						t.Errorf("Log file content = %q, want to contain %q", string(content), tt.expectedInFile)
					}

					// Verify file ends with newline
					assert.False(t, len(content) > 0 && content[len(content)-1] != '\n', "Log file does not end with newline")
				}
			}
		})
	}
}

func TestLogToFileAction_ExecuteAppendMode(t *testing.T) {
	var err error
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

	err = action.Execute(ctx1, params)
	require.NoError(t, err, "First Execute() failed")

	// Write second entry
	ctx2 := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"message": "Second entry",
		},
	}

	err = action.Execute(ctx2, params)
	require.NoError(t, err, "Second Execute() failed")

	// Verify both entries exist
	content, err := os.ReadFile(logFile)
	require.NoError(t, err, "Failed to read log file")

	contentStr := string(content)
	assert.Contains(t, contentStr, "First entry", "First entry not found in log file")
	assert.Contains(t, contentStr, "Second entry", "Second entry not found in log file")
}

func TestLogToFileAction_ExecuteTruncateMode(t *testing.T) {
	var err error
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

	err = action.Execute(ctx1, params)
	require.NoError(t, err, "First Execute() failed")

	// Write second entry (should replace first)
	ctx2 := &HookContext{
		Command: "install",
		Variables: map[string]string{
			"message": "Second entry",
		},
	}

	err = action.Execute(ctx2, params)
	require.NoError(t, err, "Second Execute() failed")

	// Verify only second entry exists
	content, err := os.ReadFile(logFile)
	require.NoError(t, err, "Failed to read log file")

	contentStr := string(content)
	assert.NotContains(t, contentStr, "First entry", "First entry should not exist in truncate mode")
	assert.Contains(t, contentStr, "Second entry", "Second entry not found in log file")
}

func TestLogToFileAction_ExecuteTimestamp(t *testing.T) {
	var err error
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

	err = action.Execute(ctx, params)
	require.NoError(t, err, "Execute() failed")

	// Verify content
	content, err := os.ReadFile(logFile)
	require.NoError(t, err, "Failed to read log file")

	contentStr := string(content)

	// Should contain interpolated values
	assert.Contains(t, contentStr, "[pre_install]", "Log file does not contain interpolated hook, content %v", contentStr)
	assert.Contains(t, contentStr, "Test message", "Log file does not contain message, content %v", contentStr)
	// Verify template variables were interpolated
	assert.False(t, strings.Contains(contentStr, "{hook}") || strings.Contains(contentStr, "{message}"), "Template variables were not interpolated, content")
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
			assert.True(t, tt.check(result), "expandTilde() = , check failed")
		})
	}
}
