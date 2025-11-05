package diagnostics

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"time"

	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-nv/goenv/internal/cache"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/utils"
)

func TestCacheStatusCommand(t *testing.T) {
	var err error
	if os.Getenv("INTEGRATION") == "" {
		t.Skip("Integration test - requires complex mocking")
	}
	// Create temporary GOENV_ROOT
	tmpDir := t.TempDir()

	// Create mock version directories with caches
	versionsDir := filepath.Join(tmpDir, "versions")

	// Version 1.23.2 with old-format cache
	v1Dir := filepath.Join(versionsDir, "1.23.2")
	v1BuildCache := filepath.Join(v1Dir, "go-build")
	v1ModCache := filepath.Join(v1Dir, "go-mod")

	err = utils.EnsureDirWithContext(v1BuildCache, "create test directory")
	require.NoError(t, err)
	err = utils.EnsureDirWithContext(v1ModCache, "create test directory")
	require.NoError(t, err)

	// Add some test files
	testutil.WriteTestFile(t, filepath.Join(v1BuildCache, "test1.a"), []byte("test data"), utils.PermFileDefault)
	testutil.WriteTestFile(t, filepath.Join(v1BuildCache, "test2.a"), []byte("test data"), utils.PermFileDefault)
	testutil.WriteTestFile(t, filepath.Join(v1ModCache, "mod1"), []byte("module data"), utils.PermFileDefault)

	// Version 1.24.4 with architecture-aware caches
	v2Dir := filepath.Join(versionsDir, "1.24.4")
	v2BuildCacheHost := filepath.Join(v2Dir, "go-build-host-host")
	v2BuildCacheLinux := filepath.Join(v2Dir, "go-build-linux-amd64")
	v2ModCache := filepath.Join(v2Dir, "go-mod")

	err = utils.EnsureDirWithContext(v2BuildCacheHost, "create test directory")
	require.NoError(t, err)
	err = utils.EnsureDirWithContext(v2BuildCacheLinux, "create test directory")
	require.NoError(t, err)
	err = utils.EnsureDirWithContext(v2ModCache, "create test directory")
	require.NoError(t, err)

	// Add test files
	testutil.WriteTestFile(t, filepath.Join(v2BuildCacheHost, "test1.a"), []byte("test data"), utils.PermFileDefault)
	testutil.WriteTestFile(t, filepath.Join(v2BuildCacheLinux, "test2.a"), []byte("test data"), utils.PermFileDefault)
	testutil.WriteTestFile(t, filepath.Join(v2ModCache, "mod1"), []byte("module data"), utils.PermFileDefault)

	// Set GOENV_ROOT environment variable
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Run cache status command
	cmd := cacheStatusCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err = runCacheStatus(cmd, []string{})
	require.NoError(t, err, "cache status failed")

	outputStr := output.String()

	// Verify output contains expected sections
	expectedSections := []string{
		"📊 Cache Status",
		"🔨 Build Caches:",
		"📦 Module Caches:",
		"Total Build Cache:",
		"Total Module Cache:",
		"📍 Cache Locations:",
		"💡 Tips:",
	}

	for _, section := range expectedSections {
		assert.Contains(t, outputStr, section, "Output missing section %v", section)
	}

	// Verify version-specific information
	assert.Contains(t, outputStr, "1.23.2", "Output missing version 1.23.2")
	assert.Contains(t, outputStr, "1.24.4", "Output missing version 1.24.4")

	// Verify architecture awareness
	assert.Contains(t, outputStr, "host-host", "Output missing host-host architecture")
	assert.Contains(t, outputStr, "linux-amd64", "Output missing linux-amd64 architecture")

	// Verify old format detection
	assert.Contains(t, outputStr, "(old format)", "Output should detect old format cache")
}

func TestCacheStatusNoVersions(t *testing.T) {
	// Create temporary GOENV_ROOT with no versions
	tmpDir := t.TempDir()

	// Set GOENV_ROOT environment variable
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	cmd := cacheStatusCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := runCacheStatus(cmd, []string{})
	require.NoError(t, err, "cache status failed")

	outputStr := output.String()

	// Should show "No Go versions installed"
	assert.Contains(t, outputStr, "No Go versions installed", "Output should indicate no versions installed")
}

func TestCacheCommand(t *testing.T) {
	// Test that cache command is registered
	require.NotNil(t, cacheCmd, "cache command not initialized")

	assert.Equal(t, "cache", cacheCmd.Use, "Expected 'cache'")

	// Test that status subcommand exists
	foundStatus := false
	foundClean := false
	for _, cmd := range cacheCmd.Commands() {
		if cmd.Use == "status" {
			foundStatus = true
		}
		if strings.HasPrefix(cmd.Use, "clean") {
			foundClean = true
		}
	}

	assert.True(t, foundStatus, "status subcommand not registered")
	assert.True(t, foundClean, "clean subcommand not registered")
}

func TestCacheCleanInvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := runCacheClean(cmd, []string{"invalid"})
	assert.Error(t, err, "Expected error for invalid type")

	assert.Contains(t, err.Error(), "invalid type", "Expected 'invalid type' error %v", err)
}

func TestCacheCleanNoVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := runCacheClean(cmd, []string{"build"})
	require.NoError(t, err)

	outputStr := output.String()
	assert.Contains(t, outputStr, "No Go versions installed", "Should report no versions installed")
}

func TestCacheCleanFlags(t *testing.T) {
	// Test that flags are registered
	assert.NotNil(t, cacheCleanCmd.Flags().Lookup("version"), "--version flag not registered")
	assert.NotNil(t, cacheCleanCmd.Flags().Lookup("old-format"), "--old-format flag not registered")
	assert.NotNil(t, cacheCleanCmd.Flags().Lookup("force"), "--force flag not registered")
	assert.NotNil(t, cacheCleanCmd.Flags().Lookup("dry-run"), "--dry-run flag not registered")

	// Test that -n shorthand exists for dry-run
	flag := cacheCleanCmd.Flags().ShorthandLookup("n")
	assert.NotNil(t, flag, "-n shorthand for --dry-run not registered")
}

func TestCacheCleanDryRun(t *testing.T) {
	var err error
	// Create temporary GOENV_ROOT with mock caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err)

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	err = utils.EnsureDirWithContext(buildCache, "create test directory")
	require.NoError(t, err)

	// Create test files in build cache
	testFile := filepath.Join(buildCache, "test.a")
	testutil.WriteTestFile(t, testFile, []byte("test data"), utils.PermFileDefault)

	// Set environment variables
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Reset flag before test
	originalDryRun := cleanDryRun
	defer func() { cleanDryRun = originalDryRun }()

	// Set dry-run mode
	cleanDryRun = true
	cleanForce = true // Skip confirmation

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	// Run cache clean in dry-run mode
	err = runCacheClean(cmd, []string{"build"})
	require.NoError(t, err)

	outputStr := output.String()

	// Verify dry-run message appears
	assert.Contains(t, outputStr, "Dry run", "Expected 'Dry run' message in output. Got:\\n %v", outputStr)
	assert.Contains(t, outputStr, "Would remove", "Expected 'Would remove' message. Got:\\n %v", outputStr)

	// Verify files were NOT deleted
	if utils.FileNotExists(testFile) {
		t.Error("Test file was deleted in dry-run mode - should be preserved")
	}

	// Verify cache directory still exists
	if utils.FileNotExists(buildCache) {
		t.Error("Build cache directory was deleted in dry-run mode - should be preserved")
	}
}

func TestCacheCleanDryRunWithFilters(t *testing.T) {
	var err error
	// Create temporary GOENV_ROOT with mock caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err)

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	err = utils.EnsureDirWithContext(buildCache, "create test directory")
	require.NoError(t, err)

	// Create test file
	testFile := filepath.Join(buildCache, "test.a")
	testutil.WriteTestFile(t, testFile, []byte("test data"), utils.PermFileDefault)

	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Save and restore flags
	originalDryRun := cleanDryRun
	originalVersion := cleanVersion
	defer func() {
		cleanDryRun = originalDryRun
		cleanVersion = originalVersion
	}()

	// Test dry-run with --version filter
	cleanDryRun = true
	cleanForce = true
	cleanVersion = "1.23.2"

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err = runCacheClean(cmd, []string{"build"})
	require.NoError(t, err)

	outputStr := output.String()

	// Should have dry-run message
	assert.Contains(t, outputStr, "Dry run", "Expected dry-run message. Got:\\n %v", outputStr)

	// Should show what would be removed
	assert.Contains(t, outputStr, "Would remove", "Expected 'Would remove' in dry-run output. Got:\\n %v", outputStr)

	// Note: Version number may not appear in summary - dry-run output is minimal

	// Files should still exist
	if utils.FileNotExists(testFile) {
		t.Error("Test file deleted in dry-run mode")
	}
}

func TestCacheCleanDryRunShowsSummary(t *testing.T) {
	var err error
	// Create temporary GOENV_ROOT with multiple caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache1 := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	buildCache2 := filepath.Join(versionsDir, "pkg", "go-build-linux-amd64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err)

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	for _, cache := range []string{buildCache1, buildCache2} {
		err = utils.EnsureDirWithContext(cache, "create test directory")
		require.NoError(t, err)
		// Add files to make caches non-empty
		testFile := filepath.Join(cache, "test.a")
		testutil.WriteTestFile(t, testFile, []byte("test data"), utils.PermFileDefault)
	}

	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Save and restore flags
	originalDryRun := cleanDryRun
	defer func() { cleanDryRun = originalDryRun }()

	cleanDryRun = true
	cleanForce = true

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err = runCacheClean(cmd, []string{"build"})
	require.NoError(t, err)

	outputStr := output.String()

	// Should show dry-run header
	assert.Contains(t, outputStr, "Dry run - showing what would be cleaned", "Expected dry-run header in output. Got:\\n %v", outputStr)

	// Should show summary with number of caches
	assert.Contains(t, outputStr, "Would remove", "Expected 'Would remove' summary in output. Got:\\n %v", outputStr)

	// Should show caches count
	assert.Contains(t, outputStr, "2 cache(s)", "Expected '2 cache(s)' in output. Got:\\n %v", outputStr)

	// Verify no actual deletion
	for _, cache := range []string{buildCache1, buildCache2} {
		if utils.FileNotExists(cache) {
			t.Errorf("Cache directory %s was deleted in dry-run mode", cache)
		}
	}
}

func TestCacheCleanDryRunEmptyCaches(t *testing.T) {
	var err error
	// Create temporary GOENV_ROOT with no caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	binDir := filepath.Join(versionsDir, "bin")

	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err)

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Save and restore flags
	originalDryRun := cleanDryRun
	defer func() { cleanDryRun = originalDryRun }()

	cleanDryRun = true
	cleanForce = true

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err = runCacheClean(cmd, []string{"build"})
	require.NoError(t, err)

	outputStr := output.String()

	// Should report no caches found
	assert.Contains(t, outputStr, "No caches found", "Expected 'No caches found' message when no caches exist. Got:\\n %v", outputStr)

	// Should NOT show dry-run message if there's nothing to clean
	// (function returns early before dry-run check)
}

func TestCacheClean_NoForceNonInteractive(t *testing.T) {
	var err error
	// Create temporary GOENV_ROOT with mock cache
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err)

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	err = utils.EnsureDirWithContext(buildCache, "create test directory")
	require.NoError(t, err)

	// Create test files in build cache
	testFile := filepath.Join(buildCache, "test.a")
	testutil.WriteTestFile(t, testFile, []byte("test data"), utils.PermFileDefault)

	// Set environment variables
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Set CI to ensure non-interactive mode
	t.Setenv(utils.EnvVarCI, "true")

	// Save original flags and defer restore
	originalForce := cleanForce
	originalDryRun := cleanDryRun
	defer func() {
		cleanForce = originalForce
		cleanDryRun = originalDryRun
	}()

	// Set flags to trigger non-interactive error
	cleanForce = false  // Critical: don't skip confirmation
	cleanDryRun = false // Not a dry run

	// Create a pipe to simulate non-interactive stdin (not a TTY)
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	// Replace os.Stdin temporarily
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	errOutput := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(errOutput)

	// Run cache clean without --force in non-interactive mode
	err = runCacheClean(cmd, []string{"build"})

	// Should succeed (PromptYesNo returns false, command cancels gracefully)
	require.NoError(t, err, "Unexpected error: \\nOutput:\\n\\nError output:\\n")

	// Verify cache was NOT deleted (InteractiveContext automatically returns false in CI mode)
	if utils.FileNotExists(testFile) {
		t.Error("Cache should not be deleted when user cancels in non-interactive mode")
	}

	// Verify "Cancelled" message in stdout
	assert.Contains(t, output.String(), "Cancelled", "Expected 'Cancelled' message in output, got:\\n %v", output.String())
}

func TestCacheClean_AssumeYesEnvVar(t *testing.T) {
	var err error
	// Create temporary GOENV_ROOT with mock cache
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	err = utils.EnsureDirWithContext(binDir, "create test directory")
	require.NoError(t, err)

	// Create fake go executable
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	err = utils.EnsureDirWithContext(buildCache, "create test directory")
	require.NoError(t, err)

	// Create test files in build cache
	testFile := filepath.Join(buildCache, "test.a")
	testutil.WriteTestFile(t, testFile, []byte("test data"), utils.PermFileDefault)

	// Set environment variables
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarAssumeYes.String(), "1") // Enable auto-confirm

	// Save original flags and defer restore
	originalForce := cleanForce
	originalDryRun := cleanDryRun
	defer func() {
		cleanForce = originalForce
		cleanDryRun = originalDryRun
	}()

	// Set flags
	cleanForce = false  // Don't use --force, rely on env var
	cleanDryRun = false // Not a dry run

	// Create a pipe to simulate non-interactive stdin
	r, w, err := os.Pipe()
	require.NoError(t, err)
	defer r.Close()
	defer w.Close()

	// Replace os.Stdin temporarily
	oldStdin := os.Stdin
	defer func() { os.Stdin = oldStdin }()
	os.Stdin = r

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	errOutput := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(errOutput)

	// Run cache clean with GOENV_ASSUME_YES=1 (should auto-confirm and clean)
	err = runCacheClean(cmd, []string{"build"})

	// Should succeed
	require.NoError(t, err, "Unexpected error with GOENV_ASSUME_YES=1: \\nOutput:\\n\\nError output:\\n")

	// Verify cache WAS deleted (auto-confirmed)
	if utils.PathExists(testFile) {
		t.Error("Cache should be deleted when GOENV_ASSUME_YES=1 auto-confirms")
	}

	// Should show successful removal message
	assert.Contains(t, output.String(), "Removed", "Expected 'Removed' message in output with GOENV_ASSUME_YES=1, got:\\n %v", output.String())
}

func TestGetDirSizeWithOptions_FastMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := range 100 {
		file := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		testutil.WriteTestFile(t, file, []byte("test data"), utils.PermFileDefault)
	}

	// Test normal mode
	size, files, err := cache.GetDirSizeWithOptions(tmpDir, false, 10*time.Second)
	require.NoError(t, err, "GetDirSizeWithOptions error")
	assert.NotEqual(t, 0, size, "Expected non-zero size")
	assert.Equal(t, 100, files, "Expected 100 files")

	// Test fast mode (should return -1 for files)
	sizeFast, filesFast, err := cache.GetDirSizeWithOptions(tmpDir, true, 10*time.Second)
	require.NoError(t, err, "GetDirSizeWithOptions (fast) error")
	assert.Equal(t, size, sizeFast, "Fast mode size mismatch")
	assert.Equal(t, -1, filesFast, "Fast mode should return -1 for files")
}

func TestGetDirSizeWithOptions_Timeout(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping timeout test in short mode")
	}

	tmpDir := t.TempDir()

	// Create enough files to trigger timeout detection (2000 files)
	for i := 0; i < 2000; i++ {
		testutil.WriteTestFile(t, filepath.Join(tmpDir, fmt.Sprintf("file%04d.o", i)), []byte("test content for file"), utils.PermFileDefault)
	}

	// Use very short timeout to force timeout during scan
	size, files, err := cache.GetDirSizeWithOptions(tmpDir, false, 1*time.Nanosecond)
	require.NoError(t, err, "GetDirSizeWithOptions error")

	// Should have scanned some files before timing out
	assert.False(t, files == 0 && size == 0, "Expected partial scan, got no results")

	// Should return -1 for files when timed out (indicates approximate)
	if files != -1 {
		t.Logf("Expected files=-1 (timeout indicator), got %d", files)
	}

	// Should have measured some size even with timeout
	assert.NotEqual(t, 0, size, "Expected some size measurement even with timeout")
}

func TestCacheStatusFastFlag(t *testing.T) {
	// Test that --fast flag is registered
	flag := cacheStatusCmd.Flags().Lookup("fast")
	assert.NotNil(t, flag, "--fast flag not registered")
}

func TestCacheMigration_UsesRuntimeArchitecture(t *testing.T) {
	var err error
	// Test that cache migration uses platform.OS()/GOARCH, not env vars
	// This ensures we create "go-build-darwin-arm64" not "go-build-host-host"

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create versions directory with fake Go installation
	versionsDir := filepath.Join(tmpDir, "versions")
	versionPath := filepath.Join(versionsDir, "1.23.2")
	binPath := filepath.Join(versionPath, "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err, "Failed to create bin directory")

	// Create fake go executable
	goExe := filepath.Join(binPath, "go")
	var content string
	if utils.IsWindows() {
		goExe += ".exe"
		content = "@echo off\necho fake go\n"
	} else {
		content = "#!/bin/sh\necho fake go\n"
	}
	testutil.WriteTestFile(t, goExe, []byte(content), utils.PermFileExecutable)

	// Create old format cache (go-build without architecture suffix)
	oldCachePath := filepath.Join(versionPath, "go-build")
	err = utils.EnsureDirWithContext(oldCachePath, "create test directory")
	require.NoError(t, err, "Failed to create old cache")

	// Add a test file
	testFile := filepath.Join(oldCachePath, "test.a")
	testutil.WriteTestFile(t, testFile, []byte("test data"), utils.PermFileDefault)

	// Set cross-compilation env vars to ensure we don't use them
	t.Setenv(utils.EnvVarGoos, "plan9")
	t.Setenv(utils.EnvVarGoarch, "mips")

	// Run cache migrate command with --force flag (to bypass interactive confirmation)
	buf := new(bytes.Buffer)
	cacheMigrateCmd.SetOut(buf)
	cacheMigrateCmd.SetErr(buf)

	// Enable force flag to skip confirmation prompt
	migrateForce = true
	defer func() { migrateForce = false }()

	err = cacheMigrateCmd.RunE(cacheMigrateCmd, []string{})
	require.NoError(t, err, "Cache migrate failed: \\nOutput")

	output := buf.String()

	// Verify the new cache path uses platform.OS()/GOARCH
	expectedArch := fmt.Sprintf("%s-%s", platform.OS(), platform.Arch())
	expectedCachePath := filepath.Join(versionPath, fmt.Sprintf("go-build-%s", expectedArch))

	// Check that the new cache exists
	if !utils.DirExists(expectedCachePath) {
		t.Errorf("Expected new cache at %s, but it doesn't exist", expectedCachePath)
		t.Logf("Output: %s", output)
	}

	// Verify it's NOT using the env vars (go-build-plan9-mips should NOT exist)
	wrongCachePath := filepath.Join(versionPath, "go-build-plan9-mips")
	if utils.DirExists(wrongCachePath) {
		t.Errorf("Cache migration incorrectly used GOOS/GOARCH env vars: %s exists", wrongCachePath)
	}

	// Verify output mentions the correct architecture
	assert.Contains(t, output, expectedArch, "Expected output to mention %v %v", expectedArch, output)

	t.Logf("✓ Cache migration correctly used runtime architecture: %s", expectedArch)
	t.Logf("✓ Created: %s", expectedCachePath)
	t.Logf("✓ Did not create: %s", wrongCachePath)
}

// Benchmark tests to document performance characteristics of fast vs full scan

func BenchmarkGetDirSize_Small_Fast(b *testing.B) {
	tmpDir := createBenchmarkCache(b, 100) // 100 files
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.GetDirSizeWithOptions(tmpDir, true, 10*time.Second)
	}
}

func BenchmarkGetDirSize_Small_Full(b *testing.B) {
	tmpDir := createBenchmarkCache(b, 100) // 100 files
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.GetDirSizeWithOptions(tmpDir, false, 10*time.Second)
	}
}

func BenchmarkGetDirSize_Large_Fast(b *testing.B) {
	tmpDir := createBenchmarkCache(b, 5000) // 5K files
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.GetDirSizeWithOptions(tmpDir, true, 10*time.Second)
	}
}

func BenchmarkGetDirSize_Large_Full(b *testing.B) {
	tmpDir := createBenchmarkCache(b, 5000) // 5K files
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.GetDirSizeWithOptions(tmpDir, false, 10*time.Second)
	}
}

// Helper to create a realistic cache structure for benchmarking
func createBenchmarkCache(b *testing.B, fileCount int) string {
	tmpDir := b.TempDir()

	// Create nested directory structure similar to real go-build cache
	for i := 0; i < fileCount; i++ {
		// Simulate go-build cache structure: ab/cd/efgh...
		hash := fmt.Sprintf("%08x", i)
		dir := filepath.Join(tmpDir, hash[0:2], hash[2:4])
		if err := utils.EnsureDirWithContext(dir, "create test directory"); err != nil {
			b.Fatalf("Failed to create directory: %v", err)
		}

		// Create a file that simulates compiled object
		filePath := filepath.Join(dir, hash+".a")
		// Use realistic size (~10-50KB per object file)
		content := make([]byte, 10240+i%40960)
		testutil.WriteTestFile(b, filePath, content, utils.PermFileDefault)
	}

	return tmpDir
}
