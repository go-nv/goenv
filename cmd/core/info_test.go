package core

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInfoCommand(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create a fake installed version
	installedVersion := "1.23.2"
	cmdtest.CreateMockGoVersion(t, tmpDir, installedVersion)

	// Set up test config
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

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
				assert.NotEmpty(t, output, "Expected output, got empty string")
				// Check for key information
				assert.True(t, bytes.Contains([]byte(output), []byte("1.23.2")), "Output should contain version number")
			},
		},
		{
			name:    "not installed version",
			version: "1.20.5",
			checkOutput: func(t *testing.T, output string) {
				assert.NotEmpty(t, output, "Expected output, got empty string")
				// Should still show info even if not installed
				assert.True(t, bytes.Contains([]byte(output), []byte("1.20.5")), "Output should contain version number")
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
			assert.False(t, tt.expectError && err == nil, "Expected error but got none")
			assert.False(t, !tt.expectError && err != nil)

			// Check output
			if tt.checkOutput != nil {
				tt.checkOutput(t, buf.String())
			}
		})
	}
}

func TestInfoJSONOutput(t *testing.T) {
	var err error
	// Create temporary test directory
	tmpDir := t.TempDir()

	// Create a fake installed version
	installedVersion := "1.23.2"
	cmdtest.CreateMockGoVersion(t, tmpDir, installedVersion)

	// Set up test config
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	originalStdout := os.Stdout
	defer func() {
		os.Stdout = originalStdout
	}()

	// Redirect stdout to capture JSON output
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Set JSON flag
	infoFlags.json = true
	defer func() { infoFlags.json = false }()

	// Run command directly
	err = runInfo(&cobra.Command{}, []string{"1.23.2"})

	// Close writer and restore stdout
	w.Close()
	os.Stdout = originalStdout

	require.NoError(t, err, "Command failed")

	// Read captured output
	var buf bytes.Buffer
	buf.ReadFrom(r)
	output := buf.Bytes()

	// Parse JSON output
	var result map[string]interface{}
	decoder := json.NewDecoder(bytes.NewReader(output))
	err = decoder.Decode(&result)
	require.NoError(t, err, "Failed to parse JSON: \\nOutput")

	// Verify JSON structure
	assert.Equal(t, "1.23.2", result["version"], "Expected version 1.23.2")
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
	var err error
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
		err = utils.EnsureDirWithContext(filepath.Dir(path), "create test directory")
		require.NoError(t, err, "Failed to create dir")
		data := make([]byte, tf.size)
		testutil.WriteTestFile(t, path, data, utils.PermFileDefault)
		expectedSize += tf.size
	}

	// Calculate size
	size, err := calculateDirSize(tmpDir)
	require.NoError(t, err, "calculateDirSize failed")

	assert.Equal(t, expectedSize, size, "Expected size")
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
			assert.Equal(t, tt.expected, result, "formatSize() = %v", tt.bytes)
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
			assert.Equal(t, tt.expected, result, "utils.ExtractMajorMinor() = %v", tt.version)
		})
	}
}

func TestInfoCommandWithConfig(t *testing.T) {
	// This test ensures the command works with the config system
	tmpDir := t.TempDir()
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Load config
	cfg := config.Load()
	require.NotNil(t, cfg, "Failed to load config")

	// Verify versions dir is set correctly
	versionsDir := cfg.VersionsDir()
	assert.NotEmpty(t, versionsDir, "Versions directory should not be empty")
}

func BenchmarkCalculateDirSize(b *testing.B) {
	// Create temporary test directory with some files
	tmpDir := b.TempDir()
	for i := 0; i < 100; i++ {
		path := filepath.Join(tmpDir, "file", string(rune(i)))
		_ = utils.EnsureDirWithContext(filepath.Dir(path), "create test directory")
		testutil.WriteTestFile(b, path, make([]byte, 1024), utils.PermFileDefault)
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
