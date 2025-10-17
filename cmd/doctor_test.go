package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorCommand_BasicRun(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create basic directory structure
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

	// Verify various checks are mentioned
	checkNames := []string{
		"goenv binary",
		"GOENV_ROOT directory",
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

	// Create complete directory structure
	shimsDir := filepath.Join(tmpDir, "shims")
	versionsDir := filepath.Join(tmpDir, "versions")
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
	if err := os.WriteFile(goBinary, []byte("#!/bin/bash\necho go1.21.0\n"), 0755); err != nil {
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

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	err := doctorCmd.RunE(doctorCmd, []string{})

	// Should return error due to missing root
	if err == nil {
		t.Error("Expected error when GOENV_ROOT doesn't exist")
	}

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

	// Check for expected formatting elements
	formatElements := []string{
		"🔍", // Magnifying glass
		"📋", // Clipboard
		"Summary:",
		"OK", // or "ok" in summary
	}

	for _, element := range formatElements {
		if !strings.Contains(output, element) {
			t.Errorf("Expected format element %q in output", element)
		}
	}
}

func TestDoctorCommand_WithCache(t *testing.T) {
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

	buf := new(bytes.Buffer)
	doctorCmd.SetOut(buf)
	doctorCmd.SetErr(buf)

	err := doctorCmd.RunE(doctorCmd, []string{})

	output := buf.String()

	// Should show summary with error count
	if !strings.Contains(output, "error") {
		t.Errorf("Expected error count in summary, got: %s", output)
	}

	// Should return error
	if err == nil {
		t.Error("Expected command to return error when errors are found")
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
	if err := os.WriteFile(goBinary, []byte("#!/bin/bash\necho go1.21.0\n"), 0755); err != nil {
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
	t.Setenv("SHELL", "/bin/bash")
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
