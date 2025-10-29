package version

import (
	"github.com/go-nv/goenv/internal/cmdtest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestVersionFileReadCommand(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		filename       string
		expectedOutput string
		expectedError  string
		noFile         bool
	}{
		{
			name:          "fails without arguments",
			noFile:        true,
			expectedError: "accepts 1 arg(s), received 0",
		},
		{
			name:          "fails for file that does not exist",
			filename:      "non-existent",
			noFile:        true,
			expectedError: "non-existent", // Both Unix ("no such file") and Windows ("cannot find file") errors contain the filename
		},
		{
			name:          "fails for file that exists but is blank",
			filename:      "my-version",
			fileContent:   "\n",
			expectedError: "no version found",
		},
		{
			name:     "reads go.mod file with go version",
			filename: "go.mod",
			fileContent: `
module github.com/go-nv/goenv

go 1.11

require (
	github.com/foo/bar v0.0.0-20220101000000-0123456789abcdef // indirect
)
`,
			expectedOutput: "1.11",
		},
		{
			name:     "reads go.mod file with toolchain specified",
			filename: "go.mod",
			fileContent: `
module github.com/go-nv/goenv

go 1.11

toolchain go1.11.4

require (
	github.com/foo/bar v0.0.0-20220101000000-0123456789abcdef // indirect
)
`,
			expectedOutput: "1.11.4",
		},
		{
			name:           "reads version file with single version",
			filename:       "my-version",
			fileContent:    "1.11.1\n",
			expectedOutput: "1.11.1",
		},
		{
			name:           "reads version file without leading and trailing spaces",
			filename:       "my-version",
			fileContent:    "         1.11.1   \n",
			expectedOutput: "1.11.1",
		},
		{
			name:     "reads version file without additional newlines",
			filename: "my-version",
			fileContent: `

1.11.1



`,
			expectedOutput: "1.11.1",
		},
		{
			name:           "reads version file that's not ending with newline",
			filename:       "my-version",
			fileContent:    "1.11.1",
			expectedOutput: "1.11.1",
		},
		{
			name:           "reads version file that ends with carriage return",
			filename:       "my-version",
			fileContent:    "1.11.1\r\n",
			expectedOutput: "1.11.1",
		},
		{
			name:     "skips relative path traversal",
			filename: "my-version",
			fileContent: `1.11.1
1.10.8
..
./*
1.9.7
`,
			expectedOutput: "1.11.1:1.10.8:1.9.7",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goenvRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			var args []string
			if !tt.noFile || tt.filename != "" {
				// Create the file
				testFile := filepath.Join(goenvRoot, tt.filename)
				if tt.fileContent != "" {
					if err := os.WriteFile(testFile, []byte(tt.fileContent), 0644); err != nil {
						t.Fatalf("Failed to create test file: %v", err)
					}
				}
				args = []string{testFile}
			}

			// Execute command
			cmd := &cobra.Command{
				Use: "version-file-read",
				RunE: func(cmd *cobra.Command, cmdArgs []string) error {
					return runVersionFileRead(cmd, cmdArgs)
				},
				Args:         cobra.ExactArgs(1),
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs(args)

			err := cmd.Execute()

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			// Check output
			got := strings.TrimSpace(output.String())
			if got != tt.expectedOutput {
				t.Errorf("Expected output %q, got %q", tt.expectedOutput, got)
			}
		})
	}
}
