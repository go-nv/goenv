package cmdtest

import (
	"fmt"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

// VersionBuilder helps build complex Go version test scenarios.
// Use NewVersionBuilder to create a new builder.
type VersionBuilder struct {
	t             *testing.T
	root          string
	version       string
	withBinaries  []string
	withTools     []string
	withGoPkgDir  bool
	withGoModFile bool
}

// NewVersionBuilder creates a new version builder for the given version.
func NewVersionBuilder(t *testing.T, root, version string) *VersionBuilder {
	return &VersionBuilder{
		t:       t,
		root:    root,
		version: version,
	}
}

// WithBinaries adds binary executables to the version's bin directory.
func (vb *VersionBuilder) WithBinaries(names ...string) *VersionBuilder {
	vb.withBinaries = append(vb.withBinaries, names...)
	return vb
}

// WithTools adds tool executables to the version's gopath/bin directory.
func (vb *VersionBuilder) WithTools(names ...string) *VersionBuilder {
	vb.withTools = append(vb.withTools, names...)
	return vb
}

// WithPkgDir creates a pkg directory structure (go-build, mod caches).
func (vb *VersionBuilder) WithPkgDir() *VersionBuilder {
	vb.withGoPkgDir = true
	return vb
}

// WithGoMod creates a go.mod file in the version directory.
func (vb *VersionBuilder) WithGoMod() *VersionBuilder {
	vb.withGoModFile = true
	return vb
}

// Build creates the version with all specified components.
func (vb *VersionBuilder) Build() {
	versionDir := filepath.Join(vb.root, "versions", vb.version)

	// Create base go binary
	CreateTestVersion(vb.t, vb.root, vb.version)

	// Add additional binaries
	for _, binary := range vb.withBinaries {
		CreateTestBinary(vb.t, vb.root, vb.version, binary)
	}

	// Add tools
	if len(vb.withTools) > 0 {
		toolDir := filepath.Join(versionDir, "gopath", "bin")
		for _, tool := range vb.withTools {
			CreateToolExecutable(vb.t, toolDir, tool)
		}
	}

	// Add pkg directory
	if vb.withGoPkgDir {
		pkgDir := filepath.Join(versionDir, "pkg")
		utils.EnsureDir(filepath.Join(pkgDir, "mod"))

		// Create platform-specific cache dir
		platform := "linux-amd64"
		if utils.IsWindows() {
			platform = "windows-amd64"
		}
		utils.EnsureDir(filepath.Join(pkgDir, "go-build-"+platform))
	}

	// Add go.mod
	if vb.withGoModFile {
		gomodPath := filepath.Join(versionDir, "go.mod")
		content := fmt.Sprintf("module goenv-test-version-%s\n\ngo %s\n", vb.version, vb.version)
		if err := utils.WriteFileWithContext(gomodPath, []byte(content), utils.PermFileDefault, "create go.mod"); err != nil {
			vb.t.Fatalf("Failed to create go.mod: %v", err)
		}
	}
}

// ToolBuilder helps build mock tool installations with metadata.
type ToolBuilder struct {
	t           *testing.T
	root        string
	version     string
	toolName    string
	packagePath string
	toolVersion string
}

// NewToolBuilder creates a builder for a tool installation.
func NewToolBuilder(t *testing.T, root, goVersion, toolName string) *ToolBuilder {
	return &ToolBuilder{
		t:        t,
		root:     root,
		version:  goVersion,
		toolName: toolName,
	}
}

// WithPackagePath sets the Go package path for the tool.
func (tb *ToolBuilder) WithPackagePath(path string) *ToolBuilder {
	tb.packagePath = path
	return tb
}

// WithVersion sets the tool version.
func (tb *ToolBuilder) WithVersion(version string) *ToolBuilder {
	tb.toolVersion = version
	return tb
}

// Build creates the tool executable and optionally metadata.
// Note: Actual `go version -m` metadata cannot be easily mocked,
// so this just creates the executable. For tests that need metadata,
// consider using real tool installations or mocking at a higher level.
func (tb *ToolBuilder) Build() string {
	toolDir := filepath.Join(tb.root, "versions", tb.version, "gopath", "bin")
	return CreateToolExecutable(tb.t, toolDir, tb.toolName)
}

// ScenarioBuilder helps build complete test scenarios with multiple versions,
// tools, and configurations in a declarative way.
type ScenarioBuilder struct {
	fixture  *Fixture
	versions []string
}

// NewScenario creates a new scenario builder.
func NewScenario(t *testing.T) *ScenarioBuilder {
	return &ScenarioBuilder{
		fixture: NewFixture(t),
	}
}

// WithVersion adds a Go version to the scenario.
func (sb *ScenarioBuilder) WithVersion(version string) *ScenarioBuilder {
	sb.versions = append(sb.versions, version)
	sb.fixture.WithVersions(version)
	return sb
}

// WithVersionAndTools adds a Go version with specific tools installed.
func (sb *ScenarioBuilder) WithVersionAndTools(version string, tools ...string) *ScenarioBuilder {
	sb.versions = append(sb.versions, version)
	sb.fixture.WithVersions(version).WithTools(version, tools...)
	return sb
}

// WithGlobal sets the global version.
func (sb *ScenarioBuilder) WithGlobal(version string) *ScenarioBuilder {
	sb.fixture.WithGlobalVersion(version)
	return sb
}

// WithLocal sets the local version.
func (sb *ScenarioBuilder) WithLocal(version string) *ScenarioBuilder {
	sb.fixture.WithLocalVersion(version)
	return sb
}

// WithGoMod creates a go.mod file.
func (sb *ScenarioBuilder) WithGoMod(version string) *ScenarioBuilder {
	sb.fixture.WithGoModFile(version)
	return sb
}

// Build returns the configured fixture.
func (sb *ScenarioBuilder) Build() *Fixture {
	return sb.fixture
}

// CommonScenarios provides pre-built common test scenarios

// MultiVersionScenario creates a fixture with multiple Go versions installed.
func MultiVersionScenario(t *testing.T, versions ...string) *Fixture {
	f := NewFixture(t)
	f.WithVersions(versions...)
	if len(versions) > 0 {
		f.WithGlobalVersion(versions[0])
	}
	return f
}

// ToolScenario creates a fixture with Go versions and tools.
func ToolScenario(t *testing.T, version string, tools ...string) *Fixture {
	f := NewFixture(t)
	f.WithVersions(version).WithTools(version, tools...).WithGlobalVersion(version)
	return f
}

// GoModScenario creates a fixture with a go.mod file.
func GoModScenario(t *testing.T, goVersion, toolchainVersion string) *Fixture {
	f := NewFixture(t)
	if toolchainVersion != "" {
		f.WithVersions(goVersion, toolchainVersion)
		f.WithGoModToolchain(goVersion, toolchainVersion)
	} else {
		f.WithVersions(goVersion)
		f.WithGoModFile(goVersion)
	}
	return f
}

// AliasScenario creates a fixture with version aliases.
func AliasScenario(t *testing.T, aliases map[string]string) *Fixture {
	f := NewFixture(t)
	for name, target := range aliases {
		f.WithAlias(name, target)
		f.WithVersions(target) // Ensure target version exists
	}
	return f
}
