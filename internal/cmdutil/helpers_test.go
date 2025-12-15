package cmdutil

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetContexts(t *testing.T) {
	// Create test config, manager, and environment
	cfg := config.Load()
	mgr := manager.NewManager(cfg)
	env := &utils.GoenvEnvironment{
		Root:  cfg.Root,
		Shell: "bash",
		Debug: "1",
	}

	// Create command with context
	cmd := &cobra.Command{Use: "test"}
	ctx := context.Background()
	ctx = config.ToContext(ctx, cfg)
	ctx = manager.ToContext(ctx, mgr)
	ctx = utils.EnvironmentToContext(ctx, env)
	cmd.SetContext(ctx)

	t.Run("get all contexts", func(t *testing.T) {
		result := GetContexts(cmd,
			config.ConfigContextKey,
			manager.ManagerContextKey,
			utils.EnvironmentContextKey,
		)

		require.NotNil(t, result)
		assert.Equal(t, cfg, result.Config)
		assert.Equal(t, mgr, result.Manager)
		assert.Equal(t, env, result.Environment)
	})

	t.Run("get only config", func(t *testing.T) {
		result := GetContexts(cmd, config.ConfigContextKey)

		require.NotNil(t, result)
		assert.Equal(t, cfg, result.Config)
		assert.Nil(t, result.Manager)
		assert.Nil(t, result.Environment)
	})

	t.Run("get config and manager", func(t *testing.T) {
		result := GetContexts(cmd,
			config.ConfigContextKey,
			manager.ManagerContextKey,
		)

		require.NotNil(t, result)
		assert.Equal(t, cfg, result.Config)
		assert.Equal(t, mgr, result.Manager)
		assert.Nil(t, result.Environment)
	})

	t.Run("get no contexts", func(t *testing.T) {
		result := GetContexts(cmd)

		require.NotNil(t, result)
		// When no keys specified, all contexts are retrieved
		assert.Equal(t, cfg, result.Config)
		assert.Equal(t, mgr, result.Manager)
		assert.Equal(t, env, result.Environment)
	})

	t.Run("fallback when context not set", func(t *testing.T) {
		// Create command WITHOUT context values set
		emptyCmd := &cobra.Command{Use: "test"}
		emptyCmd.SetContext(context.Background())

		// Should fallback to creating new instances
		result := GetContexts(emptyCmd, config.ConfigContextKey, manager.ManagerContextKey)
		require.NotNil(t, result.Config)
		require.NotNil(t, result.Manager)

		// Config and Manager should be created via fallback
		assert.NotNil(t, result.Config.Root)
		assert.NotNil(t, result.Manager)
	})
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
