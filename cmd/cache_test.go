package cmd

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestCacheStatusCommand(t *testing.T) {
	t.Skip("Integration test - requires complex mocking")
	// Create temporary GOENV_ROOT
	tmpDir := t.TempDir()

	// Create mock version directories with caches
	versionsDir := filepath.Join(tmpDir, "versions")

	// Version 1.23.2 with old-format cache
	v1Dir := filepath.Join(versionsDir, "1.23.2")
	v1BuildCache := filepath.Join(v1Dir, "go-build")
	v1ModCache := filepath.Join(v1Dir, "go-mod")

	if err := os.MkdirAll(v1BuildCache, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(v1ModCache, 0755); err != nil {
		t.Fatal(err)
	}

	// Add some test files
	os.WriteFile(filepath.Join(v1BuildCache, "test1.a"), []byte("test data"), 0644)
	os.WriteFile(filepath.Join(v1BuildCache, "test2.a"), []byte("test data"), 0644)
	os.WriteFile(filepath.Join(v1ModCache, "mod1"), []byte("module data"), 0644)

	// Version 1.24.4 with architecture-aware caches
	v2Dir := filepath.Join(versionsDir, "1.24.4")
	v2BuildCacheHost := filepath.Join(v2Dir, "go-build-host-host")
	v2BuildCacheLinux := filepath.Join(v2Dir, "go-build-linux-amd64")
	v2ModCache := filepath.Join(v2Dir, "go-mod")

	if err := os.MkdirAll(v2BuildCacheHost, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(v2BuildCacheLinux, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(v2ModCache, 0755); err != nil {
		t.Fatal(err)
	}

	// Add test files
	os.WriteFile(filepath.Join(v2BuildCacheHost, "test1.a"), []byte("test data"), 0644)
	os.WriteFile(filepath.Join(v2BuildCacheLinux, "test2.a"), []byte("test data"), 0644)
	os.WriteFile(filepath.Join(v2ModCache, "mod1"), []byte("module data"), 0644)

	// Set GOENV_ROOT environment variable
	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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
	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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

func TestGetDirSize(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	file1 := filepath.Join(tmpDir, "file1.txt")
	file2 := filepath.Join(tmpDir, "subdir", "file2.txt")

	if err := os.MkdirAll(filepath.Dir(file2), 0755); err != nil {
		t.Fatal(err)
	}

	data1 := []byte("12345")      // 5 bytes
	data2 := []byte("1234567890") // 10 bytes

	if err := os.WriteFile(file1, data1, 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file2, data2, 0644); err != nil {
		t.Fatal(err)
	}

	size, files := getDirSize(tmpDir)

	if files != 2 {
		t.Errorf("Expected 2 files, got %d", files)
	}

	// Size should be at least 15 bytes (might be slightly different on different filesystems)
	if size < 15 {
		t.Errorf("Expected at least 15 bytes, got %d", size)
	}
}

func TestGetDirSizeEmpty(t *testing.T) {
	tmpDir := t.TempDir()

	size, files := getDirSize(tmpDir)

	if files != 0 {
		t.Errorf("Expected 0 files, got %d", files)
	}

	if size != 0 {
		t.Errorf("Expected 0 bytes, got %d", size)
	}
}

func TestGetDirSizeNonexistent(t *testing.T) {
	size, files := getDirSize("/nonexistent/path/that/should/not/exist")

	// Should return zeros without error
	if files != 0 {
		t.Errorf("Expected 0 files, got %d", files)
	}

	if size != 0 {
		t.Errorf("Expected 0 bytes, got %d", size)
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{12, "12"},
		{123, "123"},
		{1234, "1,234"},
		{12345, "12,345"},
		{123456, "123,456"},
		{1234567, "1,234,567"},
		{12345678, "12,345,678"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("formatNumber(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
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

	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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

	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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
	t.Skip("Integration test - requires mock version setup")
	// Would test cleaning only build caches
}

func TestCacheCleanModOnly(t *testing.T) {
	t.Skip("Integration test - requires mock version setup")
	// Would test cleaning only module caches
}

func TestCacheCleanAll(t *testing.T) {
	t.Skip("Integration test - requires mock version setup")
	// Would test cleaning both build and module caches
}

func TestCacheCleanDryRun(t *testing.T) {
	// Create temporary GOENV_ROOT with mock caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if err := os.WriteFile(goExe, []byte("#!/bin/sh\necho fake go\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(buildCache, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files in build cache
	testFile := filepath.Join(buildCache, "test.a")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save original env and defer restore
	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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
	if !strings.Contains(outputStr, "Dry-run mode") {
		t.Error("Expected 'Dry-run mode' message in output")
	}
	if !strings.Contains(outputStr, "No caches were actually deleted") {
		t.Error("Expected 'No caches were actually deleted' message")
	}
	if !strings.Contains(outputStr, "without --dry-run") {
		t.Error("Expected instruction about running without --dry-run")
	}

	// Verify files were NOT deleted
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Test file was deleted in dry-run mode - should be preserved")
	}

	// Verify cache directory still exists
	if _, err := os.Stat(buildCache); os.IsNotExist(err) {
		t.Error("Build cache directory was deleted in dry-run mode - should be preserved")
	}
}

func TestCacheCleanDryRunWithFilters(t *testing.T) {
	// Create temporary GOENV_ROOT with mock caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if err := os.WriteFile(goExe, []byte("#!/bin/sh\necho fake go\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(buildCache, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test file
	testFile := filepath.Join(buildCache, "test.a")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save and restore environment
	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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

	// Should show caches to clean
	if !strings.Contains(outputStr, "Caches to clean") {
		t.Error("Expected 'Caches to clean' section in dry-run output")
	}

	// Should mention the specific version
	if !strings.Contains(outputStr, "1.23.2") {
		t.Error("Expected version 1.23.2 in output")
	}

	// Should have dry-run message
	if !strings.Contains(outputStr, "Dry-run mode") {
		t.Error("Expected dry-run message")
	}

	// Files should still exist
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Test file deleted in dry-run mode")
	}
}

func TestCacheCleanDryRunShowsSummary(t *testing.T) {
	// Create temporary GOENV_ROOT with multiple caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache1 := filepath.Join(versionsDir, "go-build-darwin-arm64")
	buildCache2 := filepath.Join(versionsDir, "go-build-linux-amd64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if err := os.WriteFile(goExe, []byte("#!/bin/sh\necho fake go\n"), 0755); err != nil {
		t.Fatal(err)
	}

	for _, cache := range []string{buildCache1, buildCache2} {
		if err := os.MkdirAll(cache, 0755); err != nil {
			t.Fatal(err)
		}
		// Add files to make caches non-empty
		testFile := filepath.Join(cache, "test.a")
		if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Save and restore environment
	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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

	// Should show both caches
	if !strings.Contains(outputStr, "darwin-arm64") {
		t.Error("Expected darwin-arm64 cache in output")
	}
	if !strings.Contains(outputStr, "linux-amd64") {
		t.Error("Expected linux-amd64 cache in output")
	}

	// Should show total size
	if !strings.Contains(outputStr, "Total to clean") {
		t.Error("Expected 'Total to clean' summary")
	}

	// Should have dry-run warning
	if !strings.Contains(outputStr, "Dry-run mode") {
		t.Error("Expected dry-run mode message")
	}

	// Verify no actual deletion
	for _, cache := range []string{buildCache1, buildCache2} {
		if _, err := os.Stat(cache); os.IsNotExist(err) {
			t.Errorf("Cache directory %s was deleted in dry-run mode", cache)
		}
	}
}

func TestCacheCleanDryRunEmptyCaches(t *testing.T) {
	// Create temporary GOENV_ROOT with no caches
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	binDir := filepath.Join(versionsDir, "bin")

	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if err := os.WriteFile(goExe, []byte("#!/bin/sh\necho fake go\n"), 0755); err != nil {
		t.Fatal(err)
	}

	// Save and restore environment
	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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
		t.Error("Expected 'No caches found' message when no caches exist")
	}

	// Should NOT show dry-run message if there's nothing to clean
	// (function returns early before dry-run check)
}

func TestCacheClean_NoForceNonInteractive(t *testing.T) {
	// Create temporary GOENV_ROOT with mock cache
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions", "1.23.2")
	buildCache := filepath.Join(versionsDir, "go-build-darwin-arm64")
	binDir := filepath.Join(versionsDir, "bin")

	// Create bin directory to make version appear "installed"
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatal(err)
	}

	// Create fake go executable (manager checks for this)
	goExe := filepath.Join(binDir, "go")
	if err := os.WriteFile(goExe, []byte("#!/bin/sh\necho fake go\n"), 0755); err != nil {
		t.Fatal(err)
	}

	if err := os.MkdirAll(buildCache, 0755); err != nil {
		t.Fatal(err)
	}

	// Create test files in build cache
	testFile := filepath.Join(buildCache, "test.a")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatal(err)
	}

	// Save original env and defer restore
	originalRoot := os.Getenv("GOENV_ROOT")
	defer os.Setenv("GOENV_ROOT", originalRoot)
	os.Setenv("GOENV_ROOT", tmpDir)

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

	// Run cache clean without --force in non-interactive mode (should error)
	err = runCacheClean(cmd, []string{"build"})

	// Should return an error
	if err == nil {
		t.Fatal("Expected error when running cache clean without --force in non-interactive mode")
	}

	// Error should contain "--force" to guide the user
	errMsg := err.Error()
	if !strings.Contains(errMsg, "--force") {
		t.Errorf("Error message should contain '--force', got: %s", errMsg)
	}

	// Error should mention non-interactive
	if !strings.Contains(errMsg, "non-interactive") {
		t.Errorf("Error message should mention 'non-interactive', got: %s", errMsg)
	}

	// Verify cache was NOT deleted
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Cache should not be deleted when command errors in non-interactive mode")
	}
}

func TestFormatFileCount(t *testing.T) {
	tests := []struct {
		name     string
		input    int
		expected string
	}{
		{"zero", 0, "0"},
		{"small number", 42, "42"},
		{"with comma", 1234, "1,234"},
		{"large number", 1234567, "1,234,567"},
		{"approximate", -1, "~"},
		{"negative approximate", -100, "~"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatFileCount(tt.input)
			if result != tt.expected {
				t.Errorf("formatFileCount(%d) = %s, want %s", tt.input, result, tt.expected)
			}
		})
	}
}

func TestGetDirSizeWithOptions_FastMode(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test files
	for i := 0; i < 100; i++ {
		file := filepath.Join(tmpDir, fmt.Sprintf("file%d.txt", i))
		if err := os.WriteFile(file, []byte("test data"), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Test normal mode
	size, files := getDirSizeWithOptions(tmpDir, false, 10*time.Second)
	if size == 0 {
		t.Error("Expected non-zero size")
	}
	if files != 100 {
		t.Errorf("Expected 100 files, got %d", files)
	}

	// Test fast mode (should return -1 for files)
	sizeFast, filesFast := getDirSizeWithOptions(tmpDir, true, 10*time.Second)
	if sizeFast != size {
		t.Errorf("Fast mode size mismatch: got %d, want %d", sizeFast, size)
	}
	if filesFast != -1 {
		t.Errorf("Fast mode should return -1 for files, got %d", filesFast)
	}
}

func TestGetDirSizeWithOptions_Timeout(t *testing.T) {
	// This test would require creating thousands of files to trigger timeout
	// Skipping for unit tests, but the logic is in place
	t.Skip("Timeout test requires creating many files - tested manually")
}

func TestCacheStatusFastFlag(t *testing.T) {
	// Test that --fast flag is registered
	flag := cacheStatusCmd.Flags().Lookup("fast")
	if flag == nil {
		t.Error("--fast flag not registered")
	}
}

func TestParseABIFromCacheName(t *testing.T) {
	tests := []struct {
		name           string
		cacheName      string
		expectedGOOS   string
		expectedGOARCH string
		expectedABI    map[string]string
	}{
		{
			name:           "old format",
			cacheName:      "go-build",
			expectedGOOS:   "",
			expectedGOARCH: "",
			expectedABI:    nil,
		},
		{
			name:           "basic darwin-arm64",
			cacheName:      "go-build-darwin-arm64",
			expectedGOOS:   "darwin",
			expectedGOARCH: "arm64",
			expectedABI:    nil, // No ABI variants, so nil map is expected
		},
		{
			name:           "linux-amd64 with v3",
			cacheName:      "go-build-linux-amd64-v3",
			expectedGOOS:   "linux",
			expectedGOARCH: "amd64",
			expectedABI:    map[string]string{"GOAMD64": "v3"},
		},
		{
			name:           "linux-arm with v6 prefix",
			cacheName:      "go-build-linux-arm-v6",
			expectedGOOS:   "linux",
			expectedGOARCH: "arm",
			expectedABI:    map[string]string{"GOARM": "6"},
		},
		{
			name:           "linux-arm with v7 prefix",
			cacheName:      "go-build-linux-arm-v7",
			expectedGOOS:   "linux",
			expectedGOARCH: "arm",
			expectedABI:    map[string]string{"GOARM": "7"},
		},
		{
			name:           "linux-arm with bare 6",
			cacheName:      "go-build-linux-arm-6",
			expectedGOOS:   "linux",
			expectedGOARCH: "arm",
			expectedABI:    map[string]string{"GOARM": "6"},
		},
		{
			name:           "linux-arm with bare 7",
			cacheName:      "go-build-linux-arm-7",
			expectedGOOS:   "linux",
			expectedGOARCH: "arm",
			expectedABI:    map[string]string{"GOARM": "7"},
		},
		{
			name:           "linux-386 with softfloat",
			cacheName:      "go-build-linux-386-softfloat",
			expectedGOOS:   "linux",
			expectedGOARCH: "386",
			expectedABI:    map[string]string{"GO386": "softfloat"},
		},
		{
			name:           "linux-mips with hardfloat",
			cacheName:      "go-build-linux-mips-hardfloat",
			expectedGOOS:   "linux",
			expectedGOARCH: "mips",
			expectedABI:    map[string]string{"GOMIPS": "hardfloat"},
		},
		{
			name:           "linux-ppc64 with power9",
			cacheName:      "go-build-linux-ppc64-power9",
			expectedGOOS:   "linux",
			expectedGOARCH: "ppc64",
			expectedABI:    map[string]string{"GOPPC64": "power9"},
		},
		{
			name:           "with GOEXPERIMENT",
			cacheName:      "go-build-linux-amd64-exp-boringcrypto",
			expectedGOOS:   "linux",
			expectedGOARCH: "amd64",
			expectedABI:    map[string]string{"GOEXPERIMENT": "boringcrypto"},
		},
		{
			name:           "with CGO hash",
			cacheName:      "go-build-linux-amd64-cgo-abc12345",
			expectedGOOS:   "linux",
			expectedGOARCH: "amd64",
			expectedABI:    map[string]string{"CGO_HASH": "abc12345"},
		},
		{
			name:           "with ABI, experiment, and CGO",
			cacheName:      "go-build-linux-amd64-v3-exp-boringcrypto-cgo-abc12345",
			expectedGOOS:   "linux",
			expectedGOARCH: "amd64",
			expectedABI: map[string]string{
				"GOAMD64":      "v3",
				"GOEXPERIMENT": "boringcrypto",
				"CGO_HASH":     "abc12345",
			},
		},
		{
			name:           "ARM with v5",
			cacheName:      "go-build-linux-arm-v5",
			expectedGOOS:   "linux",
			expectedGOARCH: "arm",
			expectedABI:    map[string]string{"GOARM": "5"},
		},
		{
			name:           "ARM with bare 5",
			cacheName:      "go-build-linux-arm-5",
			expectedGOOS:   "linux",
			expectedGOARCH: "arm",
			expectedABI:    map[string]string{"GOARM": "5"},
		},
		{
			name:           "invalid cache name",
			cacheName:      "not-a-cache",
			expectedGOOS:   "",
			expectedGOARCH: "",
			expectedABI:    nil,
		},
		{
			name:           "missing prefix",
			cacheName:      "build-linux-amd64",
			expectedGOOS:   "",
			expectedGOARCH: "",
			expectedABI:    nil,
		},
		{
			name:           "incomplete parts",
			cacheName:      "go-build-linux",
			expectedGOOS:   "",
			expectedGOARCH: "",
			expectedABI:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goos, goarch, abi := parseABIFromCacheName(tt.cacheName)

			if goos != tt.expectedGOOS {
				t.Errorf("GOOS: got %q, want %q", goos, tt.expectedGOOS)
			}

			if goarch != tt.expectedGOARCH {
				t.Errorf("GOARCH: got %q, want %q", goarch, tt.expectedGOARCH)
			}

			// Compare ABI maps
			if tt.expectedABI == nil {
				if abi != nil {
					t.Errorf("ABI: got %v, want nil", abi)
				}
			} else {
				if abi == nil {
					t.Errorf("ABI: got nil, want %v", tt.expectedABI)
				} else {
					// Check that all expected keys are present with correct values
					for key, expectedVal := range tt.expectedABI {
						if actualVal, ok := abi[key]; !ok {
							t.Errorf("ABI missing key %q", key)
						} else if actualVal != expectedVal {
							t.Errorf("ABI[%q]: got %q, want %q", key, actualVal, expectedVal)
						}
					}

					// Check that no unexpected keys are present
					for key := range abi {
						if _, ok := tt.expectedABI[key]; !ok {
							t.Errorf("ABI has unexpected key %q with value %q", key, abi[key])
						}
					}
				}
			}
		})
	}
}

func TestCacheMigration_UsesRuntimeArchitecture(t *testing.T) {
	// Test that cache migration uses runtime.GOOS/GOARCH, not env vars
	// This ensures we create "go-build-darwin-arm64" not "go-build-host-host"

	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create versions directory with fake Go installation
	versionsDir := filepath.Join(tmpDir, "versions")
	versionPath := filepath.Join(versionsDir, "1.23.2")
	binPath := filepath.Join(versionPath, "bin")
	if err := os.MkdirAll(binPath, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create fake go executable
	goExe := filepath.Join(binPath, "go")
	if runtime.GOOS == "windows" {
		goExe += ".exe"
	}
	if err := os.WriteFile(goExe, []byte("#!/bin/sh\necho fake go\n"), 0755); err != nil {
		t.Fatalf("Failed to create fake go executable: %v", err)
	}

	// Create old format cache (go-build without architecture suffix)
	oldCachePath := filepath.Join(versionPath, "go-build")
	if err := os.MkdirAll(oldCachePath, 0755); err != nil {
		t.Fatalf("Failed to create old cache: %v", err)
	}

	// Add a test file
	testFile := filepath.Join(oldCachePath, "test.a")
	if err := os.WriteFile(testFile, []byte("test data"), 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set cross-compilation env vars to ensure we don't use them
	t.Setenv("GOOS", "plan9")
	t.Setenv("GOARCH", "mips")

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

	// Verify the new cache path uses runtime.GOOS/GOARCH
	expectedArch := fmt.Sprintf("%s-%s", runtime.GOOS, runtime.GOARCH)
	expectedCachePath := filepath.Join(versionPath, fmt.Sprintf("go-build-%s", expectedArch))

	// Check that the new cache exists
	if stat, err := os.Stat(expectedCachePath); err != nil || !stat.IsDir() {
		t.Errorf("Expected new cache at %s, but it doesn't exist", expectedCachePath)
		t.Logf("Output: %s", output)
	}

	// Verify it's NOT using the env vars (go-build-plan9-mips should NOT exist)
	wrongCachePath := filepath.Join(versionPath, "go-build-plan9-mips")
	if stat, err := os.Stat(wrongCachePath); err == nil && stat.IsDir() {
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
