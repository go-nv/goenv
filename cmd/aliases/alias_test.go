package aliases

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/cmd/legacy"
	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
)

func TestAliasCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		setupAliases   map[string]string
		expectedOutput string
		expectedError  string
	}{
		{
			name:           "list no aliases",
			args:           []string{},
			expectedOutput: "No aliases defined",
		},
		{
			name:          "list existing aliases",
			args:          []string{},
			setupVersions: []string{"1.23.0", "1.24rc1"},
			setupAliases: map[string]string{
				"stable": "1.23.0",
				"dev":    "1.24rc1",
			},
			expectedOutput: "dev -> 1.24rc1\nstable -> 1.23.0",
		},
		{
			name:          "show specific alias",
			args:          []string{"stable"},
			setupVersions: []string{"1.23.0"},
			setupAliases: map[string]string{
				"stable": "1.23.0",
			},
			expectedOutput: "1.23.0",
		},
		{
			name:          "show non-existent alias",
			args:          []string{"nonexistent"},
			expectedError: "alias 'nonexistent' not found",
		},
		{
			name:          "create alias",
			args:          []string{"stable", "1.23.0"},
			setupVersions: []string{"1.23.0"},
		},
		{
			name:          "create alias with latest",
			args:          []string{"current", "latest"},
			setupVersions: []string{"1.23.0"},
		},
		{
			name:          "update existing alias",
			args:          []string{"stable", "1.24.0"},
			setupVersions: []string{"1.23.0", "1.24.0"},
			setupAliases: map[string]string{
				"stable": "1.23.0",
			},
		},
		{
			name:          "error on reserved name - system",
			args:          []string{"system", "1.23.0"},
			setupVersions: []string{"1.23.0"},
			expectedError: "alias name 'system' is reserved",
		},
		{
			name:          "error on reserved name - latest",
			args:          []string{"latest", "1.23.0"},
			setupVersions: []string{"1.23.0"},
			expectedError: "alias name 'latest' is reserved",
		},
		{
			name:          "error on invalid characters",
			args:          []string{"my alias", "1.23.0"},
			setupVersions: []string{"1.23.0"},
			expectedError: "invalid characters",
		},
		{
			name:          "error on too many arguments",
			args:          []string{"stable", "1.23.0", "extra"},
			expectedError: "usage: goenv alias [name] [version]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateMockGoVersion(t, tmpDir, version)
			}

			// Setup test aliases
			if len(tt.setupAliases) > 0 {
				aliasesFile := filepath.Join(tmpDir, "aliases")
				var content strings.Builder
				content.WriteString("# goenv aliases\n")
				content.WriteString("# Format: alias_name=target_version\n")
				for name, version := range tt.setupAliases {
					content.WriteString(name + "=" + version + "\n")
				}
				testutil.WriteTestFile(t, aliasesFile, []byte(content.String()), utils.PermFileDefault, "Failed to setup aliases")
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "alias",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runAlias(cmd, args)
				},
			}

			// Capture output
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			// Verify results
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			assert.NoError(t, err)

			if tt.expectedOutput != "" {
				got := strings.TrimSpace(stdout.String())
				assert.Equal(t, tt.expectedOutput, got)
			}

			// For set operations, verify the file was written correctly
			if len(tt.args) == 2 && tt.expectedError == "" {
				aliasesFile := filepath.Join(tmpDir, "aliases")
				content, err := os.ReadFile(aliasesFile)
				assert.NoError(t, err, "Failed to read aliases file")

				// Check that alias was written
				expectedLine := tt.args[0] + "=" + tt.args[1]
				assert.Contains(t, string(content), expectedLine)
			}
		})
	}
}

func TestUnaliasCommand(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		setupVersions []string
		setupAliases  map[string]string
		expectedError string
	}{
		{
			name: "remove existing alias",
			args: []string{"stable"},
			setupAliases: map[string]string{
				"stable": "1.23.0",
				"dev":    "1.24rc1",
			},
		},
		{
			name:          "error on non-existent alias",
			args:          []string{"nonexistent"},
			expectedError: "alias 'nonexistent' not found",
		},
		{
			name:          "error on no arguments",
			args:          []string{},
			expectedError: "usage: goenv unalias <name>",
		},
		{
			name:          "error on too many arguments",
			args:          []string{"stable", "extra"},
			expectedError: "usage: goenv unalias <name>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateMockGoVersion(t, tmpDir, version)
			}

			// Setup test aliases
			if len(tt.setupAliases) > 0 {
				aliasesFile := filepath.Join(tmpDir, "aliases")
				var content strings.Builder
				content.WriteString("# goenv aliases\n")
				content.WriteString("# Format: alias_name=target_version\n")
				for name, version := range tt.setupAliases {
					content.WriteString(name + "=" + version + "\n")
				}
				testutil.WriteTestFile(t, aliasesFile, []byte(content.String()), utils.PermFileDefault, "Failed to setup aliases")
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use: "unalias",
				RunE: func(cmd *cobra.Command, args []string) error {
					return runUnalias(cmd, args)
				},
			}

			// Capture output
			stdout := &strings.Builder{}
			stderr := &strings.Builder{}
			cmd.SetOut(stdout)
			cmd.SetErr(stderr)
			cmd.SetArgs(tt.args)

			err := cmd.Execute()

			// Verify results
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.expectedError, err.Error())
				}
				return
			}

			assert.NoError(t, err)

			// Verify the alias was removed from the file
			if len(tt.args) == 1 && tt.expectedError == "" {
				aliasesFile := filepath.Join(tmpDir, "aliases")
				content, err := os.ReadFile(aliasesFile)
				assert.NoError(t, err, "Failed to read aliases file")

				// Check that alias was removed
				removedAlias := tt.args[0]
				assert.NotContains(t, string(content), removedAlias+"=")

				// Check that other aliases are still present
				for name := range tt.setupAliases {
					if name != removedAlias {
						assert.Contains(t, string(content), name+"=")
					}
				}
			}
		})
	}
}

func TestAliasResolution(t *testing.T) {
	tests := []struct {
		name           string
		aliasName      string
		targetVersion  string
		useGlobal      bool
		useLocal       bool
		expectedOutput string
	}{
		{
			name:           "resolve alias in global command",
			aliasName:      "stable",
			targetVersion:  "1.23.0",
			useGlobal:      true,
			expectedOutput: "1.23.0",
		},
		{
			name:           "resolve alias in local command",
			aliasName:      "dev",
			targetVersion:  "1.24rc1",
			useLocal:       true,
			expectedOutput: "1.24rc1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Setup test version
			cmdtest.CreateMockGoVersion(t, tmpDir, tt.targetVersion)

			// Setup alias
			aliasesFile := filepath.Join(tmpDir, "aliases")
			content := "# goenv aliases\n# Format: alias_name=target_version\n"
			content += tt.aliasName + "=" + tt.targetVersion + "\n"
			testutil.WriteTestFile(t, aliasesFile, []byte(content), utils.PermFileDefault, "Failed to setup aliases")

			if tt.useGlobal {
				// Test global command with alias
				cmd := &cobra.Command{
					Use: "global",
					RunE: func(cmd *cobra.Command, args []string) error {
						return legacy.RunGlobal(cmd, []string{tt.aliasName})
					},
				}

				stdout := &strings.Builder{}
				cmd.SetOut(stdout)
				cmd.SetArgs([]string{tt.aliasName})

				err := cmd.Execute()
				assert.NoError(t, err, "Failed to set global with alias")

				// Verify the resolved version was written
				globalFile := filepath.Join(tmpDir, "version")
				content, err := os.ReadFile(globalFile)
				assert.NoError(t, err, "Failed to read global version file")

				got := strings.TrimSpace(string(content))
				assert.Equal(t, tt.expectedOutput, got)
			}

			if tt.useLocal {
				// Test local command with alias
				cmd := &cobra.Command{
					Use: "local",
					RunE: func(cmd *cobra.Command, args []string) error {
						return legacy.RunLocal(cmd, []string{tt.aliasName})
					},
				}

				stdout := &strings.Builder{}
				cmd.SetOut(stdout)
				cmd.SetArgs([]string{tt.aliasName})

				err := cmd.Execute()
				assert.NoError(t, err, "Failed to set local with alias")

				// Verify the resolved version was written
				// setupTestEnv changes to testHome directory, so local file should be there
				cwd, _ := os.Getwd()
				localFile := filepath.Join(cwd, ".go-version")

				content, err := os.ReadFile(localFile)
				assert.NoError(t, err, "Failed to read local version file")

				got := strings.TrimSpace(string(content))
				assert.Equal(t, tt.expectedOutput, got)
			}
		})
	}
}
