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

// TestInstallListUninstall_RoundTrip is a regression test for the bug where
// "goenv tools install" writes binaries to versions/<version>/bin (because
// InstallTools sets GOPATH to the version directory passed by every caller),
// but "goenv tools list"/"uninstall" only looked in versions/<version>/gopath/bin
// — so an installed tool never appeared in list and could not be uninstalled,
// even though the shims and "default-tools verify" found it fine.
//
// ToolBinDirs is the shared source of truth all four operations must agree on.
func TestInstallListUninstall_RoundTrip(t *testing.T) {
	isolateHome(t)
	tmpDir := t.TempDir()
	cfg := &config.Config{Root: tmpDir}

	version := "1.27.1"
	versionBinDir := cfg.VersionBinDir(version) // versions/<version>/bin
	require.NoError(t, utils.EnsureDirWithContext(versionBinDir, "create version bin dir"))

	// Simulate the Go distribution's own binaries, which live alongside
	// installed tools in the same directory.
	testutil.WriteTestFile(t, filepath.Join(versionBinDir, "go"), []byte("fake go"), utils.PermFileExecutable)
	testutil.WriteTestFile(t, filepath.Join(versionBinDir, "gofmt"), []byte("fake gofmt"), utils.PermFileExecutable)

	// Simulate "goenv tools install mvdan.cc/gofumpt@latest": this is exactly
	// where InstallTools places the binary (GOBIN=versions/<version>/bin).
	gofumptPath := filepath.Join(versionBinDir, "gofumpt")
	testutil.WriteTestFile(t, gofumptPath, []byte("fake gofumpt"), utils.PermFileExecutable)

	// "list" must find the installed tool...
	tools, err := ListForVersion(cfg, version)
	require.NoError(t, err)
	require.Len(t, tools, 1, "expected exactly one tool to be listed")
	assert.Equal(t, "gofumpt", tools[0].Name)

	// ...and must never report the Go distribution's own binaries as tools.
	for _, tool := range tools {
		assert.NotEqual(t, "go", tool.Name)
		assert.NotEqual(t, "gofmt", tool.Name)
	}

	assert.True(t, IsInstalled(cfg, version, "gofumpt"), "gofumpt should be reported as installed")

	mgr := &Manager{cfg: cfg}

	// "uninstall" must never remove the Go distribution's own binaries, even
	// if asked to by name.
	err = mgr.UninstallSingleTool(version, "go")
	assert.Error(t, err, "uninstalling 'go' should fail")
	assert.FileExists(t, filepath.Join(versionBinDir, "go"), "go binary must survive an uninstall attempt")

	err = mgr.UninstallSingleTool(version, "gofmt")
	assert.Error(t, err, "uninstalling 'gofmt' should fail")
	assert.FileExists(t, filepath.Join(versionBinDir, "gofmt"), "gofmt binary must survive an uninstall attempt")

	// "uninstall" must remove the actual tool.
	err = mgr.UninstallSingleTool(version, "gofumpt")
	require.NoError(t, err, "uninstalling gofumpt should succeed")
	assert.NoFileExists(t, gofumptPath, "gofumpt binary should be removed")

	// The Go distribution binaries must still be present after the tool is gone.
	assert.FileExists(t, filepath.Join(versionBinDir, "go"))
	assert.FileExists(t, filepath.Join(versionBinDir, "gofmt"))

	// list must now report no tools for this version.
	tools, err = ListForVersion(cfg, version)
	require.NoError(t, err)
	assert.Len(t, tools, 0, "expected no tools left after uninstall")
}
