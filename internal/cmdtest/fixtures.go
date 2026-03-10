package cmdtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
)

// Fixture represents a test fixture with common test data and setup
type Fixture struct {
	T       *testing.T
	Root    string
	Home    string
	Config  *config.Config
	Manager *manager.Manager
	Cleanup func()
}

// NewFixture creates a new test fixture with a complete test environment.
// This is the recommended way to set up tests that need a full goenv environment.
//
// Example:
//
//	func TestMyCommand(t *testing.T) {
//	    f := cmdtest.NewFixture(t)
//	    defer f.Cleanup()
//
//	    f.WithVersions("1.21.0", "1.22.0")
//	    f.WithGlobalVersion("1.21.0")
//	    // ... test code
//	}
func NewFixture(t *testing.T) *Fixture {
	root, cleanup := SetupTestEnv(t)

	cfg := config.Load()
	mgr := manager.NewManager(cfg, nil) // nil triggers fallback environment loading

	return &Fixture{
		T:       t,
		Root:    root,
		Home:    filepath.Join(filepath.Dir(root), "home"),
		Config:  cfg,
		Manager: mgr,
		Cleanup: cleanup,
	}
}

// WithVersions creates multiple test Go versions.
// Returns the fixture for method chaining.
func (f *Fixture) WithVersions(versions ...string) *Fixture {
	for _, version := range versions {
		CreateTestVersion(f.T, f.Root, version)
	}
	return f
}

// WithTools creates tools for a specific Go version.
// Returns the fixture for method chaining.
func (f *Fixture) WithTools(version string, tools ...string) *Fixture {
	toolDir := filepath.Join(f.Root, "versions", version, "gopath", "bin")
	for _, tool := range tools {
		CreateToolExecutable(f.T, toolDir, tool)
	}
	return f
}

// WithGlobalVersion sets the global version.
// Returns the fixture for method chaining.
func (f *Fixture) WithGlobalVersion(version string) *Fixture {
	if err := f.Manager.SetGlobalVersion(version); err != nil {
		f.T.Fatalf("Failed to set global version: %v", err)
	}
	return f
}

// WithLocalVersion sets the local version (creates .go-version file).
// Returns the fixture for method chaining.
func (f *Fixture) WithLocalVersion(version string) *Fixture {
	if err := f.Manager.SetLocalVersion(version); err != nil {
		f.T.Fatalf("Failed to set local version: %v", err)
	}
	return f
}

// WithAlias creates a version alias.
// Returns the fixture for method chaining.
func (f *Fixture) WithAlias(name, target string) *Fixture {
	CreateTestAlias(f.T, f.Root, name, target)
	return f
}

// WithGoModFile creates a go.mod file with a specific Go version requirement.
// Returns the fixture for method chaining.
func (f *Fixture) WithGoModFile(version string) *Fixture {
	gomodPath := filepath.Join(f.Home, "go.mod")
	content := "module test\n\ngo " + version + "\n"
	if err := utils.WriteFileWithContext(gomodPath, []byte(content), utils.PermFileDefault, "create go.mod"); err != nil {
		f.T.Fatalf("Failed to create go.mod: %v", err)
	}
	return f
}

// WithGoModToolchain creates a go.mod file with both go directive and toolchain directive.
// Returns the fixture for method chaining.
func (f *Fixture) WithGoModToolchain(goVersion, toolchainVersion string) *Fixture {
	gomodPath := filepath.Join(f.Home, "go.mod")
	content := "module test\n\ngo " + goVersion + "\n\ntoolchain go" + toolchainVersion + "\n"
	if err := utils.WriteFileWithContext(gomodPath, []byte(content), utils.PermFileDefault, "create go.mod"); err != nil {
		f.T.Fatalf("Failed to create go.mod: %v", err)
	}
	return f
}

// WithToolVersionsFile creates a .tool-versions file (asdf format).
// Returns the fixture for method chaining.
func (f *Fixture) WithToolVersionsFile(goVersion string, otherTools ...string) *Fixture {
	toolVersionsPath := filepath.Join(f.Home, ".tool-versions")
	content := "golang " + goVersion + "\n"
	for i := 0; i < len(otherTools); i += 2 {
		if i+1 < len(otherTools) {
			content += otherTools[i] + " " + otherTools[i+1] + "\n"
		}
	}
	if err := utils.WriteFileWithContext(toolVersionsPath, []byte(content), utils.PermFileDefault, "create .tool-versions"); err != nil {
		f.T.Fatalf("Failed to create .tool-versions: %v", err)
	}
	return f
}

// WithFile creates an arbitrary file with the given content.
// Path is relative to the test home directory.
// Returns the fixture for method chaining.
func (f *Fixture) WithFile(path, content string) *Fixture {
	fullPath := filepath.Join(f.Home, path)
	dir := filepath.Dir(fullPath)
	if err := utils.EnsureDirWithContext(dir, "create test directory"); err != nil {
		f.T.Fatalf("Failed to create directory %s: %v", dir, err)
	}
	if err := utils.WriteFileWithContext(fullPath, []byte(content), utils.PermFileDefault, "create file %s"); err != nil {
		f.T.Fatalf("Failed to create file %s: %v", path, err)
	}
	return f
}

// WithSystemGo adds a mock system Go binary to the PATH.
// Returns the fixture for method chaining.
func (f *Fixture) WithSystemGo(version string) *Fixture {
	systemBinDir := filepath.Join(f.Root, "system-bin")
	CreateGoExecutable(f.T, systemBinDir)

	// Add to PATH
	currentPath := os.Getenv(utils.EnvVarPath)
	newPath := systemBinDir + string(os.PathListSeparator) + currentPath
	os.Setenv(utils.EnvVarPath, newPath)

	return f
}

// AssertVersionInstalled checks that a version is installed.
func (f *Fixture) AssertVersionInstalled(version string) {
	if !f.Manager.IsVersionInstalled(version) {
		f.T.Fatalf("Expected version %s to be installed, but it was not", version)
	}
}

// AssertVersionNotInstalled checks that a version is NOT installed.
func (f *Fixture) AssertVersionNotInstalled(version string) {
	if f.Manager.IsVersionInstalled(version) {
		f.T.Fatalf("Expected version %s to NOT be installed, but it was", version)
	}
}

// AssertCurrentVersion checks the current active Go version.
func (f *Fixture) AssertCurrentVersion(expected string) {
	current, _, err := f.Manager.GetCurrentVersion()
	if err != nil {
		f.T.Fatalf("Failed to get current version: %v", err)
	}
	if current != expected {
		f.T.Fatalf("Expected current version %s, got %s", expected, current)
	}
}

// AssertFileExists checks that a file exists.
func (f *Fixture) AssertFileExists(path string) {
	fullPath := filepath.Join(f.Home, path)
	if utils.FileNotExists(fullPath) {
		f.T.Fatalf("Expected file %s to exist, but it does not", path)
	}
}

// AssertFileContains checks that a file contains the given substring.
func (f *Fixture) AssertFileContains(path, substr string) {
	fullPath := filepath.Join(f.Home, path)
	content, err := os.ReadFile(fullPath)
	if err != nil {
		f.T.Fatalf("Failed to read file %s: %v", path, err)
	}
	if !strings.Contains(string(content), substr) {
		f.T.Fatalf("Expected file %s to contain %q, but it does not.\nFile content:\n%s",
			path, substr, string(content))
	}
}
