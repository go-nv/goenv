package core

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/spf13/cobra"
)

func TestInfoCommand(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatalf("Failed to create test versions dir: %v", err)
	}

	// Create a fake installed version
	installedVersion := "1.23.2"
	versionDir := filepath.Join(versionsDir, installedVersion)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	// Create some fake files to simulate installation
	binDir := filepath.Join(versionDir, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		t.Fatalf("Failed to create bin dir: %v", err)
	}
	testFile := filepath.Join(binDir, "go")
	if err := os.WriteFile(testFile, []byte("fake go binary"), 0755); err != nil {
		t.Fatalf("Failed to create test file: %v", err)
	}

	// Set up test config
	originalHome := os.Getenv("GOENV_ROOT")
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Setenv("GOENV_ROOT", originalHome)

	tests := []struct {
		name        string
		version     string
		expectError bool
		checkOutput func(t *testing.T, output string)
	}{
		{
			name:    "installed version",
			version: "1.23.2",
			checkOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Expected output, got empty string")
				}
				// Check for key information
				if !bytes.Contains([]byte(output), []byte("1.23.2")) {
					t.Error("Output should contain version number")
				}
			},
		},
		{
			name:    "not installed version",
			version: "1.20.5",
			checkOutput: func(t *testing.T, output string) {
				if output == "" {
					t.Error("Expected output, got empty string")
				}
				// Should still show info even if not installed
				if !bytes.Contains([]byte(output), []byte("1.20.5")) {
					t.Error("Output should contain version number")
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create fresh command
			cmd := &cobra.Command{
				Use:  "info",
				RunE: runInfo,
			}
			cmd.SetArgs([]string{tt.version})

			// Capture output
			var buf bytes.Buffer
			cmd.SetOut(&buf)
			cmd.SetErr(&buf)

			// Run command
			err := cmd.Execute()
			if tt.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, buf.String())
			}
		})
	}
}

func TestInfoJSONOutput(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	versionsDir := filepath.Join(tmpDir, "versions")
	if err := os.MkdirAll(versionsDir, 0755); err != nil {
		t.Fatalf("Failed to create test versions dir: %v", err)
	}

	// Create a fake installed version
	installedVersion := "1.23.2"
	versionDir := filepath.Join(versionsDir, installedVersion)
	if err := os.MkdirAll(versionDir, 0755); err != nil {
		t.Fatalf("Failed to create version dir: %v", err)
	}

	// Set up test config
	originalHome := os.Getenv("GOENV_ROOT")
	originalStdout := os.Stdout
	os.Setenv("GOENV_ROOT", tmpDir)
	defer func() {
		os.Setenv("GOENV_ROOT", originalHome)
		os.Stdout = originalStdout
	}()

	// Redirect stdout to capture JSON output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set JSON flag
	infoFlags.json = true
	defer func() { infoFlags.json = false }()

	// Run command directly
	err := runInfo(&cobra.Command{}, []string{"1.23.2"})

	// Close writer and restore stdout
	w.Close()
	os.Stdout = originalStdout

	if err != nil {
		t.Fatalf("Command failed: %v", err)
	}

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.Bytes()

	// Parse JSON output
	var result map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(output))
	if err := decoder.Decode(&result); err != nil {
		t.Fatalf("Failed to parse JSON: %v\nOutput: %s", err, buf.String())
	}

	// Verify JSON structure
	if result["version"] != "1.23.2" {
		t.Errorf("Expected version 1.23.2, got %v", result["version"])
	}
	if _, ok := result["installed"]; !ok {
		t.Error("JSON should contain 'installed' field")
	}
	if _, ok := result["release_url"]; !ok {
		t.Error("JSON should contain 'release_url' field")
	}
	if _, ok := result["download_url"]; !ok {
		t.Error("JSON should contain 'download_url' field")
	}
}

func TestCalculateDirSize(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create test files
	testFiles := []struct {
		name string
		size int64
	}{
		{"file1.txt", 100},
		{"file2.txt", 200},
		{"subdir/file3.txt", 300},
	}

	expectedSize := int64(0)
	for _, tf := range testFiles {
		path := filepath.Join(tmpDir, tf.name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatalf("Failed to create dir: %v", err)
		}
		data := make([]byte, tf.size)
		if err := os.WriteFile(path, data, 0644); err != nil {
			t.Fatalf("Failed to create file: %v", err)
		}
		expectedSize += tf.size
	}

	// Calculate size
	size, err := calculateDirSize(tmpDir)
	if err != nil {
		t.Fatalf("calculateDirSize failed: %v", err)
	}

	if size != expectedSize {
		t.Errorf("Expected size %d, got %d", expectedSize, size)
	}
}

func TestFormatSize(t *testing.T) {
	tests := []struct {
		bytes    int64
		expected string
	}{
		{0, "0 B"},
		{512, "512 B"},
		{1023, "1023 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1572864, "1.5 MB"},
		{431600473, "411.6 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := formatSize(tt.bytes)
			if result != tt.expected {
				t.Errorf("formatSize(%d) = %s, want %s", tt.bytes, result, tt.expected)
			}
		})
	}
}

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		version  string
		expected string
	}{
		{"1.21.5", "1.21"},
		{"1.21", "1.21"},
		{"1.20.0", "1.20"},
		{"1.23.2", "1.23"},
		{"go1.21.5", "1.21"},   // With prefix
		{"1.21.5-rc1", "1.21"}, // With suffix
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.version, func(t *testing.T) {
			result := utils.ExtractMajorMinor(tt.version)
			if result != tt.expected {
				t.Errorf("utils.ExtractMajorMinor(%q) = %q, want %q", tt.version, result, tt.expected)
			}
		})
	}
}

func TestInfoCommandWithConfig(t *testing.T) {
	// This test ensures the command works with the config system
	tmpDir := t.TempDir()
	originalHome := os.Getenv("GOENV_ROOT")
	os.Setenv("GOENV_ROOT", tmpDir)
	defer os.Setenv("GOENV_ROOT", originalHome)

	// Load config
	cfg := config.Load()
	if cfg == nil {
		t.Fatal("Failed to load config")
	}

	// Verify versions dir is set correctly
	versionsDir := cfg.VersionsDir()
	if versionsDir == "" {
		t.Error("Versions directory should not be empty")
	}
}

func BenchmarkCalculateDirSize(b *testing.B) {
	// Create temporary test directory with some files
	tmpDir := b.TempDir()
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, "file", string(rune(i)))
		os.MkdirAll(filepath.Dir(path), 0755)
		os.WriteFile(path, make([]byte, 1024), 0644)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		calculateDirSize(tmpDir)
	}
}

func BenchmarkFormatSize(b *testing.B) {
	sizes := []int64{512, 1024, 1048576, 431600473, 1073741824}
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		for _, size := range sizes {
			formatSize(size)
		}
	}
}
