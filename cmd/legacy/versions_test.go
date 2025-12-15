package legacy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/spf13/cobra"
)

func TestVersionsCommand(t *testing.T) {
	tests := []struct {
		name           string
		args           []string
		setupVersions  []string
		setupAliases   map[string]string // alias name -> target version
		globalVersion  string
		expectSystemGo bool
		expectedOutput []string
		expectedError  string
	}{
		{
			name:           "list installed versions with current marked",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2", "1.23.0"},
			globalVersion:  "1.22.2",
			expectedOutput: []string{"  1.21.5", "* 1.22.2 (set by global)", "  1.23.0 (latest installed)"},
		},
		{
			name:           "list with system version as current",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			expectSystemGo: true,
			expectedOutput: []string{"* system", "  1.21.5", "  1.22.2 (latest installed)"},
		},
		{
			name:           "list with system explicitly set in global file",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			globalVersion:  "system",
			expectSystemGo: true,
			expectedOutput: []string{"* system (set by global)", "  1.21.5", "  1.22.2 (latest installed)"},
		},
		{
			name:           "bare output without indicators",
			args:           []string{"--bare"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: []string{"1.21.5", "1.22.2"},
		},
		{
			name:           "list with aliases displayed",
			args:           []string{},
			setupVersions:  []string{"1.21.5", "1.22.2", "1.23.0"},
			setupAliases:   map[string]string{"stable": "1.22.2", "dev": "1.23.0"},
			globalVersion:  "1.22.2",
			expectedOutput: []string{"  1.21.5", "* 1.22.2 (set by global)", "  1.23.0 (latest installed)", "", "Aliases:", "  dev -> 1.23.0", "* stable -> 1.22.2"},
		},
		{
			name:           "skip aliases flag hides aliases",
			args:           []string{"--skip-aliases"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			setupAliases:   map[string]string{"stable": "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: []string{"* 1.21.5 (set by global)", "  1.22.2 (latest installed)"},
		},
		{
			name:           "bare mode hides aliases",
			args:           []string{"--bare"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			setupAliases:   map[string]string{"stable": "1.22.2"},
			globalVersion:  "1.21.5",
			expectedOutput: []string{"1.21.5", "1.22.2"},
		},
		{
			name:           "bare and skip aliases combined",
			args:           []string{"--bare", "--skip-aliases"},
			setupVersions:  []string{"1.21.5", "1.22.2"},
			expectedOutput: []string{"1.21.5", "1.22.2"},
		},
		{
			name:          "error with invalid arguments",
			args:          []string{"invalid", "args"},
			expectedError: "usage:",
		},
		{
			name:           "completion support",
			args:           []string{"--complete"},
			expectedOutput: []string{"--bare", "--skip-aliases", "--used"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			testRoot, cleanup := cmdtest.SetupTestEnv(t)
			defer cleanup()

			// Setup test versions
			for _, version := range tt.setupVersions {
				cmdtest.CreateTestVersion(t, testRoot, version)
			}

			// Setup test aliases if specified
			for name, target := range tt.setupAliases {
				cmdtest.CreateTestAlias(t, testRoot, name, target)
			}

			// Set global version if specified
			if tt.globalVersion != "" {
				globalFile := filepath.Join(testRoot, "version")
				testutil.WriteTestFile(t, globalFile, []byte(tt.globalVersion), utils.PermFileDefault, "Failed to set global version")
			}

			// Setup system go if needed
			if tt.expectSystemGo {
				// Create a mock system go in PATH
				systemBinDir := filepath.Join(testRoot, "system_bin")
				_ = utils.EnsureDirWithContext(systemBinDir, "create test directory")
				systemGo := filepath.Join(systemBinDir, "go")
				var content string
				if utils.IsWindows() {
					systemGo += ".bat"
					content = "@echo off\necho go version go1.20.1 windows/amd64\n"
				} else {
					content = "#!/bin/sh\necho go version go1.20.1 linux/amd64\n"
				}

				testutil.WriteTestFile(t, systemGo, []byte(content), utils.PermFileExecutable, "Failed to create system go")

				// Add to PATH temporarily
				oldPath := os.Getenv(utils.EnvVarPath)
				pathSep := string(os.PathListSeparator)
				os.Setenv(utils.EnvVarPath, systemBinDir+pathSep+oldPath)
				defer os.Setenv(utils.EnvVarPath, oldPath)
			}

			// Reset global flags before each test
			VersionsFlags.Bare = false
			VersionsFlags.SkipAliases = false
			VersionsFlags.Complete = false

			// Create and execute command
			cmd := &cobra.Command{
				Use: "versions",
				RunE: func(cmd *cobra.Command, args []string) error {
					return RunVersions(cmd, args)
				},
			}

			// Add flags bound to global struct (same as real command)
			cmd.Flags().BoolVar(&VersionsFlags.Bare, "bare", false, "Display bare version numbers only")
			cmd.Flags().BoolVar(&VersionsFlags.SkipAliases, "skip-aliases", false, "Skip aliases")
			cmd.Flags().BoolVar(&VersionsFlags.Complete, "complete", false, "Internal flag for shell completions")
			_ = cmd.Flags().MarkHidden("complete")

			// Capture output
			output := &strings.Builder{}
			cmd.SetOut(output)
			cmd.SetErr(output)
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

			if len(tt.expectedOutput) > 0 {
				got := cmdtest.StripDeprecationWarning(output.String())
				got = strings.TrimRight(got, "\n")
				gotLines := strings.Split(got, "\n")

				// Adjust expected output if system Go is present (but not expected in test setup)
				expectedLines := tt.expectedOutput
				if !tt.expectSystemGo && hasSystemGoInTest() && !VersionsFlags.Bare && !VersionsFlags.Complete {
					// System Go exists on this machine but test didn't explicitly expect it
					// Insert "  system" at the beginning of expected output
					// (but not in completion mode or bare mode)
					expectedLines = append([]string{"  system"}, expectedLines...)
				}

				assert.Len(t, gotLines, len(expectedLines), "Expected lines %v %v", strings.Join(expectedLines, "\n"), got)

				for i, expectedLine := range expectedLines {
					if i >= len(gotLines) {
						t.Errorf("Missing expected line %d: '%s'", i, expectedLine)
						continue
					}
					assert.Equal(t, expectedLine, gotLines[i])
				}
			}
		})
	}
}

func TestVersionsWithLocalVersion(t *testing.T) {
	var err error
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Reset global flags
	VersionsFlags.Bare = false
	VersionsFlags.SkipAliases = false
	VersionsFlags.Complete = false

	// Setup versions
	cmdtest.CreateTestVersion(t, testRoot, "1.21.5")
	cmdtest.CreateTestVersion(t, testRoot, "1.22.2")

	// Set global version
	globalFile := filepath.Join(testRoot, "version")
	testutil.WriteTestFile(t, globalFile, []byte("1.21.5"), utils.PermFileDefault, "Failed to set global version")

	// Create local version file in current directory
	tempDir, err := os.MkdirTemp("", "goenv_local_test_")
	require.NoError(t, err, "Failed to create temp directory")
	defer os.RemoveAll(tempDir)

	localFile := filepath.Join(tempDir, ".go-version")
	testutil.WriteTestFile(t, localFile, []byte("1.22.2"), utils.PermFileDefault)

	// Change to the directory with local version
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	os.Chdir(tempDir)

	// Create and execute command
	cmd := &cobra.Command{
		Use: "versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunVersions(cmd, args)
		},
	}

	// Add flags bound to global struct
	cmd.Flags().BoolVar(&VersionsFlags.Bare, "bare", false, "Display bare version numbers only")
	cmd.Flags().BoolVar(&VersionsFlags.SkipAliases, "skip-aliases", false, "Skip aliases")
	cmd.Flags().BoolVar(&VersionsFlags.Complete, "complete", false, "Internal flag for shell completions")
	_ = cmd.Flags().MarkHidden("complete")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	assert.NoError(t, err, "Versions command failed")

	got := cmdtest.StripDeprecationWarning(output.String())
	got = strings.TrimRight(got, "\n")
	gotLines := strings.Split(got, "\n")

	// Should show local version (1.22.2) as current with source, not global (1.21.5)
	// The source path will be the tempDir/.go-version
	expectedLines := 2
	if hasSystemGoInTest() {
		// System Go adds an extra line at the beginning
		expectedLines = 3
	}

	assert.Len(t, gotLines, expectedLines, "Expected lines %v", got)

	// Adjust line indices if system Go is present
	offset := 0
	if hasSystemGoInTest() {
		// System Go appears first, so our versions are offset by 1
		offset = 1
		// Verify system line is present
		assert.Contains(t, gotLines[0], "system", "Line 0: expected system line %v", gotLines)
	}

	// Check line: non-current version
	assert.Equal(t, "  1.21.5", gotLines[0+offset], "Line : expected ' 1.21.5'")

	// Check line: current version with suffix (also the latest version)
	expectedPrefix := "* 1.22.2 (set by "
	expectedContains := ".go-version)"
	expectedSuffix := "[latest]" // latest version tag
	normalizedLine := filepath.ToSlash(gotLines[1+offset])
	assert.False(t, !strings.HasPrefix(normalizedLine, expectedPrefix) || !strings.Contains(normalizedLine, expectedContains) || !strings.HasSuffix(normalizedLine, expectedSuffix), "Line : expected to match '* 1.22.2 (set by .../.go-version) [latest]'")
}

func TestVersionsNoVersionsInstalled(t *testing.T) {
	// Skip this test if system Go is available
	// This test specifically checks the error case when NO Go is available at all
	if hasSystemGoInTest() {
		t.Skip("System Go is available, cannot test 'no Go at all' scenario")
	}

	_, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Reset global flags
	VersionsFlags.Bare = false
	VersionsFlags.SkipAliases = false
	VersionsFlags.Complete = false

	// Don't create any versions, no system go either

	// Create and execute command
	cmd := &cobra.Command{
		Use: "versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunVersions(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&VersionsFlags.Bare, "bare", false, "Display bare version numbers only")
	cmd.Flags().BoolVar(&VersionsFlags.SkipAliases, "skip-aliases", false, "Skip aliases")
	cmd.Flags().BoolVar(&VersionsFlags.Complete, "complete", false, "Internal flag for shell completions")
	_ = cmd.Flags().MarkHidden("complete")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()

	// Should fail with warning when no versions installed and no system go
	assert.Error(t, err, "Expected error when no versions installed and no system go")

	assert.Contains(t, err.Error(), "Warning: no Go detected", "Expected 'Warning: no Go detected' error %v", err)
}

func TestVersionsSystemGoOnly(t *testing.T) {
	testRoot, cleanup := cmdtest.SetupTestEnv(t)
	defer cleanup()

	// Reset global flags
	VersionsFlags.Bare = false
	VersionsFlags.SkipAliases = false
	VersionsFlags.Complete = false

	// Create system go but no installed versions
	systemBinDir := filepath.Join(testRoot, "system_bin")
	_ = utils.EnsureDirWithContext(systemBinDir, "create test directory")
	systemGo := filepath.Join(systemBinDir, "go")
	var content string
	if utils.IsWindows() {
		systemGo += ".bat"
		content = "@echo off\necho go version go1.20.1 windows/amd64\n"
	} else {
		content = "#!/bin/sh\necho go version go1.20.1 linux/amd64\n"
	}

	testutil.WriteTestFile(t, systemGo, []byte(content), utils.PermFileExecutable, "Failed to create system go")

	// Add to PATH temporarily
	oldPath := os.Getenv(utils.EnvVarPath)
	pathSep := string(os.PathListSeparator)
	os.Setenv(utils.EnvVarPath, systemBinDir+pathSep+oldPath)
	defer os.Setenv(utils.EnvVarPath, oldPath)

	// Create and execute command
	cmd := &cobra.Command{
		Use: "versions",
		RunE: func(cmd *cobra.Command, args []string) error {
			return RunVersions(cmd, args)
		},
	}

	// Add flags
	cmd.Flags().BoolVar(&VersionsFlags.Bare, "bare", false, "Display bare version numbers only")
	cmd.Flags().BoolVar(&VersionsFlags.SkipAliases, "skip-aliases", false, "Skip aliases")
	cmd.Flags().BoolVar(&VersionsFlags.Complete, "complete", false, "Internal flag for shell completions")
	_ = cmd.Flags().MarkHidden("complete")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetArgs([]string{})

	err := cmd.Execute()
	assert.NoError(t, err)

	got := cmdtest.StripDeprecationWarning(output.String())
	expected := "* system"

	assert.Equal(t, expected, got)
}

// hasSystemGoInTest checks if system Go is available during test execution
// This is needed because CI/macOS systems may have Go installed in PATH
func hasSystemGoInTest() bool {
	cfg := config.Load()
	mgr := manager.NewManager(cfg)
	return mgr.HasSystemGo()
}

func TestVersionsUsedFlag(t *testing.T) {
	var err error
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create temporary directory structure with test projects
	tmpDir := t.TempDir()

	// Project 1: .go-version
	proj1 := filepath.Join(tmpDir, "proj1")
	err = utils.EnsureDirWithContext(proj1, "create test directory")
	require.NoError(t, err)
	testutil.WriteTestFile(t, filepath.Join(proj1, ".go-version"), []byte("1.21.5\n"), utils.PermFileDefault)

	// Project 2: go.mod
	proj2 := filepath.Join(tmpDir, "proj2")
	err = utils.EnsureDirWithContext(proj2, "create test directory")
	require.NoError(t, err)
	gomodContent := "module test\n\ngo 1.22.3\n"
	testutil.WriteTestFile(t, filepath.Join(proj2, "go.mod"), []byte(gomodContent), utils.PermFileDefault)

	// Setup test environment with installed versions
	cfg := &config.Config{
		Root: t.TempDir(),
	}

	// Create mock installed versions
	for _, ver := range []string{"1.21.5", "1.22.3", "1.23.2"} {
		cmdtest.CreateMockGoVersion(t, cfg.Root, ver)
	}

	// Set environment to use test root
	t.Setenv(utils.GoenvEnvVarRoot.String(), cfg.Root)

	// Change to tmp directory for scanning
	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Create a new command instance to avoid state issues
	cmd := &cobra.Command{
		Use:  "versions",
		RunE: RunVersions,
	}

	// Add flags
	cmd.Flags().BoolVar(&VersionsFlags.Used, "used", false, "Scan projects")
	cmd.Flags().IntVar(&VersionsFlags.Depth, "depth", 3, "Scan depth")

	// Set flags
	cmd.Flags().Set("used", "true")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)
	cmd.SetArgs([]string{})

	err = cmd.Execute()
	require.NoError(t, err, "Unexpected error: \\nOutput")

	result := output.String()

	// Strip deprecation warning for easier testing
	result = cmdtest.StripDeprecationWarning(result)

	// Verify output contains expected elements
	assert.Contains(t, result, "Scanning for Go projects", "Expected scanning message in output. Got:\\n %v", result)

	assert.Contains(t, result, "Installed versions:", "Expected 'Installed versions:' section. Got:\\n %v", result)

	assert.Contains(t, result, "Projects found:", "Expected 'Projects found:' section. Got:\\n %v", result)

	assert.Contains(t, result, "proj1", "Expected proj1 in output. Got:\\n %v", result)

	assert.Contains(t, result, "proj2", "Expected proj2 in output. Got:\\n %v", result)

	// Check for tips section
	assert.Contains(t, result, "Tips:", "Expected tips section in output. Got:\\n %v", result)
}

func TestVersionsUsedDepthFlag(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create nested directory structure
	tmpDir := t.TempDir()

	// Project at depth 1
	proj1 := filepath.Join(tmpDir, "proj1")
	_ = utils.EnsureDirWithContext(proj1, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(proj1, ".go-version"), []byte("1.21.5\n"), utils.PermFileDefault)

	// Project at depth 3
	proj2 := filepath.Join(tmpDir, "a", "b", "proj2")
	_ = utils.EnsureDirWithContext(proj2, "create test directory")
	testutil.WriteTestFile(t, filepath.Join(proj2, ".go-version"), []byte("1.22.3\n"), utils.PermFileDefault)

	// Setup mock environment
	cfg := &config.Config{
		Root: t.TempDir(),
	}

	for _, ver := range []string{"1.21.5", "1.22.3"} {
		cmdtest.CreateMockGoVersion(t, cfg.Root, ver)
	}

	t.Setenv(utils.GoenvEnvVarRoot.String(), cfg.Root)

	oldWd, _ := os.Getwd()
	defer os.Chdir(oldWd)
	os.Chdir(tmpDir)

	// Test with depth 1 - should only find proj1
	cmd := &cobra.Command{
		Use:  "versions",
		RunE: RunVersions,
	}

	// Add flags
	cmd.Flags().BoolVar(&VersionsFlags.Used, "used", false, "Scan projects")
	cmd.Flags().IntVar(&VersionsFlags.Depth, "depth", 3, "Scan depth")

	cmd.Flags().Set("used", "true")
	cmd.Flags().Set("depth", "1")

	output := &strings.Builder{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := cmd.Execute()
	require.NoError(t, err, "Unexpected error: \\nOutput")

	result := cmdtest.StripDeprecationWarning(output.String())

	// Should find proj1
	assert.Contains(t, result, "proj1", "Expected to find proj1 at depth 1. Got:\\n %v", result)

	// Should NOT find proj2 (it's at depth 3)
	assert.NotContains(t, result, "proj2", "Should not find proj2 at depth 3 with --depth 1. Got:\\n %v", result)
}
