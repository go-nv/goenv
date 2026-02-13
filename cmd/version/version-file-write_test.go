package version

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

func TestVersionFileWriteCommand(t *testing.T) {
	var err error
	tests := []struct {
		name             string
		args             []string
		setupVersions    []string
		setupSystemGo    bool
		existingFile     string
		existingContent  string
		expectedOutput   string
		expectedError    string
		checkFileContent string
		checkFileRemoved bool
	}{
		{
			name:          "fails when fewer than 2 arguments specified (no args)",
			args:          []string{},
			expectedError: "requires at least 2 arg(s)",
		},
		{
			name:          "fails when fewer than 2 arguments specified (one arg)",
			args:          []string{"one"},
			expectedError: "requires at least 2 arg(s)",
		},
		{
			name:          "fails when version is non-existent",
			args:          []string{".go-version", "1.11.1"},
			expectedError: "goenv: version '1.11.1' not installed",
		},
		{
			name:             "writes version to file when version exists",
			args:             []string{"my-version", "1.11.1"},
			setupVersions:    []string{"1.11.1"},
			checkFileContent: "1.11.1\n",
		},
		{
			name:             "writes multiple versions to file",
			args:             []string{"my-version", "1.11.1", "1.10.3"},
			setupVersions:    []string{"1.11.1", "1.10.3"},
			checkFileContent: "1.11.1\n1.10.3\n",
		},
		{
			name:             "overwrites existing file content",
			args:             []string{"my-version", "1.11.1"},
			setupVersions:    []string{"1.11.1"},
			existingFile:     "my-version",
			existingContent:  "old-version\n",
			checkFileContent: "1.11.1\n",
		},
		{
			name:             "removes local version when system version is given and system exists",
			args:             []string{".go-version", "system"},
			setupSystemGo:    true,
			existingFile:     ".go-version",
			existingContent:  "1.2.3\n",
			expectedOutput:   "goenv: using system version instead of 1.2.3 now\n",
			checkFileRemoved: true,
		},
		{
			name:            "fails to set system when system Go not found in PATH",
			args:            []string{".go-version", "system"},
			setupSystemGo:   false,
			existingFile:    ".go-version",
			existingContent: "1.2.3\n",
			expectedError:   "goenv: system version not found in PATH",
		},
		{
			name:             "removes file when system version given and file has multi-line versions",
			args:             []string{".go-version", "system"},
			setupSystemGo:    true,
			existingFile:     ".go-version",
			existingContent:  "1.11.1\n1.10.3\n",
			expectedOutput:   "goenv: using system version instead of 1.11.1:1.10.3 now\n",
			checkFileRemoved: true,
		},
		{
			name:             "removes file silently when system version given and no existing file",
			args:             []string{".go-version", "system"},
			setupSystemGo:    true,
			checkFileRemoved: true,
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

			// Setup system Go if needed, or explicitly remove it
			originalPath := os.Getenv(utils.EnvVarPath)
			defer os.Setenv(utils.EnvVarPath, originalPath)

			if tt.setupSystemGo {
				// Create a bin directory in PATH with go executable
				binDir := filepath.Join(tmpDir, "system-bin")
				err = utils.EnsureDirWithContext(binDir, "create test directory")
				require.NoError(t, err, "Failed to create bin directory")
				goExec := filepath.Join(binDir, "go")
				var content string
				if utils.IsWindows() {
					goExec += ".bat"
					content = "@echo off\necho go version go1.21.0 windows/amd64\n"
				} else {
					content = "#!/bin/sh\necho go version go1.21.0 linux/amd64\n"
				}

				testutil.WriteTestFile(t, goExec, []byte(content), utils.PermFileExecutable)
				// Add to PATH
				pathSep := string(os.PathListSeparator)
				os.Setenv(utils.EnvVarPath, binDir+pathSep+originalPath)
			} else {
				// Explicitly set PATH to an empty/non-existent directory to ensure no system Go is found
				emptyDir := filepath.Join(tmpDir, "empty-bin")
				_ = utils.EnsureDirWithContext(emptyDir, "create test directory")
				os.Setenv(utils.EnvVarPath, emptyDir)
			}

			// Create existing file if specified
			var testFilePath string
			if tt.existingFile != "" {
				testFilePath = filepath.Join(tmpDir, tt.existingFile)
				testutil.WriteTestFile(t, testFilePath, []byte(tt.existingContent), utils.PermFileDefault)
			}

			// Prepare args with full paths
			args := make([]string, len(tt.args))
			for i, arg := range tt.args {
				if i == 0 {
					// First arg is the filename
					args[i] = filepath.Join(tmpDir, arg)
					if testFilePath == "" {
						testFilePath = args[i]
					}
				} else {
					// Rest are version arguments
					args[i] = arg
				}
			}

			// Execute command
			cmd := &cobra.Command{
				Use: "version-file-write",
				RunE: func(cmd *cobra.Command, cmdArgs []string) error {
					return runVersionFileWrite(cmd, cmdArgs)
				},
				Args:         cobra.MinimumNArgs(2),
				SilenceUsage: true,
			}

			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetArgs(args)

			err = cmd.Execute()

			// Check error
			if tt.expectedError != "" {
				if err == nil {
					t.Errorf("Expected error containing %q, got nil", tt.expectedError)
				} else if !strings.Contains(err.Error(), tt.expectedError) {
					t.Errorf("Expected error containing %q, got %q", tt.expectedError, err.Error())
				}
				return
			}

			require.NoError(t, err)

			// Check output
			got := output.String()
			assert.False(t, tt.expectedOutput != "" && got != tt.expectedOutput, "Expected output")

			// Check file content
			if tt.checkFileContent != "" {
				content, err := os.ReadFile(testFilePath)
				if err != nil {
					t.Errorf("Failed to read file: %v", err)
				} else if string(content) != tt.checkFileContent {
					t.Errorf("Expected file content %q, got %q", tt.checkFileContent, string(content))
				}
			}

			// Check file removal
			if tt.checkFileRemoved {
				if utils.PathExists(testFilePath) {
					t.Errorf("Expected file to be removed, but it still exists")
				}
			}
		})
	}
}
