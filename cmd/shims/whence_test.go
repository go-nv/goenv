package shims

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"path/filepath"
	"strings"
	"testing"

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

			if err != nil {
				t.Errorf("Unexpected error: %v\nStderr: %s", err, errOutput.String())
				return
			}

			got := strings.TrimSpace(output.String())

			if len(tt.expectedOutput) == 0 {
				if got != "" {
					t.Errorf("Expected empty output, got '%s'", got)
				}
				return
			}

			gotLines := strings.Split(got, "\n")

			if len(gotLines) != len(tt.expectedOutput) {
				t.Errorf("Expected %d lines, got %d:\nExpected:\n%s\nGot:\n%s",
					len(tt.expectedOutput), len(gotLines),
					strings.Join(tt.expectedOutput, "\n"), got)
				return
			}

			for i, expected := range tt.expectedOutput {
				if tt.checkContains {
					// Normalize paths for cross-platform comparison
					normalizedGot := filepath.ToSlash(gotLines[i])
					normalizedExpected := filepath.ToSlash(expected)
					if !strings.Contains(normalizedGot, normalizedExpected) {
						t.Errorf("Line %d: expected to contain '%s', got '%s'", i, expected, gotLines[i])
					}
				} else {
					if gotLines[i] != expected {
						t.Errorf("Line %d: expected '%s', got '%s'", i, expected, gotLines[i])
					}
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
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
		return
	}

	got := strings.TrimSpace(output.String())
	expected := "--path"

	if got != expected {
		t.Errorf("Expected '%s', got '%s'", expected, got)
	}
}
