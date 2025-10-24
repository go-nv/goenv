package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestDoctorCommand_BasicRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create basic directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

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
		if !strings.Contains(output, expected) {
			t.Errorf("Expected %q in output, got: %s", expected, output)
		}
	}
}

func TestDoctorCommand_ChecksExecuted(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

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
		"PATH configuration",
		"Shims directory",
	}

	for _, checkName := range checkNames {
		if !strings.Contains(output, checkName) {
			t.Errorf("Expected check %q to be mentioned in output", checkName)
		}
	}
}

func TestDoctorCommand_WithInstalledVersion(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create complete directory structure
	rootBinDir := filepath.Join(tmpDir, "bin")
	shimsDir := filepath.Join(tmpDir, "shims")
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(rootBinDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	// Create a fake installed version
	versionDir := filepath.Join(versionsDir, "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Create mock go binary
	goBinary := filepath.Join(binDir, "go")
	var content string
	if runtime.GOOS == "windows" {
		goBinary += ".bat"
		content = "@echo off\necho go1.21.0\n"
	} else {
		content = "#!/bin/bash\necho go1.21.0\n"
	}
	if err := os.WriteFile(goBinary, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock go binary: %v", err)
	}

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention installed versions
	if !strings.Contains(output, "Installed") && !strings.Contains(output, "version") {
		t.Errorf("Expected installed versions check in output, got: %s", output)
	}
}

func TestDoctorCommand_MissingGOENV_ROOT(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")
	t.Setenv("GOENV_ROOT", nonExistentDir)
	t.Setenv("GOENV_DIR", nonExistentDir)

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
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 when GOENV_ROOT doesn't exist, got: %d", exitCode)
	}

	// Error may or may not be returned (before exit is called) - doctor now calls os.Exit
	// so we just check that output contains the error
	t.Logf("Error returned: %v", err)

	output := buf.String()

	// Should show error for GOENV_ROOT
	if !strings.Contains(output, "GOENV_ROOT") {
		t.Errorf("Expected GOENV_ROOT error in output, got: %s", output)
	}
}

func TestDoctorCommand_OutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Ensure emojis are enabled for this test
	os.Unsetenv("GOENV_PLAIN")
	os.Unsetenv("NO_COLOR")

	// Create basic structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

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
		if !strings.Contains(output, element) {
			t.Errorf("Expected format element %q in output", element)
		}
	}

	// Just verify that output is not empty and contains diagnostic info
	if len(output) == 0 {
		t.Error("Expected non-empty output")
	}
	if !strings.Contains(output, "goenv") {
		t.Error("Expected output to contain 'goenv'")
	}
}

func TestDoctorCommand_WithCache(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Clear GOENV_VERSION to avoid picking up .go-version from repo
	t.Setenv("GOENV_VERSION", "system")

	// Add GOENV_ROOT/bin to PATH to avoid PATH configuration errors
	oldPath := os.Getenv("PATH")
	t.Setenv("PATH", filepath.Join(tmpDir, "bin")+string(os.PathListSeparator)+oldPath)

	// Override exit to prevent test termination
	oldExit := doctorExit
	doctorExit = func(code int) {}
	defer func() { doctorExit = oldExit }()

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "bin"), 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	// Create cache file
	cacheFile := filepath.Join(tmpDir, "cache", "releases.json")
	if err := os.MkdirAll(filepath.Dir(cacheFile), 0755); err != nil {
		t.Fatalf("Failed to create cache directory: %v", err)
	}
	if err := os.WriteFile(cacheFile, []byte("{}"), 0644); err != nil {
		t.Fatalf("Failed to create cache file: %v", err)
	}

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention cache check
	if !strings.Contains(output, "Cache") || !strings.Contains(output, "cache") {
		t.Errorf("Expected cache check in output, got: %s", output)
	}
}

func TestDoctorCommand_ErrorCount(t *testing.T) {
	tmpDir := t.TempDir()
	nonExistentDir := filepath.Join(tmpDir, "nonexistent")
	t.Setenv("GOENV_ROOT", nonExistentDir)
	t.Setenv("GOENV_DIR", nonExistentDir)

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
	if !strings.Contains(output, "error") {
		t.Errorf("Expected error count in summary, got: %s", output)
	}

	// Doctor should call exit(1) when errors are found
	if exitCode != 1 {
		t.Errorf("Expected exit code 1 when errors are found, got: %d", exitCode)
	}
}

func TestDoctorCommand_SuccessScenario(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)
	t.Setenv("PATH", filepath.Join(tmpDir, "shims")+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Create complete directory structure
	shimsDir := filepath.Join(tmpDir, "shims")
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(shimsDir, 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	// Create a version
	versionDir := filepath.Join(versionsDir, "1.21.0")
	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	goBinary := filepath.Join(binDir, "go")
	var content string
	if runtime.GOOS == "windows" {
		goBinary += ".bat"
		content = "@echo off\necho go1.21.0\n"
	} else {
		content = "#!/bin/bash\necho go1.21.0\n"
	}
	if err := os.WriteFile(goBinary, []byte(content), 0755); err != nil {
		t.Fatalf("Failed to create mock go binary: %v", err)
	}

	// Set current version
	versionFile := filepath.Join(tmpDir, "version")
	if err := os.WriteFile(versionFile, []byte("1.21.0\n"), 0644); err != nil {
		t.Fatalf("Failed to create version file: %v", err)
	}

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// In success scenario, we should see OK indicators
	if !strings.Contains(output, "OK") && !strings.Contains(output, "✅") {
		t.Errorf("Expected success indicators in output, got: %s", output)
	}

	// Should show summary
	if !strings.Contains(output, "Summary:") {
		t.Errorf("Expected summary in output, got: %s", output)
	}
}

func TestDoctorHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	err := doctorCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"doctor",
		"installation",
		"configuration",
		"verifies",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestDoctorCommand_ShellDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Set a specific shell
	originalShell := os.Getenv("SHELL")
	if runtime.GOOS == "windows" {
		t.Setenv("SHELL", "powershell")
	} else {
		t.Setenv("SHELL", "/bin/bash")
	}
	defer func() {
		if originalShell != "" {
			os.Setenv("SHELL", originalShell)
		}
	}()

	// Create basic structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention shell in output
	if !strings.Contains(output, "Shell") || !strings.Contains(output, "shell") {
		t.Errorf("Expected shell check in output, got: %s", output)
	}
}

func TestDoctorCommand_NoVersions(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create structure but NO versions
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should mention installed versions (even if none)
	if !strings.Contains(output, "version") || !strings.Contains(output, "Version") {
		t.Errorf("Expected version check in output, got: %s", output)
	}
}

// Test the checkGoToolchain function
func TestCheckGoToolchain(t *testing.T) {
	tests := []struct {
		name           string
		gotoolchain    string
		expectedStatus string
		shouldContain  string
	}{
		{
			name:           "GOTOOLCHAIN not set",
			gotoolchain:    "",
			expectedStatus: "ok",
			shouldContain:  "not set",
		},
		{
			name:           "GOTOOLCHAIN=auto (warning)",
			gotoolchain:    "auto",
			expectedStatus: "warning",
			shouldContain:  "can cause issues",
		},
		{
			name:           "GOTOOLCHAIN=local (recommended)",
			gotoolchain:    "local",
			expectedStatus: "ok",
			shouldContain:  "recommended",
		},
		{
			name:           "GOTOOLCHAIN with specific version",
			gotoolchain:    "go1.23.2",
			expectedStatus: "warning",
			shouldContain:  "may interfere",
		},
		{
			name:           "GOTOOLCHAIN=local+auto",
			gotoolchain:    "local+auto",
			expectedStatus: "warning",
			shouldContain:  "may interfere",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set GOTOOLCHAIN environment variable
			if tt.gotoolchain != "" {
				t.Setenv("GOTOOLCHAIN", tt.gotoolchain)
			} else {
				os.Unsetenv("GOTOOLCHAIN")
			}

			result := checkGoToolchain()

			if result.status != tt.expectedStatus {
				t.Errorf("Expected status %q, got %q", tt.expectedStatus, result.status)
			}

			if !strings.Contains(result.message, tt.shouldContain) {
				t.Errorf("Expected message to contain %q, got %q", tt.shouldContain, result.message)
			}

			if result.name != "GOTOOLCHAIN setting" {
				t.Errorf("Expected name %q, got %q", "GOTOOLCHAIN setting", result.name)
			}

			// Warnings should have advice
			if result.status == "warning" && result.advice == "" {
				t.Error("Warning status should have advice")
			}
		})
	}
}

// Test the checkCacheIsolationEffectiveness function
func TestCheckCacheIsolationEffectiveness(t *testing.T) {
	tests := []struct {
		name           string
		setupVersion   string
		setupCache     bool
		setupOldCache  bool
		disableCache   bool
		expectedStatus string
		shouldContain  string
	}{
		{
			name:           "No managed version active",
			setupVersion:   "",
			expectedStatus: "ok",
			shouldContain:  "Not applicable",
		},
		{
			name:           "System version active",
			setupVersion:   "system",
			expectedStatus: "ok",
			shouldContain:  "Not applicable",
		},
		{
			name:           "Cache isolation disabled",
			setupVersion:   "1.21.0",
			disableCache:   true,
			expectedStatus: "ok",
			shouldContain:  "disabled",
		},
		{
			name:           "New cache will be created",
			setupVersion:   "1.21.0",
			setupCache:     false,
			setupOldCache:  false,
			expectedStatus: "ok",
			shouldContain:  "will be created",
		},
		{
			name:           "Architecture-aware cache exists",
			setupVersion:   "1.21.0",
			setupCache:     true,
			setupOldCache:  false,
			expectedStatus: "ok",
			shouldContain:  "go-build-", // Cache path will contain go-build-
		},
		{
			name:           "Old cache exists",
			setupVersion:   "1.21.0",
			setupCache:     false,
			setupOldCache:  true,
			expectedStatus: "warning",
			shouldContain:  "old-style cache",
		},
		{
			name:           "Both caches exist",
			setupVersion:   "1.21.0",
			setupCache:     true,
			setupOldCache:  true,
			expectedStatus: "warning", // Test setup doesn't create cache matching expected name (with CGO hash), so only old cache is found
			shouldContain:  "old-style cache",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()

			// Change to tmpDir to avoid picking up .go-version files from parent directories
			oldDir, _ := os.Getwd()
			defer os.Chdir(oldDir)
			if err := os.Chdir(tmpDir); err != nil {
				t.Fatalf("Failed to change directory: %v", err)
			}

			// Unset any existing GOENV variables to ensure isolation
			os.Unsetenv("GOENV_VERSION")

			t.Setenv("GOENV_ROOT", tmpDir)
			t.Setenv("GOENV_DIR", tmpDir)

			// Create basic structure
			versionsDir := filepath.Join(tmpDir, "versions")
			if err := os.MkdirAll(versionsDir, 0755); err != nil {
				t.Fatalf("Failed to create versions directory: %v", err)
			}

			// Setup version if needed
			if tt.setupVersion != "" && tt.setupVersion != "system" {
				versionDir := filepath.Join(versionsDir, tt.setupVersion)
				binDir := filepath.Join(versionDir, "bin")
				if err := os.MkdirAll(binDir, 0755); err != nil {
					t.Fatalf("Failed to create bin directory: %v", err)
				}

				// Create version file to set current version
				versionFile := filepath.Join(tmpDir, "version")
				if err := os.WriteFile(versionFile, []byte(tt.setupVersion+"\n"), 0644); err != nil {
					t.Fatalf("Failed to create version file: %v", err)
				}

				// Setup caches if requested
				if tt.setupCache {
					// Create architecture-aware cache
					goos := os.Getenv("GOOS")
					goarch := os.Getenv("GOARCH")
					if goos == "" {
						goos = "host"
					}
					if goarch == "" {
						goarch = "host"
					}
					cacheSuffix := "go-build-" + goos + "-" + goarch
					cacheDir := filepath.Join(versionDir, cacheSuffix)
					if err := os.MkdirAll(cacheDir, 0755); err != nil {
						t.Fatalf("Failed to create cache directory: %v", err)
					}
				}

				if tt.setupOldCache {
					// Create old-style cache
					oldCacheDir := filepath.Join(versionDir, "go-build")
					if err := os.MkdirAll(oldCacheDir, 0755); err != nil {
						t.Fatalf("Failed to create old cache directory: %v", err)
					}
				}
			} else if tt.setupVersion == "system" {
				// Set system version
				versionFile := filepath.Join(tmpDir, "version")
				if err := os.WriteFile(versionFile, []byte("system\n"), 0644); err != nil {
					t.Fatalf("Failed to create version file: %v", err)
				}
			}
			// If setupVersion is empty, don't create a version file at all

			// Set cache isolation env var if requested
			if tt.disableCache {
				t.Setenv("GOENV_DISABLE_GOCACHE", "1")
			} else {
				os.Unsetenv("GOENV_DISABLE_GOCACHE")
			}

			// Load config which will read from environment variables
			cfg := config.Load()
			result := checkCacheIsolationEffectiveness(cfg)

			if result.status != tt.expectedStatus {
				t.Errorf("Expected status %q, got %q", tt.expectedStatus, result.status)
			}

			if !strings.Contains(result.message, tt.shouldContain) {
				t.Errorf("Expected message to contain %q, got %q", tt.shouldContain, result.message)
			}

			if result.name != "Architecture-aware cache isolation" {
				t.Errorf("Expected name %q, got %q", "Architecture-aware cache isolation", result.name)
			}
		})
	}
}

// Test the checkRosetta function
func TestCheckRosetta(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Load config which will read from environment variables
	cfg := config.Load()

	result := checkRosetta(cfg)

	// Should always return a valid result
	if result.name != "Rosetta detection" {
		t.Errorf("Expected name %q, got %q", "Rosetta detection", result.name)
	}

	// Status should be one of ok, warning, or error
	validStatuses := []string{"ok", "warning", "error"}
	if !slices.Contains(validStatuses, result.status) {
		t.Errorf("Invalid status %q, expected one of %v", result.status, validStatuses)
	}

	// On non-macOS systems, should say "Not applicable"
	if runtime.GOOS != "darwin" {
		if !strings.Contains(result.message, "Not applicable") && !strings.Contains(result.message, "not macOS") {
			t.Errorf("Expected non-macOS message, got %q", result.message)
		}
	}
	// On macOS, the message depends on the actual system configuration
	// We can't reliably test specific outcomes without knowing the hardware
}

func TestDoctorCommand_EnvironmentDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

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
		if !strings.Contains(output, check) {
			t.Errorf("Expected environment check %q to be in output", check)
		}
	}

	// Should see either Native, Container, or WSL
	hasEnvironmentType := strings.Contains(output, "Native") ||
		strings.Contains(output, "Container") ||
		strings.Contains(output, "WSL")

	if !hasEnvironmentType {
		t.Error("Expected to see environment type (Native, Container, or WSL) in output")
	}

	// Should see filesystem type
	hasFilesystemType := strings.Contains(output, "Filesystem type:")

	if !hasFilesystemType {
		t.Error("Expected to see 'Filesystem type:' in output")
	}
}

func TestCheckEnvironment(t *testing.T) {
	cfg := config.Load()

	result := checkEnvironment(cfg)

	if result.name != "Runtime environment" {
		t.Errorf("Expected check name 'Runtime environment', got %s", result.name)
	}

	if result.status != "ok" && result.status != "warning" {
		t.Errorf("Expected status 'ok' or 'warning', got %s", result.status)
	}

	if result.message == "" {
		t.Error("Expected non-empty message")
	}

	// Message should contain environment description
	hasEnvType := strings.Contains(result.message, "Native") ||
		strings.Contains(result.message, "Container") ||
		strings.Contains(result.message, "WSL")

	if !hasEnvType {
		t.Errorf("Expected message to contain environment type, got: %s", result.message)
	}
}

func TestCheckGoenvRootFilesystem(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Root = tmpDir

	result := checkGoenvRootFilesystem(cfg)

	if result.name != "GOENV_ROOT filesystem" {
		t.Errorf("Expected check name 'GOENV_ROOT filesystem', got %s", result.name)
	}

	if result.message == "" {
		t.Error("Expected non-empty message")
	}

	// Message should mention filesystem type
	if !strings.Contains(result.message, "Filesystem type:") {
		t.Errorf("Expected message to contain 'Filesystem type:', got: %s", result.message)
	}
}

func TestCheckMacOSDeploymentTarget(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("macOS deployment target check only works on macOS")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{}
	cfg.Root = tmpDir

	// Create a fake version directory
	versionsDir := filepath.Join(tmpDir, "versions")
	versionDir := filepath.Join(versionsDir, "1.23.0")
	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin directory: %v", err)
	}

	// Set current version
	versionFile := filepath.Join(tmpDir, "version")
	if err := os.WriteFile(versionFile, []byte("1.23.0\n"), 0644); err != nil {
		t.Fatalf("Failed to create version file: %v", err)
	}

	result := checkMacOSDeploymentTarget(cfg)

	if result.name != "macOS deployment target" {
		t.Errorf("Expected check name 'macOS deployment target', got %s", result.name)
	}

	// Should be ok or have a message about not finding binary
	if result.status != "ok" {
		t.Logf("Status: %s, Message: %s", result.status, result.message)
	}
}

func TestCheckWindowsCompiler(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows compiler check only works on Windows")
	}

	cfg := config.Load()
	result := checkWindowsCompiler(cfg)

	if result.name != "Windows compiler" {
		t.Errorf("Expected check name 'Windows compiler', got %s", result.name)
	}

	if result.message == "" {
		t.Error("Expected non-empty message")
	}

	t.Logf("Status: %s, Message: %s", result.status, result.message)
}

func TestCheckWindowsARM64(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows ARM64 check only works on Windows")
	}

	cfg := config.Load()
	result := checkWindowsARM64(cfg)

	if result.name != "Windows ARM64/ARM64EC" {
		t.Errorf("Expected check name 'Windows ARM64/ARM64EC', got %s", result.name)
	}

	if result.message == "" {
		t.Error("Expected non-empty message")
	}

	// Should mention process mode
	if !strings.Contains(result.message, "Process mode:") {
		t.Errorf("Expected message to contain 'Process mode:', got: %s", result.message)
	}

	t.Logf("Status: %s, Message: %s", result.status, result.message)
}

func TestCheckLinuxKernelVersion(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux kernel check only works on Linux")
	}

	cfg := config.Load()
	result := checkLinuxKernelVersion(cfg)

	if result.name != "Linux kernel version" {
		t.Errorf("Expected check name 'Linux kernel version', got %s", result.name)
	}

	if result.message == "" {
		t.Error("Expected non-empty message")
	}

	// Should mention kernel version
	if !strings.Contains(result.message, "Kernel:") {
		t.Errorf("Expected message to contain 'Kernel:', got: %s", result.message)
	}

	t.Logf("Status: %s, Message: %s", result.status, result.message)
}

func TestPlatformSpecificChecksInDoctor(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create directory structure
	if err := os.MkdirAll(filepath.Join(tmpDir, "shims"), 0755); err != nil {
		t.Fatalf("Failed to create shims directory: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tmpDir, "versions"), 0755); err != nil {
		t.Fatalf("Failed to create versions directory: %v", err)
	}

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	_ = doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Check for platform-specific checks based on OS
	switch runtime.GOOS {
	case "darwin":
		if !strings.Contains(output, "macOS deployment target") {
			t.Error("Expected 'macOS deployment target' check on macOS")
		}
		if strings.Contains(output, "Windows compiler") {
			t.Error("Should not have Windows checks on macOS")
		}
		if strings.Contains(output, "Linux kernel") {
			t.Error("Should not have Linux checks on macOS")
		}

	case "windows":
		if !strings.Contains(output, "Windows compiler") {
			t.Error("Expected 'Windows compiler' check on Windows")
		}
		if !strings.Contains(output, "Windows ARM64/ARM64EC") {
			t.Error("Expected 'Windows ARM64/ARM64EC' check on Windows")
		}
		if strings.Contains(output, "macOS deployment target") {
			t.Error("Should not have macOS checks on Windows")
		}
		if strings.Contains(output, "Linux kernel") {
			t.Error("Should not have Linux checks on Windows")
		}

	case "linux":
		if !strings.Contains(output, "Linux kernel version") {
			t.Error("Expected 'Linux kernel version' check on Linux")
		}
		if strings.Contains(output, "macOS deployment target") {
			t.Error("Should not have macOS checks on Linux")
		}
		if strings.Contains(output, "Windows compiler") {
			t.Error("Should not have Windows checks on Linux")
		}
	}
}

// TestPlatformChecksCrossOSBehavior tests that platform-specific checks behave correctly
// when called on the "wrong" platform (e.g., Windows check on macOS should return nil/not applicable)
func TestPlatformChecksCrossOSBehavior(t *testing.T) {
	cfg := config.Load()

	// Test Windows checks on non-Windows platforms
	if runtime.GOOS != "windows" {
		t.Run("WindowsChecksOnNonWindows", func(t *testing.T) {
			result := checkWindowsCompiler(cfg)
			if result.id != "windows-compiler" {
				t.Errorf("Expected id 'windows-compiler', got %s", result.id)
			}
			if result.status != "ok" {
				t.Errorf("Expected status 'ok' (not applicable), got %s", result.status)
			}
			if !strings.Contains(result.message, "Not applicable") {
				t.Errorf("Expected 'Not applicable' message on non-Windows, got: %s", result.message)
			}
		})
	}

	// Test macOS checks on non-macOS platforms
	if runtime.GOOS != "darwin" {
		t.Run("MacOSChecksOnNonMacOS", func(t *testing.T) {
			result := checkMacOSDeploymentTarget(cfg)
			if result.id != "macos-deployment-target" {
				t.Errorf("Expected id 'macos-deployment-target', got %s", result.id)
			}
			// Check should handle non-macOS gracefully
			if result.status == "error" {
				t.Errorf("Check should not error on non-macOS, got status: %s", result.status)
			}
		})
	}

	// Test Linux checks on non-Linux platforms
	if runtime.GOOS != "linux" {
		t.Run("LinuxChecksOnNonLinux", func(t *testing.T) {
			result := checkLinuxKernelVersion(cfg)
			if result.id != "linux-kernel-version" {
				t.Errorf("Expected id 'linux-kernel-version', got %s", result.id)
			}
			if result.status != "ok" {
				t.Errorf("Expected status 'ok' (not applicable), got %s", result.status)
			}
			if !strings.Contains(result.message, "Not applicable") {
				t.Errorf("Expected 'Not applicable' message on non-Linux, got: %s", result.message)
			}
		})
	}
}
