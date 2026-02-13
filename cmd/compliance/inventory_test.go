package compliance

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/platform"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInventoryGo_NoVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create versions directory (but empty)
	versionsDir := filepath.Join(tmpDir, "versions")
	err = utils.EnsureDirWithContext(versionsDir, "create test directory")
	require.NoError(t, err, "Failed to create versions dir")

	// Run command
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err, "Command failed")

	output := buf.String()
	assert.Contains(t, output, "No Go versions installed", "Expected 'No Go versions installed' %v", output)
}

func TestInventoryGo_WithVersions(t *testing.T) {
	var err error
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create config for helper methods
	cfg := &config.Config{Root: tmpDir}

	// Create mock Go installations
	versions := []string{"1.21.0", "1.22.0"}
	for _, version := range versions {
		versionPath := cfg.VersionBinDir(version)
		err = utils.EnsureDirWithContext(versionPath, "create test directory")
		require.NoError(t, err, "Failed to create version dir")

		// Create mock go binary
		goBinary := cfg.VersionGoBinary(version)
		if utils.IsWindows() {
			goBinary += ".exe"
		}
		testutil.WriteTestFile(t, goBinary, []byte("mock go binary"), utils.PermFileExecutable)
	}

	// Run command (text output)
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})
	inventoryJSON = false

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err, "Command failed")

	output := buf.String()

	// Check output contains versions
	for _, version := range versions {
		assert.Contains(t, output, version, "Expected version in output %v %v", version, output)
	}

	// Check for summary
	assert.Contains(t, output, "Total: 2 Go version(s)", "Expected total count %v", output)
}

func TestInventoryGo_JSON(t *testing.T) {
	var err error
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create config for helper methods
	cfg := &config.Config{Root: tmpDir}

	// Create mock Go installation
	version := "1.21.0"
	versionPath := cfg.VersionBinDir(version)
	err = utils.EnsureDirWithContext(versionPath, "create test directory")
	require.NoError(t, err, "Failed to create version dir")

	goBinary := cfg.VersionGoBinary(version)
	if utils.IsWindows() {
		goBinary += ".exe"
	}
	testutil.WriteTestFile(t, goBinary, []byte("mock go binary"), utils.PermFileExecutable)

	// Run command (JSON output)
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})
	inventoryJSON = true
	inventoryChecksums = false

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err, "Command failed")

	// Parse JSON
	var installations []goInstallation
	err = json.Unmarshal(buf.Bytes(), &installations)
	require.NoError(t, err, "Failed to parse JSON")

	// Verify
	if len(installations) != 1 {
		t.Fatalf("Expected 1 installation, got %d", len(installations))
	}

	install := installations[0]
	assert.Equal(t, version, install.Version, "Expected version")

	assert.Equal(t, platform.OS(), install.OS, "Expected OS")

	assert.Equal(t, platform.Arch(), install.Arch, "Expected arch")

	// Checksum should be empty (not requested)
	assert.Empty(t, install.SHA256, "Expected empty checksum")
}

func TestInventoryGo_WithChecksums(t *testing.T) {
	var err error
	tmpDir := t.TempDir()

	// Set GOENV_ROOT
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	// Create config for helper methods
	cfg := &config.Config{Root: tmpDir}

	// Create mock Go installation
	version := "1.21.0"
	versionPath := cfg.VersionBinDir(version)
	err = utils.EnsureDirWithContext(versionPath, "create test directory")
	require.NoError(t, err, "Failed to create version dir")

	goBinary := cfg.VersionGoBinary(version)
	if utils.IsWindows() {
		goBinary += ".exe"
	}
	testutil.WriteTestFile(t, goBinary, []byte("mock go binary"), utils.PermFileExecutable)

	// Run command with checksums
	cmd := inventoryGoCmd
	cmd.SetArgs([]string{})
	inventoryJSON = true
	inventoryChecksums = true

	var buf bytes.Buffer
	cmd.SetOut(&buf)

	err = cmd.RunE(cmd, []string{})
	require.NoError(t, err, "Command failed")

	// Parse JSON
	var installations []goInstallation
	err = json.Unmarshal(buf.Bytes(), &installations)
	require.NoError(t, err, "Failed to parse JSON")

	install := installations[0]

	// Checksum should be present
	assert.NotEmpty(t, install.SHA256, "Expected checksum to be computed")

	// Verify it's a valid SHA256 (64 hex characters)
	assert.Len(t, install.SHA256, 64, "Expected 64-character SHA256")
}

func TestCollectGoInstallation(t *testing.T) {
	var err error
	tmpDir := t.TempDir()

	cfg := &config.Config{
		Root: tmpDir,
	}

	// Create mock version
	version := "1.21.0"
	versionPath := cfg.VersionBinDir(version)
	err = utils.EnsureDirWithContext(versionPath, "create test directory")
	require.NoError(t, err, "Failed to create version dir")

	goBinary := cfg.VersionGoBinary(version)
	if utils.IsWindows() {
		goBinary += ".exe"
	}
	testutil.WriteTestFile(t, goBinary, []byte("test content"), utils.PermFileExecutable)

	// Collect without checksum
	install := collectGoInstallation(cfg, version, false)

	assert.Equal(t, version, install.Version, "Expected version")

	assert.Empty(t, install.SHA256, "Expected no checksum")

	// Collect with checksum
	installWithChecksum := collectGoInstallation(cfg, version, true)

	assert.NotEmpty(t, installWithChecksum.SHA256, "Expected checksum to be computed")
}

func TestComputeSHA256(t *testing.T) {
	tmpDir := t.TempDir()

	// Create test file
	testFile := filepath.Join(tmpDir, "test.txt")
	content := []byte("test content")
	testutil.WriteTestFile(t, testFile, content, utils.PermFileDefault)

	// Compute checksum
	checksum, err := utils.SHA256File(testFile)
	require.NoError(t, err, "Failed to compute checksum")

	// Verify it's valid hex
	assert.Len(t, checksum, 64, "Expected 64-character checksum")

	// Verify it's consistent
	checksum2, err := utils.SHA256File(testFile)
	require.NoError(t, err, "Failed to compute checksum again")

	assert.Equal(t, checksum2, checksum, "Checksums should be consistent")
}
