package cmd

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
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
			expectedError: "Usage: goenv alias [name] [version]",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup test versions
			for _, version := range tt.setupVersions {
				createTestVersion(t, testRoot, version)
			}

			// Setup test aliases
			if len(tt.setupAliases) > 0 {
				aliasesFile := filepath.Join(testRoot, "aliases")
				var content strings.Builder
				content.WriteString("# goenv aliases\n")
				content.WriteString("# Format: alias_name=target_version\n")
				for name, version := range tt.setupAliases {
					content.WriteString(name + "=" + version + "\n")
				}
				err := os.WriteFile(aliasesFile, []byte(content.String()), 0644)
				if err != nil {
					t.Fatalf("Failed to setup aliases: %v", err)
				}
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

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			if tt.expectedOutput != "" {
				got := strings.TrimSpace(stdout.String())
				if got != tt.expectedOutput {
					t.Errorf("Expected output '%s', got '%s'", tt.expectedOutput, got)
				}
			}

			// For set operations, verify the file was written correctly
			if len(tt.args) == 2 && tt.expectedError == "" {
				aliasesFile := filepath.Join(testRoot, "aliases")
				content, err := os.ReadFile(aliasesFile)
				if err != nil {
					t.Errorf("Failed to read aliases file: %v", err)
					return
				}

				// Check that alias was written
				expectedLine := tt.args[0] + "=" + tt.args[1]
				if !strings.Contains(string(content), expectedLine) {
					t.Errorf("Expected aliases file to contain '%s', got:\n%s",
						expectedLine, string(content))
				}
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
			expectedError: "Usage: goenv unalias <name>",
		},
		{
			name:          "error on too many arguments",
			args:          []string{"stable", "extra"},
			expectedError: "Usage: goenv unalias <name>",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup test versions
			for _, version := range tt.setupVersions {
				createTestVersion(t, testRoot, version)
			}

			// Setup test aliases
			if len(tt.setupAliases) > 0 {
				aliasesFile := filepath.Join(testRoot, "aliases")
				var content strings.Builder
				content.WriteString("# goenv aliases\n")
				content.WriteString("# Format: alias_name=target_version\n")
				for name, version := range tt.setupAliases {
					content.WriteString(name + "=" + version + "\n")
				}
				err := os.WriteFile(aliasesFile, []byte(content.String()), 0644)
				if err != nil {
					t.Fatalf("Failed to setup aliases: %v", err)
				}
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

			if err != nil {
				t.Errorf("Unexpected error: %v", err)
				return
			}

			// Verify the alias was removed from the file
			if len(tt.args) == 1 && tt.expectedError == "" {
				aliasesFile := filepath.Join(testRoot, "aliases")
				content, err := os.ReadFile(aliasesFile)
				if err != nil {
					t.Errorf("Failed to read aliases file: %v", err)
					return
				}

				// Check that alias was removed
				removedAlias := tt.args[0]
				if strings.Contains(string(content), removedAlias+"=") {
					t.Errorf("Expected alias '%s' to be removed, but it still exists in:\n%s",
						removedAlias, string(content))
				}

				// Check that other aliases are still present
				for name := range tt.setupAliases {
					if name != removedAlias {
						if !strings.Contains(string(content), name+"=") {
							t.Errorf("Expected alias '%s' to still exist", name)
						}
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
			testRoot, cleanup := setupTestEnv(t)
			defer cleanup()

			// Setup test version
			createTestVersion(t, testRoot, tt.targetVersion)

			// Setup alias
			aliasesFile := filepath.Join(testRoot, "aliases")
			content := "# goenv aliases\n# Format: alias_name=target_version\n"
			content += tt.aliasName + "=" + tt.targetVersion + "\n"
			err := os.WriteFile(aliasesFile, []byte(content), 0644)
			if err != nil {
				t.Fatalf("Failed to setup aliases: %v", err)
			}

			if tt.useGlobal {
				// Test global command with alias
				cmd := &cobra.Command{
					Use: "global",
					RunE: func(cmd *cobra.Command, args []string) error {
						return runGlobal(cmd, []string{tt.aliasName})
					},
				}

				stdout := &strings.Builder{}
				cmd.SetOut(stdout)
				cmd.SetArgs([]string{tt.aliasName})

				err = cmd.Execute()
				if err != nil {
					t.Errorf("Failed to set global with alias: %v", err)
					return
				}

				// Verify the resolved version was written
				globalFile := filepath.Join(testRoot, "version")
				content, err := os.ReadFile(globalFile)
				if err != nil {
					t.Errorf("Failed to read global version file: %v", err)
					return
				}

				got := strings.TrimSpace(string(content))
				if got != tt.expectedOutput {
					t.Errorf("Expected global version to be '%s', got '%s'", tt.expectedOutput, got)
				}
			}

			if tt.useLocal {
				// Test local command with alias
				cmd := &cobra.Command{
					Use: "local",
					RunE: func(cmd *cobra.Command, args []string) error {
						return runLocal(cmd, []string{tt.aliasName})
					},
				}

				stdout := &strings.Builder{}
				cmd.SetOut(stdout)
				cmd.SetArgs([]string{tt.aliasName})

				err = cmd.Execute()
				if err != nil {
					t.Errorf("Failed to set local with alias: %v", err)
					return
				}

				// Verify the resolved version was written
				// setupTestEnv changes to testHome directory, so local file should be there
				cwd, _ := os.Getwd()
				localFile := filepath.Join(cwd, ".go-version")

				content, err := os.ReadFile(localFile)
				if err != nil {
					t.Errorf("Failed to read local version file: %v", err)
					return
				}

				got := strings.TrimSpace(string(content))
				if got != tt.expectedOutput {
					t.Errorf("Expected local version to be '%s', got '%s'", tt.expectedOutput, got)
				}
			}
		})
	}
}
