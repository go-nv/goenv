package version

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestGoModToolchainEdgeCases tests various edge cases for go.mod parsing
// to ensure compatibility with newer Go toolchain settings
func TestGoModToolchainEdgeCases(t *testing.T) {
	tests := []struct {
		name           string
		fileContent    string
		expectedOutput string
		shouldFail     bool
	}{
		{
			name: "toolchain with patch version takes precedence over go directive",
			fileContent: `module github.com/example/project

go 1.22

toolchain go1.22.5

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.22.5",
		},
		{
			name: "toolchain with rc version",
			fileContent: `module github.com/example/project

go 1.23

toolchain go1.23rc2

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.23rc2",
		},
		{
			name: "toolchain with beta version",
			fileContent: `module github.com/example/project

go 1.23

toolchain go1.23beta1

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.23beta1",
		},
		{
			name: "go directive only without toolchain",
			fileContent: `module github.com/example/project

go 1.21

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.21",
		},
		{
			name: "toolchain directive with extra whitespace",
			fileContent: `module github.com/example/project

go 1.22

toolchain   	go1.22.5

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.22.5",
		},
		{
			name: "toolchain directive with tabs",
			fileContent: `module github.com/example/project

go 1.22

toolchain	go1.22.5

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.22.5",
		},
		{
			name: "toolchain directive first, then go directive (order shouldn't matter)",
			fileContent: `module github.com/example/project

toolchain go1.22.5

go 1.22

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.22.5",
		},
		{
			name: "commented out toolchain directive should be ignored",
			fileContent: `module github.com/example/project

go 1.22

// toolchain go1.22.5

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.22",
		},
		{
			name: "go directive with minor version only",
			fileContent: `module github.com/example/project

go 1.21

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.21",
		},
		{
			name: "go directive with patch version",
			fileContent: `module github.com/example/project

go 1.21.5

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.21.5",
		},
		{
			name: "toolchain 'default' should be ignored, use go directive",
			fileContent: `module github.com/example/project

go 1.22

toolchain default

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.22",
		},
		{
			name: "very new go version (future proofing)",
			fileContent: `module github.com/example/project

go 1.99

toolchain go1.99.9

require github.com/foo/bar v1.0.0
`,
			expectedOutput: "1.99.9",
		},
		{
			name: "multiline go block format (should work)",
			fileContent: `module github.com/example/project

go 1.22
toolchain go1.22.5

require (
	github.com/foo/bar v1.0.0
)
`,
			expectedOutput: "1.22.5",
		},
		{
			name:           "carriage return line endings",
			fileContent:    "module github.com/example/project\r\n\r\ngo 1.22\r\n\r\ntoolchain go1.22.5\r\n\r\nrequire github.com/foo/bar v1.0.0\r\n",
			expectedOutput: "1.22.5",
		},
		{
			name:           "mixed line endings",
			fileContent:    "module github.com/example/project\n\rgo 1.22\r\ntoolchain go1.22.5\nrequire github.com/foo/bar v1.0.0\r\n",
			expectedOutput: "1.22.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Create go.mod file
			gomodPath := filepath.Join(tmpDir, "go.mod")
			testutil.WriteTestFile(t, gomodPath, []byte(tt.fileContent), utils.PermFileDefault)

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
			cmd.SetArgs([]string{gomodPath})

			err := cmd.Execute()

			if tt.shouldFail {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				result := strings.TrimSpace(output.String())
				assert.Equal(t, tt.expectedOutput, result)
			}
		})
	}
}

// TestGoModWithCommentsAndComplexFormat tests that comments and complex formats don't break parsing
func TestGoModWithCommentsAndComplexFormat(t *testing.T) {
	var err error
	content := `// This is a comment
module github.com/example/project

// Go version comment
go 1.22 // inline comment

// Toolchain directive comment
toolchain go1.22.5 // another inline comment

require (
	github.com/foo/bar v1.0.0 // indirect
	// Another comment
	github.com/baz/qux v2.0.0
)

replace github.com/old/pkg => github.com/new/pkg v1.0.0
`

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	gomodPath := filepath.Join(tmpDir, "go.mod")
	testutil.WriteTestFile(t, gomodPath, []byte(content), utils.PermFileDefault)

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
	cmd.SetArgs([]string{gomodPath})

	err = cmd.Execute()
	require.NoError(t, err)

	expected := "1.22.5"
	result := strings.TrimSpace(output.String())
	assert.Equal(t, expected, result)
}

// TestGoModMinimalFormat tests minimal valid go.mod files
func TestGoModMinimalFormat(t *testing.T) {
	var err error
	tests := []struct {
		name           string
		fileContent    string
		expectedOutput string
	}{
		{
			name: "absolutely minimal with toolchain",
			fileContent: `module m
go 1.22
toolchain go1.22.5`,
			expectedOutput: "1.22.5",
		},
		{
			name: "minimal with only go directive",
			fileContent: `module m
go 1.22`,
			expectedOutput: "1.22",
		},
		{
			name: "toolchain only without module (malformed but should parse version)",
			fileContent: `toolchain go1.22.5
go 1.22`,
			expectedOutput: "1.22.5",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			gomodPath := filepath.Join(tmpDir, "go.mod")
			testutil.WriteTestFile(t, gomodPath, []byte(tt.fileContent), utils.PermFileDefault)

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
			cmd.SetArgs([]string{gomodPath})

			err = cmd.Execute()
			require.NoError(t, err)

			result := strings.TrimSpace(output.String())
			assert.Equal(t, tt.expectedOutput, result)
		})
	}
}
