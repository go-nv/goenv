package compliance

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
)

func TestInventoryGo_NoVersions(t *testing.T) {
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Unsetenv("GOENV_ROOT")

	// Create versions directory (but empty)
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatalf("Failed to create versions dir: %v", err)
	}

	// Run command
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No Go versions installed") {
		t.Errorf("Expected 'No Go versions installed', got: %s", output)
	}
}

func TestInventoryGo_WithVersions(t *testing.T) {
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Unsetenv("GOENV_ROOT")

	// Create mock Go installations
	versions := []string{"1.21.0", "1.22.0"}
	for _, version := range versions {
		versionPath := filepath.Join(tmpDir, "versions", version, "bin")
		if err := os.MkdirAll(versionPath, 0755); err != nil {
			t.Fatalf("Failed to create version dir: %v", err)
		}

		// Create mock go binary
		goBinary := filepath.Join(versionPath, "go")
		if runtime.GOOS == "windows" {
			goBinary += ".exe"
		}
		if err := os.WriteFile(goBinary, []byte("mock go binary"), 0755); err != nil {
			t.Fatalf("Failed to create go binary: %v", err)
		}
	}

	// Run command (text output)
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})
	inventoryJSON = false

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	output := buf.String()

	// Check output contains versions
	for _, version := range versions {
		if !strings.Contains(output, version) {
			t.Errorf("Expected version %s in output, got: %s", version, output)
		}
	}

	// Check for summary
	if !strings.Contains(output, "Total: 2 Go version(s)") {
		t.Errorf("Expected total count, got: %s", output)
	}
}

func TestInventoryGo_JSON(t *testing.T) {
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Unsetenv("GOENV_ROOT")

	// Create mock Go installation
	version := "1.21.0"
	versionPath := filepath.Join(tmpDir, "versions", version, "bin")
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	goBinary := filepath.Join(versionPath, "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	if err := os.WriteFile(goBinary, []byte("mock go binary"), 0755); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	// Run command (JSON output)
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})
	inventoryJSON = true
	inventoryChecksums = false

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Parse JSON
	var installations []goInstallation
	if err := json.Unmarshal(buf.Bytes(), &installations); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	// Verify
	if len(installations) != 1 {
		t.Fatalf("Expected 1 installation, got %d", len(installations))
	}

	install := installations[0]
	if install.Version != version {
		t.Errorf("Expected version %s, got %s", version, install.Version)
	}

	if install.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, install.OS)
	}

	if install.Arch != runtime.GOARCH {
		t.Errorf("Expected arch %s, got %s", runtime.GOARCH, install.Arch)
	}

	// Checksum should be empty (not requested)
	if install.SHA256 != "" {
		t.Errorf("Expected empty checksum, got %s", install.SHA256)
	}
}

func TestInventoryGo_WithChecksums(t *testing.T) {
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Unsetenv("GOENV_ROOT")

	// Create mock Go installation
	version := "1.21.0"
	versionPath := filepath.Join(tmpDir, "versions", version, "bin")
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	goBinary := filepath.Join(versionPath, "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	if err := os.WriteFile(goBinary, []byte("mock go binary"), 0755); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	// Run command with checksums
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})
	inventoryJSON = true
	inventoryChecksums = true

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	if err := cmd.RunE(cmd, []string{}); err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Parse JSON
	var installations []goInstallation
	if err := json.Unmarshal(buf.Bytes(), &installations); err != nil {
		t.Fatalf("Failed to parse JSON: %v", err)
	}

	install := installations[0]

	// Checksum should be present
	if install.SHA256 == "" {
		t.Error("Expected checksum to be computed")
	}

	// Verify it's a valid SHA256 (64 hex characters)
	if len(install.SHA256) != 64 {
		t.Errorf("Expected 64-character SHA256, got %d characters", len(install.SHA256))
	}
}

func TestCollectGoInstallation(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create mock version
	version := "1.21.0"
	versionPath := filepath.Join(cfg.VersionsDir(), version, "bin")
	if err := os.MkdirAll(versionPath, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	goBinary := filepath.Join(versionPath, "go")
	if runtime.GOOS == "windows" {
		goBinary += ".exe"
	}
	if err := os.WriteFile(goBinary, []byte("test content"), 0755); err != nil {
		t.Fatalf("Failed to create go binary: %v", err)
	}

	// Collect without checksum
	install := collectGoInstallation(cfg, version, false)

	if install.Version != version {
		t.Errorf("Expected version %s, got %s", version, install.Version)
	}

	if install.SHA256 != "" {
		t.Errorf("Expected no checksum, got %s", install.SHA256)
	}

	// Collect with checksum
	installWithChecksum := collectGoInstallation(cfg, version, true)

	if installWithChecksum.SHA256 == "" {
		t.Error("Expected checksum to be computed")
	}
}

func TestComputeSHA256(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	if err := os.WriteFile(testFile, content, 0644); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Compute checksum
	checksum, err := computeSHA256(testFile)
	if err != nil {
		t.Fatalf("Failed to compute checksum: %v", err)
	}

	// Verify it's valid hex
	if len(checksum) != 64 {
		t.Errorf("Expected 64-character checksum, got %d", len(checksum))
	}

	// Verify it's consistent
	checksum2, err := computeSHA256(testFile)
	if err != nil {
		t.Fatalf("Failed to compute checksum again: %v", err)
	}

	if checksum != checksum2 {
		t.Error("Checksums should be consistent")
	}
}
