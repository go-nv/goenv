package tools

import (
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	require.NotNil(t, config, "DefaultConfig returned nil")

	assert.True(t, config.Enabled, "Expected default config to be enabled")

	assert.NotEqual(t, 0, len(config.Tools), "Expected default config to have tools")

	// Check for expected default tools
	expectedTools := []string{"gopls", "golangci-lint", "staticcheck", "delve"}
	foundTools := make(map[string]bool)

	for _, tool := range config.Tools {
		foundTools[tool.Name] = true

		// Verify tool has required fields
		assert.NotEmpty(t, tool.Package, "Tool has empty package")
	}

	for _, expected := range expectedTools {
		assert.True(t, foundTools[expected], "Expected default tool not found")
	}
}

func TestLoadConfig_NonExistent(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "nonexistent.yaml")

	config, err := LoadConfig(configPath)
	require.NoError(t, err, "LoadConfig failed for non-existent file")

	// Should return default config
	require.NotNil(t, config, "LoadConfig returned nil for non-existent file")

	assert.True(t, config.Enabled, "Expected default config when file doesn't exist")
}

func TestSaveAndLoadConfig(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test-config.yaml")

	// Create a custom config
	originalConfig := &Config{
		Enabled: true,
		Tools: []Tool{
			{
				Name:    "gopls",
				Package: "golang.org/x/tools/gopls",
				Version: "@v0.14.0",
				Binary:  "gopls",
			},
			{
				Name:    "custom-tool",
				Package: "example.com/custom",
				Version: "@latest",
			},
		},
	}

	// Save config
	err := SaveConfig(configPath, originalConfig)
	require.NoError(t, err, "SaveConfig failed")

	// Verify file exists
	if utils.FileNotExists(configPath) {
		t.Fatal("Config file was not created")
	}

	// Load config back
	loadedConfig, err := LoadConfig(configPath)
	require.NoError(t, err, "LoadConfig failed")

	// Verify loaded config matches original
	assert.Equal(t, originalConfig.Enabled, loadedConfig.Enabled, "Enabled mismatch")

	if len(loadedConfig.Tools) != len(originalConfig.Tools) {
		t.Fatalf("Tools count mismatch: got %d, want %d", len(loadedConfig.Tools), len(originalConfig.Tools))
	}

	for i, tool := range loadedConfig.Tools {
		orig := originalConfig.Tools[i]
		assert.Equal(t, orig.Name, tool.Name, "Tool[] Name mismatch %v", i)
		assert.Equal(t, orig.Package, tool.Package, "Tool[] Package mismatch %v", i)
		assert.Equal(t, orig.Version, tool.Version, "Tool[] Version mismatch %v", i)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "invalid.yaml")

	// Write invalid YAML
	testutil.WriteTestFile(t, configPath, []byte("not: valid: yaml: content"), utils.PermFileDefault, "Failed to write invalid YAML")

	_, err := LoadConfig(configPath)
	assert.Error(t, err, "Expected error for invalid YAML, got nil")

	assert.Contains(t, err.Error(), "failed to parse", "Expected parse error %v", err)
}

func TestConfigPath(t *testing.T) {
	goenvRoot := "/test/goenv"
	path := ConfigPath(goenvRoot)

	expected := filepath.Join(goenvRoot, "default-tools.yaml")
	assert.Equal(t, expected, path, "ConfigPath mismatch")
}

func TestInstallTools_Disabled(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled: false,
		Tools: []Tool{
			{Name: "test", Package: "example.com/test"},
		},
	}

	hostGopath := filepath.Join(tmpDir, "host-gopath")
	err := InstallTools(config, "1.21.0", tmpDir, hostGopath, true)
	assert.NoError(t, err, "InstallTools should not error when disabled")
}

func TestInstallTools_NoTools(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled: true,
		Tools:   []Tool{},
	}

	hostGopath := filepath.Join(tmpDir, "host-gopath")
	err := InstallTools(config, "1.21.0", tmpDir, hostGopath, true)
	assert.NoError(t, err, "InstallTools should not error with no tools")
}

func TestInstallTools_GoNotFound(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Enabled: true,
		Tools: []Tool{
			{Name: "test", Package: "example.com/test"},
		},
	}

	hostGopath := filepath.Join(tmpDir, "host-gopath")
	err := InstallTools(config, "1.21.0", tmpDir, hostGopath, false)
	assert.Error(t, err, "Expected error when Go binary not found")

	assert.Contains(t, err.Error(), "go binary not found", "Expected 'Go binary not found' error %v", err)
}

func TestInstallTools_WithMockGo(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	// The version directory IS the GOROOT (no extra 'go' subdirectory)
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	goBinDir := filepath.Join(versionPath, "bin")
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	err = utils.EnsureDirWithContext(goBinDir, "create test directory")
	require.NoError(t, err, "Failed to create go bin directory")
	err = utils.EnsureDirWithContext(gopathBin, "create test directory")
	require.NoError(t, err, "Failed to create gopath bin directory")

	// Create mock go binary that succeeds
	goBinary := filepath.Join(goBinDir, "go")
	mockScript := "#!/bin/sh\nexit 0"
	if utils.IsWindows() {
		goBinary += ".bat"
		mockScript = "@echo off\nexit 0"
	}
	testutil.WriteTestFile(t, goBinary, []byte(mockScript), utils.PermFileExecutable, "Failed to create go binary")

	config := &Config{
		Enabled: true,
		Tools: []Tool{
			{
				Name:    "test-tool",
				Package: "example.com/test",
				Version: "@latest",
				Binary:  "test-tool",
			},
		},
	}

	hostGopath := filepath.Join(tmpDir, "host-gopath")
	err = InstallTools(config, goVersion, tmpDir, hostGopath, false)
	assert.NoError(t, err, "InstallTools failed")
}

func TestInstallTools_Failure(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	// The version directory IS the GOROOT (no extra 'go' subdirectory)
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	goBinDir := filepath.Join(versionPath, "bin")

	err = utils.EnsureDirWithContext(goBinDir, "create test directory")
	require.NoError(t, err, "Failed to create go bin directory")

	// Create mock go binary that fails
	goBinary := filepath.Join(goBinDir, "go")
	mockScript := "#!/bin/sh\nexit 1"
	if utils.IsWindows() {
		goBinary += ".bat"
		mockScript = "@echo off\nexit 1"
	}
	testutil.WriteTestFile(t, goBinary, []byte(mockScript), utils.PermFileExecutable, "Failed to create go binary")

	config := &Config{
		Enabled: true,
		Tools: []Tool{
			{
				Name:    "failing-tool",
				Package: "example.com/fail",
			},
		},
	}

	hostGopath := filepath.Join(tmpDir, "host-gopath")
	err = InstallTools(config, goVersion, tmpDir, hostGopath, false)
	assert.Error(t, err, "Expected error when tool installation fails")

	assert.Contains(t, err.Error(), "failed to install", "Expected 'failed to install' error %v", err)
}

func TestVerifyTools(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	err = utils.EnsureDirWithContext(gopathBin, "create test directory")
	require.NoError(t, err, "Failed to create gopath bin directory")

	// Create some tool binaries
	tool1Binary := filepath.Join(gopathBin, "gopls")
	tool2Binary := filepath.Join(gopathBin, "dlv")
	if utils.IsWindows() {
		tool1Binary += ".exe"
		tool2Binary += ".exe"
	}

	testutil.WriteTestFile(t, tool1Binary, []byte("mock binary"), utils.PermFileExecutable, "Failed to create tool1 binary")
	testutil.WriteTestFile(t, tool2Binary, []byte("mock binary"), utils.PermFileExecutable, "Failed to create tool2 binary")

	config := &Config{
		Tools: []Tool{
			{
				Name:    "gopls",
				Package: "golang.org/x/tools/gopls",
				Binary:  "gopls",
			},
			{
				Name:    "delve",
				Package: "github.com/go-delve/delve/cmd/dlv",
				Binary:  "dlv",
			},
			{
				Name:    "missing-tool",
				Package: "example.com/missing",
				Binary:  "missing",
			},
		},
	}

	results, err := VerifyTools(config, goVersion, tmpDir)
	require.NoError(t, err, "VerifyTools failed")

	// Check results
	assert.True(t, results["gopls"], "Expected gopls to be found")

	assert.True(t, results["delve"], "Expected delve to be found")

	if results["missing-tool"] {
		t.Error("Expected missing-tool to not be found")
	}
}

// TestVerifyTools_FindsToolsWhereInstallToolsWritesThem is a regression test for
// the false "not installed" reports described in issue #581.
//
// Both InstallTools call sites pass Config.SafeResolvePath(version) as the
// GOPATH, which resolves to the version directory itself once the version is
// installed, so "go install" writes binaries to versions/<version>/bin.
// VerifyTools used to look only in versions/<version>/gopath/bin and therefore
// reported successfully-installed tools as missing.
func TestVerifyTools_FindsToolsWhereInstallToolsWritesThem(t *testing.T) {
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// GOPATH=versions/<version> means "go install" lands in versions/<version>/bin.
	installedBin := filepath.Join(tmpDir, "versions", goVersion, "bin")
	require.NoError(t, utils.EnsureDirWithContext(installedBin, "create test directory"))

	goplsBinary := filepath.Join(installedBin, "gopls")
	if utils.IsWindows() {
		goplsBinary += ".exe"
	}
	testutil.WriteTestFile(t, goplsBinary, []byte("mock binary"), utils.PermFileExecutable, "Failed to create gopls binary")

	config := &Config{
		Tools: []Tool{
			{Name: "gopls", Package: "golang.org/x/tools/gopls", Binary: "gopls"},
			{Name: "missing-tool", Package: "example.com/missing", Binary: "missing"},
		},
	}

	results, err := VerifyTools(config, goVersion, tmpDir)
	require.NoError(t, err, "VerifyTools failed")

	assert.True(t, results["gopls"],
		"gopls installed at %s should be reported as installed", installedBin)
	assert.False(t, results["missing-tool"], "missing-tool should not be found")
}

// TestToolBinDirs_CoversInstallToolsTarget asserts that the directory
// InstallTools actually writes to is one of the directories VerifyTools
// searches, so the two cannot silently drift apart again.
func TestToolBinDirs_CoversInstallToolsTarget(t *testing.T) {
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	cfg := &config.Config{Root: tmpDir}
	require.NoError(t, utils.EnsureDirWithContext(cfg.VersionDir(goVersion), "create version dir"))

	// This is the argument both InstallTools call sites pass as hostGopath.
	hostGopath := cfg.SafeResolvePath(goVersion)
	installTarget := filepath.Join(hostGopath, "bin")

	assert.Contains(t, ToolBinDirs(tmpDir, goVersion), installTarget,
		"ToolBinDirs must include the directory InstallTools writes to")
}

// isolateHome points the user's home directory at a temporary location for the
// duration of a test.
//
// ToolBinDirs includes the legacy "$HOME/go/<version>/bin" location, so without
// this a developer who happens to have a tool installed there gets a different
// result from CI — and the test passes or fails for reasons unrelated to the
// code under test.
func isolateHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)        // Unix
	t.Setenv("USERPROFILE", home) // Windows
}

// TestToolBinDirs_AreVersionScoped guards against a false positive: reporting a
// tool as installed for version A because it exists for version B.
//
// Verification exists to tell the user whether a specific version has its
// tools. A shared, host-wide directory in the search list silently defeats
// that, and a silent false positive costs more than the false negative it
// replaced — the user debugs the wrong thing.
func TestToolBinDirs_AreVersionScoped(t *testing.T) {
	tmpDir := t.TempDir()

	dirsA := ToolBinDirs(tmpDir, "1.21.0")
	dirsB := ToolBinDirs(tmpDir, "1.24.0")

	for _, a := range dirsA {
		assert.NotContains(t, dirsB, a,
			"directory %q is searched for both 1.21.0 and 1.24.0, so a tool installed "+
				"for one version would be reported as installed for the other", a)
	}
}

// TestVerifyTools_DoesNotLeakAcrossVersions is the behavioural counterpart:
// installing a tool for one version must not mark it installed for another.
func TestVerifyTools_DoesNotLeakAcrossVersions(t *testing.T) {
	tmpDir := t.TempDir()

	// ToolBinDirs consults $HOME/go/<version>/bin, so the developer's real home
	// directory would otherwise decide the result of this test.
	isolateHome(t)

	installedFor := "1.24.0"
	queriedFor := "1.21.0"

	binDir := filepath.Join(tmpDir, "versions", installedFor, "bin")
	require.NoError(t, utils.EnsureDirWithContext(binDir, "create test directory"))

	goplsBinary := filepath.Join(binDir, "gopls")
	if utils.IsWindows() {
		goplsBinary += ".exe"
	}
	testutil.WriteTestFile(t, goplsBinary, []byte("mock binary"), utils.PermFileExecutable, "Failed to create gopls binary")

	config := &Config{
		Tools: []Tool{{Name: "gopls", Package: "golang.org/x/tools/gopls", Binary: "gopls"}},
	}

	results, err := VerifyTools(config, queriedFor, tmpDir)
	require.NoError(t, err)

	assert.False(t, results["gopls"],
		"a tool installed only for %s must not be reported as installed for %s",
		installedFor, queriedFor)
}

func TestVerifyTools_NoBinaryName(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	goVersion := "1.21.0"

	// Setup version structure
	versionPath := filepath.Join(tmpDir, "versions", goVersion)
	gopathBin := filepath.Join(versionPath, "gopath", "bin")

	err = utils.EnsureDirWithContext(gopathBin, "create test directory")
	require.NoError(t, err, "Failed to create gopath bin directory")

	// Create tool binary with name extracted from package path
	toolBinary := filepath.Join(gopathBin, "staticcheck")
	if utils.IsWindows() {
		toolBinary += ".exe"
	}

	testutil.WriteTestFile(t, toolBinary, []byte("mock binary"), utils.PermFileExecutable, "Failed to create tool binary")

	config := &Config{
		Tools: []Tool{
			{
				Name:    "staticcheck",
				Package: "honnef.co/go/tools/cmd/staticcheck",
				// Binary field is empty, should extract "staticcheck" from package
			},
		},
	}

	results, err := VerifyTools(config, goVersion, tmpDir)
	require.NoError(t, err, "VerifyTools failed")

	assert.True(t, results["staticcheck"], "Expected staticcheck to be found (binary name extracted from package)")
}

func TestVerifyTools_EmptyConfig(t *testing.T) {
	tmpDir := t.TempDir()

	config := &Config{
		Tools: []Tool{},
	}

	results, err := VerifyTools(config, "1.21.0", tmpDir)
	require.NoError(t, err, "VerifyTools failed")

	assert.Len(t, results, 0, "Expected empty results for empty config")
}
