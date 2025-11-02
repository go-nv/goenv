package diagnostics

import (
	"bytes"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/shellutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDoctorCommand_BasicRun(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create basic directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})
	// Error is expected since we don't have a complete setup
	// But we want to verify the command runs and produces output

	output := buf.String()
	expectedStrings := []string{
		"Checking goenv installation",
		"Diagnostic Results",
		"Summary:",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Expected in output %v %v", expected, output)
	}
}

func TestDoctorCommand_ChecksExecuted(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Verify various checks are mentioned
	checkNames := []string{
		"Runtime environment",
		"goenv binary",
		"GOENV_ROOT directory",
		"GOENV_ROOT filesystem",
		"Shell configuration",
		"Shell environment",
		"PATH configuration",
		"Shims directory",
	}

	for _, checkName := range checkNames {
		assert.Contains(t, output, checkName, "Expected check to be mentioned in output %v", checkName)
	}
}

func TestDoctorCommand_WithInstalledVersion(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create complete directory structure
	rootBinDir := filepath.Join(tmpDir, "bin")
	shimsDir := filepath.Join(tmpDir, "shims")
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(rootBinDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDirWithContext(shimsDir, "create test directory")
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	// Create a fake installed version
	versionDir := filepath.Join(versionsDir, "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create mock go binary
	goBinary := filepath.Join(binDir, "go")
	var content string
	if utils.IsWindows() {
		goBinary += ".bat"
		content = "@echo off\necho go1.21.0\n"
	} else {
		content = "#!/bin/bash\necho go1.21.0\n"
	}
	testutil.WriteTestFile(t, goBinary, []byte(content), utils.PermFileExecutable)

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention installed versions
	assert.True(t, strings.Contains(output, "Installed") || strings.Contains(output, "version"), "Expected installed versions check in output")
}

func TestDoctorCommand_MissingGOENV_ROOT(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")
	t.Setenv(utils.GoenvEnvVarRoot.String(), nonExistentDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), nonExistentDir)

	// Capture exit code
	exitCode := -1
	oldExit := doctorExit
	doctorExit = func(code int) {
		exitCode = code
	}
	defer func() { doctorExit = oldExit }()

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	err := doctorCmd.RunE(doctorCmd, []string{})

	t.Logf("Exit code: %d", exitCode)

	// Doctor should call exit(1) when GOENV_ROOT doesn't exist
	assert.Equal(t, 1, exitCode, "Expected exit code 1 when GOENV_ROOT doesn't exist")

	// Error may or may not be returned (before exit is called) - doctor now calls os.Exit
	// so we just check that output contains the error
	t.Logf("Error returned: %v", err)

	output := buf.String()

	// Should show error for GOENV_ROOT
	assert.Contains(t, output, "GOENV_ROOT", "Expected GOENV_ROOT error in output %v", output)
}

func TestDoctorCommand_OutputFormat(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Ensure emojis are enabled for this test
	t.Setenv("GOENV_PLAIN", "")
	t.Setenv(utils.EnvVarNoColor, "")

	// Create basic structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Check for expected formatting elements
	// Note: Emojis might not appear in test environment depending on terminal settings
	formatElements := []string{
		"Summary:",
		"OK", // or "ok" in summary
	}

	for _, element := range formatElements {
		assert.Contains(t, output, element, "Expected format element in output %v", element)
	}

	// Just verify that output is not empty and contains diagnostic info
	assert.NotEqual(t, 0, len(output), "Expected non-empty output")
	assert.Contains(t, output, "goenv", "Expected output to contain 'goenv'")
}

func TestDoctorCommand_WithCache(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	// Create cache file
	cacheFile := filepath.Join(tmpDir, "cache", "releases.json")
	err = utils.EnsureDirWithContext(filepath.Dir(cacheFile), "create test directory")
	require.NoError(t, err, "Failed to create cache directory")
	testutil.WriteTestFile(t, cacheFile, []byte("{}"), utils.PermFileDefault)

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention cache check
	assert.False(t, !strings.Contains(output, "Cache") || !strings.Contains(output, "cache"), "Expected cache check in output")
}

func TestDoctorCommand_ErrorCount(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")
	t.Setenv(utils.GoenvEnvVarRoot.String(), nonExistentDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), nonExistentDir)

	// Capture exit code
	exitCode := -1
	oldExit := doctorExit
	doctorExit = func(code int) {
		exitCode = code
	}
	defer func() { doctorExit = oldExit }()

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	err := doctorCmd.RunE(doctorCmd, []string{})

	t.Logf("Exit code: %d", exitCode)
	t.Logf("Error returned: %v", err)

	output := buf.String()

	// Should show summary with error count
	assert.Contains(t, output, "error", "Expected error count in summary %v", output)

	// Doctor should call exit(1) when errors are found
	assert.Equal(t, 1, exitCode, "Expected exit code 1 when errors are found")
}

func TestDoctorCommand_SuccessScenario(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Add both bin and shims to PATH
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+filepath.Join(tmpDir, "shims")+string(os.PathListSeparator)+os.Getenv(utils.EnvVarPath))

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create complete directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	shimsDir := filepath.Join(tmpDir, "shims")
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(shimsDir, "create test directory")
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	// Create a version
	versionDir := filepath.Join(versionsDir, "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	goBinary := filepath.Join(binDir, "go")
	var content string
	if utils.IsWindows() {
		goBinary += ".bat"
		content = "@echo off\necho go1.21.0\n"
	} else {
		content = "#!/bin/bash\necho go1.21.0\n"
	}
	testutil.WriteTestFile(t, goBinary, []byte(content), utils.PermFileExecutable)

	// Set current version
	versionFile := filepath.Join(tmpDir, "version")
	testutil.WriteTestFile(t, versionFile, []byte("1.21.0\n"), utils.PermFileDefault)

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// In success scenario, we should see OK indicators
	assert.True(t, strings.Contains(output, "OK") || strings.Contains(output, "✅"), "Expected success indicators in output")

	// Should show summary
	assert.Contains(t, output, "Summary:", "Expected summary in output %v", output)
}

func TestDoctorHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	err := doctorCmd.Help()
	require.NoError(t, err, "Help command failed")

	output := buf.String()
	expectedStrings := []string{
		"doctor",
		"installation",
		"configuration",
		"verifies",
	}

	for _, expected := range expectedStrings {
		assert.Contains(t, output, expected, "Help output missing %v", expected)
	}
}

func TestDoctorCommand_ShellDetection(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Set a specific shell
	originalShell := os.Getenv(utils.EnvVarShell)
	if utils.IsWindows() {
		t.Setenv(utils.EnvVarShell, "powershell")
	} else {
		t.Setenv(utils.EnvVarShell, "/bin/bash")
	}
	defer func() {
		if originalShell != "" {
			os.Setenv(utils.EnvVarShell, originalShell)
		}
	}()

	// Create basic structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention shell in output
	assert.False(t, !strings.Contains(output, "Shell") || !strings.Contains(output, "shell"), "Expected shell check in output")
}

func TestDoctorCommand_NoVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create structure but NO versions
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention installed versions (even if none)
	assert.False(t, !strings.Contains(output, "version") || !strings.Contains(output, "Version"), "Expected version check in output")
}

// Test the checkGoToolchain function
func TestCheckGoToolchain(t *testing.T) {
	tests := []struct {
		name           string
		gotoolchain    string
		expectedStatus Status
		shouldContain  string
	}{
		{
			name:           "GOTOOLCHAIN not set",
			gotoolchain:    "",
			expectedStatus: StatusOK,
			shouldContain:  "not set",
		},
		{
			name:           "GOTOOLCHAIN=auto (warning)",
			gotoolchain:    "auto",
			expectedStatus: StatusWarning,
			shouldContain:  "can cause issues",
		},
		{
			name:           "GOTOOLCHAIN=local (recommended)",
			gotoolchain:    "local",
			expectedStatus: StatusOK,
			shouldContain:  "recommended",
		},
		{
			name:           "GOTOOLCHAIN with specific version",
			gotoolchain:    "go1.23.2",
			expectedStatus: StatusWarning,
			shouldContain:  "may interfere",
		},
		{
			name:           "GOTOOLCHAIN=local+auto",
			gotoolchain:    "local+auto",
			expectedStatus: StatusWarning,
			shouldContain:  "may interfere",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set GOTOOLCHAIN environment variable
			if tt.gotoolchain != "" {
				t.Setenv(utils.EnvVarGotoolchain, tt.gotoolchain)
			} else {
				os.Unsetenv("GOTOOLCHAIN")
			}

			result := checkGoToolchain()

			assert.Equal(t, tt.expectedStatus, result.status, "Expected status")

			assert.Contains(t, result.message, tt.shouldContain, "Expected message to contain %v %v", tt.shouldContain, result.message)

			assert.Equal(t, "GOTOOLCHAIN setting", result.name, "Expected name")

			// Warnings should have advice
			assert.False(t, result.status == StatusWarning && result.advice == "", "Warning status should have advice")
		})
	}
}

// Test the checkCacheIsolationEffectiveness function
func TestCheckCacheIsolationEffectiveness(t *testing.T) {
	var err error
	tests := []struct {
		name           string
		setupVersion   string
		setupCache     bool
		setupOldCache  bool
		disableCache   bool
		expectedStatus Status
		shouldContain  string
	}{
		{
			name:           "No managed version active",
			setupVersion:   "",
			expectedStatus: StatusOK,
			shouldContain:  "Not applicable",
		},
		{
			name:           "System version active",
			setupVersion:   "system",
			expectedStatus: StatusOK,
			shouldContain:  "Not applicable",
		},
		{
			name:           "Cache isolation disabled",
			setupVersion:   "1.21.0",
			disableCache:   true,
			expectedStatus: StatusOK,
			shouldContain:  "disabled",
		},
		{
			name:           "New cache will be created",
			setupVersion:   "1.21.0",
			setupCache:     false,
			setupOldCache:  false,
			expectedStatus: StatusOK,
			shouldContain:  "will be created",
		},
		{
			name:           "Architecture-aware cache exists",
			setupVersion:   "1.21.0",
			setupCache:     true,
			setupOldCache:  false,
			expectedStatus: StatusOK,
			shouldContain:  "go-build-", // Cache path will contain go-build-
		},
		{
			name:           "Old cache exists",
			setupVersion:   "1.21.0",
			setupCache:     false,
			setupOldCache:  true,
			expectedStatus: StatusWarning,
			shouldContain:  "old-style cache",
		},
		{
			name:           "Both caches exist",
			setupVersion:   "1.21.0",
			setupCache:     true,
			setupOldCache:  true,
			expectedStatus: StatusWarning, // Test setup doesn't create cache matching expected name (with CGO hash), so only old cache is found
			shouldContain:  "old-style cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Change to tmpDir to avoid picking up .go-version files from parent directories
			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			err = os.Chdir(tmpDir)
			require.NoError(t, err, "Failed to change directory")

			// Unset any existing GOENV variables to ensure isolation
			t.Setenv(utils.GoenvEnvVarVersion.String(), "")

			t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
			t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

			// Create basic structure
			versionsDir := filepath.Join(tmpDir, "versions")
			err = utils.EnsureDirWithContext(versionsDir, "create test directory")
			require.NoError(t, err, "Failed to create versions directory")

			// Setup version if needed
			if tt.setupVersion != "" && tt.setupVersion != manager.SystemVersion {
				versionDir := filepath.Join(versionsDir, tt.setupVersion)
				binDir := filepath.Join(versionDir, "bin")
				err = utils.EnsureDirWithContext(binDir, "create test directory")
				require.NoError(t, err, "Failed to create bin directory")

				// Create version file to set current version
				versionFile := filepath.Join(tmpDir, "version")
				testutil.WriteTestFile(t, versionFile, []byte(tt.setupVersion+"\n"), utils.PermFileDefault)

				// Setup caches if requested
				if tt.setupCache {
					// Create architecture-aware cache
					goos := os.Getenv(utils.EnvVarGoos)
					goarch := os.Getenv(utils.EnvVarGoarch)
					if goos == "" {
						goos = "host"
					}
					if goarch == "" {
						goarch = "host"
					}
					cacheSuffix := "go-build-" + goos + "-" + goarch
					cacheDir := filepath.Join(versionDir, cacheSuffix)
					err = utils.EnsureDirWithContext(cacheDir, "create test directory")
					require.NoError(t, err, "Failed to create cache directory")
				}

				if tt.setupOldCache {
					// Create old-style cache
					oldCacheDir := filepath.Join(versionDir, "go-build")
					err = utils.EnsureDirWithContext(oldCacheDir, "create test directory")
					require.NoError(t, err, "Failed to create old cache directory")
				}
			} else if tt.setupVersion == manager.SystemVersion {
				// Set system version
				versionFile := filepath.Join(tmpDir, "version")
				testutil.WriteTestFile(t, versionFile, []byte("system\n"), utils.PermFileDefault)
			}
			// If setupVersion is empty, don't create a version file at all

			// Set cache isolation env var if requested
			if tt.disableCache {
				t.Setenv(utils.GoenvEnvVarDisableGocache.String(), "1")
			} else {
				t.Setenv(utils.GoenvEnvVarDisableGocache.String(), "")
			}

			// Load config which will read from environment variables
			cfg := config.Load()
			mgr := manager.NewManager(cfg)
			result := checkCacheIsolationEffectiveness(cfg, mgr)

			assert.Equal(t, tt.expectedStatus, result.status, "Expected status")

			assert.Contains(t, result.message, tt.shouldContain, "Expected message to contain %v %v", tt.shouldContain, result.message)

			assert.Equal(t, "Architecture-aware cache isolation", result.name, "Expected name")
		})
	}
}

// Test the checkRosetta function
func TestCheckRosetta(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Load config which will read from environment variables
	cfg := config.Load()

	result := checkRosetta(cfg)

	// Should always return a valid result
	assert.Equal(t, "Rosetta detection", result.name, "Expected name")

	// Status should be one of ok, warning, or error
	validStatuses := []Status{StatusOK, StatusWarning, StatusError}
	assert.True(t, slices.Contains(validStatuses, result.status), "Invalid status , expected one of")

	// On non-macOS systems, should say "Not applicable"
	if !platform.IsMacOS() {
		assert.True(t, strings.Contains(result.message, "Not applicable") || strings.Contains(result.message, "not macOS"), "Expected non-macOS message")
	}
	// On macOS, the message depends on the actual system configuration
	// We can't reliably test specific outcomes without knowing the hardware
}

func TestDoctorCommand_EnvironmentDetection(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Verify environment detection checks are present
	environmentChecks := []string{
		"Runtime environment",
		"GOENV_ROOT filesystem",
	}

	for _, check := range environmentChecks {
		assert.Contains(t, output, check, "Expected environment check to be in output %v", check)
	}

	// Should see either Native, Container, or WSL
	hasEnvironmentType := strings.Contains(output, "Native") ||
		strings.Contains(output, "Container") ||
		strings.Contains(output, "WSL")

	assert.True(t, hasEnvironmentType, "Expected to see environment type (Native, Container, or WSL) in output")

	// Should see filesystem type
	hasFilesystemType := strings.Contains(output, "Filesystem type:")

	assert.True(t, hasFilesystemType, "Expected to see 'Filesystem type:' in output")
}

func TestCheckEnvironment(t *testing.T) {
	cfg := config.Load()

	result := checkEnvironment(cfg)

	assert.Equal(t, "Runtime environment", result.name, "Expected check name 'Runtime environment'")

	assert.False(t, result.status != StatusOK && result.status != StatusWarning, "Expected status 'ok' or 'warning'")

	assert.NotEmpty(t, result.message, "Expected non-empty message")

	// Message should contain environment description
	hasEnvType := strings.Contains(result.message, "Native") ||
		strings.Contains(result.message, "Container") ||
		strings.Contains(result.message, "WSL")

	assert.True(t, hasEnvType, "Expected message to contain environment type")
}

func TestCheckGoenvRootFilesystem(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Root = tmpDir

	result := checkGoenvRootFilesystem(cfg)

	assert.Equal(t, "GOENV_ROOT filesystem", result.name, "Expected check name 'GOENV_ROOT filesystem'")

	assert.NotEmpty(t, result.message, "Expected non-empty message")

	// Message should mention filesystem type
	assert.Contains(t, result.message, "Filesystem type:", "Expected message to contain 'Filesystem type:' %v", result.message)
}

func TestCheckMacOSDeploymentTarget(t *testing.T) {
	var err error
	if !platform.IsMacOS() {
		t.Skip("macOS deployment target check only works on macOS")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Root = tmpDir

	// Create a fake version directory
	versionsDir := filepath.Join(tmpDir, "versions")
	versionDir := filepath.Join(versionsDir, "1.23.0")
	binDir := filepath.Join(versionDir, "bin")
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Set current version
	versionFile := filepath.Join(tmpDir, "version")
	testutil.WriteTestFile(t, versionFile, []byte("1.23.0\n"), utils.PermFileDefault)

	mgr := manager.NewManager(cfg)
	result := checkMacOSDeploymentTarget(cfg, mgr)

	assert.Equal(t, "macOS deployment target", result.name, "Expected check name 'macOS deployment target'")

	// Should be ok or have a message about not finding binary
	if result.status != StatusOK {
		t.Logf("Status: %s, Message: %s", result.status, result.message)
	}
}

func TestCheckWindowsCompiler(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows compiler check only works on Windows")
	}

	cfg := config.Load()
	result := checkWindowsCompiler(cfg)

	assert.Equal(t, "Windows compiler", result.name, "Expected check name 'Windows compiler'")

	assert.NotEmpty(t, result.message, "Expected non-empty message")

	t.Logf("Status: %s, Message: %s", result.status, result.message)
}

func TestCheckWindowsARM64(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows ARM64 check only works on Windows")
	}

	cfg := config.Load()
	result := checkWindowsARM64(cfg)

	assert.Equal(t, "Windows ARM64/ARM64EC", result.name, "Expected check name 'Windows ARM64/ARM64EC'")

	assert.NotEmpty(t, result.message, "Expected non-empty message")

	// Should mention process mode
	assert.Contains(t, result.message, "Process mode:", "Expected message to contain 'Process mode:' %v", result.message)

	t.Logf("Status: %s, Message: %s", result.status, result.message)
}

func TestCheckLinuxKernelVersion(t *testing.T) {
	if !platform.IsLinux() {
		t.Skip("Linux kernel check only works on Linux")
	}

	cfg := config.Load()
	result := checkLinuxKernelVersion(cfg)

	assert.Equal(t, "Linux kernel version", result.name, "Expected check name 'Linux kernel version'")

	assert.NotEmpty(t, result.message, "Expected non-empty message")

	// Should mention kernel version
	assert.Contains(t, result.message, "Kernel:", "Expected message to contain 'Kernel:' %v", result.message)

	t.Logf("Status: %s, Message: %s", result.status, result.message)
}

func TestPlatformSpecificChecksInDoctor(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv(utils.GoenvEnvVarVersion.String(), "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv(utils.EnvVarPath)
	t.Setenv(utils.EnvVarPath, filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Create directory structure
	err = utils.EnsureDir(filepath.Join(tmpDir, "bin"))
	require.NoError(t, err, "Failed to create bin directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "shims"))
	require.NoError(t, err, "Failed to create shims directory")
	err = utils.EnsureDir(filepath.Join(tmpDir, "versions"))
	require.NoError(t, err, "Failed to create versions directory")

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Check for platform-specific checks based on OS
	switch platform.OS() {
	case "darwin":
		assert.Contains(t, output, "macOS deployment target", "Expected 'macOS deployment target' check on macOS")
		assert.NotContains(t, output, "Windows compiler", "Should not have Windows checks on macOS")
		assert.NotContains(t, output, "Linux kernel", "Should not have Linux checks on macOS")

	case "windows":
		assert.Contains(t, output, "Windows compiler", "Expected 'Windows compiler' check on Windows")
		assert.Contains(t, output, "Windows ARM64/ARM64EC", "Expected 'Windows ARM64/ARM64EC' check on Windows")
		assert.NotContains(t, output, "macOS deployment target", "Should not have macOS checks on Windows")
		assert.NotContains(t, output, "Linux kernel", "Should not have Linux checks on Windows")

	case "linux":
		assert.Contains(t, output, "Linux kernel version", "Expected 'Linux kernel version' check on Linux")
		assert.NotContains(t, output, "macOS deployment target", "Should not have macOS checks on Linux")
		assert.NotContains(t, output, "Windows compiler", "Should not have Windows checks on Linux")
	}
}

// TestPlatformChecksCrossOSBehavior tests that platform-specific checks behave correctly
// when called on the "wrong" platform (e.g., Windows check on macOS should return nil/not applicable)
func TestPlatformChecksCrossOSBehavior(t *testing.T) {
	cfg := config.Load()

	// Test Windows checks on non-Windows platforms
	if !utils.IsWindows() {
		t.Run("WindowsChecksOnNonWindows", func(t *testing.T) {
			result := checkWindowsCompiler(cfg)
			assert.Equal(t, "windows-compiler", result.id, "Expected id 'windows-compiler'")
			assert.Equal(t, StatusOK, result.status, "Expected status 'ok' (not applicable)")
			assert.Contains(t, result.message, "Not applicable", "Expected 'Not applicable' message on non-Windows %v", result.message)
		})
	}

	// Test macOS checks on non-macOS platforms
	if !platform.IsMacOS() {
		t.Run("MacOSChecksOnNonMacOS", func(t *testing.T) {
			mgr := manager.NewManager(cfg)
			result := checkMacOSDeploymentTarget(cfg, mgr)
			assert.Equal(t, "macos-deployment-target", result.id, "Expected id 'macos-deployment-target'")
			// Check should handle non-macOS gracefully
			assert.NotEqual(t, StatusError, result.status, "Check should not error on non-macOS")
		})
	}

	// Test Linux checks on non-Linux platforms
	if !platform.IsLinux() {
		t.Run("LinuxChecksOnNonLinux", func(t *testing.T) {
			result := checkLinuxKernelVersion(cfg)
			assert.Equal(t, "linux-kernel-version", result.id, "Expected id 'linux-kernel-version'")
			assert.Equal(t, StatusOK, result.status, "Expected status 'ok' (not applicable)")
			assert.Contains(t, result.message, "Not applicable", "Expected 'Not applicable' message on non-Linux %v", result.message)
		})
	}
}

func TestCheckShellEnvironment(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	tests := []struct {
		name           string
		goenvShell     string
		goenvRoot      string
		expectedStatus Status
		expectedMsg    string
	}{
		{
			name:           "Both variables missing",
			goenvShell:     "",
			goenvRoot:      "",
			expectedStatus: StatusError,
			expectedMsg:    "goenv init has not been evaluated",
		},
		{
			name:           "Only GOENV_SHELL missing",
			goenvShell:     "",
			goenvRoot:      tmpDir,
			expectedStatus: StatusWarning,
			expectedMsg:    "incomplete shell integration",
		},
		{
			name:           "GOENV_ROOT mismatch",
			goenvShell:     "bash",
			goenvRoot:      "/wrong/path",
			expectedStatus: StatusWarning,
			expectedMsg:    "GOENV_ROOT mismatch",
		},
		{
			name:           "All correct - bash",
			goenvShell:     "bash",
			goenvRoot:      tmpDir,
			expectedStatus: StatusOK,
			expectedMsg:    "Shell integration active",
		},
		{
			name:           "All correct - zsh",
			goenvShell:     "zsh",
			goenvRoot:      tmpDir,
			expectedStatus: StatusOK,
			expectedMsg:    "shell: zsh",
		},
		{
			name:           "All correct - fish",
			goenvShell:     "fish",
			goenvRoot:      tmpDir,
			expectedStatus: StatusOK,
			expectedMsg:    "shell: fish",
		},
		{
			name:           "All correct - powershell",
			goenvShell:     "powershell",
			goenvRoot:      tmpDir,
			expectedStatus: StatusOK,
			expectedMsg:    "shell: powershell",
		},
		{
			name:           "All correct - cmd",
			goenvShell:     "cmd",
			goenvRoot:      tmpDir,
			expectedStatus: StatusOK,
			expectedMsg:    "shell: cmd",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set HOME to tmpDir to avoid checking user's real profile files
			t.Setenv(utils.EnvVarHome, tmpDir)
			t.Setenv(utils.EnvVarUserProfile, tmpDir) // Windows uses USERPROFILE instead of HOME

			// Force shell detection to bash by clearing Windows-specific variables
			// This prevents DetectShell() from auto-detecting PowerShell on Windows
			t.Setenv(utils.EnvVarPSModulePath, "")
			t.Setenv(utils.EnvVarPSModulePath, "")
			t.Setenv("COMSPEC", "")
			// Set SHELL to bash to ensure consistent shell detection across platforms
			t.Setenv(utils.EnvVarShell, "/bin/bash")

			// Set up PATH with shims directory when GOENV_SHELL is set
			if tt.goenvShell != "" {
				// Create shims directory using cfg.ShimsDir() to ensure path consistency
				shimsDir := cfg.ShimsDir()
				err = utils.EnsureDirWithContext(shimsDir, "create test directory")
				require.NoError(t, err, "Failed to create shims directory")

				// Create a fake goenv executable for command validation checks
				binDir := filepath.Join(tmpDir, "bin")
				err = utils.EnsureDirWithContext(binDir, "create test directory")
				require.NoError(t, err, "Failed to create bin directory")
				goenvBin := filepath.Join(binDir, "goenv")
				// Create a simple script that exits successfully
				testutil.WriteTestFile(t, goenvBin, []byte("#!/bin/sh\nexit 0\n"), utils.PermFileExecutable)

				// Add shims and bin to PATH (use cfg.ShimsDir() for consistency)
				oldPath := os.Getenv(utils.EnvVarPath)
				t.Setenv(utils.EnvVarPath, binDir+string(os.PathListSeparator)+cfg.ShimsDir()+string(os.PathListSeparator)+oldPath)
			}

			// Set environment variables
			if tt.goenvShell != "" {
				t.Setenv(utils.GoenvEnvVarShell.String(), tt.goenvShell)

				// For bash/zsh, set BASH_FUNC_goenv to simulate the shell function
				// This tells checkGoenvShellFunction that the function exists
				if tt.goenvShell == "bash" || tt.goenvShell == "zsh" {
					t.Setenv("BASH_FUNC_goenv%%", "() { echo fake; }")
				} else {
					// For non-bash/zsh shells, unset any inherited shell function
					t.Setenv("BASH_FUNC_goenv%%", "")
				}
			} else {
				// Explicitly set to empty string to ensure it's not inherited from parent process
				t.Setenv(utils.GoenvEnvVarShell.String(), "")
				// Also unset any shell function that might be inherited
				t.Setenv("BASH_FUNC_goenv%%", "")
			}
			if tt.goenvRoot != "" {
				t.Setenv(utils.GoenvEnvVarRoot.String(), tt.goenvRoot)
			} else {
				// Explicitly set to empty string to ensure it's not inherited from parent process
				t.Setenv(utils.GoenvEnvVarRoot.String(), "")
			}

			result := checkShellEnvironment(cfg)

			assert.Equal(t, "shell-environment", result.id, "Expected id 'shell-environment'")
			assert.Equal(t, tt.expectedStatus, result.status)
			assert.Contains(t, result.message, tt.expectedMsg)

			// Verify advice is present for non-ok statuses
			assert.False(t, result.status != StatusOK && result.advice == "")
		})
	}
}

func TestOfferShellEnvironmentFix(t *testing.T) {
	// Clear CI environment to ensure consistent test behavior
	defer testutil.ClearCIEnvironment(t)()

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	tests := []struct {
		name           string
		shellEnvStatus Status
		goenvShell     string
		userInput      string
		expectPrompt   bool
	}{
		{
			name:           "OK status - no prompt",
			shellEnvStatus: StatusOK,
			goenvShell:     "bash",
			userInput:      "",
			expectPrompt:   false,
		},
		{
			name:           "Error status - prompt shown, user accepts",
			shellEnvStatus: StatusError,
			goenvShell:     "",
			userInput:      "y\n",
			expectPrompt:   true,
		},
		{
			name:           "Warning status - prompt shown, user declines",
			shellEnvStatus: StatusWarning,
			goenvShell:     "",
			userInput:      "n\n",
			expectPrompt:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set environment
			if tt.goenvShell != "" {
				t.Setenv(utils.GoenvEnvVarShell.String(), tt.goenvShell)
			} else {
				t.Setenv(utils.GoenvEnvVarShell.String(), "")
			}

			// Create mock results
			results := []checkResult{
				{
					id:      "shell-environment",
					name:    "Shell environment",
					status:  tt.shellEnvStatus,
					message: "Test message",
					advice:  "Test advice",
				},
			}

			// Create mock stdin
			oldStdin := doctorStdin
			doctorStdin = strings.NewReader(tt.userInput)
			defer func() { doctorStdin = oldStdin }()

			// Capture output
			buf := new(bytes.Buffer)
			doctorCmd.SetOut(buf)
			doctorCmd.SetErr(buf)

			// Call the function
			offerShellEnvironmentFix(doctorCmd, results, cfg)

			output := buf.String()

			if tt.expectPrompt {
				assert.Contains(t, output, "Shell Environment Issue Detected", "Expected prompt header in output %v", output)
				assert.Contains(t, output, "Would you like to see the command", "Expected prompt question in output %v", output)

				if strings.Contains(tt.userInput, "y") {
					// Should show the fix command
					assert.Contains(t, output, "Run this command", "Expected fix command in output when user accepts %v", output)
				}
			} else {
				assert.NotContains(t, output, "Shell Environment Issue Detected", "Did not expect prompt for status %v %v", tt.shellEnvStatus, output)
			}
		})
	}
}

func TestIsInteractive(t *testing.T) {
	// This test is mostly for code coverage
	// The actual behavior depends on the terminal state
	result := isInteractive()
	// Just ensure it returns without panic
	t.Logf("isInteractive returned: %v", result)
}

func TestDetermineProfilePath(t *testing.T) {
	tests := []struct {
		shell    shellutil.ShellType
		expected string
	}{
		{shellutil.ShellTypeBash, "~/.bashrc or ~/.bash_profile"},
		{shellutil.ShellTypeZsh, "~/.zshrc"},
		{shellutil.ShellTypeFish, "~/.config/fish/config.fish"},
		{shellutil.ShellTypePowerShell, "$PROFILE"},
		{shellutil.ShellTypeCmd, "%USERPROFILE%\\autorun.cmd"},
		{shellutil.ShellTypeUnknown, "your shell profile"},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			result := shellutil.GetProfilePathDisplay(tt.shell)
			assert.Equal(t, tt.expected, result)
		})
	}
}
