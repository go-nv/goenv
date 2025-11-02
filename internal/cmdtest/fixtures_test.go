package cmdtest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
)

func TestNewFixture(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	// Verify fixture is properly initialized
	if f.T != t {
		t.Error("Fixture.T not set correctly")
	}
	if f.Root == "" {
		t.Error("Fixture.Root is empty")
	}
	if f.Home == "" {
		t.Error("Fixture.Home is empty")
	}
	if f.Config == nil {
		t.Error("Fixture.Config is nil")
	}
	if f.Manager == nil {
		t.Error("Fixture.Manager is nil")
	}
	if f.Cleanup == nil {
		t.Error("Fixture.Cleanup is nil")
	}
}

func TestFixtureWithVersions(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	// Add versions using method chaining
	f.WithVersions("1.21.0", "1.22.0", "1.23.0")

	// Verify versions were created using ListInstalledVersions
	versions, err := f.Manager.ListInstalledVersions()
	if err != nil {
		t.Fatalf("Failed to list installed versions: %v", err)
	}

	expectedVersions := []string{"1.21.0", "1.22.0", "1.23.0"}
	if len(versions) != len(expectedVersions) {
		t.Errorf("Expected %d versions, got %d", len(expectedVersions), len(versions))
	}

	for _, expected := range expectedVersions {
		found := false
		for _, v := range versions {
			if v == expected {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Expected version %s not found in installed versions", expected)
		}
	}
}

func TestFixtureWithTools(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithVersions("1.21.0").
		WithTools("1.21.0", "gopls", "staticcheck")

	// Verify tools were created using FindExecutable (handles .bat/.exe on Windows)
	toolDir := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")

	if _, err := utils.FindExecutable(toolDir, "gopls"); err != nil {
		t.Errorf("gopls not found in %s: %v", toolDir, err)
	}

	if _, err := utils.FindExecutable(toolDir, "staticcheck"); err != nil {
		t.Errorf("staticcheck not found in %s: %v", toolDir, err)
	}
}

func TestFixtureWithGlobalVersion(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithVersions("1.21.0").
		WithGlobalVersion("1.21.0")

	// Verify global version was set
	current, _, err := f.Manager.GetCurrentVersion()
	if err != nil {
		t.Fatalf("Failed to get current version: %v", err)
	}
	if current != "1.21.0" {
		t.Errorf("Expected current version 1.21.0, got %s", current)
	}
}

func TestFixtureWithLocalVersion(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithVersions("1.21.0", "1.22.0").
		WithGlobalVersion("1.21.0").
		WithLocalVersion("1.22.0")

	// Verify local version overrides global
	current, _, err := f.Manager.GetCurrentVersion()
	if err != nil {
		t.Fatalf("Failed to get current version: %v", err)
	}
	if current != "1.22.0" {
		t.Errorf("Expected current version 1.22.0 (local), got %s", current)
	}
}

func TestFixtureWithGoModFile(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithGoModFile("1.22.0")

	// Verify go.mod was created
	gomodPath := filepath.Join(f.Home, "go.mod")
	content, err := os.ReadFile(gomodPath)
	if err != nil {
		t.Fatalf("go.mod not created: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "module test") {
		t.Error("go.mod missing module declaration")
	}
	if !strings.Contains(contentStr, "go 1.22.0") {
		t.Error("go.mod missing go directive")
	}
}

func TestFixtureWithGoModToolchain(t *testing.T) {
	f := NewFixture(t)
	defer f.Cleanup()

	f.WithGoModToolchain("1.22.0", "1.22.1")

	// Verify go.mod contains both directives
	gomodPath := filepath.Join(f.Home, "go.mod")
	content, err := os.ReadFile(gomodPath)
	if err != nil {
		t.Fatalf("go.mod not created: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "go 1.22.0") {
		t.Error("go.mod missing go directive")
	}
	if !strings.Contains(contentStr, "toolchain go1.22.1") {
		t.Error("go.mod missing toolchain directive")
	}
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
	f := ToolScenario(t, "1.21.0", "gopls", "staticcheck")
	defer f.Cleanup()

	// Verify version and tools
	f.AssertVersionInstalled("1.21.0")

	// Verify tools were created using FindExecutable (handles .bat/.exe on Windows)
	toolDir := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")
	if _, err := utils.FindExecutable(toolDir, "gopls"); err != nil {
		t.Errorf("gopls not created in ToolScenario: %v", err)
	}
	if _, err := utils.FindExecutable(toolDir, "staticcheck"); err != nil {
		t.Errorf("staticcheck not created in ToolScenario: %v", err)
	}
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
	if err != nil {
		t.Fatalf("aliases file not created: %v", err)
	}

	contentStr := string(content)
	if !strings.Contains(contentStr, "stable=1.22.0") {
		t.Error("stable alias not found in aliases file")
	}
	if !strings.Contains(contentStr, "latest=1.23.0") {
		t.Error("latest alias not found in aliases file")
	}
}

func TestVersionBuilder(t *testing.T) {
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
	if _, err := utils.FindExecutable(binDir, "go"); err != nil {
		t.Errorf("go binary not created: %v", err)
	}
	if _, err := utils.FindExecutable(binDir, "gofmt"); err != nil {
		t.Errorf("gofmt binary not created: %v", err)
	}

	// Verify tools using FindExecutable (handles .bat/.exe on Windows)
	toolDir := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")
	if _, err := utils.FindExecutable(toolDir, "gopls"); err != nil {
		t.Errorf("gopls tool not created: %v", err)
	}

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
	if _, err := utils.FindExecutable(toolDir1, "gopls"); err != nil {
		t.Errorf("gopls not created for 1.21.0: %v", err)
	}

	toolDir2 := filepath.Join(f.Root, "versions", "1.22.0", "gopath", "bin")
	if _, err := utils.FindExecutable(toolDir2, "staticcheck"); err != nil {
		t.Errorf("staticcheck not created for 1.22.0: %v", err)
	}
}
