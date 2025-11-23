package cmdutil

import (
	"bytes"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSetupContext(t *testing.T) {
	cfg, mgr := SetupContext()

	assert.NotNil(t, cfg, "SetupContext returned nil config")

	assert.NotNil(t, mgr, "SetupContext returned nil manager")
}

func TestOutputJSON(t *testing.T) {
	tests := []struct {
		name     string
		data     interface{}
		contains string
	}{
		{
			name:     "simple map",
			data:     map[string]string{"key": "value"},
			contains: `"key": "value"`,
		},
		{
			name:     "struct",
			data:     struct{ Name string }{"test"},
			contains: `"Name": "test"`,
		},
		{
			name:     "array",
			data:     []string{"a", "b"},
			contains: `["a","b"]`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			err := OutputJSON(&buf, tt.data)
			assert.NoError(t, err, "OutputJSON() error =")

			output := buf.String()
			// Remove whitespace for comparison
			output = strings.ReplaceAll(output, " ", "")
			output = strings.ReplaceAll(output, "\n", "")
			contains := strings.ReplaceAll(tt.contains, " ", "")

			assert.Contains(t, output, contains, "OutputJSON() output = , should contain %v %v", output, contains)
		})
	}
}

func TestValidateExactArgs(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected int
		argName  string
		wantErr  bool
	}{
		{
			name:     "correct number of args",
			args:     []string{"arg1"},
			expected: 1,
			argName:  "version",
			wantErr:  false,
		},
		{
			name:     "too few args",
			args:     []string{},
			expected: 1,
			argName:  "version",
			wantErr:  true,
		},
		{
			name:     "too many args",
			args:     []string{"arg1", "arg2"},
			expected: 1,
			argName:  "version",
			wantErr:  true,
		},
		{
			name:     "multiple args correct",
			args:     []string{"arg1", "arg2"},
			expected: 2,
			argName:  "versions",
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateExactArgs(tt.args, tt.expected, tt.argName)
			assert.Equal(t, tt.wantErr, (err != nil), "ValidateExactArgs() error = , wantErr")
		})
	}
}

func TestValidateMinArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		min         int
		description string
		wantErr     bool
	}{
		{
			name:        "enough args",
			args:        []string{"arg1", "arg2"},
			min:         1,
			description: "at least one argument",
			wantErr:     false,
		},
		{
			name:        "exact minimum",
			args:        []string{"arg1"},
			min:         1,
			description: "at least one argument",
			wantErr:     false,
		},
		{
			name:        "too few args",
			args:        []string{},
			min:         1,
			description: "at least one argument",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMinArgs(tt.args, tt.min, tt.description)
			assert.Equal(t, tt.wantErr, (err != nil), "ValidateMinArgs() error = , wantErr")
		})
	}
}

func TestValidateMaxArgs(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		max         int
		description string
		wantErr     bool
	}{
		{
			name:        "within limit",
			args:        []string{"arg1"},
			max:         2,
			description: "at most two arguments",
			wantErr:     false,
		},
		{
			name:        "exact maximum",
			args:        []string{"arg1", "arg2"},
			max:         2,
			description: "at most two arguments",
			wantErr:     false,
		},
		{
			name:        "too many args",
			args:        []string{"arg1", "arg2", "arg3"},
			max:         2,
			description: "at most two arguments",
			wantErr:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateMaxArgs(tt.args, tt.max, tt.description)
			assert.Equal(t, tt.wantErr, (err != nil), "ValidateMaxArgs() error = , wantErr")
		})
	}
}

// mockManager implements the interface needed by RequireInstalledVersion
type mockManager struct {
	installedVersions map[string]bool
}

func (m *mockManager) IsVersionInstalled(version string) bool {
	return m.installedVersions[version]
}

func TestRequireInstalledVersion(t *testing.T) {
	mgr := &mockManager{
		installedVersions: map[string]bool{
			"1.23.0": true,
			"1.22.0": true,
		},
	}

	tests := []struct {
		name    string
		version string
		wantErr bool
	}{
		{
			name:    "installed version",
			version: "1.23.0",
			wantErr: false,
		},
		{
			name:    "not installed version",
			version: "1.21.0",
			wantErr: true,
		},
		{
			name:    "system version always ok",
			version: "system",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireInstalledVersion(mgr, tt.version)
			assert.Equal(t, tt.wantErr, (err != nil), "RequireInstalledVersion() error = , wantErr")
		})
	}
}

func TestMustGetVersion(t *testing.T) {
	tests := []struct {
		name        string
		args        []string
		wantVersion string
		wantErr     bool
	}{
		{
			name:        "valid version",
			args:        []string{"1.23.0"},
			wantVersion: "1.23.0",
			wantErr:     false,
		},
		{
			name:    "no args",
			args:    []string{},
			wantErr: true,
		},
		{
			name:    "too many args",
			args:    []string{"1.23.0", "1.22.0"},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			version, err := MustGetVersion(tt.args)
			assert.Equal(t, tt.wantErr, (err != nil), "MustGetVersion() error = , wantErr")
			assert.False(t, !tt.wantErr && version != tt.wantVersion, "MustGetVersion() version =")
		})
	}
}
