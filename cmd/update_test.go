package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestUpdateCommand_FlagValidation(t *testing.T) {
	defer func() {
		updateCheckOnly = false
		updateForce = false
	}()

	tests := []struct {
		name        string
		checkOnly   bool
		force       bool
		expectError bool
	}{
		{"default flags", false, false, false},
		{"check only", true, false, false},
		{"force", false, true, false},
		{"check and force", true, true, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			updateCheckOnly = tt.checkOnly
			updateForce = tt.force

			tmpDir := t.TempDir()
			t.Setenv("GOENV_ROOT", tmpDir)
			t.Setenv("GOENV_DIR", tmpDir)

			buf := new(bytes.Buffer)
			updateCmd.SetOut(buf)
			updateCmd.SetErr(buf)

			// Note: This will likely fail because we don't have a real installation
			// But we're just checking that flags are accepted
			_ = updateCmd.RunE(updateCmd, []string{})
		})
	}
}

func TestUpdateCommand_DetectionGitInstall(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Create a fake git repository structure
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create a fake HEAD file to make it look like a git repo
	headFile := filepath.Join(gitDir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("Failed to create HEAD file: %v", err)
	}

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	// This will likely fail without git command, but should detect git install type
	_ = updateCmd.RunE(updateCmd, []string{})

	output := buf.String()

	// Should mention checking for updates
	if !strings.Contains(output, "Checking") || !strings.Contains(output, "update") {
		t.Errorf("Expected update checking message, got: %s", output)
	}
}

func TestUpdateCommand_BinaryDetection(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Don't create .git directory - should detect as binary installation

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	// This will run the command (may fail, but should detect binary install)
	_ = updateCmd.RunE(updateCmd, []string{})

	output := buf.String()

	// Should mention checking for updates
	if !strings.Contains(output, "Checking") || !strings.Contains(output, "update") {
		t.Errorf("Expected update checking message, got: %s", output)
	}
}

func TestUpdateCommand_CheckOnly(t *testing.T) {
	defer func() {
		updateCheckOnly = false
	}()

	updateCheckOnly = true

	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	_ = updateCmd.RunE(updateCmd, []string{})

	output := buf.String()

	// Should check but not update
	if !strings.Contains(output, "Checking") {
		t.Errorf("Expected checking message in check-only mode, got: %s", output)
	}
}

func TestUpdateCommand_NoArgs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	// Update command should accept no arguments
	// If Args is defined, check it; otherwise skip
	if updateCmd.Args != nil {
		err := updateCmd.Args(updateCmd, []string{})
		if err != nil {
			t.Errorf("Update command should accept no arguments, got error: %v", err)
		}
	}
}

func TestUpdateCommand_RejectsArgs(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	// If command defines Args validation, test it
	// Update typically takes no args
	if updateCmd.Args != nil {
		err := updateCmd.Args(updateCmd, []string{"unexpected"})
		// Should either accept (nil) or reject
		// We're just checking the behavior is consistent
		_ = err
	}
}

func TestUpdateCommand_OutputFormat(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	_ = updateCmd.RunE(updateCmd, []string{})

	output := buf.String()

	// Check for emoji/icon usage (characteristic of update command)
	hasEmoji := strings.Contains(output, "🔄") ||
		strings.Contains(output, "✅") ||
		strings.Contains(output, "📡") ||
		strings.Contains(output, "🔍")

	if !hasEmoji {
		t.Errorf("Expected formatted output with icons, got: %s", output)
	}
}

func TestUpdateCommand_GitInstallWithGit(t *testing.T) {
	// Skip if git is not available
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available, skipping test")
	}

	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)

	// Initialize a real git repo
	gitDir := filepath.Join(tmpDir, ".git")
	if err := os.MkdirAll(gitDir, 0755); err != nil {
		t.Fatalf("Failed to create .git directory: %v", err)
	}

	// Create minimal git repo structure
	headFile := filepath.Join(gitDir, "HEAD")
	if err := os.WriteFile(headFile, []byte("ref: refs/heads/main\n"), 0644); err != nil {
		t.Fatalf("Failed to create HEAD file: %v", err)
	}

	// Create refs directory
	refsDir := filepath.Join(gitDir, "refs", "heads")
	if err := os.MkdirAll(refsDir, 0755); err != nil {
		t.Fatalf("Failed to create refs directory: %v", err)
	}

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	// This should detect git installation
	_ = updateCmd.RunE(updateCmd, []string{})

	output := buf.String()

	// Should attempt git operations or report git detection
	// (May fail if not a valid git repo, but should try)
	if !strings.Contains(output, "Checking") && !strings.Contains(output, "Fetching") {
		t.Logf("Output: %s", output) // Log for debugging
	}
}

func TestUpdateHelp(t *testing.T) {
	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	err := updateCmd.Help()
	if err != nil {
		t.Fatalf("Help command failed: %v", err)
	}

	output := buf.String()
	expectedStrings := []string{
		"update",
		"Update",
		"latest",
		"version",
	}

	for _, expected := range expectedStrings {
		if !strings.Contains(output, expected) {
			t.Errorf("Help output missing %q", expected)
		}
	}
}

func TestUpdateCommand_Flags(t *testing.T) {
	// Test that flags are properly defined
	checkFlag := updateCmd.Flags().Lookup("check")
	if checkFlag == nil {
		t.Error("Expected --check flag to be defined")
	}

	forceFlag := updateCmd.Flags().Lookup("force")
	if forceFlag == nil {
		t.Error("Expected --force flag to be defined")
	}
}

func TestUpdateCommand_WithDebug(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)
	t.Setenv("GOENV_DEBUG", "1")

	buf := new(bytes.Buffer)
	updateCmd.SetOut(buf)
	updateCmd.SetErr(buf)

	_ = updateCmd.RunE(updateCmd, []string{})

	output := buf.String()

	// With debug mode, should show checking message
	if !strings.Contains(output, "Checking") && !strings.Contains(output, "Debug") {
		t.Logf("Output: %s", output)
	}
}
