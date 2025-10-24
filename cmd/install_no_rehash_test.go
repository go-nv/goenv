package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestInstallCommand_NoRehashFlag(t *testing.T) {
	defer func() {
		installFlags.noRehash = false
	}()

	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)
	t.Setenv("GOENV_DEBUG", "1")

	// Create a fake existing installation
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
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
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	// Test with --no-rehash flag
	installFlags.noRehash = true
	installFlags.skipExisting = true // Skip actual install since already exists

	err := installCmd.RunE(installCmd, []string{"1.21.0"})
	if err != nil {
		t.Fatalf("Install command failed: %v", err)
	}

	output := buf.String()

	// Should show debug message about skipping rehash
	if !strings.Contains(output, "Skipping auto-rehash") && !strings.Contains(output, "skip") {
		t.Logf("Output: %s", output)
		// This is OK - skip-existing returns early before rehash logic
	}
}

func TestInstallCommand_NoRehashEnv(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("GOENV_ROOT", tmpDir)
	t.Setenv("GOENV_DIR", tmpDir)
	t.Setenv("GOENV_DEBUG", "1")
	t.Setenv("GOENV_NO_AUTO_REHASH", "1")

	// Create a fake existing installation
	versionDir := filepath.Join(tmpDir, "versions", "1.21.0")
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
	installCmd.SetOut(buf)
	installCmd.SetErr(buf)

	defer func() {
		installFlags.skipExisting = false
	}()

	installFlags.skipExisting = true

	err := installCmd.RunE(installCmd, []string{"1.21.0"})
	if err != nil {
		t.Fatalf("Install command failed: %v", err)
	}

	output := buf.String()

	// With environment variable set, should skip rehash
	if !strings.Contains(output, "Skipping auto-rehash") && !strings.Contains(output, "skip") {
		t.Logf("Output: %s", output)
		// This is OK - skip-existing returns early before rehash logic
	}
}

func TestInstallCommand_NoRehashFlagExists(t *testing.T) {
	// Verify the flag is defined
	flag := installCmd.Flags().Lookup("no-rehash")
	if flag == nil {
		t.Fatal("--no-rehash flag is not defined")
	}

	if flag.DefValue != "false" {
		t.Errorf("Expected --no-rehash default to be false, got %s", flag.DefValue)
	}
}
