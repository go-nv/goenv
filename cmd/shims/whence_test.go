package shims

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/stretchr/testify/assert"

	"github.com/spf13/cobra"
)

func TestWhenceCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupExecs     map[string][]string // version -> executables
		expectedOutput []string
		expectedError  string
		checkContains  bool
	}{
		{
			name:          "fails when no argument given",
			args:          []string{},
			expectedError: "usage: goenv whence",
		},
		{
			name: "prints versions when executable exists in multiple versions",
			args: []string{"go"},
			setupExecs: map[string][]string{
				"1.6.0": {"go"},
				"1.6.1": {"go"},
			},
			expectedOutput: []string{"1.6.0", "1.6.1"},
		},
		{
			name: "prints paths when --path flag given",
			args: []string{"--path", "go"},
			setupExecs: map[string][]string{
				"1.6.0": {"go"},
				"1.6.1": {"go"},
			},
			checkContains:  true,
			expectedOutput: []string{"/versions/1.6.0/bin/go", "/versions/1.6.1/bin/go"},
		},
		{
			name: "only prints version when executable in bin folder",
			args: []string{"go"},
			setupExecs: map[string][]string{
				"1.6.1": {"go"},
			},
			expectedOutput: []string{"1.6.1"},
		},
		{
			name:          "returns nothing when no executable found",
			args:          []string{"go"},
			expectedError: "no versions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Reset global flags
			whenceFlags.path = false
			whenceFlags.complete = false

			// Setup executables
			for version, execs := range tt.setupExecs {
				for _, exec := range execs {
					cmdtest.CreateExecutable(t, testRoot, version, exec)
				}
			}

			// Create and execute command
			cmd := &cobra.Command{
				Use:  "whence",
				Args: cobra.MaximumNArgs(2),
				RunE: func(cmd *cobra.Command, args []string) error {
					return runWhence(cmd, args)
				},
			}

			// Add flags
			cmd.Flags().BoolVar(&whenceFlags.path, "path", false, "Show full paths")
			cmd.Flags().BoolVar(&whenceFlags.complete, "complete", false, "Show completions")

			output := &strings.Builder{}
			errOutput := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetErr(errOutput)
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

			assert.NoError(t, err, "Unexpected error: \\nStderr")

			got := strings.TrimSpace(output.String())

			if len(tt.expectedOutput) == 0 {
				assert.Empty(t, got, "Expected empty output")
				return
			}

			gotLines := strings.Split(got, "\n")

			assert.Len(t, gotLines, len(tt.expectedOutput), "Expected lines %v %v", strings.Join(tt.expectedOutput, "\n"), got)

			for i, expected := range tt.expectedOutput {
				if tt.checkContains {
					// Normalize paths for cross-platform comparison
					normalizedGot := filepath.ToSlash(gotLines[i])
					normalizedExpected := filepath.ToSlash(expected)
					assert.Contains(t, normalizedGot, normalizedExpected)
				} else {
					assert.Equal(t, expected, gotLines[i])
				}
			}
		})
	}
}

func TestWhenceCompletion(t *testing.T) {
	_, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Reset global flags
	whenceFlags.path = false
	whenceFlags.complete = false

	// Create and execute command
	cmd := &cobra.Command{
		Use: "whence",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runWhence(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&whenceFlags.path, "path", false, "Show full paths")
	cmd.Flags().BoolVar(&whenceFlags.complete, "complete", false, "Show completions")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{"--complete"})

	err := cmd.Execute()
	assert.NoError(t, err)

	got := strings.TrimSpace(output.String())
	expected := "--path"

	assert.Equal(t, expected, got)
}
