# cmdtest Package - Test Fixtures and Helpers

The `cmdtest` package provides comprehensive test fixtures, builders, and helpers for writing clean, maintainable tests in goenv.

## Quick Start

### Basic Fixture Pattern

The simplest way to set up a test environment:

```go
func TestMyCommand(t *testing.T) {
    f := cmdtest.NewFixture(t)
    defer f.Cleanup()

    // Set up test data
    f.WithVersions("1.21.0", "1.22.0").
      WithGlobalVersion("1.21.0")

    // Run your test
    // ... test code ...
}
```

## Available Patterns

### 1. Fixture Pattern (Recommended for Most Tests)

The `Fixture` provides a fluent interface for building test environments:

```go
func TestVersionSwitch(t *testing.T) {
    f := cmdtest.NewFixture(t)
    defer f.Cleanup()

    // Chain multiple setup methods
    f.WithVersions("1.21.0", "1.22.0", "1.23.0").
      WithTools("1.21.0", "gopls", "staticcheck").
      WithGlobalVersion("1.21.0").
      WithAlias("stable", "1.22.0")

    // Assert expectations
    f.AssertCurrentVersion("1.21.0")
    f.AssertVersionInstalled("1.22.0")
}
```

### 2. Scenario Builder Pattern

For complex multi-component scenarios:

```go
func TestComplexScenario(t *testing.T) {
    f := cmdtest.NewScenario(t).
        WithVersionAndTools("1.21.0", "gopls", "golangci-lint").
        WithVersionAndTools("1.22.0", "gopls", "staticcheck").
        WithGlobal("1.21.0").
        WithGoMod("1.22.0").
        Build()
    defer f.Cleanup()

    // Test with pre-configured complex environment
}
```

### 3. Pre-Built Scenarios

Common scenarios are available as functions:

```go
// Multiple versions
f := cmdtest.MultiVersionScenario(t, "1.21.0", "1.22.0", "1.23.0")
defer f.Cleanup()

// Version with tools
f := cmdtest.ToolScenario(t, "1.21.0", "gopls", "staticcheck")
defer f.Cleanup()

// go.mod scenario
f := cmdtest.GoModScenario(t, "1.22.0", "1.22.1")
defer f.Cleanup()

// Aliases
f := cmdtest.AliasScenario(t, map[string]string{
    "stable": "1.22.0",
    "latest": "1.23.0",
})
defer f.Cleanup()
```

### 4. Builder Pattern

For fine-grained control over version setup:

```go
func TestVersionWithCache(t *testing.T) {
    f := cmdtest.NewFixture(t)
    defer f.Cleanup()

    // Build a complex version setup
    cmdtest.NewVersionBuilder(t, f.Root, "1.21.0").
        WithBinaries("go", "gofmt").
        WithTools("gopls", "staticcheck").
        WithPkgDir().  // Creates pkg/mod and go-build dirs
        WithGoMod().   // Creates a go.mod in the version dir
        Build()

    // Test code...
}
```

### 5. Legacy Pattern (For Existing Tests)

The original helpers are still available:

```go
func TestLegacyPattern(t *testing.T) {
    testRoot, cleanup := cmdtest.SetupTestEnv(t)
    defer cleanup()

    cmdtest.CreateTestVersion(t, testRoot, "1.21.0")
    cmdtest.CreateTestBinary(t, testRoot, "1.21.0", "gofmt")
    cmdtest.CreateTestAlias(t, testRoot, "stable", "1.21.0")

    // Test code...
}
```

## Fixture Methods Reference

### Environment Setup
- `NewFixture(t)` - Create new test fixture with isolated environment
- `WithVersions(versions...)` - Install multiple Go versions
- `WithTools(version, tools...)` - Add tools for a specific version
- `WithGlobalVersion(version)` - Set global version
- `WithLocalVersion(version)` - Set local version (.go-version)
- `WithAlias(name, target)` - Create version alias
- `WithSystemGo(version)` - Add mock system Go to PATH

### Version Files
- `WithGoModFile(version)` - Create go.mod with go directive
- `WithGoModToolchain(goVer, toolchainVer)` - Create go.mod with toolchain
- `WithToolVersionsFile(version, tools...)` - Create .tool-versions (asdf format)
- `WithFile(path, content)` - Create arbitrary file

### Assertions
- `AssertVersionInstalled(version)` - Check version is installed
- `AssertVersionNotInstalled(version)` - Check version is NOT installed
- `AssertCurrentVersion(expected)` - Check current active version
- `AssertFileExists(path)` - Check file exists
- `AssertFileContains(path, substr)` - Check file contains text

### Access
- `f.Root` - GOENV_ROOT directory
- `f.Home` - Test HOME directory
- `f.Config` - Config instance
- `f.Manager` - Manager instance
- `f.Cleanup()` - Cleanup function (call with defer)

## Best Practices

### 1. Use Fixture for New Tests

Prefer `NewFixture(t)` over `SetupTestEnv(t)` for new tests:

```go
// ✅ Good - Modern fixture pattern
func TestMyFeature(t *testing.T) {
    f := cmdtest.NewFixture(t)
    defer f.Cleanup()
    f.WithVersions("1.21.0")
}

// ⚠️  Old - Legacy pattern (still works, but more verbose)
func TestMyFeature(t *testing.T) {
    root, cleanup := cmdtest.SetupTestEnv(t)
    defer cleanup()
    cmdtest.CreateTestVersion(t, root, "1.21.0")
}
```

### 2. Method Chaining

Use method chaining for concise setup:

```go
f := cmdtest.NewFixture(t)
defer f.Cleanup()

f.WithVersions("1.21.0", "1.22.0").
  WithTools("1.21.0", "gopls").
  WithGlobalVersion("1.21.0").
  WithAlias("stable", "1.21.0")
```

### 3. Pre-Built Scenarios for Common Cases

Use pre-built scenarios when they match your needs:

```go
// Instead of manually setting up 3 versions
f := cmdtest.MultiVersionScenario(t, "1.21.0", "1.22.0", "1.23.0")
```

### 4. Builders for Complex Version Setup

Use builders when you need detailed control:

```go
cmdtest.NewVersionBuilder(t, f.Root, "1.21.0").
    WithBinaries("go", "gofmt", "gocov").
    WithTools("gopls", "staticcheck", "golangci-lint").
    WithPkgDir().
    Build()
```

## Examples

### Example 1: Testing Version Switching

```go
func TestVersionSwitch(t *testing.T) {
    f := cmdtest.NewFixture(t)
    defer f.Cleanup()

    // Setup
    f.WithVersions("1.21.0", "1.22.0").
      WithGlobalVersion("1.21.0")

    // Test global version
    f.AssertCurrentVersion("1.21.0")

    // Switch to local
    f.WithLocalVersion("1.22.0")
    f.AssertCurrentVersion("1.22.0")
}
```

### Example 2: Testing Tool Management

```go
func TestToolInstall(t *testing.T) {
    f := cmdtest.ToolScenario(t, "1.21.0", "gopls", "staticcheck")
    defer f.Cleanup()

    // Tools are already installed
    toolDir := filepath.Join(f.Root, "versions", "1.21.0", "gopath", "bin")

    // Verify tools exist
    f.AssertFileExists("../versions/1.21.0/gopath/bin/gopls")
    f.AssertFileExists("../versions/1.21.0/gopath/bin/staticcheck")
}
```

### Example 3: Testing go.mod Integration

```go
func TestGoModParsing(t *testing.T) {
    f := cmdtest.GoModScenario(t, "1.22.0", "1.22.1")
    defer f.Cleanup()

    // go.mod exists with go 1.22.0 and toolchain go1.22.1
    f.AssertFileExists("go.mod")
    f.AssertFileContains("go.mod", "go 1.22.0")
    f.AssertFileContains("go.mod", "toolchain go1.22.1")
}
```

### Example 4: Testing Alias Resolution

```go
func TestAliasResolution(t *testing.T) {
    f := cmdtest.NewFixture(t)
    defer f.Cleanup()

    f.WithVersions("1.21.0", "1.22.0").
      WithAlias("stable", "1.22.0").
      WithAlias("legacy", "1.21.0")

    // Use aliases in your command tests
    // The versions they point to are guaranteed to exist
}
```

## Migration Guide

### Migrating Existing Tests

Old pattern:
```go
func TestOld(t *testing.T) {
    testRoot, cleanup := cmdtest.SetupTestEnv(t)
    defer cleanup()

    cmdtest.CreateTestVersion(t, testRoot, "1.21.0")
    cmdtest.CreateTestBinary(t, testRoot, "1.21.0", "gofmt")

    cfg := config.Load()
    mgr := manager.NewManager(cfg)

    // Test code using cfg and mgr
}
```

New pattern:
```go
func TestNew(t *testing.T) {
    f := cmdtest.NewFixture(t)
    defer f.Cleanup()

    f.WithVersions("1.21.0")
    cmdtest.NewVersionBuilder(t, f.Root, "1.21.0").
        WithBinaries("gofmt").
        Build()

    // Test code using f.Config and f.Manager
}
```

## Architecture

```
cmdtest/
├── testhelpers.go  - Low-level helpers (legacy, still supported)
├── fixtures.go     - Fixture pattern with fluent interface
├── builders.go     - Builder patterns for complex scenarios
└── README.md       - This file
```

## Contributing

When adding new test helpers:

1. **Fixtures** - Add methods to `Fixture` for common setup operations
2. **Builders** - Create new builders for complex component setup
3. **Scenarios** - Add pre-built scenarios for recurring patterns
4. **Helpers** - Add low-level helpers to `testhelpers.go` only if needed by multiple builders

Keep helpers focused and composable. Prefer method chaining for better readability.
