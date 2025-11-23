package cmdtest

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFixture(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	// Verify fixture is properly initialized
	assert.Equal(t, t, f.T, "Fixture.T not set correctly")
	assert.NotEmpty(t, f.Root, "Fixture.Root is empty")
	assert.NotEmpty(t, f.Home, "Fixture.Home is empty")
	assert.NotNil(t, f.Config, "Fixture.Config is nil")
	assert.NotNil(t, f.Manager, "Fixture.Manager is nil")
	assert.NotNil(t, f.Cleanup, "Fixture.Cleanup is nil")
}

func TestFixtureWithVersions(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	// Add versions using method chaining
	f.WithVersions("1.21.0", "1.22.0", "1.23.0")

	// Verify versions were created using ListInstalledVersions
	versions, err := f.Manager.ListInstalledVersions()
	require.NoError(t, err, "Failed to list installed versions")

	expectedVersions := []string{"1.21.0", "1.22.0", "1.23.0"}
	assert.Len(t, versions, len(expectedVersions), "Expected versions")

	for _, expected := range expectedVersions {
		found := false
		for _, v := range versions {
			if v == expected {
				found = true
				break
			}
		}
		assert.True(t, found, "Expected version not found in installed versions")
	}
}

func TestFixtureWithTools(t *testing.T) {
	var err error
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithVersions("1.21.0").
		WithTools("1.21.0", "gopls", "staticcheck")

	// Verify tools were created using FindExecutable (handles .bat/.exe on Windows)
	toolDir := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")

	_, err = utils.FindExecutable(toolDir, "gopls")
	assert.NoError(t, err, "gopls not found in")

	_, err = utils.FindExecutable(toolDir, "staticcheck")
	assert.NoError(t, err, "staticcheck not found in")
}

func TestFixtureWithGlobalVersion(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithVersions("1.21.0").
		WithGlobalVersion("1.21.0")

	// Verify global version was set
	current, _, err := f.Manager.GetCurrentVersion()
	require.NoError(t, err, "Failed to get current version")
	assert.Equal(t, "1.21.0", current, "Expected current version 1.21.0")
}

func TestFixtureWithLocalVersion(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithVersions("1.21.0", "1.22.0").
		WithGlobalVersion("1.21.0").
		WithLocalVersion("1.22.0")

	// Verify local version overrides global
	current, _, err := f.Manager.GetCurrentVersion()
	require.NoError(t, err, "Failed to get current version")
	assert.Equal(t, "1.22.0", current, "Expected current version 1.22.0 (local)")
}

func TestFixtureWithGoModFile(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithGoModFile("1.22.0")

	// Verify go.mod was created
	gomodPath := filepath.Join(f.Home, "go.mod")
	content, err := os.ReadFile(gomodPath)
	require.NoError(t, err, "go.mod not created")

	contentStr := string(content)
	assert.Contains(t, contentStr, "module test", "go.mod missing module declaration")
	assert.Contains(t, contentStr, "go 1.22.0", "go.mod missing go directive")
}

func TestFixtureWithGoModToolchain(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithGoModToolchain("1.22.0", "1.22.1")

	// Verify go.mod contains both directives
	gomodPath := filepath.Join(f.Home, "go.mod")
	content, err := os.ReadFile(gomodPath)
	require.NoError(t, err, "go.mod not created")

	contentStr := string(content)
	assert.Contains(t, contentStr, "go 1.22.0", "go.mod missing go directive")
	assert.Contains(t, contentStr, "toolchain go1.22.1", "go.mod missing toolchain directive")
}

func TestFixtureAssertions(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithVersions("1.21.0").
		WithGlobalVersion("1.21.0")

	// These should not fail
	f.AssertVersionInstalled("1.21.0")
	f.AssertCurrentVersion("1.21.0")
}

func TestMultiVersionScenario(t *testing.T) {
	f := MultiVersionScenario(t, "1.21.0", "1.22.0", "1.23.0")
	defer f.Cleanup()

	// Verify all versions exist
	f.AssertVersionInstalled("1.21.0")
	f.AssertVersionInstalled("1.22.0")
	f.AssertVersionInstalled("1.23.0")

	// First version should be global
	f.AssertCurrentVersion("1.21.0")
}

func TestToolScenario(t *testing.T) {
	var err error
	f := ToolScenario(t, "1.21.0", "gopls", "staticcheck")
	defer f.Cleanup()

	// Verify version and tools
	f.AssertVersionInstalled("1.21.0")

	// Verify tools were created using FindExecutable (handles .bat/.exe on Windows)
	toolDir := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")
	_, err = utils.FindExecutable(toolDir, "gopls")
	assert.NoError(t, err, "gopls not created in ToolScenario")
	_, err = utils.FindExecutable(toolDir, "staticcheck")
	assert.NoError(t, err, "staticcheck not created in ToolScenario")
}

func TestGoModScenario(t *testing.T) {
	f := GoModScenario(t, "1.22.0", "1.22.1")
	defer f.Cleanup()

	// Verify versions exist
	f.AssertVersionInstalled("1.22.0")
	f.AssertVersionInstalled("1.22.1")

	// Verify go.mod
	f.AssertFileExists("go.mod")
	f.AssertFileContains("go.mod", "go 1.22.0")
	f.AssertFileContains("go.mod", "toolchain go1.22.1")
}

func TestAliasScenario(t *testing.T) {
	aliases := map[string]string{
		"stable": "1.22.0",
		"latest": "1.23.0",
	}
	f := AliasScenario(t, aliases)
	defer f.Cleanup()

	// Verify target versions exist
	f.AssertVersionInstalled("1.22.0")
	f.AssertVersionInstalled("1.23.0")

	// Verify aliases file was created
	aliasesPath := filepath.Join(f.Root, "aliases")
	content, err := os.ReadFile(aliasesPath)
	require.NoError(t, err, "aliases file not created")

	contentStr := string(content)
	assert.Contains(t, contentStr, "stable=1.22.0", "stable alias not found in aliases file")
	assert.Contains(t, contentStr, "latest=1.23.0", "latest alias not found in aliases file")
}

func TestVersionBuilder(t *testing.T) {
	var err error
	f := NewFixture(t)
	defer f.Cleanup()

	// Build a complex version setup
	NewVersionBuilder(t, f.Root, "1.21.0").
		WithBinaries("gofmt", "gocov").
		WithTools("gopls", "staticcheck").
		WithPkgDir().
		WithGoMod().
		Build()

	// Verify binaries using FindExecutable (handles .bat/.exe on Windows)
	binDir := filepath.Join(f.Root, "versions", "1.21.0", "bin")
	_, err = utils.FindExecutable(binDir, "go")
	assert.NoError(t, err, "go binary not created")
	_, err = utils.FindExecutable(binDir, "gofmt")
	assert.NoError(t, err, "gofmt binary not created")

	// Verify tools using FindExecutable (handles .bat/.exe on Windows)
	toolDir := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")
	_, err = utils.FindExecutable(toolDir, "gopls")
	assert.NoError(t, err, "gopls tool not created")

	// Verify pkg directory
	pkgDir := filepath.Join(f.Root, "versions", "1.21.0", "pkg")
	if utils.FileNotExists(filepath.Join(pkgDir, "mod")) {
		t.Error("pkg/mod directory not created")
	}

	// Verify go.mod
	gomodPath := filepath.Join(f.Root, "versions", "1.21.0", "go.mod")
	if utils.FileNotExists(gomodPath) {
		t.Error("go.mod not created by VersionBuilder")
	}
}

func TestScenarioBuilder(t *testing.T) {
	var err error
	f := NewScenario(t).
		WithVersionAndTools("1.21.0", "gopls").
		WithVersionAndTools("1.22.0", "staticcheck").
		WithGlobal("1.21.0").
		WithLocal("1.22.0").
		Build()
	defer f.Cleanup()

	// Verify versions
	f.AssertVersionInstalled("1.21.0")
	f.AssertVersionInstalled("1.22.0")

	// Local should override global
	f.AssertCurrentVersion("1.22.0")

	// Verify tools using FindExecutable (handles .bat/.exe on Windows)
	toolDir1 := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")
	_, err = utils.FindExecutable(toolDir1, "gopls")
	assert.NoError(t, err, "gopls not created for 1.21.0")

	toolDir2 := filepath.Join(f.Root, "versions", "1.22.0", "gopath", "bin")
	_, err = utils.FindExecutable(toolDir2, "staticcheck")
	assert.NoError(t, err, "staticcheck not created for 1.22.0")
}
