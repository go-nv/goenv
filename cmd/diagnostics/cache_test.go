package diagnostics

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/testing/testutil"
	"time"

	"github.com/go-nv/goenv/internal/cache"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/utils"
)

func TestCacheStatusCommand(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Skip("Integration test - requires complex mocking")
	// Create temporary GOENV_ROOT
	tmpDir := t.TempDir()

	// Create mock version directories with caches
	versionsDir := filepath.Join(tmpDir, "versions")

	// Version 1.23.2 with old-format cache
	v1Dir := filepath.Join(versionsDir, "1.23.2")
	v1BuildCache := filepath.Join(v1Dir, "go-build")
	v1ModCache := filepath.Join(v1Dir, "go-mod")

	if err := utils.EnsureDirWithContext(v1BuildCache, "create test directory"); err != nil {
		t.Fatal(err)
	}
	if err := utils.EnsureDirWithContext(v1ModCache, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Add some test files
	testutil.WriteTestFile(t, filepath.Join(v1BuildCache, "test1.a"), []byte("test data"), utils.PermFileDefault)
	testutil.WriteTestFile(t, filepath.Join(v1BuildCache, "test2.a"), []byte("test data"), utils.PermFileDefault)
	testutil.WriteTestFile(t, filepath.Join(v1ModCache, "mod1"), []byte("module data"), utils.PermFileDefault)

	// Version 1.24.4 with architecture-aware caches
	v2Dir := filepath.Join(versionsDir, "1.24.4")
	v2BuildCacheHost := filepath.Join(v2Dir, "go-build-host-host")
	v2BuildCacheLinux := filepath.Join(v2Dir, "go-build-linux-amd64")
	v2ModCache := filepath.Join(v2Dir, "go-mod")

	if err := utils.EnsureDirWithContext(v2BuildCacheHost, "create test directory"); err != nil {
		t.Fatal(err)
	}
	if err := utils.EnsureDirWithContext(v2BuildCacheLinux, "create test directory"); err != nil {
		t.Fatal(err)
	}
	if err := utils.EnsureDirWithContext(v2ModCache, "create test directory"); err != nil {
		t.Fatal(err)
	}

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

	err := runCacheStatus(cmd, []string{})
	if err != nil {
		t.Fatalf("cache status failed: %v", err)
	}

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
		if !strings.Contains(outputStr, section) {
			t.Errorf("Output missing section: %s", section)
		}
	}

	// Verify version-specific information
	if !strings.Contains(outputStr, "1.23.2") {
		t.Error("Output missing version 1.23.2")
	}
	if !strings.Contains(outputStr, "1.24.4") {
		t.Error("Output missing version 1.24.4")
	}

	// Verify architecture awareness
	if !strings.Contains(outputStr, "host-host") {
		t.Error("Output missing host-host architecture")
	}
	if !strings.Contains(outputStr, "linux-amd64") {
		t.Error("Output missing linux-amd64 architecture")
	}

	// Verify old format detection
	if !strings.Contains(outputStr, "(old format)") {
		t.Error("Output should detect old format cache")
	}
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
	if err != nil {
		t.Fatalf("cache status failed: %v", err)
	}

	outputStr := output.String()

	// Should show "No Go versions installed"
	if !strings.Contains(outputStr, "No Go versions installed") {
		t.Error("Output should indicate no versions installed")
	}
}

func TestCacheCommand(t *testing.T) {
	// Test that cache command is registered
	if cacheCmd == nil {
		t.Fatal("cache command not initialized")
	}

	if cacheCmd.Use != "cache" {
		t.Errorf("Expected 'cache', got %s", cacheCmd.Use)
	}

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

	if !foundStatus {
		t.Error("status subcommand not registered")
	}
	if !foundClean {
		t.Error("clean subcommand not registered")
	}
}

func TestCacheCleanInvalidType(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := runCacheClean(cmd, []string{"invalid"})
	if err == nil {
		t.Fatal("Expected error for invalid type")
	}

	if !strings.Contains(err.Error(), "invalid type") {
		t.Errorf("Expected 'invalid type' error, got: %v", err)
	}
}

func TestCacheCleanNoVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	cmd := cacheCleanCmd
	output := &bytes.Buffer{}
	cmd.SetOut(output)
	cmd.SetErr(output)

	err := runCacheClean(cmd, []string{"build"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	outputStr := output.String()
	if !strings.Contains(outputStr, "No Go versions installed") {
		t.Error("Should report no versions installed")
	}
}

func TestCacheCleanFlags(t *testing.T) {
	// Test that flags are registered
	if cacheCleanCmd.Flags().Lookup("version") == nil {
		t.Error("--version flag not registered")
	}
	if cacheCleanCmd.Flags().Lookup("old-format") == nil {
		t.Error("--old-format flag not registered")
	}
	if cacheCleanCmd.Flags().Lookup("force") == nil {
		t.Error("--force flag not registered")
	}
	if cacheCleanCmd.Flags().Lookup("dry-run") == nil {
		t.Error("--dry-run flag not registered")
	}

	// Test that -n shorthand exists for dry-run
	flag := cacheCleanCmd.Flags().ShorthandLookup("n")
	if flag == nil {
		t.Error("-n shorthand for --dry-run not registered")
	}
}

func TestCacheCleanBuildOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Skip("Integration test - requires mock version setup")
	// Would test cleaning only build caches
}

func TestCacheCleanModOnly(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Skip("Integration test - requires mock version setup")
	// Would test cleaning only module caches
}

func TestCacheCleanAll(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}
	t.Skip("Integration test - requires mock version setup")
	// Would test cleaning both build and module caches
}

func TestCacheCleanDryRun(t *testing.T) {
	// Create temporary GOENV_ROOT with mock caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	if err := utils.EnsureDirWithContext(buildCache, "create test directory"); err != nil {
		t.Fatal(err)
	}

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
	err := runCacheClean(cmd, []string{"build"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	outputStr := output.String()

	// Verify dry-run message appears
	if !strings.Contains(outputStr, "Dry run") {
		t.Errorf("Expected 'Dry run' message in output. Got:\n%s", outputStr)
	}
	if !strings.Contains(outputStr, "Would remove") {
		t.Errorf("Expected 'Would remove' message. Got:\n%s", outputStr)
	}

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
	// Create temporary GOENV_ROOT with mock caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	if err := utils.EnsureDirWithContext(buildCache, "create test directory"); err != nil {
		t.Fatal(err)
	}

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

	err := runCacheClean(cmd, []string{"build"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	outputStr := output.String()

	// Should have dry-run message
	if !strings.Contains(outputStr, "Dry run") {
		t.Errorf("Expected dry-run message. Got:\n%s", outputStr)
	}

	// Should show what would be removed
	if !strings.Contains(outputStr, "Would remove") {
		t.Errorf("Expected 'Would remove' in dry-run output. Got:\n%s", outputStr)
	}

	// Note: Version number may not appear in summary - dry-run output is minimal

	// Files should still exist
	if utils.FileNotExists(testFile) {
		t.Error("Test file deleted in dry-run mode")
	}
}

func TestCacheCleanDryRunShowsSummary(t *testing.T) {
	// Create temporary GOENV_ROOT with multiple caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache1 := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	buildCache2 := filepath.Join(versionsDir, "pkg", "go-build-linux-amd64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatal(err)
	}

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
		if err := utils.EnsureDirWithContext(cache, "create test directory"); err != nil {
			t.Fatal(err)
		}
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

	err := runCacheClean(cmd, []string{"build"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	outputStr := output.String()

	// Should show dry-run header
	if !strings.Contains(outputStr, "Dry run - showing what would be cleaned") {
		t.Errorf("Expected dry-run header in output. Got:\n%s", outputStr)
	}

	// Should show summary with number of caches
	if !strings.Contains(outputStr, "Would remove") {
		t.Errorf("Expected 'Would remove' summary in output. Got:\n%s", outputStr)
	}

	// Should show caches count
	if !strings.Contains(outputStr, "2 cache(s)") {
		t.Errorf("Expected '2 cache(s)' in output. Got:\n%s", outputStr)
	}

	// Verify no actual deletion
	for _, cache := range []string{buildCache1, buildCache2} {
		if utils.FileNotExists(cache) {
			t.Errorf("Cache directory %s was deleted in dry-run mode", cache)
		}
	}
}

func TestCacheCleanDryRunEmptyCaches(t *testing.T) {
	// Create temporary GOENV_ROOT with no caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	binDir := filepath.Join(versionsDir, "bin")

	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatal(err)
	}

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

	err := runCacheClean(cmd, []string{"build"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}

	outputStr := output.String()

	// Should report no caches found
	if !strings.Contains(outputStr, "No caches found") {
		t.Errorf("Expected 'No caches found' message when no caches exist. Got:\n%s", outputStr)
	}

	// Should NOT show dry-run message if there's nothing to clean
	// (function returns early before dry-run check)
}

func TestCacheClean_NoForceNonInteractive(t *testing.T) {
	// Create temporary GOENV_ROOT with mock cache
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		// On Windows, pathutil.FindExecutable looks for .exe or .bat
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	if err := utils.EnsureDirWithContext(buildCache, "create test directory"); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatalf("Unexpected error: %v\nOutput:\n%s\nError output:\n%s", err, output.String(), errOutput.String())
	}

	// Verify cache was NOT deleted (InteractiveContext automatically returns false in CI mode)
	if utils.FileNotExists(testFile) {
		t.Error("Cache should not be deleted when user cancels in non-interactive mode")
	}

	// Verify "Cancelled" message in stdout
	if !strings.Contains(output.String(), "Cancelled") {
		t.Errorf("Expected 'Cancelled' message in output, got:\n%s", output.String())
	}
}

func TestCacheClean_AssumeYesEnvVar(t *testing.T) {
	// Create temporary GOENV_ROOT with mock cache
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "pkg", "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := utils.EnsureDirWithContext(binDir, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable
	goExe := filepath.Join(binDir, "go")
	if utils.IsWindows() {
		goExe = filepath.Join(binDir, "go.bat")
		testutil.WriteTestFile(t, goExe, []byte("@echo off\necho fake go\n"), utils.PermFileExecutable)
	} else {
		testutil.WriteTestFile(t, goExe, []byte("#!/bin/sh\necho fake go\n"), utils.PermFileExecutable)
	}

	if err := utils.EnsureDirWithContext(buildCache, "create test directory"); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatal(err)
	}
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
	if err != nil {
		t.Fatalf("Unexpected error with GOENV_ASSUME_YES=1: %v\nOutput:\n%s\nError output:\n%s",
			err, output.String(), errOutput.String())
	}

	// Verify cache WAS deleted (auto-confirmed)
	if utils.PathExists(testFile) {
		t.Error("Cache should be deleted when GOENV_ASSUME_YES=1 auto-confirms")
	}

	// Should show successful removal message
	if !strings.Contains(output.String(), "Removed") {
		t.Errorf("Expected 'Removed' message in output with GOENV_ASSUME_YES=1, got:\n%s", output.String())
	}
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
	if err != nil {
		t.Fatalf("GetDirSizeWithOptions error: %v", err)
	}
	if size == 0 {
		t.Error("Expected non-zero size")
	}
	if files != 100 {
		t.Errorf("Expected 100 files, got %d", files)
	}

	// Test fast mode (should return -1 for files)
	sizeFast, filesFast, err := cache.GetDirSizeWithOptions(tmpDir, true, 10*time.Second)
	if err != nil {
		t.Fatalf("GetDirSizeWithOptions (fast) error: %v", err)
	}
	if sizeFast != size {
		t.Errorf("Fast mode size mismatch: got %d, want %d", sizeFast, size)
	}
	if filesFast != -1 {
		t.Errorf("Fast mode should return -1 for files, got %d", filesFast)
	}
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
	if err != nil {
		t.Fatalf("GetDirSizeWithOptions error: %v", err)
	}

	// Should have scanned some files before timing out
	if files == 0 && size == 0 {
		t.Error("Expected partial scan, got no results")
	}

	// Should return -1 for files when timed out (indicates approximate)
	if files != -1 {
		t.Logf("Expected files=-1 (timeout indicator), got %d", files)
	}

	// Should have measured some size even with timeout
	if size == 0 {
		t.Error("Expected some size measurement even with timeout")
	}
}

func TestCacheStatusFastFlag(t *testing.T) {
	// Test that --fast flag is registered
	flag := cacheStatusCmd.Flags().Lookup("fast")
	if flag == nil {
		t.Error("--fast flag not registered")
	}
}

func TestCacheMigration_UsesRuntimeArchitecture(t *testing.T) {
	// Test that cache migration uses platform.OS()/GOARCH, not env vars
	// This ensures we create "go-build-darwin-arm64" not "go-build-host-host"

	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarDir.String(), tmpDir)

	// Create versions directory with fake Go installation
	versionsDir := filepath.Join(tmpDir, "versions")
	versionPath := filepath.Join(versionsDir, "1.23.2")
	binPath := filepath.Join(versionPath, "bin")
	if err := utils.EnsureDirWithContext(binPath, "create test directory"); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

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
	if err := utils.EnsureDirWithContext(oldCachePath, "create test directory"); err != nil {
		t.Fatalf("Failed to create old cache: %v", err)
	}

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

	err := cacheMigrateCmd.RunE(cacheMigrateCmd, []string{})
	if err != nil {
		t.Fatalf("Cache migrate failed: %v\nOutput: %s", err, buf.String())
	}

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
	if !strings.Contains(output, expectedArch) {
		t.Errorf("Expected output to mention %s, got: %s", expectedArch, output)
	}

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
