package tools

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/cmdtest"
	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	toolspkg "github.com/go-nv/goenv/internal/tools"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFindToolBinaries(t *testing.T) {
	var err error
	tests := []struct {
		name          string
		toolName      string
		files         []string
		expectedCount int
		expectedFiles []string
	}{
		{
			name:          "exact match",
			toolName:      "gopls",
			files:         []string{"gopls", "othertool"},
			expectedCount: 1,
			expectedFiles: []string{"gopls"},
		},
		{
			name:          "platform variants",
			toolName:      "gopls",
			files:         []string{"gopls", "gopls.exe", "gopls.darwin"},
			expectedCount: 3,
			expectedFiles: []string{"gopls", "gopls.exe", "gopls.darwin"},
		},
		{
			name:          "no match",
			toolName:      "gopls",
			files:         []string{"othertool", "anothertool"},
			expectedCount: 0,
			expectedFiles: []string{},
		},
		{
			name:          "partial name no match",
			toolName:      "gopls",
			files:         []string{"gopls-old", "mygopls"},
			expectedCount: 0,
			expectedFiles: []string{},
		},
		{
			name:          "windows exe only",
			toolName:      "gopls",
			files:         []string{"gopls.exe"},
			expectedCount: 1,
			expectedFiles: []string{"gopls.exe"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create temp directory
			tmpDir := t.TempDir()
			binPath := filepath.Join(tmpDir, "bin")
			err = utils.EnsureDirWithContext(binPath, "create test directory")
			require.NoError(t, err)

			// Create test files
			for _, file := range tt.files {
				filePath := filepath.Join(binPath, file)
				testutil.WriteTestFile(t, filePath, []byte("fake binary"), utils.PermFileExecutable)
			}

			// Find binaries
			binaries := findToolBinaries(binPath, tt.toolName)

			// Check count
			assert.Len(t, binaries, tt.expectedCount, "unexpected number of binaries found: expected length")

			// Check expected files are present
			foundNames := make(map[string]bool)
			for _, bin := range binaries {
				foundNames[filepath.Base(bin)] = true
			}

			for _, expected := range tt.expectedFiles {
				assert.True(t, foundNames[expected], "expected file not found")
			}
		})
	}
}

func TestFindToolBinaries_NonExistentDir(t *testing.T) {
	binPath := "/nonexistent/path/bin"
	binaries := findToolBinaries(binPath, "gopls")
	assert.Len(t, binaries, 0, "should return empty slice for non-existent directory: expected length")
}

func TestFindCurrentVersionToolTargets(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)
	binPath := cfg.VersionGopathBin(version)

	// Create test tools using helper (handles .bat on Windows)
	tools := []string{"gopls", "staticcheck"}
	for _, tool := range tools {
		cmdtest.CreateToolExecutable(t, binPath, tool)
	}

	// Set current version
	versionFile := filepath.Join(tmpDir, "version")
	testutil.WriteTestFile(t, versionFile, []byte(version), utils.PermFileDefault)

	// Set environment variable to simulate current version
	t.Setenv(utils.GoenvEnvVarVersion.String(), version)

	// Find targets
	mgr := manager.NewManager(cfg, nil)
	targets := findCurrentVersionToolTargets(cfg, mgr, []string{"gopls", "staticcheck", "nonexistent"})

	if len(targets) != 3 {
		t.Fatalf("should return target for each tool name: expected length %v, got %v", 3, len(targets))
	}

	// Check gopls
	assert.Equal(t, "gopls", targets[0].ToolName)
	assert.Equal(t, version, targets[0].GoVersion)
	assert.True(t, (targets[0].Exists))
	assert.Len(t, targets[0].BinaryFiles, 1)

	// Check staticcheck
	assert.Equal(t, "staticcheck", targets[1].ToolName)
	assert.True(t, (targets[1].Exists))

	// Check nonexistent
	assert.Equal(t, "nonexistent", targets[2].ToolName)
	if targets[2].Exists {
		t.Errorf("expected false")
	}
	assert.Len(t, targets[2].BinaryFiles, 0)
}

func TestFindAllVersionToolTargets(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	versions := []string{"1.21.0", "1.22.0", "1.23.0"}

	// Create gopls in all versions, staticcheck only in some
	for i, version := range versions {
		cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)
		binPath := cfg.VersionGopathBin(version)

		// gopls in all versions
		goplsPath := filepath.Join(binPath, "gopls")
		testutil.WriteTestFile(t, goplsPath, []byte("fake"), utils.PermFileExecutable)

		// staticcheck only in 1.22.0 and 1.23.0
		if i >= 1 {
			staticPath := filepath.Join(binPath, "staticcheck")
			testutil.WriteTestFile(t, staticPath, []byte("fake"), utils.PermFileExecutable)
		}
	}

	// Find gopls across all versions
	goplsTargets := findAllVersionToolTargets(cfg, []string{"gopls"})
	assert.Len(t, goplsTargets, 3, "gopls should be found in all 3 versions: expected length")

	for _, target := range goplsTargets {
		assert.Equal(t, "gopls", target.ToolName)
		assert.True(t, (target.Exists))
		assert.True(t, utils.SliceContains(versions, target.GoVersion), "expected slice to contain")
	}

	// Find staticcheck (only in 2 versions)
	staticTargets := findAllVersionToolTargets(cfg, []string{"staticcheck"})
	assert.Len(t, staticTargets, 2, "staticcheck should be found in 2 versions: expected length")

	// Find nonexistent tool
	noneTargets := findAllVersionToolTargets(cfg, []string{"nonexistent"})
	assert.Len(t, noneTargets, 0, "nonexistent tool should return no targets: expected length")
}

func TestFindGlobalToolTargets(t *testing.T) {
	var err error
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	globalGopath := filepath.Join(tmpDir, "global-go")
	binPath := filepath.Join(globalGopath, "bin")
	err = utils.EnsureDirWithContext(binPath, "create test directory")
	require.NoError(t, err)

	// Create test tool
	goplsPath := filepath.Join(binPath, "gopls")
	testutil.WriteTestFile(t, goplsPath, []byte("fake"), utils.PermFileExecutable)

	// Override GOPATH
	t.Setenv(utils.EnvVarGopath, globalGopath)

	// Find targets
	targets := findGlobalToolTargets(cfg, []string{"gopls", "nonexistent"})

	if len(targets) != 2 {
		t.Fatalf("expected length %v, got %v", 2, len(targets))
	}

	// Check gopls
	assert.Equal(t, "gopls", targets[0].ToolName)
	assert.Equal(t, "", targets[0].GoVersion) // Empty for global
	assert.True(t, (targets[0].Exists))

	// Check nonexistent
	assert.Equal(t, "nonexistent", targets[1].ToolName)
	if targets[1].Exists {
		t.Errorf("expected false")
	}
}

func TestRunUninstall_StripVersionSuffix(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	version := "1.23.0"
	cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)
	binPath := cfg.VersionGopathBin(version)

	// Create tool
	goplsPath := filepath.Join(binPath, "gopls")
	testutil.WriteTestFile(t, goplsPath, []byte("fake"), utils.PermFileExecutable)

	t.Setenv(utils.GoenvEnvVarVersion.String(), version)

	// Test the stripping logic directly (runUninstall does the stripping)
	toolName := "gopls@v0.12.0"
	if idx := strings.Index(toolName, "@"); idx != -1 {
		toolName = toolName[:idx]
	}
	assert.Equal(t, "gopls", toolName)

	// Now verify findCurrentVersionToolTargets works with the clean name
	mgr := manager.NewManager(cfg, nil)
	targets := findCurrentVersionToolTargets(cfg, mgr, []string{toolName})
	if len(targets) != 1 {
		t.Fatalf("expected length %v, got %v", 1, len(targets))
	}
	assert.Equal(t, "gopls", targets[0].ToolName)
	assert.True(t, (targets[0].Exists))
}

func TestExecuteUninstalls(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	version := "1.23.0"
	cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)
	binPath := cfg.VersionGopathBin(version)

	// Create test files
	gopls := filepath.Join(binPath, "gopls")
	goplsExe := filepath.Join(binPath, "gopls.exe")
	testutil.WriteTestFile(t, gopls, []byte("fake"), utils.PermFileExecutable)
	testutil.WriteTestFile(t, goplsExe, []byte("fake"), utils.PermFileExecutable)

	// Create manager
	mgr := manager.NewManager(cfg, nil)
	toolsMgr := toolspkg.NewManager(cfg, mgr)

	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   version,
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{gopls, goplsExe},
		},
	}

	// Execute uninstalls
	uninstallVerbose = false
	uninstallForce = true

	err := executeUninstalls(toolsMgr, targets)
	assert.NoError(t, err)

	// Verify files are removed
	if utils.PathExists(gopls) {
		t.Errorf("gopls should be removed")
	}

	if utils.PathExists(goplsExe) {
		t.Errorf("gopls.exe should be removed")
	}
}

func TestExecuteUninstalls_MultipleTools(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	version := "1.23.0"
	cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)
	binPath := cfg.VersionGopathBin(version)

	// Create multiple tools
	gopls := filepath.Join(binPath, "gopls")
	staticcheck := filepath.Join(binPath, "staticcheck")
	testutil.WriteTestFile(t, gopls, []byte("fake"), utils.PermFileExecutable)
	testutil.WriteTestFile(t, staticcheck, []byte("fake"), utils.PermFileExecutable)

	// Create manager
	mgr := manager.NewManager(cfg, nil)
	toolsMgr := toolspkg.NewManager(cfg, mgr)

	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   version,
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{gopls},
		},
		{
			ToolName:    "staticcheck",
			GoVersion:   version,
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{staticcheck},
		},
	}

	uninstallVerbose = false
	uninstallForce = true

	err := executeUninstalls(toolsMgr, targets)
	assert.NoError(t, err)

	// Verify both removed
	if utils.PathExists(gopls) {
		t.Errorf("expected file to not exist")
	}

	if utils.PathExists(staticcheck) {
		t.Errorf("expected file to not exist")
	}
}

func TestExecuteUninstalls_PartialFailure(t *testing.T) {
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}
	version := "1.23.0"
	cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)
	binPath := cfg.VersionGopathBin(version)

	// Create one tool
	gopls := filepath.Join(binPath, "gopls")
	testutil.WriteTestFile(t, gopls, []byte("fake"), utils.PermFileExecutable)

	// Create manager
	mgr := manager.NewManager(cfg, nil)
	toolsMgr := toolspkg.NewManager(cfg, mgr)

	// Create target with non-existent file (will fail to remove)
	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   version,
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{gopls},
		},
		{
			ToolName:    "nonexistent",
			GoVersion:   version,
			BinPath:     binPath,
			Exists:      true,
			BinaryFiles: []string{filepath.Join(binPath, "nonexistent")},
		},
	}

	uninstallVerbose = false
	uninstallForce = true

	err := executeUninstalls(toolsMgr, targets)
	assert.Error(t, err, "should return error when some tools fail")
	assert.False(t, err != nil && !strings.Contains(err.Error(), "failed to uninstall"), "expected error to contain 'failed to uninstall'")

	// First tool should still be removed
	if utils.PathExists(gopls) {
		t.Errorf("expected file to not exist")
	}
}

func TestShowUninstallPlan(t *testing.T) {
	var err error
	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   "1.23.0",
			BinPath:     "/path/to/bin",
			Exists:      true,
			BinaryFiles: []string{"/path/to/bin/gopls", "/path/to/bin/gopls.exe"},
		},
		{
			ToolName:    "staticcheck",
			GoVersion:   "1.22.0",
			BinPath:     "/path/to/other/bin",
			Exists:      true,
			BinaryFiles: []string{"/path/to/other/bin/staticcheck"},
		},
	}

	// Test without verbose
	uninstallVerbose = false
	err = showUninstallPlan(targets)
	assert.NoError(t, err)

	// Test with verbose
	uninstallVerbose = true
	err = showUninstallPlan(targets)
	assert.NoError(t, err)
}

func TestShowUninstallPlan_Global(t *testing.T) {
	targets := []toolUninstallTarget{
		{
			ToolName:    "gopls",
			GoVersion:   "", // Empty for global
			BinPath:     "/home/user/go/bin",
			Exists:      true,
			BinaryFiles: []string{"/home/user/go/bin/gopls"},
		},
	}

	uninstallVerbose = false
	err := showUninstallPlan(targets)
	assert.NoError(t, err)
}

func TestRunUninstall_Integration(t *testing.T) {
	var err error
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	tmpDir := t.TempDir()
	cfg := &config.Config{
		Root: tmpDir,
	}

	// Setup: Create Go version with tools
	version := "1.23.0"
	cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)
	binPath := cfg.VersionGopathBin(version)

	gopls := filepath.Join(binPath, "gopls")
	staticcheck := filepath.Join(binPath, "staticcheck")
	testutil.WriteTestFile(t, gopls, []byte("fake"), utils.PermFileExecutable)
	testutil.WriteTestFile(t, staticcheck, []byte("fake"), utils.PermFileExecutable)

	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)
	t.Setenv(utils.GoenvEnvVarVersion.String(), version)

	// Run uninstall with dry-run
	uninstallDryRun = true
	uninstallForce = true
	uninstallAllVersions = false
	uninstallGlobal = false

	cmd := &cobra.Command{}

	cmd.SetContext(context.Background())

	err = runUninstall(cmd, []string{"gopls"})
	assert.NoError(t, err)

	// Files should still exist after dry-run
	if utils.FileNotExists(gopls) {
		t.Errorf("expected file to exist: %s", gopls)
	}

	// Run actual uninstall
	uninstallDryRun = false
	err = runUninstall(cmd, []string{"gopls"})
	assert.NoError(t, err)

	// gopls should be removed
	if utils.PathExists(gopls) {
		t.Errorf("expected file to not exist")
	}

	// staticcheck should still exist
	if utils.FileNotExists(staticcheck) {
		t.Errorf("expected file to exist: %s", staticcheck)
	}
}

func TestRunUninstall_NoToolsFound(t *testing.T) {
	tmpDir := t.TempDir()
	cmd := &cobra.Command{}

	cmd.SetContext(context.Background())

	version := "1.23.0"
	cmdtest.CreateMockGoVersionWithTools(t, tmpDir, version)

	t.Setenv(utils.GoenvEnvVarVersion.String(), version)
	t.Setenv(utils.GoenvEnvVarRoot.String(), tmpDir)

	uninstallForce = true
	err := runUninstall(cmd, []string{"nonexistent"})

	// Should not return error, just print message
	assert.NoError(t, err)
}

func TestParseToolNames(t *testing.T) {
	// Test that runUninstall strips @version
	inputs := []string{
		"gopls@v0.12.0",
		"staticcheck@latest",
		"golangci-lint",
	}

	// Just verify the stripping logic works
	var cleanNames []string
	for _, name := range inputs {
		if idx := strings.Index(name, "@"); idx != -1 {
			name = name[:idx]
		}
		cleanNames = append(cleanNames, name)
	}

	assert.Equal(t, "gopls", cleanNames[0])
	assert.Equal(t, "staticcheck", cleanNames[1])
	assert.Equal(t, "golangci-lint", cleanNames[2])
}
